package domain

import "time"

type Pipeline struct {
	ID          uint64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name        string     `json:"name" gorm:"column:name"`
	Description string     `json:"description" gorm:"column:description"`
	TriggerType string     `json:"trigger_type" gorm:"column:trigger_type"`
	Branches    string     `json:"branches" gorm:"column:branches"`
	Cron        string     `json:"cron" gorm:"column:cron"`
	ClusterID   *uint64    `json:"cluster_id" gorm:"column:cluster_id"`
	Namespace   string     `json:"namespace" gorm:"column:namespace"`
	RunnerImage string     `json:"runner_image" gorm:"column:runner_image"`
	ConfigYAML  string     `json:"config_yaml" gorm:"column:config_yaml"`
	Status      string     `json:"status" gorm:"column:status"`
	LastRunID   *uint64    `json:"last_run_id" gorm:"column:last_run_id"`
	LastRunAt   *time.Time `json:"last_run_at,omitempty" gorm:"-"`
	CreatedBy   uint64     `json:"created_by" gorm:"column:created_by"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"column:updated_at"`
}

type Run struct {
	ID                uint64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PipelineID        uint64     `json:"pipeline_id" gorm:"column:pipeline_id"`
	PipelineName      string     `json:"pipeline_name" gorm:"-"`
	TriggerType       string     `json:"trigger_type" gorm:"column:trigger_type"`
	CommitSHA         string     `json:"commit_sha" gorm:"column:commit_sha"`
	CommitMessage     string     `json:"commit_message" gorm:"column:commit_message"`
	Branch            string     `json:"branch" gorm:"column:branch"`
	Status            string     `json:"status" gorm:"column:status"`
	ExecutorJobName   string     `json:"executor_job_name,omitempty" gorm:"column:executor_job_name"`
	ExecutorNamespace string     `json:"executor_namespace,omitempty" gorm:"column:executor_namespace"`
	ExecutorClusterID *uint64    `json:"executor_cluster_id,omitempty" gorm:"column:executor_cluster_id"`
	ErrorMessage      *string    `json:"error_message" gorm:"column:error_message"`
	StartedAt         *time.Time `json:"started_at" gorm:"column:started_at"`
	FinishedAt        *time.Time `json:"finished_at" gorm:"column:finished_at"`
	CreatedAt         time.Time  `json:"created_at" gorm:"column:created_at"`
	Stages            []Stage    `json:"stages" gorm:"-"`
}

type Stage struct {
	ID         uint64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	RunID      uint64     `json:"run_id" gorm:"column:run_id"`
	StageKey   string     `json:"stage_key" gorm:"column:stage_key"`
	Name       string     `json:"name" gorm:"column:name"`
	Status     string     `json:"status" gorm:"column:status"`
	Log        string     `json:"log" gorm:"column:log"`
	StartedAt  *time.Time `json:"started_at" gorm:"column:started_at"`
	FinishedAt *time.Time `json:"finished_at" gorm:"column:finished_at"`
	SortOrder  int        `json:"sort_order" gorm:"column:sort_order"`
	Steps      []Step     `json:"steps" gorm:"-"`
}

type Step struct {
	ID         uint64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	RunID      uint64     `json:"run_id" gorm:"column:run_id"`
	StageKey   string     `json:"stage_key" gorm:"column:stage_key"`
	StepKey    string     `json:"step_key" gorm:"column:step_key"`
	Name       string     `json:"name" gorm:"column:name"`
	Plugin     string     `json:"plugin" gorm:"column:plugin"`
	Status     string     `json:"status" gorm:"column:status"`
	Log        string     `json:"log" gorm:"column:log"`
	StartedAt  *time.Time `json:"started_at" gorm:"column:started_at"`
	FinishedAt *time.Time `json:"finished_at" gorm:"column:finished_at"`
	SortOrder  int        `json:"sort_order" gorm:"column:sort_order"`
}

type Artifact struct {
	ID           uint64    `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	RunID        *uint64   `json:"run_id" gorm:"column:run_id"`
	PipelineID   *uint64   `json:"pipeline_id" gorm:"column:pipeline_id"`
	Name         string    `json:"name" gorm:"column:name"`
	ArtifactType string    `json:"artifact_type" gorm:"column:artifact_type"`
	Version      string    `json:"version" gorm:"column:version"`
	Repository   string    `json:"repository" gorm:"column:repository"`
	SizeBytes    uint64    `json:"size_bytes" gorm:"column:size_bytes"`
	Digest       string    `json:"digest" gorm:"column:digest"`
	Status       string    `json:"status" gorm:"column:status"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
}

type Environment struct {
	ID              uint64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name            string     `json:"name" gorm:"column:name"`
	Label           string     `json:"label" gorm:"column:label"`
	EnvironmentType string     `json:"environment_type" gorm:"column:environment_type"`
	ClusterID       *uint64    `json:"cluster_id" gorm:"column:cluster_id"`
	Namespace       string     `json:"namespace" gorm:"column:namespace"`
	CurrentVersion  string     `json:"current_version" gorm:"column:current_version"`
	Status          string     `json:"status" gorm:"column:status"`
	LastRunID       *uint64    `json:"last_run_id" gorm:"column:last_run_id"`
	DeployedBy      string     `json:"deployed_by" gorm:"column:deployed_by"`
	DeployCount     int        `json:"deploy_count" gorm:"column:deploy_count"`
	LastDeployedAt  *time.Time `json:"last_deployed_at" gorm:"column:last_deployed_at"`
	CreatedAt       time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"column:updated_at"`
}

func (Pipeline) TableName() string    { return "cicd_pipelines" }
func (Run) TableName() string         { return "cicd_runs" }
func (Step) TableName() string        { return "cicd_run_steps" }
func (Stage) TableName() string       { return "cicd_run_stages" }
func (Artifact) TableName() string    { return "cicd_artifacts" }
func (Environment) TableName() string { return "cicd_environments" }
