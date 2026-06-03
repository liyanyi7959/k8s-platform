package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s-platform-backend/internal/model"
)

type WorkloadActionTarget struct {
	Kind      string
	Namespace string
	Name      string
}

type PrepareWorkloadActionRequest struct {
	ClusterID  uint64
	ActionType string
	Target     WorkloadActionTarget
	Payload    model.JSONMap
	Reason     string
}

type PreparedWorkloadAction struct {
	ActionType   string
	Target       WorkloadActionTarget
	RiskLevel    string
	ConfirmLevel string
	Title        string
	Summary      string
	Change       model.JSONMap
	Preview      string
	Diff         string
}

type ExecuteWorkloadActionRequest struct {
	ClusterID     uint64
	ActionType    string
	Target        WorkloadActionTarget
	Change        model.JSONMap
	ProposalTitle string
	CreatedBy     uint64
	CreatedByName string
}

type ExecuteWorkloadActionResult struct {
	Result  model.JSONMap
	Summary string
}

type WorkloadActionService struct {
	k8sSvc      *K8sService
	manifestSvc *ManifestApplyRecordService
}

func NewWorkloadActionService(k8sSvc *K8sService, manifestSvc *ManifestApplyRecordService) *WorkloadActionService {
	return &WorkloadActionService{k8sSvc: k8sSvc, manifestSvc: manifestSvc}
}

func (s *WorkloadActionService) PrepareProposal(
	ctx context.Context,
	req PrepareWorkloadActionRequest,
) (PreparedWorkloadAction, error) {
	if s == nil || s.k8sSvc == nil {
		return PreparedWorkloadAction{}, errors.New("workload action service is required")
	}
	if req.ClusterID == 0 {
		return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "集群参数无效")
	}

	target := normalizeWorkloadActionTarget(req.Target)
	payload := req.Payload
	if payload == nil {
		payload = model.JSONMap{}
	}
	reason := strings.TrimSpace(req.Reason)

	switch normalizeAIActionType(req.ActionType) {
	case aiActionTypeRestartWorkload:
		gvr, ok := aiActionWorkloadGVR(target.Kind)
		if !ok || target.Namespace == "" || target.Name == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "重启提案的目标资源参数无效")
		}
		if _, err := s.k8sSvc.GetObject(ctx, req.ClusterID, gvr, target.Namespace, target.Name); err != nil {
			return PreparedWorkloadAction{}, err
		}
		preview := fmt.Sprintf("将为 %s %s/%s 写入 rollout restart 注解。", target.Kind, target.Namespace, target.Name)
		return PreparedWorkloadAction{
			ActionType:   aiActionTypeRestartWorkload,
			Target:       target,
			RiskLevel:    "low",
			ConfirmLevel: "single",
			Title:        fmt.Sprintf("重启 %s %s/%s", target.Kind, target.Namespace, target.Name),
			Summary: firstNonEmpty(
				reason,
				fmt.Sprintf("建议对 %s %s/%s 发起滚动重启，以验证问题是否已经恢复。", target.Kind, target.Namespace, target.Name),
			),
			Change: model.JSONMap{
				"payload": payload,
				"preview": preview,
			},
			Preview: preview,
		}, nil
	case aiActionTypeScaleWorkload:
		gvr, ok := aiActionWorkloadGVR(target.Kind)
		if !ok || target.Namespace == "" || target.Name == "" || strings.EqualFold(target.Kind, "DaemonSet") {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "扩缩容提案的目标资源参数无效")
		}
		replicas, ok := jsonIntValue(payload["replicas"])
		if !ok || replicas < 0 {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "扩缩容提案缺少有效的 replicas 参数")
		}
		obj, err := s.k8sSvc.GetObject(ctx, req.ClusterID, gvr, target.Namespace, target.Name)
		if err != nil {
			return PreparedWorkloadAction{}, err
		}
		currentReplicas := jsonNestedInt(obj, "spec", "replicas")
		preview := fmt.Sprintf("副本数变更预览: %d -> %d", currentReplicas, replicas)
		return PreparedWorkloadAction{
			ActionType:   aiActionTypeScaleWorkload,
			Target:       target,
			RiskLevel:    "low",
			ConfirmLevel: "single",
			Title:        fmt.Sprintf("调整 %s %s/%s 副本数", target.Kind, target.Namespace, target.Name),
			Summary: firstNonEmpty(
				reason,
				fmt.Sprintf("建议将 %s %s/%s 的副本数从 %d 调整为 %d。", target.Kind, target.Namespace, target.Name, currentReplicas, replicas),
			),
			Change: model.JSONMap{
				"payload":          payload,
				"current_replicas": currentReplicas,
				"target_replicas":  replicas,
				"preview":          preview,
			},
			Preview: preview,
			Diff:    fmt.Sprintf("replicas: %d -> %d", currentReplicas, replicas),
		}, nil
	case aiActionTypeApplyManifest:
		yamlText := strings.TrimSpace(aiActionString(payload["yaml"]))
		if yamlText == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "Manifest 提案缺少 YAML 内容")
		}
		defaultNamespace := strings.TrimSpace(aiActionString(payload["default_namespace"]))
		return PreparedWorkloadAction{
			ActionType:   aiActionTypeApplyManifest,
			Target:       target,
			RiskLevel:    "medium",
			ConfirmLevel: "single",
			Title:        firstNonEmpty(strings.TrimSpace(aiActionString(payload["title"])), "应用 AI 生成的 Manifest"),
			Summary: firstNonEmpty(
				reason,
				"建议执行一份已预览的 Manifest 变更，请在确认前仔细检查 YAML 内容。",
			),
			Change: model.JSONMap{
				"payload":           payload,
				"default_namespace": defaultNamespace,
				"preview":           truncateForModel(yamlText, 1000),
			},
			Preview: truncateForModel(yamlText, 240),
			Diff:    truncateForModel(yamlText, 1200),
		}, nil
	default:
		return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "当前动作类型暂不支持")
	}
}

