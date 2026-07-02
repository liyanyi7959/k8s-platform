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
		steps := s.buildNodeSteps(ctx, plan, n, srv, isFirstMaster)
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
func (s *DeployService) buildNodeSteps(ctx context.Context, plan model.DeployPlan, node model.DeployPlanNode, server model.DeployServer, isFirstMaster bool) []DryRunStep {
	stepMetas := []struct {
		stepKey string
		title   string
		phase   string
		when    func() bool
	}{
		{stepKey: "pre_check", title: "环境预检", phase: "preflight", when: func() bool { return true }},
		{stepKey: "bootstrap", title: "基础环境初始化", phase: "install", when: func() bool { return true }},
		{stepKey: "init_master", title: "初始化 Master", phase: "init", when: func() bool { return node.Role == "master" && isFirstMaster }},
		{stepKey: "join_workers", title: "节点加入集群", phase: "join", when: func() bool { return node.Role == "worker" }},
		{stepKey: "install_cni", title: "安装网络插件", phase: "addon", when: func() bool { return node.Role == "master" && isFirstMaster }},
		{stepKey: "register", title: "注册集群", phase: "finalize", when: func() bool { return node.Role == "master" && isFirstMaster }},
	}
	data := deployTemplateData{
		PlanID:       plan.ID,
		PlanName:     plan.Name,
		ClusterName:  plan.ClusterName,
		K8sVersion:   plan.K8sVersion,
		MinorVersion: strings.TrimPrefix(plan.K8sVersion, "v"),
		PodCIDR:      plan.PodCIDR,
		SvcCIDR:      plan.SvcCIDR,
		MasterIP:     server.IP,
		JoinCommand:  "kubeadm join <CONTROL_PLANE_ENDPOINT> --token <TOKEN> --discovery-token-ca-cert-hash sha256:<HASH>",
		CNICommand:   resolveCNICommand(plan.CNIType),
		NodeRole:     node.Role,
		ServerID:     node.ServerID,
		ServerIP:     server.IP,
	}
	if idx := strings.LastIndex(data.MinorVersion, "."); idx > 0 {
		data.MinorVersion = data.MinorVersion[:idx]
	}
	steps := make([]DryRunStep, 0, len(stepMetas))
	for _, meta := range stepMetas {
		if !meta.when() {
			continue
		}
		resolved, err := s.resolvePlanStep(ctx, plan, meta.stepKey, node, server, data)
		if err != nil || !resolved.Config.Enabled {
			continue
		}
		description := meta.title
		if resolved.Config.Description != nil && strings.TrimSpace(*resolved.Config.Description) != "" {
			description = *resolved.Config.Description
		}
		steps = append(steps, DryRunStep{
			Key:         resolved.Config.StepKey,
			Title:       resolved.Config.StepName,
			Description: description,
			Phase:       meta.phase,
			Commands:    resolved.Commands,
		})
	}
	return steps
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
