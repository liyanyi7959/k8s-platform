package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"k8s-platform-backend/internal/model"
)

type ResolvedDeployStep struct {
	Config   model.DeployConfig
	Commands []string
}

type deployTemplateData struct {
	PlanID       uint64
	PlanName     string
	ClusterName  string
	K8sVersion   string
	MinorVersion string
	PodCIDR      string
	SvcCIDR      string
	MasterIP     string
	JoinCommand  string
	CNICommand   string
	NodeRole     string
	ServerID     uint64
	ServerIP     string
}

func (s *DeployService) resolvePlanStep(ctx context.Context, plan model.DeployPlan, stepKey string, node model.DeployPlanNode, server model.DeployServer, data deployTemplateData) (ResolvedDeployStep, error) {
	osType := normalizeDeployOSType(server.OS)
	config, err := s.deployConfig.GetConfigByKey(ctx, stepKey, osType)
	if err != nil {
		config, err = s.deployConfig.GetConfigByKey(ctx, stepKey, "ubuntu")
		if err != nil {
			return ResolvedDeployStep{}, err
		}
	}
	resolved := *config
	if override, ok := matchStepOverride(decodePlanStepOverrides(plan.StepOverrides), stepKey, node); ok {
		resolved.CommandTemplate = override.CommandTemplate
		if override.Description != nil {
			resolved.Description = override.Description
		}
		if override.TimeoutSeconds != nil {
			resolved.TimeoutSeconds = *override.TimeoutSeconds
		}
		if override.RetryCount != nil {
			resolved.RetryCount = *override.RetryCount
		}
		if override.Enabled != nil {
			resolved.Enabled = *override.Enabled
		}
	}
	commands, err := renderCommandTemplate(resolved.CommandTemplate, data)
	if err != nil {
		return ResolvedDeployStep{}, err
	}
	return ResolvedDeployStep{Config: resolved, Commands: commands}, nil
}

func renderCommandTemplate(tpl string, data deployTemplateData) ([]string, error) {
	parsed, err := template.New("deploy-step").Parse(tpl)
	if err != nil {
		return nil, fmt.Errorf("解析命令模板失败: %w", err)
	}
	var buf bytes.Buffer
	if err := parsed.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("渲染命令模板失败: %w", err)
	}
	raw := strings.TrimSpace(buf.String())
	if raw == "" {
		return []string{}, nil
	}
	blocks := strings.Split(raw, "\n\n")
	commands := make([]string, 0, len(blocks))
	for _, block := range blocks {
		trimmed := strings.TrimSpace(block)
		if trimmed == "" {
			continue
		}
		commands = append(commands, trimmed)
	}
	return commands, nil
}

func resolveCNICommand(cniType string) string {
	switch cniType {
	case "calico":
		return "kubectl apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.27.0/manifests/calico.yaml"
	case "cilium":
		return "kubectl apply -f https://raw.githubusercontent.com/cilium/cilium/v1.15.0/install/kubernetes/quick-install.yaml"
	default:
		return "kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml"
	}
}

func normalizeDeployOSType(osName *string) string {
	if osName == nil {
		return "ubuntu"
	}
	text := strings.ToLower(strings.TrimSpace(*osName))
	switch {
	case strings.Contains(text, "rocky"):
		return "rocky"
	case strings.Contains(text, "alma"):
		return "almalinux"
	case strings.Contains(text, "red hat"):
		return "rhel"
	case strings.Contains(text, "centos"):
		return "centos"
	case strings.Contains(text, "debian"):
		return "debian"
	default:
		return "ubuntu"
	}
}

func matchStepOverride(overrides map[string]model.DeployPlanStepOverride, stepKey string, node model.DeployPlanNode) (model.DeployPlanStepOverride, bool) {
	var roleMatch *model.DeployPlanStepOverride
	var globalMatch *model.DeployPlanStepOverride
	for _, override := range overrides {
		if strings.TrimSpace(override.StepKey) != stepKey {
			continue
		}
		if override.NodeServerID != nil {
			if *override.NodeServerID == node.ServerID {
				return override, true
			}
			continue
		}
		if override.NodeRole != "" {
			if override.NodeRole == node.Role {
				copy := override
				roleMatch = &copy
			}
			continue
		}
		copy := override
		globalMatch = &copy
	}
	if roleMatch != nil {
		return *roleMatch, true
	}
	if globalMatch != nil {
		return *globalMatch, true
	}
	return model.DeployPlanStepOverride{}, false
}
