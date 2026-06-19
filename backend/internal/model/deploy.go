package model

import "time"

type DeployServer struct {
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name          string     `gorm:"column:name"`
	IP            string     `gorm:"column:ip"`
	SSHPort       int        `gorm:"column:ssh_port"`
	User          string     `gorm:"column:user"`
	AuthType      string     `gorm:"column:auth_type"`
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
	ID          uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	Name        string          `gorm:"column:name"`
	ClusterName string          `gorm:"column:cluster_name"`
	K8sVersion  string          `gorm:"column:k8s_version"`
	PodCIDR     string          `gorm:"column:pod_cidr"`
	SvcCIDR     string          `gorm:"column:svc_cidr"`
	CNIType     string          `gorm:"column:cni_type"`
	CNIConfig   JSONMap         `gorm:"column:cni_config;type:json"`
	Addons      JSONStringSlice `gorm:"column:addons;type:json"`
	Status      string          `gorm:"column:status"`
	TaskID      *uint64         `gorm:"column:task_id"`
	ClusterID   *uint64         `gorm:"column:cluster_id"`
	CreatedBy   uint64          `gorm:"column:created_by"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   *time.Time      `gorm:"column:deleted_at"`
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
