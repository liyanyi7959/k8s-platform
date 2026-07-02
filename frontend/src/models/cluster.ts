import { useCallback, useEffect, useState } from 'react'
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
 * 优先级：path /k8s/xxx > query ?cluster=xxx > path /cluster/xxx
 */
const getClusterIdFromUrl = (): string | null => {
  const { search, pathname } = history.location

  // K8s 资源页路由：/k8s/:clusterId/...
  const k8sMatch = pathname.match(/^\/k8s\/([^/]+)/)
  if (k8sMatch?.[1]) return k8sMatch[1]

  const queryCluster = new URLSearchParams(search).get('cluster')
  if (queryCluster) return queryCluster

  const match = pathname.match(/^\/cluster\/([^/]+)/)
  if (match?.[1]) return match[1]

  return null
}

/**
 * 同步当前集群 ID 到 URL query 参数
 * 注意：/k8s/:clusterId 路径已在 path 中携带集群 ID，无需追加 query
 */
const syncClusterQuery = (clusterId: string | null) => {
  const { pathname, search } = history.location

  // K8s 资源页路由自身携带 clusterId，跳过 query 同步避免冗余与循环
  if (pathname.startsWith('/k8s/')) return

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
  // 跟踪当前路径名，路由变化时触发集群状态恢复
  const [pathname, setPathname] = useState<string>(history.location.pathname)

  // 订阅路由变化，同步 pathname
  useEffect(() => {
    const unlisten = history.listen(({ location }) => {
      setPathname(location.pathname)
    })
    return unlisten
  }, [])

  // 根据 URL + clusterList 恢复/切换集群状态
  // 依赖 pathname 与 clusterList：任何路由跳转或列表加载完成都会重新校准
  useEffect(() => {
    const clusterId = getClusterIdFromUrl()

    // 非集群上下文（URL 中无 clusterId）→ 回到全局模式
    if (!clusterId) {
      if (currentCluster) setCurrentClusterState(null)
      setInitialized(true)
      return
    }

    // 已是目标集群，无需重复设置
    if (currentCluster?.id === clusterId) {
      setInitialized(true)
      return
    }

    // 从 clusterList 查找目标集群；列表未加载时等待后续触发
    const found = clusterList.find((c) => c.id === clusterId)
    if (found) {
      setCurrentClusterState({
        id: found.id,
        name: found.name,
        version: found.version,
        status: found.status,
      })
    }
    setInitialized(true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname, clusterList])

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