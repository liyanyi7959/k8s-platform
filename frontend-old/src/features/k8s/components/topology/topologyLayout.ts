import * as dagre from '@dagrejs/dagre'

export type TopologyLayoutDensity = 'compact' | 'balanced' | 'spacious'
export type TopologyLayoutStrategy = 'pod-centric' | 'service-centric' | 'storage-centric' | 'rbac-centric' | 'overview'

type LayoutNode = {
  id: string
  position: { x: number; y: number }
  data?: {
    kind?: string
    group?: string
    title?: string
    description?: string
    emphasis?: string
    cardWidth?: number
    cardHeight?: number
  }
}

type LayoutEdge = {
  source: string
  target: string
  relation?: string
  label?: string
}

type LayoutGraph<TNode extends LayoutNode, TEdge extends LayoutEdge> = {
  nodes: TNode[]
  edges: TEdge[]
}

type LayoutOptions = {
  mode?: string
  namespaceScope?: boolean
  width?: number
  height?: number
  density?: TopologyLayoutDensity
  strategy?: TopologyLayoutStrategy
  viewMode?: string
}

type DensityProfile = {
  laneGap: number
  laneGapGrowth: number
  verticalGap: number
  verticalCompression: number
  edgeGap: number
  margin: number
  coreWidthBoost: number
  widthScale: number
}

const densityProfiles: Record<TopologyLayoutDensity, DensityProfile> = {
  compact: {
    laneGap: 116,
    laneGapGrowth: 10,
    verticalGap: 30,
    verticalCompression: 10,
    edgeGap: 34,
    margin: 40,
    coreWidthBoost: 48,
    widthScale: 0.92
  },
  balanced: {
    laneGap: 144,
    laneGapGrowth: 14,
    verticalGap: 42,
    verticalCompression: 8,
    edgeGap: 44,
    margin: 54,
    coreWidthBoost: 62,
    widthScale: 1
  },
  spacious: {
    laneGap: 184,
    laneGapGrowth: 18,
    verticalGap: 56,
    verticalCompression: 6,
    edgeGap: 56,
    margin: 68,
    coreWidthBoost: 78,
    widthScale: 1.08
  }
}

const laneByStrategy: Record<TopologyLayoutStrategy, Record<string, number>> = {
  'pod-centric': {
    core: 0,
    control: -1,
    identity: -2,
    runtime: -1,
    network: 1,
    config: 1,
    storage: 2,
    workload: 1,
    service: 1,
    infra: -1,
    event: -1
  },
  'service-centric': {
    core: 0,
    network: -1,
    control: -2,
    identity: -2,
    workload: 1,
    runtime: 2,
    storage: 2,
    config: 1,
    service: 1,
    infra: 2,
    event: 2
  },
  'storage-centric': {
    core: 0,
    storage: 1,
    workload: -1,
    control: -2,
    identity: -1,
    config: -1,
    network: 2,
    runtime: 2,
    service: -1,
    infra: 2,
    event: 2
  },
  'rbac-centric': {
    core: 0,
    identity: -1,
    control: -2,
    network: 1,
    workload: 1,
    config: 1,
    storage: 2,
    runtime: 2,
    service: 1,
    infra: 2,
    event: 2
  },
  overview: {
    core: 0,
    control: -1,
    identity: -1,
    network: 1,
    workload: 0,
    storage: 1,
    config: 1,
    runtime: 2,
    service: 0,
    infra: -2,
    event: 2
  }
}

function inferStrategy(mode?: string, namespaceScope?: boolean): TopologyLayoutStrategy {
  if (mode === 'pod' || mode === 'deployment') return 'pod-centric'
  if (mode === 'service' || mode === 'ingress') return 'service-centric'
  if (mode === 'pvc' || mode === 'pv' || mode === 'node-storage' || mode === 'node') return 'storage-centric'
  if (mode === 'serviceaccount' || mode === 'networkpolicy') return 'rbac-centric'
  if (namespaceScope) return 'overview'
  return 'overview'
}

