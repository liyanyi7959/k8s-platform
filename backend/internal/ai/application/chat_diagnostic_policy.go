package application

import "strings"

// BuildChatDiagnosticEvidenceSummary produces the concise, verified evidence
// passed to an AI gateway after diagnostic tools complete. Failed tool calls
// are deliberately excluded because they are not platform evidence.
func BuildChatDiagnosticEvidenceSummary(toolCalls []ToolCallItem) string {
	summaryLines := make([]string, 0, len(toolCalls))
	for _, item := range toolCalls {
		if !strings.EqualFold(strings.TrimSpace(item.Status), "succeeded") {
			continue
		}
		toolName := strings.TrimSpace(item.ToolName)
		resultSummary := strings.TrimSpace(item.ResultSummary)
		if toolName == "" && resultSummary == "" {
			continue
		}
		if resultSummary == "" {
			summaryLines = append(summaryLines, "- "+toolName)
			continue
		}
		if toolName == "" {
			summaryLines = append(summaryLines, "- "+resultSummary)
			continue
		}
		summaryLines = append(summaryLines, "- "+toolName+": "+resultSummary)
	}

	if len(summaryLines) == 0 {
		return ""
	}
	return "Confirmed platform evidence for this round:\n" + strings.Join(summaryLines, "\n")
}

// BuildChatDiagnosticEvidenceDigest keeps the compact verified summary ahead
// of the detailed raw diagnostic notes supplied to the gateway.
func BuildChatDiagnosticEvidenceDigest(toolCalls []ToolCallItem, diagnosticNotes string) string {
	digest := BuildChatDiagnosticEvidenceSummary(toolCalls)
	notes := strings.TrimSpace(diagnosticNotes)
	if digest == "" {
		return notes
	}
	if notes == "" {
		return digest
	}
	return digest + "\n\nDetailed platform evidence:\n" + notes
}

// EffectiveChatGatewayMode promotes chat requests to diagnose mode whenever
// this round carries platform evidence. That prevents evidence-aware replies
// from being handled as generic chat.
func EffectiveChatGatewayMode(mode string, toolCalls []ToolCallItem, diagnosticNotes string) string {
	normalizedMode := NormalizeAssistantMode(mode)
	if normalizedMode == "" {
		normalizedMode = "diagnose"
	}
	if normalizedMode == "chat" && (len(toolCalls) > 0 || strings.TrimSpace(diagnosticNotes) != "") {
		return "diagnose"
	}
	return normalizedMode
}
