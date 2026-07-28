import { z } from 'zod'

const clusterNameSchema = z
  .string()
  .trim()
  .min(1, '集群名称不能为空')
  .max(120, '集群名称最长 120 字符')
  .regex(
    /^[a-zA-Z0-9][a-zA-Z0-9._-]*$/,
    '集群名称需以字母或数字开头，仅可包含字母、数字、点、横线和下划线',
  )

/** 集群导入表单校验 */
export const clusterImportSchema = z.object({
  name: clusterNameSchema,
  kubeconfig: z
    .string()
    .min(1, 'kubeconfig 不能为空')
    .max(1024 * 1024, 'kubeconfig 文件过大'),
  description: z.string().max(500, '备注最长 500 字符').optional(),
})

export type ClusterImportInput = z.infer<typeof clusterImportSchema>

/** 集群编辑表单校验 */
export const clusterEditSchema = z.object({
  name: clusterNameSchema,
  description: z.string().max(500, '备注最长 500 字符').optional(),
  kubeconfig: z.string().optional(),
})

export type ClusterEditInput = z.infer<typeof clusterEditSchema>
