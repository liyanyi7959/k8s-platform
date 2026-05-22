<template>
  <el-drawer v-model="visible" class="manifest-apply-drawer" size="82%" destroy-on-close>
    <template #header>
      <div class="manifest-apply-drawer__header">
        <div class="manifest-apply-drawer__heading">
          <div class="manifest-apply-drawer__title">Manifest 工作台</div>
          <el-tooltip placement="bottom-start" :show-after="180">
            <template #content>
              <div class="manifest-apply-drawer__tip">
                支持多文档 YAML 直接创建或更新资源。<br />
                适合临时清单、厂商安装清单和 CRD 实例投递。
              </div>
            </template>
            <button type="button" class="manifest-apply-drawer__hint" aria-label="查看工作台说明">
              <el-icon><QuestionFilled /></el-icon>
            </button>
          </el-tooltip>
        </div>
        <div class="manifest-apply-drawer__actions">
          <input ref="fileInputRef" class="manifest-apply-drawer__file-input" type="file" accept=".yaml,.yml,text/yaml,application/yaml,application/x-yaml" @change="onFileSelected" />
          <el-input v-model="defaultNamespace" clearable class="manifest-apply-drawer__namespace" placeholder="默认命名空间（可选）" />
          <el-switch v-model="dryRun" inline-prompt active-text="DryRun" inactive-text="Apply" />
          <el-button :icon="FolderOpened" @click="triggerFilePick">上传 YAML</el-button>
          <el-button :icon="Delete" :disabled="!yamlText.trim()" @click="resetEditor">清空</el-button>
          <el-button :loading="submitting" :icon="Upload" @click="submit">{{ dryRun ? '执行 Dry Run' : '执行 Apply' }}</el-button>
        </div>
      </div>
    </template>

    <div class="manifest-apply-drawer__body">
      <section class="manifest-apply-drawer__editor">
        <K8sYamlPanel
          :text="yamlText"
          :meta="editorMeta"
          :yaml-assist="yamlAssist"
          :loading="false"
          :saving="submitting"
          :read-only="false"
          :refreshable="false"
          :saveable="false"
          height="calc(100vh - 208px)"
          @update:text="(value) => (yamlText = value)"
        />
      </section>

      <aside class="manifest-apply-drawer__result">
        <div class="manifest-apply-drawer__result-head">
          <div class="manifest-apply-drawer__result-title">执行结果</div>
          <el-tag size="small" :type="dryRun ? 'warning' : 'success'">{{ dryRun ? 'DryRun' : 'Apply' }}</el-tag>
        </div>

        <el-alert
          v-if="!errorText && summaryText"
          :title="summaryText"
          :type="dryRun ? 'warning' : 'success'"
          show-icon
          :closable="false"
          class="manifest-apply-drawer__alert"
        />

        <el-alert
          v-if="errorText"
          :title="errorText"
          type="error"
          show-icon
          :closable="false"
          class="manifest-apply-drawer__alert"
        />

        <el-empty
          v-if="!errorText && results.length === 0"
          description="在左侧编辑或上传 YAML 文件后即可执行。支持多文档 YAML；命名空间资源未写 namespace 时，可使用上方默认命名空间。"
        />

        <el-scrollbar v-else class="manifest-apply-drawer__result-scroll">
          <div class="manifest-apply-drawer__result-list">
            <article v-for="item in results" :key="resultKey(item)" class="manifest-apply-drawer__result-item">
              <div class="manifest-apply-drawer__result-item-head">
                <strong>{{ item.kind }}</strong>
                <el-tag size="small" :type="item.operation === 'create' ? 'success' : 'warning'">{{ item.operation === 'create' ? '已创建' : '已更新' }}</el-tag>
              </div>
              <div class="manifest-apply-drawer__result-item-meta">{{ formatResultMeta(item) }}</div>
            </article>
          </div>
        </el-scrollbar>
      </aside>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Delete, FolderOpened, QuestionFilled, Upload } from '@element-plus/icons-vue'

