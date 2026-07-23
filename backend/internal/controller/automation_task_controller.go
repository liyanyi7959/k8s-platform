package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

// AutomationTaskController exposes the shared task lifecycle used by deployment,
// inspection, remediation and verification executors. Domain-specific endpoints
// remain responsible for creating a task and executing it.
type AutomationTaskController struct {
	svc *service.TaskService
}

func NewAutomationTaskController(svc *service.TaskService) *AutomationTaskController {
	return &AutomationTaskController{svc: svc}
}

func (ctl *AutomationTaskController) List(c *gin.Context) {
	if ctl == nil || ctl.svc == nil {
		resp.Fail(c, 5000, "自动化任务服务不可用")
		return
	}
	req := service.ListTasksRequest{
		Page:     queryInt(c, "page", 1),
		PageSize: queryInt(c, "page_size", 20),
		Type:     strings.TrimSpace(c.Query("type")),
		Status:   strings.TrimSpace(c.Query("status")),
		Keyword:  strings.TrimSpace(c.Query("keyword")),
		SortBy:   strings.TrimSpace(c.Query("sort_by")),
		Order:    strings.TrimSpace(c.Query("order")),
	}
	resp.OK(c, ctl.svc.List(req))
}

func (ctl *AutomationTaskController) Get(c *gin.Context) {
	id, ok := automationTaskID(c)
	if !ok {
		return
	}
	task, found := ctl.svc.Get(id)
	if !found {
		resp.Fail(c, 4040, "任务不存在")
		return
	}
	resp.OK(c, task)
}

func (ctl *AutomationTaskController) Logs(c *gin.Context) {
	id, ok := automationTaskID(c)
	if !ok {
		return
	}
	logs, found := ctl.svc.Logs(id, queryInt(c, "offset", 0), queryInt(c, "limit", 200))
	if !found {
		resp.Fail(c, 4040, "任务不存在")
		return
	}
	resp.OK(c, gin.H{"list": logs})
}

func (ctl *AutomationTaskController) Cancel(c *gin.Context) {
	id, ok := automationTaskID(c)
	if !ok {
		return
	}
	if err := ctl.svc.Cancel(id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id, "status": service.TaskCanceled})
}

func automationTaskID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		resp.Fail(c, 4000, "参数错误")
		return 0, false
	}
	return id, true
}

func queryInt(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}
