package service

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"k8s-platform-backend/internal/model"
)

// deployPipeline 执行 K8s 集群部署流水线
func (s *DeployService) deployPipeline(ctx context.Context, planID uint64, taskID int64) {
	task, ok := s.taskStore.Get(taskID)
	if !ok {
		return
	}

	// 注册取消函数
	ctx, cancel := context.WithCancel(ctx)
	s.taskStore.RegisterCancel(taskID, cancel)
	defer s.taskStore.UnregisterCancel(taskID)

	// 更新任务状态为运行中
	task.Status = TaskRunning
	msg := "正在执行部署流水线"
	task.Message = &msg
	_ = s.taskStore.Put(task)

	// 定义流水线步骤
	steps := []struct {
		key   string
		title string
		fn    func(ctx context.Context, plan model.DeployPlan, nodes []model.DeployPlanNode, task *Task) error
	}{
		{"pre_check", "环境预检", s.stepPreCheck},
		{"bootstrap", "基础环境初始化", s.stepBootstrap},
		{"init_master", "初始化 Master", s.stepInitMaster},
		{"join_workers", "Worker 节点加入", s.stepJoinWorkers},
		{"install_cni", "安装网络插件", s.stepInstallCNI},
		{"register", "注册集群", s.stepRegisterCluster},
	}

	// 初始化步骤状态
	task.Steps = make([]TaskStep, len(steps))
	for i, step := range steps {
		task.Steps[i] = TaskStep{Key: step.key, Title: step.title, Status: StepPending}
	}
	_ = s.taskStore.Put(task)

	// 获取部署计划和节点
	plan, nodes, err := s.getPlanWithNodes(ctx, planID)
	if err != nil {
		s.markTaskFailed(task, fmt.Sprintf("获取部署计划失败: %v", err))
		s.updatePlanStatusDirect(ctx, planID, "failed")
		return
	}

	// 检查是否被取消
	if ctx.Err() != nil {
		s.markTaskCanceled(task)
		s.updatePlanStatusDirect(ctx, planID, "cancelled")
		s.rollbackNodes(ctx, nodes, task)
		return
	}

	// 执行流水线步骤
	totalSteps := len(steps)
	for i, step := range steps {
		// 检查取消
		if ctx.Err() != nil {
			task.Steps[i].Status = StepFailed
			msg := "任务已取消"
			task.Steps[i].Message = &msg
			s.markTaskCanceled(task)
			s.updatePlanStatusDirect(ctx, planID, "cancelled")
			s.rollbackNodes(ctx, nodes, task)
			return
		}

		// 更新当前步骤为运行中
		task.Steps[i].Status = StepRunning
		stepNow := time.Now().UTC()
		task.Steps[i].StartedAt = &stepNow
		percent := i * 100 / totalSteps
		task.Percent = &percent
		msg := fmt.Sprintf("正在执行: %s", step.title)
		task.Message = &msg
		_ = s.taskStore.Put(task)

		// 执行步骤
		if err := step.fn(ctx, plan, nodes, task); err != nil {
			task.Steps[i].Status = StepFailed
			errMsg := err.Error()
			task.Steps[i].Message = &errMsg
			stepDone := time.Now().UTC()
			task.Steps[i].FinishedAt = &stepDone
			s.markTaskFailed(task, fmt.Sprintf("步骤[%s]失败: %v", step.title, err))
			s.updatePlanStatusDirect(ctx, planID, "failed")
			s.rollbackNodes(ctx, nodes, task)
			return
		}

		// 步骤成功
		task.Steps[i].Status = StepSuccess
		stepDone := time.Now().UTC()
		task.Steps[i].FinishedAt = &stepDone
		successMsg := "完成"
		task.Steps[i].Message = &successMsg
		_ = s.taskStore.Put(task)
	}

	// 全部成功
	percent := 100
	task.Percent = &percent
	task.Status = TaskSuccess
	successMsg := "集群部署成功"
	task.Message = &successMsg
	_ = s.taskStore.Put(task)
	s.updatePlanStatusDirect(ctx, planID, "success")
}

