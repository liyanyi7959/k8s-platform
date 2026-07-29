package application

import (
	"encoding/json"
	"regexp"
	"strings"

	"k8s-platform-backend/internal/ai/domain"
)

const actionsJSONTag = "AI_ACTIONS_JSON:"

var (
	toolCallBlockPattern       = regexp.MustCompile(`(?is)<tool_call>.*?</tool_call>`)
	functionCallBlockPattern   = regexp.MustCompile(`(?is)<function=[^>]+>.*?</function>`)
	excessBlankLinePattern     = regexp.MustCompile(`\n{3,}`)
	leadingTrailingSpacePatter = regexp.MustCompile(`[ \t]+\n`)
)

type actionsEnvelope struct {
	SuggestedActions []domain.AISuggestedAction `json:"suggested_actions"`
}

// NormalizeModelAnswer removes provider-side tool-call fragments and converts
// the optional AI_ACTIONS_JSON envelope into the structured message payload.
func NormalizeModelAnswer(content string) (string, domain.JSONMap) {
	cleanContent, actions := extractStructuredSuggestions(content)
	normalized := strings.TrimSpace(cleanContent)
	guardStructured := domain.JSONMap{}
	strippedToolCall := false

	next := toolCallBlockPattern.ReplaceAllString(normalized, "\n")
	if next != normalized {
		strippedToolCall = true
		normalized = next
	}
	next = functionCallBlockPattern.ReplaceAllString(normalized, "\n")
	if next != normalized {
		strippedToolCall = true
		normalized = next
	}
	normalized = leadingTrailingSpacePatter.ReplaceAllString(normalized, "\n")
	normalized = excessBlankLinePattern.ReplaceAllString(normalized, "\n\n")
	normalized = strings.TrimSpace(normalized)

	if strippedToolCall {
		guardStructured["response_guard"] = domain.JSONMap{
			"tool_call_stripped": true,
		}
	}
	if normalized == "" {
		normalized = "本轮模型返回了无效的工具调用片段，平台已忽略该内容。请重试，或缩小范围后再次提问。"
		guardStructured["response_guard"] = domain.JSONMap{
			"tool_call_stripped": true,
			"fallback_applied":   true,
		}
	}

	return normalized, MergeStructuredPayload(
		buildSuggestedActionsStructured(actions),
		guardStructured,
	)
}

