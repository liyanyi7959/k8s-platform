import { useCallback, useEffect, useRef, useState } from 'react'
import { history } from '@umijs/max'

/**
 * 集群简要信息（Model 使用，与后端 Cluster 类型独立）
 * id 使用 string 以兼容 URL 参数传递
 */
export type Cluster = {
  id: string
  name: string
  version: string
  status: string
}

/**
 * 从 URL 中提取集群 ID
 * 优先级：query ?cluster=xxx > path /cluster/xxx
 */
const getClusterIdFromUrl = (): string | null => {
  const { search, pathname } = history.location

  const queryCluster = new URLSearchParams(search).get('cluster')
  if (queryCluster) return queryCluster

  const match = pathname.match(/^\/cluster\/([^/]+)/)
  if (match?.[1]) return match[1]

  return null
}

/**
 * 同步当前集群 ID 到 URL query 参数
 */
const syncClusterQuery = (clusterId: string | null) => {
  const { pathname, search } = history.location
  const params = new URLSearchParams(search)

  if (clusterId) {
    params.set('cluster', clusterId)
  } else {
    params.delete('cluster')
  }

  const nextSearch = params.toString()
  const nextUrl = nextSearch ? `${pathname}?${nextSearch}` : pathname
  if (`${pathname}${search ? `?${search}` : ''}` !== nextUrl) {
    history.replace(nextUrl)
  }
}

/**
 * 全局集群上下文 Model
 *
 * 核心逻辑：
 * - currentCluster === null → 全局概览模式，左侧菜单显示全局菜单
 * - currentCluster 有值 → 集群钻取模式，左侧菜单切换为 K8s 资源目录树
 * - 页面刷新后根据 URL 参数自动恢复集群状态（从 clusterList 中查找）
 * - clusterList 由 app.tsx 通过真实 API 填充并同步到 Model
 */
export default function useClusterModel() {
  const [currentCluster, setCurrentClusterState] = useState<Cluster | null>(null)
  const [clusterList, setClusterListState] = useState<Cluster[]>([])
  const [initialized, setInitialized] = useState(false)
  const clusterListRef = useRef<Cluster[]>([])

  // 保持 ref 与 state 同步，供 recovery 使用
  useEffect(() => {
    clusterListRef.current = clusterList
  }, [clusterList])

  // 初始化：从 URL 恢复集群状态（从 clusterList 中查找）
  useEffect(() => {
    const clusterId = getClusterIdFromUrl()
    if (clusterId) {
      // 先从当前 clusterList 查找，如果还没加载（首次渲染），等待后续 setClusterList 触发恢复
      const found = clusterListRef.current.find((c) => c.id === clusterId)
      if (found) {
        setCurrentClusterState({
          id: found.id,
          name: found.name,
          version: found.version,
          status: found.status,
        })
      }
    }
    setInitialized(true)
  }, [])

  // 当 clusterList 更新后，尝试恢复之前未恢复的集群状态
  useEffect(() => {
    const clusterId = getClusterIdFromUrl()
    if (clusterId && !currentCluster) {
      const found = clusterList.find((c) => c.id === clusterId)
      if (found) {
        setCurrentClusterState({
          id: found.id,
          name: found.name,
          version: found.version,
          status: found.status,
        })
      }
    }
  }, [clusterList, currentCluster])

  /**
   * 设置当前集群
   * - 传入集群对象 → 进入集群钻取模式
   * - 传入 null → 返回全局概览模式
   */
  const setCurrentCluster = useCallback((cluster: Cluster | null) => {
    setCurrentClusterState(cluster)
    syncClusterQuery(cluster?.id ?? null)
  }, [])

  /**
   * 设置集群列表（由 app.tsx 调用，数据来自后端真实 API）
   */
  const setClusterList = useCallback((list: Cluster[]) => {
    setClusterListState(list)
  }, [])

  return {
    currentCluster,
    setCurrentCluster,
    clusterList,
    setClusterList,
    initialized,
  }
}