func (s *DeployService) markTaskFailed(task *Task, msg string) {
	task.Status = TaskFailed
	task.Message = &msg
	_ = s.taskStore.Put(task)
}

func (s *DeployService) markTaskCanceled(task *Task) {
	task.Status = TaskCanceled
	msg := "任务已取消"
	task.Message = &msg
	_ = s.taskStore.Put(task)
}

func (s *DeployService) updatePlanStatusDirect(ctx context.Context, planID uint64, status string) {
	_ = s.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("id = ?", planID).Update("status", status).Error
}

// rollbackNodes 回滚清理：在部署失败/取消时执行 kubeadm reset 清理节点
func (s *DeployService) rollbackNodes(ctx context.Context, nodes []model.DeployPlanNode, task *Task) {
	task.AppendLog("[warn] 开始执行回滚清理...")
	_ = s.taskStore.Put(task)

	for _, node := range nodes {
		server, cred, err := s.getServerAndCredential(ctx, node.ServerID)
		if err != nil {
			task.AppendLog(fmt.Sprintf("[warn] 回滚: 获取服务器 %d 信息失败: %v", node.ServerID, err))
			continue
		}

		client, err := s.connectSSHDirect(ctx, server, cred)
		if err != nil {
			task.AppendLog(fmt.Sprintf("[warn] 回滚: SSH 连接服务器 %s(%s) 失败: %v", server.Name, server.IP, err))
			continue
		}

		// 执行 kubeadm reset
		task.AppendLog(fmt.Sprintf("[info] 回滚: 正在清理服务器 %s(%s)...", server.Name, server.IP))
		_, _ = runSSHCommand(client, "sudo kubeadm reset -f 2>/dev/null || true")

		// 清理 iptables 规则
		_, _ = runSSHCommand(client, "sudo iptables -F && sudo iptables -t nat -F && sudo iptables -t mangle -F && sudo iptables -X 2>/dev/null || true")

		// 清理 IPVS 规则
		_, _ = runSSHCommand(client, "sudo ipvsadm --clear 2>/dev/null || true")

		// 清理 CNI 配置
		_, _ = runSSHCommand(client, "sudo rm -rf /etc/cni/net.d/* 2>/dev/null || true")

		// 停止 kubelet
		_, _ = runSSHCommand(client, "sudo systemctl stop kubelet 2>/dev/null || true")

		client.Close()
		task.AppendLog(fmt.Sprintf("[info] 回滚: 服务器 %s(%s) 清理完成", server.Name, server.IP))
		_ = s.taskStore.Put(task)
	}

	task.AppendLog("[info] 回滚清理已完成")
	_ = s.taskStore.Put(task)
}

