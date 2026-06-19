package service

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type SSHCredentialItem struct {
	ID          uint64  `json:"id"`
	Name        string  `json:"name"`
	AuthType    string  `json:"auth_type"`
	Username    string  `json:"username"`
	Remark      *string `json:"remark,omitempty"`
	ServerCount int     `json:"server_count"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type ListCredentialsRequest struct {
	Page     int
	PageSize int
	Keyword  string
	AuthType string
}

type CreateSSHCredentialRequest struct {
	Name       string `json:"name"`
	AuthType   string `json:"auth_type"`
	Username   string `json:"username"`
	Credential string `json:"credential"`
	Remark     string `json:"remark"`
}

type UpdateSSHCredentialRequest struct {
	Name       string  `json:"name"`
	AuthType   string  `json:"auth_type"`
	Username   string  `json:"username"`
	Credential string  `json:"credential"`
	Remark     *string `json:"remark"`
}

type DeployPlanNodeItem struct {
	ID        uint64 `json:"id"`
	ServerID  uint64 `json:"server_id"`
	Role      string `json:"role"`
	SortOrder int    `json:"sort_order"`
}

type DeployPlanItem struct {
	ID          uint64               `json:"id"`
	Name        string               `json:"name"`
	ClusterName string               `json:"cluster_name"`
	K8sVersion  string               `json:"k8s_version"`
	PodCIDR     string               `json:"pod_cidr"`
	SvcCIDR     string               `json:"svc_cidr"`
	CNIType     string               `json:"cni_type"`
	CNIConfig   map[string]any       `json:"cni_config,omitempty"`
	Addons      []string             `json:"addons,omitempty"`
	Status      string               `json:"status"`
	TaskID      *uint64              `json:"task_id,omitempty"`
	ClusterID   *uint64              `json:"cluster_id,omitempty"`
	CreatedBy   uint64               `json:"created_by"`
	Nodes       []DeployPlanNodeItem `json:"nodes,omitempty"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

type ListDeployPlansRequest struct {
	Page     int
	PageSize int
	Keyword  string
	Status   string
}

type CreateDeployPlanRequest struct {
	Name        string              `json:"name"`
	ClusterName string              `json:"cluster_name"`
	K8sVersion  string              `json:"k8s_version"`
	PodCIDR     string              `json:"pod_cidr"`
	SvcCIDR     string              `json:"svc_cidr"`
	CNIType     string              `json:"cni_type"`
	CNIConfig   map[string]any      `json:"cni_config"`
	Addons      []string            `json:"addons"`
	Nodes       []DeployPlanNodeReq `json:"nodes"`
}

type DeployPlanNodeReq struct {
	ServerID  uint64 `json:"server_id"`
	Role      string `json:"role"`
	SortOrder int    `json:"sort_order"`
}

func (s *DeployService) ListCredentials(ctx context.Context, req ListCredentialsRequest) (PageResult[SSHCredentialItem], error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	q := s.db.WithContext(ctx).Model(&model.SSHCredential{}).Where("deleted_at IS NULL")
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		q = q.Where("name LIKE ? OR username LIKE ?", "%"+kw+"%", "%"+kw+"%")
	}
	if at := strings.TrimSpace(req.AuthType); at != "" {
		q = q.Where("auth_type = ?", at)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[SSHCredentialItem]{}, err
	}
	var rows []model.SSHCredential
	if err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[SSHCredentialItem]{}, err
	}
	items := make([]SSHCredentialItem, 0, len(rows))
	for _, row := range rows {
		sc := countServersByCredential(s.db.WithContext(ctx), row.ID)
		items = append(items, sshCredentialToItem(row, sc))
	}
	return PageResult[SSHCredentialItem]{List: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *DeployService) CreateCredential(ctx context.Context, req CreateSSHCredentialRequest) (uint64, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "凭证名称不能为空")
	}
	authType := normalizeAuthType(req.AuthType)
	if authType == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "认证类型必须为 password 或 key")
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = "root"
	}
	credential := strings.TrimSpace(req.Credential)
	if credential == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "凭证不能为空")
	}
	enc, err := encryptText(s.encryptionKey, credential)
	if err != nil {
		return 0, err
	}
	row := model.SSHCredential{Name: name, AuthType: authType, Username: username, CredentialEnc: enc, Remark: stringPtrOrNil(req.Remark)}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *DeployService) DeleteCredential(ctx context.Context, id uint64) error {
	if id == 0 {
		return ErrInvalidParams
	}
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&model.SSHCredential{}).Where("deleted_at IS NULL AND id = ?", id).Update("deleted_at", &now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *DeployService) GetCredential(ctx context.Context, id uint64) (SSHCredentialItem, error) {
	if id == 0 {
		return SSHCredentialItem{}, ErrInvalidParams
	}
	var row model.SSHCredential
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SSHCredentialItem{}, ErrNotFound
		}
		return SSHCredentialItem{}, err
	}
	sc := countServersByCredential(s.db.WithContext(ctx), row.ID)
	return sshCredentialToItem(row, sc), nil
}

