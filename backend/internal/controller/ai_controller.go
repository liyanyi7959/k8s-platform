package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type AIController struct {
	providerSvc     *service.AIProviderService
	routeSettingsSvc *service.AIRouteSettingsService
	conversationSvc *service.AIConversationService
	chatSvc         *service.AIChatService
	actionSvc       *service.AIActionService
}

func NewAIController(
	providerSvc *service.AIProviderService,
	routeSettingsSvc *service.AIRouteSettingsService,
	conversationSvc *service.AIConversationService,
	chatSvc *service.AIChatService,
	actionSvc *service.AIActionService,
) *AIController {
	return &AIController{
		providerSvc:     providerSvc,
		routeSettingsSvc: routeSettingsSvc,
		conversationSvc: conversationSvc,
		chatSvc:         chatSvc,
		actionSvc:       actionSvc,
	}
}

func (ctl *AIController) ListProviders(c *gin.Context) {
	items, err := ctl.providerSvc.ListProviders(c.Request.Context())
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, items)
}

func (ctl *AIController) CreateProvider(c *gin.Context) {
	var req service.CreateAIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := ctl.providerSvc.CreateProvider(c.Request.Context(), req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (ctl *AIController) PatchProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req service.PatchAIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.providerSvc.PatchProvider(c.Request.Context(), id, req); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (ctl *AIController) ListModels(c *gin.Context) {
	var enabledPtr *bool
	if raw := strings.TrimSpace(c.Query("enabled")); raw != "" {
		enabled := raw == "1" || strings.EqualFold(raw, "true")
		enabledPtr = &enabled
	}
	items, err := ctl.providerSvc.ListModels(c.Request.Context(), service.ListAIModelsRequest{
		ProviderID: uint64(parseInt64(c.Query("provider_id"), 0)),
		ModelType:  c.Query("model_type"),
		Enabled:    enabledPtr,
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, items)
}

func (ctl *AIController) GetRouteSettings(c *gin.Context) {
	data, err := ctl.routeSettingsSvc.Get(c.Request.Context())
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *AIController) UpdateRouteSettings(c *gin.Context) {
	var req service.UpdateAIRouteSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "鍙傛暟閿欒")
		return
	}
	data, err := ctl.routeSettingsSvc.Update(c.Request.Context(), req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *AIController) CreateModel(c *gin.Context) {
	var req service.CreateAIModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := ctl.providerSvc.CreateModel(c.Request.Context(), req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (ctl *AIController) PatchModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req service.PatchAIModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.providerSvc.PatchModel(c.Request.Context(), id, req); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (ctl *AIController) ListConversations(c *gin.Context) {
	result, err := ctl.conversationSvc.ListConversations(c.Request.Context(), service.ListAIConversationsRequest{
		Page:          parseInt(c.DefaultQuery("page", "1"), 1),
		PageSize:      parseInt(c.DefaultQuery("page_size", "20"), 20),
		ClusterID:     uint64(parseInt64(c.Query("cluster_id"), 0)),
		Status:        c.Query("status"),
		AssistantMode: c.Query("assistant_mode"),
		Keyword:       c.Query("keyword"),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, result)
}

func (ctl *AIController) GetConversation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.conversationSvc.GetConversation(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *AIController) DeleteConversation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.conversationSvc.DeleteConversation(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (ctl *AIController) CreateConversation(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || clusterID == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	var req service.CreateAIConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	var userID uint64
	var username string
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		if claims.UserID > 0 {
			userID = uint64(claims.UserID)
		}
		username = strings.TrimSpace(claims.Username)
	}

	id, err := ctl.conversationSvc.CreateConversation(c.Request.Context(), clusterID, userID, username, req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (ctl *AIController) SendChat(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || clusterID == 0 {
		resp.Fail(c, 4000, "鍙傛暟閿欒")
		return
	}

	var req service.AIChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "鍙傛暟閿欒")
		return
	}
	req.ClusterID = clusterID

	var userID uint64
	var username string
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		if claims.UserID > 0 {
			userID = uint64(claims.UserID)
		}
		username = strings.TrimSpace(claims.Username)
	}

	data, err := ctl.chatSvc.SendMessage(c.Request.Context(), userID, username, req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *AIController) CreateActionProposal(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || clusterID == 0 {
		resp.Fail(c, 4000, "鍙傛暟閿欒")
		return
	}

	var req service.CreateAIActionProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "鍙傛暟閿欒")
		return
	}

	var userID uint64
	var username string
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		if claims.UserID > 0 {
			userID = uint64(claims.UserID)
		}
		username = strings.TrimSpace(claims.Username)
	}

	data, err := ctl.actionSvc.CreateProposal(c.Request.Context(), clusterID, userID, username, req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *AIController) ConfirmActionProposal(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || clusterID == 0 {
		resp.Fail(c, 4000, "鍙傛暟閿欒")
		return
	}
	proposalID, err := strconv.ParseUint(c.Param("actionId"), 10, 64)
	if err != nil || proposalID == 0 {
		resp.Fail(c, 4000, "鍙傛暟閿欒")
		return
	}

	var req service.ConfirmAIActionProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "鍙傛暟閿欒")
		return
	}

	var userID uint64
	var username string
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		if claims.UserID > 0 {
			userID = uint64(claims.UserID)
		}
		username = strings.TrimSpace(claims.Username)
	}

	data, err := ctl.actionSvc.ConfirmProposal(c.Request.Context(), clusterID, proposalID, userID, username, req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}
