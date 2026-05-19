package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/model"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type AutomationJobController struct {
	svc *service.AutomationJobService
}

func NewAutomationJobController(svc *service.AutomationJobService) *AutomationJobController {
	return &AutomationJobController{svc: svc}
}

type createAutomationJobReq struct {
	Name         string        `json:"name"`
	Mode         string        `json:"mode"`
	Type         string        `json:"type"`
	Env          string        `json:"env"`
	Status       string        `json:"status"`
	RiskLevel    string        `json:"risk_level"`
	ApprovalMode string        `json:"approval_mode"`
	Strategy     string        `json:"strategy"`
	Concurrency  int           `json:"concurrency"`
	TimeoutSec   int           `json:"timeout_sec"`
	TemplateID   *uint64       `json:"template_id"`
	Cron         *string       `json:"cron"`
	Targets      *string       `json:"targets"`
	Limit        *string       `json:"limit"`
	Vars         model.JSONMap `json:"vars"`
	ChangeWindow *string       `json:"change_window"`
	RollbackPlan *string       `json:"rollback_plan"`
}

func (ctl *AutomationJobController) List(c *gin.Context) {
	data, err := ctl.svc.List(c.Request.Context(), service.ListAutomationJobsRequest{
		Page:         parseInt(c.Query("page"), 1),
		PageSize:     parseInt(c.Query("page_size"), 10),
		Keyword:      c.Query("keyword"),
		Mode:         c.Query("mode"),
		Type:         c.Query("type"),
		Env:          c.Query("env"),
		RiskLevel:    c.Query("risk_level"),
		ApprovalMode: c.Query("approval_mode"),
		Status:       c.Query("status"),
		SortBy:       c.Query("sort_by"),
		Order:        c.Query("order"),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *AutomationJobController) Summary(c *gin.Context) {
	data, err := ctl.svc.Summary(c.Request.Context(), service.ListAutomationJobsRequest{
		Keyword:      c.Query("keyword"),
		Mode:         c.Query("mode"),
		Type:         c.Query("type"),
		Env:          c.Query("env"),
		RiskLevel:    c.Query("risk_level"),
		ApprovalMode: c.Query("approval_mode"),
		Status:       c.Query("status"),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *AutomationJobController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.svc.Get(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *AutomationJobController) Create(c *gin.Context) {
	var req createAutomationJobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := ctl.svc.Create(c.Request.Context(), service.CreateAutomationJobRequest{
		Name:         req.Name,
		Mode:         req.Mode,
		Type:         req.Type,
		Env:          req.Env,
		Status:       req.Status,
		RiskLevel:    req.RiskLevel,
		ApprovalMode: req.ApprovalMode,
		Strategy:     req.Strategy,
		Concurrency:  req.Concurrency,
		TimeoutSec:   req.TimeoutSec,
		TemplateID:   req.TemplateID,
		Cron:         req.Cron,
		Targets:      req.Targets,
		LimitSpec:    req.Limit,
		Vars:         req.Vars,
		ChangeWindow: req.ChangeWindow,
		RollbackPlan: req.RollbackPlan,
		CreatedBy:    automationCurrentUserID(c),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

type patchAutomationJobReq struct {
	Name         *string        `json:"name"`
	Mode         *string        `json:"mode"`
	Type         *string        `json:"type"`
	Env          *string        `json:"env"`
	Status       *string        `json:"status"`
	RiskLevel    *string        `json:"risk_level"`
	ApprovalMode *string        `json:"approval_mode"`
	Strategy     *string        `json:"strategy"`
	Concurrency  *int           `json:"concurrency"`
	TimeoutSec   *int           `json:"timeout_sec"`
	TemplateID   *uint64        `json:"template_id"`
	Cron         *string        `json:"cron"`
	Targets      *string        `json:"targets"`
	Limit        *string        `json:"limit"`
	Vars         *model.JSONMap `json:"vars"`
	ChangeWindow *string        `json:"change_window"`
	RollbackPlan *string        `json:"rollback_plan"`
}

func (ctl *AutomationJobController) Patch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req patchAutomationJobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.svc.Patch(c.Request.Context(), id, service.PatchAutomationJobRequest{
		Name:         req.Name,
		Mode:         req.Mode,
		Type:         req.Type,
		Env:          req.Env,
		Status:       req.Status,
		RiskLevel:    req.RiskLevel,
		ApprovalMode: req.ApprovalMode,
		Strategy:     req.Strategy,
		Concurrency:  req.Concurrency,
		TimeoutSec:   req.TimeoutSec,
		TemplateID:   req.TemplateID,
		Cron:         req.Cron,
		Targets:      req.Targets,
		LimitSpec:    req.Limit,
		Vars:         req.Vars,
		ChangeWindow: req.ChangeWindow,
		RollbackPlan: req.RollbackPlan,
		UpdatedBy:    automationCurrentUserID(c),
	}); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

func (ctl *AutomationJobController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.svc.Delete(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

type runAutomationJobReq struct {
	ServerIDs []uint64      `json:"server_ids"`
	Version   string        `json:"version"`
	Params    model.JSONMap `json:"params"`
}

type batchUpdateAutomationJobStatusReq struct {
	IDs    []uint64 `json:"ids"`
	Status string   `json:"status"`
}

type batchDeleteAutomationJobsReq struct {
	IDs []uint64 `json:"ids"`
}

type batchRunAutomationJobsReq struct {
	IDs       []uint64      `json:"ids"`
	ServerIDs []uint64      `json:"server_ids"`
	Version   string        `json:"version"`
	Params    model.JSONMap `json:"params"`
}

func (ctl *AutomationJobController) Run(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req runAutomationJobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	taskID, err := ctl.svc.Run(c.Request.Context(), id, service.RunAutomationJobRequest{
		ServerIDs: req.ServerIDs,
		Version:   req.Version,
		Params:    req.Params,
		CreatedBy: automationCurrentUserID(c),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"task_id": taskID})
}

func (ctl *AutomationJobController) BatchUpdateStatus(c *gin.Context) {
	var req batchUpdateAutomationJobStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	updated, err := ctl.svc.BatchUpdateStatus(c.Request.Context(), service.BatchUpdateAutomationJobStatusRequest{
		IDs:       req.IDs,
		Status:    req.Status,
		UpdatedBy: automationCurrentUserID(c),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"updated": updated})
}

func (ctl *AutomationJobController) BatchDelete(c *gin.Context) {
	var req batchDeleteAutomationJobsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	deleted, err := ctl.svc.BatchDelete(c.Request.Context(), service.BatchDeleteAutomationJobsRequest{IDs: req.IDs})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"deleted": deleted})
}

func (ctl *AutomationJobController) BatchRun(c *gin.Context) {
	var req batchRunAutomationJobsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	results, err := ctl.svc.BatchRun(c.Request.Context(), service.BatchRunAutomationJobsRequest{
		IDs:       req.IDs,
		ServerIDs: req.ServerIDs,
		Version:   req.Version,
		Params:    req.Params,
		CreatedBy: automationCurrentUserID(c),
	})
	if err != nil && len(results) == 0 {
		WriteServiceErr(c, err)
		return
	}
	summary := gin.H{"submitted": 0, "failed": 0}
	for _, item := range results {
		if item.Executed {
			summary["submitted"] = summary["submitted"].(int) + 1
		} else {
			summary["failed"] = summary["failed"].(int) + 1
		}
	}
	resp.OK(c, gin.H{"results": results, "summary": summary})
}

func automationCurrentUserID(c *gin.Context) uint64 {
	if claims, ok := middleware.GetClaims(c); ok && claims != nil && claims.UserID > 0 {
		return uint64(claims.UserID)
	}
	if v, ok := c.Get("user_id"); ok {
		if id, ok2 := v.(int64); ok2 && id > 0 {
			return uint64(id)
		}
	}
	return 1
}
