export function getMetricContainers(row: any): any[] {
  return Array.isArray(row?.containers) ? row.containers : []
}

export function getContainerCount(row: any): number {
  return getMetricContainers(row).length
}

function parseQuantity(raw: unknown, multipliers: Record<string, number>): number | null {
  const text = String(raw ?? '').trim()
  if (!text) return null
  const match = text.match(/^([+-]?\d+(?:\.\d+)?)([A-Za-z]+)?$/)
  if (!match) return null
  const value = Number(match[1])
  if (!Number.isFinite(value)) return null
  const unit = String(match[2] ?? '')
  const multiplier = multipliers[unit]
  if (multiplier == null) return unit ? null : value
  return value * multiplier
}

export function parseCpuMillicores(raw: unknown): number | null {
  return parseQuantity(raw, {
    n: 0.000001,
    u: 0.001,
    m: 1,
    '': 1000,
    k: 1000000,
    M: 1000000000,
    G: 1000000000000
  })
}

export function parseMemoryBytes(raw: unknown): number | null {
  return parseQuantity(raw, {
    Ki: 1024,
    Mi: 1024 ** 2,
    Gi: 1024 ** 3,
    Ti: 1024 ** 4,
    Pi: 1024 ** 5,
    Ei: 1024 ** 6,
    K: 1000,
    M: 1000 ** 2,
    G: 1000 ** 3,
    T: 1000 ** 4,
    P: 1000 ** 5,
    E: 1000 ** 6,
    m: 0.001,
    '': 1
  })
}

function trimDecimalText(value: number, fractionDigits: number): string {
  return value.toFixed(fractionDigits).replace(/\.0+$/, '').replace(/(\.\d*[1-9])0+$/, '$1')
}

export function formatMillicores(value: number | null): string {
  if (value == null || !Number.isFinite(value)) return '-'
  if (Math.abs(value) >= 1000) return `${trimDecimalText(value / 1000, 2)} cores`
  return `${Math.round(value)}m`
}

export function formatBytes(value: number | null): string {
  if (value == null || !Number.isFinite(value)) return '-'
  const abs = Math.abs(value)
  if (abs >= 1024 ** 3) return `${trimDecimalText(value / 1024 ** 3, 2)} Gi`
  if (abs >= 1024 ** 2) return `${trimDecimalText(value / 1024 ** 2, 1)} Mi`
  if (abs >= 1024) return `${trimDecimalText(value / 1024, 1)} Ki`
  return `${Math.round(value)} B`
}

export function sumMetricUsage(row: any, resource: 'cpu' | 'memory'): number | null {
  const parser = resource === 'cpu' ? parseCpuMillicores : parseMemoryBytes
  let total = 0
  let hasValue = false
  for (const container of getMetricContainers(row)) {
    const parsed = parser(container?.usage?.[resource])
    if (parsed == null) continue
    hasValue = true
    total += parsed
  }
  return hasValue ? total : null
}

export function formatPodCpu(row: any): string {
  return formatMillicores(sumMetricUsage(row, 'cpu'))
}

export function formatPodMemory(row: any): string {
  return formatBytes(sumMetricUsage(row, 'memory'))
}