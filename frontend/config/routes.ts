/**
 * AIOPS 智能运维平台 - 路由配置
 *
 * 架构说明：
 * - 全局路由（/dashboard, /clusters 等）→ currentCluster = null，左侧显示全局菜单
 * - 集群钻取路由（/cluster/*）→ currentCluster 有值，左侧显示 K8s 资源目录树
 * - 菜单由 app.tsx 的 menuDataRender 完全控制，路由仅负责 URL 匹配
 *
 * 菜单结构参照 frontend-old：
 * - 仪表盘 /dashboard
 * - 集群管理 /clusters（含子菜单：集群列表、集群导入、资源拓扑）
 * - 全局告警中心 /monitor（含子菜单：监控仪表盘、告警规则、事件流）
 * - 全局日志分析 /logs
 * - 系统配置 /config（含子菜单：YAML编辑、用户管理、角色管理、审计日志、系统设置）
 * - 智能根因定位 /ai（含子菜单：对话式运维、历史记录、模型配置）
 * - 自动化运维 /deploy（含子菜单：部署计划、服务器管理、凭据管理）
 */
const routes: any[] = [
  {
    path: '/login',
    component: '@/features/iam/pages/login',
    layout: false,
  },
  {
    path: '/k8s/terminal',
    component: '@/features/kops/pages/k8s/terminal',
    layout: false,
  },
  {
    path: '/',
    redirect: '/dashboard',
  },
  // ========== 全局路由（currentCluster = null） ==========
  {
    name: '仪表盘',
    path: '/dashboard',
    component: '@/features/fleet/pages/dashboard',
  },
  // ---- 集群管理（父级菜单：集群列表、集群导入、资源拓扑） ----
  {
    name: '集群管理',
    path: '/clusters',
    component: '@/features/fleet/pages/clusters',
  },
  {
    name: '部署K8S集群',
    path: '/clusters/provision',
    component: '@/features/provisioning/pages/deploy/index',
  },
  {
    path: '/clusters/provision/create',
    component: '@/features/provisioning/pages/deploy/create',
    hideInMenu: true,
  },
  {
    path: '/clusters/provision/:id',
    component: '@/features/provisioning/pages/deploy/detail',
    hideInMenu: true,
  },
  {
    path: '/clusters/provision/:id/edit',
    component: '@/features/provisioning/pages/deploy/create',
    hideInMenu: true,
  },
  {
    name: '主机资源池',
    path: '/clusters/hosts',
    component: '@/features/provisioning/pages/deploy/servers',
  },
  {
    name: '项目管理',
    path: '/projects',
    component: '@/features/workspace/pages/projects/index',
  },
  {
    name: '应用商店',
    path: '/app-store',
    routes: [
      { path: '/app-store', redirect: '/app-store/yaml' },
      { name: 'YAML 模板', path: '/app-store/yaml', component: '@/features/provisioning/pages/app-store/index' },
      { name: 'Helm Chart', path: '/app-store/helm', component: '@/features/provisioning/pages/app-store/index' },
    ],
  },
  {
    name: '集群详情',
    path: '/clusters/:id',
    hideInMenu: true,
    component: '@/features/fleet/pages/clusters/detail',
  },
  {
    name: '集群导入',
    path: '/clusters/import',
    hideInMenu: true,
    component: '@/features/fleet/pages/clusters/import',
  },
  // ---- 全局告警中心 ----
  {
    name: '监控告警',
    path: '/monitor',
    routes: [
      { path: '/monitor', redirect: '/monitor/dashboard' },
      { name: '监控仪表盘', path: '/monitor/dashboard', component: '@/features/incident/pages/monitor/dashboard' },
      { name: '告警规则', path: '/monitor/alerts', component: '@/features/incident/pages/monitor/alerts' },
      { name: '事件流', path: '/monitor/events', component: '@/features/incident/pages/monitor/events' },
    ],
  },
  // ---- 全局日志分析 ----
  {
    name: '日志分析',
    path: '/logs',
    component: '@/features/kops/pages/logs',
  },
  // ---- 系统配置 ----
  {
    name: '系统配置',
    path: '/config',
    routes: [
      { path: '/config', redirect: '/config/settings' },
      { name: '用户管理', path: '/config/users', component: '@/features/platform/pages/system/users' },
      { name: '角色管理', path: '/config/roles', component: '@/features/platform/pages/system/roles' },
      { name: '审计日志', path: '/config/audit-logs', component: '@/features/platform/pages/system/audit-logs' },
      { name: '凭据库', path: '/config/credentials', component: '@/features/provisioning/pages/deploy/credentials' },
      { name: '部署手册', path: '/config/deploy-assets', component: '@/features/provisioning/pages/deploy/config' },
      { name: '系统设置', path: '/config/settings', component: '@/features/platform/pages/system/settings' },
    ],
  },
  // ---- 智能根因定位 ----
  {
    name: 'AI 运维助手',
    path: '/ai',
    routes: [
      { path: '/ai', redirect: '/ai/chat' },
      { name: '对话式运维', path: '/ai/chat', component: '@/features/ai/pages/chat' },
      { name: '历史记录', path: '/ai/history', component: '@/features/ai/pages/history' },
      { name: '模型配置', path: '/ai/settings', component: '@/features/ai/pages/settings' },
    ],
  },
  // ---- CI/CD 持续交付 ----
  {
    name: 'CI/CD',
    path: '/cicd',
    routes: [
      { path: '/cicd', redirect: '/cicd/overview' },
      { name: '总览', path: '/cicd/overview', component: '@/features/provisioning/pages/cicd/overview' },
      { name: '流水线', path: '/cicd/pipelines', component: '@/features/provisioning/pages/cicd/pipelines' },
      { path: '/cicd/pipelines/:id', component: '@/features/provisioning/pages/cicd/pipeline-detail', hideInMenu: true },
      { name: '执行记录', path: '/cicd/runs', component: '@/features/provisioning/pages/cicd/runs' },
      { path: '/cicd/runs/:id', component: '@/features/provisioning/pages/cicd/run-detail', hideInMenu: true },
      { name: '制品仓库', path: '/cicd/artifacts', component: '@/features/provisioning/pages/cicd/artifacts' },
      { path: '/cicd/artifacts/:id', component: '@/features/provisioning/pages/cicd/artifact-detail', hideInMenu: true },
      { name: '环境管理', path: '/cicd/environments', component: '@/features/provisioning/pages/cicd/environments' },
      { path: '/cicd/environments/:id', component: '@/features/provisioning/pages/cicd/environment-detail', hideInMenu: true },
    ],
  },
  // 兼容已收藏的旧链接；实际功能已按业务域重新归位。
  { path: '/deploy', redirect: '/clusters/provision' },
  { path: '/deploy/plans', redirect: '/clusters/provision' },
  { path: '/deploy/plans/create', redirect: '/clusters/provision/create' },
  { path: '/deploy/plans/:id/edit', redirect: '/clusters/provision/:id/edit' },
  { path: '/deploy/plans/:id', redirect: '/clusters/provision/:id' },
  { path: '/deploy/servers', redirect: '/clusters/hosts' },
  { path: '/deploy/credentials', redirect: '/config/credentials' },
  { path: '/deploy/config', redirect: '/config/deploy-assets' },
  { path: '/automation/assets', redirect: '/config/deploy-assets' },
  { path: '/automation', redirect: '/cicd/overview' },
  { path: '/automation/overview', redirect: '/cicd/overview' },
  // ========== 集群钻取路由（currentCluster 有值） ==========
  {
    path: '/cluster',
    component: '@/layouts/ClusterLayout',
    hideInMenu: true,
    routes: [
      {
        path: '/cluster/dashboard',
        component: '@/features/kops/pages/cluster/dashboard',
      },
      {
        path: '/cluster/pods',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/podmetrics',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/deployments',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/statefulsets',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/daemonsets',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/replicasets',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/pdbs',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/hpas',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/jobs',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/cronjobs',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/services',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/endpoints',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/endpointslices',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/network-policies',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/ingresses',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/ingress-classes',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/pvcs',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/pvs',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/volume-snapshots',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/volume-snapshot-classes',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/volume-snapshot-contents',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/storage-classes',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/csi-drivers',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/csi-nodes',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/csi-storage-capacities',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/volume-attachments',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/resource-quotas',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/limit-ranges',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/configmaps',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/secrets',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/service-accounts',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/roles',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/role-bindings',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/cluster-roles',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/cluster-role-bindings',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/nodes',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/leases',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/events',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/crds',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/api-services',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/priority-classes',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/runtime-classes',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/validating-webhooks',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/mutating-webhooks',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/validating-admission-policies',
        component: '@/features/kops/pages/cluster/redirect',
      },
      {
        path: '/cluster/validating-admission-policy-bindings',
        component: '@/features/kops/pages/cluster/redirect',
      },
    ],
  },
  // ========== K8s 运维页面（隐藏菜单，通过集群资源树访问） ==========
  // 参照 frontend-old 的 buildTree 完整资源目录结构
  {
    path: '/k8s/:clusterId',
    hideInMenu: true,
    routes: [
      // ---- 仪表盘 ----
      {
        path: '/k8s/:clusterId/dashboard',
        component: '@/features/kops/pages/k8s/$clusterId',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/advanced-resources',
        component: '@/features/kops/pages/k8s/$clusterId/advanced-resources',
        hideInMenu: true,
      },
      // ---- 运维工具 ----
      {
        path: '/k8s/:clusterId/log-workbench',
        component: '@/features/kops/pages/k8s/$clusterId/log-workbench',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/manifest-apply',
        component: '@/features/kops/pages/k8s/$clusterId/manifest-apply',
        hideInMenu: true,
      },
      // ---- 工作负载 ----
      {
        path: '/k8s/:clusterId/pods',
        component: '@/features/kops/pages/k8s/$clusterId/pods',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/podmetrics',
        component: '@/features/kops/pages/k8s/$clusterId/podmetrics',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/workloads',
        component: '@/features/kops/pages/k8s/$clusterId/workloads',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/deployments',
        component: '@/features/kops/pages/k8s/$clusterId/deployments',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/statefulsets',
        component: '@/features/kops/pages/k8s/$clusterId/statefulsets',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/daemonsets',
        component: '@/features/kops/pages/k8s/$clusterId/daemonsets',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/replicasets',
        component: '@/features/kops/pages/k8s/$clusterId/replicasets',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/pdbs',
        component: '@/features/kops/pages/k8s/$clusterId/pdbs',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/hpas',
        component: '@/features/kops/pages/k8s/$clusterId/hpas',
        hideInMenu: true,
      },
      // ---- 作业 ----
      {
        path: '/k8s/:clusterId/jobs',
        component: '@/features/kops/pages/k8s/$clusterId/jobs',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/cronjobs',
        component: '@/features/kops/pages/k8s/$clusterId/cronjobs',
        hideInMenu: true,
      },
      // ---- 网络资源 ----
      {
        path: '/k8s/:clusterId/services',
        component: '@/features/kops/pages/k8s/$clusterId/services',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/endpoints',
        component: '@/features/kops/pages/k8s/$clusterId/endpoints',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/endpointslices',
        component: '@/features/kops/pages/k8s/$clusterId/endpointslices',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/network-policies',
        component: '@/features/kops/pages/k8s/$clusterId/network-policies',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/ingresses',
        component: '@/features/kops/pages/k8s/$clusterId/ingresses',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/ingress-classes',
        component: '@/features/kops/pages/k8s/$clusterId/ingressclasses',
        hideInMenu: true,
      },
      // ---- 数据存储 ----
      {
        path: '/k8s/:clusterId/pvcs',
        component: '@/features/kops/pages/k8s/$clusterId/pvc',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/pvs',
        component: '@/features/kops/pages/k8s/$clusterId/pv',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/volume-snapshots',
        component: '@/features/kops/pages/k8s/$clusterId/volumesnapshots',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/volume-snapshot-classes',
        component: '@/features/kops/pages/k8s/$clusterId/volumesnapshotclasses',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/volume-snapshot-contents',
        component: '@/features/kops/pages/k8s/$clusterId/volumesnapshotcontents',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/storage-classes',
        component: '@/features/kops/pages/k8s/$clusterId/storage-classes',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/csi-drivers',
        component: '@/features/kops/pages/k8s/$clusterId/csidrivers',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/csi-nodes',
        component: '@/features/kops/pages/k8s/$clusterId/csinodes',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/csi-storage-capacities',
        component: '@/features/kops/pages/k8s/$clusterId/csistoragecapacities',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/volume-attachments',
        component: '@/features/kops/pages/k8s/$clusterId/volumeattachments',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/resource-quotas',
        component: '@/features/kops/pages/k8s/$clusterId/resource-quotas',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/limit-ranges',
        component: '@/features/kops/pages/k8s/$clusterId/limitranges',
        hideInMenu: true,
      },
      // ---- 配置文件 ----
      {
        path: '/k8s/:clusterId/configmaps',
        component: '@/features/kops/pages/k8s/$clusterId/configmaps',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/secrets',
        component: '@/features/kops/pages/k8s/$clusterId/secrets',
        hideInMenu: true,
      },
      // ---- 访问控制 ----
      {
        path: '/k8s/:clusterId/service-accounts',
        component: '@/features/kops/pages/k8s/$clusterId/service-accounts',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/roles',
        component: '@/features/kops/pages/k8s/$clusterId/roles',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/role-bindings',
        component: '@/features/kops/pages/k8s/$clusterId/role-bindings',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/cluster-roles',
        component: '@/features/kops/pages/k8s/$clusterId/clusterroles',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/cluster-role-bindings',
        component: '@/features/kops/pages/k8s/$clusterId/clusterrolebindings',
        hideInMenu: true,
      },
      // ---- 集群资源 ----
      {
        path: '/k8s/:clusterId/namespaces',
        component: '@/features/kops/pages/k8s/$clusterId/namespaces',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/nodes',
        component: '@/features/kops/pages/k8s/$clusterId/nodes',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/leases',
        component: '@/features/kops/pages/k8s/$clusterId/leases',
        hideInMenu: true,
      },
      // ---- 事件 ----
      {
        path: '/k8s/:clusterId/events',
        component: '@/features/kops/pages/k8s/$clusterId/events',
        hideInMenu: true,
      },
      // ---- 扩展治理 ----
      {
        path: '/k8s/:clusterId/crds',
        component: '@/features/kops/pages/k8s/$clusterId/crds',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/api-services',
        component: '@/features/kops/pages/k8s/$clusterId/apiservices',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/priority-classes',
        component: '@/features/kops/pages/k8s/$clusterId/priorityclasses',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/runtime-classes',
        component: '@/features/kops/pages/k8s/$clusterId/runtimeclasses',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/validating-webhooks',
        component: '@/features/kops/pages/k8s/$clusterId/validatingwebhooks',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/mutating-webhooks',
        component: '@/features/kops/pages/k8s/$clusterId/mutatingwebhooks',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/validating-admission-policies',
        component: '@/features/kops/pages/k8s/$clusterId/validatingadmissionpolicies',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/validating-admission-policy-bindings',
        component: '@/features/kops/pages/k8s/$clusterId/validatingadmissionpolicybindings',
        hideInMenu: true,
      },
      // ---- 治理分析 ----
      {
        path: '/k8s/:clusterId/permission-audits',
        component: '@/features/kops/pages/k8s/$clusterId/permission-audits',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/topology',
        component: '@/features/kops/pages/k8s/$clusterId/topology',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/helm-releases',
        component: '@/features/kops/pages/k8s/$clusterId/helm-releases',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/helm-repos',
        component: '@/features/kops/pages/k8s/$clusterId/helm-repos',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/resource-metrics',
        component: '@/features/kops/pages/k8s/$clusterId/resource-metrics',
        hideInMenu: true,
      },
    ],
  },
  { path: '/403', component: '@/pages/403' },
  { path: '/*', component: '@/pages/404' },
]

export default routes