// stepPreCheck 环境预检：SSH 连通性、系统版本、CPU/内存/磁盘、端口、内核参数、hostname、时间同步
func (s *DeployService) stepPreCheck(ctx context.Context, plan model.DeployPlan, nodes []model.DeployPlanNode, task *Task) error {
	// 收集所有节点的 hostname 用于唯一性检查
	hostnames := make(map[string]string) // hostname -> serverIP

	for _, node := range nodes {
		server, cred, err := s.getServerAndCredential(ctx, node.ServerID)
		if err != nil {
			return fmt.Errorf("获取服务器 %d 信息失败: %w", node.ServerID, err)
		}

		client, err := s.connectSSHDirect(ctx, server, cred)
		if err != nil {
			return fmt.Errorf("SSH 连接服务器 %s(%s) 失败: %w", server.Name, server.IP, err)
		}

		resolved, err := s.resolvePlanStep(ctx, plan, "pre_check", node, server, deployTemplateData{
			PlanID:       plan.ID,
			PlanName:     plan.Name,
			ClusterName:  plan.ClusterName,
			K8sVersion:   plan.K8sVersion,
			MinorVersion: strings.TrimPrefix(plan.K8sVersion, "v")[:strings.LastIndex(strings.TrimPrefix(plan.K8sVersion, "v"), ".")],
			PodCIDR:      plan.PodCIDR,
			SvcCIDR:      plan.SvcCIDR,
			CNICommand:   resolveCNICommand(plan.CNIType),
			NodeRole:     node.Role,
			ServerID:     node.ServerID,
			ServerIP:     server.IP,
		})
		if err != nil {
			client.Close()
			return fmt.Errorf("解析 pre_check 命令失败: %w", err)
		}
		if len(resolved.Commands) == 0 {
			client.Close()
			return fmt.Errorf("pre_check 没有可执行命令")
		}

		output, err := runSSHCommand(client, resolved.Commands[0])
		if err != nil {
			client.Close()
			return fmt.Errorf("执行 pre_check 命令失败: %w", err)
		}
		precheck := parseDeployStepKeyValueOutput(output)

		if value := strings.TrimSpace(precheck["OS"]); value != "" {
			server.OS = &value
			task.AppendLog(fmt.Sprintf("[info] 服务器 %s OS: %s", server.IP, value))
		}
		if value := strings.TrimSpace(precheck["OS_VERSION"]); value != "" && value != "unknown" {
			server.OSVersion = &value
		}
		if value := strings.TrimSpace(precheck["KERNEL"]); value != "" {
			server.Kernel = &value
		}

		if cpuCores, ok := parseUintValue(precheck["CPU_CORES"]); ok {
			if cpuCores < 2 {
				client.Close()
				return fmt.Errorf("服务器 %s CPU 核心数不足: %d (最低要求 2 核)", server.IP, cpuCores)
			}
			cpu := uint(cpuCores)
			server.CPUCores = &cpu
			task.AppendLog(fmt.Sprintf("[info] 服务器 %s CPU: %d 核", server.IP, cpuCores))
		}

		if memMB, ok := parseUintValue(precheck["MEMORY_MB"]); ok {
			if memMB < 2048 {
				client.Close()
				return fmt.Errorf("服务器 %s 内存不足: %dMB (最低要求 2048MB)", server.IP, memMB)
			}
			server.MemoryMB = &memMB
			task.AppendLog(fmt.Sprintf("[info] 服务器 %s 内存: %dMB", server.IP, memMB))
		}

		if diskGB, ok := parseUintValue(precheck["DISK_GB"]); ok {
			if diskGB < 20 {
				client.Close()
				return fmt.Errorf("服务器 %s 可用磁盘空间不足: %dGB (最低要求 20GB)", server.IP, diskGB)
			}
			server.DiskGB = &diskGB
			task.AppendLog(fmt.Sprintf("[info] 服务器 %s 可用磁盘: %dGB", server.IP, diskGB))
		}

		hostname := strings.TrimSpace(precheck["HOSTNAME"])
		if hostname != "" {
			if existingIP, exists := hostnames[hostname]; exists {
				client.Close()
				return fmt.Errorf("hostname 冲突: 服务器 %s 和 %s 使用相同的 hostname '%s'", existingIP, server.IP, hostname)
			}
			hostnames[hostname] = server.IP
		}

		if strings.TrimSpace(precheck["SWAP_LINES"]) != "0" {
			task.AppendLog(fmt.Sprintf("[warn] 服务器 %s swap 未关闭，将在 Bootstrap 阶段关闭", server.IP))
		}

		if node.Role == "master" {
			ports := []string{"6443", "2379", "2380", "10250", "10259", "10257"}
			for _, port := range ports {
				key := "PORT_" + port
				if strings.TrimSpace(precheck[key]) != "0" {
					client.Close()
					return fmt.Errorf("服务器 %s 端口 %s 已被占用", server.IP, port)
				}
			}
		} else {
			if strings.TrimSpace(precheck["PORT_10250"]) != "0" {
				client.Close()
				return fmt.Errorf("服务器 %s 端口 10250 已被占用", server.IP)
			}
		}

		kernelParams := []string{
			"net.bridge.bridge-nf-call-iptables",
			"net.bridge.bridge-nf-call-ip6tables",
			"net.ipv4.ip_forward",
		}
		for _, param := range kernelParams {
			key := map[string]string{
				"net.bridge.bridge-nf-call-iptables":  "BR_NETFILTER",
				"net.bridge.bridge-nf-call-ip6tables": "BR_NETFILTER_IPV6",
				"net.ipv4.ip_forward":                 "IP_FORWARD",
			}[param]
			val := strings.TrimSpace(precheck[key])
			if val != "1" {
				task.AppendLog(fmt.Sprintf("[warn] 服务器 %s 内核参数 %s=%s (将在 Bootstrap 阶段配置)", server.IP, param, val))
			}
		}

		ntpOutput := strings.TrimSpace(precheck["NTP_SYNC"])
		if ntpOutput == "no" {
			task.AppendLog(fmt.Sprintf("[warn] 服务器 %s NTP 时间同步未启用", server.IP))
		}

		if containerRuntime := strings.TrimSpace(precheck["CONTAINERD"]); containerRuntime == "installed" {
			task.AppendLog(fmt.Sprintf("[info] 服务器 %s 已安装 containerd", server.IP))
		}
		if kubeadmVersion := strings.TrimSpace(precheck["KUBEADM"]); kubeadmVersion != "" && kubeadmVersion != "missing" {
			task.AppendLog(fmt.Sprintf("[info] 服务器 %s 已安装 kubeadm: %s", server.IP, kubeadmVersion))
		}
		if selinux := strings.TrimSpace(precheck["SELINUX"]); selinux != "" && selinux != "unknown" && selinux != "Disabled" {
			task.AppendLog(fmt.Sprintf("[warn] 服务器 %s SELinux 当前状态: %s (将在 Bootstrap 阶段调整)", server.IP, selinux))
		}
		if firewalld := strings.TrimSpace(precheck["FIREWALLD"]); firewalld == "active" {
			task.AppendLog(fmt.Sprintf("[warn] 服务器 %s firewalld 仍在运行 (将在 Bootstrap 阶段关闭)", server.IP))
		}

		// 更新服务器系统信息
		s.db.WithContext(ctx).Model(&model.DeployServer{}).Where("id = ?", server.ID).Updates(map[string]any{
			"os":         server.OS,
			"os_version": server.OSVersion,
			"kernel":     server.Kernel,
			"cpu_cores":  server.CPUCores,
			"memory_mb":  server.MemoryMB,
			"disk_gb":    server.DiskGB,
		})

		client.Close()
		task.AppendLog(fmt.Sprintf("[info] 服务器 %s(%s) 预检通过", server.Name, server.IP))
		_ = s.taskStore.Put(task)
	}
	return nil
}

