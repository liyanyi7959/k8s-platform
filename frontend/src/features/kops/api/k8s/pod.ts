/**
 * Pod 资源 API
 */
import { request } from '@umijs/max'
import type { Pod, PodList, PodListParams } from '@/shared/types'
import { mapPod, extractMappedList } from './shared'

/** 获取 Pod 列表 */
export function listPods(
  clusterId: number,
  params: PodListParams,
  signal?: AbortSignal,
): Promise<PodList> {
  return request(`/api/v1/clusters/${clusterId}/pods`, { params, signal }).then(extractMappedList(mapPod))
}

/** 获取 Pod 详情 */
export function getPod(
  clusterId: number,
  namespace: string,
  name: string,
  signal?: AbortSignal,
): Promise<Pod> {
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}`, { signal })
}

/** 获取 Pod YAML */
export function getPodYaml(
  clusterId: number,
  namespace: string,
  name: string,
  signal?: AbortSignal,
): Promise<{ yaml: string }> {
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}/yaml`, { signal }).then((res: { text?: string; yaml?: string }) => ({
    yaml: res.yaml || res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
  }))
}

/** 删除 Pod */
export function deletePod(clusterId: number, namespace: string, name: string, force?: boolean): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}`, {
    method: 'DELETE',
    params: force ? { force: true, grace_period_seconds: 0 } : undefined,
  })
}

/** 获取 Pod 日志 */
export function getPodLogs(
  clusterId: number,
  namespace: string,
  name: string,
  options?: { tailLines?: number; container?: string; previous?: boolean; timestamps?: boolean },
): Promise<{ logs: string }> {
  const params: Record<string, any> = { tail_lines: options?.tailLines ?? 200 }
  if (options?.container) params.container = options.container
  if (options?.previous) params.previous = true
  if (options?.timestamps) params.timestamps = true
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}/logs`, { params }).then((res: { text?: string; logs?: string }) => ({
    logs: res.text || res.logs || '',
  }))
}

/** 获取 Pod 终端 WebSocket 地址 */
export function getPodTerminalUrl(clusterId: number, namespace: string, name: string, options?: { container?: string; command?: string[]; tty?: boolean }): Promise<{ url: string }> {
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}/exec-sessions`, {
    method: 'POST',
    data: {
      container: options?.container || undefined,
      command: options?.command?.length ? options.command : ['/bin/sh'],
      tty: options?.tty ?? true,
    },
  }).then((res: { session_id?: string; ws_url?: string; url?: string }) => ({
    url: res.ws_url || res.url || '',
  }))
}

/** Pod 巡检诊断 */
export function getPodInspection(clusterId: number, namespace: string, name: string, signal?: AbortSignal): Promise<{ text: string }> {
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}/inspection`, { signal }).then((res: any) => ({
    text: res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
  }))
}

/** 获取 Pod 关联事件 */
export function getPodEvents(clusterId: number, namespace: string, name: string, signal?: AbortSignal): Promise<any[]> {
  return request(`/api/v1/clusters/${clusterId}/events`, {
    params: { namespace, field_selector: `involvedObject.name=${name}` },
    signal,
  }).then((res: any) => {
    const list = res?.list || res?.items || (Array.isArray(res) ? res : [])
    return list
  })
}

/** 创建 WebSocket 日志流会话（follow 模式） */
export function createPodLogSession(
  clusterId: number,
  namespace: string,
  name: string,
  options?: { container?: string; tailLines?: number; follow?: boolean; previous?: boolean },
): Promise<{ sessionId: string; wsUrl: string }> {
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}/log-sessions`, {
    method: 'POST',
    data: {
      container: options?.container,
      tail_lines: options?.tailLines ?? 200,
      follow: options?.follow ?? true,
      previous: options?.previous ?? false,
    },
  }).then((res: { session_id?: string; ws_url?: string }) => ({
    sessionId: res.session_id || '',
    wsUrl: res.ws_url || '',
  }))
}

/** 获取 PodMetrics 列表（资源使用率） */
export function listPodMetrics(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<any[]> {
  return request(`/api/v1/clusters/${clusterId}/podmetrics`, {
    params: namespace ? { namespace } : undefined,
    signal,
  }).then((res: any) => {
    if (Array.isArray(res)) return res
    if (Array.isArray(res?.items)) return res.items
    if (Array.isArray(res?.list)) return res.list
    return []
  })
}

/** 获取 Pod 列表（兼容拓扑图页面调用签名） */
export function getPods(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<PodList> {
  return listPods(clusterId, { namespace: namespace || undefined }, signal)
}
