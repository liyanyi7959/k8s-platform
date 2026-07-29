package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
)

// DryRunPlan loads the plan's runtime data, then delegates deterministic
// preview construction to the Provisioning application layer.
func (s *DeployService) DryRunPlan(ctx context.Context, planID uint64) (provisionapp.DryRunResult, error) {
	if planID == 0 {
		return provisionapp.DryRunResult{}, ErrInvalidParams
	}
	plan, nodes, err := s.getPlanWithNodes(ctx, planID)
	if err != nil {
		return provisionapp.DryRunResult{}, err
	}

	serverIDs := make([]uint64, 0, len(nodes))
	for _, node := range nodes {
		serverIDs = append(serverIDs, node.ServerID)
	}
	var servers []model.DeployServer
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id IN ?", serverIDs).Find(&servers).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return provisionapp.DryRunResult{}, err
	}
	serverByID := make(map[uint64]model.DeployServer, len(servers))
	for _, server := range servers {
		serverByID[server.ID] = server
	}

	input := provisionapp.DryRunPlanInput{
		PlanID: plan.ID, PlanName: plan.Name, ClusterName: plan.ClusterName, K8sVersion: plan.K8sVersion, CNIType: plan.CNIType,
		Nodes: make([]provisionapp.DryRunNodeInput, 0, len(nodes)),
	}
	for _, node := range nodes {
		server := serverByID[node.ServerID]
		input.Nodes = append(input.Nodes, provisionapp.DryRunNodeInput{
			ServerID: node.ServerID, ServerName: server.Name, IP: server.IP, Role: node.Role, SortOrder: node.SortOrder,
		})
	}
	result, err := provisionapp.BuildDryRun(input)
	if err != nil {
		return provisionapp.DryRunResult{}, legacyDryRunError(err)
	}
	return result, nil
}

func legacyDryRunError(err error) error {
	if !errors.Is(err, provisionapp.ErrInvalidParams) {
		return err
	}
	if message, ok := UserMessage(err); ok {
		return ErrWithMessage(ErrInvalidParams, message)
	}
	return ErrInvalidParams
}
