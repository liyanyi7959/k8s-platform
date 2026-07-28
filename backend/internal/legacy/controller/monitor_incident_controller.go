package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	incidentapp "k8s-platform-backend/internal/incident/application"
	"k8s-platform-backend/internal/incident/domain"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/legacy/service"
	"k8s-platform-backend/pkg/resp"
)

type MonitorIncidentController struct {
	svc         *service.MonitorIncidentService
	incidentSvc *incidentapp.Service
}

func NewMonitorIncidentController(svc *service.MonitorIncidentService, incidentSvc *incidentapp.Service) *MonitorIncidentController {
	return &MonitorIncidentController{svc: svc, incidentSvc: incidentSvc}
}

func (ctl *MonitorIncidentController) ListAlertRules(c *gin.Context) {
	data, err := ctl.svc.ListAlertRules(c.Request.Context(), parseInt(c.Query("page"), 1), parseInt(c.Query("page_size"), 20))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}
func (ctl *MonitorIncidentController) CreateAlertRule(c *gin.Context) {
	var req service.UpsertMonitorAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := ctl.svc.CreateAlertRule(c.Request.Context(), req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}
func (ctl *MonitorIncidentController) UpdateAlertRule(c *gin.Context) {
	id, ok := monitorID(c)
	if !ok {
		return
	}
	var req service.UpsertMonitorAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.svc.UpdateAlertRule(c.Request.Context(), id, req); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *MonitorIncidentController) DeleteAlertRule(c *gin.Context) {
	id, ok := monitorID(c)
	if !ok {
		return
	}
	if err := ctl.svc.DeleteAlertRule(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *MonitorIncidentController) ToggleAlertRule(c *gin.Context) {
	id, ok := monitorID(c)
	if !ok {
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.svc.ToggleAlertRule(c.Request.Context(), id, req.Enabled); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (ctl *MonitorIncidentController) IngestAlertmanager(c *gin.Context) {
	var payload service.AlertmanagerWebhook
	if err := c.ShouldBindJSON(&payload); err != nil {
		resp.Fail(c, 4000, "Alertmanager payload 格式错误")
		return
	}
	ids, err := ctl.svc.IngestAlertmanager(c.Request.Context(), payload)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"incident_ids": ids})
}
func (ctl *MonitorIncidentController) ListIncidents(c *gin.Context) {
	data, err := ctl.svc.ListIncidents(c.Request.Context(), parseInt(c.Query("page"), 1), parseInt(c.Query("page_size"), 20), c.Query("status"))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}
func (ctl *MonitorIncidentController) GetIncident(c *gin.Context) {
	id, ok := monitorID(c)
	if !ok {
		return
	}
	data, timeline, err := ctl.svc.GetIncident(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"incident": data, "timeline": timeline})
}
func (ctl *MonitorIncidentController) TransitionIncident(c *gin.Context) {
	id, ok := monitorID(c)
	if !ok {
		return
	}
	var req struct {
		Action string `json:"action"`
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	userID, username := monitorOperator(c)
	command, ok := legacyIncidentCommand(req.Action)
	if !ok {
		resp.Fail(c, 4000, "不支持的事件动作")
		return
	}
	detail, err := ctl.incidentSvc.Get(c.Request.Context(), id)
	if err == nil {
		_, err = ctl.incidentSvc.Execute(c.Request.Context(), id, command, incidentapp.CommandRequest{ExpectedVersion: detail.Incident.Version, Note: req.Note}, domain.Actor{ID: userID, Name: username})
	}
	if err != nil {
		WriteServiceErr(c, err,
			errMapping{domain.ErrNotFound, 4040, "事件不存在"},
			errMapping{domain.ErrVersionConflict, 4090, "事件已被其他用户更新，请刷新后重试"},
			errMapping{domain.ErrInvalidTransition, 4090, "当前状态不允许此操作"},
			errMapping{domain.ErrValidation, 4000, "事件处置校验失败"},
		)
		return
	}
	resp.OK[any](c, nil)
}

func legacyIncidentCommand(action string) (domain.Command, bool) {
	command, ok := map[string]domain.Command{
		"acknowledge":    domain.CommandAcknowledge,
		"diagnose":       domain.CommandDiagnose,
		"await_approval": domain.CommandRequestApproval,
		"execute":        domain.CommandStartExecution,
		"verify":         domain.CommandStartVerification,
		"resolve":        domain.CommandResolve,
	}[strings.ToLower(strings.TrimSpace(action))]
	return command, ok
}
func (ctl *MonitorIncidentController) LinkAIConversation(c *gin.Context) {
	id, ok := monitorID(c)
	if !ok {
		return
	}
	var req struct {
		ConversationID uint64 `json:"conversation_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	userID, username := monitorOperator(c)
	if err := ctl.svc.LinkAIConversation(c.Request.Context(), id, req.ConversationID, userID, username); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *MonitorIncidentController) LinkAIProposal(c *gin.Context) {
	id, ok := monitorID(c)
	if !ok {
		return
	}
	var req struct {
		ProposalID uint64 `json:"proposal_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	userID, username := monitorOperator(c)
	if err := ctl.svc.LinkAIProposal(c.Request.Context(), id, req.ProposalID, userID, username); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func monitorID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return 0, false
	}
	return id, true
}
func monitorOperator(c *gin.Context) (uint64, string) {
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		return uint64(claims.UserID), strings.TrimSpace(claims.Username)
	}
	return 0, ""
}
