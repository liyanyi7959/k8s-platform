package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type AutomationJobItem struct {
	ID            uint64        `json:"id"`
	Name          string        `json:"name"`
	Mode          string        `json:"mode"`
	Type          string        `json:"type"`
	Env           string        `json:"env"`
	Status        string        `json:"status"`
	RiskLevel     string        `json:"risk_level"`
	ApprovalMode  string        `json:"approval_mode"`
	Strategy      string        `json:"strategy"`
	Concurrency   int           `json:"concurrency"`
	TimeoutSec    int           `json:"timeout_sec"`
	TemplateID    *uint64       `json:"template_id,omitempty"`
	TemplateName  *string       `json:"template_name,omitempty"`
	Cron          *string       `json:"cron,omitempty"`
	Targets       *string       `json:"targets,omitempty"`
	LimitSpec     *string       `json:"limit,omitempty"`
	Vars          model.JSONMap `json:"vars,omitempty"`
	ChangeWindow  *string       `json:"change_window,omitempty"`
	RollbackPlan  *string       `json:"rollback_plan,omitempty"`
	LastRunTaskID *uint64       `json:"last_run_task_id,omitempty"`
	LastRunStatus *string       `json:"last_run_status,omitempty"`
	LastRunAt     *string       `json:"last_run_at,omitempty"`
	CreatedBy     uint64        `json:"created_by"`
	UpdatedBy     uint64        `json:"updated_by"`
	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
}

type ListAutomationJobsRequest struct {
	Page         int
	PageSize     int
	Keyword      string
	Mode         string
	Type         string
	Env          string
	RiskLevel    string
	ApprovalMode string
	Status       string
	SortBy       string
	Order        string
}

type AutomationJobSummary struct {
	Total             int    `json:"total"`
	ProdJobs          int    `json:"prod_jobs"`
	ManualApproval    int    `json:"manual_approval"`
	RecentSuccessRate string `json:"recent_success_rate"`
}

type CreateAutomationJobRequest struct {
	Name         string
	Mode         string
	Type         string
	Env          string
	Status       string
	RiskLevel    string
	ApprovalMode string
	Strategy     string
	Concurrency  int
	TimeoutSec   int
	TemplateID   *uint64
	Cron         *string
	Targets      *string
	LimitSpec    *string
	Vars         model.JSONMap
	ChangeWindow *string
	RollbackPlan *string
	CreatedBy    uint64
}

type PatchAutomationJobRequest struct {
	Name         *string
	Mode         *string
	Type         *string
	Env          *string
	Status       *string
	RiskLevel    *string
	ApprovalMode *string
	Strategy     *string
	Concurrency  *int
	TimeoutSec   *int
	TemplateID   *uint64
	Cron         *string
	Targets      *string
	LimitSpec    *string
	Vars         *model.JSONMap
	ChangeWindow *string
	RollbackPlan *string
	UpdatedBy    uint64
}

type RunAutomationJobRequest struct {
	ServerIDs []uint64
	Version   string
	Params    model.JSONMap
	CreatedBy uint64
}

type BatchUpdateAutomationJobStatusRequest struct {
	IDs       []uint64
	Status    string
	UpdatedBy uint64
}

type BatchDeleteAutomationJobsRequest struct {
	IDs []uint64
}

type BatchRunAutomationJobsRequest struct {
	IDs       []uint64
	ServerIDs []uint64
	Version   string
	Params    model.JSONMap
	CreatedBy uint64
}

type BatchRunAutomationJobResult struct {
	JobID    uint64  `json:"job_id"`
	JobName  string  `json:"job_name"`
	TaskID   *uint64 `json:"task_id,omitempty"`
	Status   string  `json:"status"`
	Message  string  `json:"message,omitempty"`
	Executed bool    `json:"executed"`
}

type AutomationJobService struct {
	db          *gorm.DB
	playbookSvc *PlaybookTemplateService
	ansibleSvc  *AnsibleService
}

