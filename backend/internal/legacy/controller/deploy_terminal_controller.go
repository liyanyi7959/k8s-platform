package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"

	"k8s-platform-backend/internal/legacy/service"
	"k8s-platform-backend/pkg/resp"
)

type serverTerminalFrame struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// CreateServerTerminalSession 创建一次性服务器终端会话。
func (dc *DeployController) CreateServerTerminalSession(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if dc.terminalSessions == nil {
		resp.Fail(c, 5000, "终端服务未初始化")
		return
	}
	server, err := dc.svc.GetServer(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}

	sessionID := dc.terminalSessions.NewSessionID()
	dc.terminalSessions.Put(sessionID, service.ExecSession{
		Kind:      "server",
		UserID:    currentUserID(c),
		ServerID:  id,
		CreatedAt: time.Now().UTC(),
	})
	resp.OK(c, gin.H{
		"session_id": sessionID,
		"ws_url":     "/api/v1/deploy/servers/terminal/ws?session_id=" + url.QueryEscape(sessionID),
		"server":     server,
	})
}

// ServerTerminalWS 将浏览器 WebSocket 与目标服务器的交互式 SSH 会话桥接。
func (dc *DeployController) ServerTerminalWS(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Query("session_id"))
	if sessionID == "" || dc.terminalSessions == nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	pending, ok := dc.terminalSessions.Get(sessionID)
	if !ok || pending.Kind != "server" || pending.ServerID == 0 {
		resp.Fail(c, 4040, "终端会话不存在或已过期")
		return
	}
	if pending.UserID == 0 || pending.UserID != currentUserID(c) {
		resp.Fail(c, 1003, "终端会话不属于当前用户")
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin:     sameOriginWebSocketRequest,
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zap.L().Warn("server_terminal_ws: websocket upgrade failed", zap.String("session_id", sessionID), zap.Error(err))
		return
	}
	defer conn.Close()

	terminalSession, ok := dc.terminalSessions.Take(sessionID)
	if !ok || terminalSession.Kind != "server" || terminalSession.UserID != currentUserID(c) {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "session invalid"), time.Now().Add(3*time.Second))
		return
	}

	client, server, err := dc.svc.OpenServerSSH(c.Request.Context(), terminalSession.ServerID)
	if err != nil {
		_ = conn.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte(err.Error())...))
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, websocketCloseReason(err, "SSH 连接失败")), time.Now().Add(3*time.Second))
		return
	}
	defer client.Close()

	sshSession, err := client.NewSession()
	if err != nil {
		_ = conn.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte("创建 SSH 会话失败："+err.Error())...))
		return
	}
	defer sshSession.Close()

	stdin, err := sshSession.StdinPipe()
	if err != nil {
		_ = conn.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte("终端输入初始化失败")...))
		return
	}
	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		_ = conn.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte("终端输出初始化失败")...))
		return
	}
	stderr, err := sshSession.StderrPipe()
	if err != nil {
		_ = conn.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte("终端错误输出初始化失败")...))
		return
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sshSession.RequestPty("xterm-256color", 36, 120, modes); err != nil {
		_ = conn.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte("目标主机不支持交互式终端："+err.Error())...))
		return
	}
	if err := sshSession.Shell(); err != nil {
		_ = conn.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte("启动远程 Shell 失败："+err.Error())...))
		return
	}

	zap.L().Info("server_terminal_ws: terminal started",
		zap.String("session_id", sessionID),
		zap.Uint64("server_id", terminalSession.ServerID),
		zap.String("server", server.Name),
		zap.Uint64("user_id", terminalSession.UserID),
	)

	var writeMu sync.Mutex
	writeBinary := func(channel byte, data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		payload := make([]byte, len(data)+1)
		payload[0] = channel
		copy(payload[1:], data)
		return conn.WriteMessage(websocket.BinaryMessage, payload)
	}

	var outputWG sync.WaitGroup
	copyOutput := func(channel byte, reader io.Reader) {
		defer outputWG.Done()
		_, _ = io.Copy(&wsWriter{write: func(data []byte) error { return writeBinary(channel, data) }}, reader)
	}
	outputWG.Add(2)
	go copyOutput(1, stdout)
	go copyOutput(2, stderr)

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			messageType, payload, readErr := conn.ReadMessage()
			if readErr != nil {
				return
			}
			if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
				continue
			}
			var frame serverTerminalFrame
			if messageType == websocket.TextMessage && json.Unmarshal(payload, &frame) == nil && frame.Type != "" {
				switch strings.ToLower(strings.TrimSpace(frame.Type)) {
				case "stdin":
					if frame.Data != "" {
						if _, writeErr := io.WriteString(stdin, frame.Data); writeErr != nil {
							return
						}
					}
				case "resize":
					if frame.Rows > 0 && frame.Cols > 0 {
						_ = sshSession.WindowChange(frame.Rows, frame.Cols)
					}
				}
				continue
			}
			if len(payload) > 0 {
				if _, writeErr := stdin.Write(payload); writeErr != nil {
					return
				}
			}
		}
	}()

	waitDone := make(chan error, 1)
	go func() { waitDone <- sshSession.Wait() }()

	var waitErr error
	clientDisconnected := false
	select {
	case waitErr = <-waitDone:
	case <-readDone:
		clientDisconnected = true
		_ = sshSession.Close()
		waitErr = <-waitDone
	}
	outputWG.Wait()
	if waitErr != nil && !clientDisconnected {
		_ = writeBinary(3, []byte("远程 Shell 已结束："+waitErr.Error()))
	}
}

func sameOriginWebSocketRequest(request *http.Request) bool {
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.TrimSpace(request.Host)
	if forwardedHost := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-Host"), ",")[0]); forwardedHost != "" {
		host = forwardedHost
	}
	if strings.EqualFold(parsedOrigin.Host, host) {
		return true
	}
	parsedHost, err := url.Parse("http://" + host)
	return err == nil && strings.EqualFold(parsedOrigin.Hostname(), parsedHost.Hostname())
}
