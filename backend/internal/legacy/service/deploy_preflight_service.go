package service

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"k8s-platform-backend/internal/legacy/model"
)

const (
	preflightPassed  = "passed"
	preflightWarning = "warning"
	preflightError   = "error"
)

type DeployPreflightCheck struct {
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

type DeployPreflightResult struct {
	Ready     bool                   `json:"ready"`
	CheckedAt string                 `json:"checked_at"`
	Checks    []DeployPreflightCheck `json:"checks"`
}

func (s *DeployService) PreflightPlan(ctx context.Context, planID uint64) (DeployPreflightResult, error) {
	plan, nodes, err := s.getPlanWithNodes(ctx, planID)
	if err != nil {
		return DeployPreflightResult{}, err
	}

	result := DeployPreflightResult{Ready: true, CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	add := func(check DeployPreflightCheck) {
		if check.Ignorable && containsString(plan.PreflightIgnores, check.Key) {
			check.Ignored = true
			check.Status = preflightWarning
			check.Message += "（已人工忽略）"
		}
		result.Checks = append(result.Checks, check)
		if check.Status == preflightError {
			result.Ready = false
		}
	}

	masterCount := 0
	for _, node := range nodes {
		if node.Role == "master" {
			masterCount++
		}
	}
	if masterCount != 1 {
		add(DeployPreflightCheck{Key: "plan.topology", Category: "plan", Status: preflightError, Message: fmt.Sprintf("当前方案包含 %d 个 Master，本期仅支持且必须配置 1 个 Master", masterCount), Remediation: "保留一个 Master；高可用控制平面需要配置 VIP/LB 后使用专用 HA 流程"})
	} else {
		add(DeployPreflightCheck{Key: "plan.topology", Category: "plan", Status: preflightPassed, Message: fmt.Sprintf("拓扑有效：1 Master，%d Worker", len(nodes)-1)})
	}

	if cidrsOverlap(plan.PodCIDR, plan.SvcCIDR) {
		add(DeployPreflightCheck{Key: "plan.network", Category: "plan", Status: preflightError, Message: "Pod 网段与 Service 网段重叠", Remediation: "修改两个 CIDR，确保地址范围互不重叠且不与主机网络冲突"})
	} else {
		add(DeployPreflightCheck{Key: "plan.network", Category: "plan", Status: preflightPassed, Message: "Pod 与 Service 网段格式正确且互不重叠"})
	}

	add(DeployPreflightCheck{Key: "controller.runner", Category: "controller", Status: preflightPassed, Message: "将使用首个 Master 作为临时 Ansible Runner，缺失的 Ansible 将在执行时安装"})

	if info, statErr := os.Stat(s.ansiblePlaybookPath()); statErr != nil || info.IsDir() {
		add(DeployPreflightCheck{Key: "controller.playbook", Category: "controller", Status: preflightError, Message: "未找到可执行的 ansible/site.yml", Remediation: "检查后端 ANSIBLE_DIR 配置和部署包中的 ansible 目录"})
	} else {
		add(DeployPreflightCheck{Key: "controller.playbook", Category: "controller", Status: preflightPassed, Message: "Playbook 文件可读取"})
	}

	for _, node := range nodes {
		s.checkPreflightNode(ctx, node, add)
	}
	return result, nil
}

func (s *DeployService) SetPreflightIgnore(ctx context.Context, planID uint64, key string, ignored bool) error {
	plan, nodes, err := s.getPlanWithNodes(ctx, planID)
	if err != nil {
		return err
	}
	key = strings.TrimSpace(key)
	allowed := false
	for _, node := range nodes {
		if key == fmt.Sprintf("node.%d.disk", node.ServerID) {
			allowed = true
			break
		}
	}
	if !allowed {
		return ErrWithMessage(ErrInvalidParams, "该预检项不存在或不允许人工忽略")
	}
	items := append([]string(nil), plan.PreflightIgnores...)
	if ignored && !containsString(items, key) {
		items = append(items, key)
	}
	if !ignored {
		filtered := items[:0]
		for _, item := range items {
			if item != key {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	return s.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("deleted_at IS NULL AND id = ?", planID).Update("preflight_ignores", model.JSONStringSlice(items)).Error
}

func (s *DeployService) checkPreflightNode(ctx context.Context, node model.DeployPlanNode, add func(DeployPreflightCheck)) {
	server, credential, authType, err := s.getServerCredentialForNode(ctx, node.ServerID)
	base := DeployPreflightCheck{Key: fmt.Sprintf("node.%d.ssh", node.ServerID), Category: "node", ServerID: &node.ServerID, ServerName: server.Name}
	if err != nil {
		base.Status = preflightError
		base.Message = err.Error()
		base.Remediation = "检查服务器关联的 SSH 凭据"
		add(base)
		return
	}
	server.AuthType = authType
	probe, probeErr := probeSSH(ctx, server, credential)
	if probeErr != nil {
		base.Status = preflightError
		base.Message = probeErr.Error()
		base.Remediation = "检查主机网络、SSH 服务、用户名和凭据"
		add(base)
		return
	}
	base.Status = preflightPassed
	base.Message = fmt.Sprintf("SSH 可达：%s %s", probe.OS, probe.OSVersion)
	add(base)

	osCheck := DeployPreflightCheck{Key: fmt.Sprintf("node.%d.os", node.ServerID), Category: "node", ServerID: &node.ServerID, ServerName: server.Name}
	if !supportedLinux(probe.OS) {
		osCheck.Status = preflightError
		osCheck.Message = fmt.Sprintf("不支持的操作系统：%s", probe.OS)
		osCheck.Remediation = "使用 Ubuntu/Debian 或 RHEL/CentOS/Rocky/AlmaLinux 系主机"
	} else {
		osCheck.Status = preflightPassed
		osCheck.Message = fmt.Sprintf("操作系统受支持：%s %s", probe.OS, probe.OSVersion)
	}
	add(osCheck)

	cpu := DeployPreflightCheck{Key: fmt.Sprintf("node.%d.cpu", node.ServerID), Category: "node", ServerID: &node.ServerID, ServerName: server.Name, Status: preflightPassed, Message: fmt.Sprintf("CPU %d 核", uintValue(probe.CPUCores))}
	if uintValue(probe.CPUCores) < 2 {
		cpu.Status = preflightError
		cpu.Remediation = "至少提供 2 核 CPU"
	}
	add(cpu)
	memory := DeployPreflightCheck{Key: fmt.Sprintf("node.%d.memory", node.ServerID), Category: "node", ServerID: &node.ServerID, ServerName: server.Name, Status: preflightPassed, Message: fmt.Sprintf("内存 %d MiB", uint64Value(probe.MemoryMB))}
	if uint64Value(probe.MemoryMB) < 2048 {
		memory.Status = preflightError
		memory.Remediation = "至少提供 2 GiB 内存"
	}
	add(memory)
	disk := DeployPreflightCheck{Key: fmt.Sprintf("node.%d.disk", node.ServerID), Category: "node", ServerID: &node.ServerID, ServerName: server.Name, Status: preflightPassed, Message: fmt.Sprintf("根盘 %d GiB", uint64Value(probe.DiskGB)), Ignorable: true}
	if uint64Value(probe.DiskGB) < 20 {
		disk.Status = preflightError
		disk.Message = fmt.Sprintf("根盘仅 %d GiB，低于建议的 20 GiB", uint64Value(probe.DiskGB))
		disk.Remediation = "扩容根盘，或确认容量足够后人工忽略此项"
	}
	add(disk)

	client, dialErr := dialDeploySSH(ctx, server, credential)
	capability := DeployPreflightCheck{Key: fmt.Sprintf("node.%d.runtime", node.ServerID), Category: "node", ServerID: &node.ServerID, ServerName: server.Name}
	if dialErr != nil {
		capability.Status = preflightError
		capability.Message = dialErr.Error()
	} else {
		defer client.Close()
		output, commandErr := runSSHCommand(client, `command -v python3 >/dev/null 2>&1 && echo python3=ok || echo python3=missing; if [ "$(id -u)" = "0" ] || sudo -n true >/dev/null 2>&1; then echo privilege=ok; elif command -v sudo >/dev/null 2>&1; then echo privilege=password; else echo privilege=missing; fi`)
		privilegeUsable := strings.Contains(output, "privilege=ok") || (authType != "key" && strings.Contains(output, "privilege=password"))
		if commandErr != nil || !strings.Contains(output, "python3=ok") || !privilegeUsable {
			capability.Status = preflightError
			capability.Message = "缺少 Python 3 或 root/sudo 提权能力"
			capability.Remediation = "安装 python3，并使用 root 或具备 sudo 权限的账号"
		} else {
			capability.Status = preflightPassed
			capability.Message = "Python 3 与提权能力可用"
		}
	}
	add(capability)
}

func cidrsOverlap(left, right string) bool {
	leftIP, leftNet, leftErr := net.ParseCIDR(left)
	rightIP, rightNet, rightErr := net.ParseCIDR(right)
	if leftErr != nil || rightErr != nil {
		return true
	}
	return leftNet.Contains(rightIP) || rightNet.Contains(leftIP)
}

func supportedLinux(name string) bool {
	value := strings.ToLower(name)
	for _, supported := range []string{"ubuntu", "debian", "red hat", "redhat", "centos", "rocky", "alma", "kylin", "openkylin", "麒麟"} {
		if strings.Contains(value, supported) {
			return true
		}
	}
	return false
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
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
