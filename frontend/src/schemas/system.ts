import { z } from 'zod'

/** 用户创建校验 */
export const userCreateSchema = z.object({
  username: z
    .string()
    .min(3, '用户名最少 3 个字符')
    .max(32, '用户名最长 32 字符')
    .regex(/^[a-zA-Z][a-zA-Z0-9_]*$/, '用户名必须以字母开头，只能包含字母、数字和下划线'),
  password: z
    .string()
    .min(8, '密码最少 8 个字符')
    .max(64, '密码最长 64 字符')
    .regex(
      /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[!@#$%^&*])/,
      '密码必须包含大小写字母、数字和特殊字符'
    ),
  confirmPassword: z.string().min(1, '请确认密码'),
  email: z.string().email('请输入有效的邮箱地址').optional(),
  roleIds: z.array(z.number()).min(1, '至少选择一个角色'),
  enabled: z.boolean().default(true),
}).refine((data) => data.password === data.confirmPassword, {
  message: '两次输入的密码不一致',
  path: ['confirmPassword'],
})

export type UserCreateInput = z.infer<typeof userCreateSchema>

/** 用户编辑校验 */
export const userEditSchema = z.object({
  id: z.number().min(1, '用户 ID 不能为空'),
  email: z.string().email('请输入有效的邮箱地址').optional(),
  roleIds: z.array(z.number()).min(1, '至少选择一个角色'),
  enabled: z.boolean(),
  password: z.string().optional(),
  confirmPassword: z.string().optional(),
}).refine(
  (data) => {
    if (data.password) {
      return data.password === data.confirmPassword
    }
    return true
  },
  { message: '两次输入的密码不一致', path: ['confirmPassword'] }
)

export type UserEditInput = z.infer<typeof userEditSchema>

/** 角色创建校验 */
export const roleCreateSchema = z.object({
  name: z
    .string()
    .min(2, '角色名称最少 2 个字符')
    .max(64, '角色名称最长 64 字符'),
  code: z
    .string()
    .min(2, '角色编码最少 2 个字符')
    .max(32, '角色编码最长 32 字符')
    .regex(/^[a-z][a-z0-9_]*$/, '角色编码必须以小写字母开头，只能包含小写字母、数字和下划线'),
  description: z.string().max(200, '描述最长 200 字符').optional(),
  permissionIds: z.array(z.number()).optional(),
})

export type RoleCreateInput = z.infer<typeof roleCreateSchema>

/** 角色编辑校验 */
export const roleEditSchema = roleCreateSchema.partial().extend({
  id: z.number().min(1, '角色 ID 不能为空'),
})

export type RoleEditInput = z.infer<typeof roleEditSchema>

/** 登录表单校验 */
export const loginSchema = z.object({
  username: z.string().min(1, '请输入用户名'),
  password: z.string().min(1, '请输入密码'),
})

export type LoginInput = z.infer<typeof loginSchema>

/** 修改密码校验 */
export const changePasswordSchema = z.object({
  oldPassword: z.string().min(1, '请输入原密码'),
  newPassword: z
    .string()
    .min(8, '密码最少 8 个字符')
    .max(64, '密码最长 64 字符')
    .regex(
      /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[!@#$%^&*])/,
      '密码必须包含大小写字母、数字和特殊字符'
    ),
  confirmPassword: z.string().min(1, '请确认新密码'),
}).refine((data) => data.newPassword === data.confirmPassword, {
  message: '两次输入的密码不一致',
  path: ['confirmPassword'],
})

export type ChangePasswordInput = z.infer<typeof changePasswordSchema>

/** 系统设置校验 */
export const systemSettingsSchema = z.object({
  siteName: z.string().min(1, '站点名称不能为空').max(64, '站点名称最长 64 字符'),
  sessionTimeout: z
    .number()
    .min(300, '会话超时最少 300 秒')
    .max(86400, '会话超时最多 86400 秒')
    .default(3600),
  maxLoginAttempts: z
    .number()
    .min(3, '最大登录尝试次数最少 3 次')
    .max(10, '最大登录尝试次数最多 10 次')
    .default(5),
  passwordExpirationDays: z
    .number()
    .min(30, '密码过期天数最少 30 天')
    .max(365, '密码过期天数最多 365 天')
    .default(90),
  enableAuditLog: z.boolean().default(true),
  enableTwoFactorAuth: z.boolean().default(false),
})

export type SystemSettingsInput = z.infer<typeof systemSettingsSchema>
