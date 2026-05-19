package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

const (
	composeTaskActionDeploy   = "deploy"
	composeTaskActionRollback = "rollback"
	composeTaskActionStart    = "start"
	composeTaskActionStop     = "stop"
)

type composeRevisionSnapshotRequest struct {
	Revision           int
	TemplateID         uint64
	TemplateName       string
	TemplateVersion    string
	ComposeManifest    string
	EnvContent         string
	EnvOverride        string
	Values             map[string]interface{}
	ProjectName        string
	InstallDir         string
	PullImages         bool
	AutoInstallDocker  bool
	AutoInstallCompose bool
	Operator           string
	CreatedBy          uint64
}

// ── 请求 / 响应 DTO ─────────────────────────────────────────────────────────

type ListAppReleasesRequest struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	Keyword    string `form:"keyword"`
	Namespace  string `form:"namespace"`
	ClusterID  uint64 `form:"cluster_id"`
	TargetID   uint64 `form:"target_id"`
	Status     string `form:"status"`
	TemplateID uint64 `form:"template_id"`
	SortBy     string `form:"sort_by"`
	Order      string `form:"order"`
}

type AppReleaseItem struct {
	ID              uint64    `json:"id"`
	Name            string    `json:"name"`
	TemplateEngine  string    `json:"template_engine"`
	LastTaskID      *uint64   `json:"last_task_id,omitempty"`
	ClusterID       uint64    `json:"cluster_id"`
	ClusterName     string    `json:"cluster_name"`
	Namespace       string    `json:"namespace"`
	TargetType      string    `json:"target_type"`
	TargetID        uint64    `json:"target_id"`
	TargetName      string    `json:"target_name"`
	ProjectName     string    `json:"project_name"`
	InstallDir      string    `json:"install_dir"`
	TemplateID      uint64    `json:"template_id"`
	TemplateName    string    `json:"template_name"`
	TemplateVersion string    `json:"template_version"`
	Status          string    `json:"status"`
	CurrentRevision int       `json:"current_revision"`
	Strategy        string    `json:"strategy"`
	Replicas        string    `json:"replicas"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type AppReleaseDetail struct {
	AppReleaseItem
	Values             map[string]interface{} `json:"values"`
	DesiredRevision    int                    `json:"desired_revision"`
	LastEvent          string                 `json:"last_event"`
	HealthScore        int                    `json:"health_score"`
	Operator           string                 `json:"operator"`
	EnvOverride        string                 `json:"env_override"`
	PullImages         bool                   `json:"pull_images"`
	AutoInstallDocker  bool                   `json:"auto_install_docker"`
	AutoInstallCompose bool                   `json:"auto_install_compose"`
	CreatedAt          time.Time              `json:"created_at"`
	Paused             bool                   `json:"paused"`
}

type AppReleaseRevisionItem struct {
	Revision           int                    `json:"revision"`
	TemplateID         uint64                 `json:"template_id"`
	TemplateName       string                 `json:"template_name"`
	TemplateVersion    string                 `json:"template_version"`
	ProjectName        string                 `json:"project_name"`
	InstallDir         string                 `json:"install_dir"`
	EnvOverride        string                 `json:"env_override"`
	Values             map[string]interface{} `json:"values"`
	PullImages         bool                   `json:"pull_images"`
	AutoInstallDocker  bool                   `json:"auto_install_docker"`
	AutoInstallCompose bool                   `json:"auto_install_compose"`
	Operator           string                 `json:"operator"`
	CreatedBy          uint64                 `json:"created_by"`
	CreatedAt          time.Time              `json:"created_at"`
	IsCurrent          bool                   `json:"is_current"`
	IsDesired          bool                   `json:"is_desired"`
}

type CreateAppReleaseRequest struct {
	Name               string                 `json:"name"`
	ClusterID          uint64                 `json:"cluster_id"`
	ClusterName        string                 `json:"cluster_name"`
	Namespace          string                 `json:"namespace"`
	TargetType         string                 `json:"target_type"`
	TargetID           uint64                 `json:"target_id"`
	TargetName         string                 `json:"target_name"`
	ProjectName        string                 `json:"project_name"`
	InstallDir         string                 `json:"install_dir"`
	EnvOverride        string                 `json:"env_override"`
	PullImages         bool                   `json:"pull_images"`
	AutoInstallDocker  bool                   `json:"auto_install_docker"`
	AutoInstallCompose bool                   `json:"auto_install_compose"`
	TemplateID         uint64                 `json:"template_id"`
	Strategy           string                 `json:"strategy"`
	Values             map[string]interface{} `json:"values"`
	CreatedBy          uint64
	Operator           string
}

type UpgradeAppReleaseRequest struct {
	Values      map[string]interface{} `json:"values"`
	EnvOverride string                 `json:"env_override"`
	PullImages  *bool                  `json:"pull_images"`
	UpdatedBy   uint64
	Operator    string
}

// ── Service ──────────────────────────────────────────────────────────────────

type AppReleaseService struct {
	db          *gorm.DB
	tpl         *AppTemplateService
	ansibleSvc  *AnsibleService
	playbookSvc *PlaybookTemplateService
	clusterReg  *ClusterRegistryService
	taskStore   *TaskStore
}

func NewAppReleaseService(db *gorm.DB, tpl *AppTemplateService, ansibleSvc *AnsibleService, playbookSvc *PlaybookTemplateService, clusterReg *ClusterRegistryService, taskStore *TaskStore) *AppReleaseService {
	return &AppReleaseService{db: db, tpl: tpl, ansibleSvc: ansibleSvc, playbookSvc: playbookSvc, clusterReg: clusterReg, taskStore: taskStore}
}

// List 分页查询发布实例列表。
func (s *AppReleaseService) List(ctx context.Context, req ListAppReleasesRequest) (PageResult[AppReleaseItem], error) {
	if s.db == nil {
		return PageResult[AppReleaseItem]{}, errors.New("db is required")
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)

	q := s.db.WithContext(ctx).Model(&model.AppRelease{}).Where("deleted_at IS NULL")

	if req.ClusterID > 0 {
		q = q.Where("cluster_id = ?", req.ClusterID)
	}
	if req.TargetID > 0 {
		q = q.Where("target_id = ?", req.TargetID)
	}
	if req.TemplateID > 0 {
		q = q.Where("template_id = ?", req.TemplateID)
	}
	if ns := strings.TrimSpace(req.Namespace); ns != "" {
		q = q.Where("namespace LIKE ?", "%"+ns+"%")
	}
	if st := strings.TrimSpace(req.Status); st != "" {
		q = q.Where("status LIKE ?", "%"+st+"%")
	}
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("(name LIKE ? OR cluster_name LIKE ? OR target_name LIKE ? OR project_name LIKE ? OR template_name LIKE ? OR namespace LIKE ?)", like, like, like, like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[AppReleaseItem]{}, err
	}

	orderExpr := "id DESC"
	if col := sanitizeReleaseOrderColumn(req.SortBy); col != "" {
		dir := "DESC"
		if strings.ToLower(req.Order) == "asc" {
			dir = "ASC"
		}
		orderExpr = fmt.Sprintf("%s %s", col, dir)
	}

	var rows []model.AppRelease
	if err := q.Order(orderExpr).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[AppReleaseItem]{}, err
	}
	rows = applyReleaseTaskStates(ctx, s.db, rows)

	out := make([]AppReleaseItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAppReleaseItem(r))
	}
	return PageResult[AppReleaseItem]{List: out, Total: int(total), Page: page, PageSize: pageSize}, nil
}

// Get 查询单个发布实例详情。
func (s *AppReleaseService) Get(ctx context.Context, id uint64) (*AppReleaseDetail, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	var m model.AppRelease
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	m = s.applyReleaseTaskState(ctx, m)
	return toAppReleaseDetail(m), nil
}

// ListRevisions 查询发布版本历史，当前主要用于 Compose 发布回滚选择。
func (s *AppReleaseService) ListRevisions(ctx context.Context, id uint64) ([]AppReleaseRevisionItem, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	var release model.AppRelease
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&release).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if release.TemplateEngine != "yaml" {
		return []AppReleaseRevisionItem{}, nil
	}

	var rows []model.AppReleaseRevision
	if err := s.db.WithContext(ctx).
		Where("release_id = ?", id).
		Order("revision DESC, id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]AppReleaseRevisionItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, AppReleaseRevisionItem{
			Revision:           row.Revision,
			TemplateID:         row.TemplateID,
			TemplateName:       row.TemplateName,
			TemplateVersion:    row.TemplateVersion,
			ProjectName:        row.ProjectName,
			InstallDir:         row.InstallDir,
			EnvOverride:        strPtrVal(row.EnvOverride),
			Values:             jsonToMap(row.Values),
			PullImages:         row.PullImages,
			AutoInstallDocker:  row.AutoInstallDocker,
			AutoInstallCompose: row.AutoInstallCompose,
			Operator:           row.Operator,
			CreatedBy:          row.CreatedBy,
			CreatedAt:          row.CreatedAt,
			IsCurrent:          row.Revision == release.CurrentRevision,
			IsDesired:          row.Revision == release.DesiredRevision,
		})
	}
	return out, nil
}

// Create 新建发布实例。
func (s *AppReleaseService) Create(ctx context.Context, req CreateAppReleaseRequest) (*AppReleaseDetail, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrWithMessage(ErrInvalidParams, "发布名称不能为空")
	}
	if req.ClusterID == 0 {
		req.ClusterID = 0
	}
	if req.TemplateID == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "请选择应用模板")
	}

	// 获取模板名称、版本和引擎
	var tplName, tplVersion, tplEngine string
	var tpl model.AppTemplate
	if err := s.db.WithContext(ctx).Select("id, name, version, engine").
		Where("id = ? AND deleted_at IS NULL", req.TemplateID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWithMessage(ErrNotFound, "应用模板不存在")
		}
		return nil, err
	}
	tplName = tpl.Name
	tplVersion = tpl.Version
	tplEngine = tpl.Engine
	if tplEngine == "yaml" && strings.TrimSpace(req.TargetType) == "" {
		req.TargetType = "server"
	}
	if tplEngine == "yaml" {
		if req.TargetType != "server" {
			return nil, ErrWithMessage(ErrInvalidParams, "Compose 发布仅支持服务器目标")
		}
		if req.TargetID == 0 {
			return nil, ErrWithMessage(ErrInvalidParams, "请选择目标服务器")
		}
		if err := s.ensureComposeTarget(ctx, req.TargetID, &req.TargetName); err != nil {
			return nil, err
		}
		if strings.TrimSpace(req.ProjectName) == "" {
			return nil, ErrWithMessage(ErrInvalidParams, "请填写项目名称")
		}
		if strings.TrimSpace(req.InstallDir) == "" {
			return nil, ErrWithMessage(ErrInvalidParams, "请填写部署目录")
		}
	} else {
		if req.ClusterID == 0 {
			return nil, ErrWithMessage(ErrInvalidParams, "请选择集群")
		}
		if err := s.ensureHelmTarget(ctx, req.ClusterID, &req.ClusterName); err != nil {
			return nil, err
		}
		if strings.TrimSpace(req.Namespace) == "" {
			req.Namespace = "default"
		}
	}

	strategy := req.Strategy
	if strategy == "" {
		strategy = "rolling"
	}
	valuesJSON := marshalJSONMap(req.Values)
	operator := req.Operator
	if operator == "" {
		operator = "system"
	}

	var composeTaskID *uint64
	var composeEvent string

	m := model.AppRelease{
		Name:               req.Name,
		TemplateEngine:     tplEngine,
		ClusterID:          req.ClusterID,
		ClusterName:        req.ClusterName,
		Namespace:          req.Namespace,
		TargetType:         req.TargetType,
		TargetID:           req.TargetID,
		TargetName:         req.TargetName,
		ProjectName:        req.ProjectName,
		InstallDir:         req.InstallDir,
		EnvOverride:        strPtr(req.EnvOverride),
		PullImages:         req.PullImages,
		AutoInstallDocker:  req.AutoInstallDocker,
		AutoInstallCompose: req.AutoInstallCompose,
		TemplateID:         req.TemplateID,
		TemplateName:       tplName,
		TemplateVersion:    tplVersion,
		Status:             initialReleaseStatus(tplEngine),
		CurrentRevision:    1,
		DesiredRevision:    1,
		Strategy:           strategy,
		Replicas:           initialReleaseReplicas(tplEngine),
		HealthScore:        70,
		Values:             strPtr(string(valuesJSON)),
		LastEvent:          initialReleaseEvent(tplEngine),
		Operator:           operator,
		Paused:             false,
		CreatedBy:          req.CreatedBy,
		UpdatedBy:          req.CreatedBy,
	}
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	if tplEngine == "yaml" {
		snapshotReq, err := s.buildComposeSnapshotRequest(ctx, m, composeRevisionSnapshotRequest{
			Revision:           m.DesiredRevision,
			TemplateID:         m.TemplateID,
			TemplateName:       m.TemplateName,
			TemplateVersion:    m.TemplateVersion,
			Values:             req.Values,
			EnvOverride:        req.EnvOverride,
			ProjectName:        req.ProjectName,
			InstallDir:         req.InstallDir,
			PullImages:         req.PullImages,
			AutoInstallDocker:  req.AutoInstallDocker,
			AutoInstallCompose: req.AutoInstallCompose,
			Operator:           operator,
			CreatedBy:          req.CreatedBy,
		})
		if err != nil {
			_ = s.db.WithContext(ctx).Unscoped().Delete(&model.AppRelease{}, m.ID).Error
			return nil, err
		}
		snapshot, err := s.createComposeRevisionSnapshot(ctx, m.ID, snapshotReq)
		if err != nil {
			_ = s.db.WithContext(ctx).Unscoped().Delete(&model.AppRelease{}, m.ID).Error
			return nil, err
		}
		taskID, err := s.startComposeTaskFromRevision(ctx, m, *snapshot, composeTaskActionDeploy, req.CreatedBy)
		if err != nil {
			_ = s.db.WithContext(ctx).Where("release_id = ?", m.ID).Delete(&model.AppReleaseRevision{}).Error
			_ = s.db.WithContext(ctx).Unscoped().Delete(&model.AppRelease{}, m.ID).Error
			return nil, err
		}
		composeTaskID = &taskID
		composeEvent = fmt.Sprintf("已提交 Compose 部署任务 #%d，等待服务器执行", taskID)
		if err := s.db.WithContext(ctx).Model(&m).Updates(map[string]any{
			"last_task_id": taskID,
			"last_event":   composeEvent,
		}).Error; err != nil {
			return nil, err
		}
		m.LastTaskID = composeTaskID
		m.LastEvent = composeEvent
	} else {
		snapshotReq, err := s.buildHelmSnapshotRequest(ctx, m, m.DesiredRevision, req.Values, operator, req.CreatedBy)
		if err != nil {
			_ = s.db.WithContext(ctx).Unscoped().Delete(&model.AppRelease{}, m.ID).Error
			return nil, err
		}
		snapshot, err := s.createComposeRevisionSnapshot(ctx, m.ID, snapshotReq)
		if err != nil {
			_ = s.db.WithContext(ctx).Unscoped().Delete(&model.AppRelease{}, m.ID).Error
			return nil, err
		}
		taskID, err := s.startHelmTask(m, *snapshot, nil, composeTaskActionDeploy, req.CreatedBy)
		if err != nil {
			_ = s.db.WithContext(ctx).Where("release_id = ?", m.ID).Delete(&model.AppReleaseRevision{}).Error
			_ = s.db.WithContext(ctx).Unscoped().Delete(&model.AppRelease{}, m.ID).Error
			return nil, err
		}
		helmTaskID := uint64(taskID)
		helmEvent := fmt.Sprintf("已提交 Helm 发布任务 #%d，等待集群执行", taskID)
		if err := s.db.WithContext(ctx).Model(&m).Updates(map[string]any{
			"last_task_id": helmTaskID,
			"last_event":   helmEvent,
			"values":       stringPointerValue(snapshot.Values),
		}).Error; err != nil {
			return nil, err
		}
		m.LastTaskID = &helmTaskID
		m.LastEvent = helmEvent
	}
	m = s.applyReleaseTaskState(ctx, m)
	return toAppReleaseDetail(m), nil
}

// Upgrade 升级发布实例（更新 values，revision+1，状态重置为 progressing）。
func (s *AppReleaseService) Upgrade(ctx context.Context, id uint64, req UpgradeAppReleaseRequest) (*AppReleaseDetail, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	var m model.AppRelease
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if m.TemplateEngine == "yaml" {
		if s.ansibleSvc == nil || s.playbookSvc == nil {
			return nil, ErrWithMessage(ErrConflict, "Compose 发布运行时未就绪，请检查 Playbook 与任务执行服务")
		}
	} else if s.taskStore == nil || s.clusterReg == nil {
		return nil, ErrWithMessage(ErrConflict, "Helm 发布运行时未就绪，请检查任务中心与集群配置")
	}
	operator := req.Operator
	if operator == "" {
		operator = "system"
	}
	status := "progressing"
	lastEvent := "已提交升级任务，等待执行"
	lastTaskID := m.LastTaskID
	effectiveValues := req.Values
	if m.TemplateEngine == "yaml" {
		effectiveValues = mergeReleaseValues(jsonToMap(m.Values), req.Values)
		effectiveEnvOverride := composeFirstNonEmpty(strings.TrimSpace(req.EnvOverride), strPtrVal(m.EnvOverride))
		effectivePullImages := resolveBoolValue(req.PullImages, m.PullImages)
		revision := m.DesiredRevision + 1
		snapshotReq, err := s.buildComposeSnapshotRequest(ctx, m, composeRevisionSnapshotRequest{
			Revision:           revision,
			TemplateID:         m.TemplateID,
			TemplateName:       m.TemplateName,
			TemplateVersion:    m.TemplateVersion,
			Values:             effectiveValues,
			EnvOverride:        effectiveEnvOverride,
			ProjectName:        m.ProjectName,
			InstallDir:         m.InstallDir,
			PullImages:         effectivePullImages,
			AutoInstallDocker:  m.AutoInstallDocker,
			AutoInstallCompose: m.AutoInstallCompose,
			Operator:           operator,
			CreatedBy:          req.UpdatedBy,
		})
		if err != nil {
			return nil, err
		}
		snapshot, err := s.createComposeRevisionSnapshot(ctx, m.ID, snapshotReq)
		if err != nil {
			return nil, err
		}
		taskID, err := s.startComposeTaskFromRevision(ctx, m, *snapshot, composeTaskActionDeploy, req.UpdatedBy)
		if err != nil {
			return nil, err
		}
		lastTaskID = &taskID
		status = "installing"
		lastEvent = fmt.Sprintf("已提交 Compose 更新任务 #%d，等待服务器执行", taskID)
		updates := map[string]interface{}{
			"desired_revision":     revision,
			"status":               status,
			"values":               string(marshalJSONMap(effectiveValues)),
			"env_override":         effectiveEnvOverride,
			"pull_images":          effectivePullImages,
			"operator":             operator,
			"last_event":           lastEvent,
			"updated_by":           req.UpdatedBy,
			"last_task_id":         lastTaskID,
			"auto_install_docker":  m.AutoInstallDocker,
			"auto_install_compose": m.AutoInstallCompose,
		}
		if err := s.db.WithContext(ctx).Model(&m).Updates(updates).Error; err != nil {
			return nil, err
		}
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
			return nil, err
		}
		m = s.applyComposeTaskState(ctx, m)
		return toAppReleaseDetail(m), nil
	}
	mergedValues := mergeReleaseValues(jsonToMap(m.Values), req.Values)
	revision := m.DesiredRevision + 1
	snapshotReq, err := s.buildHelmSnapshotRequest(ctx, m, revision, mergedValues, operator, req.UpdatedBy)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.createComposeRevisionSnapshot(ctx, m.ID, snapshotReq)
	if err != nil {
		return nil, err
	}
	previousSnapshot, err := s.getComposeRevisionSnapshot(ctx, m.ID, m.CurrentRevision)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	taskID, err := s.startHelmTask(m, *snapshot, previousSnapshot, composeTaskActionDeploy, req.UpdatedBy)
	if err != nil {
		return nil, err
	}
	helmTaskID := uint64(taskID)
	status = "progressing"
	lastEvent = fmt.Sprintf("已提交 Helm 升级任务 #%d，等待集群执行", taskID)
	updates := map[string]interface{}{
		"desired_revision": revision,
		"status":           status,
		"values":           string(marshalJSONMap(mergedValues)),
		"operator":         operator,
		"last_event":       lastEvent,
		"updated_by":       req.UpdatedBy,
		"last_task_id":     helmTaskID,
		"template_id":      snapshot.TemplateID,
		"template_name":    snapshot.TemplateName,
		"template_version": snapshot.TemplateVersion,
	}
	if strings.TrimSpace(req.EnvOverride) != "" {
		updates["env_override"] = req.EnvOverride
	}
	if req.PullImages != nil {
		updates["pull_images"] = *req.PullImages
	}
	if err := s.db.WithContext(ctx).Model(&m).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		return nil, err
	}
	m = s.applyReleaseTaskState(ctx, m)
	return toAppReleaseDetail(m), nil
}

// Rollback 回滚发布实例；targetRevision=0 时默认回滚到上一版本。
func (s *AppReleaseService) Rollback(ctx context.Context, id uint64, targetRevision int, updatedBy uint64, operator string) (*AppReleaseDetail, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	var m model.AppRelease
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if m.CurrentRevision <= 1 && targetRevision == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "已是最早版本，无法继续回滚")
	}
	if targetRevision == 0 {
		targetRevision = m.CurrentRevision - 1
	}
	if targetRevision < 1 {
		return nil, ErrWithMessage(ErrInvalidParams, "回滚版本不能小于 1")
	}
	if targetRevision >= m.CurrentRevision {
		return nil, ErrWithMessage(ErrInvalidParams, "请选择当前版本之前的历史版本进行回滚")
	}
	if operator == "" {
		operator = "system"
	}
	if m.TemplateEngine == "yaml" {
		snapshot, err := s.getComposeRevisionSnapshot(ctx, m.ID, targetRevision)
		if err != nil {
			return nil, err
		}
		taskID, err := s.startComposeTaskFromRevision(ctx, m, *snapshot, composeTaskActionRollback, updatedBy)
		if err != nil {
			return nil, err
		}
		updates := map[string]interface{}{
			"desired_revision":     targetRevision,
			"status":               "installing",
			"values":               stringPointerValue(snapshot.Values),
			"env_override":         stringPointerValue(snapshot.EnvOverride),
			"project_name":         snapshot.ProjectName,
			"install_dir":          snapshot.InstallDir,
			"pull_images":          snapshot.PullImages,
			"auto_install_docker":  snapshot.AutoInstallDocker,
			"auto_install_compose": snapshot.AutoInstallCompose,
			"template_id":          snapshot.TemplateID,
			"template_name":        snapshot.TemplateName,
			"template_version":     snapshot.TemplateVersion,
			"operator":             operator,
			"last_event":           fmt.Sprintf("已提交 Compose 回滚任务 #%d，等待服务器执行", taskID),
			"updated_by":           updatedBy,
			"last_task_id":         taskID,
			"paused":               false,
		}
		if err := s.db.WithContext(ctx).Model(&m).Updates(updates).Error; err != nil {
			return nil, err
		}
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
			return nil, err
		}
		m = s.applyComposeTaskState(ctx, m)
		return toAppReleaseDetail(m), nil
	}
	targetSnapshot, err := s.getComposeRevisionSnapshot(ctx, m.ID, targetRevision)
	if err != nil {
		return nil, err
	}
	currentSnapshot, err := s.getComposeRevisionSnapshot(ctx, m.ID, m.CurrentRevision)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	taskID, err := s.startHelmTask(m, *targetSnapshot, currentSnapshot, composeTaskActionRollback, updatedBy)
	if err != nil {
		return nil, err
	}
	updates := map[string]interface{}{
		"desired_revision": targetRevision,
		"status":           "progressing",
		"operator":         operator,
		"last_event":       fmt.Sprintf("已提交 Helm 回滚任务 #%d，目标版本 r%d，等待集群执行", taskID, targetRevision),
		"updated_by":       updatedBy,
		"last_task_id":     taskID,
	}
	if err := s.db.WithContext(ctx).Model(&m).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		return nil, err
	}
	m = s.applyReleaseTaskState(ctx, m)
	return toAppReleaseDetail(m), nil
}

func (s *AppReleaseService) applyReleaseTaskState(ctx context.Context, row model.AppRelease) model.AppRelease {
	if row.LastTaskID == nil || *row.LastTaskID == 0 || s.db == nil {
		return row
	}
	var task model.Task
	if err := s.db.WithContext(ctx).First(&task, *row.LastTaskID).Error; err != nil {
		return row
	}
	return applyReleaseTaskState(row, &task)
}

// Start 启动 Compose 发布。
func (s *AppReleaseService) Start(ctx context.Context, id uint64, updatedBy uint64, operator string) (*AppReleaseDetail, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	var m model.AppRelease
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if m.TemplateEngine != "yaml" {
		return nil, ErrWithMessage(ErrInvalidParams, "当前发布不支持启动操作")
	}
	if operator == "" {
		operator = "system"
	}
	taskID, err := s.startComposeLifecycleTask(ctx, m, composeTaskActionStart, updatedBy)
	if err != nil {
		return nil, err
	}
	updates := map[string]interface{}{
		"status":       "installing",
		"operator":     operator,
		"last_event":   fmt.Sprintf("已提交 Compose 启动任务 #%d，等待服务器执行", taskID),
		"updated_by":   updatedBy,
		"last_task_id": taskID,
		"paused":       false,
	}
	if err := s.db.WithContext(ctx).Model(&m).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		return nil, err
	}
	m = s.applyComposeTaskState(ctx, m)
	return toAppReleaseDetail(m), nil
}

// Stop 停止 Compose 发布。
func (s *AppReleaseService) Stop(ctx context.Context, id uint64, updatedBy uint64, operator string) (*AppReleaseDetail, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	var m model.AppRelease
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if m.TemplateEngine != "yaml" {
		return nil, ErrWithMessage(ErrInvalidParams, "当前发布不支持停止操作")
	}
	if operator == "" {
		operator = "system"
	}
	taskID, err := s.startComposeLifecycleTask(ctx, m, composeTaskActionStop, updatedBy)
	if err != nil {
		return nil, err
	}
	updates := map[string]interface{}{
		"status":       "progressing",
		"operator":     operator,
		"last_event":   fmt.Sprintf("已提交 Compose 停止任务 #%d，等待服务器执行", taskID),
		"updated_by":   updatedBy,
		"last_task_id": taskID,
		"paused":       false,
	}
	if err := s.db.WithContext(ctx).Model(&m).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		return nil, err
	}
	m = s.applyComposeTaskState(ctx, m)
	return toAppReleaseDetail(m), nil
}

// TogglePause 切换暂停状态。
func (s *AppReleaseService) TogglePause(ctx context.Context, id uint64, updatedBy uint64) (*AppReleaseDetail, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	var m model.AppRelease
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if m.TemplateEngine == "yaml" {
		if m.Status == "stopped" {
			return s.Start(ctx, id, updatedBy, "system")
		}
		return s.Stop(ctx, id, updatedBy, "system")
	}
	newPaused := !m.Paused
	event := "已恢复发布"
	if newPaused {
		event = "已暂停发布，等待变更窗口"
	}
	updates := map[string]interface{}{
		"paused":     newPaused,
		"last_event": event,
		"updated_by": updatedBy,
	}
	if err := s.db.WithContext(ctx).Model(&m).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		return nil, err
	}
	return toAppReleaseDetail(m), nil
}

// Delete 软删除发布实例。
func (s *AppReleaseService) Delete(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	now := time.Now()
	res := s.db.WithContext(ctx).Model(&model.AppRelease{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ── 私有辅助 ──────────────────────────────────────────────────────────────

func toAppReleaseItem(m model.AppRelease) AppReleaseItem {
	return AppReleaseItem{
		ID:              m.ID,
		Name:            m.Name,
		TemplateEngine:  m.TemplateEngine,
		LastTaskID:      m.LastTaskID,
		ClusterID:       m.ClusterID,
		ClusterName:     m.ClusterName,
		Namespace:       m.Namespace,
		TargetType:      m.TargetType,
		TargetID:        m.TargetID,
		TargetName:      m.TargetName,
		ProjectName:     m.ProjectName,
		InstallDir:      m.InstallDir,
		TemplateID:      m.TemplateID,
		TemplateName:    m.TemplateName,
		TemplateVersion: m.TemplateVersion,
		Status:          m.Status,
		CurrentRevision: m.CurrentRevision,
		Strategy:        m.Strategy,
		Replicas:        m.Replicas,
		UpdatedAt:       m.UpdatedAt,
	}
}

func toAppReleaseDetail(m model.AppRelease) *AppReleaseDetail {
	return &AppReleaseDetail{
		AppReleaseItem:     toAppReleaseItem(m),
		Values:             jsonToMap(m.Values),
		DesiredRevision:    m.DesiredRevision,
		LastEvent:          m.LastEvent,
		HealthScore:        m.HealthScore,
		Operator:           m.Operator,
		EnvOverride:        strPtrVal(m.EnvOverride),
		PullImages:         m.PullImages,
		AutoInstallDocker:  m.AutoInstallDocker,
		AutoInstallCompose: m.AutoInstallCompose,
		CreatedAt:          m.CreatedAt,
		Paused:             m.Paused,
	}
}

func sanitizeReleaseOrderColumn(col string) string {
	allowed := map[string]bool{
		"id": true, "name": true, "namespace": true, "status": true,
		"current_revision": true, "target_name": true, "created_at": true, "updated_at": true,
	}
	if allowed[col] {
		return col
	}
	return ""
}

func initialReleaseStatus(engine string) string {
	if engine == "yaml" {
		return "installing"
	}
	return "progressing"
}

func initialReleaseReplicas(engine string) string {
	if engine == "yaml" {
		return "1 service"
	}
	return "0/1"
}

func initialReleaseEvent(engine string) string {
	if engine == "yaml" {
		return "创建 Compose 发布任务，等待服务器执行"
	}
	return "创建发布任务，等待调度"
}

func (s *AppReleaseService) ensureComposeTarget(ctx context.Context, targetID uint64, targetName *string) error {
	var server model.Server
	if err := s.db.WithContext(ctx).Select("id, name, status").Where("id = ? AND deleted_at IS NULL", targetID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWithMessage(ErrNotFound, "目标服务器不存在")
		}
		return err
	}
	if strings.TrimSpace(server.Status) != "active" {
		return ErrWithMessage(ErrConflict, "目标服务器当前不可用，请先检查服务器状态")
	}
	if targetName != nil && strings.TrimSpace(*targetName) == "" {
		*targetName = server.Name
	}
	return nil
}

func (s *AppReleaseService) buildComposeSnapshotRequest(ctx context.Context, release model.AppRelease, req composeRevisionSnapshotRequest) (composeRevisionSnapshotRequest, error) {
	detail, err := s.tpl.Get(ctx, release.TemplateID)
	if err != nil {
		return composeRevisionSnapshotRequest{}, err
	}
	manifest := strings.TrimSpace(detail.Manifest)
	if manifest == "" {
		return composeRevisionSnapshotRequest{}, ErrWithMessage(ErrInvalidParams, "Compose 模板内容为空，无法提交部署")
	}
	req.ComposeManifest = manifest
	req.ProjectName = composeFirstNonEmpty(strings.TrimSpace(req.ProjectName), release.ProjectName, detail.ProjectNameDefault, detail.Name)
	req.InstallDir = composeFirstNonEmpty(strings.TrimSpace(req.InstallDir), release.InstallDir, detail.InstallDirDefault)
	req.EnvOverride = strings.TrimSpace(req.EnvOverride)
	req.EnvContent = mergeEnvContent(detail.EnvExample, req.EnvOverride)
	if req.Values == nil {
		req.Values = map[string]interface{}{}
	}
	return req, nil
}

func (s *AppReleaseService) createComposeRevisionSnapshot(ctx context.Context, releaseID uint64, req composeRevisionSnapshotRequest) (*model.AppReleaseRevision, error) {
	if releaseID == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "发布实例不存在，无法保存回滚快照")
	}
	record := &model.AppReleaseRevision{
		ReleaseID:          releaseID,
		Revision:           req.Revision,
		TemplateID:         req.TemplateID,
		TemplateName:       req.TemplateName,
		TemplateVersion:    req.TemplateVersion,
		ComposeManifest:    req.ComposeManifest,
		EnvContent:         nullableStringPtr(req.EnvContent),
		EnvOverride:        nullableStringPtr(req.EnvOverride),
		Values:             nullableJSONString(req.Values),
		ProjectName:        req.ProjectName,
		InstallDir:         req.InstallDir,
		PullImages:         req.PullImages,
		AutoInstallDocker:  req.AutoInstallDocker,
		AutoInstallCompose: req.AutoInstallCompose,
		Operator:           req.Operator,
		CreatedBy:          req.CreatedBy,
	}
	if err := s.db.WithContext(ctx).
		Where("release_id = ? AND revision = ?", releaseID, req.Revision).
		Assign(record).
		FirstOrCreate(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *AppReleaseService) getComposeRevisionSnapshot(ctx context.Context, releaseID uint64, revision int) (*model.AppReleaseRevision, error) {
	var snapshot model.AppReleaseRevision
	if err := s.db.WithContext(ctx).
		Where("release_id = ? AND revision = ?", releaseID, revision).
		First(&snapshot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWithMessage(ErrNotFound, "目标回滚版本快照不存在，请先补齐发布历史记录")
		}
		return nil, err
	}
	return &snapshot, nil
}

func (s *AppReleaseService) startComposeTaskFromRevision(ctx context.Context, release model.AppRelease, snapshot model.AppReleaseRevision, action string, createdBy uint64) (uint64, error) {
	if s.ansibleSvc == nil || s.playbookSvc == nil {
		return 0, ErrWithMessage(ErrConflict, "Compose 发布运行时未就绪，请检查 Playbook 与任务执行服务")
	}
	playbook, err := s.resolveComposeInstallPlaybook(ctx)
	if err != nil {
		return 0, err
	}
	taskID, err := s.ansibleSvc.RunPlaybookTask(ctx, RunPlaybookTaskRequest{
		ServerIDs: []uint64{release.TargetID},
		Playbook:  playbook,
		Vars: map[string]any{
			"app_name":             composeFirstNonEmpty(release.Name, snapshot.TemplateName),
			"install_dir":          snapshot.InstallDir,
			"project_name":         composeFirstNonEmpty(snapshot.ProjectName, release.ProjectName, release.Name),
			"compose_yaml":         snapshot.ComposeManifest,
			"env_content":          strPtrVal(snapshot.EnvContent),
			"pull_images":          snapshot.PullImages,
			"auto_install_docker":  snapshot.AutoInstallDocker,
			"auto_install_compose": snapshot.AutoInstallCompose,
		},
		CreatedBy: createdBy,
		Title:     composeTaskTitle(action, release.Name),
		Meta: map[string]any{
			"source":           "app_release",
			"action":           action,
			"app_release_id":   release.ID,
			"name":             release.Name,
			"template_id":      snapshot.TemplateID,
			"template_name":    snapshot.TemplateName,
			"template_engine":  release.TemplateEngine,
			"target_type":      "server",
			"target_id":        release.TargetID,
			"target_name":      release.TargetName,
			"project_name":     snapshot.ProjectName,
			"install_dir":      snapshot.InstallDir,
			"desired_revision": snapshot.Revision,
			"previous_status":  release.Status,
		},
	})
	if err != nil {
		return 0, ErrWithMessage(ErrConflict, composeTaskFailedMessage(action))
	}
	return uint64(taskID), nil
}

func (s *AppReleaseService) startComposeLifecycleTask(ctx context.Context, release model.AppRelease, action string, createdBy uint64) (uint64, error) {
	if s.ansibleSvc == nil {
		return 0, ErrWithMessage(ErrConflict, "Compose 发布运行时未就绪，请检查任务执行服务")
	}
	playbook, err := resolveComposeLifecyclePlaybook(action)
	if err != nil {
		return 0, err
	}
	taskID, err := s.ansibleSvc.RunPlaybookTask(ctx, RunPlaybookTaskRequest{
		ServerIDs: []uint64{release.TargetID},
		Playbook:  playbook,
		Vars: map[string]any{
			"install_dir":  release.InstallDir,
			"project_name": composeFirstNonEmpty(release.ProjectName, release.Name),
		},
		CreatedBy: createdBy,
		Title:     composeTaskTitle(action, release.Name),
		Meta: map[string]any{
			"source":          "app_release",
			"action":          action,
			"app_release_id":  release.ID,
			"name":            release.Name,
			"template_id":     release.TemplateID,
			"template_name":   release.TemplateName,
			"template_engine": release.TemplateEngine,
			"target_type":     "server",
			"target_id":       release.TargetID,
			"target_name":     release.TargetName,
			"project_name":    release.ProjectName,
			"install_dir":     release.InstallDir,
			"previous_status": release.Status,
		},
	})
	if err != nil {
		return 0, ErrWithMessage(ErrConflict, composeTaskFailedMessage(action))
	}
	return uint64(taskID), nil
}

func (s *AppReleaseService) resolveComposeInstallPlaybook(ctx context.Context) (string, error) {
	var tpl model.PlaybookTemplate
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND name = ?", "Docker Compose 应用安装").First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrWithMessage(ErrNotFound, "Compose 部署模板不存在，请联系管理员检查内置 Playbook")
		}
		return "", err
	}
	var version model.PlaybookTemplateVersion
	if err := s.db.WithContext(ctx).Where("template_id = ? AND version = ?", tpl.ID, tpl.CurrentVersion).First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrWithMessage(ErrNotFound, "Compose 部署模板版本不存在，请联系管理员检查内置 Playbook")
		}
		return "", err
	}
	return s.playbookSvc.ResolvePlaybookContent(ctx, version.Source)
}

func (s *AppReleaseService) applyComposeTaskStates(ctx context.Context, rows []model.AppRelease) []model.AppRelease {
	if len(rows) == 0 {
		return rows
	}
	taskIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		if row.TemplateEngine == "yaml" && row.LastTaskID != nil && *row.LastTaskID > 0 {
			taskIDs = append(taskIDs, *row.LastTaskID)
		}
	}
	if len(taskIDs) == 0 {
		return rows
	}
	var tasks []model.Task
	if err := s.db.WithContext(ctx).Where("id IN ?", taskIDs).Find(&tasks).Error; err != nil {
		return rows
	}
	taskMap := make(map[uint64]model.Task, len(tasks))
	for _, task := range tasks {
		taskMap[task.ID] = task
	}
	for idx := range rows {
		if rows[idx].LastTaskID == nil {
			continue
		}
		if task, ok := taskMap[*rows[idx].LastTaskID]; ok {
			rows[idx] = applyComposeTaskState(rows[idx], &task)
		}
	}
	return rows
}

func (s *AppReleaseService) applyComposeTaskState(ctx context.Context, row model.AppRelease) model.AppRelease {
	if row.TemplateEngine != "yaml" || row.LastTaskID == nil || *row.LastTaskID == 0 {
		return row
	}
	var task model.Task
	if err := s.db.WithContext(ctx).First(&task, *row.LastTaskID).Error; err != nil {
		return row
	}
	return applyComposeTaskState(row, &task)
}

func applyComposeTaskState(row model.AppRelease, task *model.Task) model.AppRelease {
	if task == nil {
		return row
	}
	action := composeTaskAction(task)
	previousStatus := composeTaskPreviousStatus(task, row.Status)
	switch task.Status {
	case model.TaskPending, model.TaskRunning:
		switch action {
		case composeTaskActionStop:
			row.Status = "progressing"
			row.LastEvent = fmt.Sprintf("Compose 停止任务 #%d 执行中", task.ID)
		case composeTaskActionStart:
			row.Status = "installing"
			row.LastEvent = fmt.Sprintf("Compose 启动任务 #%d 执行中", task.ID)
		case composeTaskActionRollback:
			row.Status = "installing"
			row.LastEvent = fmt.Sprintf("Compose 回滚任务 #%d 执行中", task.ID)
		default:
			row.Status = "installing"
			row.LastEvent = fmt.Sprintf("Compose 部署任务 #%d 执行中", task.ID)
		}
	case model.TaskSuccess:
		switch action {
		case composeTaskActionStop:
			row.Status = "stopped"
			row.LastEvent = fmt.Sprintf("Compose 停止任务 #%d 执行成功", task.ID)
			row.HealthScore = 60
		case composeTaskActionStart:
			row.Status = "running"
			row.LastEvent = fmt.Sprintf("Compose 启动任务 #%d 执行成功", task.ID)
			row.HealthScore = 90
		case composeTaskActionRollback:
			row.Status = "running"
			row.CurrentRevision = row.DesiredRevision
			row.LastEvent = fmt.Sprintf("Compose 回滚任务 #%d 执行成功", task.ID)
			row.HealthScore = 92
		default:
			row.Status = "running"
			if row.DesiredRevision > row.CurrentRevision {
				row.CurrentRevision = row.DesiredRevision
			}
			row.LastEvent = fmt.Sprintf("Compose 部署任务 #%d 执行成功", task.ID)
			row.HealthScore = 95
		}
	case model.TaskCanceled:
		row.Status = previousStatus
		row.LastEvent = fmt.Sprintf("Compose 任务 #%d 已取消", task.ID)
		row.HealthScore = 50
	case model.TaskFailed, model.TaskTimeout:
		row.Status = "failed"
		row.LastEvent = composeFirstNonEmpty(strings.TrimSpace(task.Message), composeTaskFailureEvent(action, task.ID))
		row.HealthScore = 30
	}
	return row
}

func composeTaskAction(task *model.Task) string {
	if task == nil || task.Meta == nil {
		return composeTaskActionDeploy
	}
	action := strings.TrimSpace(fmt.Sprintf("%v", task.Meta["action"]))
	if action == "" || action == "<nil>" {
		return composeTaskActionDeploy
	}
	return action
}

func composeTaskPreviousStatus(task *model.Task, fallback string) string {
	if task == nil || task.Meta == nil {
		return fallback
	}
	value := strings.TrimSpace(fmt.Sprintf("%v", task.Meta["previous_status"]))
	if value == "" || value == "<nil>" {
		return fallback
	}
	return value
}

func composeTaskTitle(action, releaseName string) string {
	name := composeFirstNonEmpty(releaseName, "未命名发布")
	switch action {
	case composeTaskActionRollback:
		return "Compose 应用回滚：" + name
	case composeTaskActionStart:
		return "Compose 应用启动：" + name
	case composeTaskActionStop:
		return "Compose 应用停止：" + name
	default:
		return "Compose 应用发布：" + name
	}
}

func composeTaskFailedMessage(action string) string {
	switch action {
	case composeTaskActionRollback:
		return "提交 Compose 回滚任务失败，请检查服务器连通性、凭据和后端 Ansible 运行环境"
	case composeTaskActionStart:
		return "提交 Compose 启动任务失败，请检查服务器连通性、凭据和后端 Ansible 运行环境"
	case composeTaskActionStop:
		return "提交 Compose 停止任务失败，请检查服务器连通性、凭据和后端 Ansible 运行环境"
	default:
		return "提交 Compose 部署任务失败，请检查服务器连通性、凭据和后端 Ansible 运行环境"
	}
}

func composeTaskFailureEvent(action string, taskID uint64) string {
	switch action {
	case composeTaskActionRollback:
		return fmt.Sprintf("Compose 回滚任务 #%d 执行失败", taskID)
	case composeTaskActionStart:
		return fmt.Sprintf("Compose 启动任务 #%d 执行失败", taskID)
	case composeTaskActionStop:
		return fmt.Sprintf("Compose 停止任务 #%d 执行失败", taskID)
	default:
		return fmt.Sprintf("Compose 部署任务 #%d 执行失败", taskID)
	}
}

func resolveComposeLifecyclePlaybook(action string) (string, error) {
	switch action {
	case composeTaskActionStart:
		return strings.TrimSpace(`---
- name: Start docker compose project
  hosts: targets
  become: true
  gather_facts: false
  vars:
    install_dir: "{{ install_dir | default('/opt/apps/app') }}"
    project_name: "{{ project_name | default('app') }}"
  tasks:
    - name: Detect docker compose command
      shell: |
        if docker compose version >/dev/null 2>&1; then
          echo docker compose
          exit 0
        fi
        if command -v docker-compose >/dev/null 2>&1; then
          echo docker-compose
          exit 0
        fi
        exit 2
      args:
        executable: /bin/sh
      register: compose_cmd
      changed_when: false

    - name: Start compose services
      shell: |
        set -e
        cd {{ install_dir | quote }}
        if [ "{{ compose_cmd.stdout }}" = "docker compose" ]; then
          docker compose -p {{ project_name | quote }} start
        else
          docker-compose -p {{ project_name | quote }} start
        fi
      args:
        executable: /bin/sh

    - name: Inspect compose services
      shell: |
        set -e
        cd {{ install_dir | quote }}
        if [ "{{ compose_cmd.stdout }}" = "docker compose" ]; then
          docker compose -p {{ project_name | quote }} ps
        else
          docker-compose -p {{ project_name | quote }} ps
        fi
      args:
        executable: /bin/sh
      changed_when: false`), nil
	case composeTaskActionStop:
		return strings.TrimSpace(`---
- name: Stop docker compose project
  hosts: targets
  become: true
  gather_facts: false
  vars:
    install_dir: "{{ install_dir | default('/opt/apps/app') }}"
    project_name: "{{ project_name | default('app') }}"
  tasks:
    - name: Detect docker compose command
      shell: |
        if docker compose version >/dev/null 2>&1; then
          echo docker compose
          exit 0
        fi
        if command -v docker-compose >/dev/null 2>&1; then
          echo docker-compose
          exit 0
        fi
        exit 2
      args:
        executable: /bin/sh
      register: compose_cmd
      changed_when: false

    - name: Stop compose services
      shell: |
        set -e
        cd {{ install_dir | quote }}
        if [ "{{ compose_cmd.stdout }}" = "docker compose" ]; then
          docker compose -p {{ project_name | quote }} stop
        else
          docker-compose -p {{ project_name | quote }} stop
        fi
      args:
        executable: /bin/sh

    - name: Inspect compose services
      shell: |
        set -e
        cd {{ install_dir | quote }}
        if [ "{{ compose_cmd.stdout }}" = "docker compose" ]; then
          docker compose -p {{ project_name | quote }} ps -a
        else
          docker-compose -p {{ project_name | quote }} ps -a
        fi
      args:
        executable: /bin/sh
      changed_when: false`), nil
	default:
		return "", ErrWithMessage(ErrInvalidParams, "不支持的 Compose 生命周期动作")
	}
}

func nullableStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func nullableJSONString(value map[string]interface{}) *string {
	if len(value) == 0 {
		return nil
	}
	encoded := string(marshalJSONMap(value))
	return &encoded
}

func stringPointerValue(value *string) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func mergeEnvContent(base, override string) string {
	base = strings.TrimSpace(base)
	override = strings.TrimSpace(override)
	if base == "" {
		return override
	}
	if override == "" {
		return base
	}
	return base + "\n" + override
}

func mergeReleaseValues(base, override map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(base)+len(override))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range override {
		out[key] = value
	}
	return out
}

func resolveBoolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func composeFirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
