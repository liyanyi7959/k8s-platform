<template>
  <div :class="nodeClass" :style="nodeStyle" :title="data.tooltip || ''" @dblclick.stop="handleOpen">
    <Handle
      v-for="handle in targetHandles"
      :id="handle.id"
      :key="handle.id"
      type="target"
      :position="handle.position"
      class="topology-flow-node__handle"
    />
    <Handle
      v-for="handle in sourceHandles"
      :id="handle.id"
      :key="handle.id"
      type="source"
      :position="handle.position"
      class="topology-flow-node__handle"
    />

    <div class="topology-flow-node__head" :style="headStyle">
      <div class="topology-flow-node__head-main">
        <div class="topology-flow-node__badge" :style="{ color: palette.color }">{{ String(data.badge ?? '').toUpperCase() }}</div>
        <div class="topology-flow-node__title">{{ String(data.title ?? '') }}</div>
      </div>
      <div v-if="data.statusText" :class="['topology-flow-node__status', `is-${data.severity ?? 'normal'}`]">{{ data.statusText }}</div>
    </div>

    <div class="topology-flow-node__body">
      <div class="topology-flow-node__desc">{{ String(data.description ?? '-') }}</div>
      <div class="topology-flow-node__ref">{{ String(data.refText ?? '') }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import type { NodeProps } from '@vue-flow/core'

type FlowNodeData = {
  badge: string
  title: string
  description: string
  refText: string
  kind: string
  severity?: 'normal' | 'warning' | 'error'
  statusText?: string
  emphasis?: 'core' | 'primary' | 'secondary'
  cardWidth?: number
  cardHeight?: number
  tooltip?: string
  dimmed?: boolean
  highlighted?: boolean
  active?: boolean
  selected?: boolean
  group?: string
  onOpen?: () => void
}

const props = defineProps<NodeProps<FlowNodeData>>()

function paletteForKind(kind: string) {
  const dark = document.documentElement.classList.contains('dark')
  const map: Record<string, { color: string; bg: string; border: string }> = {
    service: dark ? { color: '#93c5fd', bg: 'rgba(30,64,175,0.24)', border: 'rgba(96,165,250,0.38)' } : { color: '#1d4ed8', bg: '#eff6ff', border: '#93c5fd' },
    network: dark ? { color: '#67e8f9', bg: 'rgba(14,116,144,0.24)', border: 'rgba(103,232,249,0.38)' } : { color: '#0f766e', bg: '#ecfeff', border: '#99f6e4' },
    workload: dark ? { color: '#86efac', bg: 'rgba(21,128,61,0.24)', border: 'rgba(74,222,128,0.38)' } : { color: '#15803d', bg: '#f0fdf4', border: '#86efac' },
    storage: dark ? { color: '#f9a8d4', bg: 'rgba(157,23,77,0.24)', border: 'rgba(244,114,182,0.38)' } : { color: '#be185d', bg: '#fdf2f8', border: '#f9a8d4' },
    infra: dark ? { color: '#fcd34d', bg: 'rgba(161,98,7,0.24)', border: 'rgba(251,191,36,0.38)' } : { color: '#a16207', bg: '#fffbeb', border: '#fde68a' },
    event: dark ? { color: '#fda4af', bg: 'rgba(190,24,93,0.2)', border: 'rgba(251,113,133,0.34)' } : { color: '#be123c', bg: '#fff1f2', border: '#fecdd3' }
  }

  return map[kind] ?? map.infra
}

const data = computed(() => props.data)
const palette = computed(() => paletteForKind(String(data.value.kind ?? 'infra')))
const headStyle = computed(() => ({
  background: palette.value.bg,
  borderBottom: `1px solid ${palette.value.border}`
}))
const targetHandles = [
  { id: 'target-top', position: Position.Top },
  { id: 'target-right', position: Position.Right },
  { id: 'target-bottom', position: Position.Bottom },
  { id: 'target-left', position: Position.Left }
]
const sourceHandles = [
  { id: 'source-top', position: Position.Top },
  { id: 'source-right', position: Position.Right },
  { id: 'source-bottom', position: Position.Bottom },
  { id: 'source-left', position: Position.Left }
]
const nodeClass = computed(() => [
  'topology-flow-node',
  data.value.group === 'core' ? 'topology-flow-node--core' : '',
  data.value.emphasis === 'primary' ? 'topology-flow-node--primary' : '',
  data.value.dimmed ? 'is-dimmed' : '',
  data.value.active ? 'is-active' : '',
  data.value.selected || props.selected ? 'is-selected' : '',
  data.value.severity === 'warning' ? 'is-warning' : '',
  data.value.severity === 'error' ? 'is-error' : ''
])
const nodeStyle = computed(() => ({
  '--node-width': `${Number(data.value.cardWidth ?? 260)}px`,
  '--node-height': `${Number(data.value.cardHeight ?? 118)}px`
}))

function handleOpen() {
  data.value.onOpen?.()
}
</script>

<style scoped>
.topology-flow-node {
  width: var(--node-width, 260px);
  height: var(--node-height, 118px);
  border-radius: 8px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  background: #ffffff;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  transition: box-shadow 0.18s ease, transform 0.18s ease, border-color 0.18s ease;
}

.topology-flow-node:hover {
  transform: translateY(-1px);
  box-shadow: 0 14px 30px rgba(15, 23, 42, 0.12);
  border-color: rgba(59, 130, 246, 0.26);
}

.topology-flow-node.is-selected {
  border-color: rgba(59, 130, 246, 0.42);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.14), 0 22px 48px rgba(37, 99, 235, 0.16);
}

