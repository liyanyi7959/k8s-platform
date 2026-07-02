/**
 * Pod 状态标签组件
 */
import React from 'react'
import { Tag, Tooltip } from 'antd'

interface PodStatusTagProps {
  status: string
  ready?: string
  onClick?: () => void
}

const statusColorMap: Record<string, string> = {
  Running: 'success',
  Pending: 'warning',
  Failed: 'error',
  Succeeded: 'default',
  Unknown: 'default',
}

const statusTextMap: Record<string, string> = {
  Running: '运行中',
  Pending: '等待中',
  Failed: '失败',
  Succeeded: '成功',
  Unknown: '未知',
}

export const PodStatusTag: React.FC<PodStatusTagProps> = ({ status, ready, onClick }) => {
  const color = statusColorMap[status] || 'default'
  const text = statusTextMap[status] || status

  return (
    <Tooltip title={ready ? `就绪: ${ready}` : undefined}>
      <Tag color={color} onClick={onClick} style={{ cursor: onClick ? 'pointer' : 'default' }}>
        {text}
        {ready && ` (${ready})`}
      </Tag>
    </Tooltip>
  )
}
