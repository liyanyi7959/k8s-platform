package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

// DryRunStep 表示模拟运行的单个步骤
type DryRunStep struct {
	Key         string   `json:"key"`         // 步骤唯一键
	Title       string   `json:"title"`       // 步骤标题
	Description string   `json:"description"` // 步骤说明
	Phase       string   `json:"phase"`       // preflight/install/init/join/addon/finalize
	Tasks       []string `json:"tasks"`       // Ansible task 描述列表
	AppliesTo   string   `json:"applies_to"`  // all / master / worker
	DependsOn   []string `json:"depends_on,omitempty"`
}

// DryRunNodeFlow 单台机器的部署流程
type DryRunNodeFlow struct {
	ServerID   uint64       `json:"server_id"`
	ServerName string       `json:"server_name"`
	IP         string       `json:"ip"`
	Role       string       `json:"role"`
	Steps      []DryRunStep `json:"steps"`
}

// DryRunResult 整个计划的模拟运行结果
type DryRunResult struct {
	PlanID      uint64           `json:"plan_id"`
	PlanName    string           `json:"plan_name"`
	ClusterName string           `json:"cluster_name"`
	K8sVersion  string           `json:"k8s_version"`
	CNIType     string           `json:"cni_type"`
	Nodes       []DryRunNodeFlow `json:"nodes"`
	Summary     map[string]int   `json:"summary"` // 各阶段步骤数统计
}

// ansibleDryRunStepDef 定义 Ansible 部署步骤元数据
type ansibleDryRunStepDef struct {
	Key         string
	Title       string
	Description string
	Phase       string
	Tasks       []string
	AppliesTo   string // all / master / worker
	DependsOn   []string
}

// ansibleDryRunSteps Ansible 部署流水线步骤定义
var ansibleDryRunSteps = []ansibleDryRunStepDef{
	{
		Key:         "pre_check",
		Title:       "环境预检",
		Description: "检查目标节点的 CPU 核数、内存容量、磁盘空间、端口占用、hostname 唯一性等环境要求",
		Phase:       "preflight",
		AppliesTo:   "all",
		Tasks: []string{
			"Ping 测试 SSH 连通性",
			"检查 OS 发行版和版本号",
			"检查 CPU 核数 >= 2",
			"检查内存 >= 2048MB",
			"检查磁盘可用空间 >= 20GB",
			"检查 hostname 是否设置且唯一",
			"Master 节点检查端口 6443/2379/2380/10250/10259/10257 未被占用",
		},
	},
	{
		Key:         "bootstrap",
		Title:       "基础环境初始化",
		Description: "关闭 swap、加载内核模块（br_netfilter/overlay/nf_conntrack）、设置 sysctl 参数、禁用防火墙和 SELinux、安装基础依赖包、配置时间同步",
		Phase:       "install",
		AppliesTo:   "all",
		Tasks: []string{
			"关闭 swap 并注释 /etc/fstab 中的 swap 行",
			"加载内核模块 br_netfilter, overlay, nf_conntrack 并持久化",
			"设置 sysctl: net.bridge.bridge-nf-call-iptables=1, net.ipv4.ip_forward=1 等",
			"停止并禁用 firewalld",
			"设置 SELinux 为 permissive/disabled",
			"安装基础包: device-mapper-persistent-data, lvm2, wget, curl, vim, chrony",
			"配置国内 yum/apt 镜像源（来自仓库配置）",
			"自动禁用 SSL 证书有问题的仓库",
			"配置 Kubernetes yum/apt 仓库",
		},
	},
	{
		Key:         "container_runtime",
		Title:       "容器运行时安装",
		Description: "安装 containerd 容器运行时，配置 SystemdCgroup 和 sandbox 镜像，启动并设置开机自启",
		Phase:       "install",
		AppliesTo:   "all",
		Tasks: []string{
			"安装 Docker CE 仓库（使用国内镜像源）",
			"安装 containerd.io",
			"生成 containerd 默认配置",
			"设置 SystemdCgroup=true",
			"设置 sandbox_image=registry.k8s.io/pause:3.9",
			"启动 containerd 并设置开机自启",
		},
	},
	{
		Key:         "kubeadm_init",
		Title:       "Kubernetes Master 初始化",
		Description: "在 Master 节点安装 kubeadm/kubelet/kubectl，生成 kubeadm 配置文件，执行 kubeadm init 初始化集群，配置 kubeconfig，生成 worker 节点的 join 命令",
		Phase:       "init",
		AppliesTo:   "master",
		Tasks: []string{
			"安装 kubeadm, kubelet, kubectl（版本 " + "{{k8s_version}}" + "）",
			"锁定版本不自动更新",
			"生成 kubeadm-init.yaml 配置文件",
			"执行 kubeadm init --config /tmp/kubeadm-init.yaml",
			"配置 kubeconfig (~/.kube/config)",
			"生成 worker join 命令 (kubeadm token create --print-join-command)",
		},
		DependsOn: []string{"bootstrap", "container_runtime"},
	},
	{
		Key:         "join_workers",
		Title:       "Worker 节点加入集群",
		Description: "在 Worker 节点安装 kubeadm/kubelet，执行 kubeadm join 加入 Kubernetes 集群",
		Phase:       "join",
		AppliesTo:   "worker",
		Tasks: []string{
			"安装 kubeadm, kubelet（版本 " + "{{k8s_version}}" + "）",
			"锁定版本不自动更新",
			"执行 kubeadm join <master_ip>:6443 --token <token> --discovery-token-ca-cert-hash <hash>",
		},
		DependsOn: []string{"kubeadm_init"},
	},
	{
		Key:         "install_cni",
		Title:       "安装 CNI 网络插件",
		Description: "在 Master 节点安装容器网络接口插件（Flannel/Calico/Cilium），等待 CNI Pods 就绪，检查节点状态",
		Phase:       "addon",
		AppliesTo:   "master",
		Tasks: []string{
			"根据 cni_type 变量选择 CNI 插件",
			"Flannel: 下载并应用 kube-flannel.yml，替换 Pod 网段",
			"Calico: 下载并应用 calico.yaml，替换 CIDR",
			"Cilium: 安装 cilium CLI 并部署",
			"等待 CNI Pods 就绪 (kubectl wait)",
			"检查所有节点状态为 Ready",
		},
		DependsOn: []string{"kubeadm_init"},
	},
	{
		Key:         "install_addons",
		Title:       "安装 Kubernetes 扩展组件",
		Description: "按部署计划安装 metrics-server、ingress-nginx 和本地存储插件，并等待工作负载就绪",
		Phase:       "addon",
		AppliesTo:   "master",
		Tasks: []string{
			"应用计划选择的扩展组件清单",
			"等待 Deployment rollout 完成",
		},
		DependsOn: []string{"install_cni"},
	},
	{
		Key:         "register",
		Title:       "节点注册到管理平台",
		Description: "从 Master 节点提取 kubeconfig，替换 API Server 地址为实际 IP，写回到 Ansible 控制器供后端 Go 代码注册集群",
		Phase:       "finalize",
		AppliesTo:   "master",
		Tasks: []string{
			"读取 /etc/kubernetes/admin.conf",
			"替换 127.0.0.1:6443 为 Master 实际 IP:6443",
			"写入 kubeconfig 到 /tmp/k8s-deploy-{cluster_name}-kubeconfig.yml",
			"后端读取 kubeconfig 并注册集群到管理平台",
		},
		DependsOn: []string{"install_addons", "join_workers"},
	},
}

