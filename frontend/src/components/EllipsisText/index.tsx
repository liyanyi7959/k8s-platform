/**
 * 长文本省略组件
 * 用于表格单元格中承载 K8s 动态名称（namespace、资源名、控制器名等可能超长的字符串）
 * 固定最大宽度，超出部分末尾截断显示省略号，鼠标悬浮 Tooltip 展示完整原文
 */
import React from 'react'
import { Tag, Tooltip, Typography } from 'antd'

const { Text } = Typography

export interface EllipsisTextProps {
  /** 文本内容 */
  text?: string | number | null
  /** 是否用 Tag 包裹展示 */
  tag?: boolean
  /** Tag 颜色（仅 tag 模式生效） */
  color?: string
  /** 最大宽度，默认 100% 撑满单元格 */
  maxWidth?: number | string
  /** 空值占位符，默认 - */
  placeholder?: string
  /** 纯文本模式下是否使用次要颜色 */
  secondary?: boolean
}

const ellipsisStyle: React.CSSProperties = {
  display: 'inline-block',
  maxWidth: '100%',
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
  verticalAlign: 'middle',
}

/** 长文本省略 + 悬浮 Tooltip 组件 */
const EllipsisText: React.FC<EllipsisTextProps> = ({
  text,
  tag = false,
  color,
  maxWidth = '100%',
  placeholder = '-',
  secondary = false,
}) => {
  const value = text == null || text === '' ? '' : String(text)

  if (!value) {
    return <Text type="secondary">{placeholder}</Text>
  }

  const style: React.CSSProperties = { ...ellipsisStyle, maxWidth }

  if (tag) {
    return (
      <Tooltip title={value}>
        <Tag color={color} style={style}>
          {value}
        </Tag>
      </Tooltip>
    )
  }

  return (
    <Tooltip title={value}>
      <span style={style}>
        <Text type={secondary ? 'secondary' : undefined} style={{ maxWidth: '100%' }}>
          {value}
        </Text>
      </span>
    </Tooltip>
  )
}

export default EllipsisText
