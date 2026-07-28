package application

import (
	"context"
	"strings"
	"time"

	"k8s-platform-backend/internal/incident/domain"
	"k8s-platform-backend/internal/incident/ports"
)

type Service struct {
	repository ports.Repository
	clusters   ports.ClusterReader
	now        func() time.Time
}

func NewService(repository ports.Repository, clusters ports.ClusterReader) *Service {
	return &Service{repository: repository, clusters: clusters, now: time.Now}
}

type IncidentDTO struct {
	ID               uint64        `json:"id"`
	AlertName        string        `json:"alert_name"`
	ClusterID        uint64        `json:"cluster_id"`
	ClusterName      string        `json:"cluster_name"`
	Namespace        string        `json:"namespace"`
	ResourceKind     string        `json:"resource_kind"`
	ResourceName     string        `json:"resource_name"`
	Severity         string        `json:"severity"`
	Status           domain.Status `json:"status"`
	Summary          string        `json:"summary"`
	StartedAt        time.Time     `json:"started_at"`
	ResolvedAt       *time.Time    `json:"resolved_at,omitempty"`
	AIConversationID *uint64       `json:"ai_conversation_id,omitempty"`
	AIProposalID     *uint64       `json:"ai_proposal_id,omitempty"`
	AssigneeName     string        `json:"assignee_name"`
	VerificationNote string        `json:"verification_note,omitempty"`
	Version          uint64        `json:"version"`
}

type TimelineDTO struct {
	ID         uint64    `json:"id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Detail     string    `json:"detail"`
	OperatorID uint64    `json:"operator_id"`
	Operator   string    `json:"operator"`
	CreatedAt  time.Time `json:"created_at"`
}

type Page struct {
	Items    []IncidentDTO `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

type Detail struct {
	Incident IncidentDTO   `json:"incident"`
	Timeline []TimelineDTO `json:"timeline"`
}

type CommandRequest struct {
	ExpectedVersion uint64 `json:"expected_version" binding:"required"`
	Note            string `json:"note"`
}

func (s *Service) List(ctx context.Context, filter ports.ListFilter) (Page, error) {
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize)
	if filter.Status != "" && !domain.IsKnownStatus(filter.Status) {
		return Page{}, domain.ErrValidation
	}
	rows, total, err := s.repository.List(ctx, filter)
	if err != nil {
		return Page{}, err
	}
	names, err := s.clusterNames(ctx, rows)
	if err != nil {
		return Page{}, err
	}
	items := make([]IncidentDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDTO(row, names[row.ClusterID]))
	}
	return Page{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize}, nil
}

func (s *Service) Get(ctx context.Context, id uint64) (Detail, error) {
	incident, timeline, err := s.repository.Get(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	names, err := s.clusters.Names(ctx, []uint64{incident.ClusterID})
	if err != nil {
		return Detail{}, err
	}
	entries := make([]TimelineDTO, 0, len(timeline))
	for _, item := range timeline {
		entries = append(entries, TimelineDTO{ID: item.ID, Type: item.Type, Title: item.Title, Detail: item.Detail, OperatorID: item.OperatorID, Operator: item.Operator, CreatedAt: item.CreatedAt})
	}
	return Detail{Incident: toDTO(incident, names[incident.ClusterID]), Timeline: entries}, nil
}

func (s *Service) Execute(ctx context.Context, id uint64, command domain.Command, request CommandRequest, actor domain.Actor) (IncidentDTO, error) {
	incident, _, err := s.repository.Get(ctx, id)
	if err != nil {
		return IncidentDTO{}, err
	}
	expected := request.ExpectedVersion
	entry, err := incident.Execute(command, expected, request.Note, actor, s.now())
	if err != nil {
		return IncidentDTO{}, err
	}
	if err := s.repository.SaveTransition(ctx, incident, expected, entry); err != nil {
		return IncidentDTO{}, err
	}
	clusterName := ""
	if names, nameErr := s.clusters.Names(ctx, []uint64{incident.ClusterID}); nameErr == nil {
		clusterName = names[incident.ClusterID]
	}
	return toDTO(incident, clusterName), nil
}

func (s *Service) clusterNames(ctx context.Context, incidents []domain.Incident) (map[uint64]string, error) {
	seen := map[uint64]struct{}{}
	ids := make([]uint64, 0, len(incidents))
	for _, incident := range incidents {
		if _, ok := seen[incident.ClusterID]; !ok {
			seen[incident.ClusterID] = struct{}{}
			ids = append(ids, incident.ClusterID)
		}
	}
	return s.clusters.Names(ctx, ids)
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func toDTO(row domain.Incident, clusterName string) IncidentDTO {
	return IncidentDTO{ID: row.ID, AlertName: row.AlertName, ClusterID: row.ClusterID, ClusterName: strings.TrimSpace(clusterName), Namespace: row.Namespace, ResourceKind: row.ResourceKind, ResourceName: row.ResourceName, Severity: row.Severity, Status: row.Status, Summary: row.Summary, StartedAt: row.StartedAt, ResolvedAt: row.ResolvedAt, AIConversationID: row.AIConversationID, AIProposalID: row.AIProposalID, AssigneeName: row.AssigneeName, VerificationNote: row.VerificationNote, Version: row.Version}
}
