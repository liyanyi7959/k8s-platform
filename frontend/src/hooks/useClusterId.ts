import { useParams } from '@umijs/max'

/** 获取当前集群 ID，如果 URL 中没有则默认为 1 */
export function useClusterId(): number {
  const { clusterId } = useParams<{ clusterId: string }>()
  return clusterId ? Number(clusterId) : 1
}
