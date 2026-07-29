package application

import (
	"fmt"
	"strings"
	"time"
)

const (
	PreflightPassed  = "passed"
	PreflightWarning = "warning"
	PreflightError   = "error"
)

// PreflightCheck is the transport-neutral outcome of one deployment
// readiness rule. Runtime adapters supply host, SSH, and filesystem facts;
// this package owns how those facts are evaluated and presented.
type PreflightCheck struct {
	Key         string  `json:"key"`
	Category    string  `json:"category"`
	Status      string  `json:"status"`
	Message     string  `json:"message"`
	Remediation string  `json:"remediation,omitempty"`
	ServerID    *uint64 `json:"server_id,omitempty"`
	ServerName  string  `json:"server_name,omitempty"`
	Ignorable   bool    `json:"ignorable,omitempty"`
	Ignored     bool    `json:"ignored,omitempty"`
}

// PreflightResult is the complete deployment readiness response.
type PreflightResult struct {
	Ready     bool             `json:"ready"`
	CheckedAt string           `json:"checked_at"`
	Checks    []PreflightCheck `json:"checks"`
}

// PreflightAccumulator applies the deterministic aggregation and manual
// ignore policy while a runtime adapter gathers facts from the plan and hosts.
type PreflightAccumulator struct {
	result  PreflightResult
	ignored map[string]struct{}
}

func NewPreflightAccumulator(checkedAt time.Time, ignoredKeys []string) *PreflightAccumulator {
	ignored := make(map[string]struct{}, len(ignoredKeys))
	for _, key := range ignoredKeys {
		key = strings.TrimSpace(key)
		if key != "" {
			ignored[key] = struct{}{}
		}
	}
	return &PreflightAccumulator{
		result:  PreflightResult{Ready: true, CheckedAt: checkedAt.UTC().Format(time.RFC3339)},
		ignored: ignored,
	}
}

// Add applies a configured manual ignore only to ignorable checks. A failed
// check converted to a warning does not make the deployment unready.
func (a *PreflightAccumulator) Add(check PreflightCheck) {
	if check.Ignorable && a.IsIgnored(check.Key) {
		check.Ignored = true
		check.Status = PreflightWarning
		check.Message += "（已人工忽略）"
	}
	a.result.Checks = append(a.result.Checks, check)
	if check.Status == PreflightError {
		a.result.Ready = false
	}
}

func (a *PreflightAccumulator) AddAll(checks []PreflightCheck) {
	for _, check := range checks {
		a.Add(check)
	}
}

func (a *PreflightAccumulator) IsIgnored(key string) bool {
	_, ok := a.ignored[strings.TrimSpace(key)]
	return ok
}

func (a *PreflightAccumulator) Result() PreflightResult {
	result := a.result
	result.Checks = append([]PreflightCheck(nil), a.result.Checks...)
	return result
}

// IsPreflightIgnore reports whether a persisted ignore key applies to a
// specific check. It centralizes the matching rule used by preflight and the
// Ansible runtime.
func IsPreflightIgnore(ignoredKeys []string, key string) bool {
	key = strings.TrimSpace(key)
	for _, item := range ignoredKeys {
		if strings.TrimSpace(item) == key {
			return true
		}
	}
	return false
}

