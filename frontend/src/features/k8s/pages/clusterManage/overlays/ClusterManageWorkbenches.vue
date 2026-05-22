<template>
  <el-dialog v-model="createNamespaceVisible" title="创建 Namespace" class="dialog-sm">
    <el-form label-width="120px">
      <el-form-item label="名称">
        <el-input v-model="createNamespaceName" placeholder="例如：dev" clearable />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-space>
        <el-button @click="createNamespaceVisible = false">取消</el-button>
        <el-button
          type="primary"
          :disabled="!props.clusterId || !createNamespaceName.trim()"
          :loading="creatingNamespace"
          :icon="Plus"
          @click="doCreateNamespace"
        >创建</el-button>
      </el-space>
    </template>
  </el-dialog>

  <el-drawer
    v-model="execVisible"
    title="PodShell"
    size="70%"
    destroy-on-close
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :before-close="beforeCloseExec"
  >
    <div class="pod-term-toolbar">
      <div class="pod-term-meta">{{ execMeta }}</div>
      <el-space wrap size="6">
        <el-select v-model="execContainer" placeholder="container" class="w-input-md" :disabled="execContainers.length <= 1">
          <el-option v-for="container in execContainers" :key="container" :label="container" :value="container" />
        </el-select>
        <el-input v-model="execCommand" placeholder="例如：sh 或 ls -al" class="w-input-lg" @keyup.enter="connectTerminal" />

        <el-tag v-if="terminalConnected" type="success">已连接</el-tag>
        <el-tag v-else-if="terminalConnecting" type="info">连接中</el-tag>
        <el-tag v-else type="danger">未连接</el-tag>

        <el-button size="small" :icon="terminalTheme === 'dark' ? Sunny : Moon" @click="toggleTerminalTheme">
          {{ terminalTheme === 'dark' ? '浅色' : '深色' }}
        </el-button>

        <el-button v-if="!terminalConnected" size="small" :loading="terminalConnecting" :icon="Link" @click="connectTerminal">连接</el-button>
        <el-button v-else size="small" :loading="terminalConnecting" :icon="RefreshRight" @click="reconnectTerminal">重连</el-button>

        <el-button size="small" :icon="PowerSwitchIcon" @click="disconnectTerminal">断开</el-button>
        <el-button size="small" :disabled="!terminalConnected" @click="sendTerminalCtrlC">Ctrl+C</el-button>
        <el-button size="small" :icon="Delete" @click="clearTerminal">清屏</el-button>
      </el-space>
    </div>

    <div class="pod-exec-terminal-host">
      <PodExecTerminal
        v-if="execVisible && execTarget"
        ref="podExecTerminalRef"
        :cluster-id="props.clusterId"
        :namespace="execTarget.ns"
        :pod="execTarget.name"
        :container="execContainer || undefined"
        :command="execCommandArgs"
        :theme="terminalTheme"
        :auto-connect="false"
        @status="onPodExecStatus"
      />
    </div>
  </el-drawer>

  <PodLogDrawer ref="podLogDrawerRef" :cluster-id="props.clusterId" />
  <MultiPodLogDrawer ref="multiPodLogDrawerRef" :cluster-id="props.clusterId" />
  <ManifestApplyDrawer ref="manifestApplyDrawerRef" :cluster-id="props.clusterId" @recorded="emit('manifest-recorded')" />
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { Delete, Link, Moon, Plus, RefreshRight, Sunny } from '@element-plus/icons-vue'

import * as k8sApi from '@/features/k8s/api/k8s'
import PodExecTerminal from '@/features/k8s/components/PodExecTerminal.vue'
import ManifestApplyDrawer from '@/features/k8s/pages/clusterManage/overlays/ManifestApplyDrawer.vue'
import MultiPodLogDrawer from '@/features/k8s/pages/clusterManage/overlays/MultiPodLogDrawer.vue'
import PodLogDrawer from '@/features/k8s/pages/clusterManage/overlays/PodLogDrawer.vue'
import { getRowNamespace } from '@/features/k8s/pages/ClusterManageView.utils'
import { PowerSwitchIcon } from '@/shared/icons/appIcons'
import type { ApiError } from '@/shared/utils/error'
import { notifyError, notifySuccess } from '@/shared/utils/notify'

const props = defineProps<{
  clusterId: number
  editorTheme: 'auto' | 'light' | 'dark'
  editorThemeEffectiveDark: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle-editor-theme'): void
  (e: 'namespace-created'): void
  (e: 'manifest-recorded'): void
}>()

type TerminalTheme = 'dark' | 'light'
type PodRow = { metadata?: { name?: unknown; namespace?: unknown }; spec?: { containers?: Array<{ name?: unknown }> } }
type LogsTarget = { ns: string; name: string; container?: string; containers?: string[] }
type ManifestApplyOpenOptions = {
  defaultNamespace?: string
  initialYaml?: string
  sourceLabel?: string
  sourceResource?: string
  workloadKind?: string
}

