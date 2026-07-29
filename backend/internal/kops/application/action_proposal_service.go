package application

import (
	"context"
	"fmt"
	"strings"

	"k8s-platform-backend/internal/kops/domain"
)

// ActionProposalTarget identifies the Kubernetes object affected by an action
// proposal. It intentionally remains free of HTTP and AI-layer DTOs.
type ActionProposalTarget struct {
	Kind      string
	Namespace string
	Name      string
}

type ActionProposalPrepareRequest struct {
	ClusterID  uint64
	ActionType string
	Target     ActionProposalTarget
	Payload    domain.JSONMap
	Reason     string
}

type PreparedActionProposal struct {
	ActionType   string
	Target       ActionProposalTarget
	RiskLevel    string
	ConfirmLevel string
	Title        string
	Summary      string
	Change       domain.JSONMap
	Preview      string
	Diff         string
}

type ActionProposalExecutionRequest struct {
	ClusterID     uint64
	ActionType    string
	Target        ActionProposalTarget
	Change        domain.JSONMap
	ProposalTitle string
	CreatedBy     uint64
	CreatedByName string
}

type ActionProposalExecutionResult struct {
	Result  domain.JSONMap
	Summary string
}

type ActionProposalDrainOptions struct {
	TimeoutSeconds   int
	Force            bool
	IgnoreDaemonSets bool
}

type ActionProposalManifestRequest struct {
	ClusterID        uint64
	YAML             string
	DefaultNamespace string
	SourceLabel      string
	SourceResource   string
	CreatedBy        uint64
	CreatedByName    string
}

type ActionProposalManifestResult struct {
	RecordID uint64
	Status   string
	Summary  string
}

// ActionProposalRuntime is the Kops boundary for cluster mutations. The
// retained legacy runtime implements these calls while orchestration, risk
// policy and input validation live in this application service.
type ActionProposalRuntime interface {
	Inspect(context.Context, uint64, string, ActionProposalTarget) (map[string]any, error)
	Restart(context.Context, uint64, ActionProposalTarget) error
	Scale(context.Context, uint64, ActionProposalTarget, int) error
	UpdateImage(context.Context, uint64, ActionProposalTarget, string, string) error
	Pause(context.Context, uint64, ActionProposalTarget, bool) error
	Undo(context.Context, uint64, ActionProposalTarget, int) error
	DeleteWorkload(context.Context, uint64, ActionProposalTarget) error
	DeleteResource(context.Context, uint64, ActionProposalTarget) error
	DeletePod(context.Context, uint64, ActionProposalTarget, bool) error
	SetNodeSchedulable(context.Context, uint64, string, bool) error
	DrainNode(context.Context, uint64, string, ActionProposalDrainOptions) error
	TriggerCronJob(context.Context, uint64, ActionProposalTarget) (string, error)
	SuspendCronJob(context.Context, uint64, ActionProposalTarget, bool) error
	DeleteCompletedJobs(context.Context, uint64, string, int) (int, error)
	ApplyManifest(context.Context, ActionProposalManifestRequest) (ActionProposalManifestResult, error)
}

type ActionProposalService struct{ runtime ActionProposalRuntime }

func NewActionProposalService(runtime ActionProposalRuntime) *ActionProposalService {
	return &ActionProposalService{runtime: runtime}
}

const (
	proposalActionRestartWorkload      = "restart_workload"
	proposalActionScaleWorkload        = "scale_workload"
	proposalActionUpdateWorkloadImage  = "update_workload_image"
	proposalActionPauseWorkloadRollout = "pause_workload_rollout"
	proposalActionRolloutUndo          = "rollout_undo"
	proposalActionDeleteWorkload       = "delete_workload"
	proposalActionDeleteResource       = "delete_resource"
	proposalActionDeletePod            = "delete_pod"
	proposalActionCordonNode           = "cordon_node"
	proposalActionUncordonNode         = "uncordon_node"
	proposalActionDrainNode            = "drain_node"
	proposalActionTriggerCronJob       = "trigger_cronjob"
	proposalActionSuspendCronJob       = "suspend_cronjob"
	proposalActionDeleteCompletedJobs  = "delete_completed_jobs"
	proposalActionApplyManifest        = "apply_manifest"
)