function relationWeight(relation?: string) {
  const normalized = String(relation ?? '').toLowerCase()
  if (normalized.includes('controls') || normalized.includes('owns')) return 6
  if (normalized.includes('routes') || normalized.includes('selects')) return 5
  if (normalized.includes('mount') || normalized.includes('bound') || normalized.includes('attach')) return 5
  if (normalized.includes('identity') || normalized.includes('bound') || normalized.includes('ref')) return 4
  if (normalized.includes('reports')) return 2
  return 3
}

function measureNode(node: LayoutNode, density: TopologyLayoutDensity, fallbackWidth: number, fallbackHeight: number) {
  const profile = densityProfiles[density]
  const title = String(node.data?.title ?? '')
  const description = String(node.data?.description ?? '')
  const emphasis = String(node.data?.emphasis ?? (node.data?.group === 'core' ? 'core' : 'secondary'))
  const titleWeight = Math.min(18, Math.ceil(title.length / 4))
  const descWeight = Math.min(8, Math.ceil(description.length / 18))
  let width = Math.round((fallbackWidth + titleWeight * 10 + descWeight * 8) * profile.widthScale)
  let height = fallbackHeight + Math.min(20, descWeight * 6)

  if (emphasis === 'core') {
    width += profile.coreWidthBoost
    height += 12
  } else if (emphasis === 'primary') {
    width += Math.round(profile.coreWidthBoost * 0.35)
  }

  const maxWidth = density === 'spacious' ? 360 : density === 'balanced' ? 336 : 310
  const minWidth = density === 'compact' ? 188 : 204
  const maxHeight = density === 'spacious' ? 146 : density === 'balanced' ? 136 : 126
  const minHeight = density === 'compact' ? 92 : 100

  return {
    width: Math.min(maxWidth, Math.max(minWidth, width)),
    height: Math.min(maxHeight, Math.max(minHeight, height))
  }
}

function laneForNode(node: LayoutNode, strategy: TopologyLayoutStrategy) {
  if (String(node.data?.emphasis ?? '') === 'core' || String(node.data?.group ?? '') === 'core') return 0
  const group = String(node.data?.group ?? '').trim()
  if (group) return laneByStrategy[strategy][group] ?? 0

  const kind = String(node.data?.kind ?? 'infra').trim()
  return laneByStrategy[strategy][kind] ?? 0
}

function verticalGapForLane(nodeCount: number, density: TopologyLayoutDensity) {
  const profile = densityProfiles[density]
  return Math.max(18, profile.verticalGap - Math.max(0, nodeCount - 4) * profile.verticalCompression)
}

function laneColumnLimit(nodeCount: number, density: TopologyLayoutDensity) {
  if (nodeCount <= 10) return nodeCount
  if (density === 'compact') return 10
  if (density === 'balanced') return 8
  return 7
}

function layoutSpreadScale(totalNodes: number) {
  if (totalNodes <= 4) return 0.72
  if (totalNodes <= 8) return 0.88
  if (totalNodes <= 16) return 1
  return 1.1
}

function resolveNodeOrder<TNode extends LayoutNode, TEdge extends LayoutEdge>(
  graph: TGraphLike<TNode, TEdge>,
  nodeMetrics: Map<string, { width: number; height: number }>,
  density: TopologyLayoutDensity
) {
  const profile = densityProfiles[density]
  const spreadScale = layoutSpreadScale(graph.nodes.length)
  const dagreGraph = new dagre.graphlib.Graph()
  dagreGraph.setGraph({
    rankdir: 'LR',
    ranksep: Math.max(140, Math.round((profile.laneGap + 50) * spreadScale)),
    nodesep: Math.max(60, Math.round((profile.verticalGap + 36) * spreadScale)),
    edgesep: Math.max(48, Math.round(profile.edgeGap * spreadScale)),
    marginx: profile.margin,
    marginy: profile.margin,
    ranker: 'tight-tree'
  })
  dagreGraph.setDefaultEdgeLabel(() => ({}))

  graph.nodes.forEach((node) => {
    const size = nodeMetrics.get(node.id)
    dagreGraph.setNode(node.id, { width: size?.width ?? 260, height: size?.height ?? 118 })
  })

  graph.edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target, {
      weight: relationWeight(edge.relation ?? edge.label),
      minlen: relationWeight(edge.relation ?? edge.label) >= 5 ? 2 : 1
    })
  })

  dagre.layout(dagreGraph)

  return new Map(
    graph.nodes.map((node) => {
      const positioned = dagreGraph.node(node.id)
      return [node.id, { x: Number(positioned?.x ?? 0), y: Number(positioned?.y ?? 0) }]
    })
  )
}

