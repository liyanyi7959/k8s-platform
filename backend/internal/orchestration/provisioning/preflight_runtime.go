package provisioning

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
	secretcrypto "k8s-platform-backend/internal/transport/secretcrypto"
	sshtransport "k8s-platform-backend/internal/transport/ssh"
)

// PreflightRuntime owns deployment-plan readiness reads and managed SSH
// probes. The deterministic readiness policy remains in
// provisioning/application; the retained service package is used only for
// narrow credential and SSH transports.
type PreflightRuntime struct {
	db            *gorm.DB
	encryptionKey string
}

func NewPreflightRuntime(db *gorm.DB, encryptionKey string) *PreflightRuntime {
	return &PreflightRuntime{db: db, encryptionKey: encryptionKey}
}

func (r *PreflightRuntime) Preflight(ctx context.Context, planID uint64) (provisionapp.PreflightResult, error) {
	plan, nodes, err := r.planWithNodes(ctx, planID)
	if err != nil {
		return provisionapp.PreflightResult{}, err
	}

	roles := make([]string, 0, len(nodes))
	for _, node := range nodes {
		roles = append(roles, node.Role)
	}
	result := provisionapp.NewPreflightAccumulator(time.Now(), plan.PreflightIgnores)
	result.AddAll(provisionapp.PlanPreflightChecks(plan.PodCIDR, plan.SvcCIDR, roles))
	result.Add(provisionapp.ControllerRunnerCheck())

	info, statErr := os.Stat(provisioningPlaybookPath())
	result.Add(provisionapp.ControllerPlaybookCheck(statErr == nil && !info.IsDir()))
	for _, node := range nodes {
		r.checkNode(ctx, node, result)
	}
	return result.Result(), nil
}

func (r *PreflightRuntime) SetPreflightIgnore(ctx context.Context, planID uint64, key string, ignored bool) error {
	plan, nodes, err := r.planWithNodes(ctx, planID)
	if err != nil {
		return err
	}
	nodeIDs := make([]uint64, 0, len(nodes))
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ServerID)
	}
	items, err := provisionapp.UpdatePreflightIgnores(plan.PreflightIgnores, nodeIDs, key, ignored)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("deleted_at IS NULL AND id = ?", planID).Update("preflight_ignores", model.JSONStringSlice(items)).Error
}

func (r *PreflightRuntime) planWithNodes(ctx context.Context, planID uint64) (model.DeployPlan, []model.DeployPlanNode, error) {
	if r == nil || r.db == nil {
		return model.DeployPlan{}, nil, provisionapp.ErrConflict
	}
	var plan model.DeployPlan
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", planID).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.DeployPlan{}, nil, provisionapp.ErrNotFound
		}
		return model.DeployPlan{}, nil, err
	}
	var nodes []model.DeployPlanNode
	if err := r.db.WithContext(ctx).Where("plan_id = ?", planID).Order("sort_order asc, id asc").Find(&nodes).Error; err != nil {
		return model.DeployPlan{}, nil, err
	}
	return plan, nodes, nil
}

func (r *PreflightRuntime) checkNode(ctx context.Context, node model.DeployPlanNode, result *provisionapp.PreflightAccumulator) {
	server, credential, authType, err := r.serverCredential(ctx, node.ServerID)
	if err != nil {
		result.Add(provisionapp.NodeSSHFailureCheck(node.ServerID, server.Name, err.Error()))
		return
	}
	server.AuthType = authType
	probe, probeErr := sshtransport.Probe(ctx, deploymentSSHConfig(server), credential)
	if probeErr != nil {
		result.Add(provisionapp.NodeSSHConnectionFailureCheck(node.ServerID, server.Name, probeErr.Error()))
		return
	}
	result.Add(provisionapp.NodeSSHReachableCheck(node.ServerID, server.Name, probe.OS, probe.OSVersion))
	result.AddAll(provisionapp.NodeProbeChecks(provisionapp.PreflightNodeProbe{
		ServerID: node.ServerID, ServerName: server.Name, OS: probe.OS, OSVersion: probe.OSVersion,
		CPUCores: preflightUintValue(probe.CPUCores), MemoryMB: preflightUint64Value(probe.MemoryMB), DiskGB: preflightUint64Value(probe.DiskGB),
	}))

	client, dialErr := sshtransport.Dial(ctx, deploymentSSHConfig(server), credential)
	if dialErr != nil {
		result.Add(provisionapp.NodeRuntimeCheck(provisionapp.PreflightNodeRuntime{
			ServerID: node.ServerID, ServerName: server.Name, ConnectionMessage: dialErr.Error(),
		}))
		return
	}
	defer client.Close()
	output, commandErr := sshtransport.RunCommand(client, `command -v python3 >/dev/null 2>&1 && echo python3=ok || echo python3=missing; if [ "$(id -u)" = "0" ] || sudo -n true >/dev/null 2>&1; then echo privilege=ok; elif command -v sudo >/dev/null 2>&1; then echo privilege=password; else echo privilege=missing; fi`)
	privilegeUsable := strings.Contains(output, "privilege=ok") || (authType != "key" && strings.Contains(output, "privilege=password"))
	result.Add(provisionapp.NodeRuntimeCheck(provisionapp.PreflightNodeRuntime{
		ServerID: node.ServerID, ServerName: server.Name, CommandFailed: commandErr != nil,
		Python3Available: strings.Contains(output, "python3=ok"), PrivilegeUsable: privilegeUsable,
	}))
}

func (r *PreflightRuntime) serverCredential(ctx context.Context, serverID uint64) (model.DeployServer, string, string, error) {
	var server model.DeployServer
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", serverID).First(&server).Error; err != nil {
		return model.DeployServer{}, "", "", fmt.Errorf("服务器不存在: %w", err)
	}
	credentialCiphertext := server.CredentialEnc
	authType := server.AuthType
	if server.CredentialID != nil && *server.CredentialID > 0 {
		var credential model.Credential
		if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", *server.CredentialID).First(&credential).Error; err != nil {
			return model.DeployServer{}, "", "", fmt.Errorf("关联的凭据不存在: %w", err)
		}
		credentialCiphertext = credential.CredentialEnc
		authType = credential.AuthType
	}
	secret, err := secretcrypto.Decrypt(r.encryptionKey, credentialCiphertext)
	if err != nil {
		return model.DeployServer{}, "", "", fmt.Errorf("解密凭证失败: %w", err)
	}
	return server, secret, authType, nil
}

func preflightUintValue(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

func preflightUint64Value(value *uint64) uint64 {
	if value == nil {
		return 0
	}
	return *value
}
