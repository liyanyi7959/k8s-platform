package application

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"k8s-platform-backend/internal/cicd/domain"
	"strings"
	"time"
)

var ErrNotFound = errors.New("cicd resource not found")
var ErrConflict = errors.New("cicd resource conflict")

type Service struct {
	db       *gorm.DB
	executor JobExecutor
}

func NewService(db *gorm.DB, executors ...JobExecutor) *Service {
	var executor JobExecutor
	if len(executors) > 0 {
		executor = executors[0]
	}
	return &Service{db: db, executor: executor}
}

type Page[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}
type PipelineInput struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	TriggerType string  `json:"trigger_type"`
	Branches    string  `json:"branches"`
	Cron        string  `json:"cron"`
	ClusterID   *uint64 `json:"cluster_id"`
	Namespace   string  `json:"namespace"`
	RunnerImage string  `json:"runner_image"`
	ConfigYAML  string  `json:"config_yaml"`
}
type EnvironmentInput struct {
	Name            string  `json:"name"`
	Label           string  `json:"label"`
	EnvironmentType string  `json:"environment_type"`
	Namespace       string  `json:"namespace"`
	ClusterID       *uint64 `json:"cluster_id"`
}

type JobRequest struct {
	RunID        uint64
	PipelineID   uint64
	PipelineName string
	ClusterID    uint64
	Namespace    string
	RunnerImage  string
	ConfigYAML   string
}
type JobRef struct {
	Name      string
	Namespace string
	ClusterID uint64
}
type JobProgress struct {
	Status    string
	Log       string
	StageKey  string
	StepKey   string
	Finished  bool
	Succeeded bool
}
type JobExecutor interface {
	Submit(context.Context, JobRequest) (JobRef, error)
	Wait(context.Context, JobRef, func(JobProgress)) error
	Cancel(context.Context, JobRef) error
}

