// router 负责注册 HTTP 路由，并把 controller 绑定到 gin.Engine。
//
// 路由组织约定：
// - 所有 API 以 `/api/v1` 为前缀
// - 统一响应结构由 pkg/resp 负责封装（无论成功/失败通常都返回 HTTP 200，错误由 code/message 表达）
// - 认证/鉴权通过中间件完成：
//   - RequestID：为每个请求注入 request id，便于排障追踪
//   - AuthRequiredWithRBAC：解析 JWT，并加载/校验权限点
//   - RequirePerm：对具体路由进行权限点校验
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/controller"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/service"
)

// New 组装路由。接收 Deps 依赖容器，消除长参数列表。
func New(d Deps) (*gin.Engine, error) {
	r := gin.New()

	// ── 全局中间件 ──
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLogger())
	r.Use(middleware.RecoveryWithZap())
	r.Use(middleware.CORS())

	// ── 基础依赖（供当前仍在使用的 K8S 能力复用） ──
	taskStore := service.NewTaskStore(d.DB)

	// ── DB 依赖模块 ──
	var clusterManageCtl *controller.ClusterManageController
	var k8sCtl *controller.K8sController
	var dashboardCtl *controller.DashboardController
	var permissionAuditCtl *controller.K8sPermissionAuditController

	if d.DB != nil {
		clusterReg := service.NewClusterRegistryService(d.DB, d.EncryptionKey)
		k8sSvc := service.NewK8sService(clusterReg, d.CacheStore, d.CacheTTL, d.K8sInsecureTLS)
		clusterManageCtl = controller.NewClusterManageController(clusterReg, k8sSvc)
		execSessions := service.NewExecSessionStore(0)
		logSessions := service.NewPodLogSessionStore(0)
		k8sCtl = controller.NewK8sController(k8sSvc, execSessions, logSessions)
		dashboardSvc := service.NewDashboardService(d.DB, clusterReg, k8sSvc, d.CacheStore)
		dashboardCtl = controller.NewDashboardController(dashboardSvc)
		permissionAuditSvc := service.NewK8sPermissionAuditService(d.DB, taskStore, clusterReg, k8sSvc, d.CacheStore, d.EncryptionKey)
		permissionAuditCtl = controller.NewK8sPermissionAuditController(permissionAuditSvc)
	}

	// ── 健康检查 ──
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ── 路由注册 ──
	registerRoutes(r, d, clusterManageCtl, k8sCtl, dashboardCtl, permissionAuditCtl)

	return r, nil
}

//nolint:funlen // 路由注册表天然是长函数，按业务域分段组织
func registerRoutes(
	r *gin.Engine, d Deps,
	clusterManageCtl *controller.ClusterManageController,
	k8sCtl *controller.K8sController,
	dashboardCtl *controller.DashboardController,
	permissionAuditCtl *controller.K8sPermissionAuditController,
) {
	api := r.Group("/api/v1")

	// ── 公开接口（无需认证） ──
	api.POST("/auth/login", d.AuthCtl.Login)
	api.POST("/auth/logout", d.AuthCtl.Logout)
	api.GET("/auth/me", d.AuthCtl.Me)

	// ── 需认证接口 ──
	authed := api.Group("")
	authed.Use(middleware.AuthRequiredWithRBAC(d.JWTMgr, d.RbacSvc))

	authed.POST("/auth/change-password", d.AuthCtl.ChangePassword)

	registerClusterRoutes(authed, d, clusterManageCtl)
	registerDashboardRoutes(authed, d, dashboardCtl)
	registerPermissionAuditRoutes(authed, permissionAuditCtl)
	registerK8sRoutes(authed, d, k8sCtl)
	registerWebSocketRoutes(authed, k8sCtl)
}

