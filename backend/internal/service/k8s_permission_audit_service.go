package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	"k8s-platform-backend/internal/model"
)

const (
	PermissionAuditSourceManaged = "managed_cluster"
	PermissionAuditSourceAdhoc   = "adhoc_kubeconfig"

	PermissionAuditStatusPending    = "pending"
	PermissionAuditStatusRunning    = "running"
	PermissionAuditStatusSuccess    = "success"
	PermissionAuditStatusFailed     = "failed"
	PermissionAuditStatusIncomplete = "incomplete"
	PermissionAuditStatusCanceled   = "canceled"

	PermissionAuditFindingResource   = "resource"
	PermissionAuditFindingWorkload   = "workload"
	PermissionAuditFindingAppRelease = "app_release"
	PermissionAuditFindingError      = "error"

	PermissionAuditOwnershipDirect    = "direct"
	PermissionAuditOwnershipShared    = "shared"
	PermissionAuditOwnershipUnrelated = "unrelated"

	PermissionAuditPrivilegeClusterScoped = "cluster_scoped"
	PermissionAuditPrivilegeRuntimeHigh   = "runtime_high"
	PermissionAuditPrivilegeNamespaceOnly = "namespace_only_candidate"
	PermissionAuditPrivilegeSharedCluster = "shared_cluster_dependency"

	permissionAuditCredentialTTL = 2 * time.Hour
)

type PermissionAuditCreateRequest struct {
	Mode                      string   `json:"mode"`
	IncludeRuntimeRBAC        bool     `json:"include_runtime_rbac"`
	IncludePlatformMapping    bool     `json:"include_platform_mapping"`
	IncludeOwnershipDetection bool     `json:"include_ownership_detection"`
	Namespaces                []string `json:"namespaces"`
	LabelSelector             string   `json:"label_selector"`
	ResourceAllowlist         []string `json:"resource_allowlist"`
}

type PermissionAuditAdhocCreateRequest struct {
	DisplayName string `json:"display_name"`
	Kubeconfig  string `json:"kubeconfig"`
	PermissionAuditCreateRequest
}

type PermissionAuditCreateResult struct {
	AuditID    uint64 `json:"audit_id"`
	TaskID     uint64 `json:"task_id"`
	SourceType string `json:"source_type"`
}

type ListPermissionAuditsRequest struct {
	Page       int
	PageSize   int
	SourceType string
	Status     string
	RiskLevel  string
	ClusterID  uint64
	Keyword    string
	SortBy     string
	Order      string
}

type PermissionAuditListItem struct {
	ID          uint64         `json:"id"`
	SourceType  string         `json:"source_type"`
	ClusterID   *uint64        `json:"cluster_id,omitempty"`
	ClusterName string         `json:"cluster_name"`
	DisplayName string         `json:"display_name"`
	Status      string         `json:"status"`
	TaskID      *uint64        `json:"task_id,omitempty"`
	Summary     map[string]any `json:"summary"`
	CreatedAt   string         `json:"created_at"`
	CreatedBy   uint64         `json:"created_by"`
}

