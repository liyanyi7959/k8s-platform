package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type MonitorIncidentService struct{ db *gorm.DB }

func NewMonitorIncidentService(db *gorm.DB) *MonitorIncidentService {
	return &MonitorIncidentService{db: db}
}

type MonitorAlertRuleItem struct {
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

type UpsertMonitorAlertRuleRequest struct {
	Name      string   `json:"name"`
	ClusterID uint64   `json:"cluster_id"`
	Severity  string   `json:"severity"`
	Condition string   `json:"condition"`
	Duration  string   `json:"duration"`
	Receivers []string `json:"receivers"`
}

func (s *MonitorIncidentService) ListAlertRules(ctx context.Context, page, pageSize int) (PageResult[MonitorAlertRuleItem], error) {
	page, pageSize = normalizePage(page, pageSize)
	q := s.db.WithContext(ctx).Model(&model.MonitorAlertRule{}).Where("deleted_at IS NULL")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[MonitorAlertRuleItem]{}, err
	}
	var rows []model.MonitorAlertRule
	if err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[MonitorAlertRuleItem]{}, err
	}
	items := make([]MonitorAlertRuleItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, s.alertRuleItem(ctx, row))
	}
	return PageResult[MonitorAlertRuleItem]{List: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *MonitorIncidentService) alertRuleItem(ctx context.Context, row model.MonitorAlertRule) MonitorAlertRuleItem {
	clusterName := ""
	var cluster model.Cluster
	if row.ClusterID > 0 && s.db.WithContext(ctx).Select("name").Where("id = ?", row.ClusterID).First(&cluster).Error == nil {
		clusterName = cluster.Name
	}
	return MonitorAlertRuleItem{ID: row.ID, Name: row.Name, ClusterID: row.ClusterID, ClusterName: clusterName, Severity: row.Severity, Condition: row.Condition, Duration: row.Duration, Receivers: []string(row.Receivers), Enabled: row.Enabled, CreatedAt: row.CreatedAt.Format(time.RFC3339), UpdatedAt: row.UpdatedAt.Format(time.RFC3339)}
}

func (s *MonitorIncidentService) CreateAlertRule(ctx context.Context, req UpsertMonitorAlertRuleRequest) (uint64, error) {
	row, err := normalizeAlertRule(req)
	if err != nil {
		return 0, err
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *MonitorIncidentService) UpdateAlertRule(ctx context.Context, id uint64, req UpsertMonitorAlertRuleRequest) error {
	row, err := normalizeAlertRule(req)
	if err != nil {
		return err
	}
	result := s.db.WithContext(ctx).Model(&model.MonitorAlertRule{}).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]any{"name": row.Name, "cluster_id": row.ClusterID, "severity": row.Severity, "condition_text": row.Condition, "duration": row.Duration, "receivers_json": row.Receivers})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrWithMessage(ErrNotFound, "告警规则不存在")
	}
	return nil
}

func (s *MonitorIncidentService) ToggleAlertRule(ctx context.Context, id uint64, enabled bool) error {
	result := s.db.WithContext(ctx).Model(&model.MonitorAlertRule{}).Where("id = ? AND deleted_at IS NULL", id).Update("enabled", enabled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrWithMessage(ErrNotFound, "告警规则不存在")
	}
	return nil
}

func (s *MonitorIncidentService) DeleteAlertRule(ctx context.Context, id uint64) error {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&model.MonitorAlertRule{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", &now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrWithMessage(ErrNotFound, "告警规则不存在")
	}
	return nil
}

