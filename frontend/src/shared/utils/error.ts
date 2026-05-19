export class ApiError extends Error {
  code: number
  requestId?: string
  data?: unknown

  constructor(params: { code: number; message: string; requestId?: string; data?: unknown }) {
    super(params.message)
    this.name = 'ApiError'
    this.code = params.code
    this.requestId = params.requestId
    this.data = params.data
  }
}
