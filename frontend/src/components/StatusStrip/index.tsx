import type { ReactNode } from 'react'

export interface StatusStripProps {
  children: ReactNode
  tone?: 'default' | 'success' | 'warning' | 'danger'
  className?: string
}

/** 数据来源、更新时间和当前风险等运行态上下文的紧凑承载区。 */
export function StatusStrip({ children, tone = 'default', className }: StatusStripProps) {
  return (
    <div className={['app-status-strip', `is-${tone}`, className].filter(Boolean).join(' ')} role="status">
      {children}
    </div>
  )
}

export default StatusStrip