func registerPermissionAuditRoutes(authed *gin.RouterGroup, ctl *controller.K8sPermissionAuditController) {
	if ctl == nil {
		return
	}
	auditPerm := middleware.RequirePerm("k8s:permission_audit")
	clusters := authed.Group("/clusters")
	clusters.POST("/:id/permission-audits", auditPerm, ctl.CreateManaged)
	clusters.GET("/:id/permission-audits/latest", auditPerm, ctl.LatestForCluster)
	clusters.GET("/:id/permission-audits/recommend-rbac", auditPerm, ctl.RecommendRBAC)
	clusters.GET("/:id/permission-audits/rbac-matrix/default", auditPerm, ctl.DefaultRBACMatrix)
	clusters.POST("/:id/permission-audits/rbac-matrix/yaml", auditPerm, ctl.RBACFromMatrix)

	audits := authed.Group("/permission-audits")
	audits.GET("", auditPerm, ctl.List)
	audits.GET("/:id", auditPerm, ctl.Get)
	audits.GET("/:id/logs", auditPerm, ctl.Logs)
	audits.GET("/:id/compare", auditPerm, ctl.Compare)
	audits.GET("/:id/findings", auditPerm, ctl.ListFindings)
	audits.POST("/:id/cancel", auditPerm, ctl.Cancel)
	audits.POST("/adhoc", auditPerm, ctl.CreateAdhoc)
}

// ── 集群管理 ──

func registerClusterRoutes(authed *gin.RouterGroup, d Deps, ctl *controller.ClusterManageController) {
	if ctl == nil {
		return
	}
	clusters := authed.Group("/clusters")
	clusters.Use(middleware.RequirePerm("cluster:read"))
	clusters.GET("", ctl.List)
	clusters.GET("/:id", ctl.Get)
	clusters.POST("/:id/check-health", ctl.CheckHealth)
	clusters.PATCH("/:id", middleware.RequirePerm("cluster:create"), ctl.Patch)
	clusters.DELETE("/:id", middleware.RequirePerm("cluster:create"), ctl.Delete)

	authed.POST("/clusters/import", middleware.RequirePerm("cluster:create"), ctl.Import)
}

// ── 仪表盘 ──

func registerDashboardRoutes(authed *gin.RouterGroup, d Deps, ctl *controller.DashboardController) {
	if ctl == nil {
		return
	}
	dash := authed.Group("/dashboard")
	dash.Use(middleware.RequirePerm("cluster:read"))
	dash.GET("/clusters/:id/overview", middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.GetClusterOverview)
	dash.GET("/clusters/:id/certificate-risks", ctl.GetClusterCertificateRisks)
}

// ── K8s 资源 ──

