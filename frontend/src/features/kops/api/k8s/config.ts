/**
 * 配置资源 API：ConfigMap / Secret
 */
import { request } from '@umijs/max'
import type { ConfigMapList, SecretList } from '@/shared/types'
import { mapConfigMap, mapSecret, extractMappedList } from './shared'
import { applyYaml } from './generic'

// ==================== ConfigMap ====================

/** 获取 ConfigMap 列表 */
export function listConfigMaps(
  clusterId: number,
  params: { namespace: string },
  signal?: AbortSignal,
): Promise<ConfigMapList> {
  return request(`/api/v1/clusters/${clusterId}/configmaps`, { params, signal }).then(extractMappedList(mapConfigMap))
}

/** 删除 ConfigMap */
export function deleteConfigMap(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/configmaps/${namespace}/${name}`, {
    method: 'DELETE',
  })
}

/** 创建 ConfigMap */
export function createConfigMap(clusterId: number, namespace: string, data: any): Promise<any> {
  return applyYaml(
    clusterId,
    [
      'apiVersion: v1',
      'kind: ConfigMap',
      'metadata:',
      `  name: ${data.metadata?.name || data.name}`,
      `  namespace: ${namespace}`,
      'data:',
      ...Object.entries((data.data || {}) as Record<string, string>).map(([key, value]) => `  ${key}: ${JSON.stringify(value)}`),
    ].join('\n'),
  )
}

/** 更新 ConfigMap */
export function updateConfigMap(clusterId: number, namespace: string, name: string, data: any): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/configmaps/${namespace}/${name}`, {
    method: 'PATCH',
    data: {
      namespace,
      name,
      data: data.data,
      labels: data.labels,
    },
  })
}

/** 获取 ConfigMap 关联资源 */
export function getConfigMapRelated(clusterId: number, namespace: string, name: string, signal?: AbortSignal): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/configmaps/${namespace}/${name}/related`, { signal })
}

/** 获取 ConfigMap 列表（兼容拓扑图页面调用签名） */
export function getConfigMaps(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<ConfigMapList> {
  return listConfigMaps(clusterId, { namespace: namespace || '' }, signal)
}

// ==================== Secret ====================

/** 获取 Secret 列表 */
export function listSecrets(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<SecretList> {
  return request(`/api/v1/clusters/${clusterId}/secrets`, { params: { namespace }, signal }).then(extractMappedList(mapSecret))
}

/** 删除 Secret */
export function deleteSecret(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/secrets/${namespace}/${name}`, { method: 'DELETE' })
}

/** 创建 Secret */
export function createSecret(clusterId: number, namespace: string, data: any): Promise<any> {
  return applyYaml(
    clusterId,
    [
      'apiVersion: v1',
      'kind: Secret',
      'metadata:',
      `  name: ${data.metadata?.name || data.name}`,
      `  namespace: ${namespace}`,
      `type: ${data.type || 'Opaque'}`,
      'stringData:',
      ...Object.entries((data.data || {}) as Record<string, string>).map(([key, value]) => `  ${key}: ${JSON.stringify(value)}`),
    ].join('\n'),
  )
}

/** 更新 Secret */
export function updateSecret(clusterId: number, namespace: string, name: string, data: any): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/secrets/${namespace}/${name}`, {
    method: 'PATCH',
    data: {
      namespace,
      name,
      type: data.type,
      stringData: data.stringData,
      labels: data.labels,
    },
  })
}

/** 揭示 Secret 明文 */
export function getSecretReveal(clusterId: number, namespace: string, name: string, signal?: AbortSignal): Promise<{ text: string }> {
  return request(`/api/v1/clusters/${clusterId}/secrets/${namespace}/${name}/decoded-data`, { signal }).then((res: any) => ({
    text: res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
  }))
}

/** 获取 Secret 关联资源 */
export function getSecretRelated(clusterId: number, namespace: string, name: string, signal?: AbortSignal): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/secrets/${namespace}/${name}/related`, { signal })
}
