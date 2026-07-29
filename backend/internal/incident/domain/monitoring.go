package domain

import (
	"fmt"
	"strings"
	"time"
)

type AlertRule struct {
	ID          uint64
	Name        string
	ClusterID   uint64
	ClusterName string
	Severity    string
	Condition   string
	Duration    string
	Receivers   []string
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

func NewAlertRule(name string, clusterID uint64, severity, condition, duration string, receivers []string) (AlertRule, error) {
	name = strings.TrimSpace(name)
	if name == "" || clusterID == 0 {
		return AlertRule{}, fmt.Errorf("%w: 规则名称和集群不能为空", ErrValidation)
	}
	severity = NormalizeSeverity(severity)
	if severity != "info" && severity != "warning" && severity != "critical" {
		return AlertRule{}, fmt.Errorf("%w: 不支持的告警级别", ErrValidation)
	}
	duration = strings.TrimSpace(duration)
	if duration == "" {
		duration = "5m"
	}
	return AlertRule{Name: name, ClusterID: clusterID, Severity: severity, Condition: strings.TrimSpace(condition), Duration: duration, Receivers: receivers, Enabled: true}, nil
}

type AlertmanagerAlert struct {
	Status      string
	Fingerprint string
	StartsAt    time.Time
	EndsAt      time.Time
	Labels      map[string]string
	Annotations map[string]string
}

func NormalizeSeverity(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "critical" || value == "warning" || value == "info" {
		return value
	}
	return "warning"
}

type LegacyIncident struct {
	ID               uint64
	AlertName        string
	ClusterID        uint64
	ClusterName      string
	Namespace        string
	ResourceKind     string
	ResourceName     string
	Severity         string
	Status           string
	Summary          string
	StartedAt        time.Time
	ResolvedAt       *time.Time
	AIConversationID *uint64
	AIProposalID     *uint64
	AssigneeName     string
	VerificationNote string
	Version          uint64
}
