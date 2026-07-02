/** 通用分页响应 */
export interface PageResult<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

/** AI 对话会话 */
export interface Conversation {
  id: string
  title: string
  messageCount: number
  createdAt: string
  updatedAt: string
}
