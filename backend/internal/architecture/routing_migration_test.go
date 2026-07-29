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
			file:      filepath.Join(root, "internal", "router", "routes_admin_ai.go"),
			forbidden: []string{"k8sCtl.PodLogWS", "k8sCtl.PodExecWS"},
		},
		{
			file: filepath.Join(root, "internal", "router", "routes_admin_ai.go"),
			forbidden: []string{
				"ctl.ListProviders", "ctl.CreateProvider", "ctl.PatchProvider", "ctl.DeleteProvider",
				"ctl.ListModels", "ctl.CreateModel", "ctl.PatchModel", "ctl.DeleteModel",
				"ctl.GetRouteSettings", "ctl.UpdateRouteSettings", "ctl.ListConversations", "ctl.DeleteConversation", "ctl.CreateConversation",
				"ctl.ListTools", "ctl.GetConversation", "ctl.DownloadAttachmentContent", "ctl.SendChat", "ctl.SendChatStream", "ctl.CreateActionProposal", "ctl.ConfirmActionProposal",
			},
		},
		{
			file: filepath.Join(root, "internal", "router", "routes_business.go"),
			forbidden: []string{
				"ctl.ListServers)", "ctl.CreateServer)", "ctl.UpdateServer)", "ctl.DeleteServer)",
				"ctl.ListCredentials", "ctl.CreateCredential", "ctl.UpdateCredential", "ctl.DeleteCredential",
				"ctl.ListPlans", "ctl.CreatePlan", "ctl.UpdatePlan", "ctl.DeletePlan",
				"ctl.ExecutePlan", "ctl.CancelPlan", "ctl.RetryPlan", "ctl.GetDeployTask", "ctl.GetDeployTaskLogs",
				"ctl.TestSSH", "ctl.CreateServerTerminalSession", "ctl.ServerTerminalWS", "*controller.AutomationTaskController",
			},
		},
		{
			file: filepath.Join(root, "internal", "router", "routes_kops.go"),
			forbidden: []string{
				"ctl.ListManifestRecords", "ctl.GetManifestRecord", "ctl.ApplyManifest",
				"ctl.ListNamespaces", "ctl.CreateNamespace", "ctl.DeleteNamespace", "ctl.GetNamespaceYAML",
				"ctl.GetNamespaceResourcesSummary", "ctl.GetNamespaceInspection", "ctl.GetNamespaceWorkloadInventory",
				"ctl.GetPodInspection",
				"ctl.ListNodes", "ctl.GetNodeDetail", "ctl.GetNodeYAML", "ctl.ListNodePods", "ctl.ListNodeEvents",
				"ctl.CordonNode", "ctl.UncordonNode", "ctl.DrainNode", "ctl.DeleteNode",
				"ctl.ListRuntimeClasses", "ctl.GetRuntimeClassYAML", "ctl.EditRuntimeClass", "ctl.DeleteRuntimeClass",
				"ctl.ListCSIDrivers", "ctl.GetCSIDriverYAML", "ctl.EditCSIDriver", "ctl.DeleteCSIDriver",
				"ctl.ListCSINodes", "ctl.GetCSINodeYAML", "ctl.EditCSINode", "ctl.DeleteCSINode",
				"ctl.ListCSIStorageCapacities", "ctl.GetCSIStorageCapacityYAML", "ctl.EditCSIStorageCapacity", "ctl.DeleteCSIStorageCapacity",
				"ctl.ListValidatingAdmissionPolicies", "ctl.GetValidatingAdmissionPolicyYAML", "ctl.EditValidatingAdmissionPolicy", "ctl.DeleteValidatingAdmissionPolicy",
				"ctl.ListValidatingAdmissionPolicyBindings", "ctl.GetValidatingAdmissionPolicyBindingYAML", "ctl.EditValidatingAdmissionPolicyBinding", "ctl.DeleteValidatingAdmissionPolicyBinding",
				"ctl.ListReplicaSets", "ctl.GetReplicaSetYAML", "ctl.EditReplicaSet", "ctl.DeleteReplicaSet",
				"ctl.ListVolumeAttachments", "ctl.GetVolumeAttachmentYAML", "ctl.EditVolumeAttachment", "ctl.DeleteVolumeAttachment",
				"ctl.ListPDBs", "ctl.GetPDBYAML", "ctl.EditPDB", "ctl.DeletePDB",
				"ctl.ListRoles", "ctl.GetRoleYAML", "ctl.EditRole", "ctl.DeleteRole",
				"ctl.ListClusterRoles", "ctl.GetClusterRoleYAML", "ctl.EditClusterRole", "ctl.DeleteClusterRole",
				"ctl.ListRoleBindings", "ctl.GetRoleBindingYAML", "ctl.EditRoleBinding", "ctl.DeleteRoleBinding",
				"ctl.ListClusterRoleBindings", "ctl.GetClusterRoleBindingYAML", "ctl.EditClusterRoleBinding", "ctl.DeleteClusterRoleBinding",
				"ctl.ListCustomResourceDefinitions", "ctl.GetCustomResourceDefinitionYAML", "ctl.EditCustomResourceDefinition", "ctl.DeleteCustomResourceDefinition",
				"ctl.ListAPIServices", "ctl.GetAPIServiceYAML", "ctl.EditAPIService", "ctl.DeleteAPIService",
				"ctl.ListPriorityClasses", "ctl.GetPriorityClassYAML", "ctl.EditPriorityClass", "ctl.DeletePriorityClass",
				"ctl.ListValidatingWebhookConfigurations", "ctl.GetValidatingWebhookConfigurationYAML", "ctl.EditValidatingWebhookConfiguration", "ctl.DeleteValidatingWebhookConfiguration",
				"ctl.ListMutatingWebhookConfigurations", "ctl.GetMutatingWebhookConfigurationYAML", "ctl.EditMutatingWebhookConfiguration", "ctl.DeleteMutatingWebhookConfiguration",
				"ctl.ListNetworkPolicies", "ctl.GetNetworkPolicyYAML", "ctl.EditNetworkPolicy", "ctl.DeleteNetworkPolicy",
				"ctl.ListResourceQuotas", "ctl.GetResourceQuotaYAML", "ctl.EditResourceQuota", "ctl.DeleteResourceQuota",
				"ctl.ListLimitRanges", "ctl.GetLimitRangeYAML", "ctl.EditLimitRange", "ctl.DeleteLimitRange",
				"ctl.ListJobs", "ctl.GetJobYAML", "ctl.DeleteJob", "ctl.EditJob", "ctl.DeleteCompletedJobs",
				"ctl.ListCronJobs", "ctl.GetCronJobYAML", "ctl.DeleteCronJob", "ctl.TriggerCronJob", "ctl.SuspendCronJob", "ctl.EditCronJob",
				"ctl.ListPods", "ctl.ListPodMetrics", "ctl.GetPodYAML", "ctl.GetPodLogs", "ctl.CreatePodLogSession", "ctl.DeletePod",
				"ctl.CreatePodExecSession",
				"ctl.ListHelmReleases", "ctl.GetHelmReleaseDetail", "ctl.HelmPreflight", "ctl.HelmInstall", "ctl.HelmUninstall", "ctl.HelmUpgrade", "ctl.HelmRollback", "ctl.HelmRepoList", "ctl.HelmRepoAdd", "ctl.HelmRepoDelete", "ctl.HelmSearch",
				"ctl.ListWorkloads", "ctl.GetRolloutHistory", "ctl.RolloutUndo", "ctl.ScaleWorkload", "ctl.RestartWorkload", "ctl.UpdateImage", "ctl.UpdateWorkloadPaused",
				"ctl.EditDeployment", "ctl.EditStatefulSet", "ctl.EditDaemonSet", "ctl.EditWorkloadYAML", "ctl.DeleteWorkload", "ctl.GetWorkloadYAML",
				"ctl.ListEvents", "ctl.ListServiceAccounts", "ctl.GetServiceAccountYAML", "ctl.EditServiceAccount", "ctl.DeleteServiceAccount",
				"ctl.ListHPAs", "ctl.GetHPAYAML", "ctl.EditHPA", "ctl.DeleteHPA",
				"ctl.ListNodeMetrics", "ctl.ListPodMetricsUsage", "ctl.GetMetricsSource", "ctl.DetectMetricsSource",
				"ctl.SwitchMetricsSource", "ctl.GetMetricsTrend", "ctl.HealthCheckMetricsSource",
				"ctl.ListEndpoints", "ctl.GetEndpointsYAML", "ctl.EditEndpoints", "ctl.DeleteEndpoints",
				"ctl.ListEndpointSlices", "ctl.GetEndpointSliceYAML", "ctl.EditEndpointSlice", "ctl.DeleteEndpointSlice",
				"ctl.ListServices", "ctl.GetServiceYAML", "ctl.EditService", "ctl.DeleteService",
				"ctl.ListIngresses", "ctl.GetIngressYAML", "ctl.EditIngress", "ctl.DeleteIngress",
				"ctl.ListIngressClasses", "ctl.GetIngressClassYAML", "ctl.EditIngressClass", "ctl.DeleteIngressClass",
				"ctl.ListConfigMaps", "ctl.GetConfigMapYAML", "ctl.EditConfigMap", "ctl.DeleteConfigMap", "ctl.GetConfigMapRelated",
				"ctl.ListSecrets", "ctl.GetSecretYAML", "ctl.EditSecret", "ctl.DeleteSecret", "ctl.GetSecretReveal", "ctl.GetSecretRelated",
				"ctl.ListPVCs", "ctl.CreatePVC", "ctl.GetPVCYAML", "ctl.DeletePVC", "ctl.ListPVs", "ctl.GetPVYAML", "ctl.DeletePV",
				"ctl.ListStorageClasses", "ctl.GetStorageClassYAML", "ctl.EditStorageClass", "ctl.DeleteStorageClass",
				"ctl.ListVolumeSnapshots", "ctl.GetVolumeSnapshotYAML", "ctl.EditVolumeSnapshot", "ctl.DeleteVolumeSnapshot",
				"ctl.ListVolumeSnapshotClasses", "ctl.GetVolumeSnapshotClassYAML", "ctl.EditVolumeSnapshotClass", "ctl.DeleteVolumeSnapshotClass",
				"ctl.ListVolumeSnapshotContents", "ctl.GetVolumeSnapshotContentYAML", "ctl.EditVolumeSnapshotContent", "ctl.DeleteVolumeSnapshotContent",
				"ctl.GetResourceSupport", "ctl.GetStorageSnapshotSupport",
				"ctl.ListLeases", "ctl.GetLeaseYAML", "ctl.EditLease", "ctl.DeleteLease",
				"ctl.CreateDeployment", "ctl.CreateStatefulSet", "ctl.CreateDaemonSet", "ctl.CreateService", "ctl.CreateIngress",
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