type TGraphLike<TNode extends LayoutNode, TEdge extends LayoutEdge> = {
  nodes: TNode[]
  edges: TEdge[]
}

function laneGapBetween(
  leftLane: number,
  rightLane: number,
  graph: TGraphLike<LayoutNode, LayoutEdge>,
  laneByNodeId: Map<string, number>,
  density: TopologyLayoutDensity
) {
  const profile = densityProfiles[density]
  const spreadScale = layoutSpreadScale(graph.nodes.length)
  const traffic = graph.edges.filter((edge) => {
    const sourceLane = laneByNodeId.get(edge.source)
    const targetLane = laneByNodeId.get(edge.target)
    if (sourceLane === undefined || targetLane === undefined) return false
    return [sourceLane, targetLane].sort((a, b) => a - b).join(':') === [leftLane, rightLane].sort((a, b) => a - b).join(':')
  }).length

  const baseGap = Math.max(100, Math.round(profile.laneGap * spreadScale))
  const trafficGap = Math.min(100, Math.round(traffic * profile.laneGapGrowth * spreadScale))

  return baseGap + trafficGap
}

type PositionedNodeBox = {
  x: number
  y: number
  width: number
  height: number
}

function layoutClearance(density: TopologyLayoutDensity) {
  if (density === 'compact') return 16
  if (density === 'spacious') return 30
  return 22
}

function boxCenter(box: PositionedNodeBox) {
  return {
    x: box.x + box.width / 2,
    y: box.y + box.height / 2
  }
}

function boxesOverlap(boxA: PositionedNodeBox, boxB: PositionedNodeBox, padding = 0) {
  return (
    boxA.x < boxB.x + boxB.width + padding &&
    boxA.x + boxA.width + padding > boxB.x &&
    boxA.y < boxB.y + boxB.height + padding &&
    boxA.y + boxA.height + padding > boxB.y
  )
}

function resolveNodeCollisions(boxes: Map<string, PositionedNodeBox>, density: TopologyLayoutDensity) {
  const entries = [...boxes.entries()]
  const gap = layoutClearance(density)

  for (let pass = 0; pass < 3; pass += 1) {
    let moved = false

    for (let i = 0; i < entries.length; i += 1) {
      const [, boxA] = entries[i]
      for (let j = i + 1; j < entries.length; j += 1) {
        const [, boxB] = entries[j]
        if (!boxesOverlap(boxA, boxB, gap)) continue

        const overlapX = Math.min(boxA.x + boxA.width, boxB.x + boxB.width) - Math.max(boxA.x, boxB.x)
        const overlapY = Math.min(boxA.y + boxA.height, boxB.y + boxB.height) - Math.max(boxA.y, boxB.y)
        if (overlapX <= 0 || overlapY <= 0) continue

        const centerA = boxCenter(boxA)
        const centerB = boxCenter(boxB)

        if (overlapX < overlapY) {
          const delta = Math.ceil((overlapX + gap) / 2)
          if (centerA.x <= centerB.x) {
            boxA.x -= delta
            boxB.x += delta
          } else {
            boxA.x += delta
            boxB.x -= delta
          }
        } else {
          const delta = Math.ceil((overlapY + gap) / 2)
          if (centerA.y <= centerB.y) {
            boxA.y -= delta
            boxB.y += delta
          } else {
            boxA.y += delta
            boxB.y -= delta
          }
        }

        moved = true
      }
    }

    if (!moved) break
  }
}

