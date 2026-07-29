package domain

type AISuggestedAction struct {
	MessageID            *uint64 `json:"message_id,omitempty"`
	ActionType           string  `json:"action_type"`
	Title                string  `json:"title"`
	Reason               string  `json:"reason"`
	TargetKind           string  `json:"target_kind,omitempty"`
	TargetNamespace      string  `json:"target_namespace,omitempty"`
	TargetName           string  `json:"target_name,omitempty"`
	Replicas             *int    `json:"replicas,omitempty"`
	ManifestYAML         string  `json:"manifest_yaml,omitempty"`
	DefaultNamespace     string  `json:"default_namespace,omitempty"`
	RiskLevel            string  `json:"risk_level,omitempty"`
	RequiresConfirmation bool    `json:"requires_confirmation"`
	AutoProposalEligible bool    `json:"auto_proposal_eligible"`
	ProposalCreated      bool    `json:"proposal_created"`
	ProposalID           *uint64 `json:"proposal_id,omitempty"`
}