func NewAutomationJobService(db *gorm.DB, playbookSvc *PlaybookTemplateService, ansibleSvc *AnsibleService) *AutomationJobService {
	return &AutomationJobService{db: db, playbookSvc: playbookSvc, ansibleSvc: ansibleSvc}
}

type automationJobRow struct {
	model.AutomationJob
	TemplateName *string `gorm:"column:template_name"`
}

func (s *AutomationJobService) buildAutomationJobListQuery(ctx context.Context, req ListAutomationJobsRequest) *gorm.DB {
	q := s.db.WithContext(ctx).
		Table("automation_jobs aj").
		Select("aj.*, pt.name AS template_name").
		Joins("LEFT JOIN playbook_templates pt ON pt.id = aj.template_id AND pt.deleted_at IS NULL").
		Where("aj.deleted_at IS NULL")
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(aj.name LIKE ? OR pt.name LIKE ? OR aj.rollback_plan LIKE ?)", like, like, like)
	}
	if mode := strings.TrimSpace(req.Mode); mode != "" {
		q = q.Where("aj.mode = ?", mode)
	}
	if typ := strings.TrimSpace(req.Type); typ != "" {
		q = q.Where("aj.job_type = ?", typ)
	}
	if env := strings.TrimSpace(req.Env); env != "" {
		q = q.Where("aj.env = ?", env)
	}
	if risk := strings.TrimSpace(req.RiskLevel); risk != "" {
		q = q.Where("aj.risk_level = ?", risk)
	}
	if approval := strings.TrimSpace(req.ApprovalMode); approval != "" {
		q = q.Where("aj.approval_mode = ?", approval)
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		q = q.Where("aj.status = ?", status)
	}
	return q
}

func (s *AutomationJobService) List(ctx context.Context, req ListAutomationJobsRequest) (PageResult[AutomationJobItem], error) {
	if s.db == nil {
		return PageResult[AutomationJobItem]{}, errors.New("db is required")
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	q := s.buildAutomationJobListQuery(ctx, req)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[AutomationJobItem]{}, err
	}
	orderClause := "aj.id DESC"
	if strings.EqualFold(strings.TrimSpace(req.Order), "asc") {
		switch strings.TrimSpace(req.SortBy) {
		case "name":
			orderClause = "aj.name ASC"
		case "updated_at":
			orderClause = "aj.updated_at ASC"
		case "created_at":
			orderClause = "aj.created_at ASC"
		case "last_run_at":
			orderClause = "aj.last_run_at ASC"
		}
	} else {
		switch strings.TrimSpace(req.SortBy) {
		case "name":
			orderClause = "aj.name DESC"
		case "updated_at":
			orderClause = "aj.updated_at DESC"
		case "created_at":
			orderClause = "aj.created_at DESC"
		case "last_run_at":
			orderClause = "aj.last_run_at DESC"
		}
	}
	var rows []automationJobRow
	if err := q.Order(orderClause).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[AutomationJobItem]{}, err
	}
	out := make([]AutomationJobItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAutomationJobItem(row))
	}
	return PageResult[AutomationJobItem]{List: out, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *AutomationJobService) Summary(ctx context.Context, req ListAutomationJobsRequest) (AutomationJobSummary, error) {
	if s.db == nil {
		return AutomationJobSummary{}, errors.New("db is required")
	}

	summary := AutomationJobSummary{RecentSuccessRate: "—"}

	var total int64
	if err := s.buildAutomationJobListQuery(ctx, req).Count(&total).Error; err != nil {
		return AutomationJobSummary{}, err
	}
	summary.Total = int(total)

	if env := strings.TrimSpace(req.Env); env != "" && env != "prod" {
		summary.ProdJobs = 0
	} else {
		var prodCount int64
		prodReq := req
		prodReq.Env = "prod"
		if err := s.buildAutomationJobListQuery(ctx, prodReq).Count(&prodCount).Error; err != nil {
			return AutomationJobSummary{}, err
		}
		summary.ProdJobs = int(prodCount)
	}

	if approval := strings.TrimSpace(req.ApprovalMode); approval != "" && approval != "manual" {
		summary.ManualApproval = 0
	} else {
		var manualCount int64
		manualReq := req
		manualReq.ApprovalMode = "manual"
		if err := s.buildAutomationJobListQuery(ctx, manualReq).Count(&manualCount).Error; err != nil {
			return AutomationJobSummary{}, err
		}
		summary.ManualApproval = int(manualCount)
	}

	recentReq := req
	recentReq.Page = 1
	recentReq.PageSize = 8
	recentReq.SortBy = "last_run_at"
	recentReq.Order = "desc"
	recentPage, err := s.List(ctx, recentReq)
	if err != nil {
		return AutomationJobSummary{}, err
	}
	executed := make([]AutomationJobItem, 0, len(recentPage.List))
	for _, item := range recentPage.List {
		if item.LastRunStatus != nil && (*item.LastRunStatus == "success" || *item.LastRunStatus == "failed") {
			executed = append(executed, item)
		}
	}
	if len(executed) > 0 {
		success := 0
		for _, item := range executed {
			if item.LastRunStatus != nil && *item.LastRunStatus == "success" {
				success++
			}
		}
		summary.RecentSuccessRate = strconv.Itoa(int(float64(success)*100/float64(len(executed)))) + "%"
	}

	return summary, nil
}

