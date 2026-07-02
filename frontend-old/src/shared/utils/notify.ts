import { ElMessage } from 'element-plus'

export function notifySuccess(message: string): void {
  ElMessage.success(message)
}

export function notifyError(message: string): void {
  const raw = String(message ?? '')
  const cleaned = raw
    .replace(/\s*\(\s*request_id\s*=\s*[^)]+\)\s*/gi, ' ')
    .replace(/\s*\(\s*requestId\s*=\s*[^)]+\)\s*/g, ' ')
    .replace(/\s*request_id\s*=\s*[a-z0-9-]+\s*/gi, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  ElMessage.error(cleaned || '操作失败')
}
