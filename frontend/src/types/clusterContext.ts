import { z } from 'zod'

export const clusterStatusSchema = z.enum(['healthy', 'unhealthy'])

export const clusterContextSchema = z.object({
  id: z.string(),
  name: z.string(),
  version: z.string(),
  status: clusterStatusSchema,
  nodeCount: z.number().int().nonnegative(),
})

export const clusterDashboardMetricSchema = z.object({
  key: z.string(),
  label: z.string(),
  value: z.string(),
  hint: z.string(),
  tone: z.enum(['success', 'warning', 'danger', 'info']),
})

export const clusterTrendPointSchema = z.object({
  time: z.string(),
  type: z.enum(['CPU', 'Memory']),
  value: z.number(),
})

export const abnormalPodSchema = z.object({
  namespace: z.string(),
  name: z.string(),
  reason: z.string(),
  age: z.string(),
  level: z.enum(['warning', 'danger']),
})

export const clusterDashboardSchema = z.object({
  clusterId: z.string(),
  overview: z.object({
    healthLabel: z.string(),
    healthTone: z.enum(['success', 'warning', 'danger']),
    nodesReady: z.string(),
    pendingPods: z.number().int().nonnegative(),
    alerts: z.number().int().nonnegative(),
  }),
  metrics: z.array(clusterDashboardMetricSchema),
  trend: z.array(clusterTrendPointSchema),
  abnormalPods: z.array(abnormalPodSchema),
})

export type ClusterStatus = z.infer<typeof clusterStatusSchema>
export type ClusterContext = z.infer<typeof clusterContextSchema>
export type ClusterDashboardMetric = z.infer<typeof clusterDashboardMetricSchema>
export type ClusterTrendPoint = z.infer<typeof clusterTrendPointSchema>
export type AbnormalPod = z.infer<typeof abnormalPodSchema>
export type ClusterDashboard = z.infer<typeof clusterDashboardSchema>
