/**
 * 网络资源 API：Service / Ingress / NetworkPolicy
 */
import { request } from '@umijs/max'
import type {
  K8sServiceList,
  IngressList,
  NetworkPolicyList,
} from '@/features/kops/types'
import { mapService, mapIngress, mapNetworkPolicy, extractMappedList } from './shared'

/** 解析可能为 JSON 字符串的 selector */
function parseSelector(selector: unknown): Record<string, string> | undefined {
  if (!selector) return undefined
  if (typeof selector === 'string') {
    try {
      return JSON.parse(selector)
    } catch {
      return undefined
    }
  }
  return selector as Record<string, string>
}

// ==================== Service ====================

/** 获取 Service 列表 */
export function listServices(
  clusterId: number,
  params: { namespace: string },
  signal?: AbortSignal,
): Promise<K8sServiceList> {
  return request(`/api/v1/clusters/${clusterId}/services`, { params, signal }).then(extractMappedList(mapService))
}

/** 删除 Service */
export function deleteService(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/services/${namespace}/${name}`, {
    method: 'DELETE',
  })
}

/** 创建 Service */
export function createService(clusterId: number, namespace: string, data: any): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/services`, {
    method: 'POST',
    data: {
      namespace,
      name: data.name,
      type: data.type,
      selector: parseSelector(data.selector),
      ports: [
        {
          name: data.portName,
          port: Number(data.port || 80),
          target_port: Number(data.targetPort || data.port || 80),
          protocol: data.protocol || 'TCP',
        },
      ],
    },
  })
}

/** 更新 Service */
export function updateService(clusterId: number, namespace: string, name: string, data: any): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/services/${namespace}/${name}`, {
    method: 'PATCH',
    data: {
      namespace,
      name,
      type: data.type,
      selector: parseSelector(data.selector),
    },
  })
}

/** 获取 Service 列表（兼容拓扑图页面调用签名） */
export function getK8sServices(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<K8sServiceList> {
  return listServices(clusterId, { namespace: namespace || '' }, signal)
}

// ==================== Ingress ====================

/** 获取 Ingress 列表 */
export function listIngresses(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<IngressList> {
  return request(`/api/v1/clusters/${clusterId}/ingresses`, { params: { namespace }, signal }).then(extractMappedList(mapIngress))
}

/** 删除 Ingress */
export function deleteIngress(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/ingresses/${namespace}/${name}`, { method: 'DELETE' })
}

/** 创建 Ingress */
export function createIngress(clusterId: number, namespace: string, data: any): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/ingresses`, {
    method: 'POST',
    data: {
      namespace,
      name: data.name,
      ingress_class: data.ingressClassName,
      rules: [
        {
          host: data.host,
          paths: [
            {
              path: data.path || '/',
              path_type: 'Prefix',
              service_name: data.serviceName,
              service_port: Number(data.servicePort || 80),
            },
          ],
        },
      ],
    },
  })
}

/** 更新 Ingress */
export function updateIngress(clusterId: number, namespace: string, name: string, data: any): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/ingresses/${namespace}/${name}`, {
    method: 'PATCH',
    data: {
      namespace,
      name,
      ingressClassName: data.ingressClassName,
      annotations: data.annotations,
      labels: data.labels,
    },
  })
}

// ==================== NetworkPolicy ====================

/** 获取 NetworkPolicy 列表 */
export function listNetworkPolicies(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<NetworkPolicyList> {
  return request(`/api/v1/clusters/${clusterId}/networkpolicies`, { params: { namespace }, signal }).then(extractMappedList(mapNetworkPolicy))
}

/** 删除 NetworkPolicy */
export function deleteNetworkPolicy(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/networkpolicies/${namespace}/${name}`, { method: 'DELETE' })
}