func (s *ActionProposalService) Prepare(ctx context.Context, req ActionProposalPrepareRequest) (PreparedActionProposal, error) {
	if req.ClusterID == 0 {
		return PreparedActionProposal{}, ErrWithMessage(ErrInvalidParams, "集群参数无效")
	}
	if s == nil || s.runtime == nil {
		return PreparedActionProposal{}, ErrConflict
	}
	target := normalizeActionProposalTarget(req.Target)
	payload := actionProposalPayload(req.Payload)
	reason := strings.TrimSpace(req.Reason)
	actionType := normalizeActionProposalType(req.ActionType)

	inspect := func() (map[string]any, error) {
		return s.runtime.Inspect(ctx, req.ClusterID, actionType, target)
	}
	proposal := func(risk, confirm, title, summary, preview, diff string, change domain.JSONMap) (PreparedActionProposal, error) {
		return PreparedActionProposal{ActionType: actionType, Target: target, RiskLevel: risk, ConfirmLevel: confirm, Title: title, Summary: summary, Preview: preview, Diff: diff, Change: change}, nil
	}

	switch actionType {
	case proposalActionRestartWorkload:
		if !isActionWorkload(target.Kind) || target.Namespace == "" || target.Name == "" {
			return PreparedActionProposal{}, actionInvalid("restart proposal requires a workload target")
		}
		if _, err := inspect(); err != nil {
			return PreparedActionProposal{}, err
		}
		preview := fmt.Sprintf("will restart %s %s/%s", target.Kind, target.Namespace, target.Name)
		return proposal("low", "single", fmt.Sprintf("重启 %s %s/%s", target.Kind, target.Namespace, target.Name), firstActionText(reason, fmt.Sprintf("建议对 %s %s/%s 发起滚动重启", target.Kind, target.Namespace, target.Name)), preview, "", domain.JSONMap{"payload": payload, "preview": preview})

	case proposalActionScaleWorkload:
		if !isScalableActionWorkload(target.Kind) || target.Namespace == "" || target.Name == "" {
			return PreparedActionProposal{}, actionInvalid("scale proposal requires a scalable workload target")
		}
		replicas, ok := actionInteger(payload["replicas"])
		if !ok || replicas < 0 {
			return PreparedActionProposal{}, actionInvalid("scale proposal requires a non-negative replicas value")
		}
		object, err := inspect()
		if err != nil {
			return PreparedActionProposal{}, err
		}
		current := actionNestedInteger(object, "spec", "replicas")
		preview := fmt.Sprintf("replicas: %d -> %d", current, replicas)
		return proposal("low", "single", fmt.Sprintf("调整 %s %s/%s 副本数", target.Kind, target.Namespace, target.Name), firstActionText(reason, fmt.Sprintf("建议将 %s %s/%s 的副本数由 %d 调整为 %d", target.Kind, target.Namespace, target.Name, current, replicas)), preview, preview, domain.JSONMap{"payload": payload, "current_replicas": current, "target_replicas": replicas, "preview": preview})

	case proposalActionUpdateWorkloadImage:
		if !isActionWorkload(target.Kind) || target.Namespace == "" || target.Name == "" {
			return PreparedActionProposal{}, actionInvalid("update image proposal requires a workload target")
		}
		container, image := actionText(payload["container_name"]), actionText(payload["image"])
		if container == "" || image == "" {
			return PreparedActionProposal{}, actionInvalid("update image proposal requires container_name and image")
		}
		if _, err := inspect(); err != nil {
			return PreparedActionProposal{}, err
		}
		preview := fmt.Sprintf("update %s %s/%s container %s to image %s", target.Kind, target.Namespace, target.Name, container, image)
		return proposal("medium", "single", fmt.Sprintf("更新 %s %s/%s 镜像", target.Kind, target.Namespace, target.Name), firstActionText(reason, preview), preview, fmt.Sprintf("container %s image -> %s", container, image), domain.JSONMap{"payload": payload, "container": container, "image": image, "preview": preview})

	case proposalActionPauseWorkloadRollout:
		if target.Kind != "Deployment" || target.Namespace == "" || target.Name == "" {
			return PreparedActionProposal{}, actionInvalid("pause rollout proposal supports Deployment only")
		}
		if _, err := inspect(); err != nil {
			return PreparedActionProposal{}, err
		}
		paused := actionBooleanOr(payload, "paused", true)
		verb := "暂停"
		if !paused {
			verb = "恢复"
		}
		preview := fmt.Sprintf("%s Deployment %s/%s rollout", verb, target.Namespace, target.Name)
		return proposal("medium", "single", fmt.Sprintf("%s Deployment %s/%s Rollout", verb, target.Namespace, target.Name), firstActionText(reason, preview), preview, fmt.Sprintf("spec.paused -> %t", paused), domain.JSONMap{"payload": payload, "paused": paused, "preview": preview})

	case proposalActionRolloutUndo:
		if target.Kind != "Deployment" || target.Namespace == "" || target.Name == "" {
			return PreparedActionProposal{}, actionInvalid("rollout undo proposal supports Deployment only")
		}
		revision, ok := actionInteger(payload["revision"])
		if !ok {
			revision = 0
		}
		if revision < 0 {
			return PreparedActionProposal{}, actionInvalid("rollout revision must be non-negative")
		}
		if _, err := inspect(); err != nil {
			return PreparedActionProposal{}, err
		}
		preview := fmt.Sprintf("rollback Deployment %s/%s", target.Namespace, target.Name)
		if revision > 0 {
			preview += fmt.Sprintf(" to revision %d", revision)
		}
		return proposal("medium", "single", fmt.Sprintf("回滚 Deployment %s/%s", target.Namespace, target.Name), firstActionText(reason, preview), preview, "", domain.JSONMap{"payload": payload, "revision": revision, "preview": preview})

	case proposalActionDeleteWorkload:
		if !isActionWorkload(target.Kind) || target.Namespace == "" || target.Name == "" {
			return PreparedActionProposal{}, actionInvalid("delete workload proposal requires a workload target")
		}
		if _, err := inspect(); err != nil {
			return PreparedActionProposal{}, err
		}
		preview := fmt.Sprintf("delete %s %s/%s", target.Kind, target.Namespace, target.Name)
		return proposal("high", "double", fmt.Sprintf("删除 %s %s/%s", target.Kind, target.Namespace, target.Name), firstActionText(reason, preview), preview, "", domain.JSONMap{"payload": payload, "preview": preview})

	case proposalActionDeleteResource:
		namespaced, supported := actionGenericResourceScope(target.Kind)
		if !supported || target.Name == "" || (namespaced && target.Namespace == "") {
			return PreparedActionProposal{}, actionInvalid("delete resource proposal requires a supported resource target")
		}
		if !namespaced {
			target.Namespace = ""
		}
		if _, err := s.runtime.Inspect(ctx, req.ClusterID, actionType, target); err != nil {
			return PreparedActionProposal{}, err
		}
		preview := fmt.Sprintf("delete %s %s/%s", target.Kind, target.Namespace, target.Name)
		risk, confirm := actionDeleteRisk(target.Kind), "single"
		if risk == "high" {
			confirm = "double"
		}
		return PreparedActionProposal{ActionType: actionType, Target: target, RiskLevel: risk, ConfirmLevel: confirm, Title: fmt.Sprintf("删除 %s %s/%s", target.Kind, target.Namespace, target.Name), Summary: firstActionText(reason, preview), Preview: preview, Change: domain.JSONMap{"payload": payload, "preview": preview}}, nil

	case proposalActionDeletePod:
		if target.Namespace == "" || target.Name == "" {
			return PreparedActionProposal{}, actionInvalid("delete pod proposal requires namespace and name")
		}
		if _, err := inspect(); err != nil {
			return PreparedActionProposal{}, err
		}
		force := actionBooleanOr(payload, "force", false)
		preview := fmt.Sprintf("delete Pod %s/%s", target.Namespace, target.Name)
		if force {
			preview += " with force"
		}
		return proposal("medium", "single", fmt.Sprintf("删除 Pod %s/%s", target.Namespace, target.Name), firstActionText(reason, preview), preview, "", domain.JSONMap{"payload": payload, "force": force, "preview": preview})

	case proposalActionCordonNode, proposalActionUncordonNode:
		if target.Kind != "Node" || target.Name == "" {
			return PreparedActionProposal{}, actionInvalid("node scheduling proposal requires a node target")
		}
		if _, err := inspect(); err != nil {
			return PreparedActionProposal{}, err
		}
		verb, operation := "封锁调度", "cordon"
		if actionType == proposalActionUncordonNode {
			verb, operation = "恢复调度", "uncordon"
		}
		preview := fmt.Sprintf("%s node %s", operation, target.Name)
		return proposal("medium", "single", fmt.Sprintf("%s节点 %s", verb, target.Name), firstActionText(reason, preview), preview, "", domain.JSONMap{"payload": payload, "preview": preview})

	case proposalActionDrainNode:
		if target.Kind != "Node" || target.Name == "" {
			return PreparedActionProposal{}, actionInvalid("drain node proposal requires a node target")
		}
		if _, err := inspect(); err != nil {
			return PreparedActionProposal{}, err
		}
		preview := fmt.Sprintf("drain node %s", target.Name)
		return proposal("high", "double", fmt.Sprintf("驱逐节点 %s 上的工作负载", target.Name), firstActionText(reason, preview), preview, "", domain.JSONMap{"payload": payload, "preview": preview})

	case proposalActionTriggerCronJob:
		if target.Kind != "CronJob" || target.Namespace == "" || target.Name == "" {
			return PreparedActionProposal{}, actionInvalid("trigger cronjob proposal requires a CronJob target")
		}
		if _, err := inspect(); err != nil {
			return PreparedActionProposal{}, err
		}
		preview := fmt.Sprintf("trigger CronJob %s/%s once", target.Namespace, target.Name)
		return proposal("medium", "single", fmt.Sprintf("立即触发 CronJob %s/%s", target.Namespace, target.Name), firstActionText(reason, preview), preview, "", domain.JSONMap{"payload": payload, "preview": preview})

	case proposalActionSuspendCronJob:
		if target.Kind != "CronJob" || target.Namespace == "" || target.Name == "" {
			return PreparedActionProposal{}, actionInvalid("suspend cronjob proposal requires a CronJob target")
		}
		if _, err := inspect(); err != nil {
			return PreparedActionProposal{}, err
		}
		suspend := actionBooleanOr(payload, "suspend", true)
		verb := "暂停"
		if !suspend {
			verb = "恢复"
		}
		preview := fmt.Sprintf("%s CronJob %s/%s 调度", verb, target.Namespace, target.Name)
		return proposal("medium", "single", fmt.Sprintf("%s CronJob %s/%s", verb, target.Namespace, target.Name), firstActionText(reason, preview), preview, fmt.Sprintf("spec.suspend -> %t", suspend), domain.JSONMap{"payload": payload, "suspend": suspend, "preview": preview})

	case proposalActionDeleteCompletedJobs:
		namespace := strings.TrimSpace(target.Namespace)
		if namespace == "" {
			namespace = actionText(payload["namespace"])
		}
		if namespace == "" {
			return PreparedActionProposal{}, actionInvalid("delete completed jobs proposal requires a namespace")
		}
		olderThanHours, ok := actionInteger(payload["older_than_hours"])
		if !ok {
			olderThanHours = 0
		}
		target.Kind, target.Namespace, target.Name = "Job", namespace, "*completed*"
		preview := fmt.Sprintf("delete completed Jobs in namespace %s older than %d hours", namespace, olderThanHours)
		return PreparedActionProposal{ActionType: actionType, Target: target, RiskLevel: "medium", ConfirmLevel: "single", Title: fmt.Sprintf("清理命名空间 %s 的已完成 Job", namespace), Summary: firstActionText(reason, preview), Preview: preview, Change: domain.JSONMap{"payload": payload, "namespace": namespace, "older_than_hours": olderThanHours, "preview": preview}}, nil

	case proposalActionApplyManifest:
		yamlText := actionText(payload["yaml"])
		if yamlText == "" {
			return PreparedActionProposal{}, actionInvalid("manifest proposal requires YAML")
		}
		defaultNamespace := actionText(payload["default_namespace"])
		preview, diff := truncateActionText(yamlText, 240), truncateActionText(yamlText, 1200)
		return proposal("medium", "single", firstActionText(actionText(payload["title"]), "应用 AI 生成的 Manifest"), firstActionText(reason, "请在确认前检查 YAML 内容"), preview, diff, domain.JSONMap{"payload": payload, "default_namespace": defaultNamespace, "preview": truncateActionText(yamlText, 1000)})
	default:
		return PreparedActionProposal{}, actionInvalid("当前动作类型暂不支持")
	}
}

