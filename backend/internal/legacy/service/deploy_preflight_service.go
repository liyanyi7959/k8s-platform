package service

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
)

// Legacy aliases preserve the existing service and HTTP payload contract while
// the DTO and readiness policy belong to the Provisioning application layer.
type DeployPreflightCheck = provisionapp.PreflightCheck
type DeployPreflightResult = provisionapp.PreflightResult

const (
	preflightPassed  = provisionapp.PreflightPassed
	preflightWarning = provisionapp.PreflightWarning
	preflightError   = provisionapp.PreflightError
)

// PreflightPlan is the retained runtime adapter: it loads the deployment plan,
// checks local Ansible assets, and probes SSH hosts. All deterministic
// readiness decisions are delegated to Provisioning application policies.
func (s *DeployService) PreflightPlan(ctx context.Context, planID uint64) (DeployPreflightResult, error) {
	plan, nodes, err := s.getPlanWithNodes(ctx, planID)
	if err != nil {
		return DeployPreflightResult{}, err
	}

	roles := make([]string, 0, len(nodes))
	for _, node := range nodes {
		roles = append(roles, node.Role)
	}
	result := provisionapp.NewPreflightAccumulator(time.Now(), plan.PreflightIgnores)
	result.AddAll(provisionapp.PlanPreflightChecks(plan.PodCIDR, plan.SvcCIDR, roles))
	result.Add(provisionapp.ControllerRunnerCheck())

	info, statErr := os.Stat(s.ansiblePlaybookPath())
	result.Add(provisionapp.ControllerPlaybookCheck(statErr == nil && !info.IsDir()))

	for _, node := range nodes {
		s.checkPreflightNode(ctx, node, result)
	}
	return result.Result(), nil
}

func (s *DeployService) SetPreflightIgnore(ctx context.Context, planID uint64, key string, ignored bool) error {
	plan, nodes, err := s.getPlanWithNodes(ctx, planID)
	if err != nil {
		return err
	}
	nodeIDs := make([]uint64, 0, len(nodes))
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ServerID)
	}
	items, err := provisionapp.UpdatePreflightIgnores(plan.PreflightIgnores, nodeIDs, key, ignored)
	if err != nil {
		return legacyPreflightError(err)
	}
	return s.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("deleted_at IS NULL AND id = ?", planID).Update("preflight_ignores", model.JSONStringSlice(items)).Error
}

func (s *DeployService) checkPreflightNode(ctx context.Context, node model.DeployPlanNode, result *provisionapp.PreflightAccumulator) {
	server, credential, authType, err := s.getServerCredentialForNode(ctx, node.ServerID)
	if err != nil {
		result.Add(provisionapp.NodeSSHFailureCheck(node.ServerID, server.Name, err.Error()))
		return
	}
	server.AuthType = authType
	probe, probeErr := probeSSH(ctx, server, credential)
	if probeErr != nil {
		result.Add(provisionapp.NodeSSHConnectionFailureCheck(node.ServerID, server.Name, probeErr.Error()))
		return
	}
	result.Add(provisionapp.NodeSSHReachableCheck(node.ServerID, server.Name, probe.OS, probe.OSVersion))
	result.AddAll(provisionapp.NodeProbeChecks(provisionapp.PreflightNodeProbe{
		ServerID: node.ServerID, ServerName: server.Name, OS: probe.OS, OSVersion: probe.OSVersion,
		CPUCores: uintValue(probe.CPUCores), MemoryMB: uint64Value(probe.MemoryMB), DiskGB: uint64Value(probe.DiskGB),
	}))

	client, dialErr := dialDeploySSH(ctx, server, credential)
	if dialErr != nil {
		result.Add(provisionapp.NodeRuntimeCheck(provisionapp.PreflightNodeRuntime{
			ServerID: node.ServerID, ServerName: server.Name, ConnectionMessage: dialErr.Error(),
		}))
		return
	}
	defer client.Close()
	output, commandErr := runSSHCommand(client, `command -v python3 >/dev/null 2>&1 && echo python3=ok || echo python3=missing; if [ "$(id -u)" = "0" ] || sudo -n true >/dev/null 2>&1; then echo privilege=ok; elif command -v sudo >/dev/null 2>&1; then echo privilege=password; else echo privilege=missing; fi`)
	privilegeUsable := strings.Contains(output, "privilege=ok") || (authType != "key" && strings.Contains(output, "privilege=password"))
	result.Add(provisionapp.NodeRuntimeCheck(provisionapp.PreflightNodeRuntime{
		ServerID: node.ServerID, ServerName: server.Name, CommandFailed: commandErr != nil,
		Python3Available: strings.Contains(output, "python3=ok"), PrivilegeUsable: privilegeUsable,
	}))
}

func legacyPreflightError(err error) error {
	if !errors.Is(err, provisionapp.ErrInvalidParams) {
		return err
	}
	if message, ok := UserMessage(err); ok {
		return ErrWithMessage(ErrInvalidParams, message)
	}
	return ErrInvalidParams
}

func uintValue(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

func uint64Value(value *uint64) uint64 {
	if value == nil {
		return 0
	}
	return *value
}
