package router

import (
	aihttp "k8s-platform-backend/internal/ai/adapters/http"
	audithttp "k8s-platform-backend/internal/audit/adapters/http"
	auditapp "k8s-platform-backend/internal/audit/application"
	changeapp "k8s-platform-backend/internal/change/application"
	cicdhttp "k8s-platform-backend/internal/cicd/adapters/http"
	fleethttp "k8s-platform-backend/internal/fleet/adapters/http"
	iamhttp "k8s-platform-backend/internal/iam/adapters/http"
	incidenthttp "k8s-platform-backend/internal/incident/adapters/http"
	kopshttp "k8s-platform-backend/internal/kops/adapters/http"
	platformhttp "k8s-platform-backend/internal/platform/adapters/http"
	provisionhttp "k8s-platform-backend/internal/provisioning/adapters/http"
	workspacehttp "k8s-platform-backend/internal/workspace/adapters/http"
)

// applicationModules is the router's module directory. Each context is built
// in its own composition_*.go file so this type stays the only shared catalog.
type applicationModules struct {
	audit        auditModule
	iam          iamModule
	platform     platformModule
	workspace    workspaceModule
	fleet        fleetModule
	kops         kopsModule
	ai           aiModule
	change       changeModule
	provisioning provisioningModule
	cicd         cicdModule
	incident     incidentModule
}

type cicdModule struct {
	controller *cicdhttp.Controller
	logStream  *cicdhttp.CICDLogStreamController
}

type auditModule struct {
	service    *auditapp.Service
	controller *audithttp.Controller
}

type iamModule struct {
	users *iamhttp.Controller
}

type platformModule struct {
	settings *platformhttp.SettingsController
}

type workspaceModule struct {
	projects *workspacehttp.Controller
}

type fleetModule struct {
	clusters  *fleethttp.ClusterController
	dashboard *fleethttp.DashboardController
}

type kopsModule struct {
	permissionAudit *kopshttp.PermissionAuditController
	rbac            *kopshttp.RBACController
	manifests       *kopshttp.ManifestController
	namespaces      *kopshttp.NamespaceController
	metrics         *kopshttp.MetricsController
	connectivity    *kopshttp.ConnectivityController
	nodes           *kopshttp.NodeController
	platform        *kopshttp.PlatformResourceController
	relationships   *kopshttp.RelationshipResourceController
	batch           *kopshttp.BatchController
	network         *kopshttp.NetworkController
	configuration   *kopshttp.ConfigurationController
	storage         *kopshttp.StorageController
	workloads       *kopshttp.WorkloadController
	podLogStream    *kopshttp.PodLogStreamController
	podExecStream   *kopshttp.PodExecStreamController
	helm            *kopshttp.HelmController
	pods            *kopshttp.PodController
	inspection      *kopshttp.InspectionController
	creator         *kopshttp.ResourceCreatorController
}

type aiModule struct {
	runtime    *aihttp.RuntimeController
	management *aihttp.ManagementController
}

type changeModule struct {
	application *changeapp.Service
}

type provisioningModule struct {
	serverAccess *provisionhttp.ServerAccessController
	servers      *provisionhttp.ServerController
	credentials  *provisionhttp.CredentialController
	plans        *provisionhttp.DeployPlanController
	runtime      *provisionhttp.RuntimeController
	tasks        *provisionhttp.TaskController
	config       *provisionhttp.DeployConfigController
	automation   *provisionhttp.AutomationTaskController
	appTemplate  *provisionhttp.AppTemplateController
}

type incidentModule struct {
	legacy *incidenthttp.LegacyController
	v2     *incidenthttp.Controller
}

func buildApplicationModules(d Deps) applicationModules {
	if d.DB == nil {
		return applicationModules{}
	}

	runtime := buildModuleRuntime(d)
	modules := applicationModules{}
	modules.audit = buildAuditModule(d)
	modules.iam = buildIAMModule(d)
	modules.platform = buildPlatformModule(d)
	modules.workspace = buildWorkspaceModule(d, runtime)
	modules.fleet = buildFleetModule(d, runtime)
	modules.kops = buildKopsModule(d, runtime)
	modules.change = buildChangeModule(d, runtime)
	modules.ai = buildAIModule(d, runtime, modules.change)
	modules.provisioning = buildProvisioningModule(d, runtime)
	modules.cicd = buildCICDModule(d, runtime)
	modules.incident = buildIncidentModule(d)
	return modules
}