func (s *ActionProposalService) Execute(ctx context.Context, req ActionProposalExecutionRequest) (ActionProposalExecutionResult, error) {
	if req.ClusterID == 0 {
		return ActionProposalExecutionResult{}, ErrWithMessage(ErrInvalidParams, "集群参数无效")
	}
	if s == nil || s.runtime == nil {
		return ActionProposalExecutionResult{}, ErrConflict
	}
	target := normalizeActionProposalTarget(req.Target)
	change := actionProposalPayload(req.Change)
	actionType := normalizeActionProposalType(req.ActionType)
	result := func(summary string, values domain.JSONMap) (ActionProposalExecutionResult, error) {
		if values == nil {
			values = domain.JSONMap{}
		}
		values["action_type"], values["summary"] = actionType, summary
		return ActionProposalExecutionResult{Result: values, Summary: summary}, nil
	}

	switch actionType {
	case proposalActionRestartWorkload:
		if !isActionWorkload(target.Kind) || target.Namespace == "" || target.Name == "" {
			return ActionProposalExecutionResult{}, actionInvalid("restart proposal resource kind is not supported")
		}
		if err := s.runtime.Restart(ctx, req.ClusterID, target); err != nil {
			return ActionProposalExecutionResult{}, err
		}
		return result(fmt.Sprintf("已触发 %s %s/%s 的滚动重启", target.Kind, target.Namespace, target.Name), domain.JSONMap{"target": fmt.Sprintf("%s/%s/%s", target.Kind, target.Namespace, target.Name)})

	case proposalActionScaleWorkload:
		if !isScalableActionWorkload(target.Kind) || target.Namespace == "" || target.Name == "" {
			return ActionProposalExecutionResult{}, actionInvalid("scale proposal resource kind is not supported")
		}
		replicas, ok := actionInteger(change["replicas"])
		if !ok || replicas < 0 {
			replicas, ok = actionInteger(req.Change["target_replicas"])
		}
		if !ok || replicas < 0 {
			return ActionProposalExecutionResult{}, actionInvalid("scale proposal requires a non-negative replicas value")
		}
		if err := s.runtime.Scale(ctx, req.ClusterID, target, replicas); err != nil {
			return ActionProposalExecutionResult{}, err
		}
		return result(fmt.Sprintf("已将 %s %s/%s 的副本数调整为 %d", target.Kind, target.Namespace, target.Name, replicas), domain.JSONMap{"target_replicas": replicas})

	case proposalActionUpdateWorkloadImage:
		if !isActionWorkload(target.Kind) || target.Namespace == "" || target.Name == "" {
			return ActionProposalExecutionResult{}, actionInvalid("update image proposal resource kind is not supported")
		}
		container, image := actionText(change["container_name"]), actionText(change["image"])
		if container == "" {
			container = actionText(req.Change["container"])
		}
		if image == "" {
			image = actionText(req.Change["image"])
		}
		if container == "" || image == "" {
			return ActionProposalExecutionResult{}, actionInvalid("update image proposal requires container_name and image")
		}
		if err := s.runtime.UpdateImage(ctx, req.ClusterID, target, container, image); err != nil {
			return ActionProposalExecutionResult{}, err
		}
		return result(fmt.Sprintf("已将 %s %s/%s 中容器 %s 的镜像更新为 %s", target.Kind, target.Namespace, target.Name, container, image), domain.JSONMap{"container_name": container, "image": image})

	case proposalActionPauseWorkloadRollout:
		if target.Kind != "Deployment" || target.Namespace == "" || target.Name == "" {
			return ActionProposalExecutionResult{}, actionInvalid("pause rollout proposal supports Deployment only")
		}
		paused := actionBooleanOr(change, "paused", actionBooleanOr(req.Change, "paused", true))
		if err := s.runtime.Pause(ctx, req.ClusterID, target, paused); err != nil {
			return ActionProposalExecutionResult{}, err
		}
		verb := "暂停"
		if !paused {
			verb = "恢复"
		}
		return result(fmt.Sprintf("已%s Deployment %s/%s 的 rollout", verb, target.Namespace, target.Name), domain.JSONMap{"paused": paused})

	case proposalActionRolloutUndo:
		if target.Kind != "Deployment" || target.Namespace == "" || target.Name == "" {
			return ActionProposalExecutionResult{}, actionInvalid("rollout undo proposal supports Deployment only")
		}
		revision, ok := actionInteger(change["revision"])
		if !ok {
			revision, _ = actionInteger(req.Change["revision"])
		}
		if revision < 0 {
			return ActionProposalExecutionResult{}, actionInvalid("rollout revision must be non-negative")
		}
		if err := s.runtime.Undo(ctx, req.ClusterID, target, revision); err != nil {
			return ActionProposalExecutionResult{}, err
		}
		summary := fmt.Sprintf("已回滚 Deployment %s/%s", target.Namespace, target.Name)
		if revision > 0 {
			summary += fmt.Sprintf(" 到 revision %d", revision)
		}
		return result(summary, domain.JSONMap{"revision": revision})

	case proposalActionDeleteWorkload:
		if !isActionWorkload(target.Kind) || target.Namespace == "" || target.Name == "" {
			return ActionProposalExecutionResult{}, actionInvalid("delete workload proposal resource kind is not supported")
		}
		if err := s.runtime.DeleteWorkload(ctx, req.ClusterID, target); err != nil {
			return ActionProposalExecutionResult{}, err
		}
		return result(fmt.Sprintf("已删除 %s %s/%s", target.Kind, target.Namespace, target.Name), nil)

	case proposalActionDeleteResource:
		namespaced, supported := actionGenericResourceScope(target.Kind)
		if !supported || target.Name == "" || (namespaced && target.Namespace == "") {
			return ActionProposalExecutionResult{}, actionInvalid("delete resource proposal kind is not supported")
		}
		if !namespaced {
			target.Namespace = ""
		}
		if err := s.runtime.DeleteResource(ctx, req.ClusterID, target); err != nil {
			return ActionProposalExecutionResult{}, err
		}
		return result(fmt.Sprintf("已删除 %s %s/%s", target.Kind, target.Namespace, target.Name), nil)

	case proposalActionDeletePod:
		if target.Namespace == "" || target.Name == "" {
			return ActionProposalExecutionResult{}, actionInvalid("delete pod proposal requires namespace and name")
		}
		force := actionBooleanOr(change, "force", actionBooleanOr(req.Change, "force", false))
		if err := s.runtime.DeletePod(ctx, req.ClusterID, target, force); err != nil {
			return ActionProposalExecutionResult{}, err
		}
		return result(fmt.Sprintf("已删除 Pod %s/%s", target.Namespace, target.Name), domain.JSONMap{"force": force})

	case proposalActionCordonNode, proposalActionUncordonNode:
		if target.Kind != "Node" || target.Name == "" {
			return ActionProposalExecutionResult{}, actionInvalid("node scheduling proposal requires a node target")
		}
		unschedulable := actionType == proposalActionCordonNode
		if err := s.runtime.SetNodeSchedulable(ctx, req.ClusterID, target.Name, unschedulable); err != nil {
			return ActionProposalExecutionResult{}, err
		}
		verb := "恢复"
		if unschedulable {
			verb = "标记为不可"
		}
		return result(fmt.Sprintf("已将节点 %s %s调度", target.Name, verb), nil)

	case proposalActionDrainNode:
		if target.Kind != "Node" || target.Name == "" {
			return ActionProposalExecutionResult{}, actionInvalid("drain node proposal requires a node target")
		}
		timeout, ok := actionInteger(change["timeout_seconds"])
		if !ok || timeout <= 0 {
			timeout = 600
		}
		options := ActionProposalDrainOptions{TimeoutSeconds: timeout, Force: actionBooleanOr(change, "force", false), IgnoreDaemonSets: !actionBooleanOr(change, "include_daemonsets", false)}
		if err := s.runtime.DrainNode(ctx, req.ClusterID, target.Name, options); err != nil {
			return ActionProposalExecutionResult{}, err
		}
		return result(fmt.Sprintf("已对节点 %s 执行 drain", target.Name), nil)

	case proposalActionTriggerCronJob:
		if target.Kind != "CronJob" || target.Namespace == "" || target.Name == "" {
			return ActionProposalExecutionResult{}, actionInvalid("trigger cronjob proposal requires a CronJob target")
		}
		jobName, err := s.runtime.TriggerCronJob(ctx, req.ClusterID, target)
		if err != nil {
			return ActionProposalExecutionResult{}, err
		}
		return result(fmt.Sprintf("已触发 CronJob %s/%s，一次性 Job: %s", target.Namespace, target.Name, jobName), domain.JSONMap{"job_name": jobName})

	case proposalActionSuspendCronJob:
		if target.Kind != "CronJob" || target.Namespace == "" || target.Name == "" {
			return ActionProposalExecutionResult{}, actionInvalid("suspend cronjob proposal requires a CronJob target")
		}
		suspend := actionBooleanOr(change, "suspend", actionBooleanOr(req.Change, "suspend", true))
		if err := s.runtime.SuspendCronJob(ctx, req.ClusterID, target, suspend); err != nil {
			return ActionProposalExecutionResult{}, err
		}
		verb := "暂停"
		if !suspend {
			verb = "恢复"
		}
		return result(fmt.Sprintf("已%s CronJob %s/%s 调度", verb, target.Namespace, target.Name), domain.JSONMap{"suspend": suspend})

	case proposalActionDeleteCompletedJobs:
		namespace := actionText(change["namespace"])
		if namespace == "" {
			namespace = target.Namespace
		}
		if namespace == "" {
			return ActionProposalExecutionResult{}, actionInvalid("delete completed jobs proposal requires a namespace")
		}
		olderThanHours, ok := actionInteger(change["older_than_hours"])
		if !ok {
			olderThanHours = 0
		}
		if olderThanHours < 0 {
			return ActionProposalExecutionResult{}, actionInvalid("older_than_hours must be non-negative")
		}
		deleted, err := s.runtime.DeleteCompletedJobs(ctx, req.ClusterID, namespace, olderThanHours)
		if err != nil {
			return ActionProposalExecutionResult{}, err
		}
		return result(fmt.Sprintf("已清理命名空间 %s 中 %d 个已完成 Job", namespace, deleted), domain.JSONMap{"namespace": namespace, "deleted_count": deleted, "older_than_hours": olderThanHours})

	case proposalActionApplyManifest:
		yamlText := actionText(change["yaml"])
		if yamlText == "" {
			return ActionProposalExecutionResult{}, actionInvalid("manifest proposal requires YAML")
		}
		manifest, err := s.runtime.ApplyManifest(ctx, ActionProposalManifestRequest{ClusterID: req.ClusterID, YAML: yamlText, DefaultNamespace: actionText(change["default_namespace"]), SourceLabel: "AI 动作提案", SourceResource: strings.TrimSpace(req.ProposalTitle), CreatedBy: req.CreatedBy, CreatedByName: strings.TrimSpace(req.CreatedByName)})
		if err != nil {
			return ActionProposalExecutionResult{}, err
		}
		return result(firstActionText(manifest.Summary, "Manifest 已执行"), domain.JSONMap{"record_id": manifest.RecordID, "status": manifest.Status})
	default:
		return ActionProposalExecutionResult{}, actionInvalid("当前动作类型暂不支持执行")
	}
}

