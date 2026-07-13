import React, { useMemo } from 'react'

export interface ResourceTrendPoint { time: string; cpu: number | null; memory: number | null }
interface ResourceTrendChartProps { points: ResourceTrendPoint[]; height?: number; empty?: boolean }

const width = 960
const height = 280
const left = 54
const right = 18
const top = 18
const bottom = 42
const colors = { cpu: '#0891b2', memory: '#7c3aed' }

const valueOf = (value: number | null) => typeof value === 'number' && Number.isFinite(value) ? Math.max(0, Math.min(100, value)) : null

function makePath(points: ResourceTrendPoint[], key: 'cpu' | 'memory') {
  const valid = points.map((point, index) => ({ index, value: valueOf(point[key]) })).filter((item): item is { index: number; value: number } => item.value !== null)
  if (valid.length < 2) return ''
  return valid.map((item, index) => {
    const x = left + item.index / Math.max(1, points.length - 1) * (width - left - right)
    const y = top + (1 - item.value / 100) * (height - top - bottom)
    return `${index === 0 ? 'M' : 'L'} ${x.toFixed(2)} ${y.toFixed(2)}`
  }).join(' ')
}

export const ResourceTrendChart: React.FC<ResourceTrendChartProps> = ({ points, height: displayHeight = height, empty = false }) => {
  const cpuPath = useMemo(() => makePath(points, 'cpu'), [points])
  const memoryPath = useMemo(() => makePath(points, 'memory'), [points])
  const labels = points.length ? points : Array.from({ length: 24 }, (_, index) => ({ time: `${String(index).padStart(2, '0')}:00`, cpu: null, memory: null }))
  const plotHeight = height - top - bottom
  const plotWidth = width - left - right
  return <div style={{ position: 'relative', height: displayHeight, width: '100%' }}>
    <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" style={{ width: '100%', height: '100%' }} role="img" aria-label="CPU 和内存使用率趋势">
      {[0, 25, 50, 75, 100].map((tick) => { const y = top + (1 - tick / 100) * plotHeight; return <g key={tick}><line x1={left} x2={width - right} y1={y} y2={y} stroke="#e5e7eb" strokeDasharray="4 4" /><text x={left - 10} y={y + 4} textAnchor="end" fill="#9ca3af" fontSize="11">{tick}%</text></g> })}
      <line x1={left} x2={left} y1={top} y2={top + plotHeight} stroke="#d1d5db" />
      <line x1={left} x2={width - right} y1={top + plotHeight} y2={top + plotHeight} stroke="#d1d5db" />
      {labels.map((point, index) => index % Math.max(1, Math.ceil(labels.length / 8)) === 0 || index === labels.length - 1 ? <text key={`${point.time}-${index}`} x={left + index / Math.max(1, labels.length - 1) * plotWidth} y={height - 14} textAnchor="middle" fill="#9ca3af" fontSize="11">{point.time}</text> : null)}
      {cpuPath && <path d={cpuPath} fill="none" stroke={colors.cpu} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />}
      {memoryPath && <path d={memoryPath} fill="none" stroke={colors.memory} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />}
    </svg>
    <div style={{ position: 'absolute', top: 4, right: 18, display: 'flex', gap: 16, fontSize: 12, color: '#6b7280' }}><span><i style={{ display: 'inline-block', width: 10, height: 10, borderRadius: '50%', background: colors.cpu, marginRight: 5 }} />CPU</span><span><i style={{ display: 'inline-block', width: 10, height: 10, borderRadius: '50%', background: colors.memory, marginRight: 5 }} />内存</span></div>
    {(empty || (!cpuPath && !memoryPath)) && <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#9ca3af', fontSize: 13 }}>暂无可用监控采样数据</div>}
  </div>
}

export default ResourceTrendChart
