package http

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	aiapp "k8s-platform-backend/internal/ai/application"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/pkg/resp"
)

// ManagementController owns AI provider, model, routing and conversation
// lifecycle HTTP commands. Chat/tool execution remains a separate runtime
// adapter until its orchestration ports are migrated.
type ManagementController struct {
	providers     *aiapp.AIProviderService
	routeSettings *aiapp.AIRouteSettingsService
	conversations *aiapp.ConversationService
}

func NewManagementController(providers *aiapp.AIProviderService, routeSettings *aiapp.AIRouteSettingsService, conversations *aiapp.ConversationService) *ManagementController {
	return &ManagementController{providers: providers, routeSettings: routeSettings, conversations: conversations}
}

func (mc *ManagementController) ListProviders(c *gin.Context) {
	items, err := mc.providers.ListProviders(c.Request.Context())
	if err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK(c, items)
}
func (mc *ManagementController) CreateProvider(c *gin.Context) {
	var req aiapp.CreateAIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := mc.providers.CreateProvider(c.Request.Context(), req)
	if err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}
func (mc *ManagementController) PatchProvider(c *gin.Context) {
	id, ok := aiResourceID(c, "id")
	if !ok {
		return
	}
	var req aiapp.PatchAIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := mc.providers.PatchProvider(c.Request.Context(), id, req); err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (mc *ManagementController) DeleteProvider(c *gin.Context) {
	id, ok := aiResourceID(c, "id")
	if !ok {
		return
	}
	if err := mc.providers.DeleteProvider(c.Request.Context(), id); err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (mc *ManagementController) ListModels(c *gin.Context) {
	var enabled *bool
	if raw := strings.TrimSpace(c.Query("enabled")); raw != "" {
		value := raw == "1" || strings.EqualFold(raw, "true")
		enabled = &value
	}
	items, err := mc.providers.ListModels(c.Request.Context(), aiapp.ListAIModelsRequest{ProviderID: aiOptionalID(c.Query("provider_id")), ModelType: c.Query("model_type"), Enabled: enabled})
	if err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK(c, items)
}
func (mc *ManagementController) CreateModel(c *gin.Context) {
	var req aiapp.CreateAIModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := mc.providers.CreateModel(c.Request.Context(), req)
	if err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}
func (mc *ManagementController) PatchModel(c *gin.Context) {
	id, ok := aiResourceID(c, "id")
	if !ok {
		return
	}
	var req aiapp.PatchAIModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := mc.providers.PatchModel(c.Request.Context(), id, req); err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (mc *ManagementController) DeleteModel(c *gin.Context) {
	id, ok := aiResourceID(c, "id")
	if !ok {
		return
	}
	if err := mc.providers.DeleteModel(c.Request.Context(), id); err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (mc *ManagementController) GetRouteSettings(c *gin.Context) {
	data, err := mc.routeSettings.Get(c.Request.Context())
	if err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK(c, data)
}
func (mc *ManagementController) UpdateRouteSettings(c *gin.Context) {
	var req aiapp.UpdateAIRouteSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := mc.routeSettings.Update(c.Request.Context(), req)
	if err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK(c, data)
}

func (mc *ManagementController) ListConversations(c *gin.Context) {
	data, err := mc.conversations.List(c.Request.Context(), aiapp.ConversationListRequest{Page: aiPage(c.Query("page"), 1), PageSize: aiPage(c.Query("page_size"), 20), ClusterID: aiOptionalID(c.Query("cluster_id")), Status: c.Query("status"), AssistantMode: c.Query("assistant_mode"), Keyword: c.Query("keyword")})
	if err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK(c, data)
}
func (mc *ManagementController) DeleteConversation(c *gin.Context) {
	id, ok := aiResourceID(c, "id")
	if !ok {
		return
	}
	if err := mc.conversations.Delete(c.Request.Context(), id); err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (mc *ManagementController) CreateConversation(c *gin.Context) {
	clusterID, ok := aiClusterID(c)
	if !ok {
		return
	}
	var req aiapp.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	userID, username := aiCurrentUser(c)
	id, err := mc.conversations.Create(c.Request.Context(), clusterID, userID, username, req)
	if err != nil {
		writeAIError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func aiClusterID(c *gin.Context) (uint64, bool) {
	if c.GetString("api_version") != "v2" {
		return aiResourceID(c, "id")
	}
	id, err := strconv.ParseUint(strings.TrimSpace(c.Query("cluster_id")), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return 0, false
	}
	return id, true
}

func aiResourceID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return 0, false
	}
	return id, true
}
func aiOptionalID(value string) uint64 {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return id
}
func aiPage(value string, fallback int) int {
	number, err := strconv.Atoi(value)
	if err != nil || number < 1 {
		return fallback
	}
	return number
}
func aiCurrentUser(c *gin.Context) (uint64, string) {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims == nil {
		return 0, ""
	}
	return uint64(max(claims.UserID, 0)), strings.TrimSpace(claims.Username)
}

func writeAIError(c *gin.Context, err error) {
	message := "内部错误"
	var applicationError *aiapp.Error
	if errors.As(err, &applicationError) && applicationError != nil && applicationError.UserMessage() != "" {
		message = applicationError.UserMessage()
	}
	switch {
	case errors.Is(err, aiapp.ErrInvalidParams):
		resp.Fail(c, 4000, message)
	case errors.Is(err, aiapp.ErrNotFound):
		resp.Fail(c, 4040, message)
	case errors.Is(err, aiapp.ErrConflict):
		resp.Fail(c, 4090, message)
	default:
		resp.Fail(c, 5000, message)
	}
}
