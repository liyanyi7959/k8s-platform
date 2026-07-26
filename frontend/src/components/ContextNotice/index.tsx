import type { ReactNode } from 'react'
import type { AlertProps } from 'antd'
import AppAlert from '@/components/AppAlert'

export interface ContextNoticeProps extends Pick<AlertProps, 'type' | 'showIcon' | 'icon' | 'style'> {
  message: ReactNode
  description?: ReactNode
  action?: ReactNode
  className?: string
}

/** 仅用于安全边界、不可逆操作和当前异常，补充说明默认按需展开。 */
export function ContextNotice({ className, type = 'info', showIcon = true, ...props }: ContextNoticeProps) {
  return (
    <AppAlert
      {...props}
      type={type}
      showIcon={showIcon}
      className={['app-context-notice', className].filter(Boolean).join(' ')}
    />
  )
}

export default ContextNotice
