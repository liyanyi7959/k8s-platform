package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/legacy/service"
	"k8s-platform-backend/pkg/resp"
)

type AIController struct {
	providerSvc      *service.AIProviderService
	routeSettingsSvc *service.AIRouteSettingsService
	conversationSvc  *service.AIConversationService
	chatSvc          *service.AIChatService
	fileSvc          *service.AIFileService
	toolSvc          *service.AIToolService
	actionSvc        *service.AIActionService
}

func NewAIController(
	providerSvc *service.AIProviderService,
	routeSettingsSvc *service.AIRouteSettingsService,
	conversationSvc *service.AIConversationService,
	chatSvc *service.AIChatService,
	fileSvc *service.AIFileService,
	toolSvc *service.AIToolService,
	actionSvc *service.AIActionService,
) *AIController {
	return &AIController{
		providerSvc:      providerSvc,
		routeSettingsSvc: routeSettingsSvc,
		conversationSvc:  conversationSvc,
		chatSvc:          chatSvc,
		fileSvc:          fileSvc,
		toolSvc:          toolSvc,
		actionSvc:        actionSvc,
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

func (ctl *AIController) DeleteProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.providerSvc.DeleteProvider(c.Request.Context(), id); err != nil {
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

func (ctl *AIController) ListTools(c *gin.Context) {
	if ctl.toolSvc == nil {
		resp.OK(c, []service.AIToolCatalogItem{})
		return
	}
	var userPerms []string
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		userPerms = append(userPerms, claims.Perms...)
	}
	resp.OK(c, ctl.toolSvc.ListTools(userPerms))
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

func (ctl *AIController) DeleteModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.providerSvc.DeleteModel(c.Request.Context(), id); err != nil {
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

func (ctl *AIController) bindAIChatRequest(c *gin.Context) (service.AIChatRequest, error) {
	contentType := strings.ToLower(strings.TrimSpace(c.ContentType()))
	if !strings.HasPrefix(contentType, "multipart/") {
		var req service.AIChatRequest
		return req, c.ShouldBindJSON(&req)
	}

	var req service.AIChatRequest
	req.Message = strings.TrimSpace(aiChatFormValue(c, "message"))
	req.AssistantMode = strings.TrimSpace(aiChatFormValue(c, "assistant_mode", "assistantMode"))
	req.PreferModel = strings.TrimSpace(aiChatFormValue(c, "prefer_model", "preferModel"))
	req.Namespace = strings.TrimSpace(aiChatFormValue(c, "namespace"))
	req.ResourceKind = strings.TrimSpace(aiChatFormValue(c, "resource_kind", "resourceKind"))
	req.ResourceName = strings.TrimSpace(aiChatFormValue(c, "resource_name", "resourceName"))

	if id, ok := aiChatOptionalUint(aiChatFormValue(c, "conversation_id", "conversationId")); ok {
		req.ConversationID = &id
	}
	if id, ok := aiChatOptionalUint(aiChatFormValue(c, "provider_id", "providerId")); ok {
		req.ProviderID = &id
	}
	if id, ok := aiChatOptionalUint(aiChatFormValue(c, "model_id", "modelId")); ok {
		req.ModelID = &id
	}

	if rawImages := strings.TrimSpace(aiChatFormValue(c, "images")); rawImages != "" {
		if err := json.Unmarshal([]byte(rawImages), &req.Images); err != nil {
			return service.AIChatRequest{}, err
		}
	}

	if ctl.fileSvc == nil {
		return req, nil
	}
	form, err := c.MultipartForm()
	if err != nil {
		return service.AIChatRequest{}, err
	}
	if form != nil {
		uploads, err := ctl.fileSvc.NormalizeChatUploads(form.File["files"])
		if err != nil {
			return service.AIChatRequest{}, err
		}
		req.Uploads = uploads
	}
	return req, nil
}

func aiChatFormValue(c *gin.Context, names ...string) string {
	for _, name := range names {
		if value := c.PostForm(name); strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func aiChatOptionalUint(value string) (uint64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}

func (ctl *AIController) SendChat(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || clusterID == 0 {
		resp.Fail(c, 4000, "鍙傛暟閿欒")
		return
	}

	req, err := ctl.bindAIChatRequest(c)
	if err != nil {
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
		req.UserPerms = append([]string(nil), claims.Perms...)
	}

	data, err := ctl.chatSvc.SendMessage(c.Request.Context(), userID, username, req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// SendChatStream handles SSE streaming chat requests.
func (ctl *AIController) SendChatStream(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if clusterID == 0 {
		resp.Fail(c, 4000, "集群 ID 无效")
		return
	}

	req, err := ctl.bindAIChatRequest(c)
	if err != nil {
		resp.Fail(c, 4000, "请求参数无效")
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
		if len(req.UserPerms) == 0 {
			req.UserPerms = append([]string(nil), claims.Perms...)
		}
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		resp.Fail(c, 5000, "当前网关不支持流式响应")
		return
	}

	_ = ctl.chatSvc.SendChatStream(c.Request.Context(), userID, username, req, func(chunk service.AIStreamChunk) {
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		flusher.Flush()
	})
}

func (ctl *AIController) DownloadAttachmentContent(c *gin.Context) {
	if ctl.fileSvc == nil {
		resp.Fail(c, 5000, "附件服务未就绪")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	item, file, err := ctl.fileSvc.OpenAttachment(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	defer func() { _ = file.Close() }()

	contentType := strings.TrimSpace(item.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	dispositionType := "attachment"
	if strings.HasPrefix(strings.ToLower(contentType), "image/") || strings.HasPrefix(strings.ToLower(contentType), "text/") {
		dispositionType = "inline"
	}
	filename := item.OriginalName
	if filename == "" {
		filename = fmt.Sprintf("attachment-%d", item.ID)
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(item.FileSize, 10))
	c.Header("Cache-Control", "private, no-store")
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename*=UTF-8''%s", dispositionType, url.PathEscape(filename)))
	http.ServeContent(c.Writer, c.Request, filename, time.Time{}, file)
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
