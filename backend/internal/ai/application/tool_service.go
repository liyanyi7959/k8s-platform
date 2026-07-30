package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s-platform-backend/internal/ai/domain"
	"k8s-platform-backend/internal/ai/ports"
)

type ToolContextRequest struct {
	ConversationID uint64
	MessageID      uint64
	ClusterID      uint64
	UserID         uint64
	Username       string
	UserPerms      []string
	Query          string
	Namespace      string
	ResourceKind   string
	ResourceName   string
}

type ToolResult struct {
	Summary  string         `json:"summary"`
	Evidence domain.JSONMap `json:"evidence,omitempty"`
	RawRef   domain.JSONMap `json:"raw_ref,omitempty"`
}

type ToolDefinition struct {
	Name                string
	Category            string
	Description         string
	RequiredPermissions []string
	RiskLevel           string
	ConfirmLevel        string
	Timeout             time.Duration
	RedactionPolicy     string
	InputSchema         domain.JSONMap
	OutputSchema        domain.JSONMap
	Handler             func(context.Context, ToolContextRequest, map[string]any) (ToolResult, error)
}

type ToolPlanStep struct {
	ToolName string         `json:"tool_name"`
	Params   domain.JSONMap `json:"params,omitempty"`
	Reason   string         `json:"reason,omitempty"`
}

type ToolCatalogItem struct {
	Name                string         `json:"name"`
	Category            string         `json:"category"`
	Description         string         `json:"description"`
	RequiredPermissions []string       `json:"required_permissions"`
	MissingPermissions  []string       `json:"missing_permissions,omitempty"`
	RiskLevel           string         `json:"risk_level"`
	ConfirmLevel        string         `json:"confirm_level"`
	TimeoutSeconds      int64          `json:"timeout_seconds"`
	RedactionPolicy     string         `json:"redaction_policy,omitempty"`
	InputSchema         domain.JSONMap `json:"input_schema,omitempty"`
	OutputSchema        domain.JSONMap `json:"output_schema,omitempty"`
	Available           bool           `json:"available"`
}

type ToolCallItem struct {
	ID            uint64         `json:"id"`
	MessageID     *uint64        `json:"message_id,omitempty"`
	ToolName      string         `json:"tool_name"`
	ToolKind      string         `json:"tool_kind"`
	Status        string         `json:"status"`
	RiskLevel     string         `json:"risk_level"`
	ConfirmLevel  string         `json:"confirm_level"`
	ResultSummary string         `json:"result_summary"`
	Result        domain.JSONMap `json:"result,omitempty"`
	ErrorMessage  string         `json:"error_message,omitempty"`
	CreatedAt     string         `json:"created_at"`
}

// ToolRegistryPort keeps tool definitions and their infrastructure handlers
// outside the AI application service while the application owns persistence,
// timeout handling and execution state transitions.
type ToolRegistryPort interface {
	Get(string) (ToolDefinition, bool)
	List() []ToolDefinition
	ListCatalog([]string) []ToolCatalogItem
	PlanAutoDiagnostics(ToolContextRequest) []ToolPlanStep
	ValidateToolPermissions([]string, []string, string) error
}

type ToolService struct {
	repository ports.ToolRepository
	registry   ToolRegistryPort
}

func NewToolService(repository ports.ToolRepository, registry ToolRegistryPort) *ToolService {
	return &ToolService{repository: repository, registry: registry}
}

func (s *ToolService) RunAutoDiagnostics(ctx context.Context, req ToolContextRequest) ([]ToolCallItem, string, error) {
	if s == nil || s.repository == nil {
		return nil, "", errors.New("tool repository is required")
	}
	if s.registry == nil {
		return nil, "", errors.New("tool registry is required")
	}
	if req.ClusterID == 0 || req.ConversationID == 0 {
		return nil, "", ErrorWithMessage(ErrInvalidParams, "AI diagnostic context is invalid")
	}
	items := make([]ToolCallItem, 0, 6)
	contextBlocks := make([]string, 0, 6)
	for _, step := range s.registry.PlanAutoDiagnostics(req) {
		item, block := s.Execute(ctx, req, step.ToolName, map[string]any(step.Params))
		if item.ID > 0 {
			items = append(items, item)
		}
		if strings.TrimSpace(block) != "" {
			contextBlocks = append(contextBlocks, block)
		}
	}
	return items, strings.Join(contextBlocks, "\n\n"), nil
}

func (s *ToolService) ListTools(userPerms []string) []ToolCatalogItem {
	if s == nil || s.registry == nil {
		return nil
	}
	return s.registry.ListCatalog(userPerms)
}

func (s *ToolService) ListDefinitions() []ToolDefinition {
	if s == nil || s.registry == nil {
		return nil
	}
	return s.registry.List()
}

