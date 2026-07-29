package http

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/incident/application"
	"k8s-platform-backend/internal/incident/domain"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/pkg/resp"
)

type LegacyController struct {
	monitoring *application.MonitoringService
	incidents  *application.Service
}

func NewLegacyController(monitoring *application.MonitoringService, incidents *application.Service) *LegacyController {
	return &LegacyController{monitoring: monitoring, incidents: incidents}
}

func (ctl *LegacyController) ListAlertRules(c *gin.Context) {
	data, err := ctl.monitoring.ListAlertRules(c.Request.Context(), parseLegacyInt(c.Query("page"), 1), parseLegacyInt(c.Query("page_size"), 20))
	if err != nil {
		writeLegacyError(c, err)
		return
	}
	resp.OK(c, data)
}
func (ctl *LegacyController) CreateAlertRule(c *gin.Context) {
	var request application.AlertRuleRequest
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := ctl.monitoring.CreateAlertRule(c.Request.Context(), request)
	if err != nil {
		writeLegacyError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}
func (ctl *LegacyController) UpdateAlertRule(c *gin.Context) {
	id, ok := legacyID(c)
	if !ok {
		return
	}
	var request application.AlertRuleRequest
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.monitoring.UpdateAlertRule(c.Request.Context(), id, request); err != nil {
		writeLegacyError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *LegacyController) DeleteAlertRule(c *gin.Context) {
	id, ok := legacyID(c)
	if !ok {
		return
	}
	if err := ctl.monitoring.DeleteAlertRule(c.Request.Context(), id); err != nil {
		writeLegacyError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *LegacyController) ToggleAlertRule(c *gin.Context) {
	id, ok := legacyID(c)
	if !ok {
		return
	}
	var request struct {
		Enabled bool `json:"enabled"`
	}
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.monitoring.ToggleAlertRule(c.Request.Context(), id, request.Enabled); err != nil {
		writeLegacyError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *LegacyController) IngestAlertmanager(c *gin.Context) {
	var payload application.AlertmanagerWebhook
	if c.ShouldBindJSON(&payload) != nil {
		resp.Fail(c, 4000, "Alertmanager payload 格式错误")
		return
	}
	ids, err := ctl.monitoring.IngestAlertmanager(c.Request.Context(), payload)
	if err != nil {
		writeLegacyError(c, err)
		return
	}
	resp.OK(c, gin.H{"incident_ids": ids})
}
func (ctl *LegacyController) ListIncidents(c *gin.Context) {
	data, err := ctl.monitoring.ListIncidents(c.Request.Context(), parseLegacyInt(c.Query("page"), 1), parseLegacyInt(c.Query("page_size"), 20), c.Query("status"))
	if err != nil {
		writeLegacyError(c, err)
		return
	}
	resp.OK(c, data)
}
func (ctl *LegacyController) GetIncident(c *gin.Context) {
	id, ok := legacyID(c)
	if !ok {
		return
	}
	data, timeline, err := ctl.monitoring.GetIncident(c.Request.Context(), id)
	if err != nil {
		writeLegacyError(c, err)
		return
	}
	resp.OK(c, gin.H{"incident": data, "timeline": timeline})
}
func (ctl *LegacyController) TransitionIncident(c *gin.Context) {
	id, ok := legacyID(c)
	if !ok {
		return
	}
	var request struct {
		Action string `json:"action"`
		Note   string `json:"note"`
	}
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	command, ok := legacyCommand(request.Action)
	if !ok {
		resp.Fail(c, 4000, "不支持的事件动作")
		return
	}
	actor := legacyActor(c)
	detail, err := ctl.incidents.Get(c.Request.Context(), id)
	if err == nil {
		_, err = ctl.incidents.Execute(c.Request.Context(), id, command, application.CommandRequest{ExpectedVersion: detail.Incident.Version, Note: request.Note}, actor)
	}
	if err != nil {
		writeLegacyError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *LegacyController) LinkAIConversation(c *gin.Context) {
	id, ok := legacyID(c)
	if !ok {
		return
	}
	var request struct {
		ConversationID uint64 `json:"conversation_id"`
	}
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	actor := legacyActor(c)
	if err := ctl.monitoring.LinkAIConversation(c.Request.Context(), id, request.ConversationID, actor.ID, actor.Name); err != nil {
		writeLegacyError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *LegacyController) LinkAIProposal(c *gin.Context) {
	id, ok := legacyID(c)
	if !ok {
		return
	}
	var request struct {
		ProposalID uint64 `json:"proposal_id"`
	}
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	actor := legacyActor(c)
	if err := ctl.monitoring.LinkAIProposal(c.Request.Context(), id, request.ProposalID, actor.ID, actor.Name); err != nil {
		writeLegacyError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func legacyID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return 0, false
	}
	return id, true
}
func parseLegacyInt(value string, fallback int) int {
	result, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return result
}
func legacyActor(c *gin.Context) domain.Actor {
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		return domain.Actor{ID: uint64(claims.UserID), Name: strings.TrimSpace(claims.Username)}
	}
	return domain.Actor{}
}
func legacyCommand(action string) (domain.Command, bool) {
	command, ok := map[string]domain.Command{"acknowledge": domain.CommandAcknowledge, "diagnose": domain.CommandDiagnose, "await_approval": domain.CommandRequestApproval, "execute": domain.CommandStartExecution, "verify": domain.CommandStartVerification, "resolve": domain.CommandResolve}[strings.ToLower(strings.TrimSpace(action))]
	return command, ok
}
func writeLegacyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		resp.Fail(c, 4000, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		resp.Fail(c, 4040, "事件不存在")
	case errors.Is(err, domain.ErrVersionConflict):
		resp.Fail(c, 4090, "事件已被更新，请刷新后重试")
	case errors.Is(err, domain.ErrInvalidTransition):
		resp.Fail(c, 4090, "当前状态不允许此操作")
	default:
		resp.Fail(c, 5000, "内部错误")
	}
}
