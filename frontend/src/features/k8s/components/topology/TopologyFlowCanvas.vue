<template>
  <div ref="shellRef" class="topology-flow-shell">
    <div v-if="props.graph.lanes?.length" class="topology-flow-shell__lanes">
      <div
        v-for="lane in props.graph.lanes"
        :key="lane.key"
        class="topology-flow-shell__lane"
        :style="laneStyle(lane)"
      >
        <span class="topology-flow-shell__lane-label">{{ lane.label }}</span>
      </div>
    </div>
    <VueFlow
      v-model:nodes="flowNodes"
      v-model:edges="flowEdges"
      class="topology-vue-flow"
      :node-types="nodeTypes"
      :min-zoom="minZoom"
      :max-zoom="2.2"
      :default-viewport="{ x: 0, y: 0, zoom: 1 }"
      :nodes-draggable="true"
      :elements-selectable="true"
      :zoom-on-scroll="true"
      :zoom-on-pinch="true"
      :pan-on-drag="true"
      @node-drag-stop="handleNodeDragStop"
      @node-click="handleNodeClick"
      @node-mouse-enter="handleNodeMouseEnter"
      @node-mouse-leave="handleNodeMouseLeave"
      @pane-click="clearSelection"
    >
      <MiniMap
        v-if="props.minimapVisible"
        position="bottom-right"
        pannable
        zoomable
        class="topology-vue-flow__minimap"
        :node-color="resolveMiniMapColor"
      />
    </VueFlow>
  </div>
</template>

<script setup lang="ts">
import { computed, markRaw, nextTick, ref, watch } from 'vue'
import { MiniMap } from '@vue-flow/minimap'
import { MarkerType, VueFlow, useVueFlow } from '@vue-flow/core'
import type { Edge, Node } from '@vue-flow/core'

import TopologyFlowNode from './TopologyFlowNode.vue'

import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/minimap/dist/style.css'

type CanvasNodeData = {
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
  targetKind?: string
  targetNamespace?: string
  targetName?: string
  group?: string
  onOpen?: () => void
}

type CanvasGraphNode = {
  id: string
  position: { x: number; y: number }
  data: CanvasNodeData
}

type CanvasGraphEdge = {
  id: string
  source: string
  target: string
  label?: string
  relation?: string
}

type CanvasGraph = {
  nodes: CanvasGraphNode[]
  edges: CanvasGraphEdge[]
  lanes?: Array<{ key: string; label: string; x: number; y: number; width: number; height: number }>
}

type HandleSide = 'top' | 'right' | 'bottom' | 'left'

const props = defineProps<{
  graph: CanvasGraph
  dark: boolean
  viewMode: 'architecture' | 'analysis' | 'anomaly'
  minimapVisible: boolean
}>()

const emit = defineEmits<{
  (e: 'open-node', node: CanvasGraphNode): void
  (e: 'positions-change', payload: Array<{ id: string; position: { x: number; y: number } }>): void
}>()

const nodeTypes = {
  topology: markRaw(TopologyFlowNode)
}

const shellRef = ref<HTMLDivElement | null>(null)
const flowNodes = ref<Node<CanvasNodeData>[]>([])
const flowEdges = ref<Edge[]>([])
const hoveredNodeId = ref<string | null>(null)
const selectedNodeId = ref<string | null>(null)

const { fitView, zoomIn, zoomOut, getViewport, setViewport } = useVueFlow()

const activeNodeId = computed(() => hoveredNodeId.value || selectedNodeId.value)
const minZoom = computed(() => {
  if (props.graph.nodes.length > 160) return 0.05
  if (props.graph.nodes.length > 80) return 0.08
  return 0.12
})

function edgePalette(relation?: string) {
  const normalized = String(relation ?? '').trim().toLowerCase()
  if (normalized.includes('control') || normalized.includes('own')) {
    return { stroke: props.dark ? '#4ade80' : '#16a34a', width: 2.2, dasharray: undefined, animated: false }
  }
  if (normalized.includes('route') || normalized.includes('select')) {
    return { stroke: props.dark ? '#60a5fa' : '#2563eb', width: 2.4, dasharray: undefined, animated: true }
  }
  if (normalized.includes('mount') || normalized.includes('bound') || normalized.includes('attach')) {
    return { stroke: props.dark ? '#f59e0b' : '#d97706', width: 2.3, dasharray: '8 6', animated: true }
  }
  if (normalized.includes('bind') || normalized.includes('identity')) {
    return { stroke: props.dark ? '#22d3ee' : '#0891b2', width: 2, dasharray: '6 6', animated: false }
  }
  if (normalized.includes('report')) {
    return { stroke: props.dark ? '#fb7185' : '#e11d48', width: 1.9, dasharray: '3 6', animated: false }
  }
  return { stroke: props.dark ? '#94a3b8' : '#64748b', width: 1.7, dasharray: undefined, animated: false }
}