func registerK8sRoutes(authed *gin.RouterGroup, d Deps, ctl *controller.K8sController) {
	if ctl == nil {
		return
	}
	k8s := authed.Group("")
	readPerm := middleware.RequirePerm("k8s:read")
	writePerm := middleware.RequirePerm("k8s:write")
	secretRevealPerm := middleware.RequirePerm("k8s:secret_reveal")
	rbacReadPerm := middleware.RequirePerm("k8s:rbac_read")
	rbacWritePerm := middleware.RequirePerm("k8s:rbac_write")
	execPerm := middleware.RequirePerm("k8s:exec")
	resourceSupportReadPerm := middleware.RequireAnyPerm("k8s:read", "k8s:rbac_read")

	// Namespace
	k8s.GET("/clusters/:id/namespaces", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListNamespaces)
	k8s.POST("/clusters/:id/namespaces", writePerm, ctl.CreateNamespace)
	k8s.DELETE("/clusters/:id/namespaces/:ns", writePerm, ctl.DeleteNamespace)
	k8s.GET("/clusters/:id/namespaces/:ns/yaml", readPerm, ctl.GetNamespaceYAML)
	k8s.GET("/clusters/:id/namespaces/:ns/resources-summary", readPerm, ctl.GetNamespaceResourcesSummary)

	// Node
	k8s.GET("/clusters/:id/nodes", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListNodes)
	k8s.GET("/clusters/:id/nodes/:name/detail", readPerm, ctl.GetNodeDetail)
	k8s.GET("/clusters/:id/nodes/:name/yaml", readPerm, ctl.GetNodeYAML)
	k8s.GET("/clusters/:id/nodes/:name/pods", readPerm, ctl.ListNodePods)
	k8s.GET("/clusters/:id/nodes/:name/events", readPerm, ctl.ListNodeEvents)
	k8s.POST("/clusters/:id/nodes/:name/cordon", writePerm, ctl.CordonNode)
	k8s.POST("/clusters/:id/nodes/:name/uncordon", writePerm, ctl.UncordonNode)
	k8s.POST("/clusters/:id/nodes/:name/drain", writePerm, ctl.DrainNode)
	k8s.DELETE("/clusters/:id/nodes/:name", writePerm, ctl.DeleteNode)

	// Pod
	k8s.GET("/clusters/:id/pods", readPerm, ctl.ListPods)
	k8s.GET("/clusters/:id/pods/:ns/:pod/yaml", readPerm, ctl.GetPodYAML)
	k8s.GET("/clusters/:id/pods/:ns/:pod/logs", readPerm, ctl.GetPodLogs)
	k8s.POST("/clusters/:id/pods/:ns/:pod/logs/session", readPerm, ctl.CreatePodLogSession)
	k8s.DELETE("/clusters/:id/pods/:ns/:pod", writePerm, ctl.DeletePod)
	k8s.POST("/clusters/:id/pods/:ns/:pod/exec", execPerm, ctl.CreatePodExecSession)

	// Workload
	k8s.GET("/clusters/:id/workloads", readPerm, ctl.ListWorkloads)
	k8s.GET("/clusters/:id/workloads/deployments/:ns/:name/rollout-history", readPerm, ctl.GetRolloutHistory)
	k8s.POST("/clusters/:id/workloads/deployments/:ns/:name/rollout-undo", writePerm, ctl.RolloutUndo)
	k8s.PATCH("/clusters/:id/workloads/scale", writePerm, ctl.ScaleWorkload)
	k8s.PATCH("/clusters/:id/workloads/restart", writePerm, ctl.RestartWorkload)
	k8s.PATCH("/clusters/:id/workloads/image", writePerm, ctl.UpdateImage)
	k8s.PATCH("/clusters/:id/workloads/rollout-pause", writePerm, ctl.UpdateWorkloadPaused)
	k8s.PATCH("/clusters/:id/workloads/deployments/edit", writePerm, ctl.EditDeployment)
	k8s.PATCH("/clusters/:id/workloads/statefulsets/edit", writePerm, ctl.EditStatefulSet)
	k8s.PATCH("/clusters/:id/workloads/daemonsets/edit", writePerm, ctl.EditDaemonSet)
	k8s.PATCH("/clusters/:id/workloads/yaml/edit", writePerm, ctl.EditWorkloadYAML)
	k8s.DELETE("/clusters/:id/workloads/:kind/:ns/:name", writePerm, ctl.DeleteWorkload)
	k8s.GET("/clusters/:id/workloads/:kind/:ns/:name/yaml", readPerm, ctl.GetWorkloadYAML)

	// ReplicaSet
	k8s.GET("/clusters/:id/replicasets", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListReplicaSets)
	k8s.PATCH("/clusters/:id/replicasets/edit", writePerm, ctl.EditReplicaSet)
	k8s.DELETE("/clusters/:id/replicasets/:ns/:name", writePerm, ctl.DeleteReplicaSet)
	k8s.GET("/clusters/:id/replicasets/:ns/:name/yaml", readPerm, ctl.GetReplicaSetYAML)

	// Service
	k8s.GET("/clusters/:id/services", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListServices)
	k8s.PATCH("/clusters/:id/services/edit", writePerm, ctl.EditService)
	k8s.DELETE("/clusters/:id/services/:ns/:name", writePerm, ctl.DeleteService)
	k8s.GET("/clusters/:id/services/:ns/:name/yaml", readPerm, ctl.GetServiceYAML)

	// Ingress
	k8s.GET("/clusters/:id/ingresses", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListIngresses)
	k8s.PATCH("/clusters/:id/ingresses/edit", writePerm, ctl.EditIngress)
	k8s.DELETE("/clusters/:id/ingresses/:ns/:name", writePerm, ctl.DeleteIngress)
	k8s.GET("/clusters/:id/ingresses/:ns/:name/yaml", readPerm, ctl.GetIngressYAML)

	// NetworkPolicy
	k8s.GET("/clusters/:id/networkpolicies", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListNetworkPolicies)
	k8s.PATCH("/clusters/:id/networkpolicies/edit", writePerm, ctl.EditNetworkPolicy)
	k8s.DELETE("/clusters/:id/networkpolicies/:ns/:name", writePerm, ctl.DeleteNetworkPolicy)
	k8s.GET("/clusters/:id/networkpolicies/:ns/:name/yaml", readPerm, ctl.GetNetworkPolicyYAML)

	// IngressClass
	k8s.GET("/clusters/:id/ingressclasses", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListIngressClasses)
	k8s.PATCH("/clusters/:id/ingressclasses/edit", writePerm, ctl.EditIngressClass)
	k8s.DELETE("/clusters/:id/ingressclasses/:name", writePerm, ctl.DeleteIngressClass)
	k8s.GET("/clusters/:id/ingressclasses/:name/yaml", readPerm, ctl.GetIngressClassYAML)

	// ConfigMap
	k8s.GET("/clusters/:id/configmaps", readPerm, ctl.ListConfigMaps)
	k8s.PATCH("/clusters/:id/configmaps/edit", writePerm, ctl.EditConfigMap)
	k8s.DELETE("/clusters/:id/configmaps/:ns/:name", writePerm, ctl.DeleteConfigMap)
	k8s.GET("/clusters/:id/configmaps/:ns/:name/yaml", readPerm, ctl.GetConfigMapYAML)
	k8s.GET("/clusters/:id/configmaps/:ns/:name/related", readPerm, ctl.GetConfigMapRelated)

	// Secret
	k8s.GET("/clusters/:id/secrets", readPerm, ctl.ListSecrets)
	k8s.PATCH("/clusters/:id/secrets/edit", writePerm, ctl.EditSecret)
	k8s.DELETE("/clusters/:id/secrets/:ns/:name", writePerm, ctl.DeleteSecret)
	k8s.GET("/clusters/:id/secrets/:ns/:name/reveal", secretRevealPerm, ctl.GetSecretReveal)
	k8s.GET("/clusters/:id/secrets/:ns/:name/yaml", readPerm, ctl.GetSecretYAML)
	k8s.GET("/clusters/:id/secrets/:ns/:name/related", readPerm, ctl.GetSecretRelated)

	// ServiceAccount
	k8s.GET("/clusters/:id/serviceaccounts", readPerm, ctl.ListServiceAccounts)
	k8s.PATCH("/clusters/:id/serviceaccounts/edit", writePerm, ctl.EditServiceAccount)
	k8s.DELETE("/clusters/:id/serviceaccounts/:ns/:name", writePerm, ctl.DeleteServiceAccount)
	k8s.GET("/clusters/:id/serviceaccounts/:ns/:name/yaml", readPerm, ctl.GetServiceAccountYAML)

	// Endpoints
	k8s.GET("/clusters/:id/endpoints", readPerm, ctl.ListEndpoints)
	k8s.PATCH("/clusters/:id/endpoints/edit", writePerm, ctl.EditEndpoints)
	k8s.DELETE("/clusters/:id/endpoints/:ns/:name", writePerm, ctl.DeleteEndpoints)
	k8s.GET("/clusters/:id/endpoints/:ns/:name/yaml", readPerm, ctl.GetEndpointsYAML)

	// EndpointSlice
	k8s.GET("/clusters/:id/endpointslices", readPerm, ctl.ListEndpointSlices)
	k8s.PATCH("/clusters/:id/endpointslices/edit", writePerm, ctl.EditEndpointSlice)
	k8s.DELETE("/clusters/:id/endpointslices/:ns/:name", writePerm, ctl.DeleteEndpointSlice)
	k8s.GET("/clusters/:id/endpointslices/:ns/:name/yaml", readPerm, ctl.GetEndpointSliceYAML)

	// Lease
	k8s.GET("/clusters/:id/leases", readPerm, ctl.ListLeases)
	k8s.PATCH("/clusters/:id/leases/edit", writePerm, ctl.EditLease)
	k8s.DELETE("/clusters/:id/leases/:ns/:name", writePerm, ctl.DeleteLease)
	k8s.GET("/clusters/:id/leases/:ns/:name/yaml", readPerm, ctl.GetLeaseYAML)

	// PDB
	k8s.GET("/clusters/:id/pdbs", readPerm, ctl.ListPDBs)
	k8s.PATCH("/clusters/:id/pdbs/edit", writePerm, ctl.EditPDB)
	k8s.DELETE("/clusters/:id/pdbs/:ns/:name", writePerm, ctl.DeletePDB)
	k8s.GET("/clusters/:id/pdbs/:ns/:name/yaml", readPerm, ctl.GetPDBYAML)

	// Role
	k8s.GET("/clusters/:id/roles", rbacReadPerm, ctl.ListRoles)
	k8s.PATCH("/clusters/:id/roles/edit", rbacWritePerm, ctl.EditRole)
	k8s.DELETE("/clusters/:id/roles/:ns/:name", rbacWritePerm, ctl.DeleteRole)
	k8s.GET("/clusters/:id/roles/:ns/:name/yaml", rbacReadPerm, ctl.GetRoleYAML)

	// ClusterRole
	k8s.GET("/clusters/:id/clusterroles", rbacReadPerm, ctl.ListClusterRoles)
	k8s.PATCH("/clusters/:id/clusterroles/edit", rbacWritePerm, ctl.EditClusterRole)
	k8s.DELETE("/clusters/:id/clusterroles/:name", rbacWritePerm, ctl.DeleteClusterRole)
	k8s.GET("/clusters/:id/clusterroles/:name/yaml", rbacReadPerm, ctl.GetClusterRoleYAML)

	// RoleBinding
	k8s.GET("/clusters/:id/rolebindings", rbacReadPerm, ctl.ListRoleBindings)
	k8s.PATCH("/clusters/:id/rolebindings/edit", rbacWritePerm, ctl.EditRoleBinding)
	k8s.DELETE("/clusters/:id/rolebindings/:ns/:name", rbacWritePerm, ctl.DeleteRoleBinding)
	k8s.GET("/clusters/:id/rolebindings/:ns/:name/yaml", rbacReadPerm, ctl.GetRoleBindingYAML)

	// ClusterRoleBinding
	k8s.GET("/clusters/:id/clusterrolebindings", rbacReadPerm, ctl.ListClusterRoleBindings)
	k8s.PATCH("/clusters/:id/clusterrolebindings/edit", rbacWritePerm, ctl.EditClusterRoleBinding)
	k8s.DELETE("/clusters/:id/clusterrolebindings/:name", rbacWritePerm, ctl.DeleteClusterRoleBinding)
	k8s.GET("/clusters/:id/clusterrolebindings/:name/yaml", rbacReadPerm, ctl.GetClusterRoleBindingYAML)

	// HPA
	k8s.GET("/clusters/:id/hpas", readPerm, ctl.ListHPAs)
	k8s.PATCH("/clusters/:id/hpas/edit", writePerm, ctl.EditHPA)
	k8s.DELETE("/clusters/:id/hpas/:ns/:name", writePerm, ctl.DeleteHPA)
	k8s.GET("/clusters/:id/hpas/:ns/:name/yaml", readPerm, ctl.GetHPAYAML)

	// Event
	k8s.GET("/clusters/:id/events", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListEvents)

	// PVC
	k8s.GET("/clusters/:id/pvcs", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListPVCs)
	k8s.POST("/clusters/:id/pvcs", writePerm, ctl.CreatePVC)
	k8s.DELETE("/clusters/:id/pvcs/:ns/:name", writePerm, ctl.DeletePVC)
	k8s.GET("/clusters/:id/pvcs/:ns/:name/yaml", readPerm, ctl.GetPVCYAML)

	// PV
	k8s.GET("/clusters/:id/pvs", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListPVs)
	k8s.DELETE("/clusters/:id/pvs/:name", writePerm, ctl.DeletePV)
	k8s.GET("/clusters/:id/pvs/:name/yaml", readPerm, ctl.GetPVYAML)

	// StorageClass
	k8s.GET("/clusters/:id/storageclasses", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListStorageClasses)
	k8s.DELETE("/clusters/:id/storageclasses/:name", writePerm, ctl.DeleteStorageClass)
	k8s.GET("/clusters/:id/storageclasses/:name/yaml", readPerm, ctl.GetStorageClassYAML)
	k8s.PATCH("/clusters/:id/storageclasses/edit", writePerm, ctl.EditStorageClass)
	k8s.GET("/clusters/:id/csidrivers", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListCSIDrivers)
	k8s.PATCH("/clusters/:id/csidrivers/edit", writePerm, ctl.EditCSIDriver)
	k8s.DELETE("/clusters/:id/csidrivers/:name", writePerm, ctl.DeleteCSIDriver)
	k8s.GET("/clusters/:id/csidrivers/:name/yaml", readPerm, ctl.GetCSIDriverYAML)
	k8s.GET("/clusters/:id/csinodes", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListCSINodes)
	k8s.PATCH("/clusters/:id/csinodes/edit", writePerm, ctl.EditCSINode)
	k8s.DELETE("/clusters/:id/csinodes/:name", writePerm, ctl.DeleteCSINode)
	k8s.GET("/clusters/:id/csinodes/:name/yaml", readPerm, ctl.GetCSINodeYAML)
	k8s.GET("/clusters/:id/csistoragecapacities", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListCSIStorageCapacities)
	k8s.PATCH("/clusters/:id/csistoragecapacities/edit", writePerm, ctl.EditCSIStorageCapacity)
	k8s.DELETE("/clusters/:id/csistoragecapacities/:ns/:name", writePerm, ctl.DeleteCSIStorageCapacity)
	k8s.GET("/clusters/:id/csistoragecapacities/:ns/:name/yaml", readPerm, ctl.GetCSIStorageCapacityYAML)
	k8s.GET("/clusters/:id/resource-support", resourceSupportReadPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.GetResourceSupport)
	k8s.GET("/clusters/:id/storage-snapshot-support", resourceSupportReadPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.GetStorageSnapshotSupport)

	// VolumeAttachment
	k8s.GET("/clusters/:id/volumeattachments", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListVolumeAttachments)
	k8s.PATCH("/clusters/:id/volumeattachments/edit", writePerm, ctl.EditVolumeAttachment)
	k8s.DELETE("/clusters/:id/volumeattachments/:name", writePerm, ctl.DeleteVolumeAttachment)
	k8s.GET("/clusters/:id/volumeattachments/:name/yaml", readPerm, ctl.GetVolumeAttachmentYAML)

	// VolumeSnapshot
	k8s.GET("/clusters/:id/volumesnapshots", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListVolumeSnapshots)
	k8s.PATCH("/clusters/:id/volumesnapshots/edit", writePerm, ctl.EditVolumeSnapshot)
	k8s.DELETE("/clusters/:id/volumesnapshots/:ns/:name", writePerm, ctl.DeleteVolumeSnapshot)
	k8s.GET("/clusters/:id/volumesnapshots/:ns/:name/yaml", readPerm, ctl.GetVolumeSnapshotYAML)

	// VolumeSnapshotClass
	k8s.GET("/clusters/:id/volumesnapshotclasses", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListVolumeSnapshotClasses)
	k8s.PATCH("/clusters/:id/volumesnapshotclasses/edit", writePerm, ctl.EditVolumeSnapshotClass)
	k8s.DELETE("/clusters/:id/volumesnapshotclasses/:name", writePerm, ctl.DeleteVolumeSnapshotClass)
	k8s.GET("/clusters/:id/volumesnapshotclasses/:name/yaml", readPerm, ctl.GetVolumeSnapshotClassYAML)

	// VolumeSnapshotContent
	k8s.GET("/clusters/:id/volumesnapshotcontents", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListVolumeSnapshotContents)
	k8s.PATCH("/clusters/:id/volumesnapshotcontents/edit", writePerm, ctl.EditVolumeSnapshotContent)
	k8s.DELETE("/clusters/:id/volumesnapshotcontents/:name", writePerm, ctl.DeleteVolumeSnapshotContent)
	k8s.GET("/clusters/:id/volumesnapshotcontents/:name/yaml", readPerm, ctl.GetVolumeSnapshotContentYAML)

	// ResourceQuota
	k8s.GET("/clusters/:id/resourcequotas", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListResourceQuotas)
	k8s.PATCH("/clusters/:id/resourcequotas/edit", writePerm, ctl.EditResourceQuota)
	k8s.DELETE("/clusters/:id/resourcequotas/:ns/:name", writePerm, ctl.DeleteResourceQuota)
	k8s.GET("/clusters/:id/resourcequotas/:ns/:name/yaml", readPerm, ctl.GetResourceQuotaYAML)

	// LimitRange
	k8s.GET("/clusters/:id/limitranges", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListLimitRanges)
	k8s.PATCH("/clusters/:id/limitranges/edit", writePerm, ctl.EditLimitRange)
	k8s.DELETE("/clusters/:id/limitranges/:ns/:name", writePerm, ctl.DeleteLimitRange)
	k8s.GET("/clusters/:id/limitranges/:ns/:name/yaml", readPerm, ctl.GetLimitRangeYAML)

	// CustomResourceDefinition
	k8s.GET("/clusters/:id/customresourcedefinitions", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListCustomResourceDefinitions)
	k8s.PATCH("/clusters/:id/customresourcedefinitions/edit", writePerm, ctl.EditCustomResourceDefinition)
	k8s.DELETE("/clusters/:id/customresourcedefinitions/:name", writePerm, ctl.DeleteCustomResourceDefinition)
	k8s.GET("/clusters/:id/customresourcedefinitions/:name/yaml", readPerm, ctl.GetCustomResourceDefinitionYAML)

	// APIService
	k8s.GET("/clusters/:id/apiservices", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListAPIServices)
	k8s.PATCH("/clusters/:id/apiservices/edit", writePerm, ctl.EditAPIService)
	k8s.DELETE("/clusters/:id/apiservices/:name", writePerm, ctl.DeleteAPIService)
	k8s.GET("/clusters/:id/apiservices/:name/yaml", readPerm, ctl.GetAPIServiceYAML)

	// PriorityClass
	k8s.GET("/clusters/:id/priorityclasses", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListPriorityClasses)
	k8s.PATCH("/clusters/:id/priorityclasses/edit", writePerm, ctl.EditPriorityClass)
	k8s.DELETE("/clusters/:id/priorityclasses/:name", writePerm, ctl.DeletePriorityClass)
	k8s.GET("/clusters/:id/priorityclasses/:name/yaml", readPerm, ctl.GetPriorityClassYAML)
	k8s.GET("/clusters/:id/runtimeclasses", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListRuntimeClasses)
	k8s.PATCH("/clusters/:id/runtimeclasses/edit", writePerm, ctl.EditRuntimeClass)
	k8s.DELETE("/clusters/:id/runtimeclasses/:name", writePerm, ctl.DeleteRuntimeClass)
	k8s.GET("/clusters/:id/runtimeclasses/:name/yaml", readPerm, ctl.GetRuntimeClassYAML)

	// ValidatingWebhookConfiguration
	k8s.GET("/clusters/:id/validatingwebhookconfigurations", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListValidatingWebhookConfigurations)
	k8s.PATCH("/clusters/:id/validatingwebhookconfigurations/edit", writePerm, ctl.EditValidatingWebhookConfiguration)
	k8s.DELETE("/clusters/:id/validatingwebhookconfigurations/:name", writePerm, ctl.DeleteValidatingWebhookConfiguration)
	k8s.GET("/clusters/:id/validatingwebhookconfigurations/:name/yaml", readPerm, ctl.GetValidatingWebhookConfigurationYAML)

	// MutatingWebhookConfiguration
	k8s.GET("/clusters/:id/mutatingwebhookconfigurations", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListMutatingWebhookConfigurations)
	k8s.PATCH("/clusters/:id/mutatingwebhookconfigurations/edit", writePerm, ctl.EditMutatingWebhookConfiguration)
	k8s.DELETE("/clusters/:id/mutatingwebhookconfigurations/:name", writePerm, ctl.DeleteMutatingWebhookConfiguration)
	k8s.GET("/clusters/:id/mutatingwebhookconfigurations/:name/yaml", readPerm, ctl.GetMutatingWebhookConfigurationYAML)
	k8s.GET("/clusters/:id/validatingadmissionpolicies", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListValidatingAdmissionPolicies)
	k8s.PATCH("/clusters/:id/validatingadmissionpolicies/edit", writePerm, ctl.EditValidatingAdmissionPolicy)
	k8s.DELETE("/clusters/:id/validatingadmissionpolicies/:name", writePerm, ctl.DeleteValidatingAdmissionPolicy)
	k8s.GET("/clusters/:id/validatingadmissionpolicies/:name/yaml", readPerm, ctl.GetValidatingAdmissionPolicyYAML)
	k8s.GET("/clusters/:id/validatingadmissionpolicybindings", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListValidatingAdmissionPolicyBindings)
	k8s.PATCH("/clusters/:id/validatingadmissionpolicybindings/edit", writePerm, ctl.EditValidatingAdmissionPolicyBinding)
	k8s.DELETE("/clusters/:id/validatingadmissionpolicybindings/:name", writePerm, ctl.DeleteValidatingAdmissionPolicyBinding)
	k8s.GET("/clusters/:id/validatingadmissionpolicybindings/:name/yaml", readPerm, ctl.GetValidatingAdmissionPolicyBindingYAML)

	// Job
	k8s.GET("/clusters/:id/jobs", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListJobs)
	k8s.PATCH("/clusters/:id/jobs/edit", writePerm, ctl.EditJob)
	k8s.DELETE("/clusters/:id/jobs/completed", writePerm, ctl.DeleteCompletedJobs)
	k8s.DELETE("/clusters/:id/jobs/:ns/:name", writePerm, ctl.DeleteJob)
	k8s.GET("/clusters/:id/jobs/:ns/:name/yaml", readPerm, ctl.GetJobYAML)

	// CronJob
	k8s.GET("/clusters/:id/cronjobs", readPerm, middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.ListCronJobs)
	k8s.PATCH("/clusters/:id/cronjobs/edit", writePerm, ctl.EditCronJob)
	k8s.POST("/clusters/:id/cronjobs/:ns/:name/trigger", writePerm, ctl.TriggerCronJob)
	k8s.PATCH("/clusters/:id/cronjobs/:ns/:name/suspend", writePerm, ctl.SuspendCronJob)
	k8s.DELETE("/clusters/:id/cronjobs/:ns/:name", writePerm, ctl.DeleteCronJob)
	k8s.GET("/clusters/:id/cronjobs/:ns/:name/yaml", readPerm, ctl.GetCronJobYAML)
}

// ── WebSocket ──

func registerWebSocketRoutes(authed *gin.RouterGroup, k8sCtl *controller.K8sController) {
	ws := authed.Group("/ws")
	if k8sCtl != nil {
		ws.GET("/pod-log", middleware.RequirePerm("k8s:read"), k8sCtl.PodLogWS)
		ws.GET("/pod-exec", middleware.RequirePerm("k8s:exec"), k8sCtl.PodExecWS)
	}
}
