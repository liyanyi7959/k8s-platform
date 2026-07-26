import type { ReactNode } from 'react'
import { Typography } from 'antd'

const { Paragraph, Title } = Typography

export interface WorkspaceHeaderProps {
  title: ReactNode
  description?: ReactNode
  icon?: ReactNode
  actions?: ReactNode
  className?: string
}

/**
 * 页面级任务标题：仅承载当前页面名称、必要的一行上下文和操作入口。
 * 不用于展示运行统计或大段产品说明。
 */
export function WorkspaceHeader({ title, description, icon, actions, className }: WorkspaceHeaderProps) {
  return (
    <section className={['app-workspace-header', className].filter(Boolean).join(' ')}>
      <div className="app-workspace-header__identity">
        {icon ? <span className="app-workspace-header__icon">{icon}</span> : null}
        <div>
          <Title level={3}>{title}</Title>
          {description ? <Paragraph>{description}</Paragraph> : null}
        </div>
      </div>
      {actions ? <div className="app-workspace-header__actions">{actions}</div> : null}
    </section>
  )
}

export default WorkspaceHeader