function kindColor(kind: string, severity?: string) {
  if (severity === 'error') return props.dark ? '#fb7185' : '#e11d48'
  if (severity === 'warning') return props.dark ? '#f59e0b' : '#d97706'
  const map: Record<string, string> = {
    service: props.dark ? '#60a5fa' : '#2563eb',
    network: props.dark ? '#22d3ee' : '#0891b2',
    workload: props.dark ? '#4ade80' : '#16a34a',
    storage: props.dark ? '#f472b6' : '#db2777',
    infra: props.dark ? '#facc15' : '#ca8a04',
    event: props.dark ? '#fb7185' : '#e11d48'
  }
  return map[kind] ?? map.infra
}

function adjacencyMap() {
  const map = new Map<string, Set<string>>()
  props.graph.edges.forEach((edge) => {
    const source = map.get(edge.source) ?? new Set<string>()
    source.add(edge.target)
    map.set(edge.source, source)
    const target = map.get(edge.target) ?? new Set<string>()
    target.add(edge.source)
    map.set(edge.target, target)
  })
  return map
}

function isNodeFocused(nodeId: string, adjacent: Map<string, Set<string>>) {
  if (!activeNodeId.value) return true
  if (nodeId === activeNodeId.value) return true
  return adjacent.get(activeNodeId.value)?.has(nodeId) ?? false
}

function isAnomalyNode(node: CanvasGraphNode) {
  return !!node.data?.severity && node.data.severity !== 'normal'
}

function resolveMiniMapColor(node: Node<CanvasNodeData>) {
  return kindColor(String(node.data?.kind ?? 'infra'), node.data?.severity)
}

function laneStyle(lane: { key: string; x: number; y: number; width: number; height: number }) {
  const color = kindColor(lane.key === 'control' ? 'workload' : lane.key === 'identity' ? 'network' : lane.key === 'storage' ? 'storage' : lane.key === 'config' ? 'service' : lane.key === 'runtime' ? 'event' : lane.key === 'core' ? 'workload' : 'infra')
  return {
    left: `${lane.x}px`,
    top: `${lane.y}px`,
    width: `${lane.width}px`,
    height: `${lane.height}px`,
    borderColor: color,
    background: color,
    opacity: 0.12
  }
}

function fitPadding() {
  if (props.graph.nodes.length <= 4) return 0.08
  if (props.graph.nodes.length <= 8) return 0.1
  return 0.1
}

function minZoomForGraph(canvasWidth: number) {
  if (props.graph.nodes.length <= 4) return canvasWidth < 480 ? 0.28 : 0.34
  if (props.graph.nodes.length <= 8) return canvasWidth < 480 ? 0.2 : 0.24
  if (props.graph.nodes.length > 160) return 0.05
  if (props.graph.nodes.length > 80) return 0.08
  return 0.12
}

function graphBounds() {
  const bounds = props.graph.nodes.reduce((acc, node) => {
    const width = Number(node.data.cardWidth ?? 260)
    const height = Number(node.data.cardHeight ?? 118)
    const nextMinX = Math.min(acc.minX, node.position.x)
    const nextMinY = Math.min(acc.minY, node.position.y)
    const nextMaxX = Math.max(acc.maxX, node.position.x + width)
    const nextMaxY = Math.max(acc.maxY, node.position.y + height)

    return {
      minX: nextMinX,
      minY: nextMinY,
      maxX: nextMaxX,
      maxY: nextMaxY,
    }
  }, {
    minX: Number.POSITIVE_INFINITY,
    minY: Number.POSITIVE_INFINITY,
    maxX: Number.NEGATIVE_INFINITY,
    maxY: Number.NEGATIVE_INFINITY,
  })

  if (!Number.isFinite(bounds.minX) || !Number.isFinite(bounds.minY) || !Number.isFinite(bounds.maxX) || !Number.isFinite(bounds.maxY)) {
    return null
  }

  return {
    ...bounds,
    width: Math.max(1, bounds.maxX - bounds.minX),
    height: Math.max(1, bounds.maxY - bounds.minY),
  }
}

function nodeWidth(node: CanvasGraphNode) {
  return Number(node.data.cardWidth ?? 260)
}

function nodeHeight(node: CanvasGraphNode) {
  return Number(node.data.cardHeight ?? 118)
}

