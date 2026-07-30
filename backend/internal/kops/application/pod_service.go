package application

import (
	"context"
	"strings"
	"time"
)

type PodReference struct {
	ClusterID uint64
	Namespace string
	Name      string
}
type PodListQuery struct {
	ClusterID     uint64
	Namespace     string
	SortBy        string
	Order         string
	LabelSelector string
}
type PodLogsInput struct {
	PodReference
	Container string
	TailLines int64
	Previous  bool
}
type PodLogSessionInput struct {
	PodReference
	Container *string
	Follow    *bool
	TailLines *int64
	Previous  bool
}
type PodLogSessionResult struct {
	SessionID string `json:"session_id"`
	WSURL     string `json:"ws_url"`
}
type PodExecSessionInput struct {
	PodReference
	UserID    uint64
	Container *string
	Command   []string
	TTY       *bool
}
type PodExecSessionResult struct {
	SessionID string `json:"session_id"`
	WSURL     string `json:"ws_url"`
}

type PodRuntime interface {
	List(context.Context, PodListQuery) (any, error)
	Metrics(context.Context, PodListQuery) (any, error)
	YAML(context.Context, PodReference) (any, error)
	Logs(context.Context, PodLogsInput) (any, error)
	Delete(context.Context, PodReference, bool) error
}

type PodService struct {
	runtime  PodRuntime
	sessions *PodLogSessionStore
	execs    *ExecSessionStore
}

func NewPodService(runtime PodRuntime, sessions *PodLogSessionStore, execs ...*ExecSessionStore) *PodService {
	var store *ExecSessionStore
	if len(execs) > 0 {
		store = execs[0]
	}
	return &PodService{runtime: runtime, sessions: sessions, execs: store}
}
func (s *PodService) CreateExecSession(input PodExecSessionInput) (*PodExecSessionResult, error) {
	if err := validatePodReference(input.PodReference); err != nil {
		return nil, err
	}
	if s == nil || s.execs == nil {
		return nil, ErrConflict
	}
	ref := normalizePodReference(input.PodReference)
	var container *string
	if input.Container != nil {
		value := strings.TrimSpace(*input.Container)
		container = &value
	}
	command := make([]string, 0, len(input.Command))
	for _, value := range input.Command {
		if value = strings.TrimSpace(value); value != "" {
			command = append(command, value)
		}
	}
	sessionID := s.execs.NewSessionID()
	s.execs.Put(sessionID, ExecSession{Kind: "pod", UserID: input.UserID, ClusterID: ref.ClusterID, Namespace: ref.Namespace, Pod: ref.Name, Container: container, Command: command, TTY: input.TTY, CreatedAt: time.Now().UTC()})
	return &PodExecSessionResult{SessionID: sessionID, WSURL: "/streams/v2/" + sessionID + "?kind=pod-exec"}, nil
}
func (s *PodService) List(ctx context.Context, query PodListQuery) (any, error) {
	if query.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	query.Namespace, query.SortBy, query.Order, query.LabelSelector = strings.TrimSpace(query.Namespace), strings.TrimSpace(query.SortBy), strings.TrimSpace(query.Order), strings.TrimSpace(query.LabelSelector)
	return s.runtime.List(ctx, query)
}
func (s *PodService) Metrics(ctx context.Context, query PodListQuery) (any, error) {
	if query.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	query.Namespace, query.SortBy, query.Order = strings.TrimSpace(query.Namespace), strings.TrimSpace(query.SortBy), strings.TrimSpace(query.Order)
	return s.runtime.Metrics(ctx, query)
}
func (s *PodService) YAML(ctx context.Context, ref PodReference) (any, error) {
	if err := validatePodReference(ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.YAML(ctx, normalizePodReference(ref))
}
func (s *PodService) Logs(ctx context.Context, input PodLogsInput) (any, error) {
	if err := validatePodReference(input.PodReference); err != nil || input.TailLines < 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	input.PodReference = normalizePodReference(input.PodReference)
	input.Container = strings.TrimSpace(input.Container)
	return s.runtime.Logs(ctx, input)
}
func (s *PodService) CreateLogSession(input PodLogSessionInput) (*PodLogSessionResult, error) {
	if err := validatePodReference(input.PodReference); err != nil {
		return nil, err
	}
	if s == nil || s.sessions == nil {
		return nil, ErrConflict
	}
	tailLines := int64(200)
	if input.TailLines != nil {
		tailLines = *input.TailLines
	}
	if tailLines < 0 {
		return nil, ErrInvalidParams
	}
	follow := true
	if input.Follow != nil {
		follow = *input.Follow
	}
	ref := normalizePodReference(input.PodReference)
	var container *string
	if input.Container != nil {
		value := strings.TrimSpace(*input.Container)
		container = &value
	}
	sessionID := s.sessions.NewSessionID()
	s.sessions.Put(sessionID, PodLogSession{ClusterID: ref.ClusterID, Namespace: ref.Namespace, Pod: ref.Name, Container: container, Follow: follow, TailLines: tailLines, Previous: input.Previous, CreatedAt: time.Now().UTC()})
	return &PodLogSessionResult{SessionID: sessionID, WSURL: "/streams/v2/" + sessionID + "?kind=pod-log"}, nil
}
func (s *PodService) Delete(ctx context.Context, ref PodReference, force bool) error {
	if err := validatePodReference(ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Delete(ctx, normalizePodReference(ref), force)
}
func validatePodReference(ref PodReference) error {
	if ref.ClusterID == 0 || strings.TrimSpace(ref.Namespace) == "" || strings.TrimSpace(ref.Name) == "" {
		return ErrInvalidParams
	}
	return nil
}
func normalizePodReference(ref PodReference) PodReference {
	ref.Namespace, ref.Name = strings.TrimSpace(ref.Namespace), strings.TrimSpace(ref.Name)
	return ref
}
