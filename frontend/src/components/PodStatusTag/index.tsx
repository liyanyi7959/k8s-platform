/**
 * Pod 状态标签组件
 * 支持容器级状态：CrashLoopBackOff / ImagePullBackOff / ContainerCreating 等
 */
import React from 'react'
import { Tag, Tooltip } from 'antd'

interface PodStatusTagProps {
  status: string
  ready?: string
  containerReason?: string
  onClick?: () => void
}

// 容器级等待状态映射（优先级高于 phase）
const containerReasonMap: Record<string, { color: string; text: string }> = {
  CrashLoopBackOff: { color: 'error', text: '崩溃循环' },
  ImagePullBackOff: { color: 'error', text: '镜像拉取失败' },
  ErrImagePull: { color: 'error', text: '镜像拉取错误' },
  ImagePullError: { color: 'error', text: '镜像拉取错误' },
  CreateContainerError: { color: 'error', text: '容器创建错误' },
  InvalidImageName: { color: 'error', text: '镜像名无效' },
  ContainerCreating: { color: 'processing', text: '容器创建中' },
  PodInitializing: { color: 'processing', text: 'Pod 初始化中' },
  SandboxCreateError: { color: 'error', text: '沙箱创建失败' },
  OOMKilled: { color: 'error', text: '内存溢出终止' },
  DeadlineExceeded: { color: 'error', text: '超时终止' },
  Evicted: { color: 'error', text: '已驱逐' },
}

const phaseMap: Record<string, { color: string; text: string }> = {
  Running: { color: 'success', text: '运行中' },
  Pending: { color: 'warning', text: '等待中' },
  Failed: { color: 'error', text: '失败' },
  Succeeded: { color: 'default', text: '成功' },
  Unknown: { color: 'default', text: '未知' },
}

export const PodStatusTag: React.FC<PodStatusTagProps> = ({ status, ready, containerReason, onClick }) => {
  // 容器级状态优先显示
  const reasonConfig = containerReason ? containerReasonMap[containerReason] : undefined
  const phaseConfig = phaseMap[status] || { color: 'default', text: status }
  const config = reasonConfig || phaseConfig

  const tooltipContent = [
    `状态: ${phaseConfig.text}`,
    containerReason ? `原因: ${containerReason}` : '',
    ready ? `就绪: ${ready}` : '',
  ].filter(Boolean).join(' | ')

  return (
    <Tooltip title={tooltipContent || undefined}>
      <Tag
        color={config.color}
        onClick={onClick}
        style={{ cursor: onClick ? 'pointer' : 'default' }}
      >
        {config.text}
        {ready && ` (${ready})`}
      </Tag>
    </Tooltip>
  )
}
