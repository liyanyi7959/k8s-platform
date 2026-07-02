export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

// K8s API 响应泛型，替换 any 类型
export type K8sListResponse<T> = { list: T[] }
export type K8sObjectResponse<T> = { obj: T }
