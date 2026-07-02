/**
 * 通用资源详情抽屉
 * 对标 frontend-old 的各种 DetailDrawer（PodDetailDrawer/DeploymentDetailDrawer 等）
 * 通过 ProDescriptions 展示资源详情
 */
import React from 'react'
import { Drawer, Descriptions, Space, Button, message } from 'antd'
import { CopyOutlined } from '@ant-design/icons'
import { YamlEditor } from '@/components/YamlEditor'

interface DetailField {
  label: string
  key: string
  render?: (value: unknown, record: Record<string, unknown>) => React.ReactNode
  span?: number
}

interface ResourceDetailDrawerProps {
  open: boolean
  onClose: () => void
  title: string
  data: Record<string, unknown> | null | undefined
  fields: DetailField[]
  yaml?: string
  extra?: React.ReactNode
}

const defaultRender = (value: unknown): React.ReactNode => {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'boolean') return value ? '是' : '否'
  if (Array.isArray(value)) return value.join(', ')
  if (typeof value === 'object') return JSON.stringify(value, null, 2)
  return String(value)
}

export const ResourceDetailDrawer: React.FC<ResourceDetailDrawerProps> = ({
  open,
  onClose,
  title,
  data,
  fields,
  yaml,
  extra,
}) => {
  const handleCopy = () => {
    if (!data) return
    const text = JSON.stringify(data, null, 2)
    navigator.clipboard.writeText(text).then(() => {
      message.success('已复制到剪贴板')
    })
  }

  return (
    <Drawer
      title={title}
      open={open}
      onClose={onClose}
      width={720}
      extra={
        <Space>
          {extra}
          <Button icon={<CopyOutlined />} onClick={handleCopy}>复制</Button>
        </Space>
      }
    >
      {data && (
        <>
          <Descriptions bordered column={2} size="small">
            {fields.map((field) => (
              <Descriptions.Item key={field.key} label={field.label} span={field.span || 1}>
                {field.render ? field.render(data[field.key], data) : defaultRender(data[field.key])}
              </Descriptions.Item>
            ))}
          </Descriptions>
          {yaml && (
            <div style={{ marginTop: 16 }}>
              <h4 style={{ marginBottom: 8 }}>YAML 定义</h4>
              <YamlEditor value={yaml} readOnly height={300} />
            </div>
          )}
        </>
      )}
    </Drawer>
  )
}