func (s *AutomationJobService) Get(ctx context.Context, id uint64) (AutomationJobItem, error) {
	if s.db == nil {
		return AutomationJobItem{}, errors.New("db is required")
	}
	if id == 0 {
		return AutomationJobItem{}, ErrInvalidParams
	}
	var row automationJobRow
	err := s.db.WithContext(ctx).
		Table("automation_jobs aj").
		Select("aj.*, pt.name AS template_name").
		Joins("LEFT JOIN playbook_templates pt ON pt.id = aj.template_id AND pt.deleted_at IS NULL").
		Where("aj.deleted_at IS NULL AND aj.id = ?", id).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AutomationJobItem{}, ErrNotFound
		}
		return AutomationJobItem{}, err
	}
	return toAutomationJobItem(row), nil
}

func (s *AutomationJobService) Create(ctx context.Context, req CreateAutomationJobRequest) (uint64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	payload, err := buildAutomationJobModel(ctx, s.db, req)
	if err != nil {
		return 0, err
	}
	if err := s.db.WithContext(ctx).Create(&payload).Error; err != nil {
		return 0, err
	}
	return payload.ID, nil
}

func (s *AutomationJobService) Patch(ctx context.Context, id uint64, req PatchAutomationJobRequest) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrInvalidParams
	}
	var row model.AutomationJob
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return ErrWithMessage(ErrInvalidParams, "任务名称不能为空")
		}
		row.Name = name
	}
	if req.Mode != nil {
		mode := normalizeJobMode(*req.Mode)
		if mode == "" {
			return ErrWithMessage(ErrInvalidParams, "任务方式不合法")
		}
		row.Mode = mode
	}
	if req.Type != nil {
		typ := normalizeJobType(*req.Type)
		if typ == "" {
			return ErrWithMessage(ErrInvalidParams, "任务类型不合法")
		}
		row.JobType = typ
	}
	if req.Env != nil {
		env := normalizeJobEnv(*req.Env)
		if env == "" {
			return ErrWithMessage(ErrInvalidParams, "目标环境不合法")
		}
		row.Env = env
	}
	if req.Status != nil {
		status := normalizeJobStatus(*req.Status)
		if status == "" {
			return ErrWithMessage(ErrInvalidParams, "状态不合法")
		}
		row.Status = status
	}
	if req.RiskLevel != nil {
		risk := normalizeRiskLevel(*req.RiskLevel)
		if risk == "" {
			return ErrWithMessage(ErrInvalidParams, "风险等级不合法")
		}
		row.RiskLevel = risk
	}
	if req.ApprovalMode != nil {
		approval := normalizeApprovalMode(*req.ApprovalMode)
		if approval == "" {
			return ErrWithMessage(ErrInvalidParams, "审批方式不合法")
		}
		row.ApprovalMode = approval
	}
	if req.Strategy != nil {
		strategy := normalizeStrategy(*req.Strategy)
		if strategy == "" {
			return ErrWithMessage(ErrInvalidParams, "执行策略不合法")
		}
		row.Strategy = strategy
	}
	if req.Concurrency != nil {
		if *req.Concurrency <= 0 {
			return ErrWithMessage(ErrInvalidParams, "并发上限必须大于 0")
		}
		row.Concurrency = *req.Concurrency
	}
	if req.TimeoutSec != nil {
		if *req.TimeoutSec <= 0 {
			return ErrWithMessage(ErrInvalidParams, "执行超时必须大于 0")
		}
		row.TimeoutSec = *req.TimeoutSec
	}
	if req.TemplateID != nil {
		if *req.TemplateID == 0 {
			row.TemplateID = nil
		} else {
			if err := ensurePlaybookTemplateExists(ctx, s.db, *req.TemplateID); err != nil {
				return err
			}
			row.TemplateID = req.TemplateID
		}
	}
	if req.Cron != nil {
		row.Cron = trimStringPtr(req.Cron)
	}
	if req.Targets != nil {
		row.Targets = trimStringPtr(req.Targets)
	}
	if req.LimitSpec != nil {
		row.LimitSpec = trimStringPtr(req.LimitSpec)
	}
	if req.Vars != nil {
		row.Vars = *req.Vars
	}
	if req.ChangeWindow != nil {
		row.ChangeWindow = trimStringPtr(req.ChangeWindow)
	}
	if req.RollbackPlan != nil {
		row.RollbackPlan = trimStringPtr(req.RollbackPlan)
	}
	if err := validateAutomationJobRow(row); err != nil {
		return err
	}
	row.UpdatedBy = req.UpdatedBy
	return s.db.WithContext(ctx).Model(&model.AutomationJob{}).Where("deleted_at IS NULL AND id = ?", id).Updates(map[string]any{
		"name":          row.Name,
		"mode":          row.Mode,
		"job_type":      row.JobType,
		"env":           row.Env,
		"status":        row.Status,
		"risk_level":    row.RiskLevel,
		"approval_mode": row.ApprovalMode,
		"strategy":      row.Strategy,
		"concurrency":   row.Concurrency,
		"timeout_sec":   row.TimeoutSec,
		"template_id":   row.TemplateID,
		"cron":          row.Cron,
		"targets":       row.Targets,
		"limit_spec":    row.LimitSpec,
		"vars":          row.Vars,
		"change_window": row.ChangeWindow,
		"rollback_plan": row.RollbackPlan,
		"updated_by":    row.UpdatedBy,
	}).Error
}

