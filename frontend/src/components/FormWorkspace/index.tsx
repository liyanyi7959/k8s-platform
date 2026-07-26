import type { ReactNode } from 'react'
import WorkspaceHeader from '@/components/WorkspaceHeader'

export interface FormWorkspaceProps {
  title: ReactNode
  description?: ReactNode
  actions?: ReactNode
  children: ReactNode
  className?: string
}

/** Standard configuration page: one task header and one readable-width form panel. */
export function FormWorkspace({ title, description, actions, children, className }: FormWorkspaceProps) {
  return (
    <div className={['app-form-workspace', className].filter(Boolean).join(' ')}>
      <WorkspaceHeader title={title} description={description} actions={actions} />
      <section className="app-form-workspace__panel">{children}</section>
    </div>
  )
}

export default FormWorkspace