import K8sYamlPanel from '@/features/k8s/components/K8sYamlPanel.vue'
import { applyManifest, type ManifestApplyResultItem } from '@/features/k8s/api/manifest'
import type { ApiError } from '@/shared/utils/error'
import { notifyError, notifySuccess } from '@/shared/utils/notify'

type ManifestApplyOpenOptions = {
  defaultNamespace?: string
  initialYaml?: string
  sourceLabel?: string
  sourceResource?: string
  workloadKind?: string
}

const props = defineProps<{
  clusterId: number
}>()

const emit = defineEmits<{
  (e: 'recorded'): void
}>()

const visible = ref(false)
const yamlText = ref('')
const defaultNamespace = ref('')
const dryRun = ref(false)
const submitting = ref(false)
const errorText = ref('')
const summaryText = ref('')
const results = ref<ManifestApplyResultItem[]>([])
const sourceLabel = ref('通用 YAML 示例')
const sourceResource = ref('')
const workloadKind = ref('')
const fileInputRef = ref<HTMLInputElement | null>(null)

const editorMeta = computed(() => {
  const ns = defaultNamespace.value.trim()
  const meta = [sourceLabel.value, `cluster=${props.clusterId}`]
  const sourceMeta = [sourceResource.value, workloadKind.value].filter(Boolean).join('/')
  if (ns) meta.push(`defaultNamespace=${ns}`)
  if (sourceMeta) meta.push(`template=${sourceMeta}`)
  return meta.join('  ')
})

const yamlAssist = computed(() => ({
  defaultNamespace: defaultNamespace.value.trim() || undefined,
  sourceResource: sourceResource.value.trim() || undefined,
  workloadKind: workloadKind.value.trim() || undefined
}))

function open(options: ManifestApplyOpenOptions = {}) {
  visible.value = true
  dryRun.value = false
  defaultNamespace.value = String(options.defaultNamespace || '').trim()
  sourceLabel.value = String(options.sourceLabel || '通用 YAML 示例').trim() || '通用 YAML 示例'
  sourceResource.value = String(options.sourceResource || '').trim()
  workloadKind.value = String(options.workloadKind || '').trim()
  yamlText.value = typeof options.initialYaml === 'string' ? options.initialYaml : ''
  errorText.value = ''
  summaryText.value = ''
  results.value = []
}

function close() {
  visible.value = false
}

defineExpose({ open, close })

function resetEditor() {
  yamlText.value = ''
  errorText.value = ''
  summaryText.value = ''
  results.value = []
}

function triggerFilePick() {
  fileInputRef.value?.click()
}

async function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement | null
  const file = input?.files?.[0]
  if (!file) return
  try {
    yamlText.value = await file.text()
    sourceLabel.value = `已导入 ${file.name}`
    errorText.value = ''
    summaryText.value = ''
    results.value = []
    notifySuccess(`已加载 ${file.name}`)
  } catch (error) {
    notifyError(error instanceof Error && error.message ? `读取文件失败：${error.message}` : '读取文件失败')
  } finally {
    if (input) input.value = ''
  }
}

function resultKey(item: ManifestApplyResultItem) {
  return `${item.kind}:${item.namespace || '-'}:${item.name}:${item.operation}`
}

function formatResultMeta(item: ManifestApplyResultItem) {
  const nsText = item.namespace ? `${item.namespace}/` : ''
  return `${nsText}${item.name} · ${item.api_version} · ${item.resource} · ${item.scope}`
}

async function submit() {
  if (!props.clusterId) return
  const text = yamlText.value.trim()
  if (!text) {
    errorText.value = '请先输入 YAML 内容'
    return
  }
  submitting.value = true
  errorText.value = ''
  summaryText.value = ''
  try {
    const result = await applyManifest(props.clusterId, {
      yaml: text,
      default_namespace: defaultNamespace.value.trim() || undefined,
      dry_run: dryRun.value,
      source_label: sourceLabel.value.trim() || undefined,
      source_resource: sourceResource.value.trim() || undefined,
      workload_kind: workloadKind.value.trim() || undefined
    })
    summaryText.value = result.summary || ''
    results.value = result.items ?? []
    notifySuccess(dryRun.value ? 'Dry Run 校验完成，记录已更新' : 'Manifest 已应用，记录已更新')
  } catch (error) {
    const err = error as ApiError
    errorText.value = err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message
    notifyError(errorText.value)
  } finally {
    submitting.value = false
    emit('recorded')
  }
}
</script>