func (s *AutomationJobService) Delete(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrInvalidParams
	}
	res := s.db.WithContext(ctx).Model(&model.AutomationJob{}).Where("deleted_at IS NULL AND id = ?", id).Update("deleted_at", time.Now().UTC())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *AutomationJobService) BatchUpdateStatus(ctx context.Context, req BatchUpdateAutomationJobStatusRequest) (int64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	ids := normalizeAutomationJobIDs(req.IDs)
	if len(ids) == 0 {
		return 0, ErrWithMessage(ErrInvalidParams, "请选择至少一个任务")
	}
	status := normalizeJobStatus(req.Status)
	if status == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "状态不合法")
	}
	res := s.db.WithContext(ctx).Model(&model.AutomationJob{}).
		Where("deleted_at IS NULL").
		Where("id IN ?", ids).
		Updates(map[string]any{
			"status":     status,
			"updated_by": req.UpdatedBy,
		})
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, ErrNotFound
	}
	return res.RowsAffected, nil
}

func (s *AutomationJobService) BatchDelete(ctx context.Context, req BatchDeleteAutomationJobsRequest) (int64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	ids := normalizeAutomationJobIDs(req.IDs)
	if len(ids) == 0 {
		return 0, ErrWithMessage(ErrInvalidParams, "请选择至少一个任务")
	}
	res := s.db.WithContext(ctx).Model(&model.AutomationJob{}).
		Where("deleted_at IS NULL").
		Where("id IN ?", ids).
		Update("deleted_at", time.Now().UTC())
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, ErrNotFound
	}
	return res.RowsAffected, nil
}

