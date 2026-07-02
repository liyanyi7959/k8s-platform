import React, { useEffect, useState } from 'react'
import { Outlet, history, useModel } from '@umijs/max'
import { Spin, Result, Button } from 'antd'

/**
 * 集群钻取专用 Layout
 *
 * 强制校验 currentCluster 非空：
 * - 如果 URL 有 ?cluster=xxx 参数但模型未初始化 → 自动恢复
 * - 如果完全没有集群上下文 → 提示用户返回全局概览
 * - 正常情况 → 渲染子路由
 */
const ClusterLayout: React.FC = () => {
  const { currentCluster, setCurrentCluster, initialized, clusterList } = useModel('cluster')
  const [recovering, setRecovering] = useState(false)

  useEffect(() => {
    // 等待模型初始化完成
    if (!initialized) return

    // 如果已有集群上下文，直接放行
    if (currentCluster) {
      setRecovering(false)
      return
    }

    // 尝试从 URL 中恢复集群上下文
    const clusterId = new URLSearchParams(history.location.search).get('cluster')
    if (clusterId) {
      setRecovering(true)
      const cluster = clusterList.find((c) => c.id === clusterId)
      if (cluster) {
        setCurrentCluster({
          id: cluster.id,
          name: cluster.name,
          version: cluster.version,
          status: cluster.status,
        })
        setRecovering(false)
      } else {
        // 集群不存在，跳回全局
        history.replace('/dashboard')
      }
    }
    // 没有 cluster 参数也没有 currentCluster → 不自动跳转，显示错误提示
  }, [currentCluster, initialized, setCurrentCluster, clusterList])

  // 初始化中
  if (!initialized) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '60vh' }}>
        <Spin size="large" tip="正在初始化集群上下文..." />
      </div>
    )
  }

  // 正在恢复中
  if (recovering) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '60vh' }}>
        <Spin size="large" tip="正在恢复集群上下文..." />
      </div>
    )
  }

  // 没有集群上下文
  if (!currentCluster) {
    return (
      <Result
        status="warning"
        title="未选择集群"
        subTitle="请先在顶部导航栏选择一个集群，或返回全局概览。"
        extra={
          <Button type="primary" onClick={() => history.push('/dashboard')}>
            返回全局概览
          </Button>
        }
      />
    )
  }

  // 正常渲染子路由
  return <Outlet />
}

export default ClusterLayout