<style scoped>
.manifest-apply-drawer__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  flex: 1 1 auto;
  min-width: 0;
}

.manifest-apply-drawer__heading {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 0 1 auto;
}

.manifest-apply-drawer__title {
  font-size: 18px;
  font-weight: 700;
  color: var(--app-text);
  white-space: nowrap;
}

.manifest-apply-drawer__hint {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 999px;
  background: rgba(248, 250, 252, 0.92);
  color: var(--app-muted);
  cursor: help;
  transition: border-color 0.2s ease, background 0.2s ease, color 0.2s ease;
}

.manifest-apply-drawer__hint:hover {
  border-color: rgba(59, 130, 246, 0.3);
  background: #fff;
  color: var(--el-color-primary);
}

.manifest-apply-drawer__hint:focus-visible {
  outline: none;
  border-color: rgba(59, 130, 246, 0.38);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.12);
}

.manifest-apply-drawer__tip {
  max-width: 320px;
  font-size: 12px;
  line-height: 1.6;
}

.manifest-apply-drawer__actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: nowrap;
  justify-content: flex-end;
  min-width: 0;
  flex: 1 1 auto;
}

.manifest-apply-drawer__file-input {
  display: none;
}

.manifest-apply-drawer__namespace {
  width: 210px;
  flex: 0 0 210px;
}

.manifest-apply-drawer__actions :deep(.el-button) {
  white-space: nowrap;
}

.manifest-apply-drawer__actions :deep(.el-switch) {
  flex: 0 0 auto;
}

.manifest-apply-drawer__body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 16px;
  min-height: calc(100vh - 180px);
}

.manifest-apply-drawer__editor,
.manifest-apply-drawer__result {
  min-height: 0;
}

.manifest-apply-drawer__result {
  border-radius: 16px;
  border: 1px solid rgba(2, 6, 23, 0.08);
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.08);
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.manifest-apply-drawer__result-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.manifest-apply-drawer__result-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--app-text);
}

.manifest-apply-drawer__alert {
  margin-bottom: 4px;
}

.manifest-apply-drawer__result-scroll {
  min-height: 0;
  flex: 1;
}

.manifest-apply-drawer__result-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.manifest-apply-drawer__result-item {
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: linear-gradient(180deg, rgba(248, 250, 252, 0.98), rgba(241, 245, 249, 0.92));
  padding: 12px;
}

.manifest-apply-drawer__result-item-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.manifest-apply-drawer__result-item-meta {
  margin-top: 8px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--app-muted);
  word-break: break-word;
}

:global(html.dark) .manifest-apply-drawer__result {
  border-color: rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.86);
  box-shadow: 0 18px 48px rgba(2, 6, 23, 0.3);
}

:global(html.dark) .manifest-apply-drawer__result-item {
  border-color: rgba(148, 163, 184, 0.18);
  background: linear-gradient(180deg, rgba(30, 41, 59, 0.88), rgba(15, 23, 42, 0.92));
}

:global(html.dark) .manifest-apply-drawer__hint {
  border-color: rgba(148, 163, 184, 0.2);
  background: rgba(15, 23, 42, 0.82);
  color: rgba(226, 232, 240, 0.84);
}

:global(html.dark) .manifest-apply-drawer__hint:hover {
  border-color: rgba(96, 165, 250, 0.4);
  background: rgba(30, 41, 59, 0.94);
  color: rgba(191, 219, 254, 0.96);
}

@media (max-width: 1280px) {
  .manifest-apply-drawer__body {
    grid-template-columns: 1fr;
  }

  .manifest-apply-drawer__header {
    align-items: flex-start;
    flex-direction: column;
  }

  .manifest-apply-drawer__actions {
    width: 100%;
    flex-wrap: wrap;
    justify-content: flex-start;
  }
}
</style>