// MergeStructuredPayload combines independent message annotations; later
// payloads intentionally take precedence for the same field.
func MergeStructuredPayload(parts ...domain.JSONMap) domain.JSONMap {
	out := domain.JSONMap{}
	for _, part := range parts {
		for key, value := range part {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func extractStructuredSuggestions(content string) (string, []domain.AISuggestedAction) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", nil
	}

	idx := strings.LastIndex(trimmed, actionsJSONTag)
	if idx < 0 {
		return trimmed, nil
	}

	jsonText := strings.TrimSpace(trimmed[idx+len(actionsJSONTag):])
	cleanContent := strings.TrimSpace(trimmed[:idx])
	if jsonText == "" {
		return cleanContent, nil
	}

	var envelope actionsEnvelope
	if err := json.Unmarshal([]byte(jsonText), &envelope); err != nil {
		return trimmed, nil
	}
	return cleanContent, normalizeSuggestedActions(envelope.SuggestedActions)
}

func normalizeSuggestedActions(items []domain.AISuggestedAction) []domain.AISuggestedAction {
	result := make([]domain.AISuggestedAction, 0, len(items))
	for _, item := range items {
		actionType := NormalizeActionType(item.ActionType)
		if actionType == "" {
			continue
		}

		target := NormalizeActionTarget(ActionTargetResource{
			Kind: item.TargetKind, Namespace: item.TargetNamespace, Name: item.TargetName,
		})
		normalized := domain.AISuggestedAction{
			ActionType: actionType, Title: strings.TrimSpace(item.Title), Reason: strings.TrimSpace(item.Reason),
			TargetKind: target.Kind, TargetNamespace: target.Namespace, TargetName: target.Name,
			ManifestYAML: strings.TrimSpace(item.ManifestYAML), DefaultNamespace: strings.TrimSpace(item.DefaultNamespace),
			RequiresConfirmation: true,
		}
		if normalized.Title == "" {
			normalized.Title = defaultSuggestedActionTitle(normalized)
		}
		if normalized.Reason == "" {
			normalized.Reason = "AI suggested this action based on the current conversation context."
		}
		normalized.RiskLevel = normalizeSuggestedActionRisk(item.RiskLevel, normalized.ActionType)

		switch normalized.ActionType {
		case ActionTypeRestartWorkload:
			if _, ok := ActionWorkloadGVR(normalized.TargetKind); !ok || normalized.TargetNamespace == "" || normalized.TargetName == "" {
				continue
			}
			normalized.AutoProposalEligible = true
		case ActionTypeScaleWorkload:
			if _, ok := ActionWorkloadGVR(normalized.TargetKind); !ok || normalized.TargetNamespace == "" || normalized.TargetName == "" || item.Replicas == nil || *item.Replicas < 0 || strings.EqualFold(normalized.TargetKind, "DaemonSet") {
				continue
			}
			replicas := *item.Replicas
			normalized.Replicas = &replicas
			normalized.AutoProposalEligible = true
		case ActionTypeUpdateWorkloadImage, ActionTypePauseWorkloadRollout, ActionTypeRolloutUndo, ActionTypeDeleteWorkload:
			if _, ok := ActionWorkloadGVR(normalized.TargetKind); !ok || normalized.TargetNamespace == "" || normalized.TargetName == "" {
				continue
			}
			normalized.AutoProposalEligible = true
		case ActionTypeDeleteResource:
			if normalized.TargetKind == "" || normalized.TargetName == "" {
				continue
			}
			normalized.AutoProposalEligible = true
		case ActionTypeDeletePod:
			if !strings.EqualFold(normalized.TargetKind, "Pod") || normalized.TargetNamespace == "" || normalized.TargetName == "" {
				continue
			}
			normalized.AutoProposalEligible = true
		case ActionTypeCordonNode, ActionTypeUncordonNode, ActionTypeDrainNode:
			if !strings.EqualFold(normalized.TargetKind, "Node") || normalized.TargetName == "" {
				continue
			}
			normalized.AutoProposalEligible = true
		case ActionTypeTriggerCronJob, ActionTypeSuspendCronJob:
			if !strings.EqualFold(normalized.TargetKind, "CronJob") || normalized.TargetNamespace == "" || normalized.TargetName == "" {
				continue
			}
			normalized.AutoProposalEligible = true
		case ActionTypeDeleteCompletedJobs:
			if normalized.TargetNamespace == "" && normalized.DefaultNamespace == "" {
				continue
			}
			if normalized.TargetKind == "" {
				normalized.TargetKind = "Job"
			}
			if normalized.TargetNamespace == "" {
				normalized.TargetNamespace = normalized.DefaultNamespace
			}
			normalized.AutoProposalEligible = true
		case ActionTypeApplyManifest:
			if normalized.ManifestYAML == "" {
				continue
			}
			if normalized.DefaultNamespace == "" {
				normalized.DefaultNamespace = normalized.TargetNamespace
			}
			normalized.AutoProposalEligible = true
		}
		result = append(result, normalized)
	}
	return result
}

func defaultSuggestedActionTitle(item domain.AISuggestedAction) string {
	switch item.ActionType {
	case ActionTypeRestartWorkload:
		return "Suggested restart workload action"
	case ActionTypeScaleWorkload:
		return "Suggested scale workload action"
	case ActionTypeUpdateWorkloadImage:
		return "Suggested workload image update"
	case ActionTypePauseWorkloadRollout:
		return "Suggested workload rollout pause or resume"
	case ActionTypeRolloutUndo:
		return "Suggested workload rollback action"
	case ActionTypeDeleteWorkload, ActionTypeDeleteResource, ActionTypeDeletePod:
		return "Suggested delete action"
	case ActionTypeCordonNode, ActionTypeUncordonNode, ActionTypeDrainNode:
		return "Suggested node action"
	case ActionTypeTriggerCronJob, ActionTypeSuspendCronJob, ActionTypeDeleteCompletedJobs:
		return "Suggested batch action"
	case ActionTypeApplyManifest:
		return "Suggested apply manifest action"
	default:
		return "Suggested action"
	}
}

func normalizeSuggestedActionRisk(input string, actionType string) string {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(input))
	}
	switch actionType {
	case ActionTypeRestartWorkload, ActionTypeScaleWorkload:
		return "low"
	case ActionTypeDeleteWorkload, ActionTypeDrainNode:
		return "high"
	default:
		return "medium"
	}
}

func buildSuggestedActionsStructured(actions []domain.AISuggestedAction) domain.JSONMap {
	if len(actions) == 0 {
		return nil
	}
	payload := make([]map[string]any, 0, len(actions))
	for _, action := range actions {
		item := map[string]any{
			"action_type": action.ActionType, "title": action.Title, "reason": action.Reason,
			"target_kind": action.TargetKind, "target_namespace": action.TargetNamespace, "target_name": action.TargetName,
			"risk_level": action.RiskLevel, "requires_confirmation": action.RequiresConfirmation,
			"auto_proposal_eligible": action.AutoProposalEligible, "proposal_created": action.ProposalCreated,
			"default_namespace": action.DefaultNamespace,
		}
		if action.Replicas != nil {
			item["replicas"] = *action.Replicas
		}
		if action.ManifestYAML != "" {
			item["manifest_yaml"] = action.ManifestYAML
		}
		if action.ProposalID != nil {
			item["proposal_id"] = *action.ProposalID
		}
		if action.MessageID != nil {
			item["message_id"] = *action.MessageID
		}
		payload = append(payload, item)
	}
	return domain.JSONMap{"suggested_actions": payload}
}
