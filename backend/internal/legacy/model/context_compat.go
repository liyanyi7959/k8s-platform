package model

import (
	aidomain "k8s-platform-backend/internal/ai/domain"
	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
)

// Context-owned persistence entities are temporarily aliased here only for
// legacy services that have not yet crossed their application-port boundary.
// No schema is duplicated in the legacy package.
type (
	AIActionExecution = aidomain.AIActionExecution
	AIActionProposal  = aidomain.AIActionProposal
	AIConversation    = aidomain.AIConversation
	AIMessage         = aidomain.AIMessage
	AIToolCall        = aidomain.AIToolCall
	AIUploadedFile    = aidomain.AIUploadedFile
	AIUsageRecord     = aidomain.AIUsageRecord

	JSONStringSlice        = provisiondomain.JSONStringSlice
	DeployServer           = provisiondomain.DeployServer
	SSHCredential          = provisiondomain.SSHCredential
	DeployPlan             = provisiondomain.DeployPlan
	DeployPlanNode         = provisiondomain.DeployPlanNode
	DeployPlanStepOverride = provisiondomain.DeployPlanStepOverride
	DeployLog              = provisiondomain.DeployLog
)
