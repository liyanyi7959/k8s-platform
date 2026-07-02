import {
  clusterContextSchema,
  clusterDashboardSchema,
  type ClusterContext,
  type ClusterDashboard,
} from '@/types/clusterContext'

const delay = (ms: number) =>
  new Promise<void>((resolve) => {
    setTimeout(resolve, ms)
  })

export const MOCK_CLUSTERS: ClusterContext[] = clusterContextSchema.array().parse([
  {
    id: 'local',
    name: 'local',
    version: 'v1.29.0',
    status: 'healthy',
    nodeCount: 2,
  },
  {
    id: 'my-k8s',
    name: 'my-k8s',
    version: 'v1.29.2',
    status: 'healthy',
    nodeCount: 3,
  },
  {
    id: 'prod-us',
    name: 'prod-us',
    version: 'v1.28.7',
    status: 'unhealthy',
    nodeCount: 12,
  },
])

export const findMockClusterById = (clusterId?: string | null) => {
  if (!clusterId) {
    return null
  }

  return MOCK_CLUSTERS.find((cluster) => cluster.id === clusterId) ?? null
}

export async function fetchClusterOptions(): Promise<ClusterContext[]> {
  await delay(500)
  return clusterContextSchema.array().parse(MOCK_CLUSTERS)
}

export async function fetchClusterDashboard(clusterId: string): Promise<ClusterDashboard> {
  await delay(500)

  const cluster = findMockClusterById(clusterId) ?? MOCK_CLUSTERS[0]
  const isHealthy = cluster?.status === 'healthy'
  const nodesReady = cluster?.id === 'prod-us' ? '10/12' : `${cluster?.nodeCount ?? 0}/${cluster?.nodeCount ?? 0}`
  const pendingPods = cluster?.id === 'prod-us' ? 6 : 0
  const alerts = cluster?.id === 'prod-us' ? 4 : 0
  const abnormalPods =
    cluster?.id === 'prod-us'
      ? [
          {
            namespace: 'payments',
            name: 'checkout-api-67dd9c7cfd-2nk8x',
            reason: 'CrashLoopBackOff',
            age: '18m',
            level: 'danger' as const,
          },
          {
            namespace: 'observability',
            name: 'otel-collector-6df8f7f98f-vskpx',
            reason: 'ImagePullBackOff',
            age: '41m',
            level: 'warning' as const,
          },
        ]
      : []

  const trend = Array.from({ length: 12 }, (_, index) => {
    const hour = `${String(index * 2).padStart(2, '0')}:00`
    const cpuBase = cluster?.id === 'prod-us' ? 48 : 24
    const memoryBase = cluster?.id === 'prod-us' ? 62 : 31

    return [
      { time: hour, type: 'CPU' as const, value: cpuBase + ((index * 7) % 18) },
      { time: hour, type: 'Memory' as const, value: memoryBase + ((index * 5) % 16) },
    ]
  }).flat()

  return clusterDashboardSchema.parse({
    clusterId,
    overview: {
      healthLabel: isHealthy ? '优秀' : '危险',
      healthTone: isHealthy ? 'success' : 'danger',
      nodesReady,
      pendingPods,
      alerts,
    },
    metrics: [
      {
        key: 'node-availability',
        label: '节点可用性',
        value: isHealthy ? '100%' : '83%',
        hint: 'Ready 节点占比',
        tone: isHealthy ? 'success' : 'warning',
      },
      {
        key: 'pod-running',
        label: 'Pod 运行率',
        value: isHealthy ? '100%' : '94%',
        hint: 'Running Pods 占比',
        tone: isHealthy ? 'success' : 'warning',
      },
      {
        key: 'resource-pressure',
        label: '资源压力',
        value: isHealthy ? '0%' : '37%',
        hint: 'CPU / Memory 综合压力',
        tone: isHealthy ? 'success' : 'warning',
      },
      {
        key: 'warning-events',
        label: 'Warning 事件',
        value: String(alerts),
        hint: '最近 24h Warning',
        tone: alerts > 0 ? 'danger' : 'success',
      },
    ],
    trend,
    abnormalPods,
  })
}
