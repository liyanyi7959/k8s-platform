<template>
  <el-dialog v-model="visible" title="部署计划模拟运行" width="90%" min-width="1100px" top="6vh" destroy-on-close @closed="handleClosed">
    <div v-if="loading" v-loading="true" class="dry-run-loading" />
    <template v-else-if="result">
      <header class="dry-run-meta">
        <div class="meta-block">
          <span class="meta-label">计划名称</span>
          <span class="meta-value">{{ result.plan_name }}</span>
        </div>
        <div class="meta-block">
          <span class="meta-label">集群名称</span>
          <span class="meta-value">{{ result.cluster_name }}</span>
        </div>
        <div class="meta-block">
          <span class="meta-label">K8s 版本</span>
          <span class="meta-value">{{ result.k8s_version }}</span>
        </div>
        <div class="meta-block">
          <span class="meta-label">CNI</span>
          <span class="meta-value">{{ result.cni_type }}</span>
        </div>
        <div class="meta-block summary">
          <span class="meta-label">阶段统计</span>
          <span class="meta-value">
            <el-tag v-for="(count, phase) in result.summary" :key="phase" :type="phaseTagType(phase)" class="phase-tag" size="small">
              {{ phaseLabel(phase) }}: {{ count }}
            </el-tag>
          </span>
        </div>
      </header>

      <div class="dry-run-body">
        <aside class="node-panel">
          <div class="node-panel-title">执行机器</div>
          <div
            v-for="node in result.nodes"
            :key="node.server_id"
            class="node-item"
            :class="{ active: activeNodeId === node.server_id }"
            @click="selectNode(node.server_id)"
          >
            <div class="node-name">
              <el-icon><Monitor /></el-icon>
              <span>{{ node.server_name }}</span>
            </div>
            <div class="node-meta">
              <el-tag :type="node.role === 'master' ? 'success' : 'info'" size="small">{{ node.role === 'master' ? 'Master' : 'Worker' }}</el-tag>
              <span class="node-ip">{{ node.ip }}</span>
            </div>
            <div class="node-stats">{{ node.steps.length }} 步骤</div>
          </div>
        </aside>

        <section class="flow-panel">
          <div class="flow-canvas">
            <VueFlow
              v-if="flowNodes.length"
              :nodes="flowNodes"
              :edges="flowEdges"
              :default-viewport="{ x: 0, y: 0, zoom: 0.85 }"
              :nodes-draggable="false"
              :nodes-connectable="false"
              :elements-selectable="true"
              fit-view-on-init
              @node-click="onNodeClick"
            >
              <Background pattern-color="#dcdfe6" :gap="16" />
              <Controls position="bottom-right" :show-interactive="false" />
            </VueFlow>
            <el-empty v-else description="该节点暂无步骤" />
          </div>

          <aside class="step-detail">
            <template v-if="selectedStep">
              <div class="step-header">
                <el-tag :type="phaseTagType(selectedStep.phase)" size="small">{{ phaseLabel(selectedStep.phase) }}</el-tag>
                <h3>{{ selectedStep.title }}</h3>
              </div>
              <p class="step-desc">{{ selectedStep.description }}</p>
              <div class="step-deps" v-if="selectedStep.depends_on?.length">
                <span class="deps-label">依赖步骤：</span>
                <el-tag v-for="dep in selectedStep.depends_on" :key="dep" size="small" effect="plain">{{ dep }}</el-tag>
              </div>
              <div class="step-cmd-title">执行命令</div>
              <pre class="step-cmd"><code>{{ selectedStep.commands.join('\n') }}</code></pre>
            </template>
            <el-empty v-else description="点击左侧流程图节点查看详情" :image-size="80" />
          </aside>
        </section>
      </div>
    </template>
    <el-empty v-else description="暂无数据" />
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Monitor } from '@element-plus/icons-vue'
import { VueFlow, type Edge, type Node, type NodeMouseEvent } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'

import { dryRunDeployPlan, type DryRunResult, type DryRunStep } from '../api/deploy'

const props = defineProps<{ modelValue: boolean; planId: number | null }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const loading = ref(false)
const result = ref<DryRunResult | null>(null)
const activeNodeId = ref<number | null>(null)
const selectedStepKey = ref<string | null>(null)

watch(
  () => [props.modelValue, props.planId] as const,
  async ([open, id]) => {
    if (!open || !id) return
    await loadDryRun(id)
  },
  { immediate: true }
)

async function loadDryRun(id: number) {
  loading.value = true
  result.value = null
  selectedStepKey.value = null
  try {
    const data = await dryRunDeployPlan(id)
    result.value = data
    activeNodeId.value = data.nodes[0]?.server_id ?? null
  } catch (e: any) {
    ElMessage.error(e?.message || '加载模拟运行数据失败')
  } finally {
    loading.value = false
  }
}

function selectNode(serverId: number) {
  activeNodeId.value = serverId
  selectedStepKey.value = null
}

const currentNode = computed(() => result.value?.nodes.find((n) => n.server_id === activeNodeId.value) || null)

