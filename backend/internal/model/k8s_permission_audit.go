package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type JSONStringSlice []string

func (j *JSONStringSlice) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONStringSlice) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

type K8sPermissionAudit struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	SourceType  string     `gorm:"column:source_type;type:varchar(32);not null;index:idx_k8s_permission_audits_source_type"`
	ClusterID   *uint64    `gorm:"column:cluster_id;index:idx_k8s_permission_audits_cluster_id"`
	ClusterName string     `gorm:"column:cluster_name;type:varchar(128);not null;default:''"`
	DisplayName string     `gorm:"column:display_name;type:varchar(160);not null;default:''"`
	Status      string     `gorm:"column:status;type:varchar(32);not null;index:idx_k8s_permission_audits_status"`
	Mode        string     `gorm:"column:mode;type:varchar(32);not null;default:full"`
	TaskID      *uint64    `gorm:"column:task_id;index:idx_k8s_permission_audits_task_id"`
	RequestJSON JSONMap    `gorm:"column:request_json;type:json"`
	SummaryJSON JSONMap    `gorm:"column:summary_json;type:json"`
	StatsJSON   JSONMap    `gorm:"column:stats_json;type:json"`
	ErrorJSON   JSONMap    `gorm:"column:error_json;type:json"`
	CreatedBy   uint64     `gorm:"column:created_by;not null;default:0;index:idx_k8s_permission_audits_created_by"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_k8s_permission_audits_created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   *time.Time `gorm:"column:deleted_at;index"`
}

func (K8sPermissionAudit) TableName() string { return "k8s_permission_audits" }

type K8sPermissionAuditFinding struct {
	ID                               uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	AuditID                          uint64          `gorm:"column:audit_id;not null;index:idx_k8s_permission_audit_findings_audit_id"`
	FindingType                      string          `gorm:"column:finding_type;type:varchar(32);not null;index:idx_k8s_permission_audit_findings_type"`
	RiskLevel                        string          `gorm:"column:risk_level;type:varchar(16);not null;default:low;index:idx_k8s_permission_audit_findings_risk_level"`
	OwnershipClass                   string          `gorm:"column:ownership_class;type:varchar(16);not null;default:unrelated;index:idx_k8s_permission_audit_findings_ownership"`
	PrivilegeClass                   string          `gorm:"column:privilege_class;type:varchar(32);not null;default:''"`
	Scope                            string          `gorm:"column:scope;type:varchar(16);not null;default:''"`
	DeploymentBlocker                bool            `gorm:"column:deployment_blocker;not null;default:false"`
	DependsOnSharedClusterCapability bool            `gorm:"column:depends_on_shared_cluster_capability;not null;default:false"`
	APIVersion                       string          `gorm:"column:api_version;type:varchar(64);not null;default:''"`
	Kind                             string          `gorm:"column:kind;type:varchar(64);not null;default:'';index:idx_k8s_permission_audit_findings_kind"`
	Namespace                        string          `gorm:"column:namespace;type:varchar(128);not null;default:'';index:idx_k8s_permission_audit_findings_namespace"`
	Name                             string          `gorm:"column:name;type:varchar(160);not null;default:''"`
	WorkloadKind                     string          `gorm:"column:workload_kind;type:varchar(64);not null;default:''"`
	WorkloadName                     string          `gorm:"column:workload_name;type:varchar(160);not null;default:''"`
	ServiceAccountName               string          `gorm:"column:service_account_name;type:varchar(160);not null;default:''"`
	Summary                          string          `gorm:"column:summary;type:varchar(512);not null;default:''"`
	ReasonCodes                      JSONStringSlice `gorm:"column:reason_codes_json;type:json"`
	DetailJSON                       JSONMap         `gorm:"column:detail_json;type:json"`
	CreatedAt                        time.Time       `gorm:"column:created_at;autoCreateTime"`
}

func (K8sPermissionAuditFinding) TableName() string { return "k8s_permission_audit_findings" }
