package application

import (
	"regexp"
	"strconv"
	"strings"

	"k8s-platform-backend/internal/ai/domain"
)

// ChatActionIntentRequest is the resource scope that gives a chat command an
// executable target. It is deliberately separate from the HTTP request DTO.
type ChatActionIntentRequest struct {
	ResourceKind string
	Namespace    string
	ResourceName string
}

// AutoActionProposalSpec is a deterministic proposal candidate. Creation and
// confirmation still happen through the action execution service.
type AutoActionProposalSpec struct {
	ProposalType   string
	TargetResource ActionTargetResource
	Payload        domain.JSONMap
	Reason         string
}

var scaleReplicaPatterns = []*regexp.Regexp{
	regexp.MustCompile("(?i)(?:replicas?|副本(?:数)?)[^\\d]{0,8}(\\d{1,4})"),
	regexp.MustCompile("(?i)(?:调整到|改到|改成|改为|设为|设置为|to|=|为|到)[^\\d]{0,4}(\\d{1,4})\\s*(?:replicas?|副本)?"),
	regexp.MustCompile("(?i)(?:scale|扩容|缩容|扩缩容|横向扩展|横向缩容)[^\\d]{0,10}(\\d{1,4})"),
}

// BuildAutoActionProposalSpecs accepts only explicit, non-tentative commands.
// It keeps intent recognition under AI application ownership instead of tying
// it to the legacy chat transport implementation.
func BuildAutoActionProposalSpecs(request ChatActionIntentRequest, message string) []AutoActionProposalSpec {
	target := NormalizeActionTarget(ActionTargetResource{Kind: request.ResourceKind, Namespace: request.Namespace, Name: request.ResourceName})
	if target.Kind == "" || target.Name == "" || (!strings.EqualFold(target.Kind, "Node") && target.Namespace == "") {
		return nil
	}
	message = strings.TrimSpace(message)
	if message == "" || hasTentativeActionIntent(message) {
		return nil
	}

	specs := make([]AutoActionProposalSpec, 0, 4)
	if _, workload := ActionWorkloadGVR(target.Kind); workload {
		if isExplicitRestartIntent(message) {
			specs = append(specs, AutoActionProposalSpec{ProposalType: ActionTypeRestartWorkload, TargetResource: target, Payload: domain.JSONMap{}, Reason: "Auto-generated rollout restart proposal from the latest chat intent. Please verify scope before confirming."})
		}
		if !strings.EqualFold(target.Kind, "DaemonSet") {
			if replicas, ok := detectScaleReplicaTarget(message); ok && isExplicitScaleIntent(message) {
				specs = append(specs, AutoActionProposalSpec{ProposalType: ActionTypeScaleWorkload, TargetResource: target, Payload: domain.JSONMap{"replicas": replicas}, Reason: "Auto-generated scale proposal from the latest chat intent. Please verify target replicas before confirming."})
			}
		}
		if isExplicitDeleteIntent(message) {
			specs = append(specs, AutoActionProposalSpec{ProposalType: ActionTypeDeleteWorkload, TargetResource: target, Payload: domain.JSONMap{}, Reason: "Auto-generated delete proposal from chat intent. This is a HIGH RISK operation requiring double confirmation."})
		}
		if isExplicitUpdateImageIntent(message) {
			if image := detectImageFromMessage(message); image != "" {
				specs = append(specs, AutoActionProposalSpec{ProposalType: ActionTypeUpdateWorkloadImage, TargetResource: target, Payload: domain.JSONMap{"new_image": image}, Reason: "Auto-generated image update proposal. Please verify the new image before confirming."})
			}
		}
	}
	if strings.EqualFold(target.Kind, "Pod") && isExplicitDeletePodIntent(message) {
		specs = append(specs, AutoActionProposalSpec{ProposalType: ActionTypeDeletePod, TargetResource: target, Payload: domain.JSONMap{}, Reason: "Auto-generated pod deletion proposal. Pod will be recreated by its controller if managed."})
	}
	if strings.EqualFold(target.Kind, "Node") {
		if isExplicitCordonIntent(message) {
			specs = append(specs, AutoActionProposalSpec{ProposalType: ActionTypeCordonNode, TargetResource: target, Payload: domain.JSONMap{}, Reason: "Auto-generated node cordon proposal. Node will stop scheduling new pods."})
		}
		if isExplicitDrainIntent(message) {
			specs = append(specs, AutoActionProposalSpec{ProposalType: ActionTypeDrainNode, TargetResource: target, Payload: domain.JSONMap{}, Reason: "Auto-generated node drain proposal. This is a HIGH RISK operation requiring double confirmation. All pods will be evicted."})
		}
	}
	return specs
}