// DryRunPlan 根据计划生成 Ansible 部署预览
func (s *DeployService) DryRunPlan(ctx context.Context, planID uint64) (DryRunResult, error) {
	if planID == 0 {
		return DryRunResult{}, ErrInvalidParams
	}
	plan, nodes, err := s.getPlanWithNodes(ctx, planID)
	if err != nil {
		return DryRunResult{}, err
	}
	if len(nodes) == 0 {
		return DryRunResult{}, ErrWithMessage(ErrInvalidParams, "部署计划未配置节点")
	}

	// 加载所有相关服务器
	serverIDs := make([]uint64, 0, len(nodes))
	for _, n := range nodes {
		serverIDs = append(serverIDs, n.ServerID)
	}
	var servers []model.DeployServer
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id IN ?", serverIDs).Find(&servers).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return DryRunResult{}, err
		}
	}
	serverMap := make(map[uint64]model.DeployServer, len(servers))
	for _, srv := range servers {
		serverMap[srv.ID] = srv
	}

	// 排序：master 优先，按 sortOrder
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Role != nodes[j].Role {
			return nodes[i].Role == "master"
		}
		return nodes[i].SortOrder < nodes[j].SortOrder
	})

	flows := make([]DryRunNodeFlow, 0, len(nodes))
	summary := map[string]int{}
	for _, n := range nodes {
		srv := serverMap[n.ServerID]
		serverName := srv.Name
		if serverName == "" {
			serverName = fmt.Sprintf("server-%d", n.ServerID)
		}
		steps := s.buildAnsibleSteps(n.Role)
		for _, st := range steps {
			summary[st.Phase]++
		}
		flows = append(flows, DryRunNodeFlow{
			ServerID:   n.ServerID,
			ServerName: serverName,
			IP:         srv.IP,
			Role:       n.Role,
			Steps:      steps,
		})
	}

	return DryRunResult{
		PlanID:      plan.ID,
		PlanName:    plan.Name,
		ClusterName: plan.ClusterName,
		K8sVersion:  plan.K8sVersion,
		CNIType:     plan.CNIType,
		Nodes:       flows,
		Summary:     summary,
	}, nil
}

// buildAnsibleSteps 根据节点角色生成适用的 Ansible 步骤
func (s *DeployService) buildAnsibleSteps(role string) []DryRunStep {
	steps := make([]DryRunStep, 0, len(ansibleDryRunSteps))
	for _, def := range ansibleDryRunSteps {
		// 判断步骤是否适用于当前角色
		if def.AppliesTo != "all" && def.AppliesTo != role {
			continue
		}
		// 替换任务描述中的变量占位符
		tasks := make([]string, len(def.Tasks))
		for i, t := range def.Tasks {
			tasks[i] = t
		}
		steps = append(steps, DryRunStep{
			Key:         def.Key,
			Title:       def.Title,
			Description: def.Description,
			Phase:       def.Phase,
			Tasks:       tasks,
			AppliesTo:   def.AppliesTo,
			DependsOn:   def.DependsOn,
		})
	}
	return steps
}
