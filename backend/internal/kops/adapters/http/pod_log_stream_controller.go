package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

type PodLogStreamRuntime interface {
	Stream(context.Context, uint64, string, string, string, bool, int64, bool) (io.ReadCloser, error)
}

type PodLogStreamController struct {
	runtime  PodLogStreamRuntime
	sessions *kopsapp.PodLogSessionStore
}

func NewPodLogStreamController(runtime PodLogStreamRuntime, sessions *kopsapp.PodLogSessionStore) *PodLogStreamController {
	return &PodLogStreamController{runtime: runtime, sessions: sessions}
}

func (ctl *PodLogStreamController) Stream(c *gin.Context) {
	if ctl == nil || ctl.runtime == nil || ctl.sessions == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	sessionID := strings.TrimSpace(c.Query("session_id"))
	if sessionID == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	session, ok := ctl.sessions.Take(sessionID)
	if !ok {
		resp.Fail(c, 4040, "not found")
		return
	}
	upgrader := websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 4096, CheckOrigin: podStreamSameOrigin}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	var writeMu sync.Mutex
	write := func(frame podLogFrame) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(frame)
	}
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	container := ""
	if session.Container != nil {
		container = *session.Container
	}
	stream, err := ctl.runtime.Stream(ctx, session.ClusterID, session.Namespace, session.Pod, container, session.Follow, session.TailLines, session.Previous)
	if err != nil {
		_ = write(podLogFrame{Type: "error", Message: err.Error()})
		return
	}
	defer func() { _ = stream.Close() }()
	buffer := make([]byte, 4096)
	for {
		count, readErr := stream.Read(buffer)
		if count > 0 {
			if err := write(podLogFrame{Type: "chunk", Data: string(buffer[:count])}); err != nil {
				return
			}
		}
		if readErr == nil {
			continue
		}
		if errors.Is(readErr, io.EOF) {
			_ = write(podLogFrame{Type: "eof"})
			return
		}
		if errors.Is(readErr, context.Canceled) {
			return
		}
		_ = write(podLogFrame{Type: "error", Message: readErr.Error()})
		return
	}
}

type podLogFrame struct {
	Type    string `json:"type"`
	Data    string `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

func podStreamSameOrigin(request *http.Request) bool {
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.TrimSpace(request.Host)
	if forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-Host"), ",")[0]); forwarded != "" {
		host = forwarded
	}
	if strings.EqualFold(parsed.Host, host) {
		return true
	}
	requestHost, err := url.Parse("http://" + host)
	return err == nil && strings.EqualFold(parsed.Hostname(), requestHost.Hostname())
}
