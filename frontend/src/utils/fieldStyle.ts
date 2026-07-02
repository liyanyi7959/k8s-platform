/**
 * 字段样式工具
 * 用于识别和应用居中展示的字段类型
 */

/** 居中展示的字段类型 */
export type FieldCenterType = 'number' | 'status' | 'enum' | 'boolean' | 'short-id'

/** 字段配置 */
export interface FieldStyleConfig {
  /** 是否居中展示 */
  centered?: boolean
  /** 字段类型 */
  type?: FieldCenterType
  /** 自定义类名 */
  className?: string
}

/**
 * 判断字段是否应该居中展示
 * 规则：
 * 1. 纯数字字段（节点数、重启次数、端口号、数量等）
 * 2. 状态枚举字段（就绪状态、运行状态等）
 * 3. 布尔字段（是/否、启用/禁用等）
 * 4. 简短标识字段（角色、类型、级别等）
 */
export function shouldCenterField(dataIndex: string, title?: string): boolean {
  // 数字相关字段关键词
  const numberKeywords = [
    'count', 'num', 'number', 'port', 'replicas', 'ready', 'available',
    'cpu', 'memory', 'disk', 'size', 'limit', 'max', 'min', 'age',
    '次数', '数量', '端口', '节点', '副本', '就绪', 'CPU', '内存', '磁盘',
    '重启', '限制', '最大', '最小', '优先级', '排序'
  ]

  // 状态相关字段关键词
  const statusKeywords = [
    'status', 'state', 'phase', 'condition', 'health',
    '状态', '阶段', '健康', '就绪', '运行'
  ]

  // 布尔相关字段关键词
  const booleanKeywords = [
    'enabled', 'disabled', 'ready', 'ha', 'highAvailability',
    '启用', '禁用', '就绪', '高可用'
  ]

  // 枚举相关字段关键词
  const enumKeywords = [
    'role', 'type', 'mode', 'level', 'grade', 'kind', 'cni',
    '角色', '类型', '模式', '级别', '版本'
  ]

  const field = (dataIndex + ' ' + (title || '')).toLowerCase()

  return (
    numberKeywords.some(k => field.includes(k.toLowerCase())) ||
    statusKeywords.some(k => field.includes(k.toLowerCase())) ||
    booleanKeywords.some(k => field.includes(k.toLowerCase())) ||
    enumKeywords.some(k => field.includes(k.toLowerCase()))
  )
}

/**
 * 获取字段居中类名
 */
export function getFieldCenterClass(dataIndex: string, title?: string): string {
  if (!shouldCenterField(dataIndex, title)) {
    return ''
  }

  const field = (dataIndex + ' ' + (title || '')).toLowerCase()

  // 数字字段
  if (['count', 'num', 'number', 'port', 'replicas', 'cpu', 'memory', 'disk', 'size',
       '数量', '端口', '节点', '副本', 'CPU', '内存', '磁盘', '重启', '限制'].some(k => field.includes(k.toLowerCase()))) {
    return 'field-center field-number'
  }

  // 状态字段
  if (['status', 'state', 'phase', 'condition', 'health',
       '状态', '阶段', '健康', '就绪'].some(k => field.includes(k.toLowerCase()))) {
    return 'field-center field-status'
  }

  return 'field-center'
}

/**
 * 为 ProTable 列配置添加居中样式
 */
export function withCenterStyle<T extends { dataIndex?: string; title?: string; className?: string }>(
  column: T
): T {
  if (column.dataIndex && shouldCenterField(column.dataIndex as string, column.title as string)) {
    return {
      ...column,
      align: 'center',
      className: `${column.className || ''} ${getFieldCenterClass(column.dataIndex as string, column.title as string)}`.trim(),
    }
  }
  return column
}

/**
 * 批量为列配置添加居中样式
 */
export function withCenterStyleBatch<T extends { dataIndex?: string; title?: string; className?: string }>(
  columns: T[]
): T[] {
  return columns.map(withCenterStyle)
}
