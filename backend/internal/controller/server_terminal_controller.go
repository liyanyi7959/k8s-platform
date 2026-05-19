package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type ServerTerminalController struct {
	svc *service.ServerTerminalService
}

func NewServerTerminalController(svc *service.ServerTerminalService) *ServerTerminalController {
	return &ServerTerminalController{svc: svc}
}

type terminalInboundMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

type terminalOutboundMessage struct {
	Type        string `json:"type"`
	Data        string `json:"data,omitempty"`
	Message     string `json:"message,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	ConnectedAt string `json:"connected_at,omitempty"`
	Reason      string `json:"reason,omitempty"`
	DeadlineAt  string `json:"deadline_at,omitempty"`
	Command     string `json:"command,omitempty"`
	RiskLevel   string `json:"risk_level,omitempty"`
}

func currentUserID(c *gin.Context) int64 {
	v, ok := c.Get("user_id")
	if !ok {
		return 0
	}
	switch id := v.(type) {
	case int64:
		return id
	case int:
		return int64(id)
	case uint64:
		return int64(id)
	case uint:
		return int64(id)
	case float64:
		return int64(id)
	default:
		return 0
	}
}

func (tc *ServerTerminalController) writeServiceErr(c *gin.Context, err error) {
	WriteServiceErr(c, err, SSHErrMappings...)
}

// IssueTicket 为前端签发一次性终端票据。
func (tc *ServerTerminalController) IssueTicket(c *gin.Context) {
	serverID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || serverID == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := tc.svc.IssueTicket(c.Request.Context(), serverID, currentUserID(c))
	if err != nil {
		tc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (tc *ServerTerminalController) Workspace(c *gin.Context) {
	data, err := tc.svc.GetWorkspace(c.Request.Context(), currentUserID(c))
	if err != nil {
		tc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (tc *ServerTerminalController) ListFavorites(c *gin.Context) {
	data, err := tc.svc.ListFavorites(c.Request.Context(), currentUserID(c))
	if err != nil {
		tc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (tc *ServerTerminalController) AddFavorite(c *gin.Context) {
	serverID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || serverID == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := tc.svc.AddFavorite(c.Request.Context(), currentUserID(c), serverID); err != nil {
		tc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"server_id": serverID})
}

func (tc *ServerTerminalController) RemoveFavorite(c *gin.Context) {
	serverID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || serverID == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := tc.svc.RemoveFavorite(c.Request.Context(), currentUserID(c), serverID); err != nil {
		tc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"server_id": serverID})
}

func (tc *ServerTerminalController) ListAudits(c *gin.Context) {
	limit := parseInt(c.Query("limit"), 50)
	data, err := tc.svc.ListAudits(c.Request.Context(), currentUserID(c), limit)
	if err != nil {
		tc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (tc *ServerTerminalController) ActiveSessions(c *gin.Context) {
	resp.OK(c, tc.svc.ActiveSessions(currentUserID(c)))
}

func (tc *ServerTerminalController) CloseSession(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Param("sessionId"))
	if sessionID == "" {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := tc.svc.CloseSession(sessionID, currentUserID(c), "用户主动关闭"); err != nil {
		tc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"session_id": sessionID})
}

// SSHWS 使用 ticket 建立 SSH WebSocket 终端。
func (tc *ServerTerminalController) SSHWS(c *gin.Context) {
	ticket := strings.TrimSpace(c.Query("ticket"))
	claims, err := tc.svc.ConsumeTicket(ticket, currentUserID(c))
	if err != nil {
		tc.writeServiceErr(c, err)
		return
	}

	session, err := tc.svc.OpenSession(c.Request.Context(), claims.ServerID, 120, 36)
	if err != nil {
		tc.writeServiceErr(c, err)
		return
	}
	defer session.Close()

	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
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
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()
	if _, err := tc.svc.RegisterSession(c.Request.Context(), session, claims.ServerID, claims.UserID); err != nil {
		_ = conn.Close()
		return
	}

	const writeWait = 10 * time.Second
	const pongWait = 25 * time.Second
	const pingPeriod = 20 * time.Second

	var writeMu sync.Mutex
	writeJSON := func(payload terminalOutboundMessage) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		return conn.WriteJSON(payload)
	}

	closeSocket := func(reason string) {
		if strings.TrimSpace(reason) != "" {
			_ = writeJSON(terminalOutboundMessage{Type: "closed", Reason: reason})
		}
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(writeWait))
	}

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	done := make(chan struct{})
	var doneOnce sync.Once
	finish := func() {
		doneOnce.Do(func() { close(done) })
	}
	defer finish()

	if err := writeJSON(terminalOutboundMessage{
		Type:        "ready",
		SessionID:   session.SessionID,
		ConnectedAt: session.ConnectedAt.Format(time.RFC3339),
	}); err != nil {
		return
	}

	go func() {
		events, unsubscribe, err := tc.svc.SubscribeSessionEvents(session.SessionID)
		if err != nil {
			return
		}
		defer unsubscribe()
		for {
			select {
			case <-done:
				return
			case evt, ok := <-events:
				if !ok {
					return
				}
				_ = writeJSON(terminalOutboundMessage{
					Type:       evt.Type,
					Message:    evt.Message,
					SessionID:  evt.SessionID,
					DeadlineAt: evt.DeadlineAt,
					Command:    evt.Command,
					RiskLevel:  evt.RiskLevel,
					Reason:     evt.Reason,
				})
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				writeMu.Lock()
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				err := conn.WriteMessage(websocket.PingMessage, nil)
				writeMu.Unlock()
				if err != nil {
					_ = conn.Close()
					return
				}
			}
		}
	}()

	var wg sync.WaitGroup
	pump := func(r io.Reader) {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				if err := writeJSON(terminalOutboundMessage{Type: "output", Data: string(buf[:n])}); err != nil {
					return
				}
			}
			if readErr != nil {
				return
			}
		}
	}

	wg.Add(2)
	go pump(session.Stdout)
	go pump(session.Stderr)

	for {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
			continue
		}
		if len(payload) == 0 {
			continue
		}

		var msg terminalInboundMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			if _, writeErr := session.Stdin.Write(payload); writeErr != nil {
				break
			}
			continue
		}

		switch msg.Type {
		case "resize":
			if msg.Cols > 0 && msg.Rows > 0 {
				_ = session.Session.WindowChange(msg.Rows, msg.Cols)
			}
		case "input":
			if msg.Data == "" {
				continue
			}
			tc.svc.HandleSessionInput(session.SessionID, msg.Data)
			if _, err := session.Stdin.Write([]byte(msg.Data)); err != nil {
				break
			}
			tc.svc.TouchSession(session.SessionID)
		default:
			continue
		}
	}

	finish()
	_ = tc.svc.CloseSession(session.SessionID, currentUserID(c), "连接已关闭")
	closeSocket("连接已关闭")
	wg.Wait()
}
