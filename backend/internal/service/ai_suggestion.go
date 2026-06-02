package service

import (
	"encoding/json"
	"strings"

	"k8s-platform-backend/internal/model"
)

const aiActionsJSONTag = "AI_ACTIONS_JSON:"

type AISuggestedAction struct {
	MessageID              *uint64 `json:"message_id,omitempty"`
	ActionType             string  `json:"action_type"`
	Title                  string  `json:"title"`
	Reason                 string  `json:"reason"`
	TargetKind             string  `json:"target_kind,omitempty"`
	TargetNamespace        string  `json:"target_namespace,omitempty"`
	TargetName             string  `json:"target_name,omitempty"`
	Replicas               *int    `json:"replicas,omitempty"`
	ManifestYAML           string  `json:"manifest_yaml,omitempty"`
	DefaultNamespace       string  `json:"default_namespace,omitempty"`
	RiskLevel              string  `json:"risk_level,omitempty"`
	RequiresConfirmation   bool    `json:"requires_confirmation"`
	AutoProposalEligible   bool    `json:"auto_proposal_eligible"`
	ProposalCreated        bool    `json:"proposal_created"`
	ProposalID             *uint64 `json:"proposal_id,omitempty"`
}

type aiActionsEnvelope struct {
	SuggestedActions []AISuggestedAction `json:"suggested_actions"`
}

func extractStructuredSuggestions(content string) (string, []AISuggestedAction) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", nil
	}

	idx := strings.LastIndex(trimmed, aiActionsJSONTag)
	if idx < 0 {
		return trimmed, nil
	}

	jsonText := strings.TrimSpace(trimmed[idx+len(aiActionsJSONTag):])
	cleanContent := strings.TrimSpace(trimmed[:idx])
	if jsonText == "" {
		return cleanContent, nil
	}

	var envelope aiActionsEnvelope
	if err := json.Unmarshal([]byte(jsonText), &envelope); err != nil {
		return trimmed, nil
	}

	return cleanContent, normalizeSuggestedActions(envelope.SuggestedActions)
}

func normalizeSuggestedActions(items []AISuggestedAction) []AISuggestedAction {
	result := make([]AISuggestedAction, 0, len(items))
	for _, item := range items {
		actionType := normalizeAIActionType(item.ActionType)
		if actionType == "" {
			continue
		}

		targetKind := normalizeAIActionTarget(AIActionTargetResource{
			Kind:      item.TargetKind,
			Namespace: item.TargetNamespace,
			Name:      item.TargetName,
		})

		normalized := AISuggestedAction{
			ActionType:           actionType,
			Title:                strings.TrimSpace(item.Title),
			Reason:               strings.TrimSpace(item.Reason),
			TargetKind:           targetKind.Kind,
			TargetNamespace:      targetKind.Namespace,
			TargetName:           targetKind.Name,
			ManifestYAML:         strings.TrimSpace(item.ManifestYAML),
			DefaultNamespace:     strings.TrimSpace(item.DefaultNamespace),
			RequiresConfirmation: true,
		}

		if strings.TrimSpace(normalized.Title) == "" {
			normalized.Title = defaultSuggestedActionTitle(normalized)
		}
		if strings.TrimSpace(normalized.Reason) == "" {
			normalized.Reason = "AI suggested this action based on the current conversation context."
		}
		normalized.RiskLevel = normalizeSuggestedActionRisk(item.RiskLevel, normalized.ActionType)

		switch normalized.ActionType {
		case aiActionTypeRestartWorkload:
			if _, ok := aiActionWorkloadGVR(normalized.TargetKind); !ok || normalized.TargetNamespace == "" || normalized.TargetName == "" {
				continue
			}
			normalized.AutoProposalEligible = true
		case aiActionTypeScaleWorkload:
			if _, ok := aiActionWorkloadGVR(normalized.TargetKind); !ok || normalized.TargetNamespace == "" || normalized.TargetName == "" {
				continue
			}
			if item.Replicas == nil || *item.Replicas < 0 || strings.EqualFold(normalized.TargetKind, "DaemonSet") {
				continue
			}
			replicas := *item.Replicas
			normalized.Replicas = &replicas
			normalized.AutoProposalEligible = true
		case aiActionTypeApplyManifest:
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

func defaultSuggestedActionTitle(item AISuggestedAction) string {
	switch item.ActionType {
	case aiActionTypeRestartWorkload:
		return "Suggested restart workload action"
	case aiActionTypeScaleWorkload:
		return "Suggested scale workload action"
	case aiActionTypeApplyManifest:
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
	case aiActionTypeRestartWorkload, aiActionTypeScaleWorkload:
		return "low"
	case aiActionTypeApplyManifest:
		return "medium"
	default:
		return "medium"
	}
}

func buildSuggestedActionsStructured(actions []AISuggestedAction) model.JSONMap {
	if len(actions) == 0 {
		return nil
	}

	payload := make([]map[string]any, 0, len(actions))
	for _, action := range actions {
		item := map[string]any{
			"action_type":             action.ActionType,
			"title":                   action.Title,
			"reason":                  action.Reason,
			"target_kind":             action.TargetKind,
			"target_namespace":        action.TargetNamespace,
			"target_name":             action.TargetName,
			"risk_level":              action.RiskLevel,
			"requires_confirmation":   action.RequiresConfirmation,
			"auto_proposal_eligible":  action.AutoProposalEligible,
			"proposal_created":        action.ProposalCreated,
			"default_namespace":       action.DefaultNamespace,
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

	return model.JSONMap{
		"suggested_actions": payload,
	}
}
