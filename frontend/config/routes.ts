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
    component: '@/pages/login',
    layout: false,
  },
  {
    path: '/k8s/terminal',
    component: '@/pages/k8s/terminal',
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
    component: '@/pages/dashboard',
  },
  // ---- 集群管理（父级菜单：集群列表、集群导入、资源拓扑） ----
  {
    name: '集群管理',
    path: '/clusters',
    component: '@/pages/clusters',
  },
  {
    name: '项目管理',
    path: '/projects',
    component: '@/pages/projects/index',
  },
  {
    name: '应用商店',
    path: '/app-store',
    component: '@/pages/app-store/index',
  },
  {
    name: '集群详情',
    path: '/clusters/:id',
    hideInMenu: true,
    component: '@/pages/clusters/detail',
  },
  {
    name: '集群导入',
    path: '/clusters/import',
    hideInMenu: true,
    component: '@/pages/clusters/import',
  },
  // ---- 全局告警中心 ----
  {
    name: '监控告警',
    path: '/monitor',
    routes: [
      { path: '/monitor', redirect: '/monitor/dashboard' },
      { name: '监控仪表盘', path: '/monitor/dashboard', component: '@/pages/monitor/dashboard' },
      { name: '告警规则', path: '/monitor/alerts', component: '@/pages/monitor/alerts' },
      { name: '事件流', path: '/monitor/events', component: '@/pages/monitor/events' },
    ],
  },
  // ---- 全局日志分析 ----
  {
    name: '日志分析',
    path: '/logs',
    component: '@/pages/logs',
  },
  // ---- 系统配置 ----
  {
    name: '系统配置',
    path: '/config',
    routes: [
      { path: '/config', redirect: '/config/yaml' },
      { name: 'YAML 编辑', path: '/config/yaml', component: '@/pages/config/yaml' },
      { name: '用户管理', path: '/config/users', component: '@/pages/system/users' },
      { name: '角色管理', path: '/config/roles', component: '@/pages/system/roles' },
      { name: '审计日志', path: '/config/audit-logs', component: '@/pages/system/audit-logs' },
      { name: '系统设置', path: '/config/settings', component: '@/pages/system/settings' },
    ],
  },
  // ---- 智能根因定位 ----
  {
    name: 'AI 运维助手',
    path: '/ai',
    routes: [
      { path: '/ai', redirect: '/ai/chat' },
      { name: '对话式运维', path: '/ai/chat', component: '@/pages/ai/chat' },
      { name: '历史记录', path: '/ai/history', component: '@/pages/ai/history' },
      { name: '模型配置', path: '/ai/settings', component: '@/pages/ai/settings' },
    ],
  },
  // ---- 自动化运维 ----
  {
    name: '自动化部署',
    path: '/deploy',
    routes: [
      { path: '/deploy', redirect: '/deploy/plans' },
      { name: '部署计划', path: '/deploy/plans', component: '@/pages/deploy/index' },
      { path: '/deploy/plans/create', component: '@/pages/deploy/create', hideInMenu: true },
      { path: '/deploy/plans/:id/edit', component: '@/pages/deploy/create', hideInMenu: true },
      { name: '服务器管理', path: '/deploy/servers', component: '@/pages/deploy/servers' },
      { name: '凭据管理', path: '/deploy/credentials', component: '@/pages/deploy/credentials' },
      { name: '部署配置', path: '/deploy/config', component: '@/pages/deploy/config' },
    ],
  },
  // ---- 资源拓扑（"集群管理"菜单的子项，独立路由） ----
  {
    name: '资源视图',
    path: '/topology',
    component: '@/pages/topology',
  },
  // ========== 集群钻取路由（currentCluster 有值） ==========
  {
    path: '/cluster',
    component: '@/layouts/ClusterLayout',
    hideInMenu: true,
    routes: [
      {
        path: '/cluster/dashboard',
        component: '@/pages/cluster/dashboard',
      },
      {
        path: '/cluster/pods',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/podmetrics',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/deployments',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/statefulsets',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/daemonsets',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/replicasets',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/pdbs',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/hpas',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/jobs',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/cronjobs',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/services',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/endpoints',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/endpointslices',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/network-policies',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/ingresses',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/ingress-classes',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/pvcs',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/pvs',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/volume-snapshots',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/volume-snapshot-classes',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/volume-snapshot-contents',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/storage-classes',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/csi-drivers',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/csi-nodes',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/csi-storage-capacities',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/volume-attachments',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/resource-quotas',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/limit-ranges',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/configmaps',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/secrets',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/service-accounts',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/roles',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/role-bindings',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/cluster-roles',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/cluster-role-bindings',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/nodes',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/leases',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/events',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/crds',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/api-services',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/priority-classes',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/runtime-classes',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/validating-webhooks',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/mutating-webhooks',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/validating-admission-policies',
        component: '@/pages/cluster/redirect',
      },
      {
        path: '/cluster/validating-admission-policy-bindings',
        component: '@/pages/cluster/redirect',
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
        component: '@/pages/k8s/$clusterId',
        hideInMenu: true,
      },
      // ---- 运维工具 ----
      {
        path: '/k8s/:clusterId/log-workbench',
        component: '@/pages/k8s/$clusterId/log-workbench',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/manifest-apply',
        component: '@/pages/k8s/$clusterId/manifest-apply',
        hideInMenu: true,
      },
      // ---- 工作负载 ----
      {
        path: '/k8s/:clusterId/pods',
        component: '@/pages/k8s/$clusterId/pods',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/podmetrics',
        component: '@/pages/k8s/$clusterId/podmetrics',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/workloads',
        component: '@/pages/k8s/$clusterId/workloads',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/deployments',
        component: '@/pages/k8s/$clusterId/deployments',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/statefulsets',
        component: '@/pages/k8s/$clusterId/statefulsets',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/daemonsets',
        component: '@/pages/k8s/$clusterId/daemonsets',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/replicasets',
        component: '@/pages/k8s/$clusterId/replicasets',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/pdbs',
        component: '@/pages/k8s/$clusterId/pdbs',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/hpas',
        component: '@/pages/k8s/$clusterId/hpas',
        hideInMenu: true,
      },
      // ---- 作业 ----
      {
        path: '/k8s/:clusterId/jobs',
        component: '@/pages/k8s/$clusterId/jobs',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/cronjobs',
        component: '@/pages/k8s/$clusterId/cronjobs',
        hideInMenu: true,
      },
      // ---- 网络资源 ----
      {
        path: '/k8s/:clusterId/services',
        component: '@/pages/k8s/$clusterId/services',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/endpoints',
        component: '@/pages/k8s/$clusterId/endpoints',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/endpointslices',
        component: '@/pages/k8s/$clusterId/endpointslices',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/network-policies',
        component: '@/pages/k8s/$clusterId/network-policies',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/ingresses',
        component: '@/pages/k8s/$clusterId/ingresses',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/ingress-classes',
        component: '@/pages/k8s/$clusterId/ingressclasses',
        hideInMenu: true,
      },
      // ---- 数据存储 ----
      {
        path: '/k8s/:clusterId/pvcs',
        component: '@/pages/k8s/$clusterId/pvc',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/pvs',
        component: '@/pages/k8s/$clusterId/pv',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/volume-snapshots',
        component: '@/pages/k8s/$clusterId/volumesnapshots',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/volume-snapshot-classes',
        component: '@/pages/k8s/$clusterId/volumesnapshotclasses',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/volume-snapshot-contents',
        component: '@/pages/k8s/$clusterId/volumesnapshotcontents',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/storage-classes',
        component: '@/pages/k8s/$clusterId/storage-classes',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/csi-drivers',
        component: '@/pages/k8s/$clusterId/csidrivers',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/csi-nodes',
        component: '@/pages/k8s/$clusterId/csinodes',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/csi-storage-capacities',
        component: '@/pages/k8s/$clusterId/csistoragecapacities',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/volume-attachments',
        component: '@/pages/k8s/$clusterId/volumeattachments',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/resource-quotas',
        component: '@/pages/k8s/$clusterId/resource-quotas',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/limit-ranges',
        component: '@/pages/k8s/$clusterId/limitranges',
        hideInMenu: true,
      },
      // ---- 配置文件 ----
      {
        path: '/k8s/:clusterId/configmaps',
        component: '@/pages/k8s/$clusterId/configmaps',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/secrets',
        component: '@/pages/k8s/$clusterId/secrets',
        hideInMenu: true,
      },
      // ---- 访问控制 ----
      {
        path: '/k8s/:clusterId/service-accounts',
        component: '@/pages/k8s/$clusterId/service-accounts',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/roles',
        component: '@/pages/k8s/$clusterId/roles',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/role-bindings',
        component: '@/pages/k8s/$clusterId/role-bindings',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/cluster-roles',
        component: '@/pages/k8s/$clusterId/clusterroles',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/cluster-role-bindings',
        component: '@/pages/k8s/$clusterId/clusterrolebindings',
        hideInMenu: true,
      },
      // ---- 集群资源 ----
      {
        path: '/k8s/:clusterId/namespaces',
        component: '@/pages/k8s/$clusterId/namespaces',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/nodes',
        component: '@/pages/k8s/$clusterId/nodes',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/leases',
        component: '@/pages/k8s/$clusterId/leases',
        hideInMenu: true,
      },
      // ---- 事件 ----
      {
        path: '/k8s/:clusterId/events',
        component: '@/pages/k8s/$clusterId/events',
        hideInMenu: true,
      },
      // ---- 扩展治理 ----
      {
        path: '/k8s/:clusterId/crds',
        component: '@/pages/k8s/$clusterId/crds',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/api-services',
        component: '@/pages/k8s/$clusterId/apiservices',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/priority-classes',
        component: '@/pages/k8s/$clusterId/priorityclasses',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/runtime-classes',
        component: '@/pages/k8s/$clusterId/runtimeclasses',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/validating-webhooks',
        component: '@/pages/k8s/$clusterId/validatingwebhooks',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/mutating-webhooks',
        component: '@/pages/k8s/$clusterId/mutatingwebhooks',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/validating-admission-policies',
        component: '@/pages/k8s/$clusterId/validatingadmissionpolicies',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/validating-admission-policy-bindings',
        component: '@/pages/k8s/$clusterId/validatingadmissionpolicybindings',
        hideInMenu: true,
      },
      // ---- 治理分析 ----
      {
        path: '/k8s/:clusterId/permission-audits',
        component: '@/pages/k8s/$clusterId/permission-audits',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/topology',
        component: '@/pages/k8s/$clusterId/topology',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/helm-releases',
        component: '@/pages/k8s/$clusterId/helm-releases',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/helm-repos',
        component: '@/pages/k8s/$clusterId/helm-repos',
        hideInMenu: true,
      },
      {
        path: '/k8s/:clusterId/resource-metrics',
        component: '@/pages/k8s/$clusterId/resource-metrics',
        hideInMenu: true,
      },
    ],
  },
  { path: '/403', component: '@/pages/403' },
  { path: '/*', component: '@/pages/404' },
]

export default routes