const TERMINAL_THEME_KEY = 'k8s-platform:pod-exec-terminal-theme'
const terminalTheme = ref<TerminalTheme>((localStorage.getItem(TERMINAL_THEME_KEY) as TerminalTheme) || 'dark')

const createNamespaceVisible = ref(false)
const creatingNamespace = ref(false)
const createNamespaceName = ref('')

const execVisible = ref(false)
const execCommand = ref('sh')
const execTarget = ref<{ ns: string; name: string } | null>(null)
const execContainers = ref<string[]>([])
const execContainer = ref('')
const terminalConnected = ref(false)
const terminalConnecting = ref(false)
const podExecTerminalRef = ref<InstanceType<typeof PodExecTerminal> | null>(null)
const podLogDrawerRef = ref<InstanceType<typeof PodLogDrawer> | null>(null)
const multiPodLogDrawerRef = ref<InstanceType<typeof MultiPodLogDrawer> | null>(null)
const manifestApplyDrawerRef = ref<InstanceType<typeof ManifestApplyDrawer> | null>(null)

function toggleTerminalTheme() {
  terminalTheme.value = terminalTheme.value === 'dark' ? 'light' : 'dark'
  localStorage.setItem(TERMINAL_THEME_KEY, terminalTheme.value)
}

const execCommandArgs = computed(() => {
  const value = execCommand.value.trim().split(/\s+/).filter(Boolean)
  return value.length ? value : ['bash']
})

const execMeta = computed(() => {
  if (!props.clusterId || !execTarget.value) return ''
  return `cluster=${props.clusterId}  ${execTarget.value.ns}/${execTarget.value.name}`
})

async function connectTerminal() {
  if (!execVisible.value) return
  if (terminalConnected.value) {
    void podExecTerminalRef.value?.reconnect()
    return
  }
  void podExecTerminalRef.value?.connect()
}

function reconnectTerminal() {
  void podExecTerminalRef.value?.reconnect()
}

function disconnectTerminal() {
  podExecTerminalRef.value?.disconnect()
  execVisible.value = false
}

function beforeCloseExec(done: () => void) {
  podExecTerminalRef.value?.disconnect()
  done()
}

function onPodExecStatus(value: { connected: boolean; connecting: boolean }) {
  terminalConnected.value = value.connected
  terminalConnecting.value = value.connecting
}

function clearTerminal() {
  podExecTerminalRef.value?.clear()
}

function sendTerminalCtrlC() {
  podExecTerminalRef.value?.sendCtrlC()
}

function openCreateNamespace() {
  createNamespaceName.value = ''
  createNamespaceVisible.value = true
}

async function doCreateNamespace() {
  if (!props.clusterId) return
  const name = createNamespaceName.value.trim()
  if (!name) return
  creatingNamespace.value = true
  try {
    await k8sApi.createNamespace(props.clusterId, name)
    notifySuccess('已创建')
    createNamespaceVisible.value = false
    emit('namespace-created')
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    creatingNamespace.value = false
  }
}

function openPodExec(row: PodRow, container?: string) {
  const ns = getRowNamespace(row)
  if (!props.clusterId || !ns) return
  const name = String(row?.metadata?.name ?? '')
  if (!name) return
  execTarget.value = { ns, name }
  execContainers.value = (row?.spec?.containers ?? []).map((item) => String(item?.name ?? '')).filter(Boolean)
  execContainer.value = container && execContainers.value.includes(container) ? container : execContainers.value[0] ?? ''
  execCommand.value = 'bash'
  execVisible.value = true
  void nextTick().then(() => {
    clearTerminal()
    reconnectTerminal()
    podExecTerminalRef.value?.focus()
  })
}

function openPodLogs(row: PodRow) {
  const ns = getRowNamespace(row)
  if (!ns) return
  const name = String(row?.metadata?.name ?? '')
  if (!name) return
  const containers = Array.isArray(row?.spec?.containers) ? row.spec.containers.map((item) => String(item?.name ?? '')).filter(Boolean) : []
  podLogDrawerRef.value?.open({ ns, name, containers })
}

function openPodLogsTarget(target: LogsTarget) {
  const containers = target.container ? [target.container] : []
  podLogDrawerRef.value?.open({ ...target, containers })
}

function openMultiPodLogs(targets: LogsTarget[]) {
  multiPodLogDrawerRef.value?.open(targets)
}

function openManifestApply(options: ManifestApplyOpenOptions = {}) {
  manifestApplyDrawerRef.value?.open(options)
}

defineExpose({
  openCreateNamespace,
  openPodExec,
  openPodLogs,
  openPodLogsTarget,
  openMultiPodLogs,
  openManifestApply
})
</script>

<style scoped>
.dialog-sm :deep(.el-dialog) {
  width: min(520px, calc(100vw - 32px));
}

.pod-term-toolbar {
  margin-bottom: 10px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}

.pod-term-meta {
  font-size: 12px;
  color: var(--app-muted);
  padding-top: 6px;
}

.pod-exec-terminal-host {
  height: 62vh;
  border-radius: 12px;
  overflow: hidden;
}
</style>
