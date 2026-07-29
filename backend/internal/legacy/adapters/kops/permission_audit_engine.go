package kops

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"

	fleetapp "k8s-platform-backend/internal/fleet/application"
	fleetdomain "k8s-platform-backend/internal/fleet/domain"
	kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
	model "k8s-platform-backend/internal/kops/domain"
	platformapp "k8s-platform-backend/internal/platform/application"
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

	PermissionAuditFindingResource = "resource"
	PermissionAuditFindingWorkload = "workload"
	PermissionAuditFindingError    = "error"

	PermissionAuditOwnershipDirect    = kopsapp.PermissionAuditOwnershipDirect
	PermissionAuditOwnershipShared    = kopsapp.PermissionAuditOwnershipShared
	PermissionAuditOwnershipUnrelated = kopsapp.PermissionAuditOwnershipUnrelated

	PermissionAuditPrivilegeClusterScoped = kopsapp.PermissionAuditPrivilegeClusterScoped
	PermissionAuditPrivilegeRuntimeHigh   = kopsapp.PermissionAuditPrivilegeRuntimeHigh
	PermissionAuditPrivilegeNamespaceOnly = kopsapp.PermissionAuditPrivilegeNamespaceOnly
	PermissionAuditPrivilegeSharedCluster = kopsapp.PermissionAuditPrivilegeSharedCluster
)

