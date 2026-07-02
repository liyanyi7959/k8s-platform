/**
 * 资源进度条组件
 * CPU/内存等资源使用率展示
 */
import React from 'react'
import { Progress, Tooltip } from 'antd'

interface ResourceProgressProps {
  used: number
  total: number
  unit?: string
  showText?: boolean
  size?: 'small' | 'default'
}

export const ResourceProgress: React.FC<ResourceProgressProps> = ({
  used,
  total,
  unit = '',
  showText = true,
  size = 'default',
}) => {
  const percent = total > 0 ? Math.round((used / total) * 100) : 0
  const status = percent >= 90 ? 'exception' : percent >= 70 ? 'active' : 'normal'

  return (
    <Tooltip title={`${used}${unit} / ${total}${unit}`}>
      <Progress
        percent={percent}
        status={status}
        size={size === 'small' ? 'small' : 'default'}
        format={() => (showText ? `${used}/${total}${unit}` : `${percent}%`)}
      />
    </Tooltip>
  )
}
