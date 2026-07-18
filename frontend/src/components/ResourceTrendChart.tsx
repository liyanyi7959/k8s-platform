import React, { useMemo, useState } from 'react'

export interface ResourceTrendPoint {
  time: string
  sampledAt?: string
  cpu: number | null
  memory: number | null
}
interface ResourceTrendChartProps {
  points: ResourceTrendPoint[]
  height?: number
  empty?: boolean
}

const width = 960
const height = 280
const left = 54
const right = 18
const top = 18
const bottom = 42
const colors = { cpu: '#0891b2', memory: '#7c3aed' }

const valueOf = (value: number | null) =>
  typeof value === 'number' && Number.isFinite(value) ? Math.max(0, Math.min(100, value)) : null

function makePath(points: ResourceTrendPoint[], key: 'cpu' | 'memory') {
  const valid = points
    .map((point, index) => ({ index, value: valueOf(point[key]) }))
    .filter((item): item is { index: number; value: number } => item.value !== null)
  if (valid.length < 2) return ''
  const sampleGaps = valid
    .slice(1)
    .map((item, index) => {
      const previous = points[valid[index]!.index]
      const current = points[item.index]
      if (!previous?.sampledAt || !current?.sampledAt) return 0
      return new Date(current.sampledAt).getTime() - new Date(previous.sampledAt).getTime()
    })
    .filter((gap) => gap > 0)
    .sort((a, b) => a - b)
  const typicalGap = sampleGaps.length ? sampleGaps[Math.floor(sampleGaps.length / 2)]! : 0
  const disconnectAfter = Math.max(30 * 60 * 1000, typicalGap * 3)
  return valid
    .map((item, index) => {
      const x = left + (item.index / Math.max(1, points.length - 1)) * (width - left - right)
      const y = top + (1 - item.value / 100) * (height - top - bottom)
      const previousIndex = index > 0 ? valid[index - 1]?.index : undefined
      const previous = previousIndex === undefined ? undefined : points[previousIndex]
      const current = points[item.index]!
      const gap =
        previous?.sampledAt && current.sampledAt
          ? new Date(current.sampledAt).getTime() - new Date(previous.sampledAt).getTime()
          : 0
      return `${index === 0 || gap > disconnectAfter ? 'M' : 'L'} ${x.toFixed(2)} ${y.toFixed(2)}`
    })
    .join(' ')
}