const phaseOrder = ['preflight', 'install', 'init', 'join', 'addon', 'finalize']
const phaseColors: Record<string, string> = {
  preflight: '#909399',
  install: '#409EFF',
  init: '#67C23A',
  join: '#E6A23C',
  addon: '#9C27B0',
  finalize: '#F56C6C'
}

function phaseLabel(p: string) {
  const map: Record<string, string> = {
    preflight: '预检', install: '安装', init: '初始化', join: '加入', addon: '附加组件', finalize: '收尾'
  }
  return map[p] || p
}
function phaseTagType(p: string): 'info' | 'primary' | 'success' | 'warning' | 'danger' {
  const map: Record<string, 'info' | 'primary' | 'success' | 'warning' | 'danger'> = {
    preflight: 'info', install: 'primary', init: 'success', join: 'warning', addon: 'primary', finalize: 'danger'
  }
  return map[p] || 'info'
}

const flowNodes = computed<Node[]>(() => {
  const node = currentNode.value
  if (!node) return []
  // 按 phase 分列布局
  const phaseColIndex: Record<string, number> = {}
  phaseOrder.forEach((p, i) => { phaseColIndex[p] = i })
  const colCounter: Record<number, number> = {}
  return node.steps.map((step) => {
    const col = phaseColIndex[step.phase] ?? 99
    colCounter[col] = (colCounter[col] || 0) + 1
    const row = colCounter[col] - 1
    return {
      id: step.key,
      position: { x: col * 220, y: row * 110 },
      data: { label: step.title, step },
      style: {
        background: '#ffffff',
        border: `2px solid ${phaseColors[step.phase] || '#909399'}`,
        borderRadius: '8px',
        padding: '10px 12px',
        width: '190px',
        fontSize: '13px',
        boxShadow: '0 2px 6px rgba(0,0,0,0.06)'
      } as Record<string, string>,
      label: step.title
    } as Node
  })
})

const flowEdges = computed<Edge[]>(() => {
  const node = currentNode.value
  if (!node) return []
  const edges: Edge[] = []
  node.steps.forEach((step) => {
    if (step.depends_on?.length) {
      step.depends_on.forEach((dep) => {
        edges.push({
          id: `${dep}-${step.key}`,
          source: dep,
          target: step.key,
          type: 'smoothstep',
          animated: false,
          style: { stroke: '#a0cfff', strokeWidth: 2 }
        })
      })
    }
  })
  return edges
})

const selectedStep = computed<DryRunStep | null>(() => {
  if (!currentNode.value || !selectedStepKey.value) return null
  return currentNode.value.steps.find((s) => s.key === selectedStepKey.value) || null
})

function onNodeClick(evt: NodeMouseEvent) {
  selectedStepKey.value = evt.node.id
}

function handleClosed() {
  result.value = null
  selectedStepKey.value = null
  activeNodeId.value = null
}
</script>

<style scoped>
.dry-run-loading {
  height: 320px;
}

.dry-run-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 16px 32px;
  padding: 12px 16px;
  background: #f5f7fa;
  border-radius: 8px;
  margin-bottom: 16px;
}
.meta-block {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.meta-block.summary {
  flex: 1;
  min-width: 320px;
}
.meta-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.meta-value {
  font-size: 14px;
  color: var(--el-text-color-primary);
  font-weight: 500;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.phase-tag {
  margin-right: 4px;
}

.dry-run-body {
  display: flex;
  gap: 16px;
  height: 60vh;
}

.node-panel {
  width: 220px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow-y: auto;
  background: #fff;
  flex-shrink: 0;
}
.node-panel-title {
  padding: 12px 14px;
  font-weight: 600;
  font-size: 13px;
  color: var(--el-text-color-primary);
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: #fafafa;
}
.node-item {
  padding: 12px 14px;
  cursor: pointer;
  border-bottom: 1px solid var(--el-border-color-lighter);
  transition: background 0.15s;
}
.node-item:hover {
  background: #f5f7fa;
}
.node-item.active {
  background: #ecf5ff;
  border-left: 3px solid var(--el-color-primary);
  padding-left: 11px;
}
.node-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  margin-bottom: 6px;
}
.node-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.node-ip {
  font-family: ui-monospace, monospace;
}
.node-stats {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.flow-panel {
  flex: 1;
  display: flex;
  gap: 16px;
  min-width: 0;
}
.flow-canvas {
  flex: 1;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: #fafbfc;
  min-width: 0;
  position: relative;
}
.step-detail {
  width: 340px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 16px;
  overflow-y: auto;
  background: #fff;
  flex-shrink: 0;
}
.step-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.step-header h3 {
  margin: 0;
  font-size: 15px;
}
.step-desc {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin: 8px 0 12px;
  line-height: 1.5;
}
.step-deps {
  margin: 8px 0 12px;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}
.deps-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.step-cmd-title {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
  margin-top: 12px;
}
.step-cmd {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.6;
  overflow-x: auto;
  margin: 0;
  font-family: ui-monospace, "SF Mono", Menlo, Consolas, monospace;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 弹窗最小宽度覆盖 */
:deep(.el-dialog) {
  min-width: 1100px;
}
</style>
