import { z } from 'zod'

/** Namespace 创建校验 */
export const namespaceCreateSchema = z.object({
  name: z
    .string()
    .min(1, '命名空间名称不能为空')
    .max(63, '命名空间名称最长 63 字符')
    .regex(/^[a-z][a-z0-9\-]*$/, '命名空间名称必须以小写字母开头，只能包含小写字母、数字和横线'),
  labels: z.record(z.string()).optional(),
})

export type NamespaceCreateInput = z.infer<typeof namespaceCreateSchema>

/** ConfigMap 创建校验 */
export const configMapCreateSchema = z.object({
  name: z
    .string()
    .min(1, 'ConfigMap 名称不能为空')
    .max(253, 'ConfigMap 名称最长 253 字符')
    .regex(/^[a-z][a-z0-9\-]*$/, '名称必须以小写字母开头，只能包含小写字母、数字和横线'),
  namespace: z.string().min(1, '请选择命名空间'),
  data: z.record(z.string()).optional(),
  labels: z.record(z.string()).optional(),
})

export type ConfigMapCreateInput = z.infer<typeof configMapCreateSchema>

/** YAML 编辑器校验 */
export const yamlEditSchema = z.object({
  yaml: z.string().min(1, 'YAML 内容不能为空'),
})

export type YamlEditInput = z.infer<typeof yamlEditSchema>

/** 资源操作确认校验 */
export const resourceDeleteSchema = z.object({
  name: z.string().min(1, '资源名称不能为空'),
  namespace: z.string().optional(),
  confirmText: z.string().min(1, '请输入确认文本'),
})

export type ResourceDeleteInput = z.infer<typeof resourceDeleteSchema>