function nodeCenter(node: CanvasGraphNode) {
  return {
    x: node.position.x + nodeWidth(node) / 2,
    y: node.position.y + nodeHeight(node) / 2
  }
}

function resolveEdgeHandles(edge: CanvasGraphEdge, nodeMap: Map<string, CanvasGraphNode>) {
  const sourceNode = nodeMap.get(edge.source)
  const targetNode = nodeMap.get(edge.target)
  if (!sourceNode || !targetNode) {
    return {
      sourceHandle: 'source-right',
      targetHandle: 'target-left'
    }
  }

  const sourceCenter = nodeCenter(sourceNode)
  const targetCenter = nodeCenter(targetNode)
  const sourceLeft = sourceNode.position.x
  const sourceRight = sourceNode.position.x + nodeWidth(sourceNode)
  const sourceTop = sourceNode.position.y
  const sourceBottom = sourceNode.position.y + nodeHeight(sourceNode)
  const targetLeft = targetNode.position.x
  const targetRight = targetNode.position.x + nodeWidth(targetNode)
  const targetTop = targetNode.position.y
  const targetBottom = targetNode.position.y + nodeHeight(targetNode)
  const dx = targetCenter.x - sourceCenter.x
  const dy = targetCenter.y - sourceCenter.y

  if (targetLeft >= sourceRight + 24) {
    return { sourceHandle: 'source-right', targetHandle: 'target-left' }
  }

  if (sourceLeft >= targetRight + 24) {
    return { sourceHandle: 'source-left', targetHandle: 'target-right' }
  }

  if (targetTop >= sourceBottom + 24) {
    return { sourceHandle: 'source-bottom', targetHandle: 'target-top' }
  }

  if (sourceTop >= targetBottom + 24) {
    return { sourceHandle: 'source-top', targetHandle: 'target-bottom' }
  }

  let sourceSide: HandleSide
  let targetSide: HandleSide
  const horizontalBias = dx >= -Math.max(nodeWidth(sourceNode), nodeWidth(targetNode)) * 0.22

  if (horizontalBias && Math.abs(dx) >= Math.abs(dy) * 0.72) {
    sourceSide = 'right'
    targetSide = 'left'
  } else if (!horizontalBias && Math.abs(dx) > Math.abs(dy) * 1.15) {
    sourceSide = 'left'
    targetSide = 'right'
  } else if (dy >= 0) {
    sourceSide = 'bottom'
    targetSide = 'top'
  } else {
    sourceSide = 'top'
    targetSide = 'bottom'
  }

  return {
    sourceHandle: `source-${sourceSide}`,
    targetHandle: `target-${targetSide}`
  }
}

function buildEdgeLabelPolicy() {
  const counts = new Map<string, number>()
  props.graph.edges.forEach((edge) => {
    const key = String(edge.relation ?? edge.label ?? '').trim().toLowerCase()
    if (!key) return
    counts.set(key, (counts.get(key) ?? 0) + 1)
  })

  const seen = new Map<string, number>()
  return (edge: CanvasGraphEdge) => {
    const key = String(edge.relation ?? edge.label ?? '').trim().toLowerCase()
    if (!key) return edge.label
    const total = counts.get(key) ?? 0
    const index = seen.get(key) ?? 0
    seen.set(key, index + 1)

    if (props.graph.nodes.length > 18 && total > 6 && index >= 2) return undefined
    if (props.graph.nodes.length > 60 && total > 3 && index >= 1) return undefined
    return edge.label
  }
}