func parseDeployStepKeyValueOutput(output string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		result[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return result
}

func parseUintValue(raw string) (uint64, bool) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return 0, false
	}
	value, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// stepBootstrap 基础环境初始化
func (s *DeployService) stepBootstrap(ctx context.Context, plan model.DeployPlan, nodes []model.DeployPlanNode, task *Task) error {
	for _, node := range nodes {
		server, cred, err := s.getServerAndCredential(ctx, node.ServerID)
		if err != nil {
			return err
		}

		client, err := s.connectSSHDirect(ctx, server, cred)
		if err != nil {
			return fmt.Errorf("SSH 连接失败: %w", err)
		}
		resolved, err := s.resolvePlanStep(ctx, plan, "bootstrap", node, server, deployTemplateData{
			PlanID:       plan.ID,
			PlanName:     plan.Name,
			ClusterName:  plan.ClusterName,
			K8sVersion:   plan.K8sVersion,
			MinorVersion: strings.TrimPrefix(plan.K8sVersion, "v")[:strings.LastIndex(strings.TrimPrefix(plan.K8sVersion, "v"), ".")],
			PodCIDR:      plan.PodCIDR,
			SvcCIDR:      plan.SvcCIDR,
			CNICommand:   resolveCNICommand(plan.CNIType),
			NodeRole:     node.Role,
			ServerID:     node.ServerID,
			ServerIP:     server.IP,
		})
		if err != nil {
			client.Close()
			return fmt.Errorf("解析 bootstrap 命令失败: %w", err)
		}
		for _, command := range resolved.Commands {
			if _, err := runSSHCommand(client, command); err != nil {
				client.Close()
				return fmt.Errorf("执行 bootstrap 命令失败: %w", err)
			}
		}

		client.Close()
		task.AppendLog(fmt.Sprintf("[info] 服务器 %s(%s) 基础环境初始化完成", server.Name, server.IP))
		_ = s.taskStore.Put(task)
	}
	return nil
}

