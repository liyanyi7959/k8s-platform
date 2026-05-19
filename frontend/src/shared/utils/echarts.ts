/**
 * ECharts 按需引入模块
 *
 * 按需引入替代 `import * as echarts from 'echarts'`，
 * 预计减少 bundle 400KB+（全量 ~700KB → 按需 ~250KB）。
 *
 * 使用方式：
 *   import { echarts } from '@/shared/utils/echarts'
 *   // echarts.init(el)
 *   // echarts.graphic.LinearGradient(...)
 *   // type EChartsOption 从 'echarts/types/dist/echarts' 导出
 */

import * as echarts from 'echarts/core'

// ── 图表类型 ──
import { BarChart, LineChart, PieChart, GaugeChart, ScatterChart } from 'echarts/charts'

// ── 组件 ──
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  TitleComponent,
  DataZoomComponent,
  ToolboxComponent,
  MarkLineComponent,
  MarkPointComponent,
  GraphicComponent,
  PolarComponent,
} from 'echarts/components'

// ── 渲染器 ──
import { CanvasRenderer } from 'echarts/renderers'

// 注册组件
echarts.use([
  // Charts
  BarChart,
  LineChart,
  PieChart,
  GaugeChart,
  ScatterChart,
  // Components
  GridComponent,
  TooltipComponent,
  LegendComponent,
  TitleComponent,
  DataZoomComponent,
  ToolboxComponent,
  MarkLineComponent,
  MarkPointComponent,
  GraphicComponent,
  PolarComponent,
  // Renderer
  CanvasRenderer,
])

export { echarts }

// 重新导出常用类型
export type { EChartsCoreOption as EChartsOption } from 'echarts/core'
export type { ECharts } from 'echarts/core'