// UpdatePreflightIgnores validates and applies a disk-space ignore change.
// Persistence is deliberately left to the runtime repository adapter.
func UpdatePreflightIgnores(existing []string, nodeIDs []uint64, key string, ignored bool) ([]string, error) {
	key = strings.TrimSpace(key)
	if !isAllowedPreflightIgnore(nodeIDs, key) {
		return nil, ErrWithMessage(ErrInvalidParams, "该预检项不存在或不允许人工忽略")
	}

	result := append([]string(nil), existing...)
	if ignored {
		if !IsPreflightIgnore(result, key) {
			return append(result, key), nil
		}
		return result, nil
	}
	filtered := result[:0]
	for _, item := range result {
		if strings.TrimSpace(item) != key {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func isAllowedPreflightIgnore(nodeIDs []uint64, key string) bool {
	for _, id := range nodeIDs {
		if key == fmt.Sprintf("node.%d.disk", id) {
			return true
		}
	}
	return false
}

// PlanPreflightChecks evaluates topology and service-network facts already
// loaded by a runtime adapter.
func PlanPreflightChecks(podCIDR, serviceCIDR string, nodeRoles []string) []PreflightCheck {
	masterCount := 0
	for _, role := range nodeRoles {
		if strings.TrimSpace(role) == "master" {
			masterCount++
		}
	}
	checks := make([]PreflightCheck, 0, 2)
	if masterCount != 1 {
		checks = append(checks, PreflightCheck{
			Key: "plan.topology", Category: "plan", Status: PreflightError,
			Message:     fmt.Sprintf("当前方案包含 %d 个 Master，本期仅支持且必须配置 1 个 Master", masterCount),
			Remediation: "保留一个 Master；高可用控制平面需要配置 VIP/LB 后使用专用 HA 流程",
		})
	} else {
		checks = append(checks, PreflightCheck{
			Key: "plan.topology", Category: "plan", Status: PreflightPassed,
			Message: fmt.Sprintf("拓扑有效：1 Master，%d Worker", len(nodeRoles)-1),
		})
	}
	if CIDRsOverlap(podCIDR, serviceCIDR) {
		checks = append(checks, PreflightCheck{
			Key: "plan.network", Category: "plan", Status: PreflightError, Message: "Pod 网段与 Service 网段重叠",
			Remediation: "修改两个 CIDR，确保地址范围互不重叠且不与主机网络冲突",
		})
	} else {
		checks = append(checks, PreflightCheck{
			Key: "plan.network", Category: "plan", Status: PreflightPassed,
			Message: "Pod 与 Service 网段格式正确且互不重叠",
		})
	}
	return checks
}

func ControllerRunnerCheck() PreflightCheck {
	return PreflightCheck{
		Key: "controller.runner", Category: "controller", Status: PreflightPassed,
		Message: "将使用首个 Master 作为临时 Ansible Runner，缺失的 Ansible 将在执行时安装",
	}
}

func ControllerPlaybookCheck(available bool) PreflightCheck {
	if available {
		return PreflightCheck{Key: "controller.playbook", Category: "controller", Status: PreflightPassed, Message: "Playbook 文件可读取"}
	}
	return PreflightCheck{
		Key: "controller.playbook", Category: "controller", Status: PreflightError, Message: "未找到可执行的 ansible/site.yml",
		Remediation: "检查后端 ANSIBLE_DIR 配置和部署包中的 ansible 目录",
	}
}

// PreflightNodeProbe contains the resource facts reported by an SSH probe.
// Nil values are represented as zero so policy never depends on adapter types.
type PreflightNodeProbe struct {
	ServerID   uint64
	ServerName string
	OS         string
	OSVersion  string
	CPUCores   uint
	MemoryMB   uint64
	DiskGB     uint64
}

func NodeSSHFailureCheck(serverID uint64, serverName, message string) PreflightCheck {
	return nodeCheck(serverID, serverName, "ssh", PreflightError, message, "检查服务器关联的 SSH 凭据")
}

func NodeSSHConnectionFailureCheck(serverID uint64, serverName, message string) PreflightCheck {
	return nodeCheck(serverID, serverName, "ssh", PreflightError, message, "检查主机网络、SSH 服务、用户名和凭据")
}

func NodeSSHReachableCheck(serverID uint64, serverName, osName, osVersion string) PreflightCheck {
	return nodeCheck(serverID, serverName, "ssh", PreflightPassed, fmt.Sprintf("SSH 可达：%s %s", osName, osVersion), "")
}

// NodeProbeChecks evaluates OS support and minimum resources from probe
// results. Runtime code is responsible only for obtaining those results.
func NodeProbeChecks(probe PreflightNodeProbe) []PreflightCheck {
	checks := make([]PreflightCheck, 0, 4)
	if !SupportedLinux(probe.OS) {
		checks = append(checks, nodeCheck(probe.ServerID, probe.ServerName, "os", PreflightError,
			fmt.Sprintf("不支持的操作系统：%s", probe.OS), "使用 Ubuntu/Debian 或 RHEL/CentOS/Rocky/AlmaLinux 系主机"))
	} else {
		checks = append(checks, nodeCheck(probe.ServerID, probe.ServerName, "os", PreflightPassed,
			fmt.Sprintf("操作系统受支持：%s %s", probe.OS, probe.OSVersion), ""))
	}

	cpu := nodeCheck(probe.ServerID, probe.ServerName, "cpu", PreflightPassed, fmt.Sprintf("CPU %d 核", probe.CPUCores), "")
	if probe.CPUCores < 2 {
		cpu.Status, cpu.Remediation = PreflightError, "至少提供 2 核 CPU"
	}
	checks = append(checks, cpu)

	memory := nodeCheck(probe.ServerID, probe.ServerName, "memory", PreflightPassed, fmt.Sprintf("内存 %d MiB", probe.MemoryMB), "")
	if probe.MemoryMB < 2048 {
		memory.Status, memory.Remediation = PreflightError, "至少提供 2 GiB 内存"
	}
	checks = append(checks, memory)

	disk := nodeCheck(probe.ServerID, probe.ServerName, "disk", PreflightPassed, fmt.Sprintf("根盘 %d GiB", probe.DiskGB), "")
	disk.Ignorable = true
	if probe.DiskGB < 20 {
		disk.Status = PreflightError
		disk.Message = fmt.Sprintf("根盘仅 %d GiB，低于建议的 20 GiB", probe.DiskGB)
		disk.Remediation = "扩容根盘，或确认容量足够后人工忽略此项"
	}
	checks = append(checks, disk)
	return checks
}

type PreflightNodeRuntime struct {
	ServerID          uint64
	ServerName        string
	ConnectionMessage string
	CommandFailed     bool
	Python3Available  bool
	PrivilegeUsable   bool
}

func NodeRuntimeCheck(input PreflightNodeRuntime) PreflightCheck {
	if strings.TrimSpace(input.ConnectionMessage) != "" {
		return nodeCheck(input.ServerID, input.ServerName, "runtime", PreflightError, input.ConnectionMessage, "")
	}
	if input.CommandFailed || !input.Python3Available || !input.PrivilegeUsable {
		return nodeCheck(input.ServerID, input.ServerName, "runtime", PreflightError,
			"缺少 Python 3 或 root/sudo 提权能力", "安装 python3，并使用 root 或具备 sudo 权限的账户")
	}
	return nodeCheck(input.ServerID, input.ServerName, "runtime", PreflightPassed, "Python 3 与提权能力可用", "")
}

func nodeCheck(serverID uint64, serverName, suffix, status, message, remediation string) PreflightCheck {
	return PreflightCheck{
		Key: fmt.Sprintf("node.%d.%s", serverID, suffix), Category: "node", Status: status,
		Message: message, Remediation: remediation, ServerID: &serverID, ServerName: serverName,
	}
}

// SupportedLinux is the platform support policy for provisioning targets.
func SupportedLinux(name string) bool {
	value := strings.ToLower(name)
	for _, supported := range []string{"ubuntu", "debian", "red hat", "redhat", "centos", "rocky", "alma", "kylin", "openkylin", "麒麟"} {
		if strings.Contains(value, supported) {
			return true
		}
	}
	return false
}
