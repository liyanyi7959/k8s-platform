package http

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	"k8s-platform-backend/pkg/resp"
)

// AutomationTaskController exposes the platform task centre through
// provisioning/automation routes without retaining a legacy HTTP adapter.
type AutomationTaskController struct {
	service *provisionapp.AutomationTaskService
}

func NewAutomationTaskController(service *provisionapp.AutomationTaskService) *AutomationTaskController {
	return &AutomationTaskController{service: service}
}

func (ctl *AutomationTaskController) List(c *gin.Context) {
	if ctl == nil || ctl.service == nil {
		resp.Fail(c, 5000, "automation task service is unavailable")
		return
	}
	value, err := ctl.service.List(provisionapp.AutomationTaskListRequest{
		Page: automationPageValue(c, "page", 1), PageSize: automationPageValue(c, "page_size", 20), Type: strings.TrimSpace(c.Query("type")),
		Status: strings.TrimSpace(c.Query("status")), Keyword: strings.TrimSpace(c.Query("keyword")), SortBy: strings.TrimSpace(c.Query("sort_by")), Order: strings.TrimSpace(c.Query("order")),
	})
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, value)
}

func (ctl *AutomationTaskController) Get(c *gin.Context) {
	id, ok := automationTaskID(c)
	if !ok {
		return
	}
	task, err := ctl.service.Get(id)
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, task)
}

func (ctl *AutomationTaskController) Logs(c *gin.Context) {
	id, ok := automationTaskID(c)
	if !ok {
		return
	}
	logs, err := ctl.service.Logs(id, automationPageValue(c, "offset", 0), automationPageValue(c, "limit", 200))
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, gin.H{"list": logs})
}

func (ctl *AutomationTaskController) Cancel(c *gin.Context) {
	id, ok := automationTaskID(c)
	if !ok {
		return
	}
	if err := ctl.service.Cancel(id); err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id, "status": "canceled"})
}

func automationTaskID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		resp.Fail(c, 4000, "invalid params")
		return 0, false
	}
	return id, true
}

func automationPageValue(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}
