package ai

import (
	"context"
)

// ConversationProjection adapts the remaining legacy tool/action projections
// to the AI application's read-only conversation-detail port.
type ConversationProjection struct {
	tools   *AIToolService
	actions *ActionRuntime
}

func NewConversationProjection(tools *AIToolService, actions *ActionRuntime) *ConversationProjection {
	return &ConversationProjection{tools: tools, actions: actions}
}

func (p *ConversationProjection) ListToolCallItems(ctx context.Context, conversationID uint64) ([]any, error) {
	if p == nil || p.tools == nil {
		return []any{}, nil
	}
	items, err := p.tools.ListConversationToolCalls(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	result := make([]any, len(items))
	for index, item := range items {
		result[index] = item
	}
	return result, nil
}

func (p *ConversationProjection) ListActionProposalItems(ctx context.Context, conversationID uint64) ([]any, error) {
	if p == nil || p.actions == nil {
		return []any{}, nil
	}
	items, err := p.actions.ListConversationProposals(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	result := make([]any, len(items))
	for index, item := range items {
		result[index] = item
	}
	return result, nil
}
