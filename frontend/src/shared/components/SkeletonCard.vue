<template>
  <div class="skeleton-card" :class="{ 'skeleton-card--has-header': showHeader }">
    <!-- 卡片头部骨架 -->
    <div v-if="showHeader" class="skeleton-header">
      <div class="skeleton-line skeleton-title" />
      <div v-if="showSubtitle" class="skeleton-line skeleton-subtitle" />
    </div>

    <!-- 图表占位 -->
    <div v-if="variant === 'chart'" class="skeleton-chart">
      <div class="skeleton-chart-bars">
        <div v-for="i in 7" :key="i" class="skeleton-bar" :style="{ height: barHeight(i) }" />
      </div>
    </div>

    <!-- 指标卡占位 -->
    <div v-else-if="variant === 'metric'" class="skeleton-metric">
      <div class="skeleton-metric-icon skeleton-pulse" />
      <div class="skeleton-metric-body">
        <div class="skeleton-line" style="width:60%" />
        <div class="skeleton-line skeleton-big" style="width:40%" />
        <div class="skeleton-line" style="width:80%" />
      </div>
    </div>

    <!-- 表格占位 -->
    <div v-else-if="variant === 'table'" class="skeleton-table">
      <div class="skeleton-table-header">
        <div v-for="i in columns" :key="i" class="skeleton-line" :style="{ width: colWidth(i) }" />
      </div>
      <div v-for="r in rows" :key="r" class="skeleton-table-row">
        <div v-for="i in columns" :key="i" class="skeleton-line" :style="{ width: colWidth(i) }" />
      </div>
    </div>

    <!-- 通用行占位（默认） -->
    <div v-else class="skeleton-lines">
      <div v-for="r in rows" :key="r" class="skeleton-line" :style="{ width: lineWidth(r) }" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  /** 骨架类型: lines(默认) | chart | metric | table */
  variant?: 'lines' | 'chart' | 'metric' | 'table'
  /** 行数，默认 4 */
  rows?: number
  /** 表格列数，默认 5 */
  columns?: number
  /** 是否显示标题头 */
  showHeader?: boolean
  /** 是否显示副标题 */
  showSubtitle?: boolean
}>()

/** 柱状图随机高度 */
function barHeight(i: number): string {
  const heights = [65, 85, 45, 90, 55, 75, 60]
  return `${heights[(i - 1) % heights.length]}%`
}

/** 每行宽度错开 */
function lineWidth(row: number): string {
  const widths = ['100%', '85%', '70%', '92%', '60%', '78%']
  return widths[(row - 1) % widths.length]
}

/** 表格列宽 */
function colWidth(col: number): string {
  const widths = ['15%', '25%', '20%', '18%', '12%', '10%']
  return widths[(col - 1) % widths.length]
}
</script>

<style scoped>
.skeleton-card {
  @apply rounded-xl border border-slate-200/60 bg-white p-5;
}

:global(html.dark) .skeleton-card {
  @apply border-slate-700/50 bg-slate-800/60;
}

.skeleton-header {
  @apply mb-4 space-y-2;
}

/* ---- 通用脉冲动画条 ---- */
.skeleton-line {
  @apply h-3 rounded-md;
  background: linear-gradient(90deg, var(--sk-from) 25%, var(--sk-via) 50%, var(--sk-from) 75%);
  background-size: 200% 100%;
  animation: sk-shimmer 1.5s ease-in-out infinite;
}

.skeleton-pulse {
  background: linear-gradient(90deg, var(--sk-from) 25%, var(--sk-via) 50%, var(--sk-from) 75%);
  background-size: 200% 100%;
  animation: sk-shimmer 1.5s ease-in-out infinite;
}

:root {
  --sk-from: #e2e8f0;
  --sk-via: #f1f5f9;
}

:global(html.dark) {
  --sk-from: #334155;
  --sk-via: #475569;
}

.skeleton-title {
  @apply h-4 w-2/5;
}

.skeleton-subtitle {
  @apply h-3 w-1/4;
}

/* ---- Metric 变体 ---- */
.skeleton-metric {
  @apply flex items-center gap-4;
}

.skeleton-metric-icon {
  @apply h-12 w-12 flex-shrink-0 rounded-xl;
}

.skeleton-metric-body {
  @apply flex-1 space-y-2;
}

.skeleton-big {
  @apply h-5;
}

/* ---- Chart 变体 ---- */
.skeleton-chart {
  @apply h-48;
}

.skeleton-chart-bars {
  @apply flex h-full items-end gap-3 px-4;
}

.skeleton-bar {
  @apply flex-1 rounded-t-md;
  background: linear-gradient(90deg, var(--sk-from) 25%, var(--sk-via) 50%, var(--sk-from) 75%);
  background-size: 200% 100%;
  animation: sk-shimmer 1.5s ease-in-out infinite;
}

/* ---- Table 变体 ---- */
.skeleton-table {
  @apply space-y-0;
}

.skeleton-table-header {
  @apply flex gap-4 border-b border-slate-200/60 pb-3 mb-2;
}

:global(html.dark) .skeleton-table-header {
  @apply border-slate-700/50;
}

.skeleton-table-row {
  @apply flex gap-4 py-3;
}

.skeleton-table-row .skeleton-line {
  @apply h-3;
}

/* ---- Lines 变体 ---- */
.skeleton-lines {
  @apply space-y-3;
}

/* ---- 动画 ---- */
@keyframes sk-shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skeleton-line,
  .skeleton-pulse,
  .skeleton-bar {
    animation: none;
    opacity: 0.6;
  }
}
</style>