function edgeCorridorShift(source: PositionedNodeBox, target: PositionedNodeBox, candidate: PositionedNodeBox, density: TopologyLayoutDensity) {
  const clearance = layoutClearance(density) + 8
  const sourceCenter = boxCenter(source)
  const targetCenter = boxCenter(target)
  const candidateCenter = boxCenter(candidate)
  const horizontalFlow = Math.abs(targetCenter.x - sourceCenter.x) >= Math.abs(targetCenter.y - sourceCenter.y)

  if (horizontalFlow) {
    const minX = Math.min(sourceCenter.x, targetCenter.x)
    const maxX = Math.max(sourceCenter.x, targetCenter.x)
    if (candidate.x >= maxX || candidate.x + candidate.width <= minX) return null

    const lineY = (sourceCenter.y + targetCenter.y) / 2
    const limit = candidate.height / 2 + clearance
    const distance = Math.abs(candidateCenter.y - lineY)
    if (distance >= limit) return null

    return {
      x: 0,
      y: Math.ceil(limit - distance) + (candidateCenter.y <= lineY ? -clearance : clearance)
    }
  }

  const minY = Math.min(sourceCenter.y, targetCenter.y)
  const maxY = Math.max(sourceCenter.y, targetCenter.y)
  if (candidate.y >= maxY || candidate.y + candidate.height <= minY) return null

  const lineX = (sourceCenter.x + targetCenter.x) / 2
  const limit = candidate.width / 2 + clearance
  const distance = Math.abs(candidateCenter.x - lineX)
  if (distance >= limit) return null

  return {
    x: Math.ceil(limit - distance) + (candidateCenter.x <= lineX ? -clearance : clearance),
    y: 0
  }
}

function relieveEdgeOverlaps(boxes: Map<string, PositionedNodeBox>, edges: Array<Pick<LayoutEdge, 'source' | 'target'>>, density: TopologyLayoutDensity) {
  for (let pass = 0; pass < 2; pass += 1) {
    let moved = false

    edges.forEach((edge) => {
      const source = boxes.get(edge.source)
      const target = boxes.get(edge.target)
      if (!source || !target) return

      boxes.forEach((candidate, nodeId) => {
        if (nodeId === edge.source || nodeId === edge.target) return
        const shift = edgeCorridorShift(source, target, candidate, density)
        if (!shift) return
        candidate.x += shift.x
        candidate.y += shift.y
        moved = true
      })
    })

    if (!moved) break
    resolveNodeCollisions(boxes, density)
  }
}

function stabilizeNodeLayout(boxes: Map<string, PositionedNodeBox>, edges: Array<Pick<LayoutEdge, 'source' | 'target'>>, density: TopologyLayoutDensity) {
  resolveNodeCollisions(boxes, density)
  relieveEdgeOverlaps(boxes, edges, density)
  resolveNodeCollisions(boxes, density)
}

export function applyTopologyAutoLayout<
  TNode extends LayoutNode,
  TEdge extends LayoutEdge,
  TGraph extends LayoutGraph<TNode, TEdge>
