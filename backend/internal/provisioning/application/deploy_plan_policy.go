package application

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
)

var (
	clusterNamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	k8sVersionPattern  = regexp.MustCompile(`^v?1\.[0-9]+\.[0-9]+$`)
)

// DeployPlanInput is the transport-neutral command for plan normalization.
type DeployPlanInput struct {
	Name          string
	ClusterName   string
	K8sVersion    string
	PodCIDR       string
	SvcCIDR       string
	CNIType       string
	CNIConfig     map[string]any
	Addons        []string
	HelmInstall   bool
	StepOverrides map[string]provisiondomain.DeployPlanStepOverride
	Nodes         []DeployPlanNodeInput
}

type DeployPlanNodeInput struct {
	ServerID  uint64
	Role      string
	SortOrder int
}

// NormalizeDeployPlan owns deterministic validation and defaulting for a
// provisioning plan. Database uniqueness and host availability remain runtime
// concerns and are intentionally checked by the repository adapter.
func NormalizeDeployPlan(input DeployPlanInput, createdBy uint64) (provisiondomain.DeployPlan, []provisiondomain.DeployPlanNode, error) {
	name, clusterName, k8sVersion := strings.TrimSpace(input.Name), strings.TrimSpace(input.ClusterName), strings.TrimSpace(input.K8sVersion)
	if name == "" || clusterName == "" || k8sVersion == "" {
		return provisiondomain.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "计划名称、集群名称和 K8s 版本不能为空")
	}
	if len(clusterName) > 63 || !clusterNamePattern.MatchString(clusterName) {
		return provisiondomain.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "集群名称必须为 1-63 位小写字母、数字或连字符")
	}
	if !k8sVersionPattern.MatchString(k8sVersion) {
		return provisiondomain.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "K8s 版本格式必须类似 v1.31.0")
	}
	if !strings.HasPrefix(k8sVersion, "v") {
		k8sVersion = "v" + k8sVersion
	}
	podCIDR, err := normalizedCIDR(input.PodCIDR, "10.244.0.0/16", "Pod")
	if err != nil {
		return provisiondomain.DeployPlan{}, nil, err
	}
	svcCIDR, err := normalizedCIDR(input.SvcCIDR, "10.96.0.0/12", "Service")
	if err != nil {
		return provisiondomain.DeployPlan{}, nil, err
	}
	if CIDRsOverlap(podCIDR, svcCIDR) {
		return provisiondomain.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "Pod 网段与 Service 网段不能重叠")
	}
	cniType := strings.TrimSpace(input.CNIType)
	if cniType == "" {
		cniType = "flannel"
	}
	if cniType != "flannel" && cniType != "calico" && cniType != "cilium" {
		return provisiondomain.DeployPlan{}, nil, ErrWithMessage(ErrInvalidParams, "CNI 类型不支持")
	}
	nodes, err := normalizeDeployPlanNodes(input.Nodes)
	if err != nil {
		return provisiondomain.DeployPlan{}, nil, err
	}
	overrides, err := NormalizeDeployPlanStepOverrides(input.StepOverrides)
	if err != nil {
		return provisiondomain.DeployPlan{}, nil, err
	}
	addons, err := normalizeAddons(input.Addons)
	if err != nil {
		return provisiondomain.DeployPlan{}, nil, err
	}
	return provisiondomain.DeployPlan{
		Name: name, ClusterName: clusterName, K8sVersion: k8sVersion, PodCIDR: podCIDR, SvcCIDR: svcCIDR, CNIType: cniType,
		CNIConfig: provisiondomain.JSONMap(input.CNIConfig), Addons: provisiondomain.JSONStringSlice(addons), HelmInstall: input.HelmInstall,
		StepOverrides: provisiondomain.JSONMap(overrides), Status: "draft", CreatedBy: createdBy,
	}, nodes, nil
}

