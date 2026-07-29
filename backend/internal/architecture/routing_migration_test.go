package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigratedRoutesDoNotReturnToLegacyControllers(t *testing.T) {
	root := backendRoot(t)
	cases := []struct {
		file      string
		forbidden []string
	}{
		{
			file: filepath.Join(root, "internal", "router", "routes_admin_ai.go"),
			forbidden: []string{
				"ctl.ListProviders", "ctl.CreateProvider", "ctl.PatchProvider", "ctl.DeleteProvider",
				"ctl.ListModels", "ctl.CreateModel", "ctl.PatchModel", "ctl.DeleteModel",
				"ctl.GetRouteSettings", "ctl.UpdateRouteSettings", "ctl.ListConversations", "ctl.DeleteConversation", "ctl.CreateConversation",
			},
		},
		{
			file: filepath.Join(root, "internal", "router", "routes_business.go"),
			forbidden: []string{
				"ctl.ListServers)", "ctl.CreateServer)", "ctl.UpdateServer)", "ctl.DeleteServer)",
				"ctl.ListCredentials", "ctl.CreateCredential", "ctl.UpdateCredential", "ctl.DeleteCredential",
				"ctl.ListPlans", "ctl.CreatePlan", "ctl.UpdatePlan", "ctl.DeletePlan",
				"ctl.ExecutePlan", "ctl.CancelPlan", "ctl.RetryPlan", "ctl.GetDeployTask", "ctl.GetDeployTaskLogs",
			},
		},
		{
			file: filepath.Join(root, "internal", "router", "routes_kops.go"),
			forbidden: []string{
				"ctl.ListManifestRecords", "ctl.GetManifestRecord", "ctl.ApplyManifest",
				"ctl.ListNamespaces", "ctl.CreateNamespace", "ctl.DeleteNamespace", "ctl.GetNamespaceYAML",
				"ctl.GetNamespaceResourcesSummary", "ctl.GetNamespaceInspection", "ctl.GetNamespaceWorkloadInventory",
			},
		},
		{
			file: filepath.Join(root, "internal", "router", "routes_business.go"),
			forbidden: []string{
				"*controller.K8sPermissionAuditController",
			},
		},
	}
	for _, testCase := range cases {
		content, err := os.ReadFile(testCase.file)
		if err != nil {
			t.Fatalf("read %s: %v", testCase.file, err)
		}
		for _, forbidden := range testCase.forbidden {
			if strings.Contains(string(content), forbidden) {
				t.Errorf("%s rebinds migrated route to legacy handler %q", testCase.file, forbidden)
			}
		}
	}
}
