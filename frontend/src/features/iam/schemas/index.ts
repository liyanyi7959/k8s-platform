import { z } from 'zod'

const usernameSchema = z.string().min(3, '用户名最少 3 个字符').max(32, '用户名最长 32 字符').regex(/^[a-zA-Z][a-zA-Z0-9_]*$/, '用户名必须以字母开头，只能包含字母、数字和下划线')
const passwordSchema = z.string().min(8, '密码最少 8 个字符').max(64, '密码最长 64 字符').regex(/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[!@#$%^&*])/, '密码必须包含大小写字母、数字和特殊字符')
const emailSchema = z.union([z.string().email('请输入有效的邮箱地址'), z.literal(''), z.undefined()]).optional()
const roleCodeSchema = z.string().min(2, '角色编码最少 2 个字符').max(32, '角色编码最长 32 字符').regex(/^[a-z][a-z0-9_]*$/, '角色编码必须以小写字母开头，只能包含小写字母、数字和下划线')

export const userCreateSchema = z.object({
  username: usernameSchema,
  nickname: z.string().max(80, '昵称最长 80 字符').optional(),
  email: emailSchema,
  password: passwordSchema,
  confirmPassword: z.string().min(1, '请确认密码'),
  roleIds: z.array(z.number()).min(1, '至少选择一个角色'),
  enabled: z.boolean().default(true),
}).refine((data) => data.password === data.confirmPassword, { message: '两次输入的密码不一致', path: ['confirmPassword'] })

export type UserCreateInput = z.infer<typeof userCreateSchema>

export const userEditSchema = z.object({
  id: z.number().min(1, '用户 ID 不能为空'),
  nickname: z.string().max(80, '昵称最长 80 字符').optional(),
  email: emailSchema,
  roleIds: z.array(z.number()).min(1, '至少选择一个角色'),
  enabled: z.boolean(),
})

export type UserEditInput = z.infer<typeof userEditSchema>

export const resetPasswordSchema = z.object({
  newPassword: passwordSchema,
  confirmPassword: z.string().min(1, '请确认密码'),
}).refine((data) => data.newPassword === data.confirmPassword, { message: '两次输入的密码不一致', path: ['confirmPassword'] })

export type ResetPasswordInput = z.infer<typeof resetPasswordSchema>

export const roleCreateSchema = z.object({
  name: z.string().min(2, '角色名称最少 2 个字符').max(64, '角色名称最长 64 字符'),
  code: roleCodeSchema,
  description: z.string().max(200, '描述最长 200 字符').optional(),
  permissions: z.array(z.string()).default([]),
})

export type RoleCreateInput = z.infer<typeof roleCreateSchema>

export const roleEditSchema = z.object({
  id: z.number().min(1, '角色 ID 不能为空'),
  name: z.string().min(2, '角色名称最少 2 个字符').max(64, '角色名称最长 64 字符'),
  description: z.string().max(200, '描述最长 200 字符').optional(),
  permissions: z.array(z.string()).default([]),
})

export type RoleEditInput = z.infer<typeof roleEditSchema>

export const loginSchema = z.object({ username: z.string().min(1, '请输入用户名'), password: z.string().min(1, '请输入密码') })
export type LoginInput = z.infer<typeof loginSchema>

export const changePasswordSchema = z.object({
  oldPassword: z.string().min(1, '请输入原密码'),
  newPassword: passwordSchema,
  confirmPassword: z.string().min(1, '请确认新密码'),
}).refine((data) => data.newPassword === data.confirmPassword, { message: '两次输入的密码不一致', path: ['confirmPassword'] })

export type ChangePasswordInput = z.infer<typeof changePasswordSchema>
