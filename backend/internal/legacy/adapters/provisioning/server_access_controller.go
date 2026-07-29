package provisioning

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/kops/adapters/legacycompat"
	"k8s-platform-backend/internal/middleware"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
	"k8s-platform-backend/pkg/resp"
)

// ServerAccessController is the HTTP adapter for SSH diagnostics and the
// interactive terminal. Its runtime dependency is independent from
// DeploymentExecutor, which owns Ansible and deployment-plan execution.
type ServerAccessController struct {
	runtime  serverAccessRuntime
	sessions *kopsapp.ExecSessionStore
	servers  serverAccessReader
}

type serverAccessRuntime interface {
	ProbeServerSSH(context.Context, uint64) (provisionapp.SSHProbeResult, error)
	OpenServerSSH(context.Context, uint64) (*ssh.Client, string, error)
}

type serverAccessReader interface {
	Get(context.Context, uint64) (provisionapp.DeployServerItem, error)
}

func NewServerAccessController(runtime serverAccessRuntime, sessions *kopsapp.ExecSessionStore, servers serverAccessReader) *ServerAccessController {
	return &ServerAccessController{runtime: runtime, sessions: sessions, servers: servers}
}

func (ctl *ServerAccessController) TestSSH(c *gin.Context) {
	id, ok := serverAccessID(c, "id")
	if !ok {
		return
	}
	if ctl == nil || ctl.runtime == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	data, err := ctl.runtime.ProbeServerSSH(c.Request.Context(), id)
	if err != nil {
		writeServerAccessError(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *ServerAccessController) CreateTerminalSession(c *gin.Context) {
	id, ok := serverAccessID(c, "id")
	if !ok {
		return
	}
	if ctl == nil || ctl.runtime == nil || ctl.sessions == nil || ctl.servers == nil {
		resp.Fail(c, 5000, "terminal service is unavailable")
		return
	}
	server, err := ctl.servers.Get(c.Request.Context(), id)
	if err != nil {
		writeServerAccessError(c, err)
		return
	}
	sessionID := ctl.sessions.NewSessionID()
	ctl.sessions.Put(sessionID, kopsapp.ExecSession{Kind: "server", UserID: serverAccessUserID(c), ServerID: id, CreatedAt: time.Now().UTC()})
	resp.OK(c, gin.H{
		"session_id": sessionID,
		"ws_url":     "/api/v1/deploy/servers/terminal/ws?session_id=" + url.QueryEscape(sessionID),
		"server":     server,
	})
}

func (ctl *ServerAccessController) TerminalWS(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Query("session_id"))
	if sessionID == "" || ctl == nil || ctl.sessions == nil || ctl.runtime == nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	pending, ok := ctl.sessions.Get(sessionID)
	if !ok || pending.Kind != "server" || pending.ServerID == 0 {
		resp.Fail(c, 4040, "terminal session not found")
		return
	}
	if pending.UserID == 0 || pending.UserID != serverAccessUserID(c) {
		resp.Fail(c, 1003, "terminal session does not belong to the current user")
		return
	}
	upgrader := websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 4096, CheckOrigin: sameOriginServerTerminalRequest}
	connection, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zap.L().Warn("server_terminal_ws: websocket upgrade failed", zap.String("session_id", sessionID), zap.Error(err))
		return
	}
	defer connection.Close()

	terminal, ok := ctl.sessions.Take(sessionID)
	if !ok || terminal.Kind != "server" || terminal.UserID != serverAccessUserID(c) {
		_ = connection.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "session invalid"), time.Now().Add(3*time.Second))
		return
	}
	client, serverName, err := ctl.runtime.OpenServerSSH(c.Request.Context(), terminal.ServerID)
	if err != nil {
		_ = connection.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte(err.Error())...))
		_ = connection.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, serverTerminalCloseReason(err, "SSH connection failed")), time.Now().Add(3*time.Second))
		return
	}
	defer client.Close()
	sshSession, err := client.NewSession()
	if err != nil {
		_ = connection.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte("create SSH session failed: "+err.Error())...))
		return
	}
	defer sshSession.Close()
	stdin, err := sshSession.StdinPipe()
	if err != nil {
		_ = connection.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte("terminal input initialization failed")...))
		return
	}
	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		_ = connection.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte("terminal output initialization failed")...))
		return
	}
	stderr, err := sshSession.StderrPipe()
	if err != nil {
		_ = connection.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte("terminal error output initialization failed")...))
		return
	}
	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
	if err := sshSession.RequestPty("xterm-256color", 36, 120, modes); err != nil {
		_ = connection.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte("target host does not support terminal: "+err.Error())...))
		return
	}
	if err := sshSession.Shell(); err != nil {
		_ = connection.WriteMessage(websocket.BinaryMessage, append([]byte{3}, []byte("start remote shell failed: "+err.Error())...))
		return
	}
	zap.L().Info("server_terminal_ws: terminal started", zap.String("session_id", sessionID), zap.Uint64("server_id", terminal.ServerID), zap.String("server", serverName), zap.Uint64("user_id", terminal.UserID))
	serverTerminalBridge(connection, sshSession, stdin, stdout, stderr)
}

