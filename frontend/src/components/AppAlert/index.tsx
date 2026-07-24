import { useId, useState } from 'react'
import { DownOutlined, UpOutlined } from '@ant-design/icons'
import { Alert, Button, Space, type AlertProps } from 'antd'

export interface AppAlertProps extends AlertProps {
  /** 默认收起补充说明，避免提示信息打断页面主流程。 */
  compact?: boolean
}

/**
 * 平台内嵌提示的统一入口。
 * 关键信息始终可见；较长的补充说明按需展开，避免表单和列表被提示框挤占。
 */
export default function AppAlert({
  action,
  className,
  compact = true,
  description,
  ...alertProps
}: AppAlertProps) {
  const [expanded, setExpanded] = useState(false)
  const detailsId = useId()
  const hasDescription = description !== undefined && description !== null

  if (!compact || !hasDescription) {
    return (
      <Alert
        {...alertProps}
        action={action}
        className={['app-alert', className].filter(Boolean).join(' ')}
        description={description}
      />
    )
  }

  const detailsToggle = (
    <Button
      aria-controls={detailsId}
      aria-expanded={expanded}
      className="app-alert__details-toggle"
      icon={expanded ? <UpOutlined /> : <DownOutlined />}
      size="small"
      type="link"
      onClick={() => setExpanded((value) => !value)}
    >
      {expanded ? '收起说明' : '查看说明'}
    </Button>
  )

  return (
    <Alert
      {...alertProps}
      action={action ? <Space size={4}>{detailsToggle}{action}</Space> : detailsToggle}
      className={['app-alert', 'app-alert--collapsible', className].filter(Boolean).join(' ')}
      description={expanded ? <div id={detailsId}>{description}</div> : undefined}
    />
  )
}