function syncGraphToFlow() {
  const adjacent = adjacencyMap()
  const anomalyIds = new Set(props.graph.nodes.filter((node) => isAnomalyNode(node)).map((node) => node.id))
  const nodeMap = new Map(props.graph.nodes.map((node) => [node.id, node]))
  const resolveLabel = buildEdgeLabelPolicy()
  flowNodes.value = props.graph.nodes.map((node) => ({
    id: node.id,
    type: 'topology',
    position: { ...node.position },
    draggable: true,
    selectable: true,
    style: {
      width: `${Number(node.data.cardWidth ?? 260)}px`,
      height: `${Number(node.data.cardHeight ?? 118)}px`,
      zIndex: selectedNodeId.value === node.id ? 30 : node.data.emphasis === 'core' ? 20 : 10,
      opacity:
        props.viewMode === 'anomaly'
          ? (isAnomalyNode(node) || (activeNodeId.value && isNodeFocused(node.id, adjacent)) ? 1 : 0.14)
          : isNodeFocused(node.id, adjacent) ? 1 : 0.28
    },
    data: {
      ...node.data,
      dimmed: !isNodeFocused(node.id, adjacent),
      highlighted: isNodeFocused(node.id, adjacent),
      active: activeNodeId.value === node.id,
      selected: selectedNodeId.value === node.id,
      onOpen: () => emit('open-node', node)
    }
  }))

  flowEdges.value = props.graph.edges.map((edge) => {
    const palette = edgePalette(edge.relation ?? edge.label)
    const { sourceHandle, targetHandle } = resolveEdgeHandles(edge, nodeMap)

    return {
      id: edge.id,
      source: edge.source,
      target: edge.target,
      type: 'smoothstep',
      sourceHandle,
      targetHandle,
      label: resolveLabel(edge),
      class: [`topology-flow-edge`, `topology-flow-edge--${String(edge.relation ?? 'references')}`],
      animated: palette.animated,
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: 18,
        height: 18,
        color: palette.stroke,
      },
      style: {
        stroke: palette.stroke,
        strokeWidth: palette.width,
        strokeDasharray: palette.dasharray,
        strokeOpacity:
          props.viewMode === 'anomaly'
            ? (anomalyIds.has(edge.source) || anomalyIds.has(edge.target) ? 0.95 : 0.08)
            : !activeNodeId.value || edge.source === activeNodeId.value || edge.target === activeNodeId.value ? 1 : 0.12,
        opacity:
          props.viewMode === 'anomaly'
            ? (anomalyIds.has(edge.source) || anomalyIds.has(edge.target) ? 0.95 : 0.08)
            : !activeNodeId.value || edge.source === activeNodeId.value || edge.target === activeNodeId.value ? 1 : 0.12
      },
      labelStyle: {
        fill: props.dark ? '#cbd5e1' : '#334155',
        fontSize: '11px',
        fontWeight: 700,
        opacity: !activeNodeId.value || edge.source === activeNodeId.value || edge.target === activeNodeId.value ? 1 : 0.2
      },
      labelBgStyle: {
        fill: props.dark ? '#1e293b' : '#ffffff',
        fillOpacity: 0.92,
      },
      labelBgPadding: [4, 8] as [number, number],
      labelBgBorderRadius: 6,
      data: {
        relation: edge.relation
      },
      pathOptions: {
        borderRadius: 14,
        offset: 18
      }
    }
  })
}

function handleNodeDragStop(event: { node?: Node<CanvasNodeData> }) {
  if (!event.node) return
  emit('positions-change', [{ id: event.node.id, position: { ...event.node.position } }])
}

function handleNodeClick(event: { node?: Node<CanvasNodeData> }) {
  selectedNodeId.value = event.node?.id ?? null
}

function handleNodeMouseEnter(event: { node?: Node<CanvasNodeData> }) {
  hoveredNodeId.value = event.node?.id ?? null
}

function handleNodeMouseLeave() {
  hoveredNodeId.value = null
}

function clearSelection() {
  selectedNodeId.value = null
  hoveredNodeId.value = null
}

function fitCanvas() {
  requestAnimationFrame(() => {
    const canvas = shellRef.value?.querySelector('.topology-vue-flow') as HTMLDivElement | null
    const bounds = graphBounds()

    if (!canvas || !bounds) {
      void fitView({ padding: fitPadding(), duration: 240 })
      return
    }

    const padding = fitPadding()
    const availableWidth = Math.max(1, canvas.clientWidth * (1 - padding * 2))
    const availableHeight = Math.max(1, canvas.clientHeight * (1 - padding * 2))
    const computedZoom = Math.min(2.2, Math.min(availableWidth / bounds.width, availableHeight / bounds.height))
    const zoom = Math.max(minZoomForGraph(canvas.clientWidth), computedZoom)
    const x = (canvas.clientWidth - bounds.width * zoom) / 2 - bounds.minX * zoom
    const y = (canvas.clientHeight - bounds.height * zoom) / 2 - bounds.minY * zoom

    void setViewport({ x, y, zoom })
  })
}

function zoomCanvasIn() {
  void zoomIn({ duration: 180 })
}

function zoomCanvasOut() {
  void zoomOut({ duration: 180 })
}

watch(() => props.graph, () => {
  if (selectedNodeId.value && !props.graph.nodes.some((node) => node.id === selectedNodeId.value)) {
    selectedNodeId.value = null
  }
  syncGraphToFlow()
}, { immediate: true, deep: true })

