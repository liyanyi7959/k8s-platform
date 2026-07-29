package http

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
	"k8s-platform-backend/pkg/resp"
)

// DeployPlanController is the HTTP adapter for provisioning-plan definition
// and lifecycle management. Execution endpoints stay with the legacy runtime
// adapter until task orchestration is migrated.
type DeployPlanController struct {
	svc *provisionapp.DeployPlanService
}

func NewDeployPlanController(svc *provisionapp.DeployPlanService) *DeployPlanController {
	return &DeployPlanController{svc: svc}
}

func (dc *DeployPlanController) List(c *gin.Context) {
	data, err := dc.svc.List(c.Request.Context(), provisionapp.ListDeployPlansRequest{
		Page: parseInt(c.Query("page"), 1), PageSize: parseInt(c.Query("page_size"), 20),
		Keyword: c.Query("keyword"), Status: c.Query("status"),
	})
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployPlanController) Create(c *gin.Context) {
	var req provisionapp.CreateDeployPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := dc.svc.Create(c.Request.Context(), req, provisioningCurrentUserID(c))
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (dc *DeployPlanController) Get(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	data, err := dc.svc.Get(c.Request.Context(), id)
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployPlanController) Update(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	var req provisionapp.UpdateDeployPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := dc.svc.Update(c.Request.Context(), id, req); err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployPlanController) Delete(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	if err := dc.svc.Delete(c.Request.Context(), id); err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func provisioningResourceID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return 0, false
	}
	return id, true
}

func provisioningCurrentUserID(c *gin.Context) uint64 {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims == nil || claims.UserID <= 0 {
		return 0
	}
	return uint64(claims.UserID)
}
