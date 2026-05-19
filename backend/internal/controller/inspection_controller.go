package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/auth"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type InspectionController struct {
	svc *service.InspectionService
}

func NewInspectionController(svc *service.InspectionService) *InspectionController {
	return &InspectionController{svc: svc}
}

// ── Dashboard ─────────────────────────────────────────────────

// GetDashboard 获取巡检中心概览数据
// @Summary 巡检中心概览
// @Tags 巡检
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Result
// @Router /automation/inspections/overview [get]
func (ctl *InspectionController) GetDashboard(c *gin.Context) {
	data, err := ctl.svc.GetDashboard(c.Request.Context())
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// ── Run ───────────────────────────────────────────────────────

type runInspectionReq struct {
	TemplateID     uint64                        `json:"template_id" binding:"required"`
	ServerIDs      []uint64                      `json:"server_ids" binding:"required,min=1"`
	TargetSnapshot []service.InspectionTargetRef `json:"target_snapshot"`
	Options        service.RunInspectionOptions  `json:"options"`
}

// RunNow 立即执行巡检
// @Summary 立即执行巡检
// @Tags 巡检
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body runInspectionReq true "执行参数"
// @Success 200 {object} resp.Result
// @Router /automation/inspections/run [post]
func (ctl *InspectionController) RunNow(c *gin.Context) {
	var req runInspectionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误: "+err.Error())
		return
	}
	operatorName := inspectionCurrentUsername(c)
	reportID, taskID, err := ctl.svc.RunInspection(c.Request.Context(), service.RunInspectionRequest{
		TemplateID:     req.TemplateID,
		ServerIDs:      req.ServerIDs,
		TargetSnapshot: req.TargetSnapshot,
		Options:        req.Options,
		OperatorName:   operatorName,
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"report_id": reportID, "task_id": taskID})
}

// ── Templates ─────────────────────────────────────────────────

