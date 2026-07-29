package mysql

import (
	"context"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/incident/domain"
)

func (r *Repository) ListAlertRules(ctx context.Context, page, pageSize int) ([]domain.AlertRule, int64, error) {
	query := r.db.WithContext(ctx).Model(&alertRuleRow{}).Where("deleted_at IS NULL")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []alertRuleRow
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	clusterIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		clusterIDs = append(clusterIDs, row.ClusterID)
	}
	names, err := r.Names(ctx, clusterIDs)
	if err != nil {
		return nil, 0, err
	}
	result := make([]domain.AlertRule, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.AlertRule{ID: row.ID, Name: row.Name, ClusterID: row.ClusterID, ClusterName: names[row.ClusterID], Severity: row.Severity, Condition: row.Condition, Duration: row.Duration, Receivers: []string(row.Receivers), Enabled: row.Enabled, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, DeletedAt: row.DeletedAt})
	}
	return result, total, nil
}
func (r *Repository) CreateAlertRule(ctx context.Context, rule *domain.AlertRule) error {
	row := alertRuleRow{Name: rule.Name, ClusterID: rule.ClusterID, Severity: rule.Severity, Condition: rule.Condition, Duration: rule.Duration, Receivers: jsonStringSlice(rule.Receivers), Enabled: true}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	rule.ID, rule.CreatedAt, rule.UpdatedAt = row.ID, row.CreatedAt, row.UpdatedAt
	return nil
}
func (r *Repository) UpdateAlertRule(ctx context.Context, id uint64, rule domain.AlertRule) error {
	result := r.db.WithContext(ctx).Model(&alertRuleRow{}).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]any{"name": rule.Name, "cluster_id": rule.ClusterID, "severity": rule.Severity, "condition_text": rule.Condition, "duration": rule.Duration, "receivers_json": jsonStringSlice(rule.Receivers)})
	return affected(result, domain.ErrNotFound)
}
func (r *Repository) ToggleAlertRule(ctx context.Context, id uint64, enabled bool) error {
	return affected(r.db.WithContext(ctx).Model(&alertRuleRow{}).Where("id = ? AND deleted_at IS NULL", id).Update("enabled", enabled), domain.ErrNotFound)
}
func (r *Repository) DeleteAlertRule(ctx context.Context, id uint64) error {
	return affected(r.db.WithContext(ctx).Model(&alertRuleRow{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", time.Now()), domain.ErrNotFound)
}

func (r *Repository) UpsertAlert(ctx context.Context, alert domain.AlertmanagerAlert) (uint64, error) {
	labels, annotations := alert.Labels, alert.Annotations
	if labels == nil {
		labels = map[string]string{}
	}
	if annotations == nil {
		annotations = map[string]string{}
	}
	clusterID := parseUint(labels["cluster_id"])
	if clusterID == 0 {
		return 0, fmt.Errorf("%w: Alertmanager 告警缺少 cluster_id 标签", domain.ErrValidation)
	}
	fingerprint := strings.TrimSpace(alert.Fingerprint)
	if fingerprint == "" {
		fingerprint = fingerprintFor(clusterID, labels["alertname"], labels["namespace"], labels["pod"])
	}
	status := "open"
	if strings.EqualFold(alert.Status, "resolved") {
		status = "resolved"
	}
	startedAt := alert.StartsAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	row := incidentRow{Fingerprint: fingerprint, AlertName: first(labels["alertname"], "Alertmanager 告警"), ClusterID: clusterID, Namespace: labels["namespace"], ResourceKind: first(labels["resource_kind"], labels["kind"]), ResourceName: first(labels["resource_name"], labels["pod"], labels["deployment"]), Severity: domain.NormalizeSeverity(labels["severity"]), Status: status, Summary: first(annotations["summary"], annotations["message"], labels["alertname"]), Description: annotations["description"], LabelsJSON: toJSONMap(labels), AnnotationsJSON: toJSONMap(annotations), StartedAt: startedAt, Version: 1}
	if status == "resolved" {
		resolved := alert.EndsAt
		if resolved.IsZero() {
			resolved = time.Now()
		}
		row.ResolvedAt = &resolved
	}
	var id uint64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old incidentRow
		findErr := tx.Where("fingerprint = ?", fingerprint).First(&old).Error
		if findErr == nil {
			updates := map[string]any{"status": row.Status, "summary": row.Summary, "description": row.Description, "severity": row.Severity, "labels_json": row.LabelsJSON, "annotations_json": row.AnnotationsJSON, "version": gorm.Expr("version + 1")}
			if row.ResolvedAt != nil {
				updates["resolved_at"] = row.ResolvedAt
			}
			if err := tx.Model(&old).Updates(updates).Error; err != nil {
				return err
			}
			id = old.ID
			return appendTimeline(tx, id, "alert_update", statusTitle(status), row.Summary, domain.Actor{})
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		id = row.ID
		return appendTimeline(tx, id, "alert_received", "接收 Alertmanager 告警", row.Summary, domain.Actor{})
	})
	return id, err
}

func (r *Repository) ListLegacyIncidents(ctx context.Context, page, pageSize int, status string) ([]domain.LegacyIncident, int64, error) {
	query := r.db.WithContext(ctx).Model(&incidentRow{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []incidentRow
	if err := query.Order("started_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ClusterID)
	}
	names, err := r.Names(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	result := make([]domain.LegacyIncident, 0, len(rows))
	for _, row := range rows {
		result = append(result, toLegacy(row, names[row.ClusterID]))
	}
	return result, total, nil
}
func (r *Repository) GetLegacyIncident(ctx context.Context, id uint64) (domain.LegacyIncident, []domain.TimelineEntry, error) {
	incident, timeline, err := r.Get(ctx, id)
	if err != nil {
		return domain.LegacyIncident{}, nil, err
	}
	names, err := r.Names(ctx, []uint64{incident.ClusterID})
	if err != nil {
		return domain.LegacyIncident{}, nil, err
	}
	var row incidentRow
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.LegacyIncident{}, nil, err
	}
	return toLegacy(row, names[row.ClusterID]), timeline, nil
}
func (r *Repository) LinkAI(ctx context.Context, id uint64, field string, value uint64, actor domain.Actor) error {
	title := "关联 AI 诊断会话"
	if field == "ai_proposal_id" {
		title = "关联 AI 变更提案"
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&incidentRow{}).Where("id = ?", id).Updates(map[string]any{field: value, "version": gorm.Expr("version + 1")})
		if err := affected(result, domain.ErrNotFound); err != nil {
			return err
		}
		return appendTimeline(tx, id, "ai_link", title, fmt.Sprintf("ID: %d", value), actor)
	})
}

func affected(result *gorm.DB, missing error) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return missing
	}
	return nil
}
func appendTimeline(tx *gorm.DB, id uint64, kind, title, detail string, actor domain.Actor) error {
	return tx.Create(&timelineRow{IncidentID: id, Type: kind, Title: title, Detail: detail, OperatorID: actor.ID, Operator: actor.Name, CreatedAt: time.Now()}).Error
}
func toLegacy(row incidentRow, clusterName string) domain.LegacyIncident {
	return domain.LegacyIncident{ID: row.ID, AlertName: row.AlertName, ClusterID: row.ClusterID, ClusterName: clusterName, Namespace: row.Namespace, ResourceKind: row.ResourceKind, ResourceName: row.ResourceName, Severity: row.Severity, Status: row.Status, Summary: row.Summary, StartedAt: row.StartedAt, ResolvedAt: row.ResolvedAt, AIConversationID: row.AIConversationID, AIProposalID: row.AIProposalID, AssigneeName: row.AssigneeName, VerificationNote: row.VerificationNote, Version: row.Version}
}
func parseUint(value string) uint64 {
	var result uint64
	_, _ = fmt.Sscan(strings.TrimSpace(value), &result)
	return result
}
func fingerprintFor(clusterID uint64, values ...string) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(fmt.Sprintf("%d|%s", clusterID, strings.Join(values, "|"))))
	return hex.EncodeToString(hash.Sum(nil))
}
func first(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
func statusTitle(status string) string {
	if status == "resolved" {
		return "告警恢复"
	}
	return "事件触发"
}

type jsonMap map[string]any

func (value *jsonMap) Scan(input any) error {
	if input == nil {
		*value = nil
		return nil
	}
	bytes, ok := input.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, value)
}
func (value jsonMap) Value() (driver.Value, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}

type jsonStringSlice []string

func (value *jsonStringSlice) Scan(input any) error {
	if input == nil {
		*value = nil
		return nil
	}
	bytes, ok := input.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, value)
}
func (value jsonStringSlice) Value() (driver.Value, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}
func toJSONMap(input map[string]string) jsonMap {
	output := jsonMap{}
	for key, value := range input {
		output[key] = value
	}
	return output
}

type alertRuleRow struct {
	ID        uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string          `gorm:"column:name"`
	ClusterID uint64          `gorm:"column:cluster_id"`
	Severity  string          `gorm:"column:severity"`
	Condition string          `gorm:"column:condition_text"`
	Duration  string          `gorm:"column:duration"`
	Receivers jsonStringSlice `gorm:"column:receivers_json;type:json"`
	Enabled   bool            `gorm:"column:enabled"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time       `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt *time.Time      `gorm:"column:deleted_at"`
}

func (alertRuleRow) TableName() string { return "monitor_alert_rules" }