// stepInitMaster 初始化 Master 节点
func (s *DeployService) stepInitMaster(ctx context.Context, plan model.DeployPlan, nodes []model.DeployPlanNode, task *Task) error {
	masterNode := findMasterNode(nodes)
	if masterNode == nil {
		return fmt.Errorf("未找到 Master 节点")
	}

	server, cred, err := s.getServerAndCredential(ctx, masterNode.ServerID)
	if err != nil {
		return err
	}

	client, err := s.connectSSHDirect(ctx, server, cred)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer client.Close()

	resolved, err := s.resolvePlanStep(ctx, plan, "init_master", *masterNode, server, deployTemplateData{
		PlanID:       plan.ID,
		PlanName:     plan.Name,
		ClusterName:  plan.ClusterName,
		K8sVersion:   plan.K8sVersion,
		MinorVersion: strings.TrimPrefix(plan.K8sVersion, "v")[:strings.LastIndex(strings.TrimPrefix(plan.K8sVersion, "v"), ".")],
		PodCIDR:      plan.PodCIDR,
		SvcCIDR:      plan.SvcCIDR,
		MasterIP:     server.IP,
		CNICommand:   resolveCNICommand(plan.CNIType),
		NodeRole:     masterNode.Role,
		ServerID:     masterNode.ServerID,
		ServerIP:     server.IP,
	})
	if err != nil {
		return fmt.Errorf("解析 init_master 命令失败: %w", err)
	}
	if len(resolved.Commands) == 0 {
		return fmt.Errorf("init_master 没有可执行命令")
	}
	output, err := runSSHCommand(client, resolved.Commands[0])
	if err != nil {
		_, _ = runSSHCommand(client, "sudo kubeadm reset -f")
		return fmt.Errorf("kubeadm init 失败: %w\n%s", err, output)
	}
	for _, command := range resolved.Commands[1:] {
		if _, err := runSSHCommand(client, command); err != nil {
			return fmt.Errorf("执行 init_master 后续命令失败: %w", err)
		}
	}
	joinOutput, err := runSSHCommand(client, "sudo kubeadm token create --print-join-command")
	if err != nil {
		return fmt.Errorf("获取 join token 失败: %w", err)
	}

	// 保存 join 命令到任务元数据
	if task.Meta == nil {
		task.Meta = make(map[string]any)
	}
	task.Meta["join_command"] = strings.TrimSpace(joinOutput)
	_ = s.taskStore.Put(task)

	task.AppendLog(fmt.Sprintf("[info] Master 节点 %s 初始化完成", server.IP))
	return nil
}

