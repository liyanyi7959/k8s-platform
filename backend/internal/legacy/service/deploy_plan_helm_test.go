package service

import (
	"testing"
	"time"

	"k8s-platform-backend/internal/legacy/model"
)

func TestDeployPlanToItemPreservesHelmInstall(t *testing.T) {
	row := model.DeployPlan{
		ID:          42,
		Name:        "platform-cluster",
		ClusterName: "platform-k8s",
		HelmInstall: true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	item := deployPlanToItem(row, nil)
	if !item.HelmInstall {
		t.Fatal("helm_install should be returned with deploy plan detail")
	}
}
