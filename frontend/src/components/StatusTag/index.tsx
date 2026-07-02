/**
 * 通用状态标签组件
 * 统一的状态展示样式
 */
import React from 'react'
import { Tag } from 'antd'

type StatusType =
  | 'active'
  | 'inactive'
  | 'error'
  | 'pending'
  | 'running'
  | 'stopped'
  | 'degraded'
  | 'warning'
  | 'unknown'
  | 'healthy'
  | 'connected'
  | 'disconnected'
  | 'failed'
  | 'success'

interface StatusTagProps {
  status: StatusType | string
  text?: string
}

const statusConfig: Record<string, { color: string; label: string }> = {
  active: { color: 'success', label: '正常' },
  healthy: { color: 'success', label: '健康' },
  connected: { color: 'success', label: '已连接' },
  success: { color: 'success', label: '成功' },
  inactive: { color: 'default', label: '停用' },
  error: { color: 'error', label: '异常' },
  failed: { color: 'error', label: '失败' },
  disconnected: { color: 'error', label: '断连' },
  pending: { color: 'warning', label: '等待中' },
  running: { color: 'processing', label: '运行中' },
  stopped: { color: 'default', label: '已停止' },
  degraded: { color: 'warning', label: '降级' },
  warning: { color: 'warning', label: '预警' },
  unknown: { color: 'default', label: '未知' },
}

export const StatusTag: React.FC<StatusTagProps> = ({ status, text }) => {
  const normalizedStatus = String(status || 'unknown').trim().toLowerCase()
  const config = statusConfig[normalizedStatus] || {
    color: 'default',
    label: String(status || '未知'),
  }

  return (
    <Tag className="app-status-tag" color={config.color} bordered={false}>
      {text || config.label}
    </Tag>
  )
}
