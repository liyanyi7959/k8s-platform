/**
 * 空状态组件
 * 统一的空数据展示
 */
import React from 'react'
import { Empty, Button } from 'antd'

interface EmptyStateProps {
  description?: string
  actionText?: string
  onAction?: () => void
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  description = '暂无数据',
  actionText,
  onAction,
}) => {
  return (
    <Empty
      description={description}
      style={{ padding: '60px 0' }}
    >
      {actionText && onAction && (
        <Button type="primary" onClick={onAction}>
          {actionText}
        </Button>
      )}
    </Empty>
  )
}
