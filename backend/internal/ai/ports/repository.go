// Package ports contains persistence contracts owned by the AI application.
//
// The contracts deliberately use domain values and explicit commands instead
// of ORM handles.  This keeps transactions and SQL details inside adapters
// while allowing application services to express the business operation that
// needs to be atomic.
package ports

import (
	"context"
	"errors"
	"time"

	"k8s-platform-backend/internal/ai/domain"
)

var (
	ErrNotFound              = errors.New("ai persistence record not found")
	ErrConflict              = errors.New("ai persistence conflict")
	ErrClusterNotFound       = errors.New("ai persistence cluster not found")
	ErrProviderNotFound      = errors.New("ai persistence provider not found")
	ErrModelNotFound         = errors.New("ai persistence model not found")
	ErrModelProviderMismatch = errors.New("ai persistence model/provider mismatch")
	ErrProviderInUse         = errors.New("ai persistence provider is in use")
)

// Repository is the composition-root friendly aggregate implemented by the
// database adapter. Services depend only on the narrower interfaces below.
type Repository interface {
	ActionRepository
	ConversationRepository
	FileRepository
	ProviderRepository
	RouteSettingsRepository
	ToolRepository
}

type ActionRepository interface {
	FindConversationInCluster(context.Context, uint64, uint64) (domain.AIConversation, error)
	CreateActionProposal(context.Context, *domain.AIActionProposal, domain.AIMessage, time.Time) error
	FindActionProposalInCluster(context.Context, uint64, uint64) (domain.AIActionProposal, error)
	ListActionProposals(context.Context, uint64) ([]domain.AIActionProposal, error)
	ListActionExecutions(context.Context, []uint64) ([]domain.AIActionExecution, error)
	NextActionExecutionNumber(context.Context, uint64) (int, error)
	CreateActionExecution(context.Context, *domain.AIActionExecution) error
	MarkActionExecuting(context.Context, uint64, ActionApproval) error
	CompleteActionExecution(context.Context, ActionCompletion) error
	FailActionExecution(context.Context, ActionFailure) error
}

type ActionApproval struct {
	ProposalID         uint64
	ApprovedBy         *uint64
	ApprovedByName     string
	ApprovedAt         *time.Time
	SecondApprovedBy   *uint64
	SecondApprovedName string
	SecondApprovedAt   *time.Time
}

type ActionCompletion struct {
	ExecutionID    uint64
	ProposalID     uint64
	ConversationID uint64
	UserID         uint64
	Title          string
	Summary        string
	FinishedAt     time.Time
	Result         domain.JSONMap
}

type ActionFailure struct {
	ExecutionID    uint64
	ProposalID     uint64
	ConversationID uint64
	UserID         uint64
	Title          string
	Message        string
	FinishedAt     time.Time
}

type ConversationRepository interface {
	ListConversations(context.Context, ConversationFilter) ([]domain.AIConversation, int, error)
	CountMessages(context.Context, []uint64) (map[uint64]int, error)
	CreateConversation(context.Context, ConversationCreate) (uint64, error)
	DeleteConversation(context.Context, uint64, time.Time) error
	FindConversation(context.Context, uint64) (domain.AIConversation, error)
	ListConversationMessages(context.Context, uint64) ([]domain.AIMessage, error)
}

type ConversationFilter struct {
	Page          int
	PageSize      int
	ClusterID     uint64
	Status        string
	AssistantMode string
	Keyword       string
}

type ConversationCreate struct {
	Conversation   domain.AIConversation
	OpeningMessage string
	UserID         uint64
}

type FileRepository interface {
	CreateUploadedFiles(context.Context, []domain.AIUploadedFile) error
	FindUploadedFile(context.Context, uint64) (domain.AIUploadedFile, error)
	ListConversationFiles(context.Context, uint64) ([]domain.AIUploadedFile, error)
}

type ProviderRepository interface {
	ListProviders(context.Context) ([]domain.AIProvider, error)
	CreateProvider(context.Context, *domain.AIProvider) error
	PatchProvider(context.Context, uint64, ProviderPatch) error
	DeleteProvider(context.Context, uint64, time.Time) error
	ListModels(context.Context, ModelFilter) ([]ModelWithProviderName, error)
	CreateModel(context.Context, *domain.AIModel) error
	PatchModel(context.Context, uint64, ModelPatch) error
	DeleteModel(context.Context, uint64, time.Time) error
}

type ProviderPatch struct {
	Name         *string
	ProviderType *string
	VendorCode   *string
	BaseURL      *string
	AuthScheme   *string
	APIKeyEnc    **string
	Enabled      *bool
	Priority     *int
	Meta         *domain.JSONMap
}

type ModelFilter struct {
	ProviderID uint64
	ModelType  string
	Enabled    *bool
}

type ModelWithProviderName struct {
	Model        domain.AIModel
	ProviderName string
}

type ModelPatch struct {
	ProviderID               *uint64
	Name                     *string
	ModelCode                *string
	ModelType                *string
	Enabled                  *bool
	SupportsTools            *bool
	SupportsVision           *bool
	SupportsStreaming        *bool
	SupportsReasoning        *bool
	SupportsStructuredOutput *bool
	SupportsImageGeneration  *bool
	SupportsFileInput        *bool
	MaxInputTokens           *int
	MaxOutputTokens          *int
	ContextWindow            *int
	Meta                     *domain.JSONMap
}

type RouteSettingsRepository interface {
	GetOrCreateRouteSettings(context.Context) (domain.AIRouteSetting, error)
	UpdateRouteSettings(context.Context, uint64, RouteSettingsPatch) error
}

type RouteSettingsPatch struct {
	DefaultChatModelID            *uint64
	DefaultDiagnoseModelID        *uint64
	DefaultVisionModelID          *uint64
	DefaultImageGenerationModelID *uint64
	DefaultFallbackProviderID     *uint64
	RoutingStrategy               string
	AllowFallback                 *bool
	Meta                          *domain.JSONMap
}

type ToolRepository interface {
	ListToolCalls(context.Context, uint64) ([]domain.AIToolCall, error)
	CreateToolCall(context.Context, *domain.AIToolCall) error
	UpdateToolCall(context.Context, uint64, ToolCallUpdate) error
}

type ToolCallUpdate struct {
	Status        string
	ResultSummary string
	Result        domain.JSONMap
	ErrorMessage  string
}
