package http

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	"k8s-platform-backend/pkg/resp"
)

// TaskController owns deployment task reads and log streaming independently
// from the retained execution runtime implementation.
type TaskController struct{ svc *provisionapp.TaskService }

func NewTaskController(svc *provisionapp.TaskService) *TaskController {
	return &TaskController{svc: svc}
}

func (tc *TaskController) Get(c *gin.Context) {
	taskID, ok := deploymentTaskID(c)
	if !ok {
		return
	}
	task, err := tc.svc.Get(c.Request.Context(), taskID)
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, task)
}

func (tc *TaskController) Logs(c *gin.Context) {
	taskID, ok := deploymentTaskID(c)
	if !ok {
		return
	}
	offset, limit, stepKey := parseInt(c.Query("offset"), 0), parseInt(c.Query("limit"), 200), c.Query("step_key")
	entries, err := tc.svc.Logs(c.Request.Context(), taskID, offset, limit, stepKey)
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	logs := make([]string, len(entries))
	for index, entry := range entries {
		logs[index] = entry.Content
	}
	resp.OK(c, gin.H{"task_id": taskID, "logs": logs, "entries": entries, "total": len(logs), "step_key": stepKey})
}

func (tc *TaskController) LogsSSE(c *gin.Context) {
	taskID, ok := deploymentTaskID(c)
	if !ok {
		return
	}
	stepKey := c.Query("step_key")
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	offset := 0
	if lastEventID := c.GetHeader("Last-Event-ID"); lastEventID != "" {
		_, _ = fmt.Sscanf(lastEventID, "%d", &offset)
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	tc.sendLogs(c, taskID, stepKey, &offset)
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			task, err := tc.svc.Get(c.Request.Context(), taskID)
			if err != nil {
				fmt.Fprint(c.Writer, "event: done\ndata: {\"status\":\"not_found\"}\n\n")
				c.Writer.Flush()
				return
			}
			tc.sendLogs(c, taskID, stepKey, &offset)
			if provisionapp.IsTerminalDeploymentStatus(task.Status) {
				fmt.Fprintf(c.Writer, "event: done\ndata: {\"status\":%q,\"message\":%q}\n\n", task.Status, deploymentTaskMessage(task.Message))
				c.Writer.Flush()
				return
			}
			fmt.Fprint(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()
		}
	}
}

func (tc *TaskController) sendLogs(c *gin.Context, taskID int64, stepKey string, offset *int) {
	entries, err := tc.svc.Logs(c.Request.Context(), taskID, *offset, 100, stepKey)
	if err != nil {
		return
	}
	for _, entry := range entries {
		payload, err := json.Marshal(gin.H{"log": entry.Content, "timestamp": entry.CreatedAt.UTC().Format(time.RFC3339Nano)})
		if err == nil {
			fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		}
	}
	*offset += len(entries)
	c.Writer.Flush()
}

func deploymentTaskID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("taskId"), 10, 64)
	if err != nil || id <= 0 {
		resp.Fail(c, 4000, "参数错误")
		return 0, false
	}
	return id, true
}

func deploymentTaskMessage(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
