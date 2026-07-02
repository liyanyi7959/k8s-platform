/**
 * 骨架屏组件
 * 加载状态占位符
 */
import React from 'react'
import { Skeleton, Card } from 'antd'

interface SkeletonCardProps {
  rows?: number
  active?: boolean
}

export const SkeletonCard: React.FC<SkeletonCardProps> = ({ rows = 4, active = true }) => {
  return (
    <Card style={{ margin: 16 }}>
      <Skeleton active={active} paragraph={{ rows }} />
    </Card>
  )
}

export default SkeletonCard
