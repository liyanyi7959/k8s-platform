package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

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
	case aiActionTypeUpdateWorkloadImage:
		gvr, ok := aiActionWorkloadGVR(target.Kind)
		if !ok || target.Namespace == "" || target.Name == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "update image proposal requires a workload target")
		}
		containerName := strings.TrimSpace(aiActionString(payload["container_name"]))
		image := strings.TrimSpace(aiActionString(payload["image"]))
		if containerName == "" || image == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "update image proposal requires container_name and image")
		}
		if _, err := s.k8sSvc.GetObject(ctx, req.ClusterID, gvr, target.Namespace, target.Name); err != nil {
			return PreparedWorkloadAction{}, err
		}
		preview := fmt.Sprintf("update %s %s/%s container %s to image %s", target.Kind, target.Namespace, target.Name, containerName, image)
		return PreparedWorkloadAction{
			ActionType:   aiActionTypeUpdateWorkloadImage,
			Target:       target,
			RiskLevel:    "medium",
			ConfirmLevel: "single",
			Title:        fmt.Sprintf("更新 %s %s/%s 镜像", target.Kind, target.Namespace, target.Name),
			Summary: firstNonEmpty(
				reason,
				fmt.Sprintf("建议将 %s %s/%s 中容器 %s 的镜像更新为 %s。", target.Kind, target.Namespace, target.Name, containerName, image),
			),
			Change: model.JSONMap{
				"payload":  payload,
				"preview":  preview,
				"container": containerName,
				"image":    image,
			},
			Preview: preview,
			Diff:    fmt.Sprintf("container %s image -> %s", containerName, image),
		}, nil
	case aiActionTypePauseWorkloadRollout:
		gvr, ok := aiActionWorkloadGVR(target.Kind)
		if !ok || !strings.EqualFold(target.Kind, "Deployment") || target.Namespace == "" || target.Name == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "pause rollout proposal currently supports Deployment only")
		}
		paused := true
		if rawPaused, exists := payload["paused"]; exists {
			paused = aiBoolValue(rawPaused)
		}
		if _, err := s.k8sSvc.GetObject(ctx, req.ClusterID, gvr, target.Namespace, target.Name); err != nil {
			return PreparedWorkloadAction{}, err
		}
		verb := "暂停"
		if !paused {
			verb = "恢复"
		}
		preview := fmt.Sprintf("%s Deployment %s/%s 的 rollout", verb, target.Namespace, target.Name)
		return PreparedWorkloadAction{
			ActionType:   aiActionTypePauseWorkloadRollout,
			Target:       target,
			RiskLevel:    "medium",
			ConfirmLevel: "single",
			Title:        fmt.Sprintf("%s Deployment %s/%s Rollout", verb, target.Namespace, target.Name),
			Summary: firstNonEmpty(
				reason,
				fmt.Sprintf("建议%s Deployment %s/%s 的 rollout。", verb, target.Namespace, target.Name),
			),
			Change: model.JSONMap{
				"payload": payload,
				"paused":  paused,
				"preview": preview,
			},
			Preview: preview,
			Diff:    fmt.Sprintf("spec.paused -> %t", paused),
		}, nil
	case aiActionTypeRolloutUndo:
		gvr, ok := aiActionWorkloadGVR(target.Kind)
		if !ok || !strings.EqualFold(target.Kind, "Deployment") || target.Namespace == "" || target.Name == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "rollout undo proposal currently supports Deployment only")
		}
		revision, ok := jsonIntValue(payload["revision"])
		if !ok {
			revision = 0
		}
		if revision < 0 {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "rollout undo revision must be >= 0")
		}
		if _, err := s.k8sSvc.GetObject(ctx, req.ClusterID, gvr, target.Namespace, target.Name); err != nil {
			return PreparedWorkloadAction{}, err
		}
		preview := fmt.Sprintf("rollback Deployment %s/%s to revision %d", target.Namespace, target.Name, revision)
		if revision == 0 {
			preview = fmt.Sprintf("rollback Deployment %s/%s to the previous revision", target.Namespace, target.Name)
		}
		return PreparedWorkloadAction{
			ActionType:   aiActionTypeRolloutUndo,
			Target:       target,
			RiskLevel:    "medium",
			ConfirmLevel: "single",
			Title:        fmt.Sprintf("回滚 Deployment %s/%s", target.Namespace, target.Name),
			Summary: firstNonEmpty(
				reason,
				fmt.Sprintf("建议回滚 Deployment %s/%s。", target.Namespace, target.Name),
			),
			Change: model.JSONMap{
				"payload":  payload,
				"revision": revision,
				"preview":  preview,
			},
			Preview: preview,
		}, nil
	case aiActionTypeDeleteWorkload:
		gvr, ok := aiActionWorkloadGVR(target.Kind)
		if !ok || target.Namespace == "" || target.Name == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "delete workload proposal requires a workload target")
		}
		if _, err := s.k8sSvc.GetObject(ctx, req.ClusterID, gvr, target.Namespace, target.Name); err != nil {
			return PreparedWorkloadAction{}, err
		}
		preview := fmt.Sprintf("delete %s %s/%s", target.Kind, target.Namespace, target.Name)
		return PreparedWorkloadAction{
			ActionType:   aiActionTypeDeleteWorkload,
			Target:       target,
			RiskLevel:    "high",
			ConfirmLevel: "double",
			Title:        fmt.Sprintf("删除 %s %s/%s", target.Kind, target.Namespace, target.Name),
			Summary: firstNonEmpty(reason, fmt.Sprintf("建议删除 %s %s/%s。", target.Kind, target.Namespace, target.Name)),
			Change: model.JSONMap{
				"payload": payload,
				"preview": preview,
			},
			Preview: preview,
		}, nil
	case aiActionTypeDeleteResource:
		gvr, namespaced, ok := aiSupportedResourceGVR(target.Kind)
		if !ok || target.Name == "" || (namespaced && target.Namespace == "") {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "delete resource proposal requires a valid supported resource target")
		}
		if !namespaced {
			target.Namespace = ""
		}
		if _, err := s.k8sSvc.GetObject(ctx, req.ClusterID, gvr, target.Namespace, target.Name); err != nil {
			return PreparedWorkloadAction{}, err
		}
		preview := fmt.Sprintf("delete %s %s/%s", target.Kind, target.Namespace, target.Name)
		return PreparedWorkloadAction{
			ActionType:   aiActionTypeDeleteResource,
			Target:       target,
			RiskLevel:    aiDeleteResourceRiskLevel(target.Kind),
			ConfirmLevel: aiDeleteResourceConfirmLevel(target.Kind),
			Title:        fmt.Sprintf("删除 %s %s/%s", target.Kind, target.Namespace, target.Name),
			Summary:      firstNonEmpty(reason, fmt.Sprintf("建议删除 %s %s/%s。", target.Kind, target.Namespace, target.Name)),
			Change: model.JSONMap{
				"payload": payload,
				"preview": preview,
			},
			Preview: preview,
		}, nil
	case aiActionTypeDeletePod:
		if target.Namespace == "" || target.Name == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "delete pod proposal requires namespace and pod name")
		}
		if _, err := s.k8sSvc.GetObject(ctx, req.ClusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, target.Namespace, target.Name); err != nil {
			return PreparedWorkloadAction{}, err
		}
		force := aiBoolValue(payload["force"])
		preview := fmt.Sprintf("delete Pod %s/%s", target.Namespace, target.Name)
		if force {
			preview += " with force"
		}
		return PreparedWorkloadAction{
			ActionType:   aiActionTypeDeletePod,
			Target:       target,
			RiskLevel:    "medium",
			ConfirmLevel: "single",
			Title:        fmt.Sprintf("删除 Pod %s/%s", target.Namespace, target.Name),
			Summary:      firstNonEmpty(reason, fmt.Sprintf("建议删除 Pod %s/%s。", target.Namespace, target.Name)),
			Change: model.JSONMap{
				"payload": payload,
				"force":   force,
				"preview": preview,
			},
			Preview: preview,
		}, nil
	case aiActionTypeCordonNode, aiActionTypeUncordonNode:
		if !strings.EqualFold(target.Kind, "Node") || target.Name == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "node scheduling proposal requires a node target")
		}
		if _, err := s.k8sSvc.GetObject(ctx, req.ClusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}, "", target.Name); err != nil {
			return PreparedWorkloadAction{}, err
		}
		verb := "封锁调度"
		previewVerb := "cordon"
		if normalizeAIActionType(req.ActionType) == aiActionTypeUncordonNode {
			verb = "恢复调度"
			previewVerb = "uncordon"
		}
		preview := fmt.Sprintf("%s node %s", previewVerb, target.Name)
		return PreparedWorkloadAction{
			ActionType:   normalizeAIActionType(req.ActionType),
			Target:       target,
			RiskLevel:    "medium",
			ConfirmLevel: "single",
			Title:        fmt.Sprintf("%s节点 %s", verb, target.Name),
			Summary:      firstNonEmpty(reason, fmt.Sprintf("建议%s节点 %s。", verb, target.Name)),
			Change: model.JSONMap{
				"payload": payload,
				"preview": preview,
			},
			Preview: preview,
		}, nil
	case aiActionTypeDrainNode:
		if !strings.EqualFold(target.Kind, "Node") || target.Name == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "drain node proposal requires a node target")
		}
		if _, err := s.k8sSvc.GetObject(ctx, req.ClusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}, "", target.Name); err != nil {
			return PreparedWorkloadAction{}, err
		}
		preview := fmt.Sprintf("drain node %s", target.Name)
		return PreparedWorkloadAction{
			ActionType:   aiActionTypeDrainNode,
			Target:       target,
			RiskLevel:    "high",
			ConfirmLevel: "double",
			Title:        fmt.Sprintf("驱逐节点 %s 上的工作负载", target.Name),
			Summary:      firstNonEmpty(reason, fmt.Sprintf("建议对节点 %s 执行 drain。", target.Name)),
			Change: model.JSONMap{
				"payload": payload,
				"preview": preview,
			},
			Preview: preview,
		}, nil
	case aiActionTypeTriggerCronJob:
		if !strings.EqualFold(target.Kind, "CronJob") || target.Namespace == "" || target.Name == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "trigger cronjob proposal requires a CronJob target")
		}
		if _, err := s.k8sSvc.GetObject(ctx, req.ClusterID, schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}, target.Namespace, target.Name); err != nil {
			return PreparedWorkloadAction{}, err
		}
		preview := fmt.Sprintf("trigger CronJob %s/%s once", target.Namespace, target.Name)
		return PreparedWorkloadAction{
			ActionType:   aiActionTypeTriggerCronJob,
			Target:       target,
			RiskLevel:    "medium",
			ConfirmLevel: "single",
			Title:        fmt.Sprintf("立即触发 CronJob %s/%s", target.Namespace, target.Name),
			Summary:      firstNonEmpty(reason, fmt.Sprintf("建议立即触发 CronJob %s/%s 一次执行。", target.Namespace, target.Name)),
			Change: model.JSONMap{
				"payload": payload,
				"preview": preview,
			},
			Preview: preview,
		}, nil
	case aiActionTypeSuspendCronJob:
		if !strings.EqualFold(target.Kind, "CronJob") || target.Namespace == "" || target.Name == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "suspend cronjob proposal requires a CronJob target")
		}
		suspend := true
		if rawSuspend, exists := payload["suspend"]; exists {
			suspend = aiBoolValue(rawSuspend)
		}
		if _, err := s.k8sSvc.GetObject(ctx, req.ClusterID, schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}, target.Namespace, target.Name); err != nil {
			return PreparedWorkloadAction{}, err
		}
		verb := "暂停"
		if !suspend {
			verb = "恢复"
		}
		preview := fmt.Sprintf("%s CronJob %s/%s 调度", verb, target.Namespace, target.Name)
		return PreparedWorkloadAction{
			ActionType:   aiActionTypeSuspendCronJob,
			Target:       target,
			RiskLevel:    "medium",
			ConfirmLevel: "single",
			Title:        fmt.Sprintf("%s CronJob %s/%s", verb, target.Namespace, target.Name),
			Summary:      firstNonEmpty(reason, fmt.Sprintf("建议%s CronJob %s/%s 的调度。", verb, target.Namespace, target.Name)),
			Change: model.JSONMap{
				"payload": payload,
				"suspend": suspend,
				"preview": preview,
			},
			Preview: preview,
			Diff:    fmt.Sprintf("spec.suspend -> %t", suspend),
		}, nil
	case aiActionTypeDeleteCompletedJobs:
		ns := strings.TrimSpace(target.Namespace)
		if ns == "" {
			ns = strings.TrimSpace(aiActionString(payload["namespace"]))
		}
		if ns == "" {
			return PreparedWorkloadAction{}, ErrWithMessage(ErrInvalidParams, "delete completed jobs proposal requires a namespace")
		}
		olderThanHours, ok := jsonIntValue(payload["older_than_hours"])
		if !ok {
			olderThanHours = 0
		}
		preview := fmt.Sprintf("delete completed Jobs in namespace %s older than %d hours", ns, olderThanHours)
		target.Namespace = ns
		target.Kind = "Job"
		target.Name = "*completed*"
		return PreparedWorkloadAction{
			ActionType:   aiActionTypeDeleteCompletedJobs,
			Target:       target,
			RiskLevel:    "medium",
			ConfirmLevel: "single",
			Title:        fmt.Sprintf("清理命名空间 %s 的已完成 Job", ns),
			Summary:      firstNonEmpty(reason, fmt.Sprintf("建议清理命名空间 %s 中已完成的 Job。", ns)),
			Change: model.JSONMap{
				"payload":          payload,
				"namespace":        ns,
				"older_than_hours": olderThanHours,
				"preview":          preview,
			},
			Preview: preview,
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
	case aiActionTypeUpdateWorkloadImage:
		if _, ok := aiActionWorkloadGVR(target.Kind); !ok {
			return ExecuteWorkloadActionResult{}, ErrWithMessage(ErrInvalidParams, "update image proposal resource kind is not supported")
		}
		containerName := strings.TrimSpace(aiActionString(changePayload["container_name"]))
		image := strings.TrimSpace(aiActionString(changePayload["image"]))
		if containerName == "" {
			containerName = strings.TrimSpace(aiActionString(req.Change["container"]))
		}
		if image == "" {
			image = strings.TrimSpace(aiActionString(req.Change["image"]))
		}
		if containerName == "" || image == "" {
			return ExecuteWorkloadActionResult{}, ErrWithMessage(ErrInvalidParams, "update image proposal requires container_name and image")
		}
		if err := s.k8sSvc.UpdateWorkloadImage(ctx, req.ClusterID, target.Namespace, target.Name, target.Kind, containerName, image); err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := fmt.Sprintf("已将 %s %s/%s 中容器 %s 的镜像更新为 %s。", target.Kind, target.Namespace, target.Name, containerName, image)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type":    req.ActionType,
				"container_name": containerName,
				"image":          image,
				"summary":        summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypePauseWorkloadRollout:
		paused := true
		if rawPaused, exists := changePayload["paused"]; exists {
			paused = aiBoolValue(rawPaused)
		} else if rawPaused, exists := req.Change["paused"]; exists {
			paused = aiBoolValue(rawPaused)
		}
		if err := s.k8sSvc.UpdateWorkloadPaused(ctx, req.ClusterID, target.Namespace, target.Name, target.Kind, paused); err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		verb := "暂停"
		if !paused {
			verb = "恢复"
		}
		summary := fmt.Sprintf("已%s Deployment %s/%s 的 rollout。", verb, target.Namespace, target.Name)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type": req.ActionType,
				"paused":      paused,
				"summary":     summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypeRolloutUndo:
		revision, ok := jsonIntValue(changePayload["revision"])
		if !ok {
			revision, _ = jsonIntValue(req.Change["revision"])
		}
		if err := s.k8sSvc.RolloutUndo(ctx, req.ClusterID, target.Namespace, target.Name, target.Kind, revision); err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := fmt.Sprintf("已回滚 Deployment %s/%s。", target.Namespace, target.Name)
		if revision > 0 {
			summary = fmt.Sprintf("已回滚 Deployment %s/%s 到 revision %d。", target.Namespace, target.Name, revision)
		}
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type": req.ActionType,
				"revision":    revision,
				"summary":     summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypeDeleteWorkload:
		gvr, ok := aiActionWorkloadGVR(target.Kind)
		if !ok {
			return ExecuteWorkloadActionResult{}, ErrWithMessage(ErrInvalidParams, "delete workload proposal resource kind is not supported")
		}
		if err := s.k8sSvc.Delete(ctx, req.ClusterID, gvr, target.Namespace, target.Name); err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := fmt.Sprintf("已删除 %s %s/%s。", target.Kind, target.Namespace, target.Name)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type": req.ActionType,
				"summary":     summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypeDeleteResource:
		gvr, namespaced, ok := aiSupportedResourceGVR(target.Kind)
		if !ok {
			return ExecuteWorkloadActionResult{}, ErrWithMessage(ErrInvalidParams, "delete resource proposal kind is not supported")
		}
		targetNamespace := target.Namespace
		if !namespaced {
			targetNamespace = ""
		}
		if err := s.k8sSvc.Delete(ctx, req.ClusterID, gvr, targetNamespace, target.Name); err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := fmt.Sprintf("已删除 %s %s/%s。", target.Kind, targetNamespace, target.Name)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type": req.ActionType,
				"summary":     summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypeDeletePod:
		force := false
		if rawForce, exists := changePayload["force"]; exists {
			force = aiBoolValue(rawForce)
		} else if rawForce, exists := req.Change["force"]; exists {
			force = aiBoolValue(rawForce)
		}
		if err := s.k8sSvc.DeletePod(ctx, req.ClusterID, target.Namespace, target.Name, force); err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := fmt.Sprintf("已删除 Pod %s/%s。", target.Namespace, target.Name)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type": req.ActionType,
				"force":       force,
				"summary":     summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypeCordonNode:
		if err := s.k8sSvc.UpdateNodeSchedulable(ctx, req.ClusterID, target.Name, true); err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := fmt.Sprintf("已将节点 %s 标记为不可调度。", target.Name)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type": req.ActionType,
				"summary":     summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypeUncordonNode:
		if err := s.k8sSvc.UpdateNodeSchedulable(ctx, req.ClusterID, target.Name, false); err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := fmt.Sprintf("已恢复节点 %s 调度。", target.Name)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type": req.ActionType,
				"summary":     summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypeDrainNode:
		timeoutSeconds, ok := jsonIntValue(changePayload["timeout_seconds"])
		if !ok || timeoutSeconds <= 0 {
			timeoutSeconds = 600
		}
		if err := s.k8sSvc.DrainNode(ctx, req.ClusterID, target.Name, DrainNodeOptions{
			TimeoutSeconds:   timeoutSeconds,
			Force:            aiBoolValue(changePayload["force"]),
			IgnoreDaemonSets: !changePayloadBoolOrDefault(changePayload, "include_daemonsets", false),
		}); err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := fmt.Sprintf("已对节点 %s 执行 drain。", target.Name)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type": req.ActionType,
				"summary":     summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypeTriggerCronJob:
		result, err := s.k8sSvc.TriggerCronJob(ctx, req.ClusterID, target.Namespace, target.Name)
		if err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := fmt.Sprintf("已触发 CronJob %s/%s，一次性 Job: %s。", target.Namespace, target.Name, result.JobName)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type": req.ActionType,
				"job_name":    result.JobName,
				"summary":     summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypeSuspendCronJob:
		suspend := true
		if rawSuspend, exists := changePayload["suspend"]; exists {
			suspend = aiBoolValue(rawSuspend)
		} else if rawSuspend, exists := req.Change["suspend"]; exists {
			suspend = aiBoolValue(rawSuspend)
		}
		if err := s.k8sSvc.SuspendCronJob(ctx, req.ClusterID, target.Namespace, target.Name, suspend); err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		verb := "暂停"
		if !suspend {
			verb = "恢复"
		}
		summary := fmt.Sprintf("已%s CronJob %s/%s 调度。", verb, target.Namespace, target.Name)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type": req.ActionType,
				"suspend":     suspend,
				"summary":     summary,
			},
			Summary: summary,
		}, nil
	case aiActionTypeDeleteCompletedJobs:
		namespace := strings.TrimSpace(aiActionString(changePayload["namespace"]))
		if namespace == "" {
			namespace = strings.TrimSpace(target.Namespace)
		}
		olderThanHours, ok := jsonIntValue(changePayload["older_than_hours"])
		if !ok {
			olderThanHours = 0
		}
		deletedCount, err := s.k8sSvc.DeleteCompletedJobs(ctx, req.ClusterID, namespace, olderThanHours)
		if err != nil {
			return ExecuteWorkloadActionResult{}, err
		}
		summary := fmt.Sprintf("已清理命名空间 %s 中 %d 个已完成 Job。", namespace, deletedCount)
		return ExecuteWorkloadActionResult{
			Result: model.JSONMap{
				"action_type":      req.ActionType,
				"namespace":        namespace,
				"deleted_count":    deletedCount,
				"older_than_hours": olderThanHours,
				"summary":          summary,
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

func aiDeleteResourceRiskLevel(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "secret", "persistentvolume", "storageclass", "namespace":
		return "high"
	default:
		return "medium"
	}
}

func aiDeleteResourceConfirmLevel(kind string) string {
	switch aiDeleteResourceRiskLevel(kind) {
	case "high":
		return "double"
	default:
		return "single"
	}
}

func changePayloadBoolOrDefault(payload model.JSONMap, key string, fallback bool) bool {
	if payload == nil {
		return fallback
	}
	raw, exists := payload[key]
	if !exists {
		return fallback
	}
	return aiBoolValue(raw)
}
