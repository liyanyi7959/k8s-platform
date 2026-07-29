package http

import (
	"strings"

	"github.com/gin-gonic/gin"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	"k8s-platform-backend/pkg/resp"
)

// RuntimeController owns the deployment command HTTP surface while the
// underlying Ansible executor is still supplied through DeploymentRuntime.
type RuntimeController struct{ svc *provisionapp.RuntimeService }

func NewRuntimeController(svc *provisionapp.RuntimeService) *RuntimeController {
	return &RuntimeController{svc: svc}
}

func (rc *RuntimeController) DryRun(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	data, err := rc.svc.DryRun(c.Request.Context(), id)
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, data)
}

func (rc *RuntimeController) Preflight(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	data, err := rc.svc.Preflight(c.Request.Context(), id)
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, data)
}

func (rc *RuntimeController) SetPreflightIgnore(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	var req struct {
		Key     string `json:"key"`
		Ignored bool   `json:"ignored"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Key) == "" {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := rc.svc.SetPreflightIgnore(c.Request.Context(), id, req.Key, req.Ignored); err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (rc *RuntimeController) AnsibleConfig(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	data, err := rc.svc.AnsibleConfig(c.Request.Context(), id)
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, data)
}

func (rc *RuntimeController) Execute(c *gin.Context) {
	rc.taskResponse(c, func(id, userID uint64) (uint64, error) { return rc.svc.Execute(c.Request.Context(), id, userID) })
}
func (rc *RuntimeController) Retry(c *gin.Context) {
	rc.taskResponse(c, func(id, userID uint64) (uint64, error) { return rc.svc.Retry(c.Request.Context(), id, userID) })
}
func (rc *RuntimeController) RetryAddons(c *gin.Context) {
	rc.taskResponse(c, func(id, userID uint64) (uint64, error) { return rc.svc.RetryAddons(c.Request.Context(), id, userID) })
}

func (rc *RuntimeController) Cancel(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	if err := rc.svc.Cancel(c.Request.Context(), id); err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (rc *RuntimeController) RetryStep(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	stepKey := strings.TrimSpace(c.Param("stepKey"))
	if stepKey == "" {
		resp.Fail(c, 4000, "步骤 key 不能为空")
		return
	}
	taskID, err := rc.svc.RetryStep(c.Request.Context(), id, stepKey, provisioningCurrentUserID(c))
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, gin.H{"task_id": taskID})
}

func (rc *RuntimeController) InstallAddons(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	var req struct {
		Addons []string `json:"addons"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Addons) == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	taskID, err := rc.svc.InstallAddons(c.Request.Context(), id, req.Addons, provisioningCurrentUserID(c))
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, gin.H{"task_id": taskID})
}

func (rc *RuntimeController) LatestAddonTask(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	task, err := rc.svc.LatestAddonTask(c.Request.Context(), id)
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, gin.H{"task": task})
}

func (rc *RuntimeController) taskResponse(c *gin.Context, command func(uint64, uint64) (uint64, error)) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	taskID, err := command(id, provisioningCurrentUserID(c))
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, gin.H{"task_id": taskID})
}
