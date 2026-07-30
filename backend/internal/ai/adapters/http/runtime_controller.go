package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	aiapp "k8s-platform-backend/internal/ai/application"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/pkg/resp"
)

// RuntimeController owns the AI interaction endpoints. Its orchestration
// dependencies stay behind legacy service ports while HTTP ownership belongs
// to the AI bounded context.
type RuntimeController struct {
	runtime aiapp.RuntimePort
	files   *aiapp.AIFileService
}

func NewRuntimeController(
	runtime aiapp.RuntimePort,
	files *aiapp.AIFileService,
) *RuntimeController {
	return &RuntimeController{runtime: runtime, files: files}
}

func (ctl *RuntimeController) ListTools(c *gin.Context) {
	if ctl == nil || ctl.runtime == nil {
		resp.OK(c, []any{})
		return
	}
	var permissions []string
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		permissions = append(permissions, claims.Perms...)
	}
	resp.OK(c, ctl.runtime.ListTools(permissions))
}

func (ctl *RuntimeController) GetConversation(c *gin.Context) {
	id, ok := aiResourceID(c, "id")
	if !ok {
		return
	}
	if ctl == nil || ctl.runtime == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	data, err := ctl.runtime.Conversation(c.Request.Context(), id)
	if err != nil {
		writeAIRuntimeError(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *RuntimeController) SendChat(c *gin.Context) {
	request, ok := ctl.bindChatRequest(c)
	if !ok {
		return
	}
	clusterID, ok := aiClusterID(c)
	if !ok {
		return
	}
	if ctl == nil || ctl.runtime == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	request.ClusterID = clusterID
	userID, username, permissions := aiRuntimeCurrentUser(c)
	request.UserPerms = permissions
	data, err := ctl.runtime.SendMessage(c.Request.Context(), userID, username, request)
	if err != nil {
		writeAIRuntimeError(c, err)
		return
	}
	resp.OK(c, data)
}

// SendChatOrStream keeps the message resource as the single write endpoint.
// Clients select the streamed representation with Accept: text/event-stream.
func (ctl *RuntimeController) SendChatOrStream(c *gin.Context) {
	if strings.Contains(strings.ToLower(c.GetHeader("Accept")), "text/event-stream") {
		ctl.SendChatStream(c)
		return
	}
	ctl.SendChat(c)
}

func (ctl *RuntimeController) SendChatStream(c *gin.Context) {
	request, ok := ctl.bindChatRequest(c)
	if !ok {
		return
	}
	clusterID, ok := aiClusterID(c)
	if !ok {
		return
	}
	if ctl == nil || ctl.runtime == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	request.ClusterID = clusterID
	userID, username, permissions := aiRuntimeCurrentUser(c)
	if len(request.UserPerms) == 0 {
		request.UserPerms = permissions
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		resp.Fail(c, 5000, "streaming is not supported")
		return
	}
	_ = ctl.runtime.SendChatStream(c.Request.Context(), userID, username, request, func(chunk aiapp.RuntimeStreamChunk) {
		data, _ := json.Marshal(chunk)
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		flusher.Flush()
	})
}

func (ctl *RuntimeController) DownloadAttachmentContent(c *gin.Context) {
	if ctl == nil || ctl.files == nil {
		resp.Fail(c, 5000, "attachment service is unavailable")
		return
	}
	id, ok := aiResourceID(c, "id")
	if !ok {
		return
	}
	item, file, err := ctl.files.OpenAttachment(c.Request.Context(), id)
	if err != nil {
		writeAIRuntimeError(c, err)
		return
	}
	defer func() { _ = file.Close() }()

	contentType := strings.TrimSpace(item.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	disposition := "attachment"
	if strings.HasPrefix(strings.ToLower(contentType), "image/") || strings.HasPrefix(strings.ToLower(contentType), "text/") {
		disposition = "inline"
	}
	filename := item.OriginalName
	if filename == "" {
		filename = fmt.Sprintf("attachment-%d", item.ID)
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(item.FileSize, 10))
	c.Header("Cache-Control", "private, no-store")
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename*=UTF-8''%s", disposition, url.PathEscape(filename)))
	http.ServeContent(c.Writer, c.Request, filename, time.Time{}, file)
}

func (ctl *RuntimeController) CreateActionProposal(c *gin.Context) {
	var request aiapp.CreateActionProposalRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	clusterID, ok := aiClusterID(c)
	if !ok {
		return
	}
	if ctl == nil || ctl.runtime == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	userID, username, _ := aiRuntimeCurrentUser(c)
	data, err := ctl.runtime.CreateProposal(c.Request.Context(), clusterID, userID, username, request)
	if err != nil {
		writeAIRuntimeError(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *RuntimeController) ConfirmActionProposal(c *gin.Context) {
	clusterID, ok := aiClusterID(c)
	if !ok {
		return
	}
	proposalParameter := "actionId"
	if c.GetString("api_version") == "v2" {
		proposalParameter = "id"
	}
	proposalID, ok := aiResourceID(c, proposalParameter)
	if !ok {
		return
	}
	var request aiapp.ConfirmActionProposalRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if ctl == nil || ctl.runtime == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	userID, username, _ := aiRuntimeCurrentUser(c)
	data, err := ctl.runtime.ConfirmProposal(c.Request.Context(), clusterID, proposalID, userID, username, request)
	if err != nil {
		writeAIRuntimeError(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *RuntimeController) bindChatRequest(c *gin.Context) (aiapp.RuntimeChatRequest, bool) {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.ContentType())), "multipart/") {
		var request aiapp.RuntimeChatRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			resp.Fail(c, 4000, "invalid params")
			return aiapp.RuntimeChatRequest{}, false
		}
		return request, true
	}
	request := aiapp.RuntimeChatRequest{
		Message:       strings.TrimSpace(aiChatFormValue(c, "message")),
		AssistantMode: strings.TrimSpace(aiChatFormValue(c, "assistant_mode", "assistantMode")),
		PreferModel:   strings.TrimSpace(aiChatFormValue(c, "prefer_model", "preferModel")),
		Namespace:     strings.TrimSpace(aiChatFormValue(c, "namespace")),
		ResourceKind:  strings.TrimSpace(aiChatFormValue(c, "resource_kind", "resourceKind")),
		ResourceName:  strings.TrimSpace(aiChatFormValue(c, "resource_name", "resourceName")),
	}
	if id, ok := aiChatOptionalUint(aiChatFormValue(c, "conversation_id", "conversationId")); ok {
		request.ConversationID = &id
	}
	if id, ok := aiChatOptionalUint(aiChatFormValue(c, "provider_id", "providerId")); ok {
		request.ProviderID = &id
	}
	if id, ok := aiChatOptionalUint(aiChatFormValue(c, "model_id", "modelId")); ok {
		request.ModelID = &id
	}
	if rawImages := strings.TrimSpace(aiChatFormValue(c, "images")); rawImages != "" {
		if err := json.Unmarshal([]byte(rawImages), &request.Images); err != nil {
			resp.Fail(c, 4000, "invalid params")
			return aiapp.RuntimeChatRequest{}, false
		}
	}
	if ctl == nil || ctl.files == nil {
		return request, true
	}
	form, err := c.MultipartForm()
	if err != nil {
		resp.Fail(c, 4000, "invalid params")
		return aiapp.RuntimeChatRequest{}, false
	}
	if form != nil {
		uploads, err := ctl.files.NormalizeChatUploads(form.File["files"])
		if err != nil {
			writeAIRuntimeError(c, err)
			return aiapp.RuntimeChatRequest{}, false
		}
		request.Uploads = uploads
	}
	return request, true
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
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	return id, err == nil && id > 0
}

func aiRuntimeCurrentUser(c *gin.Context) (uint64, string, []string) {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims == nil {
		return 0, "", nil
	}
	userID := uint64(max(claims.UserID, 0))
	return userID, strings.TrimSpace(claims.Username), append([]string(nil), claims.Perms...)
}

func writeAIRuntimeError(c *gin.Context, err error) {
	writeAIError(c, err)
}