func (s *DeployService) UpdateCredential(ctx context.Context, id uint64, req UpdateSSHCredentialRequest) error {
	if id == 0 {
		return ErrInvalidParams
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ErrWithMessage(ErrInvalidParams, "凭证名称不能为空")
	}
	authType := normalizeAuthType(req.AuthType)
	if authType == "" {
		return ErrWithMessage(ErrInvalidParams, "认证类型必须为 password 或 key")
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = "root"
	}

	updates := map[string]any{
		"name":      name,
		"auth_type": authType,
		"username":  username,
		"remark":    req.Remark,
	}

	credential := strings.TrimSpace(req.Credential)
	if credential != "" {
		enc, err := encryptText(s.encryptionKey, credential)
		if err != nil {
			return err
		}
		updates["credential_enc"] = enc
	}

	res := s.db.WithContext(ctx).Model(&model.SSHCredential{}).Where("deleted_at IS NULL AND id = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *DeployService) ListPlans(ctx context.Context, req ListDeployPlansRequest) (PageResult[DeployPlanItem], error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	q := s.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("deleted_at IS NULL")
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		q = q.Where("name LIKE ? OR cluster_name LIKE ?", "%"+kw+"%", "%"+kw+"%")
	}
	if st := strings.TrimSpace(req.Status); st != "" {
		q = q.Where("status = ?", st)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[DeployPlanItem]{}, err
	}
	var rows []model.DeployPlan
	if err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[DeployPlanItem]{}, err
	}
	items := make([]DeployPlanItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, deployPlanToItem(row, nil))
	}
	return PageResult[DeployPlanItem]{List: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *DeployService) CreatePlan(ctx context.Context, req CreateDeployPlanRequest, createdBy uint64) (uint64, error) {
	plan, nodes, err := normalizeDeployPlan(req, createdBy)
	if err != nil {
		return 0, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureClusterNameUnique(tx, plan.ClusterName, 0); err != nil {
			return err
		}
		if err := validatePlanServers(tx, nodes); err != nil {
			return err
		}
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		for i := range nodes {
			nodes[i].PlanID = plan.ID
		}
		return tx.Create(&nodes).Error
	})
	if err != nil {
		return 0, err
	}
	return plan.ID, nil
}

func (s *DeployService) GetPlan(ctx context.Context, id uint64) (DeployPlanItem, error) {
	plan, nodes, err := s.getPlanWithNodes(ctx, id)
	if err != nil {
		return DeployPlanItem{}, err
	}
	return deployPlanToItem(plan, nodes), nil
}

func (s *DeployService) DeletePlan(ctx context.Context, id uint64) error {
	if id == 0 {
		return ErrInvalidParams
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var plan model.DeployPlan
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&plan).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if plan.Status == "running" || plan.Status == "success" {
			return ErrWithMessage(ErrConflict, "当前状态不允许删除部署计划")
		}
		return tx.Model(&model.DeployPlan{}).Where("id = ?", id).Update("deleted_at", &now).Error
	})
}