type PermissionAuditCreateRequest struct {
	Mode                      string   `json:"mode"`
	IncludeRuntimeRBAC        bool     `json:"include_runtime_rbac"`
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

// PermissionAuditPage is the scanner read-model page retained at the Kops
// adapter boundary while HTTP keeps its existing response shape.
type PermissionAuditPage[T any] struct {
	List     []T `json:"list"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
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

// PermissionAuditCredentials isolates the short-lived ad-hoc credential
// lifecycle from scan coordination. Its concrete encrypted cache stays in the
// retained service infrastructure package and is injected by the router.
type PermissionAuditCredentials interface {
	Put(context.Context, uint64, string, time.Duration) error
	Get(context.Context, uint64) (value string, found bool, err error)
	Delete(context.Context, uint64)
}

// PermissionAuditEngine is the Kops runtime adapter that coordinates the
// asynchronous scan, persistence, and task lifecycle. Business policies are
// delegated to the Kops application package.
type PermissionAuditEngine struct {
	db            *gorm.DB
	taskStore     *platformapp.TaskStore
	clusterReg    *fleetapp.Registry
	transport     *kopsclient.PermissionAuditTransport
	credentialTTL time.Duration
	creds         PermissionAuditCredentials
}

type permissionAuditClients struct {
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

type permissionAuditScannedObject struct {
	GVR        schema.GroupVersionResource
	Kind       string
	Namespaced bool
	Object     unstructured.Unstructured
}

func permissionAuditResourceMeta(item permissionAuditScannedObject) kopsapp.PermissionAuditResourceMeta {
	return kopsapp.PermissionAuditResourceMeta{
		Kind:       firstNonEmpty(item.Object.GetKind(), item.Kind),
		Name:       item.Object.GetName(),
		Namespace:  item.Object.GetNamespace(),
		Namespaced: item.Namespaced,
	}
}

func NewPermissionAuditEngine(db *gorm.DB, taskStore *platformapp.TaskStore, clusterReg *fleetapp.Registry, transport *kopsclient.PermissionAuditTransport, creds PermissionAuditCredentials, credentialTTL time.Duration) *PermissionAuditEngine {
	if credentialTTL <= 0 {
		credentialTTL = 2 * time.Hour
	}
	return &PermissionAuditEngine{
		db:            db,
		taskStore:     taskStore,
		clusterReg:    clusterReg,
		transport:     transport,
		credentialTTL: credentialTTL,
		creds:         creds,
	}
}

func (s *PermissionAuditEngine) CreateManagedAudit(ctx context.Context, clusterID uint64, req PermissionAuditCreateRequest, createdBy uint64) (PermissionAuditCreateResult, error) {
	if s.db == nil || s.taskStore == nil || s.clusterReg == nil || s.transport == nil {
		return PermissionAuditCreateResult{}, fmt.Errorf("permission audit service dependencies are incomplete")
	}
	if clusterID == 0 {
		return PermissionAuditCreateResult{}, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "集群ID无效")
	}
	if err := s.normalizeCreateRequest(&req); err != nil {
		return PermissionAuditCreateResult{}, err
	}
	cluster, err := s.clusterReg.Get(ctx, clusterID)
	if err != nil {
		return PermissionAuditCreateResult{}, permissionAuditClusterError(err)
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

func (s *PermissionAuditEngine) CreateAdhocAudit(ctx context.Context, req PermissionAuditAdhocCreateRequest, createdBy uint64) (PermissionAuditCreateResult, error) {
	if s.db == nil || s.taskStore == nil || s.transport == nil || s.creds == nil {
		return PermissionAuditCreateResult{}, fmt.Errorf("permission audit service dependencies are incomplete")
	}
	if err := s.normalizeCreateRequest(&req.PermissionAuditCreateRequest); err != nil {
		return PermissionAuditCreateResult{}, err
	}
	kubeconfig := strings.TrimSpace(req.Kubeconfig)
	if kubeconfig == "" {
		return PermissionAuditCreateResult{}, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "管理员凭据不能为空")
	}
	clusterName, kubeconfig, err := s.validateAdhocKubeconfig(ctx, kubeconfig)
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
		return PermissionAuditCreateResult{}, permissionAuditCredentialError(err)
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

func (s *PermissionAuditEngine) ListAudits(ctx context.Context, req ListPermissionAuditsRequest) (PermissionAuditPage[PermissionAuditListItem], error) {
	if s.db == nil {
		return PermissionAuditPage[PermissionAuditListItem]{}, fmt.Errorf("db is required")
	}
	page, pageSize := normalizePermissionAuditPage(req.Page, req.PageSize)
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
		return PermissionAuditPage[PermissionAuditListItem]{}, err
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
		return PermissionAuditPage[PermissionAuditListItem]{}, err
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
	return PermissionAuditPage[PermissionAuditListItem]{List: out, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *PermissionAuditEngine) GetAudit(ctx context.Context, auditID uint64) (*PermissionAuditDetail, error) {
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

func (s *PermissionAuditEngine) ListFindings(ctx context.Context, auditID uint64, req ListPermissionAuditFindingsRequest) (PermissionAuditPage[PermissionAuditFindingItem], error) {
	if s.db == nil {
		return PermissionAuditPage[PermissionAuditFindingItem]{}, fmt.Errorf("db is required")
	}
	if auditID == 0 {
		return PermissionAuditPage[PermissionAuditFindingItem]{}, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "分析任务ID无效")
	}
	page, pageSize := normalizePermissionAuditPage(req.Page, req.PageSize)
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
		q = q.Where("name LIKE ? OR workload_name LIKE ? OR service_account_name LIKE ? OR summary LIKE ?", like, like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PermissionAuditPage[PermissionAuditFindingItem]{}, err
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
		return PermissionAuditPage[PermissionAuditFindingItem]{}, err
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
	return PermissionAuditPage[PermissionAuditFindingItem]{List: out, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *PermissionAuditEngine) GetLatestClusterAudit(ctx context.Context, clusterID uint64) (*PermissionAuditDetail, error) {
	if s.db == nil {
		return nil, fmt.Errorf("db is required")
	}
	if clusterID == 0 {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "集群ID无效")
	}
	var row model.K8sPermissionAudit
	err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND cluster_id = ? AND source_type = ? AND status IN ?", clusterID, PermissionAuditSourceManaged, []string{PermissionAuditStatusSuccess, PermissionAuditStatusIncomplete}).
		Order("id DESC").
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, kopsapp.ErrNotFound
		}
		return nil, err
	}
	return s.GetAudit(ctx, row.ID)
}

func (s *PermissionAuditEngine) CancelAudit(ctx context.Context, auditID uint64) error {
	if s == nil || s.taskStore == nil {
		return fmt.Errorf("task store is required")
	}
	row, err := s.getAuditRow(ctx, auditID)
	if err != nil {
		return err
	}
	if row.TaskID == nil || *row.TaskID == 0 {
		return kopsapp.ErrWithMessage(kopsapp.ErrConflict, "当前分析没有可取消任务")
	}
	if err := platformapp.NewTaskService(s.taskStore).Cancel(int64(*row.TaskID)); err != nil {
		switch err {
		case platformapp.ErrTaskNotFound:
			return kopsapp.ErrNotFound
		case platformapp.ErrTaskCannotCancel:
			return kopsapp.ErrConflict
		default:
			return err
		}
	}
	return nil
}

func (s *PermissionAuditEngine) AuditTaskLogs(ctx context.Context, auditID uint64, offset, limit int) (PermissionAuditTaskLogsResult, error) {
	if s == nil || s.taskStore == nil {
		return PermissionAuditTaskLogsResult{}, fmt.Errorf("task store is required")
	}
	row, err := s.getAuditRow(ctx, auditID)
	if err != nil {
		return PermissionAuditTaskLogsResult{}, err
	}
	if row.TaskID == nil || *row.TaskID == 0 {
		return PermissionAuditTaskLogsResult{}, kopsapp.ErrNotFound
	}
	if limit <= 0 {
		limit = 200
	}
	task, ok := s.taskStore.Get(int64(*row.TaskID))
	if !ok || task == nil {
		return PermissionAuditTaskLogsResult{}, kopsapp.ErrNotFound
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

func (s *PermissionAuditEngine) CompareAudits(ctx context.Context, auditID, baselineAuditID uint64) (PermissionAuditCompareResult, error) {
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

func (s *PermissionAuditEngine) resolveCompareBaseline(ctx context.Context, current *model.K8sPermissionAudit, baselineAuditID uint64) (*model.K8sPermissionAudit, error) {
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
			return nil, kopsapp.ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (s *PermissionAuditEngine) loadAuditFindings(ctx context.Context, auditID uint64) ([]PermissionAuditFindingItem, error) {
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

func (s *PermissionAuditEngine) loadFindingRows(ctx context.Context, auditID uint64) ([]model.K8sPermissionAuditFinding, error) {
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

func (s *PermissionAuditEngine) normalizeCreateRequest(req *PermissionAuditCreateRequest) error {
	if req == nil {
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "请求不能为空")
	}
	req.Mode = firstNonEmpty(strings.TrimSpace(req.Mode), "full")
	if req.Mode != "full" {
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "当前仅支持 full 分析模式")
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
	column, _ := kopsapp.NormalizePermissionAuditListSort(col, "desc")
	return column
}

func sanitizePermissionAuditFindingOrderColumn(col string) string {
	column, _ := kopsapp.NormalizePermissionAuditFindingSort(col, "desc")
	return column
}

func (s *PermissionAuditEngine) createAuditTask(ctx context.Context, auditID uint64, sourceType, displayName string, createdBy uint64) (*platformapp.Task, error) {
	title := fmt.Sprintf("K8s 权限分析：%s", firstNonEmpty(displayName, fmt.Sprintf("audit-%d", auditID)))
	task := &platformapp.Task{
		Type:      "k8s_permission_audit",
		Status:    platformapp.TaskPending,
		Title:     &title,
		CreatedBy: int64(createdBy),
		Meta: map[string]any{
			"audit_id":    auditID,
			"source_type": sourceType,
		},
		Steps: []platformapp.TaskStep{
			{Key: "prepare", Title: "准备分析", Status: platformapp.StepPending},
			{Key: "scan", Title: "扫描资源", Status: platformapp.StepPending},
			{Key: "analyze", Title: "分析权限", Status: platformapp.StepPending},
			{Key: "persist", Title: "保存结果", Status: platformapp.StepPending},
		},
	}
	if err := s.taskStore.Put(task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *PermissionAuditEngine) getAuditRow(ctx context.Context, auditID uint64) (*model.K8sPermissionAudit, error) {
	if auditID == 0 {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "分析任务ID无效")
	}
	var row model.K8sPermissionAudit
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", auditID).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, kopsapp.ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (s *PermissionAuditEngine) validateAdhocKubeconfig(ctx context.Context, kubeconfig string) (string, string, error) {
	if s.transport == nil {
		return "", "", fmt.Errorf("permission audit transport is required")
	}
	validated, err := s.transport.ProbeAdhoc(ctx, kubeconfig)
	if err != nil {
		return "", "", legacyPermissionAuditAdhocTransportError(err)
	}
	return validated.ClusterName, validated.Content, nil
}

func legacyPermissionAuditAdhocTransportError(err error) error {
	switch {
	case errors.Is(err, kopsclient.ErrKubeconfigEmpty), errors.Is(err, kopsclient.ErrKubeconfigTooLarge), errors.Is(err, kopsclient.ErrInvalidKubeconfig):
		return permissionAuditKubeconfigError(err)
	default:
		return normalizePermissionAuditK8sError(err)
	}
}

func (s *PermissionAuditEngine) runAudit(auditID uint64) {
	ctx := context.Background()
	row, err := s.getAuditRow(ctx, auditID)
	if err != nil {
		return
	}
	var task *platformapp.Task
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
	updateStep := func(index int, status platformapp.TaskStepStatus, message string) {
		if task == nil || index < 0 || index >= len(task.Steps) {
			return
		}
		now := time.Now().UTC()
		task.Steps[index].Status = status
		if status == platformapp.StepRunning {
			task.Steps[index].StartedAt = &now
		}
		if status == platformapp.StepSuccess || status == platformapp.StepFailed {
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
		task.Status = platformapp.TaskRunning
		startMsg := "K8s 权限分析开始"
		task.Message = &startMsg
		_ = task.Update()
	}
	setAuditStatus(PermissionAuditStatusRunning, nil)
	updateStep(0, platformapp.StepRunning, "准备 K8s 客户端与分析上下文")
	clients, auditReq, err := s.loadClientsForAudit(runCtx, row)
	if err != nil {
		if errors.Is(err, context.Canceled) || runCtx.Err() != nil {
			s.finishAuditCanceled(row, task)
			return
		}
		s.finishAuditWithError(row, task, PermissionAuditStatusFailed, err)
		updateStep(0, platformapp.StepFailed, fmt.Sprintf("准备失败：%v", err))
		return
	}
	updateStep(0, platformapp.StepSuccess, "准备完成")
	updateStep(1, platformapp.StepRunning, "开始扫描集群资源")
	available, discoveryErrors := s.discoverAvailableResources(runCtx, clients.discovery)
	objects, scanErrors, err := s.scanTargetResources(runCtx, clients.dynamic, available, auditReq, task)
	if err != nil {
		if errors.Is(err, context.Canceled) || runCtx.Err() != nil {
			s.finishAuditCanceled(row, task)
			return
		}
		s.finishAuditWithError(row, task, PermissionAuditStatusFailed, err)
		updateStep(1, platformapp.StepFailed, fmt.Sprintf("扫描失败：%v", err))
		return
	}
	if runCtx.Err() != nil {
		s.finishAuditCanceled(row, task)
		return
	}
	updateStep(1, platformapp.StepSuccess, fmt.Sprintf("扫描完成，资源数：%d", len(objects)))
	updateStep(2, platformapp.StepRunning, "开始分析权限与资源归属")
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
	updateStep(2, platformapp.StepSuccess, fmt.Sprintf("分析完成，结论数：%d", len(findings)))
	updateStep(3, platformapp.StepRunning, "保存分析结果")
	if err := s.persistAuditResult(runCtx, row.ID, status, summary, stats, findings, allErrors); err != nil {
		if errors.Is(err, context.Canceled) || runCtx.Err() != nil {
			s.finishAuditCanceled(row, task)
			return
		}
		s.finishAuditWithError(row, task, PermissionAuditStatusFailed, err)
		updateStep(3, platformapp.StepFailed, fmt.Sprintf("保存失败：%v", err))
		return
	}
	updateStep(3, platformapp.StepSuccess, "结果已保存")
	if task != nil {
		msg := "K8s 权限分析完成"
		if status == PermissionAuditStatusIncomplete {
			msg = "K8s 权限分析完成，但存在部分资源扫描失败"
			task.Status = platformapp.TaskSuccess
		} else {
			task.Status = platformapp.TaskStatus(status)
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

func (s *PermissionAuditEngine) finishAuditWithError(row *model.K8sPermissionAudit, task *platformapp.Task, status string, err error) {
	msg := firstNonEmpty(permissionAuditErrorMessage(err), "权限分析执行失败")
	_ = s.db.WithContext(context.Background()).Model(&model.K8sPermissionAudit{}).Where("id = ?", row.ID).Updates(map[string]any{
		"status":     status,
		"error_json": model.JSONMap{"message": msg},
	}).Error
	if task != nil {
		task.Status = platformapp.TaskFailed
		task.Message = &msg
		_ = task.Update()
		task.AppendLog("[Error] " + msg)
	}
	if row.SourceType == PermissionAuditSourceAdhoc {
		s.creds.Delete(context.Background(), row.ID)
	}
}

func (s *PermissionAuditEngine) finishAuditCanceled(row *model.K8sPermissionAudit, task *platformapp.Task) {
	msg := "K8s 权限分析已取消"
	_ = s.db.WithContext(context.Background()).Model(&model.K8sPermissionAudit{}).Where("id = ?", row.ID).Updates(map[string]any{
		"status":     PermissionAuditStatusCanceled,
		"error_json": model.JSONMap{"message": msg},
	}).Error
	if task != nil {
		task.Status = platformapp.TaskCanceled
		task.Message = &msg
		_ = task.Update()
		task.AppendLog(msg)
	}
	if row.SourceType == PermissionAuditSourceAdhoc {
		s.creds.Delete(context.Background(), row.ID)
	}
}

func permissionAuditErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if msg, ok := permissionAuditUserMessage(err); ok {
		return msg
	}
	return strings.TrimSpace(err.Error())
}

func (s *PermissionAuditEngine) loadClientsForAudit(ctx context.Context, row *model.K8sPermissionAudit) (*permissionAuditClients, PermissionAuditCreateRequest, error) {
	request := PermissionAuditCreateRequest{Mode: "full", IncludeRuntimeRBAC: true, IncludeOwnershipDetection: true}
	if row.RequestJSON != nil {
		b, _ := json.Marshal(row.RequestJSON)
		_ = json.Unmarshal(b, &request)
	}
	var clients *kopsclient.PermissionAuditClients
	var err error
	switch row.SourceType {
	case PermissionAuditSourceManaged:
		if row.ClusterID == nil || *row.ClusterID == 0 {
			return nil, request, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "缺少集群标识")
		}
		clients, err = s.transport.ManagedClients(ctx, *row.ClusterID)
	case PermissionAuditSourceAdhoc:
		kubeconfig, found, getErr := s.creds.Get(ctx, row.ID)
		if getErr != nil {
			return nil, request, permissionAuditCredentialError(getErr)
		}
		if !found {
			return nil, request, kopsapp.ErrWithMessage(kopsapp.ErrNotFound, "临时管理员凭据已失效，请重新发起分析")
		}
		clients, err = s.transport.AdhocClients(kubeconfig)
	default:
		return nil, request, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "未知的分析来源")
	}
	if err != nil {
		return nil, request, legacyPermissionAuditClientTransportError(err)
	}
	dyn := clients.Dynamic
	disc := clients.Discovery
	groupResources, err := restmapper.GetAPIGroupResources(disc)
	if err != nil {
		return nil, request, normalizePermissionAuditK8sError(err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	return &permissionAuditClients{dynamic: dyn, discovery: disc, mapper: mapper}, request, nil
}

func legacyPermissionAuditClientTransportError(err error) error {
	var lookup *kopsclient.KubeconfigLookupError
	if errors.As(err, &lookup) {
		return permissionAuditClusterError(lookup.Unwrap())
	}
	return normalizePermissionAuditK8sError(err)
}

func (s *PermissionAuditEngine) discoverAvailableResources(ctx context.Context, disc discovery.DiscoveryInterface) (map[string]permissionAuditAvailableResource, []string) {
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
			resources[kopsapp.PermissionAuditGVRKey(gvr.Group, gvr.Version, gvr.Resource)] = permissionAuditAvailableResource{GVR: gvr, Kind: apiRes.Kind, Namespaced: apiRes.Namespaced, Verbs: verbs}
		}
	}
	return resources, partialErrors
}

func (s *PermissionAuditEngine) scanTargetResources(ctx context.Context, dyn dynamic.Interface, available map[string]permissionAuditAvailableResource, req PermissionAuditCreateRequest, task *platformapp.Task) ([]permissionAuditScannedObject, []string, error) {
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
	opts := metav1.ListOptions{Limit: permissionAuditListPageLimit}
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
				return normalizePermissionAuditK8sError(err)
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
	specs := kopsapp.PermissionAuditScanTargets(allowlist)
	targets := make([]permissionAuditResourceTarget, 0, len(specs))
	for _, spec := range specs {
		targets = append(targets, permissionAuditResourceTarget{
			GVR: schema.GroupVersionResource{Group: spec.Group, Version: spec.Version, Resource: spec.Resource},
		})
	}
	return targets
}

func normalizePermissionAuditAllowlistItem(value string) string {
	return kopsapp.NormalizePermissionAuditAllowlistItem(value)
}

func resolvePermissionAuditTarget(available map[string]permissionAuditAvailableResource, target permissionAuditResourceTarget) (permissionAuditAvailableResource, bool) {
	for _, candidate := range candidateGVRs(target.GVR) {
		if res, ok := available[kopsapp.PermissionAuditGVRKey(candidate.Group, candidate.Version, candidate.Resource)]; ok {
			return res, true
		}
	}
	return permissionAuditAvailableResource{}, false
}

func (s *PermissionAuditEngine) analyzeScannedResources(ctx context.Context, row *model.K8sPermissionAudit, req PermissionAuditCreateRequest, mapper meta.RESTMapper, objects []permissionAuditScannedObject) ([]model.K8sPermissionAuditFinding, map[string]any, map[string]any, []string) {
	targetNamespaces := permissionAuditNamespaceSet(req.Namespaces)
	ownershipEnabled := req.IncludeOwnershipDetection
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
				roles[kopsapp.PermissionAuditRoleKey(kind, item.Object.GetNamespace(), item.Object.GetName())] = analysis
			}
		case "RoleBinding", "ClusterRoleBinding":
			if req.IncludeRuntimeRBAC {
				for _, binding := range analyzePermissionAuditBinding(item, roles) {
					key := kopsapp.PermissionAuditSubjectKey(subjectNamespaceForBinding(item, binding.Namespace), binding.SubjectName())
					bindingsBySubject[key] = append(bindingsBySubject[key], binding)
				}
			}
		case "Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob":
			workloadObjects = append(workloadObjects, item)
			sa := extractPermissionAuditWorkloadServiceAccount(item.Object)
			serviceAccountRefs[kopsapp.PermissionAuditSubjectKey(item.Object.GetNamespace(), sa)] = true
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
	for _, item := range resourceObjects {
		kind := firstNonEmpty(item.Object.GetKind(), item.Kind)
		apiVersion := item.Object.GetAPIVersion()
		namespace := item.Object.GetNamespace()
		name := item.Object.GetName()
		ownership := PermissionAuditOwnershipUnrelated
		if ownershipEnabled {
			ownership = kopsapp.PermissionAuditResourceOwnership(permissionAuditResourceMeta(item), targetNamespaces, serviceAccountRefs, storageClassRefs, ingressClassRefs)
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
		if ownership != PermissionAuditOwnershipUnrelated {
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
		riskLevel := kopsapp.PermissionAuditResourceRiskLevel(permissionAuditResourceMeta(item), ownership, deploymentBlocker)
		riskCounts[riskLevel]++
		reasonCodes := kopsapp.PermissionAuditResourceReasonCodes(permissionAuditResourceMeta(item), ownership, deploymentBlocker)
		detail := map[string]any{
			"api_version": item.Object.GetAPIVersion(),
			"scope":       scope,
		}
		finding := model.K8sPermissionAuditFinding{
			AuditID:                          row.ID,
			FindingType:                      PermissionAuditFindingResource,
			RiskLevel:                        riskLevel,
			OwnershipClass:                   ownership,
			PrivilegeClass:                   kopsapp.PermissionAuditResourcePrivilegeClass(permissionAuditResourceMeta(item), ownership, deploymentBlocker),
			Scope:                            scope,
			DeploymentBlocker:                deploymentBlocker,
			DependsOnSharedClusterCapability: dependsOnShared,
			APIVersion:                       apiVersion,
			Kind:                             kind,
			Namespace:                        namespace,
			Name:                             name,
			Summary:                          kopsapp.PermissionAuditResourceSummary(kind, ownership, scope, deploymentBlocker),
			ReasonCodes:                      model.JSONStringSlice(reasonCodes),
			DetailJSON:                       model.JSONMap(detail),
		}
		findings = append(findings, finding)
	}
	for _, item := range workloadObjects {
		kind := firstNonEmpty(item.Object.GetKind(), item.Kind)
		apiVersion := item.Object.GetAPIVersion()
		namespace := item.Object.GetNamespace()
		name := item.Object.GetName()
		ownership := PermissionAuditOwnershipUnrelated
		if ownershipEnabled {
			ownership = kopsapp.PermissionAuditWorkloadOwnership(namespace, targetNamespaces)
		}
		saName := extractPermissionAuditWorkloadServiceAccount(item.Object)
		bindingKey := kopsapp.PermissionAuditSubjectKey(namespace, saName)
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
		riskLevel := kopsapp.PermissionAuditWorkloadRiskLevel(runtimeHigh, hasRBACWrite, hasSecretWrite, ownership)
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
			Summary:            kopsapp.PermissionAuditWorkloadSummary(kind, name, saName, privilegeClass),
			ReasonCodes:        model.JSONStringSlice(reasonCodes),
			DetailJSON:         model.JSONMap(detail),
		}
		findings = append(findings, finding)
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

func analyzePermissionAuditRole(item permissionAuditScannedObject) *permissionAuditRoleAnalysis {
	rules, _, _ := unstructured.NestedSlice(item.Object.Object, "rules")
	analysis := &permissionAuditRoleAnalysis{Scope: ternaryString(item.Namespaced, "namespace", "cluster"), Kind: firstNonEmpty(item.Object.GetKind(), item.Kind), Namespace: item.Object.GetNamespace(), Name: item.Object.GetName(), Rules: []permissionAuditRoleRuleSummary{}}
	for _, raw := range rules {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		apiGroups := kopsapp.PermissionAuditStringSlice(m["apiGroups"])
		resources := kopsapp.PermissionAuditStringSlice(m["resources"])
		verbs := kopsapp.PermissionAuditStringSlice(m["verbs"])
		summary := permissionAuditRoleRuleSummary{APIGroups: apiGroups, Resources: resources, Verbs: verbs}
		for _, res := range resources {
			if res == "*" || kopsapp.PermissionAuditIsClusterScopedResource(res) {
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
			if kopsapp.PermissionAuditIsHighRiskVerb(verb) {
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
			Role:               roles[kopsapp.PermissionAuditRoleKey(roleRefKind, lookupNamespace, roleRefName)],
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

func permissionAuditNamespaceSet(namespaces []string) map[string]bool {
	return kopsapp.NewPermissionAuditNamespaceSet(namespaces)
}

func ternaryString(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}

type PermissionAuditTaskLogsResult struct {
	TaskID    uint64   `json:"task_id"`
	Offset    int      `json:"offset"`
	Limit     int      `json:"limit"`
	Lines     []string `json:"lines"`
	Status    string   `json:"status"`
	CanCancel bool     `json:"can_cancel"`
}

func (s *PermissionAuditEngine) persistAuditResult(ctx context.Context, auditID uint64, status string, summary, stats map[string]any, findings []model.K8sPermissionAuditFinding, errors []string) error {
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

const permissionAuditListPageLimit int64 = 500

func normalizePermissionAuditPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func permissionAuditKubeconfigError(err error) error {
	switch {
	case errors.Is(err, kopsclient.ErrKubeconfigEmpty):
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "kubeconfig 不能为空")
	case errors.Is(err, kopsclient.ErrKubeconfigTooLarge):
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "kubeconfig 内容不能超过 1MB")
	default:
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "kubeconfig 格式无效，请上传原始 YAML/JSON 或其 Base64 内容")
	}
}

func permissionAuditClusterError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, fleetdomain.ErrValidation):
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, err.Error())
	case errors.Is(err, fleetdomain.ErrNotFound):
		return kopsapp.ErrNotFound
	case errors.Is(err, fleetdomain.ErrConflict):
		return kopsapp.ErrConflict
	case errors.Is(err, fleetdomain.ErrCrypto):
		return kopsapp.ErrRuntime
	default:
		return err
	}
}

func normalizePermissionAuditK8sError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return kopsapp.ErrRuntimeTimeout
	}
	switch {
	case apierrors.IsNotFound(err):
		return kopsapp.ErrNotFound
	case apierrors.IsAlreadyExists(err):
		return kopsapp.ErrConflict
	case apierrors.IsInvalid(err), apierrors.IsBadRequest(err):
		return kopsapp.ErrInvalidParams
	case apierrors.IsUnauthorized(err):
		return kopsapp.ErrRuntimeUnauthorized
	case apierrors.IsForbidden(err):
		return kopsapp.ErrRuntimeForbidden
	case apierrors.IsTimeout(err):
		return kopsapp.ErrRuntimeTimeout
	default:
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr != nil {
			if inner := urlErr.Unwrap(); inner != nil {
				err = inner
			}
		}
		var unknownAuthority *x509.UnknownAuthorityError
		var hostname x509.HostnameError
		var invalidCert x509.CertificateInvalidError
		lower := strings.ToLower(err.Error())
		if errors.As(err, &unknownAuthority) || errors.As(err, &hostname) || errors.As(err, &invalidCert) || strings.Contains(lower, "x509:") {
			return kopsapp.ErrRuntimeTLS
		}
		var networkErr net.Error
		if errors.As(err, &networkErr) {
			if networkErr.Timeout() {
				return kopsapp.ErrRuntimeTimeout
			}
			return kopsapp.ErrRuntimeNetwork
		}
		if strings.Contains(lower, "connection refused") || strings.Contains(lower, "no route to host") || strings.Contains(lower, "i/o timeout") || strings.Contains(lower, "timeout") {
			return kopsapp.ErrRuntimeNetwork
		}
		return kopsapp.ErrRuntime
	}
}

func permissionAuditCredentialError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if message, ok := permissionAuditUserMessage(err); ok {
		return kopsapp.ErrWithMessage(kopsapp.ErrRuntime, message)
	}
	return kopsapp.ErrRuntime
}

func permissionAuditUserMessage(err error) (string, bool) {
	var carrier interface{ UserMessage() string }
	if errors.As(err, &carrier) && carrier != nil {
		if message := strings.TrimSpace(carrier.UserMessage()); message != "" {
			return message, true
		}
	}
	return "", false
}

// candidateGVRs keeps the version fallbacks needed by the permission-audit
// target set. Resource listing remains adapter-local and does not reach back
// into the legacy K8s service.
func candidateGVRs(gvr schema.GroupVersionResource) []schema.GroupVersionResource {
	key := gvr.Group + "/" + gvr.Resource
	versions := map[string][]string{
		"autoscaling/horizontalpodautoscalers":                         {"v2", "v2beta2", "v1"},
		"policy/poddisruptionbudgets":                                  {"v1", "v1beta1"},
		"batch/cronjobs":                                               {"v1", "v1beta1"},
		"networking.k8s.io/ingresses":                                  {"v1", "v1beta1"},
		"networking.k8s.io/ingressclasses":                             {"v1", "v1beta1"},
		"rbac.authorization.k8s.io/clusterroles":                       {"v1", "v1beta1"},
		"apiextensions.k8s.io/customresourcedefinitions":               {"v1", "v1beta1"},
		"admissionregistration.k8s.io/validatingwebhookconfigurations": {"v1", "v1beta1"},
		"admissionregistration.k8s.io/mutatingwebhookconfigurations":   {"v1", "v1beta1"},
		"apiregistration.k8s.io/apiservices":                           {"v1", "v1beta1"},
		"scheduling.k8s.io/priorityclasses":                            {"v1", "v1beta1"},
	}
	candidates, ok := versions[key]
	if !ok {
		return []schema.GroupVersionResource{gvr}
	}
	result := make([]schema.GroupVersionResource, 0, len(candidates)+1)
	seen := map[string]struct{}{}
	for _, version := range append([]string{gvr.Version}, candidates...) {
		candidate := schema.GroupVersionResource{Group: gvr.Group, Version: version, Resource: gvr.Resource}
		candidateKey := candidate.Group + "/" + candidate.Version + "/" + candidate.Resource
		if _, exists := seen[candidateKey]; exists {
			continue
		}
		seen[candidateKey] = struct{}{}
		result = append(result, candidate)
	}
	if gvr.Group == "networking.k8s.io" && gvr.Resource == "ingresses" {
		result = append(result, schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: gvr.Resource})
	}
	return result
}