>(
  graph: TGraph,
  options: LayoutOptions = {}
): TGraph {
  if (graph.nodes.length === 0) return graph

  const density = options.density ?? 'balanced'
  const strategy = options.strategy ?? inferStrategy(options.mode, options.namespaceScope)
  const fallbackWidth = options.width ?? 260
  const fallbackHeight = options.height ?? 118
  const nodeMetrics = new Map<string, { width: number; height: number }>()

  graph.nodes.forEach((node) => {
    nodeMetrics.set(node.id, measureNode(node, density, fallbackWidth, fallbackHeight))
  })

  const dagreOrder = resolveNodeOrder(graph, nodeMetrics, density)

  // Determine how many distinct ranks (x-columns) dagre produced
  const xPositions = new Set([...dagreOrder.values()].map((pos) => Math.round(pos.x / 40)))
  const edgeDensity = graph.edges.length / Math.max(1, graph.nodes.length)
  const useFullDagre = !options.namespaceScope && graph.nodes.length <= 64 && edgeDensity <= 2.6 && (xPositions.size >= 4 || (graph.nodes.length <= 24 && xPositions.size >= 2))

  if (useFullDagre) {
    // Use dagre positions directly for proper tree layout
    const profile = densityProfiles[density]
    const rawPositions = new Map<string, PositionedNodeBox>()

    graph.nodes.forEach((node) => {
      const pos = dagreOrder.get(node.id)
      const size = nodeMetrics.get(node.id)
      if (!pos || !size) return
      rawPositions.set(node.id, {
        x: Math.round(pos.x),
        y: Math.round(pos.y),
        width: size.width,
        height: size.height
      })
    })

    stabilizeNodeLayout(rawPositions, graph.edges, density)

    const minX = Math.min(...[...rawPositions.values()].map((p) => p.x))
    const minY = Math.min(...[...rawPositions.values()].map((p) => p.y))

    return {
      ...graph,
      nodes: graph.nodes.map((node) => {
        const positioned = rawPositions.get(node.id)
        const size = nodeMetrics.get(node.id)
        if (!positioned || !size) return node
        return {
          ...node,
          position: {
            x: Math.round(positioned.x - minX + profile.margin),
            y: Math.round(positioned.y - minY + profile.margin)
          },
          data: {
            ...node.data,
            cardWidth: size.width,
            cardHeight: size.height,
            emphasis: String(node.data?.emphasis ?? (node.data?.group === 'core' ? 'core' : 'secondary'))
          } as TNode['data']
        }
      })
    }
  }

  // Small graph: use lane-based layout for visual consistency
  const laneByNodeId = new Map<string, number>()
  graph.nodes.forEach((node) => {
    laneByNodeId.set(node.id, laneForNode(node, strategy))
  })
  const laneNodes = new Map<number, TNode[]>()
  graph.nodes.forEach((node) => {
    const lane = laneByNodeId.get(node.id) ?? 0
    const items = laneNodes.get(lane) ?? []
    items.push(node)
    laneNodes.set(lane, items)
  })

  const sortedLanes = [...laneNodes.keys()].sort((left, right) => left - right)
  const laneWidths = new Map<number, number>()
  const laneHeights = new Map<number, number>()
  const laneColumnCounts = new Map<number, number>()
  const laneRowsPerColumn = new Map<number, number>()
  const laneColumnWidths = new Map<number, number>()

  sortedLanes.forEach((lane) => {
    const nodesInLane = laneNodes.get(lane) ?? []
    const gap = verticalGapForLane(nodesInLane.length, density)
    const maxNodeWidth = Math.max(...nodesInLane.map((node) => nodeMetrics.get(node.id)?.width ?? fallbackWidth), fallbackWidth)
    const rowsPerColumn = laneColumnLimit(nodesInLane.length, density)
    const columnCount = Math.max(1, Math.ceil(nodesInLane.length / rowsPerColumn))
    const columnGap = Math.max(56, Math.round(densityProfiles[density].edgeGap * 1.45))

    nodesInLane.sort((left, right) => {
      const leftOrder = dagreOrder.get(left.id)?.y ?? 0
      const rightOrder = dagreOrder.get(right.id)?.y ?? 0
      if (leftOrder !== rightOrder) return leftOrder - rightOrder
      return left.id.localeCompare(right.id)
    })

    let totalHeight = fallbackHeight
    for (let columnIndex = 0; columnIndex < columnCount; columnIndex += 1) {
      const columnNodes = nodesInLane.slice(columnIndex * rowsPerColumn, (columnIndex + 1) * rowsPerColumn)
      const columnHeight = columnNodes.reduce((sum, node, index) => sum + (nodeMetrics.get(node.id)?.height ?? fallbackHeight) + (index === 0 ? 0 : gap), 0)
      totalHeight = Math.max(totalHeight, columnHeight)
    }

    laneWidths.set(lane, maxNodeWidth * columnCount + columnGap * Math.max(0, columnCount - 1))
    laneHeights.set(lane, totalHeight)
    laneColumnCounts.set(lane, columnCount)
    laneRowsPerColumn.set(lane, rowsPerColumn)
    laneColumnWidths.set(lane, maxNodeWidth)
  })

  const laneCenters = new Map<number, number>()
  laneCenters.set(0, 0)
  for (let index = 1; index < sortedLanes.length; index += 1) {
    const lane = sortedLanes[index]
    const prevLane = sortedLanes[index - 1]
    const prevCenter = laneCenters.get(prevLane) ?? 0
    const prevWidth = laneWidths.get(prevLane) ?? fallbackWidth
    const nextWidth = laneWidths.get(lane) ?? fallbackWidth
    const gap = laneGapBetween(prevLane, lane, graph, laneByNodeId, density)
    laneCenters.set(lane, prevCenter + prevWidth / 2 + gap + nextWidth / 2)
  }

  for (let index = sortedLanes.indexOf(0) - 1; index >= 0; index -= 1) {
    const lane = sortedLanes[index]
    const nextLane = sortedLanes[index + 1]
    const nextCenter = laneCenters.get(nextLane) ?? 0
    const currentWidth = laneWidths.get(lane) ?? fallbackWidth
    const nextWidth = laneWidths.get(nextLane) ?? fallbackWidth
    const gap = laneGapBetween(lane, nextLane, graph, laneByNodeId, density)
    laneCenters.set(lane, nextCenter - nextWidth / 2 - gap - currentWidth / 2)
  }

  const canvasHeight = Math.max(...sortedLanes.map((lane) => laneHeights.get(lane) ?? 0), fallbackHeight)
  const profile = densityProfiles[density]
  const rawPositions = new Map<string, { x: number; y: number; width: number; height: number }>()

  sortedLanes.forEach((lane) => {
    const nodesInLane = laneNodes.get(lane) ?? []
    const gap = verticalGapForLane(nodesInLane.length, density)
    const centerX = laneCenters.get(lane) ?? 0
    const columnCount = laneColumnCounts.get(lane) ?? 1
    const rowsPerColumn = laneRowsPerColumn.get(lane) ?? Math.max(1, nodesInLane.length)
    const columnWidth = laneColumnWidths.get(lane) ?? fallbackWidth
    const columnGap = Math.max(56, Math.round(densityProfiles[density].edgeGap * 1.45))
    const totalLaneWidth = laneWidths.get(lane) ?? fallbackWidth
    const laneLeft = centerX - totalLaneWidth / 2

    nodesInLane.forEach((node, index) => {
      const size = nodeMetrics.get(node.id) ?? { width: fallbackWidth, height: fallbackHeight }
      const columnIndex = Math.min(columnCount - 1, Math.floor(index / rowsPerColumn))
      const rowIndex = index % rowsPerColumn
      const columnNodes = nodesInLane.slice(columnIndex * rowsPerColumn, (columnIndex + 1) * rowsPerColumn)
      const columnHeight = columnNodes.reduce((sum, item, itemIndex) => {
        const itemSize = nodeMetrics.get(item.id) ?? { width: fallbackWidth, height: fallbackHeight }
        return sum + itemSize.height + (itemIndex === 0 ? 0 : gap)
      }, 0)
      let cursorY = Math.max(profile.margin, Math.round((canvasHeight - columnHeight) / 2) + profile.margin)
      for (let row = 0; row < rowIndex; row += 1) {
        const item = columnNodes[row]
        const itemSize = item ? (nodeMetrics.get(item.id) ?? { width: fallbackWidth, height: fallbackHeight }) : { width: fallbackWidth, height: fallbackHeight }
        cursorY += itemSize.height + gap
      }

      rawPositions.set(node.id, {
        x: Math.round(laneLeft + columnIndex * (columnWidth + columnGap) + (columnWidth - size.width) / 2),
        y: Math.round(cursorY),
        width: size.width,
        height: size.height
      })
    })
  })

  stabilizeNodeLayout(rawPositions, graph.edges, density)

  const minX = Math.min(...[...rawPositions.values()].map((value) => value.x), 0)
  const minY = Math.min(...[...rawPositions.values()].map((value) => value.y), 0)

  return {
    ...graph,
    nodes: graph.nodes.map((node) => {
      const positioned = rawPositions.get(node.id)
      const size = nodeMetrics.get(node.id)
      if (!positioned || !size) return node
      return {
        ...node,
        position: {
          x: positioned.x - minX + profile.margin,
          y: positioned.y - minY + profile.margin
        },
        data: {
          ...node.data,
          cardWidth: size.width,
          cardHeight: size.height,
          emphasis: String(node.data?.emphasis ?? (node.data?.group === 'core' ? 'core' : 'secondary'))
        } as TNode['data']
      }
    })
  }
}