// stepJoinWorkers Worker 节点加入集群
func (s *DeployService) stepJoinWorkers(ctx context.Context, plan model.DeployPlan, nodes []model.DeployPlanNode, task *Task) error {
	joinCmd, ok := task.Meta["join_command"].(string)
	if !ok || joinCmd == "" {
		return fmt.Errorf("未找到 join 命令")
	}

	workerNodes := findWorkerNodes(nodes)
	for _, node := range workerNodes {
		server, cred, err := s.getServerAndCredential(ctx, node.ServerID)
		if err != nil {
			return err
		}

		client, err := s.connectSSHDirect(ctx, server, cred)
		if err != nil {
			return fmt.Errorf("SSH 连接 Worker %s 失败: %w", server.IP, err)
		}
		resolved, err := s.resolvePlanStep(ctx, plan, "join_workers", node, server, deployTemplateData{
			PlanID:       plan.ID,
			PlanName:     plan.Name,
			ClusterName:  plan.ClusterName,
			K8sVersion:   plan.K8sVersion,
			MinorVersion: strings.TrimPrefix(plan.K8sVersion, "v")[:strings.LastIndex(strings.TrimPrefix(plan.K8sVersion, "v"), ".")],
			PodCIDR:      plan.PodCIDR,
			SvcCIDR:      plan.SvcCIDR,
			JoinCommand:  joinCmd,
			CNICommand:   resolveCNICommand(plan.CNIType),
			NodeRole:     node.Role,
			ServerID:     node.ServerID,
			ServerIP:     server.IP,
		})
		if err != nil {
			client.Close()
			return fmt.Errorf("解析 join_workers 命令失败: %w", err)
		}
		if len(resolved.Commands) == 0 {
			client.Close()
			return fmt.Errorf("join_workers 没有可执行命令")
		}
		if _, err := runSSHCommand(client, resolved.Commands[0]); err != nil {
			client.Close()
			return fmt.Errorf("Worker %s 加入集群失败: %w", server.IP, err)
		}

		client.Close()
		task.AppendLog(fmt.Sprintf("[info] Worker %s 已加入集群", server.IP))
		_ = s.taskStore.Put(task)
	}
	return nil
}

// stepInstallCNI 安装网络插件
func (s *DeployService) stepInstallCNI(ctx context.Context, plan model.DeployPlan, nodes []model.DeployPlanNode, task *Task) error {
	masterNode := findMasterNode(nodes)
	if masterNode == nil {
		return fmt.Errorf("未找到 Master 节点")
	}

	server, cred, err := s.getServerAndCredential(ctx, masterNode.ServerID)
	if err != nil {
		return err
	}

	client, err := s.connectSSHDirect(ctx, server, cred)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer client.Close()

	resolved, err := s.resolvePlanStep(ctx, plan, "install_cni", *masterNode, server, deployTemplateData{
		PlanID:       plan.ID,
		PlanName:     plan.Name,
		ClusterName:  plan.ClusterName,
		K8sVersion:   plan.K8sVersion,
		MinorVersion: strings.TrimPrefix(plan.K8sVersion, "v")[:strings.LastIndex(strings.TrimPrefix(plan.K8sVersion, "v"), ".")],
		PodCIDR:      plan.PodCIDR,
		SvcCIDR:      plan.SvcCIDR,
		MasterIP:     server.IP,
		CNICommand:   resolveCNICommand(plan.CNIType),
		NodeRole:     masterNode.Role,
		ServerID:     masterNode.ServerID,
		ServerIP:     server.IP,
	})
	if err != nil {
		return fmt.Errorf("解析 install_cni 命令失败: %w", err)
	}
	for _, command := range resolved.Commands {
		if _, err := runSSHCommand(client, command); err != nil {
			return fmt.Errorf("安装 CNI(%s) 失败: %w", plan.CNIType, err)
		}
	}

	task.AppendLog(fmt.Sprintf("[info] CNI(%s) 安装完成", plan.CNIType))
	return nil
}