func (s *AutomationJobService) BatchRun(ctx context.Context, req BatchRunAutomationJobsRequest) ([]BatchRunAutomationJobResult, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	ids := normalizeAutomationJobIDs(req.IDs)
	if len(ids) == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "请选择至少一个任务")
	}
	if len(req.ServerIDs) == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "缺少执行目标")
	}
	results := make([]BatchRunAutomationJobResult, 0, len(ids))
	var successCount int
	for _, id := range ids {
		item, err := s.Get(ctx, id)
		if err != nil {
			results = append(results, BatchRunAutomationJobResult{
				JobID:    id,
				Status:   "failed",
				Message:  batchRunResultMessage(err),
				Executed: false,
			})
			continue
		}
		taskID, runErr := s.Run(ctx, id, RunAutomationJobRequest{
			ServerIDs: req.ServerIDs,
			Version:   req.Version,
			Params:    req.Params,
			CreatedBy: req.CreatedBy,
		})
		if runErr != nil {
			results = append(results, BatchRunAutomationJobResult{
				JobID:    id,
				JobName:  item.Name,
				Status:   "failed",
				Message:  batchRunResultMessage(runErr),
				Executed: false,
			})
			continue
		}
		tid := taskID
		results = append(results, BatchRunAutomationJobResult{
			JobID:    id,
			JobName:  item.Name,
			TaskID:   &tid,
			Status:   "submitted",
			Message:  "已提交执行",
			Executed: true,
		})
		successCount++
	}
	if successCount == 0 {
		return results, ErrWithMessage(ErrConflict, "批量执行全部失败")
	}
	return results, nil
}

func (s *AutomationJobService) Run(ctx context.Context, id uint64, req RunAutomationJobRequest) (uint64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	if s.playbookSvc == nil || s.ansibleSvc == nil {
		return 0, errors.New("automation runtime is not ready")
	}
	if id == 0 || len(req.ServerIDs) == 0 {
		return 0, ErrWithMessage(ErrInvalidParams, "缺少执行目标")
	}
	var row model.AutomationJob
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if row.TemplateID == nil || *row.TemplateID == 0 {
		return 0, ErrWithMessage(ErrInvalidParams, "当前任务未关联 Playbook 模板")
	}
	pl, err := s.playbookSvc.Get(ctx, *row.TemplateID)
	if err != nil {
		return 0, err
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		version = strings.TrimSpace(pl.CurrentVersion)
	}
	versions, err := s.playbookSvc.ListVersions(ctx, *row.TemplateID)
	if err != nil {
		return 0, err
	}
	var selected *PlaybookTemplateVersionItem
	for _, item := range versions {
		if item.Version == version {
			copy := item
			selected = &copy
			break
		}
	}
	if selected == nil {
		return 0, ErrWithMessage(ErrNotFound, "Playbook 版本不存在")
	}
	content, err := s.playbookSvc.ResolvePlaybookContent(ctx, selected.Source)
	if err != nil {
		return 0, err
	}
	params := mergeJSONMaps(selected.Defaults, row.Vars, req.Params)
	taskID, err := s.ansibleSvc.RunPlaybookTask(ctx, RunPlaybookTaskRequest{
		ServerIDs: req.ServerIDs,
		Playbook:  content,
		Vars:      params,
		CreatedBy: req.CreatedBy,
		Title:     "自动化任务：" + row.Name,
		Meta: map[string]any{
			"source":            "automation_job",
			"automation_job_id": id,
			"name":              row.Name,
			"env":               row.Env,
			"mode":              row.Mode,
			"risk_level":        row.RiskLevel,
			"template_id":       *row.TemplateID,
		},
	})
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	status := "running"
	if err := s.db.WithContext(ctx).Model(&model.AutomationJob{}).Where("id = ?", id).Updates(map[string]any{
		"last_run_task_id": taskID,
		"last_run_status":  status,
		"last_run_at":      now,
		"updated_by":       req.CreatedBy,
	}).Error; err != nil {
		return 0, err
	}
	return uint64(taskID), nil
}

