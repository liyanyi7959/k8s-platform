import { z } from 'zod'

/** 集群导入表单校验 */
export const clusterImportSchema = z.object({
  name: z
    .string()
    .min(1, '集群名称不能为空')
    .max(64, '集群名称最长 64 字符')
    .regex(/^[a-zA-Z0-9\-_]+$/, '集群名称只能包含字母、数字、横线和下划线'),
  kubeconfig: z
    .string()
    .min(1, 'kubeconfig 不能为空')
    .max(1024 * 1024, 'kubeconfig 文件过大'),
  description: z.string().max(200, '备注最长 200 字符').optional(),
})

export type ClusterImportInput = z.infer<typeof clusterImportSchema>

/** 集群编辑表单校验 */
export const clusterEditSchema = z.object({
  name: z
    .string()
    .min(1, '集群名称不能为空')
    .max(64, '集群名称最长 64 字符')
    .regex(/^[a-zA-Z0-9\-_]+$/, '集群名称只能包含字母、数字、横线和下划线'),
  description: z.string().max(200, '备注最长 200 字符').optional(),
  kubeconfig: z.string().optional(),
})

export type ClusterEditInput = z.infer<typeof clusterEditSchema>
