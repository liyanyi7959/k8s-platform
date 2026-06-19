package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type DeployController struct {
	svc *service.DeployService
}

func NewDeployController(svc *service.DeployService) *DeployController {
	return &DeployController{svc: svc}
}

func (dc *DeployController) ListServers(c *gin.Context) {
	data, err := dc.svc.ListServers(c.Request.Context(), service.ListDeployServersRequest{Page: parseInt(c.Query("page"), 1), PageSize: parseInt(c.Query("page_size"), 20), Keyword: c.Query("keyword"), Status: c.Query("status")})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployController) CreateServer(c *gin.Context) {
	var req service.CreateDeployServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := dc.svc.CreateServer(c.Request.Context(), req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (dc *DeployController) GetServer(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	data, err := dc.svc.GetServer(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployController) UpdateServer(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req service.UpdateDeployServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := dc.svc.UpdateServer(c.Request.Context(), id, req); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployController) DeleteServer(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := dc.svc.DeleteServer(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployController) TestSSH(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	data, err := dc.svc.ProbeServerSSH(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployController) ListCredentials(c *gin.Context) {
	data, err := dc.svc.ListCredentials(c.Request.Context(), parseInt(c.Query("page"), 1), parseInt(c.Query("page_size"), 20))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployController) CreateCredential(c *gin.Context) {
	var req service.CreateSSHCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := dc.svc.CreateCredential(c.Request.Context(), req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (dc *DeployController) DeleteCredential(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := dc.svc.DeleteCredential(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployController) ListPlans(c *gin.Context) {
	data, err := dc.svc.ListPlans(c.Request.Context(), service.ListDeployPlansRequest{Page: parseInt(c.Query("page"), 1), PageSize: parseInt(c.Query("page_size"), 20), Keyword: c.Query("keyword"), Status: c.Query("status")})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployController) CreatePlan(c *gin.Context) {
	var req service.CreateDeployPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := dc.svc.CreatePlan(c.Request.Context(), req, currentUserID(c))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (dc *DeployController) GetPlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	data, err := dc.svc.GetPlan(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployController) DeletePlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := dc.svc.DeletePlan(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployController) ExecutePlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	taskID, err := dc.svc.ExecutePlan(c.Request.Context(), id, currentUserID(c))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"task_id": taskID})
}

func (dc *DeployController) CancelPlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := dc.svc.CancelPlan(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployController) RetryPlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	taskID, err := dc.svc.RetryPlan(c.Request.Context(), id, currentUserID(c))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"task_id": taskID})
}

func parseUintParam(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return 0, false
	}
	return id, true
}

func currentUserID(c *gin.Context) uint64 {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims == nil || claims.UserID <= 0 {
		return 0
	}
	return uint64(claims.UserID)
}
