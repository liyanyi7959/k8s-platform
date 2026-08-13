package domain

import "testing"

func TestDeployPlanPersistenceEntities(t *testing.T) {
	if (DeployServer{}).TableName() != "deploy_servers" || (Credential{}).TableName() != "credentials" || (DeployPlan{}).TableName() != "deploy_plans" || (DeployPlanNode{}).TableName() != "deploy_plan_nodes" || (DeployLog{}).TableName() != "deploy_logs" {
		t.Fatal("deployment persistence table mapping changed")
	}
	value, err := JSONStringSlice{"cilium", "metrics-server"}.Value()
	if err != nil || string(value.([]byte)) != `["cilium","metrics-server"]` {
		t.Fatalf("JSON string value = %v, %v", value, err)
	}
}