func (s *DeployService) ExecutePlan(ctx context.Context, id uint64, userID uint64) (uint64, error) {
	if id == 0 {
		return 0, ErrInvalidParams
	}
	var taskID uint64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var plan model.DeployPlan
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&plan).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if plan.Status != "draft" && plan.Status != "failed" && plan.Status != "cancelled" {
			return ErrWithMessage(ErrConflict, "当前状态不允许执行部署")
		}
		title := "部署集群 " + plan.ClusterName
		percent := 0
		message := "部署任务已创建，正在执行"
		task := &Task{Type: "deploy_cluster", Status: TaskPending, Title: &title, CreatedBy: int64(userID), Percent: &percent, Message: &message, Meta: map[string]any{"deploy_plan_id": plan.ID}}
		if err := s.taskStore.Put(task); err != nil {
			return err
		}
		taskID = uint64(task.ID)
		return tx.Model(&model.DeployPlan{}).Where("id = ?", id).Updates(map[string]any{"status": "running", "task_id": taskID}).Error
	})
	if err != nil {
		return 0, err
	}
	// 异步启动部署流水线
	go s.deployPipeline(context.Background(), id, int64(taskID))
	return taskID, nil
}

func (s *DeployService) CancelPlan(ctx context.Context, id uint64) error {
	// 获取关联的任务 ID 并取消
	var plan model.DeployPlan
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if plan.Status != "running" {
		return ErrWithMessage(ErrConflict, "仅运行中的计划可取消")
	}
	if plan.TaskID != nil && *plan.TaskID > 0 {
		s.taskStore.CancelExecution(int64(*plan.TaskID))
	}
	return s.updatePlanStatus(ctx, id, "running", "cancelled")
}

func (s *DeployService) RetryPlan(ctx context.Context, id uint64, userID uint64) (uint64, error) {
	var plan model.DeployPlan
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if plan.Status != "failed" && plan.Status != "cancelled" {
		return 0, ErrWithMessage(ErrConflict, "仅失败或已取消的计划可重试")
	}
	return s.ExecutePlan(ctx, id, userID)
}

func (s *DeployService) updatePlanStatus(ctx context.Context, id uint64, from string, to string) error {
	res := s.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("deleted_at IS NULL AND id = ? AND status = ?", id, from).Update("status", to)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrWithMessage(ErrConflict, "部署计划状态不允许执行该操作")
	}
	return nil
}

func (s *DeployService) getPlanWithNodes(ctx context.Context, id uint64) (model.DeployPlan, []model.DeployPlanNode, error) {
	var plan model.DeployPlan
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.DeployPlan{}, nil, ErrNotFound
		}
		return model.DeployPlan{}, nil, err
	}
	var nodes []model.DeployPlanNode
	if err := s.db.WithContext(ctx).Where("plan_id = ?", id).Order("sort_order asc, id asc").Find(&nodes).Error; err != nil {
		return model.DeployPlan{}, nil, err
	}
	return plan, nodes, nil
}

