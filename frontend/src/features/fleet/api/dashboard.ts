import {
  dashboardOverviewSchema,
  type DashboardOverview,
  type DashboardTimeRange,
} from '@/features/fleet/types/dashboard'

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

export async function fetchDashboardOverview(
  timeRange: DashboardTimeRange = '24h',
): Promise<DashboardOverview> {
  await wait(500)

  return dashboardOverviewSchema.parse({
    timeRange,
    refreshedAt: new Date().toISOString(),
    statistics: [
      { key: 'clusters', title: '集群总数', value: 0, unit: '个', description: '当前纳管集群' },
      { key: 'alerts', title: '活跃告警', value: 0, unit: '条', description: '待处理事件' },
      { key: 'automation', title: '自动化任务', value: 0, unit: '次', description: '近周期执行' },
      { key: 'rca', title: '根因定位', value: 0, unit: '次', description: 'AI 分析记录' },
    ],
  })
}
