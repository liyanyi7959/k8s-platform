import { z } from 'zod'

/** 告警规则创建校验 */
export const alertRuleCreateSchema = z.object({
  name: z
    .string()
    .min(1, '告警规则名称不能为空')
    .max(128, '告警规则名称最长 128 字符'),
  clusterId: z.number().min(1, '请选择集群'),
  namespace: z.string().optional(),
  metricType: z.enum(['cpu', 'memory', 'disk', 'network', 'pod_restart', 'custom'], {
    required_error: '请选择指标类型',
  }),
  condition: z.enum(['gt', 'lt', 'eq', 'gte', 'lte'], {
    required_error: '请选择比较条件',
  }),
  threshold: z.number({ required_error: '请输入阈值' }),
  duration: z
    .number()
    .min(60, '持续时间最少 60 秒')
    .max(86400, '持续时间最多 86400 秒')
    .default(300),
  severity: z.enum(['info', 'warning', 'critical'], {
    required_error: '请选择告警级别',
  }),
  notificationChannels: z
    .array(z.enum(['email', 'webhook', 'sms']))
    .min(1, '至少选择一个通知渠道'),
  enabled: z.boolean().default(true),
  description: z.string().max(500, '描述最长 500 字符').optional(),
})

export type AlertRuleCreateInput = z.infer<typeof alertRuleCreateSchema>

/** 告警规则编辑校验 */
export const alertRuleEditSchema = alertRuleCreateSchema.partial().extend({
  id: z.number().min(1, '告警规则 ID 不能为空'),
})

export type AlertRuleEditInput = z.infer<typeof alertRuleEditSchema>

/** 监控查询参数校验 */
export const monitorQuerySchema = z.object({
  clusterId: z.number().min(1, '请选择集群'),
  namespace: z.string().optional(),
  podName: z.string().optional(),
  metricType: z.enum(['cpu', 'memory', 'disk', 'network']),
  startTime: z.string().min(1, '请选择开始时间'),
  endTime: z.string().min(1, '请选择结束时间'),
  step: z.number().min(15, '步长最少 15 秒').max(3600, '步长最多 3600 秒').default(60),
})

export type MonitorQueryInput = z.infer<typeof monitorQuerySchema>