func (s *ToolService) ListConversationToolCalls(ctx context.Context, conversationID uint64) ([]ToolCallItem, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("tool repository is required")
	}
	if conversationID == 0 {
		return nil, ErrorWithMessage(ErrInvalidParams, "conversation ID is invalid")
	}
	rows, err := s.repository.ListToolCalls(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	items := make([]ToolCallItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, BuildToolCallItem(row))
	}
	return items, nil
}

func (s *ToolService) Execute(ctx context.Context, req ToolContextRequest, toolName string, params map[string]any) (ToolCallItem, string) {
	def, ok := s.registry.Get(toolName)
	if !ok {
		return ToolCallItem{}, ""
	}
	row := domain.AIToolCall{ConversationID: req.ConversationID, MessageID: &req.MessageID, ClusterID: req.ClusterID,
		ToolName: toolName, ToolKind: def.Category, ExecutionMode: "auto", Status: "executing", RiskLevel: def.RiskLevel,
		ConfirmLevel: def.ConfirmLevel, CreatedBy: req.UserID, CreatedByName: strings.TrimSpace(req.Username)}
	if payload, err := json.Marshal(params); err == nil {
		row.CommandText, row.ParamsJSON = string(payload), domain.JSONMap(params)
	}
	if err := s.repository.CreateToolCall(ctx, &row); err != nil {
		return ToolCallItem{}, ""
	}
	if err := s.registry.ValidateToolPermissions(def.RequiredPermissions, req.UserPerms, toolName); err != nil {
		row.Status, row.ErrorMessage, row.ResultSummary = "failed", userFacingError(err), userFacingError(err)
		s.updateFailure(ctx, row)
		return BuildToolCallItem(row), ""
	}
	handlerCtx, cancel := ctx, func() {}
	if def.Timeout > 0 {
		handlerCtx, cancel = context.WithTimeout(ctx, def.Timeout)
	}
	defer cancel()
	result, err := def.Handler(handlerCtx, req, params)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(handlerCtx.Err(), context.DeadlineExceeded) {
			err = ErrorWithMessage(ErrConflict, fmt.Sprintf("tool %s execution timed out", toolName))
		}
		row.Status, row.ErrorMessage, row.ResultSummary = "failed", userFacingError(err), userFacingError(err)
		s.updateFailure(ctx, row)
		return BuildToolCallItem(row), ""
	}
	resultJSON := toolEvidenceMap(result)
	row.Status, row.ResultSummary, row.ResultJSON = "succeeded", result.Summary, resultJSON
	_ = s.repository.UpdateToolCall(ctx, row.ID, ports.ToolCallUpdate{Status: row.Status, ResultSummary: row.ResultSummary, Result: row.ResultJSON})
	return BuildToolCallItem(row), "工具 " + toolName + ": " + result.Summary + "\n" + compactToolResult(resultJSON)
}

func (s *ToolService) updateFailure(ctx context.Context, row domain.AIToolCall) {
	_ = s.repository.UpdateToolCall(ctx, row.ID, ports.ToolCallUpdate{Status: row.Status, ResultSummary: row.ResultSummary, ErrorMessage: row.ErrorMessage})
}

func BuildToolCallItem(row domain.AIToolCall) ToolCallItem {
	return ToolCallItem{ID: row.ID, MessageID: row.MessageID, ToolName: row.ToolName, ToolKind: row.ToolKind,
		Status: row.Status, RiskLevel: row.RiskLevel, ConfirmLevel: row.ConfirmLevel, ResultSummary: row.ResultSummary,
		Result: row.ResultJSON, ErrorMessage: row.ErrorMessage, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339)}
}

func toolEvidenceMap(result ToolResult) domain.JSONMap {
	out := domain.JSONMap{"summary": result.Summary}
	if len(result.Evidence) > 0 {
		out["evidence"] = result.Evidence
	}
	if len(result.RawRef) > 0 {
		out["raw_ref"] = result.RawRef
	}
	return out
}

func compactToolResult(value any) string {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ""
	}
	return truncateToolResult(string(encoded), 5000)
}

func truncateToolResult(input string, limit int) string {
	raw := strings.TrimSpace(input)
	if limit <= 0 || len([]rune(raw)) <= limit {
		return raw
	}
	return string([]rune(raw)[:limit]) + "..."
}

func userFacingError(err error) string {
	if err == nil {
		return ""
	}
	if value, ok := err.(interface{ UserMessage() string }); ok && strings.TrimSpace(value.UserMessage()) != "" {
		return strings.TrimSpace(value.UserMessage())
	}
	return strings.TrimSpace(err.Error())
}
