package model

import "time"

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

// DeployConfig 部署流程配置模型
type DeployConfig struct {
	ID              uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	StepKey         string    `gorm:"column:step_key" json:"step_key"`
	OSType          string    `gorm:"column:os_type;default:ubuntu" json:"os_type"`
	StepName        string    `gorm:"column:step_name" json:"step_name"`
	StepOrder       int       `gorm:"column:step_order" json:"step_order"`
	CommandTemplate string    `gorm:"column:command_template;type:text" json:"command_template"`
	Description     *string   `gorm:"column:description;type:text" json:"description"`
	Enabled         bool      `gorm:"column:enabled;default:true" json:"enabled"`
	TimeoutSeconds  int       `gorm:"column:timeout_seconds;default:600" json:"timeout_seconds"`
	RetryCount      int       `gorm:"column:retry_count;default:0" json:"retry_count"`
	CreatedBy       uint64    `gorm:"column:created_by" json:"created_by"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (DeployConfig) TableName() string { return "deploy_configs" }

// DeployConfigVersion 部署配置版本历史模型
type DeployConfigVersion struct {
	ID              uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ConfigID        uint64    `gorm:"column:config_id" json:"config_id"`
	StepKey         string    `gorm:"column:step_key" json:"step_key"`
	CommandTemplate string    `gorm:"column:command_template;type:text" json:"command_template"`
	Description     *string   `gorm:"column:description;type:text" json:"description"`
	ChangeType      string    `gorm:"column:change_type" json:"change_type"`
	ChangedBy       uint64    `gorm:"column:changed_by" json:"changed_by"`
	ChangedAt       time.Time `gorm:"column:changed_at;autoCreateTime" json:"changed_at"`
	ChangeSummary   *string   `gorm:"column:change_summary" json:"change_summary"`
}

func (DeployConfigVersion) TableName() string { return "deploy_config_versions" }

// DeployRepository 仓库配置模型
type DeployRepository struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RepoType    string     `gorm:"column:repo_type" json:"repo_type"`
	Name        string     `gorm:"column:name" json:"name"`
	URL         string     `gorm:"column:url" json:"url"`
	Description *string    `gorm:"column:description;type:text" json:"description"`
	AuthType    string     `gorm:"column:auth_type;default:none" json:"auth_type"`
	AuthConfig  JSONMap    `gorm:"column:auth_config;type:json" json:"auth_config"`
	Priority    int        `gorm:"column:priority;default:100" json:"priority"`
	Enabled     bool       `gorm:"column:enabled;default:true" json:"enabled"`
	IsDefault   bool       `gorm:"column:is_default;default:false" json:"is_default"`
	MirrorOf    *string    `gorm:"column:mirror_of" json:"mirror_of"`
	CreatedBy   uint64     `gorm:"column:created_by" json:"created_by"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt   *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (DeployRepository) TableName() string { return "deploy_repositories" }
