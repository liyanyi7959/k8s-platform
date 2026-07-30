package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/auth"
	fleethttp "k8s-platform-backend/internal/fleet/adapters/http"
	iamhttp "k8s-platform-backend/internal/iam/adapters/http"
	incidenthttp "k8s-platform-backend/internal/incident/adapters/http"
	kopshttp "k8s-platform-backend/internal/kops/adapters/http"
	"k8s-platform-backend/internal/middleware"
	orchestrationprovision "k8s-platform-backend/internal/orchestration/provisioning"
	provisionhttp "k8s-platform-backend/internal/provisioning/adapters/http"
)

func TestRegisterK8sRoutes_RejectsInsufficientPerms(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		method   string
		path     string
		perms    []string
		wantCode int
	}{
		// ── Secret / Exec（已有） ──
		{
			name:     "secret reveal does not reuse read perm",
			method:   http.MethodGet,
			path:     "/api/v2/clusters/1/secrets/default/demo/decoded-data",
			perms:    []string{"k8s:read"},
			wantCode: 1003,
		},
		{
			name:     "pod exec session does not reuse write perm",
			method:   http.MethodPost,
			path:     "/api/v2/clusters/1/pods/default/demo/exec-sessions",
			perms:    []string{"k8s:write"},
			wantCode: 1003,
		},
		{
			name:     "node delete still requires write perm",
			method:   http.MethodDelete,
			path:     "/api/v2/clusters/1/nodes/node-a",
			perms:    []string{"k8s:read"},
			wantCode: 1003,
		},
		{
			name:     "rbac delete does not accept generic k8s write",
			method:   http.MethodDelete,
			path:     "/api/v2/clusters/1/roles/default/demo",
			perms:    []string{"k8s:write"},
			wantCode: 1003,
		},
		// ── RBAC：rbac_read / rbac_write 权限隔离 ──
		{
			name:     "clusterrole list requires rbac_read not k8s:read",
			method:   http.MethodGet,
			path:     "/api/v2/clusters/1/clusterroles",
			perms:    []string{"k8s:read"},
			wantCode: 1003,
		},
		{
			name:     "rolebinding edit requires rbac_write not k8s:write",
			method:   http.MethodPatch,
			path:     "/api/v2/clusters/1/rolebindings/default/demo",
			perms:    []string{"k8s:write"},
			wantCode: 1003,
		},
		{
			name:     "clusterrolebinding delete requires rbac_write",
			method:   http.MethodDelete,
			path:     "/api/v2/clusters/1/clusterrolebindings/admin",
			perms:    []string{"k8s:write"},
			wantCode: 1003,
		},
		// ── PDB ──
		{
			name:     "pdb list requires k8s:read",
			method:   http.MethodGet,
			path:     "/api/v2/clusters/1/pdbs",
			perms:    []string{},
			wantCode: 1003,
		},
		{
			name:     "pdb delete requires k8s:write",
			method:   http.MethodDelete,
			path:     "/api/v2/clusters/1/pdbs/default/my-pdb",
			perms:    []string{"k8s:read"},
			wantCode: 1003,
		},
		// ── Lease ──
		{
			name:     "lease list requires k8s:read",
			method:   http.MethodGet,
			path:     "/api/v2/clusters/1/leases",
			perms:    []string{},
			wantCode: 1003,
		},
		// ── CRD ──
		{
			name:     "crd delete requires k8s:write",
			method:   http.MethodDelete,
			path:     "/api/v2/clusters/1/customresourcedefinitions/my-crd",
			perms:    []string{"k8s:read"},
			wantCode: 1003,
		},
		// ── Webhook ──
		{
			name:     "validating webhook list requires k8s:read",
			method:   http.MethodGet,
			path:     "/api/v2/clusters/1/validatingwebhookconfigurations",
			perms:    []string{},
			wantCode: 1003,
		},
		{
			name:     "mutating webhook delete requires k8s:write",
			method:   http.MethodDelete,
			path:     "/api/v2/clusters/1/mutatingwebhookconfigurations/my-wh",
			perms:    []string{"k8s:read"},
			wantCode: 1003,
		},
		// ── EndpointSlice ──
		{
			name:     "endpointslice list requires k8s:read",
			method:   http.MethodGet,
			path:     "/api/v2/clusters/1/endpointslices",
			perms:    []string{},
			wantCode: 1003,
		},
		{
			name:     "endpointslice delete requires k8s:write",
			method:   http.MethodDelete,
			path:     "/api/v2/clusters/1/endpointslices/default/my-eps",
			perms:    []string{"k8s:read"},
			wantCode: 1003,
		},
		// ── Namespace（namespace:read / namespace:write 权限隔离） ──
		{
			name:     "namespace create accepts namespace:write",
			method:   http.MethodPost,
			path:     "/api/v2/clusters/1/namespaces",
			perms:    []string{"namespace:write"},
			wantCode: 4000, // 有权限但缺少 body → 参数错误
		},
		{
			name:     "namespace delete rejects k8s:read",
			method:   http.MethodDelete,
			path:     "/api/v2/clusters/1/namespaces/test-ns",
			perms:    []string{"k8s:read"},
			wantCode: 1003,
		},
		// ── ValidatingAdmissionPolicy ──
		{
			name:     "validatingadmissionpolicy list requires k8s:read",
			method:   http.MethodGet,
			path:     "/api/v2/clusters/1/validatingadmissionpolicies",
			perms:    []string{},
			wantCode: 1003,
		},
		// ── PriorityClass ──
		{
			name:     "priorityclass delete requires k8s:write",
			method:   http.MethodDelete,
			path:     "/api/v2/clusters/1/priorityclasses/my-pc",
			perms:    []string{"k8s:read"},
			wantCode: 1003,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := performPermissionRequest(t, tt.method, tt.path, tt.perms, func(group *gin.RouterGroup) {
				registerK8sRoutes(group, Deps{}, &kopshttp.ManifestController{}, &kopshttp.NamespaceController{}, &kopshttp.MetricsController{}, &kopshttp.ConnectivityController{}, &kopshttp.NodeController{}, &kopshttp.PlatformResourceController{}, &kopshttp.RelationshipResourceController{}, &kopshttp.BatchController{}, &kopshttp.NetworkController{}, &kopshttp.ConfigurationController{}, &kopshttp.StorageController{}, &kopshttp.HelmController{}, &kopshttp.WorkloadController{}, &kopshttp.PodController{}, &kopshttp.InspectionController{})
			})
			assertPermissionCode(t, resp, tt.wantCode)
		})
	}
}