.topology-flow-node.is-active {
  box-shadow: 0 0 0 2px rgba(14, 165, 233, 0.18), 0 26px 56px rgba(15, 23, 42, 0.18);
}

.topology-flow-node--core {
  border-color: rgba(59, 130, 246, 0.42);
  box-shadow: 0 14px 30px rgba(37, 99, 235, 0.16);
}

.topology-flow-node--primary {
  box-shadow: 0 12px 26px rgba(15, 23, 42, 0.12);
}

.topology-flow-node.is-dimmed {
  transform: scale(0.985);
}

.topology-flow-node.is-warning {
  border-color: rgba(245, 158, 11, 0.42);
}

.topology-flow-node.is-error {
  border-color: rgba(244, 63, 94, 0.48);
}

.topology-flow-node__head {
  padding: 8px 10px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.topology-flow-node__head-main {
  min-width: 0;
  flex: 1 1 auto;
}

.topology-flow-node__badge {
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0;
  text-transform: uppercase;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.topology-flow-node__title {
  margin-top: 4px;
  font-size: 13px;
  font-weight: 800;
  color: #0f172a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.topology-flow-node__status {
  flex: 0 0 auto;
  max-width: 96px;
  padding: 3px 6px;
  border-radius: 8px;
  font-size: 10px;
  font-weight: 700;
  line-height: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  background: rgba(148, 163, 184, 0.18);
  color: #475569;
}

.topology-flow-node__status.is-warning {
  background: rgba(245, 158, 11, 0.18);
  color: #b45309;
}

.topology-flow-node__status.is-error {
  background: rgba(244, 63, 94, 0.16);
  color: #be123c;
}

.topology-flow-node__body {
  padding: 8px 10px 10px;
  min-height: 0;
  flex: 1 1 auto;
}

.topology-flow-node__desc {
  font-size: 11px;
  line-height: 16px;
  color: #334155;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  word-break: break-all;
}

.topology-flow-node__ref {
  margin-top: 6px;
  font-size: 10px;
  color: #64748b;
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.topology-flow-node__handle {
  width: 8px;
  height: 8px;
  opacity: 0;
  border: none;
  background: transparent;
}

:global(html.dark) .topology-flow-node {
  background: rgba(15, 23, 42, 0.995);
  border-color: rgba(148, 163, 184, 0.18);
  box-shadow: 0 18px 40px rgba(2, 6, 23, 0.36);
}

:global(html.dark) .topology-flow-node__title {
  color: #e2e8f0;
}

:global(html.dark) .topology-flow-node__desc {
  color: #cbd5e1;
}

:global(html.dark) .topology-flow-node__ref {
  color: #94a3b8;
}

:global(html.dark) .topology-flow-node__status {
  background: rgba(148, 163, 184, 0.16);
  color: #cbd5e1;
}

:global(html.dark) .topology-flow-node__status.is-warning {
  background: rgba(245, 158, 11, 0.16);
  color: #fbbf24;
}

:global(html.dark) .topology-flow-node__status.is-error {
  background: rgba(244, 63, 94, 0.16);
  color: #fda4af;
}
</style>