// ListTemplates 获取巡检模板列表
// @Summary 巡检模板列表
// @Tags 巡检
// @Security BearerAuth
// @Produce json
// @Param keyword query string false "关键字"
// @Param category query string false "分类"
// @Success 200 {object} resp.Result
// @Router /automation/inspection-templates [get]
func (ctl *InspectionController) ListTemplates(c *gin.Context) {
	data, err := ctl.svc.ListTemplates(c.Request.Context(), service.ListInspectionTemplatesRequest{
		Keyword:  c.Query("keyword"),
		Category: c.Query("category"),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

// GetTemplate 获取巡检模板详情
// @Summary 巡检模板详情
// @Tags 巡检
// @Security BearerAuth
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} resp.Result
// @Router /automation/inspection-templates/{id} [get]
func (ctl *InspectionController) GetTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "无效的ID")
		return
	}
	data, err := ctl.svc.GetTemplate(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type saveTemplateReq struct {
	ID          *uint64                   `json:"id"`
	Name        string                    `json:"name" binding:"required"`
	Description string                    `json:"description"`
	Category    string                    `json:"category"`
	Version     string                    `json:"version"`
	Tags        []string                  `json:"tags"`
	Recommended bool                      `json:"recommended"`
	Checks      []service.InspectionCheck `json:"checks"`
}

// SaveTemplate 创建或更新巡检模板
// @Summary 保存巡检模板
// @Tags 巡检
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body saveTemplateReq true "模板数据"
// @Success 200 {object} resp.Result
// @Router /automation/inspection-templates [post]
func (ctl *InspectionController) SaveTemplate(c *gin.Context) {
	var req saveTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误: "+err.Error())
		return
	}
	operatorID := inspectionCurrentUserID(c)
	ver := req.Version
	if ver == "" {
		ver = "v1.0"
	}
	cat := req.Category
	if cat == "" {
		cat = "baseline"
	}
	id, err := ctl.svc.SaveTemplate(c.Request.Context(), service.SaveInspectionTemplateRequest{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Category:    cat,
		Version:     ver,
		Tags:        req.Tags,
		Recommended: req.Recommended,
		Checks:      req.Checks,
		OperatorID:  operatorID,
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// PatchTemplate 更新巡检模板（与 SaveTemplate 合并路由）
func (ctl *InspectionController) PatchTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "无效的ID")
		return
	}
	var req saveTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误: "+err.Error())
		return
	}
	req.ID = &id
	operatorID2 := inspectionCurrentUserID(c)
	ver := req.Version
	if ver == "" {
		ver = "v1.0"
	}
	cat := req.Category
	if cat == "" {
		cat = "baseline"
	}
	_, err2 := ctl.svc.SaveTemplate(c.Request.Context(), service.SaveInspectionTemplateRequest{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Category:    cat,
		Version:     ver,
		Tags:        req.Tags,
		Recommended: req.Recommended,
		Checks:      req.Checks,
		OperatorID:  operatorID2,
	})
	if err2 != nil {
		WriteServiceErr(c, err2)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// DeleteTemplate 删除巡检模板
// @Summary 删除巡检模板
// @Tags 巡检
// @Security BearerAuth
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} resp.Result
// @Router /automation/inspection-templates/{id} [delete]
func (ctl *InspectionController) DeleteTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "无效的ID")
		return
	}
	if err := ctl.svc.DeleteTemplate(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// ── Schedules ─────────────────────────────────────────────────

// ListSchedules 获取巡检计划列表
// @Summary 巡检计划列表
// @Tags 巡检
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Result
// @Router /automation/inspection-schedules [get]
func (ctl *InspectionController) ListSchedules(c *gin.Context) {
	data, err := ctl.svc.ListSchedules(c.Request.Context(), service.ListInspectionSchedulesRequest{
		Keyword: c.Query("keyword"),
		Status:  c.Query("status"),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

type saveScheduleReq struct {
	ID         *uint64  `json:"id"`
	Name       string   `json:"name" binding:"required"`
	TemplateID uint64   `json:"template_id" binding:"required"`
	Cron       string   `json:"cron" binding:"required"`
	Status     string   `json:"status"`
	ServerIDs  []uint64 `json:"server_ids"`
}

// SaveSchedule 创建或更新巡检计划
// @Summary 保存巡检计划
// @Tags 巡检
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body saveScheduleReq true "计划数据"
// @Success 200 {object} resp.Result
// @Router /automation/inspection-schedules [post]
func (ctl *InspectionController) SaveSchedule(c *gin.Context) {
	var req saveScheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误: "+err.Error())
		return
	}
	id, err := ctl.svc.SaveSchedule(c.Request.Context(), service.SaveInspectionScheduleRequest{
		ID:         req.ID,
		Name:       req.Name,
		TemplateID: req.TemplateID,
		Cron:       req.Cron,
		Status:     req.Status,
		ServerIDs:  req.ServerIDs,
		OperatorID: inspectionCurrentUserID(c),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// PatchSchedule 更新巡检计划
func (ctl *InspectionController) PatchSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "无效的ID")
		return
	}
	var req saveScheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误: "+err.Error())
		return
	}
	req.ID = &id
	_, err2 := ctl.svc.SaveSchedule(c.Request.Context(), service.SaveInspectionScheduleRequest{
		ID:         req.ID,
		Name:       req.Name,
		TemplateID: req.TemplateID,
		Cron:       req.Cron,
		Status:     req.Status,
		ServerIDs:  req.ServerIDs,
		OperatorID: inspectionCurrentUserID(c),
	})
	if err2 != nil {
		WriteServiceErr(c, err2)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// DeleteSchedule 删除巡检计划
func (ctl *InspectionController) DeleteSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "无效的ID")
		return
	}
	if err := ctl.svc.DeleteSchedule(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// ── Reports ───────────────────────────────────────────────────

// ListReports 获取巡检报告列表
// @Summary 巡检报告列表
// @Tags 巡检
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Result
// @Router /automation/inspection-reports [get]
func (ctl *InspectionController) ListReports(c *gin.Context) {
	var tplID *uint64
	if raw := c.Query("template_id"); raw != "" {
		if v, err := strconv.ParseUint(raw, 10, 64); err == nil && v > 0 {
			tplID = &v
		}
	}
	data, err := ctl.svc.ListReports(c.Request.Context(), service.ListInspectionReportsRequest{
		Keyword:    c.Query("keyword"),
		RiskLevel:  c.Query("risk_level"),
		TemplateID: tplID,
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

// GetReport 获取巡检报告详情
// @Summary 巡检报告详情
// @Tags 巡检
// @Security BearerAuth
// @Produce json
// @Param id path int true "报告ID"
// @Success 200 {object} resp.Result
// @Router /automation/inspection-reports/{id} [get]
func (ctl *InspectionController) GetReport(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "无效的ID")
		return
	}
	data, err := ctl.svc.GetReport(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// ── 内部辅助函数 ──────────────────────────────────────────────

func inspectionCurrentUserID(c *gin.Context) uint64 {
	if claims, ok := middleware.GetClaims(c); ok && claims != nil && claims.UserID > 0 {
		return uint64(claims.UserID)
	}
	return 1
}

func inspectionCurrentUsername(c *gin.Context) string {
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		return claims.Username
	}
	return "system"
}

// 确保 auth 包被引用（用于 Claims 类型检查）
var _ *auth.Claims
