import { z } from 'zod'

export const systemSettingsSchema = z.object({
  siteName: z.string().min(1, '站点名称不能为空').max(64, '站点名称最长 64 字符'),
  sessionTimeout: z.number().min(300, '会话超时最少 300 秒').max(86400, '会话超时最多 86400 秒').default(3600),
  maxLoginAttempts: z.number().min(3, '最大登录尝试次数最少 3 次').max(10, '最大登录尝试次数最多 10 次').default(5),
  passwordExpirationDays: z.number().min(30, '密码过期天数最少 30 天').max(365, '密码过期天数最多 365 天').default(90),
  enableAuditLog: z.boolean().default(true),
  enableTwoFactorAuth: z.boolean().default(false),
})

export type SystemSettingsInput = z.infer<typeof systemSettingsSchema>
