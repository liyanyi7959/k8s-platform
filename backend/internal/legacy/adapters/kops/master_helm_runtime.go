package kops

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/kops/adapters/legacycompat"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
	secretcrypto "k8s-platform-backend/internal/transport/secretcrypto"
	sshtransport "k8s-platform-backend/internal/transport/ssh"
)

// MasterHelmRuntime owns the managed-Master SSH boundary used by Helm.
// Helm command policy and output parsing belong to provisioning/application;
// the retained service package provides only credential and SSH transports.
type MasterHelmRuntime struct {
	db            *gorm.DB
	encryptionKey string
}

func NewMasterHelmRuntime(db *gorm.DB, encryptionKey string) *MasterHelmRuntime {
	return &MasterHelmRuntime{db: db, encryptionKey: encryptionKey}
}

func (r *MasterHelmRuntime) Ensure(ctx context.Context, clusterID uint64) (provisionapp.HelmMasterPreflight, error) {
	if r == nil || r.db == nil {
		return provisionapp.HelmMasterPreflight{}, fmt.Errorf("Helm 前置校验不可用：部署服务未初始化")
	}
	if clusterID == 0 {
		return provisionapp.HelmMasterPreflight{}, service.ErrInvalidParams
	}

	master, credential, err := r.master(ctx, clusterID, "Master 主机当前不可用，请先在主机资源池完成 SSH 巡检后再部署")
	if err != nil {
		return provisionapp.HelmMasterPreflight{}, err
	}
	client, err := sshtransport.Dial(ctx, masterSSHConfig(master), credential)
	if err != nil {
		return provisionapp.HelmMasterPreflight{}, service.ErrWithMessage(service.ErrConflict, "Master SSH 连接失败，已阻止 Helm 部署："+err.Error())
	}
	defer client.Close()

	output, err := sshtransport.RunPrivilegedCommand(client, master.AuthType, credential, provisionapp.HelmMasterEnsureScript())
	if err != nil {
		return provisionapp.HelmMasterPreflight{}, service.ErrWithMessage(service.ErrConflict, "Master Helm 安装或校验失败："+err.Error())
	}
	result, err := provisionapp.ParseHelmMasterPreflight(clusterID, master.Name, master.IP, output)
	if err != nil {
		return provisionapp.HelmMasterPreflight{}, masterHelmRuntimeError(err)
	}
	return result, nil
}

func (r *MasterHelmRuntime) Install(ctx context.Context, clusterID uint64, request provisionapp.HelmMasterInstallRequest) (provisionapp.HelmMasterInstallResult, error) {
	if r == nil || r.db == nil {
		return provisionapp.HelmMasterInstallResult{}, fmt.Errorf("Helm 部署服务未初始化")
	}
	if clusterID == 0 {
		return provisionapp.HelmMasterInstallResult{}, service.ErrInvalidParams
	}
	output, err := r.run(ctx, clusterID, provisionapp.HelmMasterInstallScript(request), true)
	if err != nil {
		if strings.Contains(output, "AIOPS_REPOSITORY_UNAVAILABLE") {
			return provisionapp.HelmMasterInstallResult{}, service.ErrWithMessage(service.ErrConflict, "Master 无法访问 Helm 仓库。系统已在 Master 上重试 3 次；请检查该节点的 DNS、HTTPS 出口或改用可访问的内部仓库")
		}
		return provisionapp.HelmMasterInstallResult{}, service.ErrWithMessage(service.ErrConflict, "Master 执行 Helm 安装失败："+err.Error())
	}
	return provisionapp.HelmMasterInstallResult{Output: output}, nil
}

func (r *MasterHelmRuntime) ListRepositories(ctx context.Context, clusterID uint64) ([]provisionapp.HelmRepository, error) {
	output, err := r.run(ctx, clusterID, provisionapp.HelmMasterListRepositoriesScript(), false)
	if err != nil {
		return nil, err
	}
	items, err := provisionapp.ParseHelmMasterRepositories(output)
	if err != nil {
		return nil, masterHelmRuntimeError(err)
	}
	return items, nil
}

func (r *MasterHelmRuntime) AddRepository(ctx context.Context, clusterID uint64, name, repositoryURL string) error {
	_, err := r.run(ctx, clusterID, provisionapp.HelmMasterAddRepositoryScript(name, repositoryURL), true)
	return err
}

func (r *MasterHelmRuntime) DeleteRepository(ctx context.Context, clusterID uint64, name string) error {
	_, err := r.run(ctx, clusterID, provisionapp.HelmMasterDeleteRepositoryScript(name), false)
	return err
}