func NormalizeDeployPlanStepOverrides(input map[string]provisiondomain.DeployPlanStepOverride) (map[string]any, error) {
	if len(input) == 0 {
		return nil, nil
	}
	result := make(map[string]any, len(input))
	for entryKey, item := range input {
		key := strings.TrimSpace(entryKey)
		item.StepKey = strings.TrimSpace(item.StepKey)
		if item.StepKey == "" {
			item.StepKey = key
		}
		if item.StepKey == "" {
			return nil, ErrWithMessage(ErrInvalidParams, "步骤覆盖的 step_key 不能为空")
		}
		if key == "" {
			key = item.StepKey
			if item.NodeServerID != nil {
				key += "#server:" + strconv.FormatUint(*item.NodeServerID, 10)
			} else if item.NodeRole != "" {
				key += "#role:" + item.NodeRole
			}
		}
		item.CommandTemplate = strings.TrimSpace(item.CommandTemplate)
		if item.CommandTemplate == "" {
			return nil, ErrWithMessage(ErrInvalidParams, "步骤覆盖命令不能为空")
		}
		item.NodeRole = strings.TrimSpace(item.NodeRole)
		if item.NodeRole != "" && item.NodeRole != "master" && item.NodeRole != "worker" {
			return nil, ErrWithMessage(ErrInvalidParams, "步骤覆盖的节点角色必须为 master 或 worker")
		}
		result[key] = item
	}
	return result, nil
}

func normalizeDeployPlanNodes(input []DeployPlanNodeInput) ([]provisiondomain.DeployPlanNode, error) {
	if len(input) == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "至少选择一个部署节点")
	}
	result, seen, masterCount := make([]provisiondomain.DeployPlanNode, 0, len(input)), map[uint64]bool{}, 0
	for _, node := range input {
		if node.ServerID == 0 || seen[node.ServerID] {
			return nil, ErrWithMessage(ErrInvalidParams, "节点配置不正确")
		}
		seen[node.ServerID] = true
		role := strings.TrimSpace(node.Role)
		if role != "master" && role != "worker" {
			return nil, ErrWithMessage(ErrInvalidParams, "节点角色必须为 master 或 worker")
		}
		if role == "master" {
			masterCount++
		}
		result = append(result, provisiondomain.DeployPlanNode{ServerID: node.ServerID, Role: role, SortOrder: node.SortOrder})
	}
	if masterCount != 1 {
		return nil, ErrWithMessage(ErrInvalidParams, "当前部署模式必须且只能配置一个 master 节点")
	}
	return result, nil
}

func normalizedCIDR(value, fallback, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if _, _, err := net.ParseCIDR(value); err != nil {
		return "", ErrWithMessage(ErrInvalidParams, fmt.Sprintf("%s 网段格式不正确", label))
	}
	return value, nil
}

// CIDRsOverlap treats malformed CIDRs as conflicting. Plan normalization
// validates CIDRs before calling it, while preflight needs malformed persisted
// values to remain a blocking readiness error.
func CIDRsOverlap(left, right string) bool {
	leftIP, leftNet, leftErr := net.ParseCIDR(left)
	rightIP, rightNet, rightErr := net.ParseCIDR(right)
	if leftErr != nil || rightErr != nil || leftNet == nil || rightNet == nil {
		return true
	}
	return leftNet.Contains(rightIP) || rightNet.Contains(leftIP)
}

func normalizeAddons(input []string) ([]string, error) {
	allowed := map[string]bool{"metrics-server": true, "ingress-nginx": true, "local-storage": true}
	result, seen := make([]string, 0, len(input)), map[string]bool{}
	for _, addon := range input {
		addon = strings.TrimSpace(addon)
		if !allowed[addon] {
			return nil, ErrWithMessage(ErrInvalidParams, "包含不支持的扩展组件")
		}
		if !seen[addon] {
			result, seen[addon] = append(result, addon), true
		}
	}
	return result, nil
}