type PermissionAuditDetail struct {
	ID          uint64         `json:"id"`
	SourceType  string         `json:"source_type"`
	Cluster     map[string]any `json:"cluster"`
	DisplayName string         `json:"display_name"`
	Status      string         `json:"status"`
	TaskID      *uint64        `json:"task_id,omitempty"`
	Request     map[string]any `json:"request"`
	Summary     map[string]any `json:"summary"`
	Stats       map[string]any `json:"stats"`
	Error       map[string]any `json:"error,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
	CreatedBy   uint64         `json:"created_by"`
}

type ListPermissionAuditFindingsRequest struct {
	Page              int
	PageSize          int
	FindingType       string
	RiskLevel         string
	OwnershipClass    string
	PrivilegeClass    string
	Namespace         string
	Kind              string
	DeploymentBlocker *bool
	Keyword           string
	SortBy            string
	Order             string
}

type PermissionAuditFindingItem struct {
	ID                uint64         `json:"id"`
	FindingType       string         `json:"finding_type"`
	RiskLevel         string         `json:"risk_level"`
	OwnershipClass    string         `json:"ownership_class"`
	PrivilegeClass    string         `json:"privilege_class"`
	Namespace         string         `json:"namespace"`
	Kind              string         `json:"kind"`
	Name              string         `json:"name"`
	DeploymentBlocker bool           `json:"deployment_blocker"`
	Summary           string         `json:"summary"`
	Detail            map[string]any `json:"detail"`
}

type PermissionAuditCompareSummary struct {
	AddedCount   int `json:"added_count"`
	RemovedCount int `json:"removed_count"`
	ChangedCount int `json:"changed_count"`
}

type PermissionAuditComparePair struct {
	Current  *PermissionAuditFindingItem `json:"current,omitempty"`
	Baseline *PermissionAuditFindingItem `json:"baseline,omitempty"`
}

type PermissionAuditCompareResult struct {
	AuditID         uint64                        `json:"audit_id"`
	BaselineAuditID uint64                        `json:"baseline_audit_id"`
	BaselineLabel   string                        `json:"baseline_label"`
	Summary         PermissionAuditCompareSummary `json:"summary"`
	Added           []PermissionAuditFindingItem  `json:"added"`
	Removed         []PermissionAuditFindingItem  `json:"removed"`
	Changed         []PermissionAuditComparePair  `json:"changed"`
}

type K8sPermissionAuditService struct {
	db            *gorm.DB
	taskStore     *TaskStore
	clusterReg    *ClusterRegistryService
	k8sSvc        *K8sService
	credentialTTL time.Duration
	creds         *permissionAuditCredentialStore
}

type permissionAuditCredentialStore struct {
	cache  CacheStore
	secret string
	mu     sync.RWMutex
	mem    map[uint64]string
}

type permissionAuditClients struct {
	config    *rest.Config
	dynamic   dynamic.Interface
	discovery discovery.DiscoveryInterface
	mapper    meta.RESTMapper
}

type permissionAuditResourceTarget struct {
	GVR schema.GroupVersionResource
}

type permissionAuditAvailableResource struct {
	GVR        schema.GroupVersionResource
	Kind       string
	Namespaced bool
	Verbs      map[string]bool
}

type permissionAuditRoleRuleSummary struct {
	APIGroups         []string `json:"api_groups"`
	Resources         []string `json:"resources"`
	Verbs             []string `json:"verbs"`
	ClusterScopedRule bool     `json:"cluster_scoped_rule"`
	HighRiskVerb      bool     `json:"high_risk_verb"`
	RBACWrite         bool     `json:"rbac_write"`
	SecretWrite       bool     `json:"secret_write"`
}

type permissionAuditRoleAnalysis struct {
	Scope               string
	Kind                string
	Namespace           string
	Name                string
	Rules               []permissionAuditRoleRuleSummary
	HasClusterRule      bool
	HasHighRiskVerb     bool
	HasRBACWrite        bool
	HasSecretWrite      bool
	HasClusterRuleWrite bool
}

type permissionAuditSubjectBinding struct {
	BindingKind        string
	BindingName        string
	Namespace          string
	RoleRefKind        string
	RoleRefName        string
	Role               *permissionAuditRoleAnalysis
	ClusterRoleBinding bool
}

type permissionAuditReleaseMatch struct {
	ID              uint64
	Name            string
	Namespace       string
	TemplateName    string
	CurrentRevision int
}

type permissionAuditScannedObject struct {
	GVR        schema.GroupVersionResource
	Kind       string
	Namespaced bool
	Object     unstructured.Unstructured
}

func NewK8sPermissionAuditService(db *gorm.DB, taskStore *TaskStore, clusterReg *ClusterRegistryService, k8sSvc *K8sService, cache CacheStore, secret string) *K8sPermissionAuditService {
	return &K8sPermissionAuditService{
		db:            db,
		taskStore:     taskStore,
		clusterReg:    clusterReg,
		k8sSvc:        k8sSvc,
		credentialTTL: permissionAuditCredentialTTL,
		creds:         newPermissionAuditCredentialStore(cache, secret),
	}
}

func newPermissionAuditCredentialStore(cache CacheStore, secret string) *permissionAuditCredentialStore {
	return &permissionAuditCredentialStore{cache: cache, secret: secret, mem: map[uint64]string{}}
}

func (s *permissionAuditCredentialStore) key(auditID uint64) string {
	return fmt.Sprintf("k8s:permission-audit:cred:%d", auditID)
}

func (s *permissionAuditCredentialStore) Put(ctx context.Context, auditID uint64, kubeconfig string, ttl time.Duration) error {
	if auditID == 0 || strings.TrimSpace(kubeconfig) == "" {
		return ErrWithMessage(ErrInvalidParams, "临时凭据不能为空")
	}
	if s.cache != nil && s.cache.Enabled() {
		payload := []byte(kubeconfig)
		if strings.TrimSpace(s.secret) != "" {
			enc, err := encryptText(s.secret, kubeconfig)
			if err != nil {
				return err
			}
			payload = []byte(enc)
		}
		if err := s.cache.Set(ctx, s.key(auditID), payload, ttl); err != nil {
			return err
		}
	}
	s.mu.Lock()
	s.mem[auditID] = kubeconfig
	s.mu.Unlock()
	return nil
}

func (s *permissionAuditCredentialStore) Get(ctx context.Context, auditID uint64) (string, error) {
	if auditID == 0 {
		return "", ErrWithMessage(ErrInvalidParams, "分析任务ID无效")
	}
	if s.cache != nil && s.cache.Enabled() {
		b, ok, err := s.cache.Get(ctx, s.key(auditID))
		if err == nil && ok && len(b) > 0 {
			if strings.TrimSpace(s.secret) != "" {
				return decryptText(s.secret, string(b))
			}
			return string(b), nil
		}
	}
	s.mu.RLock()
	v, ok := s.mem[auditID]
	s.mu.RUnlock()
	if !ok || strings.TrimSpace(v) == "" {
		return "", ErrNotFound
	}
	return v, nil
}

func (s *permissionAuditCredentialStore) Delete(ctx context.Context, auditID uint64) {
	if auditID == 0 {
		return
	}
	if s.cache != nil && s.cache.Enabled() {
		_ = s.cache.Del(ctx, s.key(auditID))
	}
	s.mu.Lock()
	delete(s.mem, auditID)
	s.mu.Unlock()
}

func (s *K8sPermissionAuditService) CreateManagedAudit(ctx context.Context, clusterID uint64, req PermissionAuditCreateRequest, createdBy uint64) (PermissionAuditCreateResult, error) {
	if s.db == nil || s.taskStore == nil || s.clusterReg == nil || s.k8sSvc == nil {
		return PermissionAuditCreateResult{}, fmt.Errorf("permission audit service dependencies are incomplete")
	}
	if clusterID == 0 {
		return PermissionAuditCreateResult{}, ErrWithMessage(ErrInvalidParams, "集群ID无效")
	}
	if err := s.normalizeCreateRequest(&req); err != nil {
		return PermissionAuditCreateResult{}, err
	}
	cluster, err := s.clusterReg.GetCluster(ctx, clusterID)
	if err != nil {
		return PermissionAuditCreateResult{}, err
	}
	audit := model.K8sPermissionAudit{
		SourceType:  PermissionAuditSourceManaged,
		ClusterID:   &clusterID,
		ClusterName: cluster.Name,
		DisplayName: cluster.Name,
		Status:      PermissionAuditStatusPending,
		Mode:        req.Mode,
		RequestJSON: model.JSONMap(permissionAuditCreateRequestMap(req)),
		CreatedBy:   createdBy,
	}
	if err := s.db.WithContext(ctx).Create(&audit).Error; err != nil {
		return PermissionAuditCreateResult{}, err
	}
	task, err := s.createAuditTask(ctx, audit.ID, PermissionAuditSourceManaged, cluster.Name, createdBy)
	if err != nil {
		_ = s.db.WithContext(ctx).Delete(&audit).Error
		return PermissionAuditCreateResult{}, err
	}
	taskID := uint64(task.ID)
	if err := s.db.WithContext(ctx).Model(&model.K8sPermissionAudit{}).Where("id = ?", audit.ID).Update("task_id", taskID).Error; err != nil {
		return PermissionAuditCreateResult{}, err
	}
	go s.runAudit(audit.ID)
	return PermissionAuditCreateResult{AuditID: audit.ID, TaskID: taskID, SourceType: PermissionAuditSourceManaged}, nil
}

func (s *K8sPermissionAuditService) CreateAdhocAudit(ctx context.Context, req PermissionAuditAdhocCreateRequest, createdBy uint64) (PermissionAuditCreateResult, error) {
	if s.db == nil || s.taskStore == nil || s.k8sSvc == nil {
		return PermissionAuditCreateResult{}, fmt.Errorf("permission audit service dependencies are incomplete")
	}
	if err := s.normalizeCreateRequest(&req.PermissionAuditCreateRequest); err != nil {
		return PermissionAuditCreateResult{}, err
	}
	kubeconfig := strings.TrimSpace(req.Kubeconfig)
	if kubeconfig == "" {
		return PermissionAuditCreateResult{}, ErrWithMessage(ErrInvalidParams, "管理员凭据不能为空")
	}
	clusterName, err := s.validateAdhocKubeconfig(ctx, kubeconfig)
	if err != nil {
		return PermissionAuditCreateResult{}, err
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = firstNonEmpty(clusterName, "adhoc-permission-audit")
	}
	audit := model.K8sPermissionAudit{
		SourceType:  PermissionAuditSourceAdhoc,
		ClusterName: clusterName,
		DisplayName: displayName,
		Status:      PermissionAuditStatusPending,
		Mode:        req.Mode,
		RequestJSON: model.JSONMap(permissionAuditCreateRequestMap(req.PermissionAuditCreateRequest)),
		CreatedBy:   createdBy,
	}
	if err := s.db.WithContext(ctx).Create(&audit).Error; err != nil {
		return PermissionAuditCreateResult{}, err
	}
	if err := s.creds.Put(ctx, audit.ID, kubeconfig, s.credentialTTL); err != nil {
		_ = s.db.WithContext(ctx).Delete(&audit).Error
		return PermissionAuditCreateResult{}, err
	}
	task, err := s.createAuditTask(ctx, audit.ID, PermissionAuditSourceAdhoc, displayName, createdBy)
	if err != nil {
		s.creds.Delete(ctx, audit.ID)
		_ = s.db.WithContext(ctx).Delete(&audit).Error
		return PermissionAuditCreateResult{}, err
	}
	taskID := uint64(task.ID)
	if err := s.db.WithContext(ctx).Model(&model.K8sPermissionAudit{}).Where("id = ?", audit.ID).Update("task_id", taskID).Error; err != nil {
		return PermissionAuditCreateResult{}, err
	}
	go s.runAudit(audit.ID)
	return PermissionAuditCreateResult{AuditID: audit.ID, TaskID: taskID, SourceType: PermissionAuditSourceAdhoc}, nil
}

func (s *K8sPermissionAuditService) ListAudits(ctx context.Context, req ListPermissionAuditsRequest) (PageResult[PermissionAuditListItem], error) {
	if s.db == nil {
		return PageResult[PermissionAuditListItem]{}, fmt.Errorf("db is required")
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	q := s.db.WithContext(ctx).Model(&model.K8sPermissionAudit{}).Where("deleted_at IS NULL")
	if req.ClusterID > 0 {
		q = q.Where("cluster_id = ?", req.ClusterID)
	}
	if st := strings.TrimSpace(req.SourceType); st != "" {
		q = q.Where("source_type = ?", st)
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		q = q.Where("status = ?", status)
	}
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("display_name LIKE ? OR cluster_name LIKE ?", like, like)
	}
	if rl := strings.TrimSpace(req.RiskLevel); rl != "" {
		sub := s.db.WithContext(ctx).
			Model(&model.K8sPermissionAuditFinding{}).
			Select("DISTINCT audit_id").
			Where("risk_level = ?", rl)
		q = q.Where("id IN (?)", sub)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[PermissionAuditListItem]{}, err
	}
	orderExpr := "id DESC"
	if col := sanitizePermissionAuditOrderColumn(req.SortBy); col != "" {
		dir := "DESC"
		if strings.ToLower(req.Order) == "asc" {
			dir = "ASC"
		}
		orderExpr = fmt.Sprintf("%s %s", col, dir)
	}
	var rows []model.K8sPermissionAudit
	if err := q.Order(orderExpr).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[PermissionAuditListItem]{}, err
	}
	out := make([]PermissionAuditListItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, PermissionAuditListItem{
			ID:          row.ID,
			SourceType:  row.SourceType,
			ClusterID:   row.ClusterID,
			ClusterName: row.ClusterName,
			DisplayName: row.DisplayName,
			Status:      row.Status,
			TaskID:      row.TaskID,
			Summary:     jsonMapToMap(row.SummaryJSON),
			CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
			CreatedBy:   row.CreatedBy,
		})
	}
	return PageResult[PermissionAuditListItem]{List: out, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *K8sPermissionAuditService) GetAudit(ctx context.Context, auditID uint64) (*PermissionAuditDetail, error) {
	if s.db == nil {
		return nil, fmt.Errorf("db is required")
	}
	row, err := s.getAuditRow(ctx, auditID)
	if err != nil {
		return nil, err
	}
	cluster := map[string]any{"id": 0, "name": row.ClusterName}
	if row.ClusterID != nil {
		cluster["id"] = *row.ClusterID
	}
	return &PermissionAuditDetail{
		ID:          row.ID,
		SourceType:  row.SourceType,
		Cluster:     cluster,
		DisplayName: row.DisplayName,
		Status:      row.Status,
		TaskID:      row.TaskID,
		Request:     jsonMapToMap(row.RequestJSON),
		Summary:     jsonMapToMap(row.SummaryJSON),
		Stats:       jsonMapToMap(row.StatsJSON),
		Error:       jsonMapToMap(row.ErrorJSON),
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   row.UpdatedAt.UTC().Format(time.RFC3339),
		CreatedBy:   row.CreatedBy,
	}, nil
}

func (s *K8sPermissionAuditService) ListFindings(ctx context.Context, auditID uint64, req ListPermissionAuditFindingsRequest) (PageResult[PermissionAuditFindingItem], error) {
	if s.db == nil {
		return PageResult[PermissionAuditFindingItem]{}, fmt.Errorf("db is required")
	}
	if auditID == 0 {
		return PageResult[PermissionAuditFindingItem]{}, ErrWithMessage(ErrInvalidParams, "分析任务ID无效")
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	q := s.db.WithContext(ctx).Model(&model.K8sPermissionAuditFinding{}).Where("audit_id = ?", auditID)
	if v := strings.TrimSpace(req.FindingType); v != "" {
		q = q.Where("finding_type = ?", v)
	}
	if v := strings.TrimSpace(req.RiskLevel); v != "" {
		q = q.Where("risk_level = ?", v)
	}
	if v := strings.TrimSpace(req.OwnershipClass); v != "" {
		q = q.Where("ownership_class = ?", v)
	}
	if v := strings.TrimSpace(req.PrivilegeClass); v != "" {
		q = q.Where("privilege_class = ?", v)
	}
	if v := strings.TrimSpace(req.Namespace); v != "" {
		q = q.Where("namespace = ?", v)
	}
	if v := strings.TrimSpace(req.Kind); v != "" {
		q = q.Where("kind = ?", v)
	}
	if req.DeploymentBlocker != nil {
		q = q.Where("deployment_blocker = ?", *req.DeploymentBlocker)
	}
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR workload_name LIKE ? OR service_account_name LIKE ? OR app_release_name LIKE ? OR summary LIKE ?", like, like, like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[PermissionAuditFindingItem]{}, err
	}
	orderExpr := "id DESC"
	if col := sanitizePermissionAuditFindingOrderColumn(req.SortBy); col != "" {
		dir := "DESC"
		if strings.ToLower(req.Order) == "asc" {
			dir = "ASC"
		}
		orderExpr = fmt.Sprintf("%s %s", col, dir)
	}
	var rows []model.K8sPermissionAuditFinding
	if err := q.Order(orderExpr).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[PermissionAuditFindingItem]{}, err
	}
	out := make([]PermissionAuditFindingItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, PermissionAuditFindingItem{
			ID:                row.ID,
			FindingType:       row.FindingType,
			RiskLevel:         row.RiskLevel,
			OwnershipClass:    row.OwnershipClass,
			PrivilegeClass:    row.PrivilegeClass,
			Namespace:         row.Namespace,
			Kind:              row.Kind,
			Name:              row.Name,
			DeploymentBlocker: row.DeploymentBlocker,
			Summary:           row.Summary,
			Detail:            jsonMapToMap(row.DetailJSON),
		})
	}
	return PageResult[PermissionAuditFindingItem]{List: out, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *K8sPermissionAuditService) GetLatestClusterAudit(ctx context.Context, clusterID uint64) (*PermissionAuditDetail, error) {
	if s.db == nil {
		return nil, fmt.Errorf("db is required")
	}
	if clusterID == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "集群ID无效")
	}
	var row model.K8sPermissionAudit
	err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND cluster_id = ? AND source_type = ? AND status IN ?", clusterID, PermissionAuditSourceManaged, []string{PermissionAuditStatusSuccess, PermissionAuditStatusIncomplete}).
		Order("id DESC").
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.GetAudit(ctx, row.ID)
}

func (s *K8sPermissionAuditService) CancelAudit(ctx context.Context, auditID uint64) error {
	if s == nil || s.taskStore == nil {
		return fmt.Errorf("task store is required")
	}
	row, err := s.getAuditRow(ctx, auditID)
	if err != nil {
		return err
	}
	if row.TaskID == nil || *row.TaskID == 0 {
		return ErrWithMessage(ErrConflict, "当前分析没有可取消任务")
	}
	if err := NewTaskService(s.taskStore).Cancel(int64(*row.TaskID)); err != nil {
		switch err {
		case ErrTaskNotFound:
			return ErrNotFound
		case ErrTaskCannotCancel:
			return ErrConflict
		default:
			return err
		}
	}
	return nil
}

func (s *K8sPermissionAuditService) AuditTaskLogs(ctx context.Context, auditID uint64, offset, limit int) (PermissionAuditTaskLogsResult, error) {
	if s == nil || s.taskStore == nil {
		return PermissionAuditTaskLogsResult{}, fmt.Errorf("task store is required")
	}
	row, err := s.getAuditRow(ctx, auditID)
	if err != nil {
		return PermissionAuditTaskLogsResult{}, err
	}
	if row.TaskID == nil || *row.TaskID == 0 {
		return PermissionAuditTaskLogsResult{}, ErrNotFound
	}
	if limit <= 0 {
		limit = 200
	}
	task, ok := s.taskStore.Get(int64(*row.TaskID))
	if !ok || task == nil {
		return PermissionAuditTaskLogsResult{}, ErrNotFound
	}
	return PermissionAuditTaskLogsResult{
		TaskID:    *row.TaskID,
		Offset:    offset,
		Limit:     limit,
		Lines:     task.Logs(offset, limit),
		Status:    string(task.Status),
		CanCancel: task.CanCancel(),
	}, nil
}

func (s *K8sPermissionAuditService) CompareAudits(ctx context.Context, auditID, baselineAuditID uint64) (PermissionAuditCompareResult, error) {
	currentAudit, err := s.getAuditRow(ctx, auditID)
	if err != nil {
		return PermissionAuditCompareResult{}, err
	}
	baselineAudit, err := s.resolveCompareBaseline(ctx, currentAudit, baselineAuditID)
	if err != nil {
		return PermissionAuditCompareResult{}, err
	}
	currentFindings, err := s.loadAuditFindings(ctx, currentAudit.ID)
	if err != nil {
		return PermissionAuditCompareResult{}, err
	}
	baselineFindings, err := s.loadAuditFindings(ctx, baselineAudit.ID)
	if err != nil {
		return PermissionAuditCompareResult{}, err
	}
	currentMap := map[string]PermissionAuditFindingItem{}
	baselineMap := map[string]PermissionAuditFindingItem{}
	keys := map[string]bool{}
	for _, item := range currentFindings {
		key := permissionAuditCompareKey(item)
		currentMap[key] = item
		keys[key] = true
	}
	for _, item := range baselineFindings {
		key := permissionAuditCompareKey(item)
		baselineMap[key] = item
		keys[key] = true
	}
	added := make([]PermissionAuditFindingItem, 0)
	removed := make([]PermissionAuditFindingItem, 0)
	changed := make([]PermissionAuditComparePair, 0)
	orderedKeys := make([]string, 0, len(keys))
	for key := range keys {
		orderedKeys = append(orderedKeys, key)
	}
	sort.Strings(orderedKeys)
	for _, key := range orderedKeys {
		current, hasCurrent := currentMap[key]
		baseline, hasBaseline := baselineMap[key]
		switch {
		case hasCurrent && !hasBaseline:
			added = append(added, current)
		case !hasCurrent && hasBaseline:
			removed = append(removed, baseline)
		case hasCurrent && hasBaseline:
			if permissionAuditFindingsEqual(current, baseline) {
				continue
			}
			currentCopy := current
			baselineCopy := baseline
			changed = append(changed, PermissionAuditComparePair{Current: &currentCopy, Baseline: &baselineCopy})
		}
	}
	return PermissionAuditCompareResult{
		AuditID:         currentAudit.ID,
		BaselineAuditID: baselineAudit.ID,
		BaselineLabel:   fmt.Sprintf("#%d %s", baselineAudit.ID, baselineAudit.CreatedAt.UTC().Format("2006-01-02 15:04:05")),
		Summary: PermissionAuditCompareSummary{
			AddedCount:   len(added),
			RemovedCount: len(removed),
			ChangedCount: len(changed),
		},
		Added:   added,
		Removed: removed,
		Changed: changed,
	}, nil
}

func (s *K8sPermissionAuditService) resolveCompareBaseline(ctx context.Context, current *model.K8sPermissionAudit, baselineAuditID uint64) (*model.K8sPermissionAudit, error) {
	if baselineAuditID > 0 {
		return s.getAuditRow(ctx, baselineAuditID)
	}
	if s.db == nil {
		return nil, fmt.Errorf("db is required")
	}
	var row model.K8sPermissionAudit
	query := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id < ? AND source_type = ? AND status IN ?", current.ID, current.SourceType, []string{PermissionAuditStatusSuccess, PermissionAuditStatusIncomplete})
	if current.ClusterID != nil {
		query = query.Where("cluster_id = ?", *current.ClusterID)
	} else {
		query = query.Where("cluster_id IS NULL")
	}
	err := query.Order("id DESC").First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (s *K8sPermissionAuditService) loadAuditFindings(ctx context.Context, auditID uint64) ([]PermissionAuditFindingItem, error) {
	rows, err := s.loadFindingRows(ctx, auditID)
	if err != nil {
		return nil, err
	}
	items := make([]PermissionAuditFindingItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, permissionAuditFindingRowToItem(row))
	}
	return items, nil
}

func (s *K8sPermissionAuditService) loadFindingRows(ctx context.Context, auditID uint64) ([]model.K8sPermissionAuditFinding, error) {
	if s.db == nil {
		return nil, fmt.Errorf("db is required")
	}
	var rows []model.K8sPermissionAuditFinding
	if err := s.db.WithContext(ctx).Where("audit_id = ?", auditID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func permissionAuditFindingRowToItem(row model.K8sPermissionAuditFinding) PermissionAuditFindingItem {
	return PermissionAuditFindingItem{
		ID:                row.ID,
		FindingType:       row.FindingType,
		RiskLevel:         row.RiskLevel,
		OwnershipClass:    row.OwnershipClass,
		PrivilegeClass:    row.PrivilegeClass,
		Namespace:         row.Namespace,
		Kind:              row.Kind,
		Name:              row.Name,
		DeploymentBlocker: row.DeploymentBlocker,
		Summary:           row.Summary,
		Detail:            jsonMapToMap(row.DetailJSON),
	}
}

func permissionAuditCompareKey(item PermissionAuditFindingItem) string {
	return strings.Join([]string{item.FindingType, item.Kind, item.Namespace, item.Name}, "|")
}

func permissionAuditFindingsEqual(a, b PermissionAuditFindingItem) bool {
	return a.RiskLevel == b.RiskLevel &&
		a.OwnershipClass == b.OwnershipClass &&
		a.PrivilegeClass == b.PrivilegeClass &&
		a.DeploymentBlocker == b.DeploymentBlocker &&
		a.Summary == b.Summary
}

func (s *K8sPermissionAuditService) normalizeCreateRequest(req *PermissionAuditCreateRequest) error {
	if req == nil {
		return ErrWithMessage(ErrInvalidParams, "请求不能为空")
	}
	req.Mode = firstNonEmpty(strings.TrimSpace(req.Mode), "full")
	if req.Mode != "full" {
		return ErrWithMessage(ErrInvalidParams, "当前仅支持 full 分析模式")
	}
	trimmedNamespaces := make([]string, 0, len(req.Namespaces))
	seen := map[string]bool{}
	for _, ns := range req.Namespaces {
		value := strings.TrimSpace(ns)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		trimmedNamespaces = append(trimmedNamespaces, value)
	}
	req.Namespaces = trimmedNamespaces
	req.LabelSelector = strings.TrimSpace(req.LabelSelector)
	trimmedAllowlist := make([]string, 0, len(req.ResourceAllowlist))
	allowSeen := map[string]bool{}
	for _, item := range req.ResourceAllowlist {
		value := normalizePermissionAuditAllowlistItem(item)
		if value == "" || allowSeen[value] {
			continue
		}
		allowSeen[value] = true
		trimmedAllowlist = append(trimmedAllowlist, value)
	}
	req.ResourceAllowlist = trimmedAllowlist
	return nil
}

func permissionAuditCreateRequestMap(req PermissionAuditCreateRequest) map[string]any {
	return map[string]any{
		"mode":                        req.Mode,
		"include_runtime_rbac":        req.IncludeRuntimeRBAC,
		"include_platform_mapping":    req.IncludePlatformMapping,
		"include_ownership_detection": req.IncludeOwnershipDetection,
		"namespaces":                  req.Namespaces,
		"label_selector":              req.LabelSelector,
		"resource_allowlist":          req.ResourceAllowlist,
	}
}

func jsonMapToMap(v model.JSONMap) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return map[string]any(v)
}

func sanitizePermissionAuditOrderColumn(col string) string {
	allowed := map[string]bool{"id": true, "display_name": true, "status": true, "created_at": true, "updated_at": true}
	if allowed[col] {
		return col
	}
	return ""
}

func sanitizePermissionAuditFindingOrderColumn(col string) string {
	allowed := map[string]bool{"id": true, "risk_level": true, "kind": true, "namespace": true, "name": true, "created_at": true}
	if allowed[col] {
		return col
	}
	return ""
}

func (s *K8sPermissionAuditService) createAuditTask(ctx context.Context, auditID uint64, sourceType, displayName string, createdBy uint64) (*Task, error) {
	title := fmt.Sprintf("K8s 权限分析：%s", firstNonEmpty(displayName, fmt.Sprintf("audit-%d", auditID)))
	task := &Task{
		Type:      "k8s_permission_audit",
		Status:    TaskPending,
		Title:     &title,
		CreatedBy: int64(createdBy),
		Meta: map[string]any{
			"audit_id":    auditID,
			"source_type": sourceType,
		},
		Steps: []TaskStep{
			{Key: "prepare", Title: "准备分析", Status: StepPending},
			{Key: "scan", Title: "扫描资源", Status: StepPending},
			{Key: "analyze", Title: "分析权限", Status: StepPending},
			{Key: "persist", Title: "保存结果", Status: StepPending},
		},
	}
	if err := s.taskStore.Put(task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *K8sPermissionAuditService) getAuditRow(ctx context.Context, auditID uint64) (*model.K8sPermissionAudit, error) {
	if auditID == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "分析任务ID无效")
	}
	var row model.K8sPermissionAudit
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", auditID).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (s *K8sPermissionAuditService) validateAdhocKubeconfig(ctx context.Context, kubeconfig string) (string, error) {
	kc := strings.TrimSpace(kubeconfig)
	if kc == "" {
		return "", ErrWithMessage(ErrInvalidParams, "管理员凭据不能为空")
	}
	loaded, err := clientcmd.Load([]byte(kc))
	if err != nil {
		return "", ErrWithMessage(ErrInvalidParams, "管理员凭据格式无效")
	}
	clusterName := ""
	if loaded != nil {
		clusterName = strings.TrimSpace(loaded.CurrentContext)
		if ctxCfg, ok := loaded.Contexts[loaded.CurrentContext]; ok && ctxCfg != nil {
			clusterName = firstNonEmpty(clusterName, strings.TrimSpace(ctxCfg.Cluster))
		}
	}
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kc))
	if err != nil {
		return "", ErrWithMessage(ErrInvalidParams, "管理员凭据格式无效")
	}
	cfg.Timeout = 15 * time.Second
	disc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return clusterName, normalizeK8sErr(err)
	}
	_, err = disc.ServerVersion()
	if err != nil {
		return clusterName, normalizeK8sErr(err)
	}
	return clusterName, nil
}

func (s *K8sPermissionAuditService) runAudit(auditID uint64) {
	ctx := context.Background()
	row, err := s.getAuditRow(ctx, auditID)
	if err != nil {
		return
	}
	var task *Task
	if row.TaskID != nil {
		if t, ok := s.taskStore.Get(int64(*row.TaskID)); ok {
			task = t
		}
	}
	runCtx, cancel := context.WithCancel(context.Background())
	if task != nil {
		s.taskStore.RegisterCancel(task.ID, cancel)
		defer s.taskStore.UnregisterCancel(task.ID)
	}
	defer cancel()
	updateStep := func(index int, status TaskStepStatus, message string) {
		if task == nil || index < 0 || index >= len(task.Steps) {
			return
		}
		now := time.Now().UTC()
		task.Steps[index].Status = status
		if status == StepRunning {
			task.Steps[index].StartedAt = &now
		}
		if status == StepSuccess || status == StepFailed {
			task.Steps[index].FinishedAt = &now
		}
		if strings.TrimSpace(message) != "" {
			msg := strings.TrimSpace(message)
			task.Steps[index].Message = &msg
			task.AppendLog(msg)
		}
		_ = task.Update()
	}
	setAuditStatus := func(status string, errorMap map[string]any) {
		updates := map[string]any{"status": status}
		if errorMap != nil {
			updates["error_json"] = model.JSONMap(errorMap)
		}
		_ = s.db.WithContext(context.Background()).Model(&model.K8sPermissionAudit{}).Where("id = ?", auditID).Updates(updates).Error
	}
	if task != nil {
		task.Status = TaskRunning
		startMsg := "K8s 权限分析开始"
		task.Message = &startMsg
		_ = task.Update()
	}
	setAuditStatus(PermissionAuditStatusRunning, nil)
	updateStep(0, StepRunning, "准备 K8s 客户端与分析上下文")
	clients, auditReq, err := s.loadClientsForAudit(runCtx, row)
	if err != nil {
		if errors.Is(err, context.Canceled) || runCtx.Err() != nil {
			s.finishAuditCanceled(row, task)
			return
		}
		s.finishAuditWithError(row, task, PermissionAuditStatusFailed, err)
		updateStep(0, StepFailed, fmt.Sprintf("准备失败：%v", err))
		return
	}
	updateStep(0, StepSuccess, "准备完成")
	updateStep(1, StepRunning, "开始扫描集群资源")
	available, discoveryErrors := s.discoverAvailableResources(runCtx, clients.discovery)
	objects, scanErrors, err := s.scanTargetResources(runCtx, clients.dynamic, available, auditReq, task)
	if err != nil {
		if errors.Is(err, context.Canceled) || runCtx.Err() != nil {
			s.finishAuditCanceled(row, task)
			return
		}
		s.finishAuditWithError(row, task, PermissionAuditStatusFailed, err)
		updateStep(1, StepFailed, fmt.Sprintf("扫描失败：%v", err))
		return
	}
	if runCtx.Err() != nil {
		s.finishAuditCanceled(row, task)
		return
	}
	updateStep(1, StepSuccess, fmt.Sprintf("扫描完成，资源数：%d", len(objects)))
	updateStep(2, StepRunning, "开始分析权限与资源归属")
	findings, summary, stats, analyzeErrors := s.analyzeScannedResources(runCtx, row, auditReq, clients.mapper, objects)
	if runCtx.Err() != nil {
		s.finishAuditCanceled(row, task)
		return
	}
	status := PermissionAuditStatusSuccess
	allErrors := append(discoveryErrors, scanErrors...)
	allErrors = append(allErrors, analyzeErrors...)
	if len(allErrors) > 0 {
		status = PermissionAuditStatusIncomplete
	}
	updateStep(2, StepSuccess, fmt.Sprintf("分析完成，结论数：%d", len(findings)))
	updateStep(3, StepRunning, "保存分析结果")
	if err := s.persistAuditResult(runCtx, row.ID, status, summary, stats, findings, allErrors); err != nil {
		if errors.Is(err, context.Canceled) || runCtx.Err() != nil {
			s.finishAuditCanceled(row, task)
			return
		}
		s.finishAuditWithError(row, task, PermissionAuditStatusFailed, err)
		updateStep(3, StepFailed, fmt.Sprintf("保存失败：%v", err))
		return
	}
	updateStep(3, StepSuccess, "结果已保存")
	if task != nil {
		msg := "K8s 权限分析完成"
		if status == PermissionAuditStatusIncomplete {
			msg = "K8s 权限分析完成，但存在部分资源扫描失败"
			task.Status = TaskSuccess
		} else {
			task.Status = TaskStatus(status)
		}
		task.Message = &msg
		percent := 100
		task.Percent = &percent
		_ = task.Update()
	}
	if row.SourceType == PermissionAuditSourceAdhoc {
		s.creds.Delete(context.Background(), row.ID)
	}
}

func (s *K8sPermissionAuditService) finishAuditWithError(row *model.K8sPermissionAudit, task *Task, status string, err error) {
	msg := firstNonEmpty(serviceErrorMessage(err), "权限分析执行失败")
	_ = s.db.WithContext(context.Background()).Model(&model.K8sPermissionAudit{}).Where("id = ?", row.ID).Updates(map[string]any{
		"status":     status,
		"error_json": model.JSONMap{"message": msg},
	}).Error
	if task != nil {
		task.Status = TaskFailed
		task.Message = &msg
		_ = task.Update()
		task.AppendLog("[Error] " + msg)
	}
	if row.SourceType == PermissionAuditSourceAdhoc {
		s.creds.Delete(context.Background(), row.ID)
	}
}

func (s *K8sPermissionAuditService) finishAuditCanceled(row *model.K8sPermissionAudit, task *Task) {
	msg := "K8s 权限分析已取消"
	_ = s.db.WithContext(context.Background()).Model(&model.K8sPermissionAudit{}).Where("id = ?", row.ID).Updates(map[string]any{
		"status":     PermissionAuditStatusCanceled,
		"error_json": model.JSONMap{"message": msg},
	}).Error
	if task != nil {
		task.Status = TaskCanceled
		task.Message = &msg
		_ = task.Update()
		task.AppendLog(msg)
	}
	if row.SourceType == PermissionAuditSourceAdhoc {
		s.creds.Delete(context.Background(), row.ID)
	}
}

func serviceErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if msg, ok := UserMessage(err); ok {
		return msg
	}
	return strings.TrimSpace(err.Error())
}

func (s *K8sPermissionAuditService) loadClientsForAudit(ctx context.Context, row *model.K8sPermissionAudit) (*permissionAuditClients, PermissionAuditCreateRequest, error) {
	request := PermissionAuditCreateRequest{Mode: "full", IncludeRuntimeRBAC: true, IncludeOwnershipDetection: true}
	if row.RequestJSON != nil {
		b, _ := json.Marshal(row.RequestJSON)
		_ = json.Unmarshal(b, &request)
	}
	var cfg *rest.Config
	var err error
	switch row.SourceType {
	case PermissionAuditSourceManaged:
		if row.ClusterID == nil || *row.ClusterID == 0 {
			return nil, request, ErrWithMessage(ErrInvalidParams, "缺少集群标识")
		}
		cfg, err = s.k8sSvc.restConfig(ctx, *row.ClusterID)
	case PermissionAuditSourceAdhoc:
		kubeconfig, getErr := s.creds.Get(ctx, row.ID)
		if getErr != nil {
			return nil, request, ErrWithMessage(ErrNotFound, "临时管理员凭据已失效，请重新发起分析")
		}
		cfg, err = clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
		if err == nil {
			cfg.Timeout = k8sRequestTimeout
		}
	default:
		return nil, request, ErrWithMessage(ErrInvalidParams, "未知的分析来源")
	}
	if err != nil {
		return nil, request, normalizeK8sErr(err)
	}
	dyn, err := dynamic.NewForConfig(rest.CopyConfig(cfg))
	if err != nil {
		return nil, request, normalizeK8sErr(err)
	}
	disc, err := discovery.NewDiscoveryClientForConfig(rest.CopyConfig(cfg))
	if err != nil {
		return nil, request, normalizeK8sErr(err)
	}
	groupResources, err := restmapper.GetAPIGroupResources(disc)
	if err != nil {
		return nil, request, normalizeK8sErr(err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	return &permissionAuditClients{config: cfg, dynamic: dyn, discovery: disc, mapper: mapper}, request, nil
}

func (s *K8sPermissionAuditService) discoverAvailableResources(ctx context.Context, disc discovery.DiscoveryInterface) (map[string]permissionAuditAvailableResource, []string) {
	resources := map[string]permissionAuditAvailableResource{}
	partialErrors := []string{}
	list, err := disc.ServerPreferredResources()
	if err != nil {
		partialErrors = append(partialErrors, err.Error())
	}
	for _, rl := range list {
		gv, parseErr := schema.ParseGroupVersion(rl.GroupVersion)
		if parseErr != nil {
			partialErrors = append(partialErrors, parseErr.Error())
			continue
		}
		for _, apiRes := range rl.APIResources {
			if strings.Contains(apiRes.Name, "/") {
				continue
			}
			verbs := map[string]bool{}
			for _, v := range apiRes.Verbs {
				verbs[strings.ToLower(strings.TrimSpace(v))] = true
			}
			gvr := schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: apiRes.Name}
			resources[permissionAuditGVRKey(gvr)] = permissionAuditAvailableResource{GVR: gvr, Kind: apiRes.Kind, Namespaced: apiRes.Namespaced, Verbs: verbs}
		}
	}
	return resources, partialErrors
}

func (s *K8sPermissionAuditService) scanTargetResources(ctx context.Context, dyn dynamic.Interface, available map[string]permissionAuditAvailableResource, req PermissionAuditCreateRequest, task *Task) ([]permissionAuditScannedObject, []string, error) {
	targets := permissionAuditTargets(req.ResourceAllowlist)
	objects := make([]permissionAuditScannedObject, 0, 256)
	partialErrors := []string{}
	processed := 0
	for _, target := range targets {
		if ctx.Err() != nil {
			return nil, partialErrors, ctx.Err()
		}
		resolved, ok := resolvePermissionAuditTarget(available, target)
		if !ok {
			continue
		}
		items, err := listPermissionAuditObjects(ctx, dyn, resolved, req.Namespaces, req.LabelSelector)
		if err != nil {
			partialErrors = append(partialErrors, fmt.Sprintf("%s list failed: %v", resolved.GVR.Resource, err))
			if task != nil {
				task.AppendLog(fmt.Sprintf("[Audit] 资源 %s 扫描失败：%v", resolved.GVR.Resource, err))
			}
			continue
		}
		for _, item := range items {
			objects = append(objects, permissionAuditScannedObject{GVR: resolved.GVR, Kind: resolved.Kind, Namespaced: resolved.Namespaced, Object: item})
		}
		processed++
		if task != nil {
			task.AppendLog(fmt.Sprintf("[Audit] 扫描 %s 完成，数量=%d", resolved.GVR.Resource, len(items)))
			percent := 10 + int(float64(processed)/float64(len(targets))*45)
			task.Percent = &percent
			_ = task.Update()
		}
	}
	return objects, partialErrors, nil
}

func listPermissionAuditObjects(ctx context.Context, dyn dynamic.Interface, meta permissionAuditAvailableResource, namespaces []string, labelSelector string) ([]unstructured.Unstructured, error) {
	opts := metav1.ListOptions{Limit: k8sListPageLimit}
	if strings.TrimSpace(labelSelector) != "" {
		opts.LabelSelector = strings.TrimSpace(labelSelector)
	}
	all := make([]unstructured.Unstructured, 0, 64)
	listNamespace := func(ns string) error {
		listOpts := opts
		for {
			var list *unstructured.UnstructuredList
			var err error
			if meta.Namespaced {
				list, err = dyn.Resource(meta.GVR).Namespace(ns).List(ctx, listOpts)
			} else {
				list, err = dyn.Resource(meta.GVR).List(ctx, listOpts)
			}
			if err != nil {
				return normalizeK8sErr(err)
			}
			all = append(all, list.Items...)
			if strings.TrimSpace(list.GetContinue()) == "" {
				break
			}
			listOpts.Continue = list.GetContinue()
		}
		return nil
	}
	if meta.Namespaced {
		if len(namespaces) == 0 {
			if err := listNamespace(metav1.NamespaceAll); err != nil {
				return nil, err
			}
		} else {
			for _, ns := range namespaces {
				if err := listNamespace(ns); err != nil {
					return nil, err
				}
			}
		}
	} else {
		if err := listNamespace(""); err != nil {
			return nil, err
		}
	}
	return all, nil
}

func permissionAuditTargets(allowlist []string) []permissionAuditResourceTarget {
	targets := []permissionAuditResourceTarget{
		{GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}},
		{GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}},
		{GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}},
		{GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}},
		{GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}},
		{GVR: schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}},
		{GVR: schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"}},
		{GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}},
		{GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}},
		{GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}},
		{GVR: schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"}},
		{GVR: schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"}},
		{GVR: schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"}},
		{GVR: schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"}},
		{GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"}},
		{GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}},
		{GVR: schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"}},
		{GVR: schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}},
		{GVR: schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}},
		{GVR: schema.GroupVersionResource{Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"}},
		{GVR: schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}},
		{GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}},
		{GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}},
		{GVR: schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}},
		{GVR: schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"}},
		{GVR: schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"}},
		{GVR: schema.GroupVersionResource{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"}},
		{GVR: schema.GroupVersionResource{Group: "scheduling.k8s.io", Version: "v1", Resource: "priorityclasses"}},
	}
	if len(allowlist) == 0 {
		return targets
	}
	allowed := map[string]bool{}
	for _, item := range allowlist {
		if value := normalizePermissionAuditAllowlistItem(item); value != "" {
			allowed[value] = true
		}
	}
	filtered := make([]permissionAuditResourceTarget, 0, len(targets))
	for _, target := range targets {
		resource := normalizePermissionAuditAllowlistItem(target.GVR.Resource)
		if allowed[resource] {
			filtered = append(filtered, target)
		}
	}
	if len(filtered) == 0 {
		return targets
	}
	return filtered
}

func normalizePermissionAuditAllowlistItem(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	v = strings.ReplaceAll(v, "_", "")
	v = strings.ReplaceAll(v, "-", "")
	v = strings.ReplaceAll(v, ".", "")
	v = strings.ReplaceAll(v, "/", "")
	switch v {
	case "deployment", "deployments":
		return "deployments"
	case "statefulset", "statefulsets":
		return "statefulsets"
	case "daemonset", "daemonsets":
		return "daemonsets"
	case "pod", "pods":
		return "pods"
	case "service", "services":
		return "services"
	case "ingress", "ingresses":
		return "ingresses"
	case "ingressclass", "ingressclasses":
		return "ingressclasses"
	case "configmap", "configmaps":
		return "configmaps"
	case "secret", "secrets":
		return "secrets"
	case "serviceaccount", "serviceaccounts":
		return "serviceaccounts"
	case "role", "roles":
		return "roles"
	case "rolebinding", "rolebindings":
		return "rolebindings"
	case "clusterrole", "clusterroles":
		return "clusterroles"
	case "clusterrolebinding", "clusterrolebindings":
		return "clusterrolebindings"
	case "persistentvolume", "persistentvolumes", "pv", "pvs":
		return "persistentvolumes"
	case "persistentvolumeclaim", "persistentvolumeclaims", "pvc", "pvcs":
		return "persistentvolumeclaims"
	case "storageclass", "storageclasses":
		return "storageclasses"
	case "job", "jobs":
		return "jobs"
	case "cronjob", "cronjobs":
		return "cronjobs"
	case "poddisruptionbudget", "poddisruptionbudgets", "pdb", "pdbs":
		return "poddisruptionbudgets"
	case "horizontalpodautoscaler", "horizontalpodautoscalers", "hpa", "hpas":
		return "horizontalpodautoscalers"
	case "namespace", "namespaces":
		return "namespaces"
	case "node", "nodes":
		return "nodes"
	case "customresourcedefinition", "customresourcedefinitions", "crd", "crds":
		return "customresourcedefinitions"
	case "validatingwebhookconfiguration", "validatingwebhookconfigurations":
		return "validatingwebhookconfigurations"
	case "mutatingwebhookconfiguration", "mutatingwebhookconfigurations":
		return "mutatingwebhookconfigurations"
	case "apiservice", "apiservices":
		return "apiservices"
	case "priorityclass", "priorityclasses":
		return "priorityclasses"
	default:
		return ""
	}
}

func resolvePermissionAuditTarget(available map[string]permissionAuditAvailableResource, target permissionAuditResourceTarget) (permissionAuditAvailableResource, bool) {
	for _, candidate := range candidateGVRs(target.GVR) {
		if res, ok := available[permissionAuditGVRKey(candidate)]; ok {
			return res, true
		}
	}
	return permissionAuditAvailableResource{}, false
}

func permissionAuditGVRKey(gvr schema.GroupVersionResource) string {
	return strings.ToLower(strings.Join([]string{gvr.Group, gvr.Version, gvr.Resource}, "|"))
}

func permissionAuditResourceKey(apiVersion, kind, namespace, name string) string {
	return strings.ToLower(strings.Join([]string{strings.TrimSpace(apiVersion), strings.TrimSpace(kind), strings.TrimSpace(namespace), strings.TrimSpace(name)}, "|"))
}

func permissionAuditRoleKey(kind, namespace, name string) string {
	return strings.ToLower(strings.Join([]string{kind, namespace, name}, "|"))
}

func permissionAuditSubjectKey(namespace, name string) string {
	return strings.ToLower(strings.Join([]string{namespace, name}, "|"))
}

func (s *K8sPermissionAuditService) analyzeScannedResources(ctx context.Context, row *model.K8sPermissionAudit, req PermissionAuditCreateRequest, mapper meta.RESTMapper, objects []permissionAuditScannedObject) ([]model.K8sPermissionAuditFinding, map[string]any, map[string]any, []string) {
	resourceToRelease := map[string]permissionAuditReleaseMatch{}
	releaseNamespaces := map[string]bool{}
	ownershipEnabled := req.IncludeOwnershipDetection
	if ownershipEnabled && req.IncludePlatformMapping && row.SourceType == PermissionAuditSourceManaged && row.ClusterID != nil {
		resourceToRelease, releaseNamespaces = s.buildPlatformReleaseIndex(ctx, *row.ClusterID, mapper)
	}
	roles := map[string]*permissionAuditRoleAnalysis{}
	bindingsBySubject := map[string][]permissionAuditBinding{}
	workloadObjects := make([]permissionAuditScannedObject, 0)
	resourceObjects := make([]permissionAuditScannedObject, 0, len(objects))
	storageClassRefs := map[string]bool{}
	ingressClassRefs := map[string]bool{}
	serviceAccountRefs := map[string]bool{}
	for _, item := range objects {
		resourceObjects = append(resourceObjects, item)
		kind := firstNonEmpty(item.Object.GetKind(), item.Kind)
		switch kind {
		case "Role", "ClusterRole":
			if req.IncludeRuntimeRBAC {
				analysis := analyzePermissionAuditRole(item)
				roles[permissionAuditRoleKey(kind, item.Object.GetNamespace(), item.Object.GetName())] = analysis
			}
		case "RoleBinding", "ClusterRoleBinding":
			if req.IncludeRuntimeRBAC {
				for _, binding := range analyzePermissionAuditBinding(item, roles) {
					key := permissionAuditSubjectKey(subjectNamespaceForBinding(item, binding.Namespace), binding.SubjectName())
					bindingsBySubject[key] = append(bindingsBySubject[key], binding)
				}
			}
		case "Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob":
			workloadObjects = append(workloadObjects, item)
			sa := extractPermissionAuditWorkloadServiceAccount(item.Object)
			serviceAccountRefs[permissionAuditSubjectKey(item.Object.GetNamespace(), sa)] = true
		case "Ingress":
			if className := extractPermissionAuditIngressClass(item.Object); className != "" {
				ingressClassRefs[className] = true
			}
		case "PersistentVolumeClaim":
			if className := extractPermissionAuditStorageClass(item.Object); className != "" {
				storageClassRefs[className] = true
			}
		}
	}
	findings := make([]model.K8sPermissionAuditFinding, 0, len(resourceObjects)+len(workloadObjects))
	partialErrors := []string{}
	summaryCounts := map[string]int{
		"total_resources":                    0,
		"cluster_scoped_resources":           0,
		"namespaced_resources":               0,
		"high_privilege_workloads":           0,
		"namespace_only_candidate_workloads": 0,
		"unmapped_resources":                 0,
	}
	ownershipCounts := map[string]int{PermissionAuditOwnershipDirect: 0, PermissionAuditOwnershipShared: 0, PermissionAuditOwnershipUnrelated: 0}
	riskCounts := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0}
	blockerCount := 0
	sharedCount := 0
	appAggregates := map[uint64][]model.K8sPermissionAuditFinding{}
	for _, item := range resourceObjects {
		kind := firstNonEmpty(item.Object.GetKind(), item.Kind)
		apiVersion := item.Object.GetAPIVersion()
		namespace := item.Object.GetNamespace()
		name := item.Object.GetName()
		resourceKey := permissionAuditResourceKey(apiVersion, kind, namespace, name)
		ownership := PermissionAuditOwnershipUnrelated
		var releaseMatch permissionAuditReleaseMatch
		if ownershipEnabled {
			if match, ok := resourceToRelease[resourceKey]; ok {
				ownership = PermissionAuditOwnershipDirect
				releaseMatch = match
			} else if isPermissionAuditSharedResource(item, releaseNamespaces, serviceAccountRefs, storageClassRefs, ingressClassRefs) {
				ownership = PermissionAuditOwnershipShared
			}
		}
		scope := "cluster"
		if item.Namespaced {
			scope = "namespace"
			summaryCounts["namespaced_resources"]++
		} else {
			summaryCounts["cluster_scoped_resources"]++
		}
		summaryCounts["total_resources"]++
		ownershipCounts[ownership]++
		if ownership != PermissionAuditOwnershipUnrelated && releaseMatch.ID == 0 {
			summaryCounts["unmapped_resources"]++
		}
		deploymentBlocker := scope == "cluster" && ownership == PermissionAuditOwnershipDirect
		dependsOnShared := ownership == PermissionAuditOwnershipShared
		if deploymentBlocker {
			blockerCount++
		}
		if dependsOnShared {
			sharedCount++
		}
		riskLevel := permissionAuditResourceRiskLevel(item, ownership, deploymentBlocker)
		riskCounts[riskLevel]++
		reasonCodes := permissionAuditReasonCodesForResource(item, ownership, deploymentBlocker)
		detail := map[string]any{
			"api_version": item.Object.GetAPIVersion(),
			"scope":       scope,
		}
		if releaseMatch.ID > 0 {
			detail["platform_app_release"] = map[string]any{
				"id":               releaseMatch.ID,
				"name":             releaseMatch.Name,
				"namespace":        releaseMatch.Namespace,
				"current_revision": releaseMatch.CurrentRevision,
				"template_name":    releaseMatch.TemplateName,
			}
		}
		finding := model.K8sPermissionAuditFinding{
			AuditID:                          row.ID,
			FindingType:                      PermissionAuditFindingResource,
			RiskLevel:                        riskLevel,
			OwnershipClass:                   ownership,
			PrivilegeClass:                   permissionAuditPrivilegeClassForResource(item, ownership, deploymentBlocker),
			Scope:                            scope,
			DeploymentBlocker:                deploymentBlocker,
			DependsOnSharedClusterCapability: dependsOnShared,
			APIVersion:                       apiVersion,
			Kind:                             kind,
			Namespace:                        namespace,
			Name:                             name,
			AppReleaseName:                   releaseMatch.Name,
			Summary:                          permissionAuditSummaryForResource(kind, ownership, scope, deploymentBlocker),
			ReasonCodes:                      model.JSONStringSlice(reasonCodes),
			DetailJSON:                       model.JSONMap(detail),
		}
		if releaseMatch.ID > 0 {
			finding.AppReleaseID = &releaseMatch.ID
		}
		findings = append(findings, finding)
		if finding.AppReleaseID != nil {
			appAggregates[*finding.AppReleaseID] = append(appAggregates[*finding.AppReleaseID], finding)
		}
	}
	for _, item := range workloadObjects {
		kind := firstNonEmpty(item.Object.GetKind(), item.Kind)
		apiVersion := item.Object.GetAPIVersion()
		namespace := item.Object.GetNamespace()
		name := item.Object.GetName()
		resourceKey := permissionAuditResourceKey(apiVersion, kind, namespace, name)
		ownership := PermissionAuditOwnershipUnrelated
		var releaseMatch permissionAuditReleaseMatch
		if ownershipEnabled {
			if match, ok := resourceToRelease[resourceKey]; ok {
				ownership = PermissionAuditOwnershipDirect
				releaseMatch = match
			} else if releaseNamespaces[namespace] {
				ownership = PermissionAuditOwnershipShared
			}
		}
		saName := extractPermissionAuditWorkloadServiceAccount(item.Object)
		bindingKey := permissionAuditSubjectKey(namespace, saName)
		bindings := []permissionAuditBinding{}
		if req.IncludeRuntimeRBAC {
			bindings = bindingsBySubject[bindingKey]
		}
		runtimeHigh := false
		hasClusterBinding := false
		hasClusterRule := false
		hasRBACWrite := false
		hasSecretWrite := false
		bindingSummaries := make([]map[string]any, 0, len(bindings))
		for _, binding := range bindings {
			if binding.ClusterRoleBinding {
				runtimeHigh = true
				hasClusterBinding = true
			}
			if binding.Role != nil {
				if binding.Role.HasClusterRule {
					runtimeHigh = true
					hasClusterRule = true
				}
				if binding.Role.HasRBACWrite {
					runtimeHigh = true
					hasRBACWrite = true
				}
				if binding.Role.HasSecretWrite {
					hasSecretWrite = true
				}
			}
			bindingSummaries = append(bindingSummaries, binding.toMap())
		}
		privilegeClass := PermissionAuditPrivilegeNamespaceOnly
		if runtimeHigh {
			privilegeClass = PermissionAuditPrivilegeRuntimeHigh
			summaryCounts["high_privilege_workloads"]++
		} else if ownership == PermissionAuditOwnershipDirect {
			summaryCounts["namespace_only_candidate_workloads"]++
		} else if ownership == PermissionAuditOwnershipShared {
			privilegeClass = PermissionAuditPrivilegeSharedCluster
		}
		riskLevel := permissionAuditWorkloadRiskLevel(runtimeHigh, hasRBACWrite, hasSecretWrite, ownership)
		riskCounts[riskLevel]++
		reasonCodes := []string{}
		if hasClusterBinding {
			reasonCodes = append(reasonCodes, "cluster_role_binding")
		}
		if hasClusterRule {
			reasonCodes = append(reasonCodes, "cluster_scoped_rule")
		}
		if hasRBACWrite {
			reasonCodes = append(reasonCodes, "rbac_write")
		}
		if len(reasonCodes) == 0 && ownership == PermissionAuditOwnershipDirect {
			reasonCodes = append(reasonCodes, "namespace_only_candidate")
		}
		detail := map[string]any{
			"service_account_name":                 saName,
			"reason_codes":                         reasonCodes,
			"evidence_chain":                       bindingSummaries,
			"depends_on_shared_cluster_capability": ownership == PermissionAuditOwnershipShared,
		}
		if releaseMatch.ID > 0 {
			detail["platform_app_release"] = map[string]any{
				"id":               releaseMatch.ID,
				"name":             releaseMatch.Name,
				"namespace":        releaseMatch.Namespace,
				"current_revision": releaseMatch.CurrentRevision,
				"template_name":    releaseMatch.TemplateName,
			}
		}
		finding := model.K8sPermissionAuditFinding{
			AuditID:            row.ID,
			FindingType:        PermissionAuditFindingWorkload,
			RiskLevel:          riskLevel,
			OwnershipClass:     ownership,
			PrivilegeClass:     privilegeClass,
			Scope:              "namespace",
			APIVersion:         apiVersion,
			Kind:               kind,
			Namespace:          namespace,
			Name:               name,
			WorkloadKind:       kind,
			WorkloadName:       name,
			ServiceAccountName: saName,
			AppReleaseName:     releaseMatch.Name,
			Summary:            permissionAuditSummaryForWorkload(kind, name, saName, privilegeClass),
			ReasonCodes:        model.JSONStringSlice(reasonCodes),
			DetailJSON:         model.JSONMap(detail),
		}
		if releaseMatch.ID > 0 {
			finding.AppReleaseID = &releaseMatch.ID
			appAggregates[*finding.AppReleaseID] = append(appAggregates[*finding.AppReleaseID], finding)
		}
		findings = append(findings, finding)
	}
	for releaseID, agg := range appAggregates {
		if len(agg) == 0 {
			continue
		}
		first := agg[0]
		maxRisk := "low"
		ownership := PermissionAuditOwnershipUnrelated
		highCount := 0
		for _, item := range agg {
			if permissionAuditRiskRank(item.RiskLevel) > permissionAuditRiskRank(maxRisk) {
				maxRisk = item.RiskLevel
			}
			if item.OwnershipClass == PermissionAuditOwnershipDirect {
				ownership = PermissionAuditOwnershipDirect
			}
			if item.PrivilegeClass == PermissionAuditPrivilegeRuntimeHigh {
				highCount++
			}
		}
		appID := releaseID
		findings = append(findings, model.K8sPermissionAuditFinding{
			AuditID:        row.ID,
			FindingType:    PermissionAuditFindingAppRelease,
			RiskLevel:      maxRisk,
			OwnershipClass: ownership,
			PrivilegeClass: first.PrivilegeClass,
			Namespace:      first.Namespace,
			Name:           first.AppReleaseName,
			AppReleaseID:   &appID,
			AppReleaseName: first.AppReleaseName,
			Summary:        fmt.Sprintf("应用 %s 关联 %d 条资源或工作负载结论，高权限工作负载=%d", first.AppReleaseName, len(agg), highCount),
			ReasonCodes:    model.JSONStringSlice{"app_release_aggregate"},
			DetailJSON: model.JSONMap{
				"resource_count":       len(agg),
				"high_privilege_count": highCount,
			},
		})
	}
	for _, errText := range partialErrors {
		findings = append(findings, model.K8sPermissionAuditFinding{
			AuditID:        row.ID,
			FindingType:    PermissionAuditFindingError,
			RiskLevel:      "medium",
			OwnershipClass: PermissionAuditOwnershipUnrelated,
			Summary:        errText,
			ReasonCodes:    model.JSONStringSlice{"partial_failure"},
			DetailJSON:     model.JSONMap{"message": errText},
		})
	}
	summary := map[string]any{
		"total_resources":                    summaryCounts["total_resources"],
		"cluster_scoped_resources":           summaryCounts["cluster_scoped_resources"],
		"namespaced_resources":               summaryCounts["namespaced_resources"],
		"high_privilege_workloads":           summaryCounts["high_privilege_workloads"],
		"namespace_only_candidate_workloads": summaryCounts["namespace_only_candidate_workloads"],
		"unmapped_resources":                 summaryCounts["unmapped_resources"],
		"ownership":                          ownershipCounts,
		"risk":                               riskCounts,
	}
	stats := map[string]any{
		"risk": riskCounts,
		"blockers": map[string]any{
			"deployment_blockers": blockerCount,
			"shared_capabilities": sharedCount,
			"partial_failures":    len(partialErrors),
		},
	}
	return findings, summary, stats, partialErrors
}

func (s *K8sPermissionAuditService) buildPlatformReleaseIndex(ctx context.Context, clusterID uint64, mapper meta.RESTMapper) (map[string]permissionAuditReleaseMatch, map[string]bool) {
	index := map[string]permissionAuditReleaseMatch{}
	namespaces := map[string]bool{}
	if s.db == nil || clusterID == 0 {
		return index, namespaces
	}
	var releases []model.AppRelease
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND cluster_id = ?", clusterID).Find(&releases).Error; err != nil {
		return index, namespaces
	}
	parser := &AppReleaseService{}
	for _, release := range releases {
		namespaces[strings.TrimSpace(release.Namespace)] = true
		var revision model.AppReleaseRevision
		if err := s.db.WithContext(ctx).Where("release_id = ? AND revision = ?", release.ID, release.CurrentRevision).First(&revision).Error; err != nil {
			continue
		}
		docs, err := parser.parseReleaseManifestDocuments(revision.ComposeManifest, release.Namespace, mapper)
		if err != nil {
			continue
		}
		for _, doc := range docs {
			key := permissionAuditResourceKey(doc.Ref.APIVersion, doc.Ref.Kind, doc.Ref.Namespace, doc.Ref.Name)
			index[key] = permissionAuditReleaseMatch{ID: release.ID, Name: release.Name, Namespace: release.Namespace, TemplateName: release.TemplateName, CurrentRevision: release.CurrentRevision}
		}
	}
	return index, namespaces
}

func analyzePermissionAuditRole(item permissionAuditScannedObject) *permissionAuditRoleAnalysis {
	rules, _, _ := unstructured.NestedSlice(item.Object.Object, "rules")
	analysis := &permissionAuditRoleAnalysis{Scope: ternaryString(item.Namespaced, "namespace", "cluster"), Kind: firstNonEmpty(item.Object.GetKind(), item.Kind), Namespace: item.Object.GetNamespace(), Name: item.Object.GetName(), Rules: []permissionAuditRoleRuleSummary{}}
	for _, raw := range rules {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		apiGroups := permissionAuditToStringSlice(m["apiGroups"])
		resources := permissionAuditToStringSlice(m["resources"])
		verbs := permissionAuditToStringSlice(m["verbs"])
		summary := permissionAuditRoleRuleSummary{APIGroups: apiGroups, Resources: resources, Verbs: verbs}
		for _, res := range resources {
			if res == "*" || permissionAuditClusterScopedResourceNames()[strings.ToLower(strings.TrimSpace(res))] {
				summary.ClusterScopedRule = true
				analysis.HasClusterRule = true
			}
			if strings.EqualFold(strings.TrimSpace(res), "secrets") {
				summary.SecretWrite = true
			}
			if strings.EqualFold(strings.TrimSpace(res), "roles") || strings.EqualFold(strings.TrimSpace(res), "rolebindings") || strings.EqualFold(strings.TrimSpace(res), "clusterroles") || strings.EqualFold(strings.TrimSpace(res), "clusterrolebindings") {
				summary.RBACWrite = true
			}
		}
		for _, verb := range verbs {
			if permissionAuditHighRiskVerbs()[strings.ToLower(strings.TrimSpace(verb))] {
				summary.HighRiskVerb = true
				analysis.HasHighRiskVerb = true
				if summary.RBACWrite {
					analysis.HasRBACWrite = true
				}
				if summary.SecretWrite {
					analysis.HasSecretWrite = true
				}
				if summary.ClusterScopedRule {
					analysis.HasClusterRuleWrite = true
				}
			}
		}
		analysis.Rules = append(analysis.Rules, summary)
	}
	return analysis
}

type permissionAuditBindingSubject struct {
	Kind      string
	Namespace string
	Name      string
}

type permissionAuditBinding struct {
	Subject            permissionAuditBindingSubject
	BindingKind        string
	BindingName        string
	Namespace          string
	RoleRefKind        string
	RoleRefName        string
	Role               *permissionAuditRoleAnalysis
	ClusterRoleBinding bool
}

func (b permissionAuditBinding) SubjectName() string {
	return b.Subject.Name
}

func (b permissionAuditBinding) toMap() map[string]any {
	out := map[string]any{
		"binding_kind":      b.BindingKind,
		"binding_name":      b.BindingName,
		"binding_namespace": b.Namespace,
		"role_ref_kind":     b.RoleRefKind,
		"role_ref_name":     b.RoleRefName,
		"subject":           map[string]any{"kind": b.Subject.Kind, "namespace": b.Subject.Namespace, "name": b.Subject.Name},
	}
	if b.Role != nil {
		out["role"] = map[string]any{
			"scope":              b.Role.Scope,
			"kind":               b.Role.Kind,
			"namespace":          b.Role.Namespace,
			"name":               b.Role.Name,
			"has_cluster_rule":   b.Role.HasClusterRule,
			"has_rbac_write":     b.Role.HasRBACWrite,
			"has_secret_write":   b.Role.HasSecretWrite,
			"has_high_risk_verb": b.Role.HasHighRiskVerb,
		}
		if len(b.Role.Rules) > 0 {
			out["role_rules"] = b.Role.Rules
		}
	}
	return out
}

func analyzePermissionAuditBinding(item permissionAuditScannedObject, roles map[string]*permissionAuditRoleAnalysis) []permissionAuditBinding {
	subjects, _, _ := unstructured.NestedSlice(item.Object.Object, "subjects")
	roleRef, _, _ := unstructured.NestedMap(item.Object.Object, "roleRef")
	roleRefKind := strings.TrimSpace(fmt.Sprintf("%v", roleRef["kind"]))
	roleRefName := strings.TrimSpace(fmt.Sprintf("%v", roleRef["name"]))
	bindingKind := firstNonEmpty(item.Object.GetKind(), item.Kind)
	bindings := make([]permissionAuditBinding, 0, len(subjects))
	for _, raw := range subjects {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		subjectKind := strings.TrimSpace(fmt.Sprintf("%v", m["kind"]))
		if subjectKind != "ServiceAccount" {
			continue
		}
		subjectNamespace := strings.TrimSpace(fmt.Sprintf("%v", m["namespace"]))
		bindingNamespace := item.Object.GetNamespace()
		lookupNamespace := bindingNamespace
		if strings.EqualFold(roleRefKind, "ClusterRole") {
			lookupNamespace = ""
		}
		bindings = append(bindings, permissionAuditBinding{
			Subject:            permissionAuditBindingSubject{Kind: subjectKind, Namespace: subjectNamespace, Name: strings.TrimSpace(fmt.Sprintf("%v", m["name"]))},
			BindingKind:        bindingKind,
			BindingName:        item.Object.GetName(),
			Namespace:          bindingNamespace,
			RoleRefKind:        roleRefKind,
			RoleRefName:        roleRefName,
			Role:               roles[permissionAuditRoleKey(roleRefKind, lookupNamespace, roleRefName)],
			ClusterRoleBinding: bindingKind == "ClusterRoleBinding",
		})
	}
	return bindings
}

func subjectNamespaceForBinding(item permissionAuditScannedObject, bindingNamespace string) string {
	if item.Object.GetKind() == "ClusterRoleBinding" {
		return ""
	}
	return bindingNamespace
}

func extractPermissionAuditWorkloadServiceAccount(obj unstructured.Unstructured) string {
	kind := obj.GetKind()
	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet":
		if v, found, _ := unstructured.NestedString(obj.Object, "spec", "template", "spec", "serviceAccountName"); found && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	case "Job":
		if v, found, _ := unstructured.NestedString(obj.Object, "spec", "template", "spec", "serviceAccountName"); found && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	case "CronJob":
		if v, found, _ := unstructured.NestedString(obj.Object, "spec", "jobTemplate", "spec", "template", "spec", "serviceAccountName"); found && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	case "Pod":
		if v, found, _ := unstructured.NestedString(obj.Object, "spec", "serviceAccountName"); found && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return "default"
}

func extractPermissionAuditIngressClass(obj unstructured.Unstructured) string {
	if v, found, _ := unstructured.NestedString(obj.Object, "spec", "ingressClassName"); found {
		return strings.TrimSpace(v)
	}
	return ""
}

func extractPermissionAuditStorageClass(obj unstructured.Unstructured) string {
	if v, found, _ := unstructured.NestedString(obj.Object, "spec", "storageClassName"); found {
		return strings.TrimSpace(v)
	}
	return ""
}

func isPermissionAuditSharedResource(item permissionAuditScannedObject, releaseNamespaces, serviceAccounts, storageClasses, ingressClasses map[string]bool) bool {
	kind := firstNonEmpty(item.Object.GetKind(), item.Kind)
	name := item.Object.GetName()
	namespace := item.Object.GetNamespace()
	switch kind {
	case "Namespace":
		return releaseNamespaces[name]
	case "ServiceAccount":
		return serviceAccounts[permissionAuditSubjectKey(namespace, name)]
	case "StorageClass":
		return storageClasses[name]
	case "IngressClass":
		return ingressClasses[name]
	default:
		return false
	}
}

func permissionAuditResourceRiskLevel(item permissionAuditScannedObject, ownership string, deploymentBlocker bool) string {
	kind := firstNonEmpty(item.Object.GetKind(), item.Kind)
	if deploymentBlocker && (kind == "ClusterRole" || kind == "ClusterRoleBinding" || kind == "CustomResourceDefinition" || kind == "ValidatingWebhookConfiguration" || kind == "MutatingWebhookConfiguration") {
		return "critical"
	}
	if !item.Namespaced && ownership != PermissionAuditOwnershipUnrelated {
		return "high"
	}
	if kind == "Secret" && ownership != PermissionAuditOwnershipUnrelated {
		return "medium"
	}
	return "low"
}

func permissionAuditWorkloadRiskLevel(runtimeHigh, hasRBACWrite, hasSecretWrite bool, ownership string) string {
	if runtimeHigh && hasRBACWrite {
		return "critical"
	}
	if runtimeHigh {
		return "high"
	}
	if hasSecretWrite || ownership == PermissionAuditOwnershipShared {
		return "medium"
	}
	return "low"
}

func permissionAuditPrivilegeClassForResource(item permissionAuditScannedObject, ownership string, deploymentBlocker bool) string {
	if deploymentBlocker {
		return PermissionAuditPrivilegeClusterScoped
	}
	if ownership == PermissionAuditOwnershipShared && !item.Namespaced {
		return PermissionAuditPrivilegeSharedCluster
	}
	if item.Namespaced {
		return PermissionAuditPrivilegeNamespaceOnly
	}
	return PermissionAuditPrivilegeClusterScoped
}

func permissionAuditReasonCodesForResource(item permissionAuditScannedObject, ownership string, deploymentBlocker bool) []string {
	codes := []string{}
	if !item.Namespaced {
		codes = append(codes, "cluster_scoped_resource")
	}
	if ownership == PermissionAuditOwnershipDirect {
		codes = append(codes, "platform_direct")
	}
	if ownership == PermissionAuditOwnershipShared {
		codes = append(codes, "shared_cluster_capability")
	}
	if deploymentBlocker {
		codes = append(codes, "deployment_blocker")
	}
	if len(codes) == 0 {
		codes = append(codes, "resource_observed")
	}
	return codes
}

func permissionAuditSummaryForResource(kind, ownership, scope string, deploymentBlocker bool) string {
	if deploymentBlocker {
		return fmt.Sprintf("%s 为平台直接使用的 %s 级资源，当前属于部署阻塞项", kind, scope)
	}
	if ownership == PermissionAuditOwnershipShared {
		return fmt.Sprintf("%s 为平台依赖的共享集群能力", kind)
	}
	if ownership == PermissionAuditOwnershipDirect {
		return fmt.Sprintf("%s 已归属到平台资源清单", kind)
	}
	return fmt.Sprintf("%s 已扫描，但未确认归属到平台", kind)
}

func permissionAuditSummaryForWorkload(kind, name, saName, privilegeClass string) string {
	switch privilegeClass {
	case PermissionAuditPrivilegeRuntimeHigh:
		return fmt.Sprintf("%s %s 绑定 ServiceAccount %s，存在运行期高权限依赖", kind, name, saName)
	case PermissionAuditPrivilegeSharedCluster:
		return fmt.Sprintf("%s %s 依赖共享集群能力或共享 ServiceAccount %s", kind, name, saName)
	default:
		return fmt.Sprintf("%s %s 当前可作为命名空间权限部署候选，ServiceAccount=%s", kind, name, saName)
	}
}

func permissionAuditRiskRank(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}

func permissionAuditClusterScopedResourceNames() map[string]bool {
	return map[string]bool{
		"nodes": true, "namespaces": true, "persistentvolumes": true, "storageclasses": true,
		"clusterroles": true, "clusterrolebindings": true, "customresourcedefinitions": true,
		"validatingwebhookconfigurations": true, "mutatingwebhookconfigurations": true,
		"apiservices": true, "priorityclasses": true, "ingressclasses": true,
	}
}

func permissionAuditHighRiskVerbs() map[string]bool {
	return map[string]bool{"create": true, "update": true, "patch": true, "delete": true, "deletecollection": true, "bind": true, "escalate": true, "impersonate": true}
}

func permissionAuditToStringSlice(value any) []string {
	arr, ok := value.([]any)
	if !ok {
		if typed, ok2 := value.([]string); ok2 {
			return typed
		}
		return []string{}
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		text := strings.TrimSpace(fmt.Sprintf("%v", item))
		if text == "" {
			continue
		}
		out = append(out, text)
	}
	sort.Strings(out)
	return out
}

func ternaryString(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}

// ──────────────────────────────────────────────────────────────────────────────
// RBAC 建议生成
// ──────────────────────────────────────────────────────────────────────────────

// RBACRecommendationResult 保存生成的最小权限 RBAC YAML 及元信息。
type RBACRecommendationResult struct {
	YAMLContent      string   `json:"yaml_content"`
	ServiceAccount   string   `json:"service_account"`
	SANamespace      string   `json:"sa_namespace"`
	TargetNamespaces []string `json:"target_namespaces"`
}

type PermissionAuditTaskLogsResult struct {
	TaskID    uint64   `json:"task_id"`
	Offset    int      `json:"offset"`
	Limit     int      `json:"limit"`
	Lines     []string `json:"lines"`
	Status    string   `json:"status"`
	CanCancel bool     `json:"can_cancel"`
}

// RBACMatrixRow 描述权限矩阵中的单行（一个资源/资源组）
type RBACMatrixRow struct {
	// Group: "" (core), "apps", "batch", etc.
	APIGroup string `json:"api_group"`
	// Resources: ["pods"], ["deployments","statefulsets"], etc.
	Resources []string `json:"resources"`
	// Verbs: ["get","list","watch","create","update","patch","delete"]
	Verbs []string `json:"verbs"`
	// Scope: "cluster" | "namespace"
	Scope string `json:"scope"`
	// Label 仅用于 YAML 注释
	Label string `json:"label"`
}

// RBACMatrixRequest 前端提交的权限矩阵
type RBACMatrixRequest struct {
	ServiceAccount   string   `json:"service_account"`
	SANamespace      string   `json:"sa_namespace"`
	TargetNamespaces []string `json:"target_namespaces"`
	// ClusterRows: cluster-scoped rules
	ClusterRows []RBACMatrixRow `json:"cluster_rows"`
	// NamespaceRows: namespace-scoped rules
	NamespaceRows []RBACMatrixRow `json:"namespace_rows"`
}

// DefaultRBACMatrix 返回推荐的默认权限矩阵
func DefaultRBACMatrix(namespaces []string) RBACMatrixRequest {
	if len(namespaces) == 0 {
		namespaces = []string{"default"}
	}
	return RBACMatrixRequest{
		ServiceAccount:   "xingku-platform",
		SANamespace:      "kube-system",
		TargetNamespaces: namespaces,
		ClusterRows: []RBACMatrixRow{
			{APIGroup: "", Resources: []string{"nodes"}, Verbs: []string{"get", "list", "watch", "update", "patch", "delete"}, Scope: "cluster", Label: "节点"},
			{APIGroup: "", Resources: []string{"nodes/status"}, Verbs: []string{"get", "list", "watch"}, Scope: "cluster", Label: "节点状态"},
			{APIGroup: "metrics.k8s.io", Resources: []string{"nodes"}, Verbs: []string{"get", "list"}, Scope: "cluster", Label: "节点指标"},
			{APIGroup: "metrics.k8s.io", Resources: []string{"pods"}, Verbs: []string{"get", "list"}, Scope: "cluster", Label: "Pod 指标"},
			{APIGroup: "", Resources: []string{"namespaces"}, Verbs: []string{"get", "list", "watch", "create", "delete"}, Scope: "cluster", Label: "命名空间"},
			{APIGroup: "", Resources: []string{"persistentvolumes"}, Verbs: []string{"get", "list", "watch", "delete"}, Scope: "cluster", Label: "PersistentVolume"},
			{APIGroup: "storage.k8s.io", Resources: []string{"storageclasses"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "cluster", Label: "StorageClass"},
			{APIGroup: "networking.k8s.io", Resources: []string{"ingressclasses"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "cluster", Label: "IngressClass"},
			{APIGroup: "rbac.authorization.k8s.io", Resources: []string{"clusterroles"}, Verbs: []string{"get", "list", "watch"}, Scope: "cluster", Label: "ClusterRole"},
			{APIGroup: "rbac.authorization.k8s.io", Resources: []string{"clusterrolebindings"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "cluster", Label: "ClusterRoleBinding"},
			{APIGroup: "", Resources: []string{"events"}, Verbs: []string{"get", "list", "watch"}, Scope: "cluster", Label: "事件（集群）"},
		},
		NamespaceRows: []RBACMatrixRow{
			{APIGroup: "apps", Resources: []string{"deployments", "statefulsets", "daemonsets", "replicasets"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "工作负载"},
			{APIGroup: "apps", Resources: []string{"deployments/scale", "statefulsets/scale"}, Verbs: []string{"get", "update", "patch"}, Scope: "namespace", Label: "工作负载扩缩容"},
			{APIGroup: "batch", Resources: []string{"jobs", "cronjobs"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "Job/CronJob"},
			{APIGroup: "autoscaling", Resources: []string{"horizontalpodautoscalers"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "HPA"},
			{APIGroup: "", Resources: []string{"pods"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "Pod"},
			{APIGroup: "", Resources: []string{"pods/log"}, Verbs: []string{"get"}, Scope: "namespace", Label: "Pod 日志"},
			{APIGroup: "", Resources: []string{"pods/exec"}, Verbs: []string{"create"}, Scope: "namespace", Label: "Pod Exec（终端）"},
			{APIGroup: "", Resources: []string{"pods/eviction"}, Verbs: []string{"create"}, Scope: "namespace", Label: "Pod Eviction（drain）"},
			{APIGroup: "", Resources: []string{"services"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "Service"},
			{APIGroup: "", Resources: []string{"endpoints"}, Verbs: []string{"get", "list", "watch"}, Scope: "namespace", Label: "Endpoints"},
			{APIGroup: "networking.k8s.io", Resources: []string{"ingresses"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "Ingress"},
			{APIGroup: "", Resources: []string{"configmaps"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "ConfigMap"},
			{APIGroup: "", Resources: []string{"secrets"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "Secret"},
			{APIGroup: "", Resources: []string{"persistentvolumeclaims"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "PVC"},
			{APIGroup: "", Resources: []string{"serviceaccounts"}, Verbs: []string{"get", "list", "watch"}, Scope: "namespace", Label: "ServiceAccount"},
			{APIGroup: "rbac.authorization.k8s.io", Resources: []string{"roles", "rolebindings"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "Role/RoleBinding"},
			{APIGroup: "", Resources: []string{"events"}, Verbs: []string{"get", "list", "watch"}, Scope: "namespace", Label: "事件"},
		},
	}
}

// BuildRBACFromMatrix 根据权限矩阵生成 RBAC YAML
func BuildRBACFromMatrix(req RBACMatrixRequest) string {
	sa := req.ServiceAccount
	if sa == "" {
		sa = "xingku-platform"
	}
	saNs := req.SANamespace
	if saNs == "" {
		saNs = "kube-system"
	}
	nsList := req.TargetNamespaces
	if len(nsList) == 0 {
		nsList = []string{"default"}
	}

	var b strings.Builder

	// Header
	b.WriteString("# =============================================================\n")
	b.WriteString("# 星枢K8S管理平台 接入账号 自定义权限 RBAC\n")
	b.WriteString("# 由平台权限矩阵自动生成，可直接 kubectl apply -f <此文件>\n")
	b.WriteString("# =============================================================\n\n")

	// 1. ServiceAccount
	fmt.Fprintf(&b, `apiVersion: v1
kind: ServiceAccount
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/component: rbac

`, sa, saNs, sa)

	// 2. ClusterRole: cluster-ops
	clusterRules := make([]RBACMatrixRow, 0)
	for _, row := range req.ClusterRows {
		if len(row.Verbs) > 0 && len(row.Resources) > 0 {
			clusterRules = append(clusterRules, row)
		}
	}
	if len(clusterRules) > 0 {
		fmt.Fprintf(&b, `---
# ClusterRole：集群级资源操作
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: %s-cluster-ops
  labels:
    app.kubernetes.io/name: %s
rules:
`, sa, sa)
		for _, row := range clusterRules {
			verbsJSON := `["` + strings.Join(row.Verbs, `", "`) + `"]`
			resourcesJSON := `["` + strings.Join(row.Resources, `", "`) + `"]`
			fmt.Fprintf(&b, "  - apiGroups: [\"%s\"]\n    resources: %s\n    verbs: %s\n", row.APIGroup, resourcesJSON, verbsJSON)
		}
		// API 发现和健康检查必需：nonResourceURLs 不属于任何资源矩阵行c，始终附加
		b.WriteString("  - nonResourceURLs: [\"/api\", \"/api/*\", \"/apis\", \"/apis/*\", \"/version\", \"/healthz\"]\n    verbs: [\"get\"]\n")
		b.WriteString("\n")
		fmt.Fprintf(&b, `---
# ClusterRoleBinding：集群级权限绑定
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: %s-cluster-ops
  labels:
    app.kubernetes.io/name: %s
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: %s-cluster-ops
subjects:
  - kind: ServiceAccount
    name: %s
    namespace: %s

`, sa, sa, sa, sa, saNs)
	}

	// 3. ClusterRole: namespace-read (will be applied per-namespace via RoleBinding)
	nsRules := make([]RBACMatrixRow, 0)
	for _, row := range req.NamespaceRows {
		if len(row.Verbs) > 0 && len(row.Resources) > 0 {
			nsRules = append(nsRules, row)
		}
	}
	if len(nsRules) > 0 {
		fmt.Fprintf(&b, `---
# ClusterRole：命名空间资源权限（通过 RoleBinding 授予各目标命名空间）
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: %s-ns-ops
  labels:
    app.kubernetes.io/name: %s
rules:
`, sa, sa)
		for _, row := range nsRules {
			verbsJSON := `["` + strings.Join(row.Verbs, `", "`) + `"]`
			resourcesJSON := `["` + strings.Join(row.Resources, `", "`) + `"]`
			fmt.Fprintf(&b, "  - apiGroups: [\"%s\"]\n    resources: %s\n    verbs: %s\n", row.APIGroup, resourcesJSON, verbsJSON)
		}
		b.WriteString("\n")

		// Per-namespace RoleBindings
		for i, ns := range nsList {
			fmt.Fprintf(&b, `---
# RoleBinding %d：%s 命名空间
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: %s-ns-ops
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: %s-ns-ops
subjects:
  - kind: ServiceAccount
    name: %s
    namespace: %s

`, i+1, ns, sa, ns, sa, sa, sa, saNs)
		}
	}

	// 4. Token Secret
	fmt.Fprintf(&b, `---
# Token Secret（K8s >= 1.24 需要手动创建）
apiVersion: v1
kind: Secret
metadata:
  name: %s-token
  namespace: %s
  annotations:
    kubernetes.io/service-account.name: %s
type: kubernetes.io/service-account-token
`, sa, saNs, sa)

	return b.String()
}

// GenerateRBACRecommendation 根据目标命名空间生成平台最小权限 RBAC YAML。
// 不需要 K8s API 调用，纯模板生成。
func (s *K8sPermissionAuditService) GenerateRBACRecommendation(ctx context.Context, clusterID uint64, targetNamespaces []string) RBACRecommendationResult {
	namespaces := make([]string, 0, len(targetNamespaces))
	for _, ns := range targetNamespaces {
		if n := strings.TrimSpace(ns); n != "" {
			namespaces = append(namespaces, n)
		}
	}
	allowlist := []string{}
	if clusterID > 0 {
		allowlist = s.loadLatestAuditAllowlist(ctx, clusterID)
		if len(namespaces) == 0 {
			namespaces = s.loadLatestAuditNamespaces(ctx, clusterID)
		}
	}
	if len(namespaces) == 0 {
		namespaces = []string{"default"}
	}
	matrix := DefaultRBACMatrix(namespaces)
	if len(allowlist) > 0 {
		matrix = filterRBACMatrixByAllowlist(matrix, allowlist)
	}
	yaml := BuildRBACFromMatrix(matrix)
	return RBACRecommendationResult{
		YAMLContent:      yaml,
		ServiceAccount:   matrix.ServiceAccount,
		SANamespace:      matrix.SANamespace,
		TargetNamespaces: matrix.TargetNamespaces,
	}
}

func buildMinimumRBACYAML(namespaces []string) string {
	matrix := DefaultRBACMatrix(namespaces)
	return BuildRBACFromMatrix(matrix)
}

func (s *K8sPermissionAuditService) loadLatestAuditNamespaces(ctx context.Context, clusterID uint64) []string {
	row := s.loadLatestAuditRow(ctx, clusterID)
	if row == nil {
		return nil
	}
	return permissionAuditRequestNamespaces(row.RequestJSON)
}

func (s *K8sPermissionAuditService) loadLatestAuditAllowlist(ctx context.Context, clusterID uint64) []string {
	row := s.loadLatestAuditRow(ctx, clusterID)
	if row == nil {
		return nil
	}
	return permissionAuditRequestAllowlist(row.RequestJSON)
}

func (s *K8sPermissionAuditService) loadLatestAuditRow(ctx context.Context, clusterID uint64) *model.K8sPermissionAudit {
	if s == nil || s.db == nil || clusterID == 0 {
		return nil
	}
	var row model.K8sPermissionAudit
	err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND cluster_id = ? AND status IN ?", clusterID, []string{PermissionAuditStatusSuccess, PermissionAuditStatusIncomplete}).
		Order("created_at desc").
		First(&row).Error
	if err != nil {
		return nil
	}
	return &row
}

func permissionAuditRequestNamespaces(request model.JSONMap) []string {
	items, _ := request["namespaces"].([]any)
	if len(items) == 0 {
		if direct, ok := request["namespaces"].([]string); ok {
			return direct
		}
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		value := strings.TrimSpace(fmt.Sprintf("%v", item))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func permissionAuditRequestAllowlist(request model.JSONMap) []string {
	items, _ := request["resource_allowlist"].([]any)
	if len(items) == 0 {
		if direct, ok := request["resource_allowlist"].([]string); ok {
			return direct
		}
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		value := normalizePermissionAuditAllowlistItem(fmt.Sprintf("%v", item))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func filterRBACMatrixByAllowlist(matrix RBACMatrixRequest, allowlist []string) RBACMatrixRequest {
	allowed := map[string]bool{}
	for _, item := range allowlist {
		if value := normalizePermissionAuditAllowlistItem(item); value != "" {
			allowed[value] = true
		}
	}
	if len(allowed) == 0 {
		return matrix
	}
	clusterRows := make([]RBACMatrixRow, 0, len(matrix.ClusterRows))
	for _, row := range matrix.ClusterRows {
		if rbACRowMatchesAllowlist(row, allowed) {
			clusterRows = append(clusterRows, row)
		}
	}
	namespaceRows := make([]RBACMatrixRow, 0, len(matrix.NamespaceRows))
	for _, row := range matrix.NamespaceRows {
		if rbACRowMatchesAllowlist(row, allowed) {
			namespaceRows = append(namespaceRows, row)
		}
	}
	matrix.ClusterRows = clusterRows
	matrix.NamespaceRows = namespaceRows
	return matrix
}

func rbACRowMatchesAllowlist(row RBACMatrixRow, allowed map[string]bool) bool {
	for _, resource := range row.Resources {
		base := strings.SplitN(resource, "/", 2)[0]
		if allowed[normalizePermissionAuditAllowlistItem(base)] {
			return true
		}
	}
	return false
}

func (s *K8sPermissionAuditService) persistAuditResult(ctx context.Context, auditID uint64, status string, summary, stats map[string]any, findings []model.K8sPermissionAuditFinding, errors []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("audit_id = ?", auditID).Delete(&model.K8sPermissionAuditFinding{}).Error; err != nil {
			return err
		}
		for i := range findings {
			findings[i].AuditID = auditID
		}
		if len(findings) > 0 {
			if err := tx.CreateInBatches(&findings, 200).Error; err != nil {
				return err
			}
		}
		updates := map[string]any{
			"status":       status,
			"summary_json": model.JSONMap(summary),
			"stats_json":   model.JSONMap(stats),
		}
		if len(errors) > 0 {
			updates["error_json"] = model.JSONMap{"partial_failures": errors}
		}
		return tx.Model(&model.K8sPermissionAudit{}).Where("id = ?", auditID).Updates(updates).Error
	})
}