func (s *WorkloadActionService) ExecuteProposalAction(
	ctx context.Context,
	req ExecuteWorkloadActionRequest,
) (ExecuteWorkloadActionResult, error) {
	if s == nil || s.k8sSvc == nil {
		return ExecuteWorkloadActionResult{}, errors.New("workload action service is required")
	}
	if req.ClusterID == 0 {
		return ExecuteWorkloadActionResult{}, ErrWithMessage(ErrInvalidParams, "集群参数无效")
	}

	target := normalizeWorkloadActionTarget(req.Target)
	changePayload := aiActionPayload(req.Change)

	switch normalizeAIActionType(req.ActionType) {
	case aiActionTypeRestartWorkload:
		gvr, ok := aiActionWorkloadGVR(target.Kind)
		if !ok {
			return ExecuteWorkloadActionResult{}, ErrWithMessage(ErrInvalidParams, "重启提案资源类型不支持")
		}
		patch := map[string]any{
			"spec": map[string]any{
				"template": map[string]any{
					"metadata": map[string]any{
						"annotations": map[string]any{
							"kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339),
						},
					},
				},
			},
		}
		if err := s.k8sSvc.PatchJSON(ctx, req.ClusterID, gvr, target.Namespace, target.Name, patch); err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := fmt.Sprintf("已触发 %s %s/%s 的滚动重启。", target.Kind, target.Namespace, target.Name)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type": req.ActionType,
				"target":      fmt.Sprintf("%s/%s/%s", target.Kind, target.Namespace, target.Name),
				"summary":     summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypeScaleWorkload:
		gvr, ok := aiActionWorkloadGVR(target.Kind)
		if !ok {
			return ExecuteWorkloadActionResult{}, ErrWithMessage(ErrInvalidParams, "扩缩容提案资源类型不支持")
		}
		targetReplicas, ok := jsonIntValue(changePayload["replicas"])
		if !ok || targetReplicas < 0 {
			targetReplicas, ok = jsonIntValue(req.Change["target_replicas"])
		}
		if !ok || targetReplicas < 0 {
			return ExecuteWorkloadActionResult{}, ErrWithMessage(ErrInvalidParams, "扩缩容提案缺少有效副本数")
		}
		if err := s.k8sSvc.PatchJSON(ctx, req.ClusterID, gvr, target.Namespace, target.Name, map[string]any{
			"spec": map[string]any{
				"replicas": targetReplicas,
			},
		}); err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := fmt.Sprintf("已将 %s %s/%s 的副本数调整为 %d。", target.Kind, target.Namespace, target.Name, targetReplicas)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type":     req.ActionType,
				"target_replicas": targetReplicas,
				"summary":         summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypeApplyManifest:
		if s.manifestSvc == nil {
			return ExecuteWorkloadActionResult{}, ErrWithMessage(ErrConflict, "Manifest 执行服务未初始化")
		}
		yamlText := strings.TrimSpace(aiActionString(changePayload["yaml"]))
		if yamlText == "" {
			return ExecuteWorkloadActionResult{}, ErrWithMessage(ErrInvalidParams, "Manifest 提案缺少 YAML 内容")
		}
		defaultNamespace := strings.TrimSpace(aiActionString(changePayload["default_namespace"]))
		result, err := s.manifestSvc.Execute(ctx, ManifestApplyExecuteRequest{
			ClusterID:        req.ClusterID,
			YAML:             yamlText,
			DefaultNamespace: defaultNamespace,
			DryRun:           false,
			SourceLabel:      "AI 动作提案",
			SourceResource:   strings.TrimSpace(req.ProposalTitle),
			CreatedBy:        req.CreatedBy,
			CreatedByName:    strings.TrimSpace(req.CreatedByName),
		})
		if err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := firstNonEmpty(strings.TrimSpace(result.Summary), "Manifest 已执行")
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type": req.ActionType,
				"record_id":   result.RecordID,
				"status":      result.Status,
				"summary":     summary,
			},
			Summary: summary,
		}, nil
	default:
		return ExecuteWorkloadActionResult{}, ErrWithMessage(ErrInvalidParams, "当前动作类型暂不支持执行")
	}
}

func normalizeWorkloadActionTarget(target WorkloadActionTarget) WorkloadActionTarget {
	kind := strings.TrimSpace(target.Kind)
	switch strings.ToLower(kind) {
	case "deployment":
		kind = "Deployment"
	case "statefulset":
		kind = "StatefulSet"
	case "daemonset":
		kind = "DaemonSet"
	}
	return WorkloadActionTarget{
		Kind:      kind,
		Namespace: strings.TrimSpace(target.Namespace),
		Name:      strings.TrimSpace(target.Name),
	}
}
