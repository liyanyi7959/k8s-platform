/**
 * K8s 资源 API 统一出口
 * 按资源域拆分为多个模块，此处 barrel 重导出，保持 `@/services/k8s` 导入路径不变
 */
export * from './shared'
export * from './generic'
export * from './pod'
export * from './workload'
export * from './network'
export * from './storage'
export * from './config'
export * from './batch'
export * from './rbac'
export * from './cluster'
