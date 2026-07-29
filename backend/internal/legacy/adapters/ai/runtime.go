package ai

import (
	"context"
	"errors"
	"strings"

	aiapp "k8s-platform-backend/internal/ai/application"
	"k8s-platform-backend/internal/fleet/ports"
	"k8s-platform-backend/internal/legacy/service"
)

type Runtime struct {
	conversations *aiapp.ConversationDetailService
	chat          *ChatRuntime
	tools         *AIToolService
	actions       *ActionRuntime
}

// ClusterReadPort adapts the Fleet dashboard read model to the AI application
// port without coupling the AI context to a concrete Fleet implementation.
type ClusterReadPort struct{ dashboard ports.DashboardReader }

func NewClusterReadPort(dashboard ports.DashboardReader) *ClusterReadPort {
	return &ClusterReadPort{dashboard: dashboard}
}

func (p *ClusterReadPort) ClusterOverview(ctx context.Context, clusterID uint64) (map[string]any, error) {
	return p.dashboard.GetClusterOverview(ctx, clusterID)
}

func (p *ClusterReadPort) ClusterCertificateRisks(ctx context.Context, clusterID uint64) ([]map[string]any, error) {
	return p.dashboard.GetClusterCertificateRisks(ctx, clusterID)
}

func NewRuntime(conversations *aiapp.ConversationDetailService, chat *ChatRuntime, tools *AIToolService, actions *ActionRuntime) *Runtime {
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
	value, err := r.chat.SendMessage(ctx, userID, username, input)
	return value, translateRuntimeError(err)
}

func (r *Runtime) SendChatStream(ctx context.Context, userID uint64, username string, input aiapp.RuntimeChatRequest, emit func(aiapp.RuntimeStreamChunk)) error {
	if r == nil || r.chat == nil {
		return aiapp.ErrConflict
	}
	return translateRuntimeError(r.chat.SendChatStream(ctx, userID, username, input, emit))
}

func (r *Runtime) CreateProposal(ctx context.Context, clusterID, userID uint64, username string, input aiapp.CreateActionProposalRequest) (any, error) {
	if r == nil || r.actions == nil {
		return nil, aiapp.ErrConflict
	}
	value, err := r.actions.CreateProposal(ctx, clusterID, userID, username, input)
	return value, translateRuntimeError(err)
}

func (r *Runtime) ConfirmProposal(ctx context.Context, clusterID, proposalID, userID uint64, username string, input aiapp.ConfirmActionProposalRequest) (any, error) {
	if r == nil || r.actions == nil {
		return nil, aiapp.ErrConflict
	}
	value, err := r.actions.ConfirmProposal(ctx, clusterID, proposalID, userID, username, input)
	return value, translateRuntimeError(err)
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
