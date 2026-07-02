import { history, useModel, type RequestConfig } from '@umijs/max'
import { useEffect } from 'react'
import {
  App as AntdApp,
  Avatar,
  Badge,
  ConfigProvider,
  Dropdown,
  Radio,
  Select,
  Space,
  message,
  type MenuProps,
} from 'antd'
import {
  AlertOutlined,
  ApartmentOutlined,
  ArrowLeftOutlined,
  AuditOutlined,
  CloudServerOutlined,
  ClusterOutlined,
  DashboardOutlined,
  DownOutlined,
  FileSearchOutlined,
  HddOutlined,
  LogoutOutlined,
  NodeIndexOutlined,
  PlusOutlined,
  RobotOutlined,
  RocketOutlined,
  SettingOutlined,
  UnorderedListOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { getCurrentUser, logout as requestLogout } from '@/services/auth'
import { listClusters } from '@/services/clusters'
import type { User } from '@/types'
import type { Cluster as ModelCluster } from '@/models/cluster'
import defaultSettings from '../config/defaultSettings'

// ============================================================
// React Query 全局配置
// ============================================================
export const reactQuery = {
  queryClient: {
    defaultOptions: {
      queries: {
        retry: 1,
        refetchOnWindowFocus: false,
        staleTime: 30_000,
        gcTime: 5 * 60_000,
        refetchInterval: false,
      },
      mutations: {
        retry: 0,
      },
    },
  },
}

// ============================================================
// Antd Static Holder (用于 message/notification 等静态方法)
// ============================================================
let staticHolderConfigured = false

const ensureStaticHolder = () => {
  if (staticHolderConfigured) return
  ConfigProvider.config({
    holderRender: (children) => <AntdApp>{children}</AntdApp>,
  })
  staticHolderConfigured = true
}

export const rootContainer = (container: React.ReactNode) => {
  ensureStaticHolder()
  return <AntdApp>{container}</AntdApp>
}

// ============================================================
// 初始状态
// ============================================================
export async function getInitialState(): Promise<{
  currentUser?: User
}> {
  const { location } = history

  if (location.pathname === '/login') {
    return {}
  }

  try {
    const currentUser = await getCurrentUser()
    if (!currentUser?.id) {
      history.push('/login')
      return {}
    }
    return { currentUser }
  } catch {
    history.push('/login')
    return {}
  }
}

// ============================================================
// 类型定义
// ============================================================
type MenuItem = {
  name?: React.ReactNode
  path?: string
  key?: string
  icon?: React.ReactNode
  children?: MenuItem[]
  component?: string
  redirect?: string
  className?: string
  [key: string]: any
}

// ============================================================
// 菜单 A：全局模式（currentCluster === null）
// 参照 frontend-old 的菜单结构，确保所有父级菜单可正常展开
// ============================================================
const buildGlobalMenuItems = (): MenuItem[] => [
  // ---- 仪表盘 ----
  {
    key: '/dashboard',
    path: '/dashboard',
    name: '仪表盘',
    icon: <DashboardOutlined />,
  },
  // ---- 集群管理（参照 frontend-old 的 K8S 管理分组，包含子菜单） ----
  {
    key: '/clusters',
    path: '/clusters',
    name: '集群管理',
    icon: <ClusterOutlined />,
    children: [
      { key: '/clusters', path: '/clusters', name: '集群列表', icon: <UnorderedListOutlined /> },
      { key: '/clusters/import', path: '/clusters/import', name: '集群导入', icon: <PlusOutlined /> },
      { key: '/topology', path: '/topology', name: '资源视图', icon: <ApartmentOutlined /> },
    ],
  },
  // ---- 自动化部署 ----
  {
    key: '/deploy',
    path: '/deploy/plans',
    name: '自动化部署',
    icon: <RocketOutlined />,
    children: [
      { key: '/deploy/plans', path: '/deploy/plans', name: '部署计划' },
      { key: '/deploy/servers', path: '/deploy/servers', name: '服务器管理' },
      { key: '/deploy/credentials', path: '/deploy/credentials', name: '凭据管理' },
      { key: '/deploy/config', path: '/deploy/config', name: '部署配置' },
    ],
  },
  // ---- 监控告警 ----
  {
    key: '/monitor',
    path: '/monitor/dashboard',
    name: '监控告警',
    icon: <AlertOutlined />,
    children: [
      { key: '/monitor/dashboard', path: '/monitor/dashboard', name: '监控仪表盘' },
      { key: '/monitor/alerts', path: '/monitor/alerts', name: '告警规则' },
      { key: '/monitor/events', path: '/monitor/events', name: '事件流' },
    ],
  },
  // ---- 日志分析 ----
  {
    key: '/logs',
    path: '/logs',
    name: '日志分析',
    icon: <FileSearchOutlined />,
  },
  // ---- AI 运维助手 ----
  {
    key: '/ai',
    path: '/ai/chat',
    name: 'AI 运维助手',
    icon: <RobotOutlined />,
    children: [
      { key: '/ai/chat', path: '/ai/chat', name: '对话式运维' },
      { key: '/ai/history', path: '/ai/history', name: '历史记录' },
      { key: '/ai/settings', path: '/ai/settings', name: '模型配置' },
    ],
  },
]

const buildAdminMenuItems = (): MenuItem[] => [
  {
    key: 'return-global',
    path: '/dashboard',
    name: (
      <span style={{ fontWeight: 600 }}>返回工作台</span>
    ),
    icon: <ArrowLeftOutlined />,
    className: 'app-menu-return-global',
  },
  {
    key: 'group-admin-access',
    path: '/config/users',
    name: '账号与权限',
    icon: <UserOutlined />,
    children: [
      { key: '/config/users', path: '/config/users', name: '用户管理' },
      { key: '/config/roles', path: '/config/roles', name: '角色管理' },
    ],
  },
  {
    key: 'group-admin-platform',
    path: '/config/yaml',
    name: '平台配置',
    icon: <SettingOutlined />,
    children: [
      { key: '/config/yaml', path: '/config/yaml', name: 'YAML 编辑' },
      { key: '/config/settings', path: '/config/settings', name: '系统设置' },
    ],
  },
  {
    key: 'group-admin-audit',
    path: '/config/audit-logs',
    name: '审计与追踪',
    icon: <AuditOutlined />,
    children: [
      { key: '/config/audit-logs', path: '/config/audit-logs', name: '审计日志' },
    ],
  },
]

// ============================================================
// 菜单 B：集群钻取模式（currentCluster 有值）
// K8s 资源目录树 — 参照 frontend-old 的 buildTree 完整结构
// ============================================================
const buildClusterMenuItems = (clusterId: string): MenuItem[] => [
  // ---- 特殊置顶项：返回全局概览 ----
  {
    key: 'return-global',
    path: '/dashboard',
    name: (
      <span style={{ fontWeight: 600 }}>返回全局概览</span>
    ),
    icon: <ArrowLeftOutlined />,
    className: 'app-menu-return-global',
  },
  // ---- 仪表盘 ----
  {
    key: 'group-dashboard',
    path: `/k8s/${clusterId}/dashboard`,
    name: '仪表盘',
    icon: <DashboardOutlined />,
    children: [
      { key: `/k8s/${clusterId}/dashboard`, path: `/k8s/${clusterId}/dashboard`, name: '概览' },
    ],
  },
  // ---- 运维工具 ----
  {
    key: 'group-operations',
    path: `/k8s/${clusterId}/log-workbench`,
    name: '运维工具',
    icon: <CloudServerOutlined />,
    children: [
      { key: `/k8s/${clusterId}/log-workbench`, path: `/k8s/${clusterId}/log-workbench`, name: '日志工作台' },
      { key: `/k8s/${clusterId}/manifest-apply`, path: `/k8s/${clusterId}/manifest-apply`, name: 'YAML 部署' },
    ],
  },
  // ---- 工作负载 ----
  {
    key: 'group-workloads',
    path: `/k8s/${clusterId}/pods`,
    name: '工作负载',
    icon: <ClusterOutlined />,
    children: [
      { key: `/k8s/${clusterId}/pods`, path: `/k8s/${clusterId}/pods`, name: 'Pods' },
      { key: `/k8s/${clusterId}/podmetrics`, path: `/k8s/${clusterId}/podmetrics`, name: 'PodMetrics' },
      { key: `/k8s/${clusterId}/deployments`, path: `/k8s/${clusterId}/deployments`, name: 'Deployments' },
      { key: `/k8s/${clusterId}/statefulsets`, path: `/k8s/${clusterId}/statefulsets`, name: 'StatefulSets' },
      { key: `/k8s/${clusterId}/daemonsets`, path: `/k8s/${clusterId}/daemonsets`, name: 'DaemonSets' },
      { key: `/k8s/${clusterId}/replicasets`, path: `/k8s/${clusterId}/replicasets`, name: 'ReplicaSets' },
      { key: `/k8s/${clusterId}/pdbs`, path: `/k8s/${clusterId}/pdbs`, name: 'PDBs' },
      { key: `/k8s/${clusterId}/hpas`, path: `/k8s/${clusterId}/hpas`, name: 'HPAs' },
    ],
  },
  // ---- 作业 ----
  {
    key: 'group-jobs',
    path: `/k8s/${clusterId}/jobs`,
    name: '作业',
    icon: <RocketOutlined />,
    children: [
      { key: `/k8s/${clusterId}/jobs`, path: `/k8s/${clusterId}/jobs`, name: 'Jobs' },
      { key: `/k8s/${clusterId}/cronjobs`, path: `/k8s/${clusterId}/cronjobs`, name: 'CronJobs' },
    ],
  },
  // ---- 网络资源 ----
  {
    key: 'group-network',
    path: `/k8s/${clusterId}/services`,
    name: '网络资源',
    icon: <NodeIndexOutlined />,
    children: [
      { key: `/k8s/${clusterId}/services`, path: `/k8s/${clusterId}/services`, name: 'Services' },
      { key: `/k8s/${clusterId}/endpoints`, path: `/k8s/${clusterId}/endpoints`, name: 'Endpoints' },
      { key: `/k8s/${clusterId}/endpointslices`, path: `/k8s/${clusterId}/endpointslices`, name: 'EndpointSlices' },
      { key: `/k8s/${clusterId}/network-policies`, path: `/k8s/${clusterId}/network-policies`, name: 'NetworkPolicies' },
      { key: `/k8s/${clusterId}/ingresses`, path: `/k8s/${clusterId}/ingresses`, name: 'Ingresses' },
      { key: `/k8s/${clusterId}/ingress-classes`, path: `/k8s/${clusterId}/ingress-classes`, name: 'IngressClasses' },
    ],
  },
  // ---- 数据存储 ----
  {
    key: 'group-storage',
    path: `/k8s/${clusterId}/pvcs`,
    name: '数据存储',
    icon: <HddOutlined />,
    children: [
      { key: `/k8s/${clusterId}/pvcs`, path: `/k8s/${clusterId}/pvcs`, name: 'PVCs' },
      { key: `/k8s/${clusterId}/pvs`, path: `/k8s/${clusterId}/pvs`, name: 'PVs' },
      { key: `/k8s/${clusterId}/volume-snapshots`, path: `/k8s/${clusterId}/volume-snapshots`, name: 'VolumeSnapshots' },
      { key: `/k8s/${clusterId}/volume-snapshot-classes`, path: `/k8s/${clusterId}/volume-snapshot-classes`, name: 'VolumeSnapshotClasses' },
      { key: `/k8s/${clusterId}/volume-snapshot-contents`, path: `/k8s/${clusterId}/volume-snapshot-contents`, name: 'VolumeSnapshotContents' },
      { key: `/k8s/${clusterId}/storage-classes`, path: `/k8s/${clusterId}/storage-classes`, name: 'StorageClasses' },
      { key: `/k8s/${clusterId}/csi-drivers`, path: `/k8s/${clusterId}/csi-drivers`, name: 'CSIDrivers' },
      { key: `/k8s/${clusterId}/csi-nodes`, path: `/k8s/${clusterId}/csi-nodes`, name: 'CSINodes' },
      { key: `/k8s/${clusterId}/csi-storage-capacities`, path: `/k8s/${clusterId}/csi-storage-capacities`, name: 'CSIStorageCapacities' },
      { key: `/k8s/${clusterId}/volume-attachments`, path: `/k8s/${clusterId}/volume-attachments`, name: 'VolumeAttachments' },
      { key: `/k8s/${clusterId}/resource-quotas`, path: `/k8s/${clusterId}/resource-quotas`, name: 'ResourceQuotas' },
      { key: `/k8s/${clusterId}/limit-ranges`, path: `/k8s/${clusterId}/limit-ranges`, name: 'LimitRanges' },
    ],
  },
  // ---- 配置文件 ----
  {
    key: 'group-config',
    path: `/k8s/${clusterId}/configmaps`,
    name: '配置文件',
    icon: <FileSearchOutlined />,
    children: [
      { key: `/k8s/${clusterId}/configmaps`, path: `/k8s/${clusterId}/configmaps`, name: 'ConfigMaps' },
      { key: `/k8s/${clusterId}/secrets`, path: `/k8s/${clusterId}/secrets`, name: 'Secrets' },
    ],
  },
  // ---- 访问控制 ----
  {
    key: 'group-auth',
    path: `/k8s/${clusterId}/service-accounts`,
    name: '访问控制',
    icon: <AuditOutlined />,
    children: [
      { key: `/k8s/${clusterId}/service-accounts`, path: `/k8s/${clusterId}/service-accounts`, name: 'ServiceAccounts' },
      { key: `/k8s/${clusterId}/roles`, path: `/k8s/${clusterId}/roles`, name: 'Roles' },
      { key: `/k8s/${clusterId}/role-bindings`, path: `/k8s/${clusterId}/role-bindings`, name: 'RoleBindings' },
      { key: `/k8s/${clusterId}/cluster-roles`, path: `/k8s/${clusterId}/cluster-roles`, name: 'ClusterRoles' },
      { key: `/k8s/${clusterId}/cluster-role-bindings`, path: `/k8s/${clusterId}/cluster-role-bindings`, name: 'ClusterRoleBindings' },
    ],
  },
  // ---- 集群资源 ----
  {
    key: 'group-cluster',
    path: `/k8s/${clusterId}/namespaces`,
    name: '集群资源',
    icon: <ApartmentOutlined />,
    children: [
      { key: `/k8s/${clusterId}/namespaces`, path: `/k8s/${clusterId}/namespaces`, name: 'Namespaces' },
      { key: `/k8s/${clusterId}/nodes`, path: `/k8s/${clusterId}/nodes`, name: 'Nodes' },
      { key: `/k8s/${clusterId}/leases`, path: `/k8s/${clusterId}/leases`, name: 'Leases' },
    ],
  },
  // ---- 事件 ----
  {
    key: 'group-events',
    path: `/k8s/${clusterId}/events`,
    name: '事件',
    icon: <AlertOutlined />,
    children: [
      { key: `/k8s/${clusterId}/events`, path: `/k8s/${clusterId}/events`, name: 'Events' },
    ],
  },
  // ---- 扩展治理 ----
  {
    key: 'group-extensions',
    path: `/k8s/${clusterId}/crds`,
    name: '扩展治理',
    icon: <SettingOutlined />,
    children: [
      { key: `/k8s/${clusterId}/crds`, path: `/k8s/${clusterId}/crds`, name: 'CRDs' },
      { key: `/k8s/${clusterId}/api-services`, path: `/k8s/${clusterId}/api-services`, name: 'APIServices' },
      { key: `/k8s/${clusterId}/priority-classes`, path: `/k8s/${clusterId}/priority-classes`, name: 'PriorityClasses' },
      { key: `/k8s/${clusterId}/runtime-classes`, path: `/k8s/${clusterId}/runtime-classes`, name: 'RuntimeClasses' },
      { key: `/k8s/${clusterId}/validating-webhooks`, path: `/k8s/${clusterId}/validating-webhooks`, name: 'ValidatingWebhooks' },
      { key: `/k8s/${clusterId}/mutating-webhooks`, path: `/k8s/${clusterId}/mutating-webhooks`, name: 'MutatingWebhooks' },
      { key: `/k8s/${clusterId}/validating-admission-policies`, path: `/k8s/${clusterId}/validating-admission-policies`, name: 'ValidatingAdmissionPolicies' },
      { key: `/k8s/${clusterId}/validating-admission-policy-bindings`, path: `/k8s/${clusterId}/validating-admission-policy-bindings`, name: 'ValidatingAdmissionPolicyBindings' },
    ],
  },
  // ---- 治理分析 ----
  {
    key: 'group-audit',
    path: `/k8s/${clusterId}/permission-audits`,
    name: '治理分析',
    icon: <ApartmentOutlined />,
    children: [
      { key: `/k8s/${clusterId}/permission-audits`, path: `/k8s/${clusterId}/permission-audits`, name: '权限分析' },
      { key: `/k8s/${clusterId}/topology`, path: `/k8s/${clusterId}/topology`, name: '资源关系图' },
    ],
  },
]

// ============================================================
// 根据当前 pathname 确定需要展开的菜单 key
// 用于全局模式（"集群管理"）和集群模式（K8s 资源组）
// ============================================================
const getGlobalOpenKeys = (pathname: string): string[] => {
  const openKeys: string[] = []
  if (pathname.startsWith('/clusters') || pathname.startsWith('/topology')) {
    openKeys.push('/clusters')
  }
  if (pathname.startsWith('/deploy')) {
    openKeys.push('/deploy')
  }
  if (pathname.startsWith('/monitor')) {
    openKeys.push('/monitor')
  }
  if (pathname.startsWith('/ai')) {
    openKeys.push('/ai')
  }
  return openKeys
}

const getAdminOpenKeys = (pathname: string): string[] => {
  if (pathname.startsWith('/config/users') || pathname.startsWith('/config/roles')) {
    return ['group-admin-access']
  }
  if (pathname.startsWith('/config/audit-logs')) {
    return ['group-admin-audit']
  }
  return ['group-admin-platform']
}

const getClusterOpenKeys = (pathname: string): string[] => {
  const resource = pathname.match(/\/k8s\/[^/]+\/([^/?]+)/)?.[1] || ''

  const dashboard = ['dashboard']
  const operations = ['log-workbench', 'manifest-apply']
  const workloads = ['pods', 'podmetrics', 'deployments', 'statefulsets', 'daemonsets', 'replicasets', 'pdbs', 'hpas']
  const jobs = ['jobs', 'cronjobs']
  const network = ['services', 'endpoints', 'endpointslices', 'network-policies', 'ingresses', 'ingress-classes']
  const storage = ['pvcs', 'pvs', 'volume-snapshots', 'volume-snapshot-classes', 'volume-snapshot-contents', 'storage-classes', 'csi-drivers', 'csi-nodes', 'csi-storage-capacities', 'volume-attachments', 'resource-quotas', 'limit-ranges']
  const config = ['configmaps', 'secrets']
  const auth = ['service-accounts', 'roles', 'role-bindings', 'cluster-roles', 'cluster-role-bindings']
  const cluster = ['namespaces', 'nodes', 'leases']
  const events = ['events']
  const extensions = ['crds', 'api-services', 'priority-classes', 'runtime-classes', 'validating-webhooks', 'mutating-webhooks', 'validating-admission-policies', 'validating-admission-policy-bindings']
  const audit = ['permission-audits', 'topology']

  if (dashboard.includes(resource)) return ['group-dashboard']
  if (operations.includes(resource)) return ['group-operations']
  if (workloads.includes(resource)) return ['group-workloads']
  if (jobs.includes(resource)) return ['group-jobs']
  if (network.includes(resource)) return ['group-network']
  if (storage.includes(resource)) return ['group-storage']
  if (config.includes(resource)) return ['group-config']
  if (auth.includes(resource)) return ['group-auth']
  if (cluster.includes(resource)) return ['group-cluster']
  if (events.includes(resource)) return ['group-events']
  if (extensions.includes(resource)) return ['group-extensions']
  if (audit.includes(resource)) return ['group-audit']
  return ['group-dashboard']
}

// ============================================================
// 辅助函数
// ============================================================
const getUserDisplayName = (user?: Partial<User> & { name?: string }) =>
  user?.nickname || user?.name || user?.username || 'Admin'

// ============================================================
// Layout 导出（ProLayout 配置）
// ============================================================
export const layout = ({ initialState, setInitialState }: any) => {
  const { currentCluster, setCurrentCluster, setClusterList } = useModel('cluster')

  // 使用真实后端 API 获取集群列表（替代 mock 数据）
  const { data: clusterListRes } = useQuery({
    queryKey: ['cluster-options'],
    queryFn: () => listClusters(),
  })

  // 将后端 Cluster 映射为 Model Cluster 格式，并同步到 Model
  useEffect(() => {
    const backendList = clusterListRes?.items
    if (backendList && backendList.length > 0) {
      const mapped: ModelCluster[] = backendList.map((c) => ({
        id: String(c.id),
        name: c.name,
        version: c.k8sVersion || '',
        status: c.status,
      }))
      setClusterList(mapped)
    }
  }, [clusterListRes, setClusterList])

  const pathname = history.location.pathname
  const currentClusterId = currentCluster?.id ?? null
  const isAdminMode = pathname.startsWith('/config')
  const isClusterMode = pathname.startsWith('/k8s/') && Boolean(currentClusterId)
  const currentUserName = getUserDisplayName(initialState?.currentUser)
  const currentUsername = initialState?.currentUser?.username || 'admin'

  // 集群列表（用于 Select 下拉）
  const clusters = clusterListRes?.items || []

  // 根据 currentCluster 决定使用哪套菜单
  const menuData = isAdminMode
    ? buildAdminMenuItems()
    : isClusterMode
      ? buildClusterMenuItems(currentClusterId!)
      : buildGlobalMenuItems()

  // 根据当前路径自动展开对应子菜单
  const defaultOpenKeys = isAdminMode
    ? getAdminOpenKeys(pathname)
    : isClusterMode
      ? getClusterOpenKeys(pathname)
      : getGlobalOpenKeys(pathname)

  // ---- 退出登录 ----
  const handleLogout = async () => {
    try {
      await requestLogout()
    } catch {
      // ignore
    } finally {
      localStorage.removeItem('token')
      setInitialState({ currentUser: undefined })
      history.push('/login')
    }
  }

  // ---- 用户菜单 ----
  const userMenuItems: MenuProps['items'] = [
    {
      key: 'identity',
      disabled: true,
      label: (
        <div className="app-sider-account-dropdown__identity">
          <strong>{currentUserName}</strong>
          <span>{currentUsername}</span>
        </div>
      ),
    },
    { type: 'divider' },
    { key: 'admin', label: '管理后台', icon: <UserOutlined /> },
    { type: 'divider' },
    { key: 'logout', label: '退出登录', icon: <LogoutOutlined />, danger: true },
  ]

  const handleUserMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (key === 'logout') { handleLogout(); return }
    if (key === 'admin') { history.push('/config/users'); return }
  }

  return {
    ...defaultSettings,

    // ========== 动态菜单数据 ==========
    menuDataRender: () => menuData,

    // ========== 菜单选中与展开控制 ==========
    menuProps: {
      selectedKeys: [pathname],
      defaultOpenKeys,
    },

    // ========== 菜单项渲染（处理点击跳转） ==========
    // 关键修复：父级菜单项有 path 时，通过 children 判断是否为 SubMenu
    // 有 children 的项交给 ProLayout 处理展开/折叠，不拦截点击事件
    menuItemRender: (item: MenuItem, dom: React.ReactNode) => {
      // 特殊处理：返回全局概览按钮
      if (item.key === 'return-global') {
        return (
          <a
            onClick={() => {
              setCurrentCluster(null)
              history.push('/dashboard')
            }}
          >
            {dom}
          </a>
        )
      }

      // 父级菜单项（有 children）—— 交给 ProLayout 控制展开/折叠
      if (item.children && item.children.length > 0) {
        return dom
      }

      // 叶子菜单项（无 children）—— 包装点击跳转
      if (!item.path) {
        return dom
      }

      return (
        <a
          onClick={() => {
            const targetPath = item.path!
            // 集群模式下，k8s 路由带上 cluster 参数
            if (currentClusterId && targetPath.startsWith('/k8s/')) {
              history.push(`${targetPath}?cluster=${currentClusterId}`)
            } else {
              history.push(targetPath)
            }
          }}
        >
          {dom}
        </a>
      )
    },

    // ========== 收起菜单项中告警中心带 Badge ==========
    subMenuItemRender: (item: MenuItem, dom: React.ReactNode) => {
      if (item.name === '监控告警') {
        return (
          <span className="app-menu-label-with-badge">
            {dom}
            <Badge color="#ef4444" />
          </span>
        )
      }
      return dom
    },

    avatarProps: false,
    onMenuHeaderClick: () => history.push('/dashboard'),

    // ========== 右侧内容区：集群选择器 + 时间范围 + 用户头像 ==========
    rightContentRender: () => (
      <Space size={12} className="app-layout-header-actions">
        {!isAdminMode ? (
          <Select
            allowClear
            value={currentCluster?.id}
            placeholder="选择集群..."
            style={{ width: 200 }}
            options={clusters.map((cluster) => ({
              value: String(cluster.id),
              label: (
                <Space>
                  <span>{cluster.name}</span>
                  <span style={{ color: '#999', fontSize: 12 }}>{cluster.k8sVersion}</span>
                </Space>
              ),
            }))}
            onChange={(clusterId) => {
              if (!clusterId) {
                setCurrentCluster(null)
                history.push('/dashboard')
                return
              }
              const cluster = clusters.find((item) => String(item.id) === clusterId)
              if (cluster) {
                setCurrentCluster({
                  id: String(cluster.id),
                  name: cluster.name,
                  version: cluster.k8sVersion || '',
                  status: cluster.status,
                })
                history.push(`/cluster/dashboard?cluster=${cluster.id}`)
              }
            }}
          />
        ) : null}
        <Radio.Group
          size="small"
          defaultValue={localStorage.getItem('aiops-time-range') || '24h'}
          optionType="button"
          buttonStyle="solid"
          onChange={(event) => {
            localStorage.setItem('aiops-time-range', event.target.value)
          }}
          options={[
            { label: '近1h', value: '1h' },
            { label: '近24h', value: '24h' },
            { label: '近7d', value: '7d' },
          ]}
        />
        <Dropdown
          overlayClassName="app-sider-account-dropdown"
          menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
          placement="bottomRight"
          trigger={['click']}
        >
          <button type="button" className="app-layout-user-trigger">
            <Avatar size={28} icon={<UserOutlined />} />
            <span className="app-layout-user-trigger__name">{currentUserName}</span>
            <DownOutlined className="app-layout-user-trigger__arrow" />
          </button>
        </Dropdown>
      </Space>
    ),

    // ========== 内容区域渲染 ==========
    childrenRender: (children: React.ReactNode) => (
      <div className="app-layout-main">
        <div className="app-layout-main__body">
          <div className="app-layout-content-frame">{children}</div>
        </div>
        <div className="app-layout-footer-frame">
          <footer className="app-layout-statusbar">
            <div className="app-layout-statusbar__group">
              <span className="app-layout-statusbar__dot" />
              <span>All Agents Healthy</span>
            </div>
            <div className="app-layout-statusbar__group app-layout-statusbar__group--right">
              <span>v3.2.1</span>
            </div>
          </footer>
        </div>
      </div>
    ),
    // ========== 请求配置 ==========
  }
}

// ============================================================
// 请求配置
// ============================================================
export const request: RequestConfig = {
  timeout: 30_000,
  errorConfig: {},
  requestInterceptors: [
    (config: any) => {
      const token = localStorage.getItem('token')
      if (token) {
        config.headers = {
          ...config.headers,
          Authorization: `Bearer ${token}`,
        }
      }
      return config
    },
  ],
  responseInterceptors: [
    (response: any) => {
      const { status, data: payload } = response

      if (status === 401) {
        localStorage.removeItem('token')
        history.push('/login')
        message.error('登录已过期，请重新登录')
        throw new Error('登录已过期，请重新登录')
      }

      const isApiEnvelope = payload && typeof payload === 'object' && 'code' in payload && 'message' in payload && 'data' in payload
      if (!isApiEnvelope) {
        return response
      }

      if (payload.code === 401) {
        localStorage.removeItem('token')
        history.push('/login')
        message.error(payload.message || '登录已过期，请重新登录')
        throw new Error(payload.message || '登录已过期，请重新登录')
      }

      if (payload.code !== 0) {
        throw new Error(payload.message || '请求失败')
      }

      response.data = payload.data
      return response
    },
  ],
}