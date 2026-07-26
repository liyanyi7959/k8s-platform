import type { ReactNode } from 'react'
import WorkspaceHeader from '@/components/WorkspaceHeader'

export interface ListWorkspaceProps {
  title?: ReactNode
  description?: ReactNode
  /** A live list summary, such as “当前共 12 个项目”. */
  summary?: ReactNode
  actions?: ReactNode
  filters?: ReactNode
  children: ReactNode
  className?: string
}

/**
 * 默认资源列表工作台：摘要/操作、可选筛选、带统一内嵌间距的表格或空状态。
 * 以“部署 K8S 集群”页面为视觉基线，禁止页面自行让表格贴边。
 */
export function ListWorkspace({
  title,
  description,
  summary,
  actions,
  filters,
  children,
  className,
}: ListWorkspaceProps) {
  return (
    <div className={['app-list-workspace', className].filter(Boolean).join(' ')}>
      <section className="app-list-workspace__surface">
        {summary ? (
          <div className="app-list-workspace__summary-row">
            <div className="app-list-workspace__summary">{summary}</div>
            {actions ? <div className="app-list-workspace__actions">{actions}</div> : null}
          </div>
        ) : title ? (
          <WorkspaceHeader title={title} description={description} actions={actions} />
        ) : actions ? (
          <div className="app-list-workspace__summary-row app-list-workspace__summary-row--actions-only">
            <div />
            <div className="app-list-workspace__actions">{actions}</div>
          </div>
        ) : null}
        {filters ? <div className="app-list-workspace__toolbar">{filters}</div> : null}
        <div className="app-list-workspace__content">{children}</div>
      </section>
    </div>
  )
}

export default ListWorkspace
