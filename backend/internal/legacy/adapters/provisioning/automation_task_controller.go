package provisioning

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/legacy/service"
	"k8s-platform-backend/pkg/resp"
)

// AutomationTaskController adapts the retained shared task store to the
// automation HTTP routes until that store receives its own bounded context.
type AutomationTaskController struct{ tasks *service.TaskService }

func NewAutomationTaskController(tasks *service.TaskService) *AutomationTaskController {
	return &AutomationTaskController{tasks: tasks}
}

func (ctl *AutomationTaskController) List(c *gin.Context) {
	if ctl == nil || ctl.tasks == nil {
		resp.Fail(c, 5000, "automation task service is unavailable")
		return
	}
	resp.OK(c, ctl.tasks.List(service.ListTasksRequest{
		Page: pageValue(c, "page", 1), PageSize: pageValue(c, "page_size", 20), Type: strings.TrimSpace(c.Query("type")),
		Status: strings.TrimSpace(c.Query("status")), Keyword: strings.TrimSpace(c.Query("keyword")), SortBy: strings.TrimSpace(c.Query("sort_by")), Order: strings.TrimSpace(c.Query("order")),
	}))
}

func (ctl *AutomationTaskController) Get(c *gin.Context) {
	id, ok := automationTaskID(c)
	if !ok {
		return
	}
	task, found := ctl.tasks.Get(id)
	if !found {
		resp.Fail(c, 4040, "task not found")
		return
	}
	resp.OK(c, task)
}

func (ctl *AutomationTaskController) Logs(c *gin.Context) {
	id, ok := automationTaskID(c)
	if !ok {
		return
	}
	logs, found := ctl.tasks.Logs(id, pageValue(c, "offset", 0), pageValue(c, "limit", 200))
	if !found {
		resp.Fail(c, 4040, "task not found")
		return
	}
	resp.OK(c, gin.H{"list": logs})
}

func (ctl *AutomationTaskController) Cancel(c *gin.Context) {
	id, ok := automationTaskID(c)
	if !ok {
		return
	}
	if err := ctl.tasks.Cancel(id); err != nil {
		if errors.Is(err, service.ErrTaskNotFound) {
			resp.Fail(c, 4040, "task not found")
			return
		}
		if errors.Is(err, service.ErrTaskCannotCancel) {
			resp.Fail(c, 4090, "task cannot be canceled")
			return
		}
		resp.Fail(c, 5000, "internal error")
		return
	}
	resp.OK(c, gin.H{"id": id, "status": service.TaskCanceled})
}

func automationTaskID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		resp.Fail(c, 4000, "invalid params")
		return 0, false
	}
	return id, true
}

func pageValue(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}
