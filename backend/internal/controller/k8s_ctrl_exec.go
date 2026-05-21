package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"k8s.io/client-go/tools/remotecommand"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

func websocketCloseReason(err error, fallback string) string {
	message := strings.TrimSpace(fallback)
	if userMessage, ok := service.UserMessage(err); ok && strings.TrimSpace(userMessage) != "" {
		message = strings.TrimSpace(userMessage)
	} else if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = strings.TrimSpace(err.Error())
	}
	if message == "" {
		return ""
	}
	const maxBytes = 120
	if len(message) <= maxBytes {
		return message
	}
	for len(message) > maxBytes {
		_, size := utf8.DecodeLastRuneInString(message)
		if size <= 0 {
			break
		}
		message = message[:len(message)-size]
	}
	return strings.TrimSpace(message)
}

// ──────────────────────────────────────────────────────────
//  Pod Exec WebSocket 接口
// ──────────────────────────────────────────────────────────

// PodExecWS 通过 WebSocket 建立交互式 exec 通道（类终端）。
// 约定：
//   - 连接时必须携带 session_id（由 CreatePodExecSession 创建）；
//   - 服务端会一次性消费该 session，防止 session 重放/复用；
//   - 服务端向前端发送 BinaryMessage，首字节为通道标识：
//     1=stdout，2=stderr，3=error（非致命错误文本）。
//
// @Summary Pod Exec WebSocket
// @Description 通过 WebSocket 建立交互式 exec 通道（携带 session_id）
// @Tags WebSocket 接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param session_id query string true "Exec 会话ID"
// @Success 200 {object} resp.Result "升级 WebSocket"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 404 {object} resp.Result "会话不存在"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /ws/pod-exec [get]
func (kc *K8sController) PodExecWS(c *gin.Context) {
	sid := strings.TrimSpace(c.Query("session_id"))
	if sid == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if _, ok := kc.execSessions.Get(sid); !ok {
		resp.Fail(c, 4040, "not found")
		return
	}

	up := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			// 仅允许同源（Origin host 与请求 host 相同），避免浏览器环境下的跨站 WS 连接。
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin == "" {
				return true
			}
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			host := strings.TrimSpace(r.Host)
			if xf := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0]); xf != "" {
				host = xf
			}
			if strings.EqualFold(u.Host, host) {
				return true
			}
			h, err := url.Parse("http://" + host)
			if err == nil && strings.EqualFold(u.Hostname(), h.Hostname()) {
				return true
			}
			return false
		},
	}
	conn, err := up.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	sess, ok := kc.execSessions.Take(sid)
	if !ok {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "session not found"), time.Now().Add(3*time.Second))
		return
	}
	tty := true
	if sess.TTY != nil {
		tty = *sess.TTY
	}
	cmd := sess.Command
	if len(cmd) == 0 {
		cmd = []string{"sh"}
	}

	// io.Pipe 将 WebSocket 输入转为 io.Reader，供 PodExec 作为 stdin 使用。
	pr, pw := io.Pipe()
	defer func() { _ = pr.Close() }()

	type termFrame struct {
		Type string `json:"type"`
		Data string `json:"data"`
		Cols int    `json:"cols"`
		Rows int    `json:"rows"`
	}

	resizeCh := make(chan remotecommand.TerminalSize, 8)
	resizeQueue := &terminalSizeQueue{ch: resizeCh}

	var wmu sync.Mutex
	writeBin := func(ch byte, b []byte) error {
		// gorilla/websocket 的 Conn 写入不保证并发安全，统一加锁避免数据交错。
		wmu.Lock()
		defer wmu.Unlock()
		msg := make([]byte, 1+len(b))
		msg[0] = ch
		copy(msg[1:], b)
		return conn.WriteMessage(websocket.BinaryMessage, msg)
	}

	stdout := &wsWriter{write: func(p []byte) error { return writeBin(1, p) }}
	stderr := &wsWriter{write: func(p []byte) error { return writeBin(2, p) }}
	const closeWait = 5 * time.Second
	var closeOnce sync.Once
	closeSocket := func(code int, text string) {
		closeOnce.Do(func() {
			wmu.Lock()
			defer wmu.Unlock()
			_ = conn.SetWriteDeadline(time.Now().Add(closeWait))
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, text), time.Now().Add(closeWait))
		})
	}
	defer closeSocket(websocket.CloseNormalClosure, "")

	execCtx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	var stopExecOnce sync.Once
	stopExec := func() {
		stopExecOnce.Do(func() {
			cancel()
			_ = pw.Close()
			close(resizeCh)
		})
	}
	defer stopExec()

	go func() {
		defer func() {
			stopExec()
		}()
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
				continue
			}
			if len(msg) == 0 {
				continue
			}

			var f termFrame
			if mt == websocket.TextMessage && json.Unmarshal(msg, &f) == nil && f.Type != "" {
				switch strings.ToLower(strings.TrimSpace(f.Type)) {
				case "stdin":
					if f.Data == "" {
						continue
					}
					if _, err := pw.Write([]byte(f.Data)); err != nil {
						return
					}
					continue
				case "resize":
					if f.Cols > 0 && f.Rows > 0 {
						select {
						case resizeCh <- remotecommand.TerminalSize{Width: uint16(f.Cols), Height: uint16(f.Rows)}:
						default:
						}
					}
					continue
				default:
					continue
				}
			}

			if _, err := pw.Write(msg); err != nil {
				return
			}
		}
	}()

	err = kc.svc.PodExec(execCtx, sess.ClusterID, sess.Namespace, sess.Pod, sess.Container, cmd, tty, pr, stdout, stderr, resizeQueue)
	if err != nil && !errors.Is(err, context.Canceled) {
		// 将错误发给前端，但不再中断后续 defer 清理流程。
		_ = writeBin(3, []byte(err.Error()))
		closeSocket(websocket.CloseInternalServerErr, websocketCloseReason(err, "exec failed"))
	}
}