func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}
	return page, size
}
func (s *Service) ListPipelines(ctx context.Context, page, size int, keyword, status string) (Page[domain.Pipeline], error) {
	page, size = normalizePage(page, size)
	q := s.db.WithContext(ctx).Model(&domain.Pipeline{})
	if keyword != "" {
		q = q.Where("name LIKE ?", "%"+keyword+"%")
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return Page[domain.Pipeline]{}, err
	}
	var rows []domain.Pipeline
	if err := q.Order("updated_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return Page[domain.Pipeline]{}, err
	}
	for i := range rows {
		if rows[i].LastRunID == nil {
			continue
		}
		var run domain.Run
		if s.db.WithContext(ctx).Select("started_at").First(&run, *rows[i].LastRunID).Error == nil {
			rows[i].LastRunAt = run.StartedAt
		}
	}
	return Page[domain.Pipeline]{rows, total, page, size}, nil
}
func (s *Service) CreatePipeline(ctx context.Context, in PipelineInput, userID uint64) (domain.Pipeline, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return domain.Pipeline{}, errors.New("pipeline name is required")
	}
	if in.TriggerType == "" {
		in.TriggerType = "manual"
	}
	if in.ConfigYAML == "" {
		in.ConfigYAML = "stages: []"
	}
	if in.Namespace == "" {
		in.Namespace = "cicd"
	}
	if in.RunnerImage == "" {
		in.RunnerImage = "alpine:3.20"
	}
	p := domain.Pipeline{Name: in.Name, Description: in.Description, TriggerType: in.TriggerType, Branches: in.Branches, Cron: in.Cron, ClusterID: in.ClusterID, Namespace: in.Namespace, RunnerImage: in.RunnerImage, ConfigYAML: in.ConfigYAML, Status: "idle", CreatedBy: userID}
	if err := s.db.WithContext(ctx).Create(&p).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return p, ErrConflict
		}
		return p, err
	}
	return p, nil
}
func (s *Service) GetPipeline(ctx context.Context, id uint64) (domain.Pipeline, error) {
	var p domain.Pipeline
	err := s.db.WithContext(ctx).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return p, ErrNotFound
	}
	return p, err
}
func (s *Service) UpdatePipeline(ctx context.Context, id uint64, in PipelineInput) error {
	r := s.db.WithContext(ctx).Model(&domain.Pipeline{}).Where("id = ?", id).Updates(map[string]any{"name": strings.TrimSpace(in.Name), "description": in.Description, "trigger_type": in.TriggerType, "branches": in.Branches, "cron": in.Cron, "cluster_id": in.ClusterID, "namespace": in.Namespace, "runner_image": in.RunnerImage, "config_yaml": in.ConfigYAML})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Service) DeletePipeline(ctx context.Context, id uint64) error {
	r := s.db.WithContext(ctx).Delete(&domain.Pipeline{}, id)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) ListRuns(ctx context.Context, page, size int, pipelineID uint64, status, keyword string) (Page[domain.Run], error) {
	page, size = normalizePage(page, size)
	q := s.db.WithContext(ctx).Model(&domain.Run{})
	if pipelineID > 0 {
		q = q.Where("pipeline_id = ?", pipelineID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if keyword != "" {
		q = q.Where("commit_message LIKE ? OR commit_sha LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return Page[domain.Run]{}, err
	}
	var rows []domain.Run
	if err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return Page[domain.Run]{}, err
	}
	for i := range rows {
		var p domain.Pipeline
		if s.db.Select("name").First(&p, rows[i].PipelineID).Error == nil {
			rows[i].PipelineName = p.Name
		}
	}
	return Page[domain.Run]{rows, total, page, size}, nil
}
func (s *Service) GetRun(ctx context.Context, id uint64) (domain.Run, error) {
	var r domain.Run
	err := s.db.WithContext(ctx).First(&r, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	var p domain.Pipeline
	if s.db.Select("name").First(&p, r.PipelineID).Error == nil {
		r.PipelineName = p.Name
	}
	s.db.WithContext(ctx).Where("run_id = ?", id).Order("sort_order").Find(&r.Stages)
	var steps []domain.Step
	s.db.WithContext(ctx).Where("run_id = ?", id).Order("sort_order").Find(&steps)
	for i := range r.Stages {
		for j := range steps {
			if steps[j].StageKey == r.Stages[i].StageKey {
				r.Stages[i].Steps = append(r.Stages[i].Steps, steps[j])
			}
		}
	}
	return r, nil
}
func (s *Service) CreateRun(ctx context.Context, pipelineID uint64, trigger, branch, sha, message string) (domain.Run, error) {
	p, err := s.GetPipeline(ctx, pipelineID)
	if err != nil {
		return domain.Run{}, err
	}
	workflow, err := ParseWorkflow(p.ConfigYAML)
	if err != nil {
		return domain.Run{}, err
	}
	now := time.Now().UTC()
	r := domain.Run{PipelineID: p.ID, TriggerType: trigger, Branch: branch, CommitSHA: sha, CommitMessage: message, Status: "running", StartedAt: &now}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&r).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Pipeline{}).Where("id = ?", p.ID).Updates(map[string]any{"status": "running", "last_run_id": r.ID}).Error; err != nil {
			return err
		}
		stages := make([]domain.Stage, 0, len(workflow.Stages))
		steps := make([]domain.Step, 0)
		for stageIndex, stage := range workflow.Stages {
			stages = append(stages, domain.Stage{RunID: r.ID, StageKey: stage.Key, Name: stage.Name, Status: "queued", Log: "waiting for Kubernetes Job", SortOrder: stageIndex})
			for stepIndex, step := range stage.Steps {
				steps = append(steps, domain.Step{RunID: r.ID, StageKey: stage.Key, StepKey: step.Key, Name: step.Name, Plugin: step.Plugin, Status: "queued", Log: "queued", SortOrder: stepIndex})
			}
		}
		if len(stages) == 0 {
			stages = append(stages, domain.Stage{RunID: r.ID, StageKey: "build", Name: "Build", Status: "queued", Log: "build queued", SortOrder: 0})
		}
		if err := tx.Create(&stages).Error; err != nil {
			return err
		}
		if len(steps) > 0 {
			return tx.Create(&steps).Error
		}
		return nil
	})
	if err != nil {
		return domain.Run{}, err
	}
	if s.executor == nil {
		s.markRunFailed(r.ID, p.ID, "kubernetes job executor is not configured")
		return s.GetRun(ctx, r.ID)
	}
	if valueOf(p.ClusterID) == 0 {
		s.markRunFailed(r.ID, p.ID, "pipeline cluster_id is required")
		return s.GetRun(ctx, r.ID)
	}
	ref, submitErr := s.executor.Submit(ctx, JobRequest{RunID: r.ID, PipelineID: p.ID, PipelineName: p.Name, ClusterID: valueOf(p.ClusterID), Namespace: p.Namespace, RunnerImage: p.RunnerImage, ConfigYAML: p.ConfigYAML})
	if submitErr != nil {
		s.markRunFailed(r.ID, p.ID, submitErr.Error())
		return s.GetRun(ctx, r.ID)
	}
	s.db.WithContext(ctx).Model(&domain.Run{}).Where("id = ?", r.ID).Updates(map[string]any{"executor_job_name": ref.Name, "executor_namespace": ref.Namespace, "executor_cluster_id": ref.ClusterID})
	go s.monitorRun(r.ID, p.ID, ref)
	return s.GetRun(ctx, r.ID)
}
func valueOf(value *uint64) uint64 {
	if value == nil {
		return 0
	}
	return *value
}
func (s *Service) monitorRun(id, pipelineID uint64, ref JobRef) {
	if err := s.executor.Wait(context.Background(), ref, func(progress JobProgress) {
		updates := map[string]any{"status": progress.Status}
		if progress.Log != "" {
			updates["log"] = progress.Log
		}
		if progress.StepKey != "" {
			s.db.Model(&domain.Step{}).Where("run_id = ? AND step_key = ?", id, progress.StepKey).Updates(updates)
		}
		if progress.StageKey != "" {
			s.db.Model(&domain.Stage{}).Where("run_id = ? AND stage_key = ?", id, progress.StageKey).Updates(updates)
		}
	}); err != nil {
		var current domain.Run
		if s.db.Select("status").First(&current, id).Error == nil && current.Status == "canceled" {
			return
		}
		s.markRunFailed(id, pipelineID, err.Error())
		return
	}
	var current domain.Run
	if s.db.Select("status").First(&current, id).Error == nil && current.Status == "canceled" {
		return
	}
	now := time.Now().UTC()
	s.db.Model(&domain.Stage{}).Where("run_id = ? AND status NOT IN ('failed','canceled')", id).Updates(map[string]any{"status": "success", "finished_at": now})
	s.db.Model(&domain.Step{}).Where("run_id = ? AND status NOT IN ('failed','canceled')", id).Updates(map[string]any{"status": "success", "finished_at": now})
	s.db.Model(&domain.Run{}).Where("id = ?", id).Updates(map[string]any{"status": "success", "finished_at": now})
	s.db.Model(&domain.Pipeline{}).Where("id = ?", pipelineID).Updates(map[string]any{"status": "success"})
}
func (s *Service) markRunFailed(id, pipelineID uint64, message string) {
	now := time.Now().UTC()
	s.db.Model(&domain.Stage{}).Where("run_id = ? AND status NOT IN ('success','failed','canceled')", id).Updates(map[string]any{"status": "failed", "log": message, "finished_at": now})
	s.db.Model(&domain.Step{}).Where("run_id = ? AND status NOT IN ('success','failed','canceled')", id).Updates(map[string]any{"status": "failed", "log": message, "finished_at": now})
	s.db.Model(&domain.Run{}).Where("id = ?", id).Updates(map[string]any{"status": "failed", "error_message": message, "finished_at": now})
	s.db.Model(&domain.Pipeline{}).Where("id = ?", pipelineID).Updates(map[string]any{"status": "failed"})
}
func (s *Service) CancelRun(ctx context.Context, id uint64) error {
	if s.executor != nil {
		var run domain.Run
		if s.db.WithContext(ctx).First(&run, id).Error == nil && run.Status == "running" {
			_ = s.executor.Cancel(ctx, JobRef{Name: run.ExecutorJobName, Namespace: run.ExecutorNamespace, ClusterID: valueOf(run.ExecutorClusterID)})
		}
	}
	r := s.db.WithContext(ctx).Model(&domain.Run{}).Where("id = ? AND status IN ('queued','running')", id).Updates(map[string]any{"status": "canceled", "finished_at": time.Now().UTC()})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}