// stepRegisterCluster 提取 kubeconfig 并注册到平台
func (s *DeployService) stepRegisterCluster(ctx context.Context, plan model.DeployPlan, nodes []model.DeployPlanNode, task *Task) error {
	masterNode := findMasterNode(nodes)
	if masterNode == nil {
		return fmt.Errorf("未找到 Master 节点")
	}

	server, cred, err := s.getServerAndCredential(ctx, masterNode.ServerID)
	if err != nil {
		return err
	}

	client, err := s.connectSSHDirect(ctx, server, cred)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer client.Close()

	resolved, err := s.resolvePlanStep(ctx, plan, "register", *masterNode, server, deployTemplateData{
		PlanID:       plan.ID,
		PlanName:     plan.Name,
		ClusterName:  plan.ClusterName,
		K8sVersion:   plan.K8sVersion,
		MinorVersion: strings.TrimPrefix(plan.K8sVersion, "v")[:strings.LastIndex(strings.TrimPrefix(plan.K8sVersion, "v"), ".")],
		PodCIDR:      plan.PodCIDR,
		SvcCIDR:      plan.SvcCIDR,
		MasterIP:     server.IP,
		CNICommand:   resolveCNICommand(plan.CNIType),
		NodeRole:     masterNode.Role,
		ServerID:     masterNode.ServerID,
		ServerIP:     server.IP,
	})
	if err != nil {
		return fmt.Errorf("解析 register 命令失败: %w", err)
	}
	if len(resolved.Commands) == 0 {
		return fmt.Errorf("register 没有可执行命令")
	}
	kubeconfig, err := runSSHCommand(client, resolved.Commands[0])
	if err != nil {
		return fmt.Errorf("提取 kubeconfig 失败: %w", err)
	}

	// 替换 server 地址
	kubeconfig = strings.Replace(kubeconfig, "https://127.0.0.1:6443", fmt.Sprintf("https://%s:6443", server.IP), 1)

	// 使用 ClusterRegistryService 注册集群
	clusterID, err := s.clusterRegistry.ImportCluster(ctx, plan.ClusterName, kubeconfig)
	if err != nil {
		return fmt.Errorf("注册集群失败: %w", err)
	}

	// 更新部署计划的 cluster_id
	_ = s.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("id = ?", plan.ID).Update("cluster_id", clusterID).Error

	task.AppendLog(fmt.Sprintf("[info] 集群 %s 注册成功，ID: %d", plan.ClusterName, clusterID))
	return nil
}

// 辅助函数

func (s *DeployService) getServerAndCredential(ctx context.Context, serverID uint64) (model.DeployServer, string, error) {
	var server model.DeployServer
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", serverID).First(&server).Error; err != nil {
		return model.DeployServer{}, "", fmt.Errorf("服务器不存在: %w", err)
	}

	cred, err := decryptText(s.encryptionKey, server.CredentialEnc)
	if err != nil {
		return model.DeployServer{}, "", fmt.Errorf("解密凭证失败: %w", err)
	}

	return server, cred, nil
}

func (s *DeployService) connectSSHDirect(ctx context.Context, server model.DeployServer, credential string) (*ssh.Client, error) {
	auth, err := buildSSHAuth(server.AuthType, credential)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User:            server.User,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := net.JoinHostPort(server.IP, fmt.Sprintf("%d", server.SSHPort))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("TCP 连接失败: %w", err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SSH 握手失败: %w", err)
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

func findMasterNode(nodes []model.DeployPlanNode) *model.DeployPlanNode {
	for _, node := range nodes {
		if node.Role == "master" {
			return &node
		}
	}
	return nil
}

func findWorkerNodes(nodes []model.DeployPlanNode) []model.DeployPlanNode {
	var workers []model.DeployPlanNode
	for _, node := range nodes {
		if node.Role == "worker" {
			workers = append(workers, node)
		}
	}
	return workers
}