func normalizeAlertRule(req UpsertMonitorAlertRuleRequest) (model.MonitorAlertRule, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || req.ClusterID == 0 {
		return model.MonitorAlertRule{}, ErrWithMessage(ErrInvalidParams, "规则名称和集群不能为空")
	}
	severity := strings.ToLower(strings.TrimSpace(req.Severity))
	if severity == "" {
		severity = "warning"
	}
	if severity != "info" && severity != "warning" && severity != "critical" {
		return model.MonitorAlertRule{}, ErrWithMessage(ErrInvalidParams, "不支持的告警级别")
	}
	duration := strings.TrimSpace(req.Duration)
	if duration == "" {
		duration = "5m"
	}
	return model.MonitorAlertRule{Name: name, ClusterID: req.ClusterID, Severity: severity, Condition: strings.TrimSpace(req.Condition), Duration: duration, Receivers: model.JSONStringSlice(req.Receivers), Enabled: true}, nil
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

func (s *MonitorIncidentService) IngestAlertmanager(ctx context.Context, payload AlertmanagerWebhook) ([]uint64, error) {
	ids := make([]uint64, 0, len(payload.Alerts))
	for _, alert := range payload.Alerts {
		id, err := s.upsertAlertmanagerIncident(ctx, alert)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *MonitorIncidentService) upsertAlertmanagerIncident(ctx context.Context, alert AlertmanagerAlert) (uint64, error) {
	labels := alert.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	annotations := alert.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	clusterID := parseUint(labels["cluster_id"])
	if clusterID == 0 {
		return 0, ErrWithMessage(ErrInvalidParams, "Alertmanager 告警缺少 cluster_id 标签")
	}
	fingerprint := strings.TrimSpace(alert.Fingerprint)
	if fingerprint == "" {
		fingerprint = incidentFingerprint(clusterID, labels["alertname"], labels["namespace"], labels["pod"])
	}
	status := "open"
	if strings.EqualFold(alert.Status, "resolved") {
		status = "resolved"
	}
	startedAt := alert.StartsAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	row := model.MonitorIncident{Fingerprint: fingerprint, AlertName: monitorFirstNonEmpty(labels["alertname"], "Alertmanager 告警"), ClusterID: clusterID, Namespace: labels["namespace"], ResourceKind: monitorFirstNonEmpty(labels["resource_kind"], labels["kind"]), ResourceName: monitorFirstNonEmpty(labels["resource_name"], labels["pod"], labels["deployment"]), Severity: normalizeSeverity(labels["severity"]), Status: status, Summary: monitorFirstNonEmpty(annotations["summary"], annotations["message"], labels["alertname"]), Description: annotations["description"], LabelsJSON: stringMap(labels), AnnotationsJSON: stringMap(annotations), StartedAt: startedAt}
	if status == "resolved" {
		resolved := alert.EndsAt
		if resolved.IsZero() {
			resolved = time.Now()
		}
		row.ResolvedAt = &resolved
	}
	var old model.MonitorIncident
	err := s.db.WithContext(ctx).Where("fingerprint = ?", fingerprint).First(&old).Error
	if err == nil {
		updates := map[string]any{"status": row.Status, "summary": row.Summary, "description": row.Description, "severity": row.Severity, "labels_json": row.LabelsJSON, "annotations_json": row.AnnotationsJSON}
		if row.ResolvedAt != nil {
			updates["resolved_at"] = row.ResolvedAt
		}
		if err := s.db.WithContext(ctx).Model(&old).Updates(updates).Error; err != nil {
			return 0, err
		}
		_ = s.appendTimeline(ctx, old.ID, "alert_update", statusTitle(status), row.Summary, 0, "", nil)
		return old.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, err
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	if err := s.appendTimeline(ctx, row.ID, "alert_received", "接收 Alertmanager 告警", row.Summary, 0, "", nil); err != nil {
		return 0, err
	}
	return row.ID, nil
}

type MonitorIncidentItem struct {
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
}

func (s *MonitorIncidentService) ListIncidents(ctx context.Context, page, pageSize int, status string) (PageResult[MonitorIncidentItem], error) {
	page, pageSize = normalizePage(page, pageSize)
	q := s.db.WithContext(ctx).Model(&model.MonitorIncident{})
	if status = strings.TrimSpace(status); status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[MonitorIncidentItem]{}, err
	}
	var rows []model.MonitorIncident
	if err := q.Order("started_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[MonitorIncidentItem]{}, err
	}
	items := make([]MonitorIncidentItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, s.incidentItem(ctx, row))
	}
	return PageResult[MonitorIncidentItem]{List: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *MonitorIncidentService) GetIncident(ctx context.Context, id uint64) (MonitorIncidentItem, []model.MonitorIncidentTimeline, error) {
	var row model.MonitorIncident
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return MonitorIncidentItem{}, nil, err
	}
	var timeline []model.MonitorIncidentTimeline
	if err := s.db.WithContext(ctx).Where("incident_id = ?", id).Order("created_at asc").Find(&timeline).Error; err != nil {
		return MonitorIncidentItem{}, nil, err
	}
	return s.incidentItem(ctx, row), timeline, nil
}

func (s *MonitorIncidentService) incidentItem(ctx context.Context, row model.MonitorIncident) MonitorIncidentItem {
	clusterName := ""
	var cluster model.Cluster
	if s.db.WithContext(ctx).Select("name").Where("id = ?", row.ClusterID).First(&cluster).Error == nil {
		clusterName = cluster.Name
	}
	item := MonitorIncidentItem{ID: row.ID, AlertName: row.AlertName, ClusterID: row.ClusterID, ClusterName: clusterName, Namespace: row.Namespace, ResourceKind: row.ResourceKind, ResourceName: row.ResourceName, Severity: row.Severity, Status: row.Status, Summary: row.Summary, StartedAt: row.StartedAt.Format(time.RFC3339), AIConversationID: row.AIConversationID, AIProposalID: row.AIProposalID, AssigneeName: row.AssigneeName, VerificationNote: row.VerificationNote}
	if row.ResolvedAt != nil {
		v := row.ResolvedAt.Format(time.RFC3339)
		item.ResolvedAt = &v
	}
	return item
}

func (s *MonitorIncidentService) Transition(ctx context.Context, id uint64, action, note string, userID uint64, username string) error {
	action = strings.ToLower(strings.TrimSpace(action))
	next := map[string]string{"acknowledge": "acknowledged", "diagnose": "diagnosing", "await_approval": "awaiting_approval", "execute": "executing", "verify": "verifying", "resolve": "resolved"}[action]
	if next == "" {
		return ErrWithMessage(ErrInvalidParams, "不支持的事件动作")
	}
	now := time.Now()
	updates := map[string]any{"status": next}
	if action == "acknowledge" {
		updates["acknowledged_at"] = &now
		updates["assignee_id"] = userID
		updates["assignee_name"] = username
	}
	if action == "resolve" {
		updates["resolved_at"] = &now
	}
	if action == "verify" {
		updates["verified_at"] = &now
		updates["verification_note"] = strings.TrimSpace(note)
	}
	result := s.db.WithContext(ctx).Model(&model.MonitorIncident{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrWithMessage(ErrNotFound, "事件不存在")
	}
	return s.appendTimeline(ctx, id, action, statusTitle(next), strings.TrimSpace(note), userID, username, nil)
}

func (s *MonitorIncidentService) LinkAIConversation(ctx context.Context, id, conversationID uint64, userID uint64, username string) error {
	return s.linkAI(ctx, id, "ai_conversation_id", conversationID, "关联 AI 诊断会话", userID, username)
}
func (s *MonitorIncidentService) LinkAIProposal(ctx context.Context, id, proposalID uint64, userID uint64, username string) error {
	return s.linkAI(ctx, id, "ai_proposal_id", proposalID, "关联 AI 变更提案", userID, username)
}
func (s *MonitorIncidentService) linkAI(ctx context.Context, id uint64, field string, value uint64, title string, userID uint64, username string) error {
	if value == 0 {
		return ErrWithMessage(ErrInvalidParams, "关联对象不能为空")
	}
	result := s.db.WithContext(ctx).Model(&model.MonitorIncident{}).Where("id = ?", id).Update(field, value)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrWithMessage(ErrNotFound, "事件不存在")
	}
	return s.appendTimeline(ctx, id, "ai_link", title, fmt.Sprintf("ID: %d", value), userID, username, model.JSONMap{"id": value})
}
func (s *MonitorIncidentService) appendTimeline(ctx context.Context, incidentID uint64, kind, title, detail string, userID uint64, username string, meta model.JSONMap) error {
	return s.db.WithContext(ctx).Create(&model.MonitorIncidentTimeline{IncidentID: incidentID, Type: kind, Title: title, Detail: detail, MetaJSON: meta, OperatorID: userID, Operator: username}).Error
}
func parseUint(v string) uint64 { var n uint64; _, _ = fmt.Sscan(strings.TrimSpace(v), &n); return n }
func stringMap(in map[string]string) model.JSONMap {
	out := model.JSONMap{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
func incidentFingerprint(clusterID uint64, values ...string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(fmt.Sprintf("%d|%s", clusterID, strings.Join(values, "|"))))
	return hex.EncodeToString(h.Sum(nil))
}
func monitorFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
func normalizeSeverity(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "critical" || v == "warning" || v == "info" {
		return v
	}
	return "warning"
}
func statusTitle(status string) string {
	return map[string]string{"open": "事件触发", "resolved": "告警恢复", "acknowledged": "已认领", "diagnosing": "AI 诊断中", "awaiting_approval": "等待变更确认", "executing": "自动化执行中", "verifying": "验证中"}[status]
}