func isExplicitRestartIntent(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if strings.Contains(lower, "rollout restart") || strings.Contains(message, "\u6eda\u52a8\u91cd\u542f") {
		return true
	}
	if !strings.Contains(lower, "restart") && !strings.Contains(message, "\u91cd\u542f") {
		return false
	}
	if strings.HasPrefix(lower, "restart ") || strings.HasPrefix(lower, "please restart") {
		return true
	}
	return containsAny(message, "\u5e2e\u6211\u91cd\u542f", "\u8bf7\u91cd\u542f", "\u6267\u884c\u91cd\u542f", "\u53d1\u8d77\u91cd\u542f", "\u5148\u91cd\u542f", "\u7acb\u5373\u91cd\u542f")
}

func isExplicitScaleIntent(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	return containsAny(lower, "scale", "replica", "replicas") || containsAny(message, "\u6269\u5bb9", "\u7f29\u5bb9", "\u6269\u7f29\u5bb9", "\u526f\u672c", "\u526f\u672c\u6570", "\u8c03\u6574\u526f\u672c", "\u8c03\u6574\u5230", "\u6539\u6210", "\u6539\u4e3a")
}

func detectScaleReplicaTarget(message string) (int, bool) {
	for _, pattern := range scaleReplicaPatterns {
		matches := pattern.FindStringSubmatch(message)
		if len(matches) < 2 {
			continue
		}
		replicas, err := strconv.Atoi(matches[1])
		if err == nil && replicas >= 0 {
			return replicas, true
		}
	}
	return 0, false
}

func hasTentativeActionIntent(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if strings.Contains(lower, "?") || strings.Contains(message, "\uff1f") {
		return true
	}
	return containsAny(lower, "should restart", "should scale", "can we restart", "can we scale", "could we restart", "could we scale", "need to restart", "need to scale") || containsAny(message, "\u662f\u5426\u91cd\u542f", "\u8981\u4e0d\u8981\u91cd\u542f", "\u662f\u4e0d\u662f\u8981\u91cd\u542f", "\u9700\u4e0d\u9700\u8981\u91cd\u542f", "\u662f\u5426\u6269\u5bb9", "\u662f\u5426\u7f29\u5bb9", "\u8981\u4e0d\u8981\u6269\u5bb9", "\u8981\u4e0d\u8981\u7f29\u5bb9", "\u9700\u4e0d\u9700\u8981\u6269\u5bb9", "\u9700\u4e0d\u9700\u8981\u7f29\u5bb9", "\u53ef\u4ee5\u91cd\u542f\u5417", "\u53ef\u4ee5\u6269\u5bb9\u5417", "\u53ef\u4ee5\u7f29\u5bb9\u5417")
}

func isExplicitDeleteIntent(message string) bool {
	lower := strings.ToLower(message)
	for _, keyword := range []string{"delete", "remove", "删除", "移除"} {
		if strings.Contains(lower, keyword) {
			return !hasTentativeActionIntent(message)
		}
	}
	return false
}

func isExplicitDeletePodIntent(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "delete") || strings.Contains(lower, "删除") || strings.Contains(lower, "移除")
}

func isExplicitCordonIntent(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "cordon") || strings.Contains(lower, "封锁") || strings.Contains(lower, "停止调度")
}

func isExplicitDrainIntent(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "drain") || strings.Contains(lower, "驱逐") || strings.Contains(lower, "排空")
}

func isExplicitUpdateImageIntent(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "update image") || strings.Contains(lower, "change image") || strings.Contains(lower, "更新镜像") || strings.Contains(lower, "修改镜像") || strings.Contains(lower, "rollout image")
}

func detectImageFromMessage(message string) string {
	lower := strings.ToLower(message)
	index := strings.Index(lower, "image:")
	if index < 0 {
		return ""
	}
	value := message[index+len("image:"):]
	if end := strings.IndexAny(value, " \n\t,，"); end > 0 {
		return strings.TrimSpace(value[:end])
	}
	return strings.TrimSpace(value)
}

func containsAny(text string, candidates ...string) bool {
	for _, candidate := range candidates {
		if candidate != "" && strings.Contains(text, candidate) {
			return true
		}
	}
	return false
}
