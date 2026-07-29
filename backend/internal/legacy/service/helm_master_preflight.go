package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
)

// Compatibility aliases keep the Kops HTTP adapter stable while Helm's DTOs
// and deterministic policies are now owned by Provisioning.
type HelmMasterPreflight = provisionapp.HelmMasterPreflight
type HelmMasterInstallRequest = provisionapp.HelmMasterInstallRequest
type HelmMasterInstallResult = provisionapp.HelmMasterInstallResult
type HelmRepository = provisionapp.HelmRepository
type HelmChartSearchResult = provisionapp.HelmChartSearchResult

// EnsureClusterMasterHelm is the retained runtime adapter. It resolves the
// Master from GORM, opens managed SSH, and delegates script/output policy to
// Provisioning application code.
func (s *DeployService) EnsureClusterMasterHelm(ctx context.Context, clusterID uint64) (HelmMasterPreflight, error) {
	if s == nil || s.db == nil {
		return HelmMasterPreflight{}, fmt.Errorf("Helm 前置校验不可用：部署服务未初始化")
	}
	if clusterID == 0 {
		return HelmMasterPreflight{}, ErrInvalidParams
	}

	var plan model.DeployPlan
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND cluster_id = ?", clusterID).
		Order("id DESC").
		First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return HelmMasterPreflight{}, ErrWithMessage(ErrNotFound, "当前集群未关联平台部署计划，无法安全定位 Master 并校验 Helm；请先纳管 Master SSH 资产后再部署")
		}
		return HelmMasterPreflight{}, err
	}

	var masterNode model.DeployPlanNode
	if err := s.db.WithContext(ctx).
		Where("plan_id = ? AND role = ?", plan.ID, "master").
		Order("sort_order ASC, id ASC").
		First(&masterNode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return HelmMasterPreflight{}, ErrWithMessage(ErrNotFound, "部署计划未配置 Master 节点，无法执行 Helm 前置校验")
		}
		return HelmMasterPreflight{}, err
	}

	master, credential, authType, err := s.getServerCredentialForNode(ctx, masterNode.ServerID)
	if err != nil {
		return HelmMasterPreflight{}, ErrWithMessage(ErrNotFound, "无法读取 Master SSH 凭据："+err.Error())
	}
	if master.Status != "available" {
		return HelmMasterPreflight{}, ErrWithMessage(ErrConflict, "Master 主机当前不可用，请先在主机资源池完成 SSH 巡检后再部署")
	}
	master.AuthType = authType
	client, err := dialDeploySSH(ctx, master, credential)
	if err != nil {
		return HelmMasterPreflight{}, ErrWithMessage(ErrConflict, "Master SSH 连接失败，已阻止 Helm 部署："+err.Error())
	}
	defer client.Close()

	output, err := runPrivilegedSSHCommand(client, master, credential, provisionapp.HelmMasterEnsureScript())
	if err != nil {
		return HelmMasterPreflight{}, ErrWithMessage(ErrConflict, "Master Helm 安装或校验失败："+err.Error())
	}
	result, err := provisionapp.ParseHelmMasterPreflight(clusterID, master.Name, master.IP, output)
	if err != nil {
		return HelmMasterPreflight{}, legacyHelmMasterError(err)
	}
	return result, nil
}

// InstallClusterMasterHelm executes a Provisioning-generated script on the
// prepared Master. The adapter owns only SSH transport and legacy errors.
func (s *DeployService) InstallClusterMasterHelm(ctx context.Context, clusterID uint64, req HelmMasterInstallRequest) (HelmMasterInstallResult, error) {
	if s == nil || s.db == nil {
		return HelmMasterInstallResult{}, fmt.Errorf("Helm 部署服务未初始化")
	}
	if clusterID == 0 {
		return HelmMasterInstallResult{}, ErrInvalidParams
	}

	output, err := s.runClusterMasterHelmScript(ctx, clusterID, provisionapp.HelmMasterInstallScript(req), true)
	if err != nil {
		if strings.Contains(output, "AIOPS_REPOSITORY_UNAVAILABLE") {
			return HelmMasterInstallResult{}, ErrWithMessage(ErrConflict, "Master 无法访问 Helm 仓库。系统已在 Master 上重试 3 次；请检查该节点的 DNS、HTTPS 出口或改用可访问的内部仓库")
		}
		return HelmMasterInstallResult{}, ErrWithMessage(ErrConflict, "Master 执行 Helm 安装失败："+err.Error())
	}
	return HelmMasterInstallResult{Output: output}, nil
}