export const ResourceTrendChart: React.FC<ResourceTrendChartProps> = ({
  points,
  height: displayHeight = height,
  empty = false,
}) => {
  const [hoverIndex, setHoverIndex] = useState<number | null>(null)
  const cpuPath = useMemo(() => makePath(points, 'cpu'), [points])
  const memoryPath = useMemo(() => makePath(points, 'memory'), [points])
  const labels = points.length
    ? points
    : Array.from({ length: 24 }, (_, index) => ({
        time: `${String(index).padStart(2, '0')}:00`,
        cpu: null,
        memory: null,
      }))
  const plotHeight = height - top - bottom
  const plotWidth = width - left - right
  const hasSamples = points.some(
    (point) => valueOf(point.cpu) !== null || valueOf(point.memory) !== null,
  )
  const hoveredPoint = hoverIndex === null ? null : points[hoverIndex]
  const hoveredX =
    hoverIndex === null ? null : left + (hoverIndex / Math.max(1, points.length - 1)) * plotWidth

  const handleMouseMove = (event: React.MouseEvent<SVGSVGElement>) => {
    if (!points.length) return
    const bounds = event.currentTarget.getBoundingClientRect()
    const svgX = ((event.clientX - bounds.left) / bounds.width) * width
    const ratio = Math.max(0, Math.min(1, (svgX - left) / plotWidth))
    setHoverIndex(Math.round(ratio * Math.max(0, points.length - 1)))
  }

  return (
    <div style={{ position: 'relative', height: displayHeight, width: '100%' }}>
      <svg
        viewBox={`0 0 ${width} ${height}`}
        preserveAspectRatio="none"
        style={{ width: '100%', height: '100%', cursor: hasSamples ? 'crosshair' : 'default' }}
        role="img"
        aria-label="CPU 和内存使用率趋势"
        onMouseMove={handleMouseMove}
        onMouseLeave={() => setHoverIndex(null)}
      >
        {[0, 25, 50, 75, 100].map((tick) => {
          const y = top + (1 - tick / 100) * plotHeight
          return (
            <g key={tick}>
              <line
                x1={left}
                x2={width - right}
                y1={y}
                y2={y}
                stroke="#e5e7eb"
                strokeDasharray="4 4"
              />
              <text x={left - 10} y={y + 4} textAnchor="end" fill="#9ca3af" fontSize="11">
                {tick}%
              </text>
            </g>
          )
        })}
        <line x1={left} x2={left} y1={top} y2={top + plotHeight} stroke="#d1d5db" />
        <line
          x1={left}
          x2={width - right}
          y1={top + plotHeight}
          y2={top + plotHeight}
          stroke="#d1d5db"
        />
        {labels.map((point, index) =>
          index % Math.max(1, Math.ceil(labels.length / 8)) === 0 || index === labels.length - 1 ? (
            <text
              key={`${point.time}-${index}`}
              x={left + (index / Math.max(1, labels.length - 1)) * plotWidth}
              y={height - 14}
              textAnchor="middle"
              fill="#9ca3af"
              fontSize="11"
            >
              {point.time}
            </text>
          ) : null,
        )}
        {cpuPath && (
          <path
            d={cpuPath}
            fill="none"
            stroke={colors.cpu}
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        )}
        {memoryPath && (
          <path
            d={memoryPath}
            fill="none"
            stroke={colors.memory}
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        )}
        {points.map((point, index) => {
          const x = left + (index / Math.max(1, points.length - 1)) * plotWidth
          return (
            <React.Fragment key={`${point.time}-${index}`}>
              {valueOf(point.cpu) !== null && (
                <circle
                  cx={x}
                  cy={top + (1 - valueOf(point.cpu)! / 100) * plotHeight}
                  r={hoverIndex === index ? 4 : 2.5}
                  fill={colors.cpu}
                />
              )}
              {valueOf(point.memory) !== null && (
                <circle
                  cx={x}
                  cy={top + (1 - valueOf(point.memory)! / 100) * plotHeight}
                  r={hoverIndex === index ? 4 : 2.5}
                  fill={colors.memory}
                />
              )}
            </React.Fragment>
          )
        })}
        {hoveredX !== null && hoveredPoint && (
          <line
            x1={hoveredX}
            x2={hoveredX}
            y1={top}
            y2={top + plotHeight}
            stroke="#94a3b8"
            strokeWidth="1"
            strokeDasharray="3 3"
          />
        )}
      </svg>
      <div
        style={{
          position: 'absolute',
          top: 4,
          right: 18,
          display: 'flex',
          gap: 16,
          fontSize: 12,
          color: '#6b7280',
        }}
      >
        <span>
          <i
            style={{
              display: 'inline-block',
              width: 10,
              height: 10,
              borderRadius: '50%',
              background: colors.cpu,
              marginRight: 5,
            }}
          />
          CPU
        </span>
        <span>
          <i
            style={{
              display: 'inline-block',
              width: 10,
              height: 10,
              borderRadius: '50%',
              background: colors.memory,
              marginRight: 5,
            }}
          />
          内存
        </span>
      </div>
      {hoveredPoint && hoveredX !== null && (
        <div
          style={{
            position: 'absolute',
            left: `${(hoveredX / width) * 100}%`,
            top: 34,
            transform:
              hoveredX > width * 0.75
                ? 'translateX(-100%)'
                : hoveredX > width * 0.25
                  ? 'translateX(-50%)'
                  : undefined,
            minWidth: 150,
            padding: '8px 10px',
            borderRadius: 6,
            background: 'rgba(255,255,255,0.96)',
            border: '1px solid #e5e7eb',
            boxShadow: '0 4px 14px rgba(15,23,42,0.14)',
            pointerEvents: 'none',
            zIndex: 2,
          }}
        >
          <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 5 }}>{hoveredPoint.time}</div>
          <div style={{ color: colors.cpu, fontSize: 12 }}>
            CPU 使用率：{valueOf(hoveredPoint.cpu)?.toFixed(1) ?? '--'}%
          </div>
          <div style={{ color: colors.memory, fontSize: 12 }}>
            内存使用率：{valueOf(hoveredPoint.memory)?.toFixed(1) ?? '--'}%
          </div>
        </div>
      )}
      {(empty || !hasSamples) && (
        <div
          style={{
            position: 'absolute',
            inset: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#9ca3af',
            fontSize: 13,
          }}
        >
          暂无真实监控采样数据
        </div>
      )}
    </div>
  )
}

export default ResourceTrendChart
