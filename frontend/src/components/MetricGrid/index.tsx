import type { ReactNode } from 'react'

export interface MetricItem {
  key: string
  label: ReactNode
  value: ReactNode
  detail?: ReactNode
  icon?: ReactNode
  tone?: 'default' | 'success' | 'warning' | 'danger'
}

export interface MetricGridProps {
  items: MetricItem[]
  className?: string
}

/** 只展示能驱动运维决策的实时指标，统一标签、数值与辅助信息层级。 */
export function MetricGrid({ items, className }: MetricGridProps) {
  return (
    <section className={['app-metric-grid', className].filter(Boolean).join(' ')} aria-label="关键指标">
      {items.map((item) => (
        <article key={item.key} className={['app-metric-grid__item', item.tone ? `is-${item.tone}` : ''].filter(Boolean).join(' ')}>
          <div className="app-metric-grid__label">
            {item.icon ? <span className="app-metric-grid__icon">{item.icon}</span> : null}
            <span>{item.label}</span>
          </div>
          <strong className="app-metric-grid__value">{item.value}</strong>
          {item.detail ? <span className="app-metric-grid__detail">{item.detail}</span> : null}
        </article>
      ))}
    </section>
  )
}

export default MetricGrid