func serverTerminalBridge(connection *websocket.Conn, sshSession *ssh.Session, stdin io.WriteCloser, stdout, stderr io.Reader) {
	var writes sync.Mutex
	writeBinary := func(channel byte, data []byte) error {
		writes.Lock()
		defer writes.Unlock()
		payload := make([]byte, len(data)+1)
		payload[0] = channel
		copy(payload[1:], data)
		return connection.WriteMessage(websocket.BinaryMessage, payload)
	}
	var output sync.WaitGroup
	copyOutput := func(channel byte, reader io.Reader) {
		defer output.Done()
		_, _ = io.Copy(serverTerminalWriter{write: func(data []byte) error { return writeBinary(channel, data) }}, reader)
	}
	output.Add(2)
	go copyOutput(1, stdout)
	go copyOutput(2, stderr)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			messageType, payload, err := connection.ReadMessage()
			if err != nil {
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
						if _, err := io.WriteString(stdin, frame.Data); err != nil {
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
				if _, err := stdin.Write(payload); err != nil {
					return
				}
			}
		}
	}()
	waitDone := make(chan error, 1)
	go func() { waitDone <- sshSession.Wait() }()
	var waitErr error
	disconnected := false
	select {
	case waitErr = <-waitDone:
	case <-readDone:
		disconnected = true
		_ = sshSession.Close()
		waitErr = <-waitDone
	}
	output.Wait()
	if waitErr != nil && !disconnected {
		_ = writeBinary(3, []byte("remote shell ended: "+waitErr.Error()))
	}
}

type serverTerminalFrame struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

type serverTerminalWriter struct{ write func([]byte) error }

func (w serverTerminalWriter) Write(data []byte) (int, error) {
	if w.write == nil {
		return 0, io.ErrClosedPipe
	}
	if err := w.write(data); err != nil {
		return 0, err
	}
	return len(data), nil
}

func serverAccessID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "invalid params")
		return 0, false
	}
	return id, true
}

func serverAccessUserID(c *gin.Context) uint64 {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims == nil || claims.UserID <= 0 {
		return 0
	}
	return uint64(claims.UserID)
}

func writeServerAccessError(c *gin.Context, err error) {
	message := "internal error"
	if userMessage, ok := service.UserMessage(err); ok && strings.TrimSpace(userMessage) != "" {
		message = userMessage
	}
	var provisioningError *provisionapp.Error
	if errors.As(err, &provisioningError) && strings.TrimSpace(provisioningError.UserMessage()) != "" {
		message = provisioningError.UserMessage()
	}
	switch {
	case errors.Is(err, service.ErrInvalidParams), errors.Is(err, provisionapp.ErrInvalidParams):
		resp.Fail(c, 4000, message)
	case errors.Is(err, service.ErrNotFound), errors.Is(err, provisionapp.ErrNotFound):
		resp.Fail(c, 4040, message)
	case errors.Is(err, service.ErrConflict), errors.Is(err, provisionapp.ErrConflict):
		resp.Fail(c, 4090, message)
	default:
		resp.Fail(c, 5000, message)
	}
}

func sameOriginServerTerminalRequest(request *http.Request) bool {
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

func serverTerminalCloseReason(err error, fallback string) string {
	message := strings.TrimSpace(fallback)
	if userMessage, ok := service.UserMessage(err); ok && strings.TrimSpace(userMessage) != "" {
		message = strings.TrimSpace(userMessage)
	} else if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = strings.TrimSpace(err.Error())
	}
	for len(message) > 120 {
		_, size := utf8.DecodeLastRuneInString(message)
		if size <= 0 {
			break
		}
		message = message[:len(message)-size]
	}
	return strings.TrimSpace(message)
}
