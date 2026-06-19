package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

// DryRunStep 表示模拟运行的单个步骤
type DryRunStep struct {
	Key         string   `json:"key"`         // 步骤唯一键
	Title       string   `json:"title"`       // 步骤标题
	Description string   `json:"description"` // 步骤说明
	Phase       string   `json:"phase"`       // preflight/install/init/join/addon/finalize
	Commands    []string `json:"commands"`    // 实际执行命令
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

// DryRunPlan 根据计划生成模拟运行流程
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

	// 找出第一个 master（用于 init 步骤），其余 master 用于 controlplane join
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Role != nodes[j].Role {
			return nodes[i].Role == "master"
		}
		return nodes[i].SortOrder < nodes[j].SortOrder
	})
	var firstMasterID uint64
	for _, n := range nodes {
		if n.Role == "master" {
			firstMasterID = n.ServerID
			break
		}
	}

	flows := make([]DryRunNodeFlow, 0, len(nodes))
	summary := map[string]int{}
	for _, n := range nodes {
		srv := serverMap[n.ServerID]
		serverName := srv.Name
		if serverName == "" {
			serverName = fmt.Sprintf("server-%d", n.ServerID)
		}
		isFirstMaster := n.ServerID == firstMasterID
		steps := buildNodeSteps(plan, n.Role, isFirstMaster)
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

// buildNodeSteps 根据计划与节点角色生成步骤序列
func buildNodeSteps(plan model.DeployPlan, role string, isFirstMaster bool) []DryRunStep {
	k8sVer := strings.TrimPrefix(plan.K8sVersion, "v")
	steps := []DryRunStep{
		{
			Key: "preflight.os", Title: "操作系统预检", Phase: "preflight",
			Description: "校验系统版本、关闭 swap、调整内核参数",
			Commands: []string{
				"swapoff -a",
				"sed -i '/ swap / s/^\\(.*\\)$/#\\1/g' /etc/fstab",
				"modprobe br_netfilter && modprobe overlay",
				"sysctl --system",
			},
		},
		{
			Key: "preflight.firewall", Title: "防火墙与 SELinux", Phase: "preflight",
			Description: "关闭 firewalld、设置 SELinux 为 permissive",
			Commands: []string{
				"systemctl stop firewalld && systemctl disable firewalld",
				"setenforce 0 || true",
				"sed -i 's/^SELINUX=.*/SELINUX=permissive/' /etc/selinux/config || true",
			},
			DependsOn: []string{"preflight.os"},
		},
		{
			Key: "install.containerd", Title: "安装容器运行时", Phase: "install",
			Description: "安装并配置 containerd 作为 K8s 容器运行时",
			Commands: []string{
				"yum install -y yum-utils device-mapper-persistent-data lvm2",
				"yum install -y containerd.io",
				"containerd config default > /etc/containerd/config.toml",
				"sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml",
				"systemctl enable --now containerd",
			},
			DependsOn: []string{"preflight.firewall"},
		},
		{
			Key: "install.kube", Title: "安装 kubeadm/kubelet/kubectl", Phase: "install",
			Description: fmt.Sprintf("安装 K8s %s 工具链", plan.K8sVersion),
			Commands: []string{
				"cat > /etc/yum.repos.d/kubernetes.repo <<EOF\n[kubernetes]\nname=Kubernetes\nbaseurl=https://mirrors.aliyun.com/kubernetes/yum/repos/kubernetes-el7-x86_64\nenabled=1\ngpgcheck=0\nEOF",
				fmt.Sprintf("yum install -y kubelet-%s kubeadm-%s kubectl-%s --disableexcludes=kubernetes", k8sVer, k8sVer, k8sVer),
				"systemctl enable kubelet",
			},
			DependsOn: []string{"install.containerd"},
		},
	}

	if role == "master" && isFirstMaster {
		steps = append(steps,
			DryRunStep{
				Key: "init.controlplane", Title: "初始化控制平面", Phase: "init",
				Description: fmt.Sprintf("通过 kubeadm 初始化集群 %s", plan.ClusterName),
				Commands: []string{
					fmt.Sprintf("kubeadm init --kubernetes-version=%s --pod-network-cidr=%s --service-cidr=%s --image-repository=registry.aliyuncs.com/google_containers", plan.K8sVersion, plan.PodCIDR, plan.SvcCIDR),
					"mkdir -p $HOME/.kube && cp /etc/kubernetes/admin.conf $HOME/.kube/config",
					"chown $(id -u):$(id -g) $HOME/.kube/config",
				},
				DependsOn: []string{"install.kube"},
			},
			buildCNIStep(plan.CNIType, plan.PodCIDR),
		)
		// addons
		for _, addon := range plan.Addons {
			steps = append(steps, buildAddonStep(addon))
		}
	} else if role == "master" {
		steps = append(steps, DryRunStep{
			Key: "join.controlplane", Title: "加入控制平面", Phase: "join",
			Description: "以 control-plane 角色加入集群",
			Commands: []string{
				"# 在首个 master 上获取 join 命令：kubeadm token create --print-join-command",
				"kubeadm join <CONTROL_PLANE_ENDPOINT> --token <TOKEN> --discovery-token-ca-cert-hash sha256:<HASH> --control-plane --certificate-key <CERT_KEY>",
			},
			DependsOn: []string{"install.kube"},
		})
	} else {
		steps = append(steps, DryRunStep{
			Key: "join.worker", Title: "加入工作节点", Phase: "join",
			Description: "以 worker 角色加入集群",
			Commands: []string{
				"# 在首个 master 上获取 join 命令：kubeadm token create --print-join-command",
				"kubeadm join <CONTROL_PLANE_ENDPOINT> --token <TOKEN> --discovery-token-ca-cert-hash sha256:<HASH>",
			},
			DependsOn: []string{"install.kube"},
		})
	}

	steps = append(steps, DryRunStep{
		Key: "finalize.verify", Title: "节点状态校验", Phase: "finalize",
		Description: "等待节点就绪并验证 kubelet 状态",
		Commands: []string{
			"systemctl status kubelet --no-pager",
			"kubectl get nodes -o wide || true",
		},
	})

	return steps
}

func buildCNIStep(cniType, podCIDR string) DryRunStep {
	switch cniType {
	case "calico":
		return DryRunStep{
			Key: "addon.cni", Title: "部署 Calico CNI", Phase: "addon",
			Description: "应用 Calico 网络插件清单",
			Commands: []string{
				"kubectl apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.27.0/manifests/calico.yaml",
			},
			DependsOn: []string{"init.controlplane"},
		}
	case "cilium":
		return DryRunStep{
			Key: "addon.cni", Title: "部署 Cilium CNI", Phase: "addon",
			Description: "通过 helm 部署 Cilium",
			Commands: []string{
				"helm repo add cilium https://helm.cilium.io/",
				"helm install cilium cilium/cilium --namespace kube-system",
			},
			DependsOn: []string{"init.controlplane"},
		}
	default:
		return DryRunStep{
			Key: "addon.cni", Title: "部署 Flannel CNI", Phase: "addon",
			Description: fmt.Sprintf("应用 Flannel 网络插件（Pod CIDR: %s）", podCIDR),
			Commands: []string{
				"kubectl apply -f https://raw.githubusercontent.com/flannel-io/flannel/master/Documentation/kube-flannel.yml",
			},
			DependsOn: []string{"init.controlplane"},
		}
	}
}

func buildAddonStep(addon string) DryRunStep {
	switch strings.ToLower(addon) {
	case "metrics-server":
		return DryRunStep{
			Key: "addon.metrics-server", Title: "部署 Metrics Server", Phase: "addon",
			Description: "提供节点和 Pod 资源使用指标",
			Commands: []string{
				"kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml",
			},
			DependsOn: []string{"addon.cni"},
		}
	case "ingress-nginx":
		return DryRunStep{
			Key: "addon.ingress-nginx", Title: "部署 Ingress NGINX", Phase: "addon",
			Description: "提供 HTTP/HTTPS 路由能力",
			Commands: []string{
				"kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml",
			},
			DependsOn: []string{"addon.cni"},
		}
	case "dashboard":
		return DryRunStep{
			Key: "addon.dashboard", Title: "部署 Kubernetes Dashboard", Phase: "addon",
			Description: "提供集群可视化管理面板",
			Commands: []string{
				"kubectl apply -f https://raw.githubusercontent.com/kubernetes/dashboard/v2.7.0/aio/deploy/recommended.yaml",
			},
			DependsOn: []string{"addon.cni"},
		}
	default:
		return DryRunStep{
			Key: "addon." + addon, Title: "部署 " + addon, Phase: "addon",
			Description: "自定义附加组件部署",
			Commands: []string{
				fmt.Sprintf("# 待提供 %s 组件的部署清单", addon),
			},
			DependsOn: []string{"addon.cni"},
		}
	}
}
