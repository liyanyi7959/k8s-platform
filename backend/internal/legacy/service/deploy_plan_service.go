package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	platformapp "k8s-platform-backend/internal/platform/application"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
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

func (s *DeployService) ExecutePlan(ctx context.Context, id uint64, userID uint64) (uint64, error) {
	return s.executePlanWithRetryStep(ctx, id, userID, "")
}

func (s *DeployService) executePlanWithRetryStep(ctx context.Context, id uint64, userID uint64, retryFromStep string) (uint64, error) {
	if id == 0 {
		return 0, ErrInvalidParams
	}
	preflight, err := s.PreflightPlan(ctx, id)
	if err != nil {
		return 0, err
	}
	if !preflight.Ready {
		return 0, ErrWithMessage(ErrInvalidParams, preflightFailureMessage(preflight))
	}
	var taskID uint64
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		if retryFromStep != "" {
			title += "（从步骤重试）"
		}
		percent := 0
		message := "部署任务已创建，正在执行"
		meta := map[string]any{"deploy_plan_id": plan.ID}
		if retryFromStep != "" {
			meta["retry_from_step"] = retryFromStep
		}
		task := &platformapp.Task{Type: "deploy_cluster", Status: platformapp.TaskPending, Title: &title, CreatedBy: int64(userID), Percent: &percent, Message: &message, Meta: meta}
		if err := s.taskStore.Put(task); err != nil {
			return err
		}
		taskID = uint64(task.ID)
		return tx.Model(&model.DeployPlan{}).Where("id = ?", id).Updates(map[string]any{"status": "running", "task_id": taskID}).Error
	})
	if err != nil {
		return 0, err
	}
	// 异步启动 Ansible 部署流水线
	go s.ansiblePipeline(context.Background(), id, int64(taskID))
	return taskID, nil
}

// RetryStep 从指定步骤开始重试部署。
func (s *DeployService) RetryStep(ctx context.Context, id uint64, stepKey string, userID uint64) (uint64, error) {
	if stepKey == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "步骤 key 不能为空")
	}
	var found bool
	for _, st := range provisionapp.DefaultAnsibleSteps() {
		if st.Key == stepKey {
			found = true
			break
		}
	}
	if !found {
		return 0, ErrWithMessage(ErrInvalidParams, "未知的步骤 key")
	}
	var plan model.DeployPlan
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if plan.Status != "failed" && plan.Status != "cancelled" {
		return 0, ErrWithMessage(ErrConflict, "仅失败或已取消的计划可重试步骤")
	}
	return s.executePlanWithRetryStep(ctx, id, userID, stepKey)
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
	// 默认从上次任务的第一个失败步骤开始重试，避免从头全部重跑
	retryFromStep := ""
	if plan.TaskID != nil && *plan.TaskID > 0 {
		if task, ok := s.taskStore.Get(int64(*plan.TaskID)); ok {
			ansibleSteps := provisionapp.DefaultAnsibleSteps()
			validKeys := make(map[string]struct{}, len(ansibleSteps))
			for _, st := range ansibleSteps {
				validKeys[st.Key] = struct{}{}
			}
			for _, st := range task.Steps {
				if st.Status == platformapp.StepFailed {
					if _, ok := validKeys[st.Key]; ok {
						retryFromStep = st.Key
						break
					}
				}
			}
		}
	}
	return s.executePlanWithRetryStep(ctx, id, userID, retryFromStep)
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

func preflightFailureMessage(result DeployPreflightResult) string {
	messages := make([]string, 0, 3)
	for _, check := range result.Checks {
		if check.Status == preflightError {
			messages = append(messages, check.Message)
		}
		if len(messages) == 3 {
			break
		}
	}
	if len(messages) == 0 {
		return "部署预检未通过"
	}
	return fmt.Sprintf("部署预检未通过：%s", strings.Join(messages, "；"))
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