func (s *DeployService) ListClusterMasterHelmRepos(ctx context.Context, clusterID uint64) ([]HelmRepository, error) {
	output, err := s.runClusterMasterHelmScript(ctx, clusterID, provisionapp.HelmMasterListRepositoriesScript(), false)
	if err != nil {
		return nil, err
	}
	value, err := provisionapp.ParseHelmMasterRepositories(output)
	if err != nil {
		return nil, legacyHelmMasterError(err)
	}
	return value, nil
}

func (s *DeployService) AddClusterMasterHelmRepo(ctx context.Context, clusterID uint64, name, repoURL string) error {
	_, err := s.runClusterMasterHelmScript(ctx, clusterID, provisionapp.HelmMasterAddRepositoryScript(name, repoURL), true)
	return err
}

func (s *DeployService) DeleteClusterMasterHelmRepo(ctx context.Context, clusterID uint64, name string) error {
	_, err := s.runClusterMasterHelmScript(ctx, clusterID, provisionapp.HelmMasterDeleteRepositoryScript(name), false)
	return err
}

func (s *DeployService) SearchClusterMasterHelmCharts(ctx context.Context, clusterID uint64, keyword string) ([]HelmChartSearchResult, error) {
	output, err := s.runClusterMasterHelmScript(ctx, clusterID, provisionapp.HelmMasterSearchChartsScript(), false)
	if err != nil {
		return nil, err
	}
	value, err := provisionapp.ParseHelmMasterChartSearch(output, keyword)
	if err != nil {
		return nil, legacyHelmMasterError(err)
	}
	return value, nil
}

// runClusterMasterHelmScript locates a managed control-plane host and
// executes a pre-built command. Script content is owned by Provisioning, and
// this method contains no Helm-specific business policy.
func (s *DeployService) runClusterMasterHelmScript(ctx context.Context, clusterID uint64, script string, ensureHelm bool) (string, error) {
	if s == nil || s.db == nil {
		return "", fmt.Errorf("Helm 部署服务未初始化")
	}
	if clusterID == 0 {
		return "", ErrInvalidParams
	}
	if ensureHelm {
		if _, err := s.EnsureClusterMasterHelm(ctx, clusterID); err != nil {
			return "", err
		}
	}

	var plan model.DeployPlan
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND cluster_id = ?", clusterID).
		Order("id DESC").
		First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrWithMessage(ErrNotFound, "当前集群未关联平台部署计划，无法安全定位 Master")
		}
		return "", err
	}
	var masterNode model.DeployPlanNode
	if err := s.db.WithContext(ctx).
		Where("plan_id = ? AND role = ?", plan.ID, "master").
		Order("sort_order ASC, id ASC").
		First(&masterNode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrWithMessage(ErrNotFound, "部署计划未配置 Master 节点，无法执行 Helm 操作")
		}
		return "", err
	}
	master, credential, authType, err := s.getServerCredentialForNode(ctx, masterNode.ServerID)
	if err != nil {
		return "", ErrWithMessage(ErrNotFound, "无法读取 Master SSH 凭据："+err.Error())
	}
	if master.Status != "available" {
		return "", ErrWithMessage(ErrConflict, "Master 主机当前不可用，请先完成 SSH 巡检后再操作 Helm")
	}
	master.AuthType = authType
	client, err := dialDeploySSH(ctx, master, credential)
	if err != nil {
		return "", ErrWithMessage(ErrConflict, "Master SSH 连接失败，已阻止 Helm 操作："+err.Error())
	}
	defer client.Close()

	output, err := runPrivilegedSSHCommand(client, master, credential, script)
	if err != nil {
		return output, ErrWithMessage(ErrConflict, "Master 执行 Helm 命令失败："+err.Error())
	}
	return strings.TrimSpace(output), nil
}

func legacyHelmMasterError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, provisionapp.ErrInvalidParams):
		if message, ok := UserMessage(err); ok {
			return ErrWithMessage(ErrInvalidParams, message)
		}
		return ErrInvalidParams
	case errors.Is(err, provisionapp.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, provisionapp.ErrConflict):
		if message, ok := UserMessage(err); ok {
			return ErrWithMessage(ErrConflict, message)
		}
		return ErrConflict
	default:
		return err
	}
}
