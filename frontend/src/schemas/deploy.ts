import { z } from 'zod'

/** 部署计划创建校验 */
export const deployPlanCreateSchema = z.object({
  name: z
    .string()
    .min(1, '部署计划名称不能为空')
    .max(128, '部署计划名称最长 128 字符'),
  description: z.string().max(500, '描述最长 500 字符').optional(),
  targetHosts: z
    .array(z.string().ip('请输入有效的 IP 地址'))
    .min(1, '至少选择一个目标主机'),
  deployPath: z
    .string()
    .min(1, '部署路径不能为空')
    .regex(/^\//, '部署路径必须以 / 开头'),
  commands: z
    .array(z.string().min(1, '命令不能为空'))
    .min(1, '至少添加一条部署命令'),
  credentialId: z.number().min(1, '请选择凭据'),
  environmentVariables: z.record(z.string()).optional(),
  timeout: z
    .number()
    .min(30, '超时时间最少 30 秒')
    .max(3600, '超时时间最多 3600 秒')
    .default(300),
  dryRun: z.boolean().default(false),
})

export type DeployPlanCreateInput = z.infer<typeof deployPlanCreateSchema>

/** 凭据基础 schema（不含 refine，用于派生编辑 schema） */
const credentialBaseSchema = z.object({
  name: z
    .string()
    .min(1, '凭据名称不能为空')
    .max(64, '凭据名称最长 64 字符'),
  type: z.enum(['ssh_password', 'ssh_key', 'kubeconfig'], {
    required_error: '请选择凭据类型',
  }),
  username: z.string().min(1, '用户名不能为空').optional(),
  password: z.string().min(1, '密码不能为空').optional(),
  privateKey: z.string().min(1, '私钥不能为空').optional(),
  passphrase: z.string().optional(),
  kubeconfig: z.string().optional(),
  description: z.string().max(200, '描述最长 200 字符').optional(),
})

/** 凭据创建校验 */
export const credentialCreateSchema = credentialBaseSchema.refine(
  (data) => {
    if (data.type === 'ssh_password') {
      return data.username && data.password
    }
    if (data.type === 'ssh_key') {
      return data.username && data.privateKey
    }
    if (data.type === 'kubeconfig') {
      return data.kubeconfig
    }
    return false
  },
  { message: '请填写完整的凭据信息' }
)

export type CredentialCreateInput = z.infer<typeof credentialCreateSchema>

/** 凭据编辑校验 */
export const credentialEditSchema = credentialBaseSchema.partial().extend({
  id: z.number().min(1, '凭据 ID 不能为空'),
})

export type CredentialEditInput = z.infer<typeof credentialEditSchema>
