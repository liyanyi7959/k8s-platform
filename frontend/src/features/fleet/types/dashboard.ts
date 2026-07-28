import { z } from 'zod'

export const dashboardTimeRangeSchema = z.enum(['1h', '24h', '7d'])

export const dashboardStatisticSchema = z.object({
  key: z.string(),
  title: z.string(),
  value: z.number(),
  unit: z.string().optional(),
  description: z.string().optional(),
})

export const dashboardOverviewSchema = z.object({
  timeRange: dashboardTimeRangeSchema,
  statistics: z.array(dashboardStatisticSchema).length(4),
  refreshedAt: z.string().datetime(),
})

export type DashboardTimeRange = z.infer<typeof dashboardTimeRangeSchema>
export type DashboardStatistic = z.infer<typeof dashboardStatisticSchema>
export type DashboardOverview = z.infer<typeof dashboardOverviewSchema>