func (r *MasterHelmRuntime) SearchCharts(ctx context.Context, clusterID uint64, keyword string) ([]provisionapp.HelmChartSearchResult, error) {
	output, err := r.run(ctx, clusterID, provisionapp.HelmMasterSearchChartsScript(), false)
	if err != nil {
		return nil, err
	}
	items, err := provisionapp.ParseHelmMasterChartSearch(output, keyword)
	if err != nil {
		return nil, masterHelmRuntimeError(err)
	}
	return items, nil
}

func (r *MasterHelmRuntime) run(ctx context.Context, clusterID uint64, script string, ensureHelm bool) (string, error) {
	if r == nil || r.db == nil {
		return "", fmt.Errorf("Helm 部署服务未初始化")
	}
	if clusterID == 0 {
		return "", service.ErrInvalidParams
	}
	if ensureHelm {
		if _, err := r.Ensure(ctx, clusterID); err != nil {
			return "", err
		}
	}

	master, credential, err := r.master(ctx, clusterID, "Master 主机当前不可用，请先完成 SSH 巡检后再操作 Helm")
	if err != nil {
		return "", err
	}
	client, err := sshtransport.Dial(ctx, masterSSHConfig(master), credential)
	if err != nil {
		return "", service.ErrWithMessage(service.ErrConflict, "Master SSH 连接失败，已阻止 Helm 操作："+err.Error())
	}
	defer client.Close()

	output, err := sshtransport.RunPrivilegedCommand(client, master.AuthType, credential, script)
	if err != nil {
		return output, service.ErrWithMessage(service.ErrConflict, "Master 执行 Helm 命令失败："+err.Error())
	}
	return strings.TrimSpace(output), nil
}

func (r *MasterHelmRuntime) master(ctx context.Context, clusterID uint64, unavailableMessage string) (model.DeployServer, string, error) {
	var plan model.DeployPlan
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND cluster_id = ?", clusterID).
		Order("id DESC").
		First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.DeployServer{}, "", service.ErrWithMessage(service.ErrNotFound, "当前集群未关联平台部署计划，无法安全定位 Master 并校验 Helm；请先纳管 Master SSH 资产后再部署")
		}
		return model.DeployServer{}, "", err
	}
	var node model.DeployPlanNode
	if err := r.db.WithContext(ctx).
		Where("plan_id = ? AND role = ?", plan.ID, "master").
		Order("sort_order ASC, id ASC").
		First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.DeployServer{}, "", service.ErrWithMessage(service.ErrNotFound, "部署计划未配置 Master 节点，无法执行 Helm 操作")
		}
		return model.DeployServer{}, "", err
	}

	master, credential, err := r.serverCredential(ctx, node.ServerID)
	if err != nil {
		return model.DeployServer{}, "", service.ErrWithMessage(service.ErrNotFound, "无法读取 Master SSH 凭据："+err.Error())
	}
	if master.Status != "available" {
		return model.DeployServer{}, "", service.ErrWithMessage(service.ErrConflict, unavailableMessage)
	}
	return master, credential, nil
}

func (r *MasterHelmRuntime) serverCredential(ctx context.Context, serverID uint64) (model.DeployServer, string, error) {
	var server model.DeployServer
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", serverID).First(&server).Error; err != nil {
		return model.DeployServer{}, "", err
	}
	credentialCiphertext := server.CredentialEnc
	if server.CredentialID != nil && *server.CredentialID > 0 {
		var credential model.SSHCredential
		if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", *server.CredentialID).First(&credential).Error; err != nil {
			return model.DeployServer{}, "", err
		}
		credentialCiphertext = credential.CredentialEnc
		server.AuthType = credential.AuthType
	}
	credential, err := secretcrypto.Decrypt(r.encryptionKey, credentialCiphertext)
	if err != nil {
		return model.DeployServer{}, "", err
	}
	return server, credential, nil
}

func masterSSHConfig(server model.DeployServer) sshtransport.Config {
	return sshtransport.Config{Host: server.IP, Port: server.SSHPort, User: server.User, AuthType: server.AuthType}
}

func masterHelmRuntimeError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, provisionapp.ErrInvalidParams):
		if message, ok := service.UserMessage(err); ok {
			return service.ErrWithMessage(service.ErrInvalidParams, message)
		}
		return service.ErrInvalidParams
	case errors.Is(err, provisionapp.ErrNotFound):
		return service.ErrNotFound
	case errors.Is(err, provisionapp.ErrConflict):
		if message, ok := service.UserMessage(err); ok {
			return service.ErrWithMessage(service.ErrConflict, message)
		}
		return service.ErrConflict
	default:
		return err
	}
}
