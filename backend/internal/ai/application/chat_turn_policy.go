package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	aidomain "k8s-platform-backend/internal/ai/domain"
)

// ChatRequestScope limits a chat turn and its reusable history to the same
// assistant mode and Kubernetes resource scope.
type ChatRequestScope struct {
	AssistantMode string
	Namespace     string
	ResourceKind  string
	ResourceName  string
}

func NewChatRequestScope(assistantMode, namespace, resourceKind, resourceName string) ChatRequestScope {
	return ChatRequestScope{
		AssistantMode: strings.TrimSpace(assistantMode),
		Namespace:     strings.TrimSpace(namespace),
		ResourceKind:  strings.TrimSpace(resourceKind),
		ResourceName:  strings.TrimSpace(resourceName),
	}
}

// ChatHistoryMessage is the provider-neutral subset of a persisted message
// required to filter and build an AI gateway history.
type ChatHistoryMessage struct {
	ID         uint64
	Role       string
	Content    string
	Structured aidomain.JSONMap
}

// ChatMessageScopeSnapshotInput captures the deterministic metadata saved on
// user and assistant messages for a chat turn.
type ChatMessageScopeSnapshotInput struct {
	ClusterID      uint64
	ConversationID uint64
	Scope          ChatRequestScope
	PreferModel    string
	ProviderID     *uint64
	ModelID        *uint64
	Images         []aidomain.JSONMap
	Attachments    []AIMessageAttachmentItem
}

// ChatInvocationTargetInput separates the conversation defaults from an
// optional per-request override without coupling this policy to an HTTP DTO.
type ChatInvocationTargetInput struct {
	ConversationProviderID *uint64
	ConversationModelID    *uint64
	RequestProviderID      *uint64
	RequestModelID         *uint64
}

// ResolveChatInvocationTarget gives explicit, positive request IDs precedence
// over the conversation's saved model and provider selections.
func ResolveChatInvocationTarget(input ChatInvocationTargetInput) (*uint64, *uint64) {
	providerID := input.ConversationProviderID
	if input.RequestProviderID != nil && *input.RequestProviderID > 0 {
		providerID = input.RequestProviderID
	}
	modelID := input.ConversationModelID
	if input.RequestModelID != nil && *input.RequestModelID > 0 {
		modelID = input.RequestModelID
	}
	return providerID, modelID
}

// BuildChatMessageScopeSnapshot produces the stable persisted scope contract
// consumed by scoped-history filtering in later turns.
func BuildChatMessageScopeSnapshot(input ChatMessageScopeSnapshotInput) aidomain.JSONMap {
	scope := NewChatRequestScope(
		input.Scope.AssistantMode,
		input.Scope.Namespace,
		input.Scope.ResourceKind,
		input.Scope.ResourceName,
	)
	requestScope := aidomain.JSONMap{
		"cluster_id":      input.ClusterID,
		"conversation_id": input.ConversationID,
		"assistant_mode":  scope.AssistantMode,
		"namespace":       scope.Namespace,
		"resource_kind":   scope.ResourceKind,
		"resource_name":   scope.ResourceName,
		"prefer_model":    strings.TrimSpace(input.PreferModel),
	}
	result := aidomain.JSONMap{"request_scope": requestScope}
	if input.ProviderID != nil && *input.ProviderID > 0 {
		requestScope["provider_id"] = *input.ProviderID
	}
	if input.ModelID != nil && *input.ModelID > 0 {
		requestScope["model_id"] = *input.ModelID
	}
	if len(input.Images) > 0 {
		requestScope["image_count"] = len(input.Images)
		result["request_images"] = input.Images
	}
	if len(input.Attachments) > 0 {
		requestScope["attachment_count"] = len(input.Attachments)
		result["request_attachments"] = chatMessageAttachmentRefs(input.Attachments)
	}
	return result
}

func chatMessageAttachmentRefs(attachments []AIMessageAttachmentItem) []aidomain.JSONMap {
	if len(attachments) == 0 {
		return nil
	}
	out := make([]aidomain.JSONMap, 0, len(attachments))
	for _, item := range attachments {
		out = append(out, aidomain.JSONMap{
			"id":            item.ID,
			"original_name": item.OriginalName,
			"content_type":  item.ContentType,
			"file_size":     item.FileSize,
			"file_kind":     item.FileKind,
			"download_url":  item.DownloadURL,
		})
	}
	return out
}