watch(
  () => [props.graph.nodes.length, props.graph.edges.length, props.minimapVisible] as const,
  async ([nodeCount, edgeCount, minimapVisible], [prevNodeCount, prevEdgeCount, prevMinimapVisible]) => {
    if (nodeCount === 0) return
    if (nodeCount === prevNodeCount && edgeCount === prevEdgeCount && minimapVisible === prevMinimapVisible) return
    await nextTick()
    fitCanvas()
  },
  { flush: 'post' }
)

watch(() => props.dark, () => {
  syncGraphToFlow()
})

watch([hoveredNodeId, selectedNodeId, () => props.viewMode], () => {
  syncGraphToFlow()
})

defineExpose({
  fitView: fitCanvas,
  zoomIn: zoomCanvasIn,
  zoomOut: zoomCanvasOut,
  getViewport: () => getViewport(),
  setViewport: (v: { x: number; y: number; zoom: number }) => setViewport(v),
})
</script>

<style scoped>

@import '@vue-flow/core/dist/style.css';
@import '@vue-flow/core/dist/theme-default.css';
@import '@vue-flow/minimap/dist/style.css';

.topology-flow-shell {
  min-height: 100%;
  height: 100%;
  padding: 10px 14px 14px;
  position: relative;
  flex: 1 1 auto;
  display: flex;
  overflow: hidden;
}

.topology-vue-flow {
  flex: 1 1 auto;
  min-height: 100%;
  height: 100%;
}

.topology-flow-shell__lanes {
  position: absolute;
  inset: 10px 14px 14px;
  pointer-events: none;
  z-index: 1;
}

.topology-flow-shell__lane {
  position: absolute;
  border: 1px dashed;
  border-radius: 8px;
  backdrop-filter: blur(1px);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.25);
}

.topology-flow-shell__lane-label {
  position: absolute;
  top: 10px;
  left: 14px;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #475569;
  background: rgba(255, 255, 255, 0.82);
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 8px;
  padding: 4px 8px;
}

.topology-vue-flow {
  border-radius: 8px;
  background-color: #f8fafc;
  background-image:
    linear-gradient(rgba(148, 163, 184, 0.12) 1px, transparent 1px),
    linear-gradient(90deg, rgba(148, 163, 184, 0.12) 1px, transparent 1px);
  background-size: 28px 28px;
  position: relative;
  z-index: 2;
}

.topology-vue-flow :deep(.vue-flow__pane) {
  cursor: grab;
}

.topology-vue-flow :deep(.vue-flow__pane.dragging) {
  cursor: grabbing;
}

.topology-vue-flow :deep(.vue-flow__background) {
  opacity: 0.55;
}

.topology-vue-flow :deep(.vue-flow__edge-path),
.topology-vue-flow :deep(.vue-flow__connection-path),
.topology-vue-flow :deep(.vue-flow__edge-interaction) {
  fill: transparent !important;
}

.topology-vue-flow :deep(.vue-flow__edge-path) {
  stroke-linecap: round;
}

.topology-vue-flow :deep(.vue-flow__edge.animated path) {
  animation-duration: 2.8s;
}

.topology-vue-flow :deep(.vue-flow__edge-path[stroke-dasharray='8 6']) {
  stroke-linecap: round;
}

.topology-vue-flow :deep(.vue-flow__edge-path[stroke-dasharray='6 6']) {
  stroke-linecap: round;
}

.topology-vue-flow :deep(.vue-flow__edge-textbg) {
  fill: #ffffff;
  fill-opacity: 0.92;
}

.topology-vue-flow__minimap {
  width: clamp(112px, 9vw, 132px);
  height: clamp(72px, 7vw, 92px);
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 8px;
  box-shadow: 0 12px 24px rgba(15, 23, 42, 0.08);
  overflow: hidden;
  z-index: 5;
}

.topology-vue-flow :deep(.vue-flow__minimap-mask) {
  fill: rgba(255, 255, 255, 0.2);
}

.topology-vue-flow :deep(.vue-flow__viewport) {
  transition: transform 0.22s ease;
}

.topology-vue-flow :deep(.vue-flow__edge-text) {
  pointer-events: none;
}

:global(html.dark) .topology-vue-flow {
  background-color: #0f172a;
  background-image:
    linear-gradient(rgba(148, 163, 184, 0.12) 1px, transparent 1px),
    linear-gradient(90deg, rgba(148, 163, 184, 0.12) 1px, transparent 1px);
}

:global(html.dark) .topology-vue-flow__minimap {
  background: rgba(15, 23, 42, 0.92);
  border-color: rgba(148, 163, 184, 0.16);
}

:global(html.dark) .topology-flow-shell__lane-label {
  color: #cbd5e1;
  background: rgba(15, 23, 42, 0.78);
  border-color: rgba(148, 163, 184, 0.16);
}
</style>
