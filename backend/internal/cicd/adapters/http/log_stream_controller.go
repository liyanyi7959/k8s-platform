package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"k8s-platform-backend/pkg/resp"
)

// CICDLogStreamRuntime provides streaming access to CICD run logs.
type CICDLogStreamRuntime interface {
	StreamRunLogs(ctx context.Context, runID uint64, follow bool) (io.ReadCloser, error)
}

type CICDLogStreamController struct {
	runtime CICDLogStreamRuntime
}

func NewCICDLogStreamController(runtime CICDLogStreamRuntime) *CICDLogStreamController {
	return &CICDLogStreamController{runtime: runtime}
}

// Stream handles a WebSocket connection and pushes CICD run logs in real time.
// Route: GET /streams/v2/cicd-run-logs/:run_id
func (ctl *CICDLogStreamController) Stream(c *gin.Context) {
	if ctl == nil || ctl.runtime == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	runIDStr := strings.TrimSpace(c.Param("run_id"))
	runID, err := strconv.ParseUint(runIDStr, 10, 64)
	if err != nil || runID == 0 {
		resp.Fail(c, 4000, "invalid run id")
		return
	}
	follow := c.Query("follow") != "false"

	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin:     cicdLogStreamSameOrigin,
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Read client messages to detect disconnection.
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	stream, err := ctl.runtime.StreamRunLogs(ctx, runID, follow)
	if err != nil {
		_ = writeLogFrame(conn, "error", err.Error())
		return
	}
	defer func() { _ = stream.Close() }()

	var writeMu sync.Mutex
	buffer := make([]byte, 4096)
	for {
		count, readErr := stream.Read(buffer)
		if count > 0 {
			writeMu.Lock()
			if err := conn.WriteJSON(logFrame{Type: "chunk", Data: string(buffer[:count])}); err != nil {
				writeMu.Unlock()
				return
			}
			writeMu.Unlock()
		}
		if readErr == nil {
			continue
		}
		if errors.Is(readErr, io.EOF) {
			_ = writeLogFrame(conn, "eof", "")
			return
		}
		if errors.Is(readErr, context.Canceled) {
			return
		}
		_ = writeLogFrame(conn, "error", readErr.Error())
		return
	}
}

type logFrame struct {
	Type    string `json:"type"`
	Data    string `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

func writeLogFrame(conn *websocket.Conn, frameType, message string) error {
	return conn.WriteJSON(logFrame{Type: frameType, Message: message})
}

func cicdLogStreamSameOrigin(request *http.Request) bool {
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		return false
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