func (s *Service) ListArtifacts(ctx context.Context, page, size int, typ, keyword string) (Page[domain.Artifact], error) {
	page, size = normalizePage(page, size)
	q := s.db.WithContext(ctx).Model(&domain.Artifact{})
	if typ != "" {
		q = q.Where("artifact_type = ?", typ)
	}
	if keyword != "" {
		q = q.Where("name LIKE ? OR version LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return Page[domain.Artifact]{}, err
	}
	var rows []domain.Artifact
	if err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return Page[domain.Artifact]{}, err
	}
	return Page[domain.Artifact]{rows, total, page, size}, nil
}
func (s *Service) GetArtifact(ctx context.Context, id uint64) (domain.Artifact, error) {
	var a domain.Artifact
	err := s.db.WithContext(ctx).First(&a, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return a, ErrNotFound
	}
	return a, err
}
func (s *Service) ListEnvironments(ctx context.Context) ([]domain.Environment, error) {
	var rows []domain.Environment
	return rows, s.db.WithContext(ctx).Order("name").Find(&rows).Error
}
func (s *Service) CreateEnvironment(ctx context.Context, in EnvironmentInput) (domain.Environment, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return domain.Environment{}, errors.New("environment name is required")
	}
	if in.EnvironmentType == "" {
		in.EnvironmentType = "development"
	}
	if in.Namespace == "" {
		in.Namespace = "default"
	}
	e := domain.Environment{Name: in.Name, Label: in.Label, EnvironmentType: in.EnvironmentType, Namespace: in.Namespace, ClusterID: in.ClusterID, Status: "idle"}
	if err := s.db.WithContext(ctx).Create(&e).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return e, ErrConflict
		}
		return e, err
	}
	return e, nil
}
func (s *Service) UpdateEnvironment(ctx context.Context, id uint64, in EnvironmentInput) error {
	r := s.db.WithContext(ctx).Model(&domain.Environment{}).Where("id = ?", id).Updates(map[string]any{"name": strings.TrimSpace(in.Name), "label": in.Label, "environment_type": in.EnvironmentType, "namespace": in.Namespace, "cluster_id": in.ClusterID})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Service) DeleteEnvironment(ctx context.Context, id uint64) error {
	r := s.db.WithContext(ctx).Delete(&domain.Environment{}, id)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Service) GetEnvironment(ctx context.Context, id uint64) (domain.Environment, error) {
	var e domain.Environment
	err := s.db.WithContext(ctx).First(&e, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return e, ErrNotFound
	}
	return e, err
}
func (s *Service) Summary(ctx context.Context) (map[string]any, error) {
	var pipelines, runs, artifacts, success int64
	since := time.Now().UTC().AddDate(0, 0, -30)
	if err := s.db.WithContext(ctx).Model(&domain.Pipeline{}).Count(&pipelines).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&domain.Run{}).Where("created_at >= ?", since).Count(&runs).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&domain.Artifact{}).Count(&artifacts).Error; err != nil {
		return nil, err
	}
	s.db.WithContext(ctx).Model(&domain.Run{}).Where("created_at >= ? AND status = 'success'", since).Count(&success)
	rate := float64(0)
	if runs > 0 {
		rate = float64(success) * 100 / float64(runs)
	}
	return map[string]any{"pipelines": pipelines, "runs30d": runs, "success_rate": rate, "artifacts": artifacts}, nil
}
