package application

import (
	"strconv"
	"strings"
)

type AnsibleStep struct {
	Key      string
	Title    string
	PlayName string
}

func DefaultAnsibleSteps() []AnsibleStep {
	return []AnsibleStep{
		{Key: "pre_check", Title: "环境预检", PlayName: "环境预检"},
		{Key: "bootstrap", Title: "基础环境初始化", PlayName: "基础环境初始化"},
		{Key: "container_runtime", Title: "容器运行时安装", PlayName: "容器运行时安装"},
		{Key: "kubeadm_init", Title: "Kubernetes Master 初始化", PlayName: "Kubernetes Master 初始化"},
		{Key: "join_workers", Title: "Worker 节点加入集群", PlayName: "Worker 节点加入集群"},
		{Key: "install_cni", Title: "安装 CNI 网络插件", PlayName: "安装 CNI 网络插件"},
		{Key: "install_helm", Title: "安装 Helm", PlayName: "安装 Helm"},
		{Key: "install_addons", Title: "安装 Kubernetes 扩展组件", PlayName: "安装 Kubernetes 扩展组件"},
		{Key: "register", Title: "节点注册到管理平台", PlayName: "节点注册到管理平台"},
	}
}

func ComputeEnabledAnsibleSteps(retryFromStep string) []string {
	if strings.TrimSpace(retryFromStep) == "" {
		return nil
	}
	steps := DefaultAnsibleSteps()
	result := make([]string, 0, len(steps))
	started := false
	for _, step := range steps {
		if step.Key == retryFromStep {
			started = true
		}
		if started {
			result = append(result, step.Key)
		}
	}
	return result
}

func HasAnsibleRecapFailure(line string) bool {
	for _, field := range strings.Fields(line) {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 || (parts[0] != "failed" && parts[0] != "unreachable") {
			continue
		}
		count, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err == nil && count > 0 {
			return true
		}
	}
	return false
}

func ExtractAnsiblePlayName(line string) string {
	start := strings.Index(line, "[")
	end := strings.Index(line, "]")
	if start < 0 || end < 0 || end <= start {
		return ""
	}
	return line[start+1 : end]
}
