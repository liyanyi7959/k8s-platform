/**
 * 格式化工具函数
 * 日期、容量、CPU、数字等格式化
 */
import dayjs from 'dayjs'

/** 日期格式化 */
export function formatDate(date: string | Date, format?: string) {
  const defaultFormat = 'YYYY-MM-DD HH:mm:ss'
  return dayjs(date).format(format || defaultFormat)
}

/** 相对时间 */
export function formatRelativeTime(date: string | Date) {
  const now = dayjs()
  const target = dayjs(date)
  const diffSeconds = now.diff(target, 'second')

  if (diffSeconds < 60) return `${diffSeconds}秒前`
  if (diffSeconds < 3600) return `${Math.floor(diffSeconds / 60)}分钟前`
  if (diffSeconds < 86400) return `${Math.floor(diffSeconds / 3600)}小时前`
  return `${Math.floor(diffSeconds / 86400)}天前`
}

/** 容量格式化 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${units[i]}`
}

/** CPU 格式化 (millicore → core) */
export function formatCPU(millicores: number): string {
  if (millicores >= 1000) {
    return `${(millicores / 1000).toFixed(2)} Core`
  }
  return `${millicores} m`
}

/** 数字格式化 (千分位) */
export function formatNumber(num: number): string {
  return new Intl.NumberFormat().format(num)
}

/** 百分比格式化 */
export function formatPercent(value: number, decimals = 1): string {
  return `${(value * 100).toFixed(decimals)}%`
}
