package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// JSONStringSlice is the JSON-column value object used by deployment-plan
// lists such as add-ons and preflight ignore rules.
type JSONStringSlice []string

func (j *JSONStringSlice) Scan(value any) error {
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

// DeployServer is a target host registered for cluster provisioning. Secrets
// remain encrypted at rest in CredentialEnc and are never part of API DTOs.
type DeployServer struct {
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name          string     `gorm:"column:name"`
	IP            string     `gorm:"column:ip"`
	SSHPort       int        `gorm:"column:ssh_port"`
	User          string     `gorm:"column:user"`
	AuthType      string     `gorm:"column:auth_type"`
	CredentialID  *uint64    `gorm:"column:credential_id"`
	CredentialEnc string     `gorm:"column:credential_enc"`
	OS            *string    `gorm:"column:os"`
	OSVersion     *string    `gorm:"column:os_version"`
	Kernel        *string    `gorm:"column:kernel"`
	CPUCores      *uint      `gorm:"column:cpu_cores"`
	MemoryMB      *uint64    `gorm:"column:memory_mb"`
	DiskGB        *uint64    `gorm:"column:disk_gb"`
	Status        string     `gorm:"column:status"`
	Labels        JSONMap    `gorm:"column:labels;type:json"`
	Remark        *string    `gorm:"column:remark"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt     *time.Time `gorm:"column:deleted_at"`
}

func (DeployServer) TableName() string { return "deploy_servers" }

type SSHCredential struct {
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name          string     `gorm:"column:name"`
	AuthType      string     `gorm:"column:auth_type"`
	Username      string     `gorm:"column:username"`
	CredentialEnc string     `gorm:"column:credential_enc"`
	Remark        *string    `gorm:"column:remark"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt     *time.Time `gorm:"column:deleted_at"`
}

func (SSHCredential) TableName() string { return "ssh_credentials" }

// DeployPlan owns the desired cluster-installation configuration and its
// execution state. Transport and Ansible execution remain adapter concerns.
type DeployPlan struct {
	ID               uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	Name             string          `gorm:"column:name"`
	ClusterName      string          `gorm:"column:cluster_name"`
	K8sVersion       string          `gorm:"column:k8s_version"`
	PodCIDR          string          `gorm:"column:pod_cidr"`
	SvcCIDR          string          `gorm:"column:svc_cidr"`
	CNIType          string          `gorm:"column:cni_type"`
	CNIConfig        JSONMap         `gorm:"column:cni_config;type:json"`
	Addons           JSONStringSlice `gorm:"column:addons;type:json"`
	HelmInstall      bool            `gorm:"column:helm_install;default:false"`
	StepOverrides    JSONMap         `gorm:"column:step_overrides;type:json"`
	PreflightIgnores JSONStringSlice `gorm:"column:preflight_ignores;type:json"`
	Status           string          `gorm:"column:status"`
	TaskID           *uint64         `gorm:"column:task_id"`
	ClusterID        *uint64         `gorm:"column:cluster_id"`
	CreatedBy        uint64          `gorm:"column:created_by"`
	CreatedAt        time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time       `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt        *time.Time      `gorm:"column:deleted_at"`
}

func (DeployPlan) TableName() string { return "deploy_plans" }

type DeployPlanNode struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	PlanID    uint64    `gorm:"column:plan_id"`
	ServerID  uint64    `gorm:"column:server_id"`
	Role      string    `gorm:"column:role"`
	SortOrder int       `gorm:"column:sort_order"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (DeployPlanNode) TableName() string { return "deploy_plan_nodes" }

type DeployPlanStepOverride struct {
	StepKey         string  `json:"step_key"`
	NodeRole        string  `json:"node_role,omitempty"`
	NodeServerID    *uint64 `json:"node_server_id,omitempty"`
	CommandTemplate string  `json:"command_template"`
	Description     *string `json:"description,omitempty"`
	TimeoutSeconds  *int    `json:"timeout_seconds,omitempty"`
	RetryCount      *int    `json:"retry_count,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
}

type DeployLog struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	PlanID    uint64    `gorm:"column:plan_id"`
	TaskID    *uint64   `gorm:"column:task_id"`
	StepName  *string   `gorm:"column:step_name"`
	ServerIP  *string   `gorm:"column:server_ip"`
	Level     string    `gorm:"column:level"`
	Message   string    `gorm:"column:message"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (DeployLog) TableName() string { return "deploy_logs" }