// FilterScopedChatHistory keeps the most recent contiguous history matching
// the active resource scope. It keeps an immediately associated system
// message, but stops before a different explicitly scoped turn.
func FilterScopedChatHistory(rows []ChatHistoryMessage, currentScope ChatRequestScope) []ChatHistoryMessage {
	if len(rows) == 0 {
		return nil
	}
	currentScope = NewChatRequestScope(
		currentScope.AssistantMode,
		currentScope.Namespace,
		currentScope.ResourceKind,
		currentScope.ResourceName,
	)
	if !currentScope.hasExplicitScope() {
		return append([]ChatHistoryMessage(nil), rows...)
	}

	filteredReversed := make([]ChatHistoryMessage, 0, len(rows))
	seenScopedCurrent := false
	for index := len(rows) - 1; index >= 0; index-- {
		row := rows[index]
		rowScope := ExtractChatRequestScope(row.Structured)
		if rowScope.hasExplicitScope() {
			if !currentScope.matches(rowScope) {
				break
			}
			seenScopedCurrent = true
			filteredReversed = append(filteredReversed, row)
			continue
		}
		if seenScopedCurrent && strings.EqualFold(strings.TrimSpace(row.Role), "system") {
			filteredReversed = append(filteredReversed, row)
		}
	}

	if len(filteredReversed) == 0 {
		return append([]ChatHistoryMessage(nil), rows[len(rows)-1])
	}
	filtered := make([]ChatHistoryMessage, 0, len(filteredReversed))
	for index := len(filteredReversed) - 1; index >= 0; index-- {
		filtered = append(filtered, filteredReversed[index])
	}
	return filtered
}

// ExtractChatRequestScope accepts both the domain JSONMap representation and
// a decoded map[string]any from older persisted message rows.
func ExtractChatRequestScope(structured aidomain.JSONMap) ChatRequestScope {
	if len(structured) == 0 {
		return ChatRequestScope{}
	}
	raw, _ := structured["request_scope"].(aidomain.JSONMap)
	if raw == nil {
		if mapValue, ok := structured["request_scope"].(map[string]any); ok {
			raw = aidomain.JSONMap(mapValue)
		}
	}
	if raw == nil {
		return ChatRequestScope{}
	}
	return NewChatRequestScope(
		fmt.Sprint(raw["assistant_mode"]),
		fmt.Sprint(raw["namespace"]),
		fmt.Sprint(raw["resource_kind"]),
		fmt.Sprint(raw["resource_name"]),
	)
}

func (scope ChatRequestScope) hasExplicitScope() bool {
	return strings.TrimSpace(scope.Namespace) != "" || strings.TrimSpace(scope.ResourceKind) != "" || strings.TrimSpace(scope.ResourceName) != ""
}

func (scope ChatRequestScope) matches(other ChatRequestScope) bool {
	return strings.EqualFold(strings.TrimSpace(scope.AssistantMode), strings.TrimSpace(other.AssistantMode)) &&
		strings.EqualFold(strings.TrimSpace(scope.Namespace), strings.TrimSpace(other.Namespace)) &&
		strings.EqualFold(strings.TrimSpace(scope.ResourceKind), strings.TrimSpace(other.ResourceKind)) &&
		strings.EqualFold(strings.TrimSpace(scope.ResourceName), strings.TrimSpace(other.ResourceName))
}

func ConversationRunStatusFromError(err error) string {
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	return "failed"
}

func MessageStatusFromError(err error) string {
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	return "failed"
}

func BuildAssistantFailureReply(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "本轮回答已取消，平台已停止继续生成结果。你可以调整问题或范围后重新发送。"
	case errors.Is(err, context.DeadlineExceeded):
		return "本轮回答处理超时，AI 未能在限定时间内完成。建议缩小排查范围后重试。"
	default:
		message := UserFacingError(err)
		if message == "" {
			message = "本轮回答失败，请稍后重试。"
		}
		return "本轮回答未成功完成。原因：" + message
	}
}

// UserFacingError extracts a safe display message from an error without
// depending on a particular runtime error implementation. Adapters can expose
// their error details through the small UserMessage contract.
func UserFacingError(err error) string {
	if err == nil {
		return ""
	}
	var carrier interface{ UserMessage() string }
	if errors.As(err, &carrier) && carrier != nil {
		if message := strings.TrimSpace(carrier.UserMessage()); message != "" {
			return message
		}
	}
	return strings.TrimSpace(err.Error())
}
