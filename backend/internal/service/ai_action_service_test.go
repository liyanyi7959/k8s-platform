package service

import (
	"testing"

	"k8s-platform-backend/internal/model"
)

func TestValidateAIActionConfirmation(t *testing.T) {
	proposal := model.AIActionProposal{ID: 42}

	tests := []struct {
		name    string
		req     ConfirmAIActionProposalRequest
		wantErr bool
	}{
		{
			name: "missing risk acknowledgement",
			req: ConfirmAIActionProposalRequest{
				ConfirmationText: buildAIActionConfirmationText(proposal.ID),
				ConfirmRisk:      false,
			},
			wantErr: true,
		},
		{
			name: "missing confirmation text",
			req: ConfirmAIActionProposalRequest{
				ConfirmationText: "",
				ConfirmRisk:      true,
			},
			wantErr: true,
		},
		{
			name: "wrong confirmation text",
			req: ConfirmAIActionProposalRequest{
				ConfirmationText: "confirm-proposal-99",
				ConfirmRisk:      true,
			},
			wantErr: true,
		},
		{
			name: "valid confirmation",
			req: ConfirmAIActionProposalRequest{
				ConfirmationText: buildAIActionConfirmationText(proposal.ID),
				ConfirmRisk:      true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAIActionConfirmation(proposal, tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateAIActionConfirmation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