func normalizeDeployPlan(req CreateDeployPlanRequest, createdBy uint64) (model.DeployPlan, []model.DeployPlanNode, error) {
	name := strings.TrimSpace(req.Name)
	clusterName := strings.TrimSpace(req.ClusterName)
	k8sVersion := strings.TrimSpace(req.K8sVersion)
	if name == "" || clusterName == "" || k8sVersion == "" {
		return model.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "计划名称、集群名称和 K8s 版本不能为空")
	}
	podCIDR := strings.TrimSpace(req.PodCIDR)
	if podCIDR == "" {
		podCIDR = "10.244.0.0/16"
	}
	if _, _, err := net.ParseCIDR(podCIDR); err != nil {
		return model.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "Pod 网段格式不正确")
	}
	svcCIDR := strings.TrimSpace(req.SvcCIDR)
	if svcCIDR == "" {
		svcCIDR = "10.96.0.0/12"
	}
	if _, _, err := net.ParseCIDR(svcCIDR); err != nil {
		return model.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "Service 网段格式不正确")
	}
	cniType := strings.TrimSpace(req.CNIType)
	if cniType == "" {
		cniType = "flannel"
	}
	if cniType != "flannel" && cniType != "calico" && cniType != "cilium" {
		return model.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "CNI 类型不支持")
	}
	if len(req.Nodes) == 0 {
		return model.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "至少选择一个部署节点")
	}
	nodes := make([]model.DeployPlanNode, 0, len(req.Nodes))
	masterCount := 0
	seen := map[uint64]bool{}
	for _, n := range req.Nodes {
		if n.ServerID == 0 || seen[n.ServerID] {
			return model.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "节点配置不正确")
		}
		seen[n.ServerID] = true
		role := strings.TrimSpace(n.Role)
		if role != "master" && role != "worker" {
			return model.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "节点角色必须为 master 或 worker")
		}
		if role == "master" {
			masterCount++
		}
		nodes = append(nodes, model.DeployPlanNode{ServerID: n.ServerID, Role: role, SortOrder: n.SortOrder})
	}
	if masterCount == 0 {
		return model.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "至少需要一个 master 节点")
	}
	plan := model.DeployPlan{Name: name, ClusterName: clusterName, K8sVersion: k8sVersion, PodCIDR: podCIDR, SvcCIDR: svcCIDR, CNIType: cniType, CNIConfig: model.JSONMap(req.CNIConfig), Addons: model.JSONStringSlice(req.Addons), Status: "draft", CreatedBy: createdBy}
	return plan, nodes, nil
}

func validatePlanServers(tx *gorm.DB, nodes []model.DeployPlanNode) error {
	ids := make([]uint64, 0, len(nodes))
	for _, n := range nodes {
		ids = append(ids, n.ServerID)
	}
	var count int64
	if err := tx.Model(&model.DeployServer{}).Where("deleted_at IS NULL AND id IN ? AND status IN ?", ids, []string{"available", "registered"}).Count(&count).Error; err != nil {
		return err
	}
	if int(count) != len(ids) {
		return ErrWithMessage(ErrInvalidParams, "存在不可用或不存在的服务器")
	}
	return nil
}

func ensureClusterNameUnique(tx *gorm.DB, name string, excludeID uint64) error {
	q := tx.Where("deleted_at IS NULL AND cluster_name = ?", name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var existing model.DeployPlan
	if err := q.Select("id").First(&existing).Error; err == nil {
		return ErrWithMessage(ErrConflict, "集群名称已被部署计划使用")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func deployPlanToItem(row model.DeployPlan, nodes []model.DeployPlanNode) DeployPlanItem {
	item := DeployPlanItem{ID: row.ID, Name: row.Name, ClusterName: row.ClusterName, K8sVersion: row.K8sVersion, PodCIDR: row.PodCIDR, SvcCIDR: row.SvcCIDR, CNIType: row.CNIType, CNIConfig: map[string]any(row.CNIConfig), Addons: []string(row.Addons), Status: row.Status, TaskID: row.TaskID, ClusterID: row.ClusterID, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339)}
	if nodes != nil {
		item.Nodes = make([]DeployPlanNodeItem, 0, len(nodes))
		for _, n := range nodes {
			item.Nodes = append(item.Nodes, DeployPlanNodeItem{ID: n.ID, ServerID: n.ServerID, Role: n.Role, SortOrder: n.SortOrder})
		}
	}
	return item
}

func sshCredentialToItem(row model.SSHCredential, serverCount int) SSHCredentialItem {
	return SSHCredentialItem{ID: row.ID, Name: row.Name, AuthType: row.AuthType, Username: row.Username, Remark: row.Remark, ServerCount: serverCount, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339)}
}

func countServersByCredential(db *gorm.DB, credentialID uint64) int {
	var count int64
	db.Model(&model.DeployServer{}).Where("deleted_at IS NULL AND credential_id = ?", credentialID).Count(&count)
	return int(count)
}

func (s *DeployService) BatchDeleteCredentials(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return ErrInvalidParams
	}
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&model.SSHCredential{}).Where("deleted_at IS NULL AND id IN ?", ids).Update("deleted_at", &now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
