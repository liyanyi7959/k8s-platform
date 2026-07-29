package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s-platform-backend/internal/incident/domain"
	"k8s-platform-backend/internal/incident/ports"
)

type MonitoringService struct{ repository ports.MonitoringRepository }

func NewMonitoringService(repository ports.MonitoringRepository) *MonitoringService {
	return &MonitoringService{repository: repository}
}

type AlertRuleDTO struct {
	ID          uint64   `json:"id"`
	Name        string   `json:"name"`
	ClusterID   uint64   `json:"cluster_id"`
	ClusterName string   `json:"cluster_name"`
	Severity    string   `json:"severity"`
	Condition   string   `json:"condition"`
	Duration    string   `json:"duration"`
	Receivers   []string `json:"receivers"`
	Enabled     bool     `json:"enabled"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type AlertRuleRequest struct {
	Name      string   `json:"name"`
	ClusterID uint64   `json:"cluster_id"`
	Severity  string   `json:"severity"`
	Condition string   `json:"condition"`
	Duration  string   `json:"duration"`
	Receivers []string `json:"receivers"`
}

type AlertRulePage struct {
	List     []AlertRuleDTO `json:"list"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

func (s *MonitoringService) ListAlertRules(ctx context.Context, page, pageSize int) (AlertRulePage, error) {
	page, pageSize = normalizePage(page, pageSize)
	rows, total, err := s.repository.ListAlertRules(ctx, page, pageSize)
	if err != nil {
		return AlertRulePage{}, err
	}
	items := make([]AlertRuleDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, AlertRuleDTO{ID: row.ID, Name: row.Name, ClusterID: row.ClusterID, ClusterName: row.ClusterName, Severity: row.Severity, Condition: row.Condition, Duration: row.Duration, Receivers: row.Receivers, Enabled: row.Enabled, CreatedAt: row.CreatedAt.Format(time.RFC3339), UpdatedAt: row.UpdatedAt.Format(time.RFC3339)})
	}
	return AlertRulePage{List: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *MonitoringService) CreateAlertRule(ctx context.Context, request AlertRuleRequest) (uint64, error) {
	rule, err := domain.NewAlertRule(request.Name, request.ClusterID, request.Severity, request.Condition, request.Duration, request.Receivers)
	if err != nil {
		return 0, err
	}
	if err := s.repository.CreateAlertRule(ctx, &rule); err != nil {
		return 0, err
	}
	return rule.ID, nil
}
func (s *MonitoringService) UpdateAlertRule(ctx context.Context, id uint64, request AlertRuleRequest) error {
	rule, err := domain.NewAlertRule(request.Name, request.ClusterID, request.Severity, request.Condition, request.Duration, request.Receivers)
	if err != nil {
		return err
	}
	return s.repository.UpdateAlertRule(ctx, id, rule)
}
func (s *MonitoringService) ToggleAlertRule(ctx context.Context, id uint64, enabled bool) error {
	return s.repository.ToggleAlertRule(ctx, id, enabled)
}
func (s *MonitoringService) DeleteAlertRule(ctx context.Context, id uint64) error {
	return s.repository.DeleteAlertRule(ctx, id)
}

type AlertmanagerWebhook struct {
	Status string              `json:"status"`
	Alerts []AlertmanagerAlert `json:"alerts"`
}
type AlertmanagerAlert struct {
	Status      string            `json:"status"`
	Fingerprint string            `json:"fingerprint"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

func (s *MonitoringService) IngestAlertmanager(ctx context.Context, payload AlertmanagerWebhook) ([]uint64, error) {
	ids := make([]uint64, 0, len(payload.Alerts))
	for _, alert := range payload.Alerts {
		id, err := s.repository.UpsertAlert(ctx, domain.AlertmanagerAlert{Status: alert.Status, Fingerprint: alert.Fingerprint, StartsAt: alert.StartsAt, EndsAt: alert.EndsAt, Labels: alert.Labels, Annotations: alert.Annotations})
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

type LegacyIncidentDTO struct {
	ID               uint64  `json:"id"`
	AlertName        string  `json:"alert_name"`
	ClusterID        uint64  `json:"cluster_id"`
	ClusterName      string  `json:"cluster_name"`
	Namespace        string  `json:"namespace"`
	ResourceKind     string  `json:"resource_kind"`
	ResourceName     string  `json:"resource_name"`
	Severity         string  `json:"severity"`
	Status           string  `json:"status"`
	Summary          string  `json:"summary"`
	StartedAt        string  `json:"started_at"`
	ResolvedAt       *string `json:"resolved_at,omitempty"`
	AIConversationID *uint64 `json:"ai_conversation_id,omitempty"`
	AIProposalID     *uint64 `json:"ai_proposal_id,omitempty"`
	AssigneeName     string  `json:"assignee_name"`
	VerificationNote string  `json:"verification_note,omitempty"`
	Version          uint64  `json:"version"`
}
type LegacyIncidentPage struct {
	List     []LegacyIncidentDTO `json:"list"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}
type LegacyTimelineDTO struct {
	ID         uint64    `json:"id"`
	IncidentID uint64    `json:"incident_id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Detail     string    `json:"detail"`
	OperatorID uint64    `json:"operator_id"`
	Operator   string    `json:"operator"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *MonitoringService) ListIncidents(ctx context.Context, page, pageSize int, status string) (LegacyIncidentPage, error) {
	page, pageSize = normalizePage(page, pageSize)
	rows, total, err := s.repository.ListLegacyIncidents(ctx, page, pageSize, strings.TrimSpace(status))
	if err != nil {
		return LegacyIncidentPage{}, err
	}
	items := make([]LegacyIncidentDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, legacyDTO(row))
	}
	return LegacyIncidentPage{List: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}
func (s *MonitoringService) GetIncident(ctx context.Context, id uint64) (LegacyIncidentDTO, []LegacyTimelineDTO, error) {
	row, timeline, err := s.repository.GetLegacyIncident(ctx, id)
	if err != nil {
		return LegacyIncidentDTO{}, nil, err
	}
	items := make([]LegacyTimelineDTO, 0, len(timeline))
	for _, entry := range timeline {
		items = append(items, LegacyTimelineDTO{ID: entry.ID, IncidentID: id, Type: entry.Type, Title: entry.Title, Detail: entry.Detail, OperatorID: entry.OperatorID, Operator: entry.Operator, CreatedAt: entry.CreatedAt})
	}
	return legacyDTO(row), items, nil
}
func (s *MonitoringService) LinkAIConversation(ctx context.Context, id, value, userID uint64, username string) error {
	return s.linkAI(ctx, id, "ai_conversation_id", value, userID, username)
}
func (s *MonitoringService) LinkAIProposal(ctx context.Context, id, value, userID uint64, username string) error {
	return s.linkAI(ctx, id, "ai_proposal_id", value, userID, username)
}
func (s *MonitoringService) linkAI(ctx context.Context, id uint64, field string, value, userID uint64, username string) error {
	if value == 0 {
		return fmt.Errorf("%w: 关联对象不能为空", domain.ErrValidation)
	}
	return s.repository.LinkAI(ctx, id, field, value, domain.Actor{ID: userID, Name: username})
}
func legacyDTO(row domain.LegacyIncident) LegacyIncidentDTO {
	dto := LegacyIncidentDTO{ID: row.ID, AlertName: row.AlertName, ClusterID: row.ClusterID, ClusterName: row.ClusterName, Namespace: row.Namespace, ResourceKind: row.ResourceKind, ResourceName: row.ResourceName, Severity: row.Severity, Status: row.Status, Summary: row.Summary, StartedAt: row.StartedAt.Format(time.RFC3339), AIConversationID: row.AIConversationID, AIProposalID: row.AIProposalID, AssigneeName: row.AssigneeName, VerificationNote: row.VerificationNote, Version: row.Version}
	if row.ResolvedAt != nil {
		value := row.ResolvedAt.Format(time.RFC3339)
		dto.ResolvedAt = &value
	}
	return dto
}