func actionInvalid(message string) error { return ErrWithMessage(ErrInvalidParams, message) }

func normalizeActionProposalType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case proposalActionRestartWorkload, proposalActionScaleWorkload, proposalActionUpdateWorkloadImage, proposalActionPauseWorkloadRollout,
		proposalActionRolloutUndo, proposalActionDeleteWorkload, proposalActionDeleteResource, proposalActionDeletePod,
		proposalActionCordonNode, proposalActionUncordonNode, proposalActionDrainNode, proposalActionTriggerCronJob,
		proposalActionSuspendCronJob, proposalActionDeleteCompletedJobs, proposalActionApplyManifest:
		return value
	default:
		return ""
	}
}

func normalizeActionProposalTarget(target ActionProposalTarget) ActionProposalTarget {
	kind := strings.TrimSpace(target.Kind)
	switch strings.ToLower(kind) {
	case "deployment":
		kind = "Deployment"
	case "statefulset":
		kind = "StatefulSet"
	case "daemonset":
		kind = "DaemonSet"
	case "replicaset":
		kind = "ReplicaSet"
	case "pod":
		kind = "Pod"
	case "node":
		kind = "Node"
	case "job":
		kind = "Job"
	case "cronjob":
		kind = "CronJob"
	case "namespace":
		kind = "Namespace"
	case "service":
		kind = "Service"
	case "ingress":
		kind = "Ingress"
	case "configmap":
		kind = "ConfigMap"
	case "secret":
		kind = "Secret"
	case "pvc", "persistentvolumeclaim":
		kind = "PersistentVolumeClaim"
	case "pv", "persistentvolume":
		kind = "PersistentVolume"
	case "storageclass":
		kind = "StorageClass"
	case "serviceaccount":
		kind = "ServiceAccount"
	case "networkpolicy":
		kind = "NetworkPolicy"
	case "role":
		kind = "Role"
	case "clusterrole":
		kind = "ClusterRole"
	case "rolebinding":
		kind = "RoleBinding"
	case "clusterrolebinding":
		kind = "ClusterRoleBinding"
	case "horizontalpodautoscaler", "hpa":
		kind = "HorizontalPodAutoscaler"
	case "poddisruptionbudget", "pdb":
		kind = "PodDisruptionBudget"
	case "endpoint":
		kind = "Endpoint"
	case "endpointslice":
		kind = "EndpointSlice"
	case "lease":
		kind = "Lease"
	case "resourcequota":
		kind = "ResourceQuota"
	case "limitrange":
		kind = "LimitRange"
	case "customresourcedefinition", "crd":
		kind = "CustomResourceDefinition"
	case "apiservice":
		kind = "APIService"
	case "priorityclass":
		kind = "PriorityClass"
	case "runtimeclass":
		kind = "RuntimeClass"
	case "ingressclass":
		kind = "IngressClass"
	case "csidriver":
		kind = "CSIDriver"
	case "csinode":
		kind = "CSINode"
	case "csistoragecapacity":
		kind = "CSIStorageCapacity"
	case "volumeattachment":
		kind = "VolumeAttachment"
	case "volumesnapshot":
		kind = "VolumeSnapshot"
	case "volumesnapshotclass":
		kind = "VolumeSnapshotClass"
	case "volumesnapshotcontent":
		kind = "VolumeSnapshotContent"
	case "validatingwebhookconfiguration":
		kind = "ValidatingWebhookConfiguration"
	case "mutatingwebhookconfiguration":
		kind = "MutatingWebhookConfiguration"
	case "validatingadmissionpolicy":
		kind = "ValidatingAdmissionPolicy"
	case "validatingadmissionpolicybinding":
		kind = "ValidatingAdmissionPolicyBinding"
	}
	return ActionProposalTarget{Kind: kind, Namespace: strings.TrimSpace(target.Namespace), Name: strings.TrimSpace(target.Name)}
}

