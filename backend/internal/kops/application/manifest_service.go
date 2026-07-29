package application

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidParams = errors.New("invalid params")
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")

	ErrRuntime             = errors.New("kubernetes runtime error")
	ErrRuntimeNetwork      = errors.New("kubernetes runtime network error")
	ErrRuntimeTimeout      = errors.New("kubernetes runtime timeout")
	ErrRuntimeUnauthorized = errors.New("kubernetes runtime unauthorized")
	ErrRuntimeForbidden    = errors.New("kubernetes runtime forbidden")
	ErrRuntimeTLS          = errors.New("kubernetes runtime tls error")
)

// ManifestResultItem identifies one object affected by a manifest apply.
// It is intentionally transport-neutral so the application service does not
// expose the legacy Kubernetes executor's result model.
type ManifestResultItem struct {
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
	Operation  string `json:"operation"`
	Resource   string `json:"resource"`
	Scope      string `json:"scope"`
}

type ManifestApplyResult struct {
	RecordID uint64               `json:"record_id"`
	Status   string               `json:"status"`
	DryRun   bool                 `json:"dry_run"`
	Summary  string               `json:"summary"`
	Items    []ManifestResultItem `json:"items"`
}

type ManifestRecordQuery struct {
	ClusterID        uint64
	Page             int
	PageSize         int
	Keyword          string
	Status           string
	Mode             string
	DefaultNamespace string
}

type ManifestRecordListItem struct {
	ID               uint64    `json:"id"`
	Status           string    `json:"status"`
	DryRun           bool      `json:"dry_run"`
	DefaultNamespace string    `json:"default_namespace"`
	SourceLabel      string    `json:"source_label"`
	SourceResource   string    `json:"source_resource"`
	WorkloadKind     string    `json:"workload_kind"`
	ResultCount      int       `json:"result_count"`
	Summary          string    `json:"summary"`
	ErrorMessage     string    `json:"error_message"`
	CreatedBy        uint64    `json:"created_by"`
	CreatedByName    string    `json:"created_by_name"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ManifestRecordDetail struct {
	ManifestRecordListItem
	ClusterID   uint64               `json:"cluster_id"`
	YAMLContent string               `json:"yaml_content"`
	ResultItems []ManifestResultItem `json:"result_items"`
}

type ManifestRecordPage struct {
	List     []ManifestRecordListItem `json:"list"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

// ManifestRuntime is the Kops boundary for persistence and Kubernetes API
// execution. Its implementation may use the retained executor while the
// application and HTTP layers remain independent of it.
type ManifestRuntime interface {
	Execute(context.Context, ManifestApplyInput) (*ManifestApplyResult, error)
	List(context.Context, ManifestRecordQuery) (*ManifestRecordPage, error)
	Get(context.Context, uint64, uint64) (*ManifestRecordDetail, error)
}

type ManifestService struct{ runtime ManifestRuntime }

func NewManifestService(runtime ManifestRuntime) *ManifestService {
	return &ManifestService{runtime: runtime}
}

func (s *ManifestService) Apply(ctx context.Context, input ManifestApplyInput) (*ManifestApplyResult, error) {
	input, err := NormalizeManifestApply(input)
	if err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Execute(ctx, input)
}

func (s *ManifestService) List(ctx context.Context, query ManifestRecordQuery) (*ManifestRecordPage, error) {
	if query.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	query.Page, query.PageSize = NormalizeManifestPage(query.Page, query.PageSize)
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Status = strings.TrimSpace(query.Status)
	query.Mode = strings.TrimSpace(query.Mode)
	query.DefaultNamespace = strings.TrimSpace(query.DefaultNamespace)
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.List(ctx, query)
}

func (s *ManifestService) Get(ctx context.Context, clusterID, recordID uint64) (*ManifestRecordDetail, error) {
	if clusterID == 0 || recordID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Get(ctx, clusterID, recordID)
}

type Error struct {
	Kind    error
	Message string
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Kind }

func (e *Error) ErrorCode() string {
	switch {
	case errors.Is(e.Kind, ErrInvalidParams):
		return "invalid_params"
	case errors.Is(e.Kind, ErrNotFound):
		return "not_found"
	case errors.Is(e.Kind, ErrConflict):
		return "conflict"
	default:
		return ""
	}
}

func (e *Error) UserMessage() string { return strings.TrimSpace(e.Message) }

func ErrWithMessage(kind error, message string) error {
	if kind == nil {
		return errors.New(strings.TrimSpace(message))
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return kind
	}
	return &Error{Kind: kind, Message: message}
}
