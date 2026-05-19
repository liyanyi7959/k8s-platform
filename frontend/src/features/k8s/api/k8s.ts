// k8s.ts — Barrel re-export，保持向后兼容。
// 实际实现已按资源类型拆分到同目录下的独立模块。
//
// 新代码建议直接从子模块导入，例如：
//   import { listNodes } from '@/features/k8s/api/node'

export * from './namespace'
export * from './node'
export * from './permissionAudit'
export * from './pod'
export * from './workload'
export * from './network'
export * from './coordination'
export * from './config'
export * from './storage'
export * from './batch'
export * from './policy'
export * from './extensions'
export * from './governance'
export * from './relationships'