func buildAutomationJobModel(ctx context.Context, db *gorm.DB, req CreateAutomationJobRequest) (model.AutomationJob, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return model.AutomationJob{}, ErrWithMessage(ErrInvalidParams, "任务名称不能为空")
	}
	mode := normalizeJobMode(req.Mode)
	if mode == "" {
		return model.AutomationJob{}, ErrWithMessage(ErrInvalidParams, "任务方式不合法")
	}
	typ := normalizeJobType(req.Type)
	if typ == "" {
		return model.AutomationJob{}, ErrWithMessage(ErrInvalidParams, "任务类型不合法")
	}
	env := normalizeJobEnv(req.Env)
	if env == "" {
		return model.AutomationJob{}, ErrWithMessage(ErrInvalidParams, "目标环境不合法")
	}
	status := normalizeJobStatus(req.Status)
	if status == "" {
		return model.AutomationJob{}, ErrWithMessage(ErrInvalidParams, "状态不合法")
	}
	risk := normalizeRiskLevel(req.RiskLevel)
	if risk == "" {
		return model.AutomationJob{}, ErrWithMessage(ErrInvalidParams, "风险等级不合法")
	}
	approval := normalizeApprovalMode(req.ApprovalMode)
	if approval == "" {
		return model.AutomationJob{}, ErrWithMessage(ErrInvalidParams, "审批方式不合法")
	}
	strategy := normalizeStrategy(req.Strategy)
	if strategy == "" {
		return model.AutomationJob{}, ErrWithMessage(ErrInvalidParams, "执行策略不合法")
	}
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	timeoutSec := req.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 1800
	}
	var templateID *uint64
	if req.TemplateID != nil && *req.TemplateID > 0 {
		if err := ensurePlaybookTemplateExists(ctx, db, *req.TemplateID); err != nil {
			return model.AutomationJob{}, err
		}
		templateID = req.TemplateID
	}
	row := model.AutomationJob{
		Name:         name,
		Mode:         mode,
		JobType:      typ,
		Env:          env,
		Status:       status,
		RiskLevel:    risk,
		ApprovalMode: approval,
		Strategy:     strategy,
		Concurrency:  concurrency,
		TimeoutSec:   timeoutSec,
		TemplateID:   templateID,
		Cron:         trimStringPtr(req.Cron),
		Targets:      trimStringPtr(req.Targets),
		LimitSpec:    trimStringPtr(req.LimitSpec),
		Vars:         req.Vars,
		ChangeWindow: trimStringPtr(req.ChangeWindow),
		RollbackPlan: trimStringPtr(req.RollbackPlan),
		CreatedBy:    req.CreatedBy,
		UpdatedBy:    req.CreatedBy,
	}
	if err := validateAutomationJobRow(row); err != nil {
		return model.AutomationJob{}, err
	}
	return row, nil
}