func TestCanonicalCompatibilityRoutesAreRegisteredAndProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		method   string
		path     string
		register func(*gin.RouterGroup)
	}{
		{
			name:   "cluster health check",
			method: http.MethodPost,
			path:   "/api/v2/clusters/1/health-checks",
			register: func(group *gin.RouterGroup) {
				registerClusterRoutes(group, Deps{}, &fleethttp.ClusterController{})
			},
		},
		{
			name:   "server connection check",
			method: http.MethodPost,
			path:   "/api/v2/provisioning/servers/1/connection-checks",
			register: func(group *gin.RouterGroup) {
				registerDeployRoutes(group, &orchestrationprovision.ServerAccessController{}, &provisionhttp.ServerController{}, &provisionhttp.CredentialController{}, &provisionhttp.DeployPlanController{}, &provisionhttp.RuntimeController{}, &provisionhttp.TaskController{}, nil)
			},
		},
		{
			name:   "alert state patch",
			method: http.MethodPatch,
			path:   "/api/v2/monitoring/alert-rules/1",
			register: func(group *gin.RouterGroup) {
				registerMonitorIncidentRoutes(group, &incidenthttp.LegacyController{})
			},
		},
		{
			name:   "canonical configmap update",
			method: http.MethodPatch,
			path:   "/api/v2/clusters/1/configmaps/default/demo",
			register: func(group *gin.RouterGroup) {
				registerK8sRoutes(group, Deps{}, &kopshttp.ManifestController{}, &kopshttp.NamespaceController{}, &kopshttp.MetricsController{}, &kopshttp.ConnectivityController{}, &kopshttp.NodeController{}, &kopshttp.PlatformResourceController{}, &kopshttp.RelationshipResourceController{}, &kopshttp.BatchController{}, &kopshttp.NetworkController{}, &kopshttp.ConfigurationController{}, &kopshttp.StorageController{}, &kopshttp.HelmController{}, &kopshttp.WorkloadController{}, &kopshttp.PodController{}, &kopshttp.InspectionController{})
			},
		},
		{
			name:   "canonical helm detail",
			method: http.MethodGet,
			path:   "/api/v2/clusters/1/helm/releases/default/demo",
			register: func(group *gin.RouterGroup) {
				registerK8sRoutes(group, Deps{}, &kopshttp.ManifestController{}, &kopshttp.NamespaceController{}, &kopshttp.MetricsController{}, &kopshttp.ConnectivityController{}, &kopshttp.NodeController{}, &kopshttp.PlatformResourceController{}, &kopshttp.RelationshipResourceController{}, &kopshttp.BatchController{}, &kopshttp.NetworkController{}, &kopshttp.ConfigurationController{}, &kopshttp.StorageController{}, &kopshttp.HelmController{}, &kopshttp.WorkloadController{}, &kopshttp.PodController{}, &kopshttp.InspectionController{})
			},
		},
		{
			name:   "automation cancellation request",
			method: http.MethodPost,
			path:   "/api/v2/automation/tasks/1/cancellation-requests",
			register: func(group *gin.RouterGroup) {
				registerAutomationTaskRoutes(group, &provisionhttp.AutomationTaskController{})
			},
		},
		{
			name:   "deployment preflight check",
			method: http.MethodPost,
			path:   "/api/v2/provisioning/plans/1/preflight-runs",
			register: func(group *gin.RouterGroup) {
				registerDeployRoutes(group, &orchestrationprovision.ServerAccessController{}, &provisionhttp.ServerController{}, &provisionhttp.CredentialController{}, &provisionhttp.DeployPlanController{}, &provisionhttp.RuntimeController{}, &provisionhttp.TaskController{}, nil)
			},
		},
		{
			name:   "permission audit cancellation request",
			method: http.MethodPost,
			path:   "/api/v2/permission-audits/1/cancellation-requests",
			register: func(group *gin.RouterGroup) {
				registerPermissionAuditRoutes(group, &kopshttp.PermissionAuditController{}, nil)
			},
		},
		{
			name:   "node drain request",
			method: http.MethodPost,
			path:   "/api/v2/clusters/1/nodes/node-a/drain-requests",
			register: func(group *gin.RouterGroup) {
				registerK8sRoutes(group, Deps{}, &kopshttp.ManifestController{}, &kopshttp.NamespaceController{}, &kopshttp.MetricsController{}, &kopshttp.ConnectivityController{}, &kopshttp.NodeController{}, &kopshttp.PlatformResourceController{}, &kopshttp.RelationshipResourceController{}, &kopshttp.BatchController{}, &kopshttp.NetworkController{}, &kopshttp.ConfigurationController{}, &kopshttp.StorageController{}, &kopshttp.HelmController{}, &kopshttp.WorkloadController{}, &kopshttp.PodController{}, &kopshttp.InspectionController{})
			},
		},
		{
			name:   "manifest application",
			method: http.MethodPost,
			path:   "/api/v2/clusters/1/manifest-applications",
			register: func(group *gin.RouterGroup) {
				registerK8sRoutes(group, Deps{}, &kopshttp.ManifestController{}, &kopshttp.NamespaceController{}, &kopshttp.MetricsController{}, &kopshttp.ConnectivityController{}, &kopshttp.NodeController{}, &kopshttp.PlatformResourceController{}, &kopshttp.RelationshipResourceController{}, &kopshttp.BatchController{}, &kopshttp.NetworkController{}, &kopshttp.ConfigurationController{}, &kopshttp.StorageController{}, &kopshttp.HelmController{}, &kopshttp.WorkloadController{}, &kopshttp.PodController{}, &kopshttp.InspectionController{})
			},
		},
		{
			name:   "pod exec session",
			method: http.MethodPost,
			path:   "/api/v2/clusters/1/pods/default/demo/exec-sessions",
			register: func(group *gin.RouterGroup) {
				registerK8sRoutes(group, Deps{}, &kopshttp.ManifestController{}, &kopshttp.NamespaceController{}, &kopshttp.MetricsController{}, &kopshttp.ConnectivityController{}, &kopshttp.NodeController{}, &kopshttp.PlatformResourceController{}, &kopshttp.RelationshipResourceController{}, &kopshttp.BatchController{}, &kopshttp.NetworkController{}, &kopshttp.ConfigurationController{}, &kopshttp.StorageController{}, &kopshttp.HelmController{}, &kopshttp.WorkloadController{}, &kopshttp.PodController{}, &kopshttp.InspectionController{})
			},
		},
		{
			name:   "helm rollback attempt",
			method: http.MethodPost,
			path:   "/api/v2/clusters/1/helm/releases/default/demo/rollback-attempts",
			register: func(group *gin.RouterGroup) {
				registerK8sRoutes(group, Deps{}, &kopshttp.ManifestController{}, &kopshttp.NamespaceController{}, &kopshttp.MetricsController{}, &kopshttp.ConnectivityController{}, &kopshttp.NodeController{}, &kopshttp.PlatformResourceController{}, &kopshttp.RelationshipResourceController{}, &kopshttp.BatchController{}, &kopshttp.NetworkController{}, &kopshttp.ConfigurationController{}, &kopshttp.StorageController{}, &kopshttp.HelmController{}, &kopshttp.WorkloadController{}, &kopshttp.PodController{}, &kopshttp.InspectionController{})
			},
		},
		{
			name:   "user password reset request",
			method: http.MethodPost,
			path:   "/api/v2/users/1/password-reset-requests",
			register: func(group *gin.RouterGroup) {
				registerUserRoutes(group, &iamhttp.Controller{})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performPermissionRequest(t, test.method, test.path, nil, test.register)
			assertPermissionCode(t, response, 1003)
		})
	}
}

func performPermissionRequest(t *testing.T, method string, path string, perms []string, register func(group *gin.RouterGroup)) *httptest.ResponseRecorder {
	t.Helper()

	r := gin.New()
	authed := r.Group("/api/v2")
	authed.Use(middleware.V2Contract())
	authed.Use(func(c *gin.Context) {
		c.Set("auth_claims", &auth.Claims{UserID: 1, Username: "tester", Perms: perms})
		c.Next()
	})
	register(authed)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	return w
}

func assertPermissionCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode int) {
	t.Helper()

	wantStatus := http.StatusForbidden
	if wantCode == 4000 {
		wantStatus = http.StatusBadRequest
	} else if wantCode != 1003 {
		t.Fatalf("unsupported expected legacy code %d", wantCode)
	}
	if recorder.Code != wantStatus {
		t.Fatalf("unexpected HTTP status: got %d want %d, body=%s", recorder.Code, wantStatus, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "application/problem+json") {
		t.Fatalf("expected Problem Details response, got Content-Type %q", contentType)
	}
}