func isActionWorkload(kind string) bool {
	return kind == "Deployment" || kind == "StatefulSet" || kind == "DaemonSet"
}

func isScalableActionWorkload(kind string) bool { return kind == "Deployment" || kind == "StatefulSet" }

func actionGenericResourceScope(kind string) (bool, bool) {
	_, namespaced := map[string]bool{
		"Namespace": false, "Node": false, "Pod": true, "Deployment": true, "StatefulSet": true, "DaemonSet": true, "ReplicaSet": true,
		"Service": true, "Ingress": true, "IngressClass": false, "NetworkPolicy": true, "ConfigMap": true, "Secret": true,
		"ServiceAccount": true, "Endpoint": true, "EndpointSlice": true, "Lease": true, "PodDisruptionBudget": true,
		"Role": true, "ClusterRole": false, "RoleBinding": true, "ClusterRoleBinding": false, "HorizontalPodAutoscaler": true,
		"Job": true, "CronJob": true, "PersistentVolumeClaim": true, "PersistentVolume": false, "StorageClass": false,
		"CSIDriver": false, "CSINode": false, "CSIStorageCapacity": true, "VolumeAttachment": false, "VolumeSnapshot": true,
		"VolumeSnapshotClass": false, "VolumeSnapshotContent": false, "ResourceQuota": true, "LimitRange": true,
		"CustomResourceDefinition": false, "APIService": false, "PriorityClass": false, "RuntimeClass": false,
		"ValidatingWebhookConfiguration": false, "MutatingWebhookConfiguration": false, "ValidatingAdmissionPolicy": false,
		"ValidatingAdmissionPolicyBinding": false,
	}[kind]
	_, supported := map[string]struct{}{
		"Namespace": {}, "Node": {}, "Pod": {}, "Deployment": {}, "StatefulSet": {}, "DaemonSet": {}, "ReplicaSet": {}, "Service": {}, "Ingress": {}, "IngressClass": {}, "NetworkPolicy": {}, "ConfigMap": {}, "Secret": {}, "ServiceAccount": {}, "Endpoint": {}, "EndpointSlice": {}, "Lease": {}, "PodDisruptionBudget": {}, "Role": {}, "ClusterRole": {}, "RoleBinding": {}, "ClusterRoleBinding": {}, "HorizontalPodAutoscaler": {}, "Job": {}, "CronJob": {}, "PersistentVolumeClaim": {}, "PersistentVolume": {}, "StorageClass": {}, "CSIDriver": {}, "CSINode": {}, "CSIStorageCapacity": {}, "VolumeAttachment": {}, "VolumeSnapshot": {}, "VolumeSnapshotClass": {}, "VolumeSnapshotContent": {}, "ResourceQuota": {}, "LimitRange": {}, "CustomResourceDefinition": {}, "APIService": {}, "PriorityClass": {}, "RuntimeClass": {}, "ValidatingWebhookConfiguration": {}, "MutatingWebhookConfiguration": {}, "ValidatingAdmissionPolicy": {}, "ValidatingAdmissionPolicyBinding": {},
	}[kind]
	return namespaced, supported
}

func actionDeleteRisk(kind string) string {
	switch kind {
	case "Secret", "PersistentVolume", "StorageClass", "Namespace":
		return "high"
	default:
		return "medium"
	}
}

func actionProposalPayload(value domain.JSONMap) domain.JSONMap {
	if value == nil {
		return domain.JSONMap{}
	}
	if payload, ok := value["payload"].(map[string]any); ok && payload != nil {
		return domain.JSONMap(payload)
	}
	return value
}

func actionText(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func actionInteger(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case uint:
		return int(typed), true
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return int(typed), true
	case uint64:
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func actionNestedInteger(object map[string]any, keys ...string) int {
	if len(keys) == 0 {
		return 0
	}
	current := object
	for _, key := range keys[:len(keys)-1] {
		next, _ := current[key].(map[string]any)
		if next == nil {
			return 0
		}
		current = next
	}
	value, ok := actionInteger(current[keys[len(keys)-1]])
	if !ok {
		return 0
	}
	return value
}

func actionBooleanOr(values map[string]any, key string, fallback bool) bool {
	value, found := values[key]
	if !found {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return fallback
	}
}

func firstActionText(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func truncateActionText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len([]rune(value)) <= limit {
		return value
	}
	return string([]rune(value)[:limit]) + "..."
}
