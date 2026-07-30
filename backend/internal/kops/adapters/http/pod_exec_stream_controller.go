package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"k8s.io/client-go/tools/remotecommand"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

type PodExecStreamRuntime interface {
	Stream(context.Context, uint64, string, string, string, []string, bool, io.Reader, io.Writer, io.Writer, remotecommand.TerminalSizeQueue) error
}

type PodExecStreamController struct {
	runtime  PodExecStreamRuntime
	sessions *kopsapp.ExecSessionStore
}

func NewPodExecStreamController(runtime PodExecStreamRuntime, sessions *kopsapp.ExecSessionStore) *PodExecStreamController {
	return &PodExecStreamController{runtime: runtime, sessions: sessions}
}

func (ctl *PodExecStreamController) Stream(c *gin.Context) {
	if ctl == nil || ctl.runtime == nil || ctl.sessions == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	sessionID := streamTicketID(c)
	if sessionID == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	pending, ok := ctl.sessions.Get(sessionID)
	if !ok || (pending.Kind != "" && pending.Kind != "pod") {
		resp.Fail(c, 4040, "not found")
		return
	}
	upgrader := websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 4096, CheckOrigin: podStreamSameOrigin}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()
	session, ok := ctl.sessions.Take(sessionID)
	if !ok || (session.Kind != "" && session.Kind != "pod") {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "session not found"), time.Now().Add(3*time.Second))
		return
	}
	tty := true
	if session.TTY != nil {
		tty = *session.TTY
	}
	command := session.Command
	if len(command) == 0 {
		command = []string{"sh"}
	}
	inputReader, inputWriter := io.Pipe()
	defer func() { _ = inputReader.Close() }()
	resizeCh := make(chan remotecommand.TerminalSize, 8)
	resizeQueue := &podExecSizeQueue{ch: resizeCh}
	var writeMu sync.Mutex
	write := func(channel byte, value []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		payload := make([]byte, len(value)+1)
		payload[0] = channel
		copy(payload[1:], value)
		return conn.WriteMessage(websocket.BinaryMessage, payload)
	}
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	var closeOnce sync.Once
	stop := func() { closeOnce.Do(func() { cancel(); _ = inputWriter.Close(); close(resizeCh) }) }
	defer stop()
	go podExecReadFrames(conn, inputWriter, resizeCh, stop)
	container := ""
	if session.Container != nil {
		container = *session.Container
	}
	err = ctl.runtime.Stream(ctx, session.ClusterID, session.Namespace, session.Pod, container, command, tty, inputReader, podExecWriter{write: func(value []byte) error { return write(1, value) }}, podExecWriter{write: func(value []byte) error { return write(2, value) }}, resizeQueue)
	if err != nil && !errors.Is(err, context.Canceled) {
		_ = write(3, []byte(err.Error()))
		writeMu.Lock()
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, podExecCloseMessage(err)), time.Now().Add(3*time.Second))
		writeMu.Unlock()
	}
}

type podExecFrame struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

func podExecReadFrames(conn *websocket.Conn, writer *io.PipeWriter, sizes chan<- remotecommand.TerminalSize, stop func()) {
	defer stop()
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.BinaryMessage && messageType != websocket.TextMessage || len(payload) == 0 {
			continue
		}
		var frame podExecFrame
		if messageType == websocket.TextMessage && json.Unmarshal(payload, &frame) == nil && frame.Type != "" {
			switch strings.ToLower(strings.TrimSpace(frame.Type)) {
			case "stdin":
				if frame.Data != "" {
					if _, err := writer.Write([]byte(frame.Data)); err != nil {
						return
					}
				}
				continue
			case "resize":
				if frame.Cols > 0 && frame.Rows > 0 {
					select {
					case sizes <- remotecommand.TerminalSize{Width: uint16(frame.Cols), Height: uint16(frame.Rows)}:
					default:
					}
				}
				continue
			default:
				continue
			}
		}
		if _, err := writer.Write(payload); err != nil {
			return
		}
	}
}

type podExecWriter struct{ write func([]byte) error }

func (w podExecWriter) Write(value []byte) (int, error) {
	if w.write == nil {
		return 0, io.ErrClosedPipe
	}
	if err := w.write(value); err != nil {
		return 0, err
	}
	return len(value), nil
}

type podExecSizeQueue struct {
	ch <-chan remotecommand.TerminalSize
}

func (q *podExecSizeQueue) Next() *remotecommand.TerminalSize {
	if q == nil || q.ch == nil {
		return nil
	}
	value, ok := <-q.ch
	if !ok {
		return nil
	}
	return &value
}
func podExecCloseMessage(err error) string {
	value := strings.TrimSpace(err.Error())
	for len(value) > 120 {
		value = value[:len(value)-1]
	}
	return value
}