func validateAutomationJobRow(row model.AutomationJob) error {
	if row.Mode == "schedule" && (row.Cron == nil || strings.TrimSpace(*row.Cron) == "") {
		return ErrWithMessage(ErrInvalidParams, "定时任务必须填写 Cron")
	}
	if strings.TrimSpace(anyString(row.Targets)) == "" {
		return ErrWithMessage(ErrInvalidParams, "请填写执行范围")
	}
	if row.ApprovalMode == "manual" && strings.TrimSpace(anyString(row.ChangeWindow)) == "" {
		return ErrWithMessage(ErrInvalidParams, "人工审批任务需要填写变更窗口")
	}
	if row.RiskLevel != "low" && strings.TrimSpace(anyString(row.RollbackPlan)) == "" {
		return ErrWithMessage(ErrInvalidParams, "中高风险任务必须提供回滚方案")
	}
	return nil
}

func ensurePlaybookTemplateExists(ctx context.Context, db *gorm.DB, id uint64) error {
	if id == 0 {
		return ErrInvalidParams
	}
	var count int64
	if err := db.WithContext(ctx).Model(&model.PlaybookTemplate{}).Where("deleted_at IS NULL AND id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrWithMessage(ErrNotFound, "关联的 Playbook 模板不存在")
	}
	return nil
}

func trimStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}

func anyString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func normalizeJobMode(v string) string {
	switch strings.TrimSpace(v) {
	case "manual", "schedule":
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func normalizeJobType(v string) string {
	switch strings.TrimSpace(v) {
	case "inspection", "env_install", "hardening", "k8s_deploy", "host_deploy":
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func normalizeJobEnv(v string) string {
	switch strings.TrimSpace(v) {
	case "dev", "test", "staging", "prod":
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func normalizeJobStatus(v string) string {
	switch strings.TrimSpace(v) {
	case "enabled", "disabled":
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func normalizeRiskLevel(v string) string {
	switch strings.TrimSpace(v) {
	case "low", "medium", "high":
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func normalizeApprovalMode(v string) string {
	switch strings.TrimSpace(v) {
	case "auto", "manual":
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func normalizeStrategy(v string) string {
	switch strings.TrimSpace(v) {
	case "parallel", "serial", "batch", "canary":
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func mergeJSONMaps(maps ...model.JSONMap) map[string]any {
	out := map[string]any{}
	for _, item := range maps {
		for k, v := range item {
			out[k] = v
		}
	}
	return out
}

func normalizeAutomationJobIDs(ids []uint64) []uint64 {
	out := make([]uint64, 0, len(ids))
	seen := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func batchRunResultMessage(err error) string {
	if msg, ok := UserMessage(err); ok {
		return msg
	}
	return "执行失败"
}

func toAutomationJobItem(row automationJobRow) AutomationJobItem {
	createdAt := row.CreatedAt.UTC().Format(time.RFC3339)
	updatedAt := row.UpdatedAt.UTC().Format(time.RFC3339)
	var lastRunAt *string
	if row.LastRunAt != nil {
		v := row.LastRunAt.UTC().Format(time.RFC3339)
		lastRunAt = &v
	}
	return AutomationJobItem{
		ID:            row.ID,
		Name:          row.Name,
		Mode:          row.Mode,
		Type:          row.JobType,
		Env:           row.Env,
		Status:        row.Status,
		RiskLevel:     row.RiskLevel,
		ApprovalMode:  row.ApprovalMode,
		Strategy:      row.Strategy,
		Concurrency:   row.Concurrency,
		TimeoutSec:    row.TimeoutSec,
		TemplateID:    row.TemplateID,
		TemplateName:  row.TemplateName,
		Cron:          row.Cron,
		Targets:       row.Targets,
		LimitSpec:     row.LimitSpec,
		Vars:          row.Vars,
		ChangeWindow:  row.ChangeWindow,
		RollbackPlan:  row.RollbackPlan,
		LastRunTaskID: row.LastRunTaskID,
		LastRunStatus: row.LastRunStatus,
		LastRunAt:     lastRunAt,
		CreatedBy:     row.CreatedBy,
		UpdatedBy:     row.UpdatedBy,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}
