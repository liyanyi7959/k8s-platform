package ai

import (
	"context"
	"errors"
	"strings"

	aiapp "k8s-platform-backend/internal/ai/application"
	aidomain "k8s-platform-backend/internal/ai/domain"
	"k8s-platform-backend/internal/legacy/service"
)

type Runtime struct {
	conversations *aiapp.ConversationDetailService
	chat          *service.AIChatService
	tools         *service.AIToolService
	actions       *service.AIActionService
}

func NewRuntime(conversations *aiapp.ConversationDetailService, chat *service.AIChatService, tools *service.AIToolService, actions *service.AIActionService) *Runtime {
	return &Runtime{conversations: conversations, chat: chat, tools: tools, actions: actions}
}

func (r *Runtime) ListTools(permissions []string) []any {
	if r == nil || r.tools == nil {
		return []any{}
	}
	items := r.tools.ListTools(permissions)
	result := make([]any, len(items))
	for index, item := range items {
		result[index] = item
	}
	return result
}

func (r *Runtime) Conversation(ctx context.Context, id uint64) (any, error) {
	if r == nil || r.conversations == nil {
		return nil, aiapp.ErrConflict
	}
	value, err := r.conversations.Get(ctx, id)
	return value, translateRuntimeError(err)
}

func (r *Runtime) SendMessage(ctx context.Context, userID uint64, username string, input aiapp.RuntimeChatRequest) (any, error) {
	if r == nil || r.chat == nil {
		return nil, aiapp.ErrConflict
	}
	value, err := r.chat.SendMessage(ctx, userID, username, runtimeChatRequest(input))
	return value, translateRuntimeError(err)
}

func (r *Runtime) SendChatStream(ctx context.Context, userID uint64, username string, input aiapp.RuntimeChatRequest, emit func(aiapp.RuntimeStreamChunk)) error {
	if r == nil || r.chat == nil {
		return aiapp.ErrConflict
	}
	return translateRuntimeError(r.chat.SendChatStream(ctx, userID, username, runtimeChatRequest(input), func(chunk service.AIStreamChunk) {
		emit(aiapp.RuntimeStreamChunk{Type: chunk.Type, Content: chunk.Content, Error: chunk.Error, Progress: chunk.Progress})
	}))
}

func (r *Runtime) CreateProposal(ctx context.Context, clusterID, userID uint64, username string, input aiapp.CreateActionProposalRequest) (any, error) {
	if r == nil || r.actions == nil {
		return nil, aiapp.ErrConflict
	}
	value, err := r.actions.CreateProposal(ctx, clusterID, userID, username, service.CreateAIActionProposalRequest{
		ConversationID: input.ConversationID,
		MessageID:      input.MessageID,
		ProposalType:   input.ProposalType,
		TargetResource: service.AIActionTargetResource{Kind: input.TargetResource.Kind, Namespace: input.TargetResource.Namespace, Name: input.TargetResource.Name},
		Payload:        aidomain.JSONMap(input.Payload),
		Reason:         input.Reason,
	})
	return value, translateRuntimeError(err)
}

func (r *Runtime) ConfirmProposal(ctx context.Context, clusterID, proposalID, userID uint64, username string, input aiapp.ConfirmActionProposalRequest) (any, error) {
	if r == nil || r.actions == nil {
		return nil, aiapp.ErrConflict
	}
	value, err := r.actions.ConfirmProposal(ctx, clusterID, proposalID, userID, username, service.ConfirmAIActionProposalRequest{
		ConfirmationText: input.ConfirmationText,
		ConfirmRisk:      input.ConfirmRisk,
		OperatorComment:  input.OperatorComment,
	})
	return value, translateRuntimeError(err)
}

func runtimeChatRequest(input aiapp.RuntimeChatRequest) service.AIChatRequest {
	images := make([]service.AIChatImageInput, len(input.Images))
	for index, image := range input.Images {
		images[index] = service.AIChatImageInput{Name: image.Name, ContentType: image.ContentType, DataURL: image.DataURL, Size: image.Size}
	}
	return service.AIChatRequest{
		ClusterID: input.ClusterID, ConversationID: input.ConversationID, Message: input.Message, AssistantMode: input.AssistantMode,
		ProviderID: input.ProviderID, ModelID: input.ModelID, PreferModel: input.PreferModel, Namespace: input.Namespace,
		ResourceKind: input.ResourceKind, ResourceName: input.ResourceName, Images: images, Uploads: input.Uploads,
		UserPerms: append([]string(nil), input.UserPerms...),
	}
}

func translateRuntimeError(err error) error {
	if err == nil {
		return nil
	}
	for _, candidate := range []struct{ legacy, target error }{
		{service.ErrInvalidParams, aiapp.ErrInvalidParams},
		{service.ErrNotFound, aiapp.ErrNotFound},
		{service.ErrConflict, aiapp.ErrConflict},
		{service.ErrCrypto, aiapp.ErrCrypto},
	} {
		if errors.Is(err, candidate.legacy) {
			if message, ok := service.UserMessage(err); ok {
				return aiapp.ErrWithMessage(candidate.target, message)
			}
			return candidate.target
		}
	}
	if message, ok := service.UserMessage(err); ok && strings.TrimSpace(message) != "" {
		return aiapp.ErrWithMessage(nil, message)
	}
	return err
}
