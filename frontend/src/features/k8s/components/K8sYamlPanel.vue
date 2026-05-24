<template>
  <div class="yaml-panel">
    <div class="yaml-toolbar">
      <div class="yaml-toolbar__left">
        <div v-if="meta" class="meta">{{ meta }}</div>
        <div class="yaml-toolbar__status">
          <el-tag size="small" effect="plain" :type="modeTagType">{{ modeText }}</el-tag>
          <el-tag size="small" effect="plain">{{ docStatusText }}</el-tag>
          <el-tag size="small" effect="plain" :type="issueTagType">{{ issueStatusText }}</el-tag>
          <el-tag size="small" effect="plain">{{ associationText }}</el-tag>
          <el-tag v-if="assistHintText" size="small" effect="plain" type="info">{{ assistHintText }}</el-tag>
        </div>
      </div>

      <el-space wrap class="yaml-toolbar__actions">
        <el-tooltip content="复制" placement="top">
          <el-button :disabled="!hasText" :icon="CopyDocument" circle @click="copy" />
        </el-tooltip>
        <el-tooltip content="搜索" placement="top">
          <el-button :disabled="!hasText" :icon="Search" circle @click="openSearch" />
        </el-tooltip>
        <el-tooltip v-if="editable" content="格式化" placement="top">
          <el-button :disabled="!hasText" :icon="EditPen" circle @click="formatYaml" />
        </el-tooltip>
        <el-tooltip v-if="editable" content="智能修正" placement="top">
          <el-button :disabled="!hasText" :icon="MagicStick" circle @click="applySmartFix" />
        </el-tooltip>
        <el-tooltip content="折叠全部" placement="top">
          <el-button :disabled="!hasText" :icon="Fold" circle @click="foldAll" />
        </el-tooltip>
        <el-tooltip content="展开全部" placement="top">
          <el-button :disabled="!hasText" :icon="Expand" circle @click="unfoldAll" />
        </el-tooltip>
        <el-switch v-model="wrap" inline-prompt active-text="换行" inactive-text="单行" />
        <el-switch v-model="lineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
        <el-tooltip :content="editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
          <el-button :icon="editorThemeEffectiveDark ? Sunny : Moon" circle @click="toggleEditorTheme" />
        </el-tooltip>
        <el-tooltip content="刷新" placement="top">
          <el-button :disabled="!refreshable" :loading="loading" :icon="RefreshRight" circle @click="emit('refresh')" />
        </el-tooltip>
        <el-tooltip v-if="saveable" content="保存" placement="top">
          <el-button type="primary" :loading="saving" :icon="Check" circle @click="emit('save')" />
        </el-tooltip>
      </el-space>
    </div>

    <CodeMirrorViewer
      ref="viewerRef"
      :text="text"
      language="yaml"
      :theme="editorTheme"
      :wrap="wrap"
      :line-numbers="lineNumbers"
      :read-only="readOnly"
      :yaml-assist="yamlAssist"
      :height="height"
      class="yaml-panel__editor k8s-detail-box k8s-detail-box--fill"
      @update:text="(value) => emit('update:text', value)"
    />
  </div>
</template>

<script setup lang="ts">
import { Check, CopyDocument, EditPen, Expand, Fold, MagicStick, Moon, RefreshRight, Search, Sunny } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import { analyzeK8sYamlSummary, formatK8sYaml, smartFixK8sYaml, type K8sYamlAssistContext } from '@/shared/components/codeMirrorYamlAssist'
import { copyToClipboard } from '@/shared/utils/text'
import { notifyError, notifySuccess } from '@/shared/utils/notify'

type EditorTheme = 'auto' | 'light' | 'dark'

const props = withDefaults(
  defineProps<{
    text: string
    meta?: string
    loading?: boolean
    saving?: boolean
    height?: string
    readOnly?: boolean
    refreshable?: boolean
    saveable?: boolean
    yamlAssist?: K8sYamlAssistContext | null
  }>(),
  {
    meta: '',
    loading: false,
    saving: false,
    height: '100%',
    readOnly: true,
    refreshable: true,
    saveable: false,
    yamlAssist: null
  }
)

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'save'): void
  (e: 'update:text', value: string): void
}>()

const EDITOR_THEME_KEY = 'k8s-platform:viewer-theme:yaml'

function getStoredTheme(): EditorTheme {
  if (typeof window === 'undefined') return 'auto'
  const stored = window.localStorage.getItem(EDITOR_THEME_KEY)
  return stored === 'light' || stored === 'dark' || stored === 'auto' ? stored : 'auto'
}

const wrap = ref(true)
const lineNumbers = ref(true)
const viewerRef = ref<{ openSearch: () => void; foldAll: () => void; unfoldAll: () => void } | null>(null)
const editorTheme = ref<EditorTheme>(getStoredTheme())

const hasText = computed(() => Boolean(String(props.text ?? '').trim()))
const editable = computed(() => !props.readOnly)
const modeText = computed(() => (editable.value ? '可编辑 YAML' : '只读 YAML'))
const modeTagType = computed<'success' | 'info'>(() => (editable.value ? 'success' : 'info'))
const yamlSummary = computed(() => analyzeK8sYamlSummary(props.text, props.yamlAssist ?? undefined))
const docStatusText = computed(() => (hasText.value ? `文档 ${yamlSummary.value.docsCount}` : '空清单'))
const issueStatusText = computed(() => {
  if (!hasText.value) return '待校验'
  return yamlSummary.value.issueCount > 0 ? `诊断 ${yamlSummary.value.issueCount}` : '已通过基础检查'
})
const issueTagType = computed<'info' | 'success' | 'warning'>(() => {
  if (!hasText.value) return 'info'
  return yamlSummary.value.issueCount > 0 ? 'warning' : 'success'
})
const associationText = computed(() => {
  if (!hasText.value) return yamlSummary.value.association
  if (yamlSummary.value.docsCount > 1 && yamlSummary.value.kinds.length > 1) {
    return `多资源 ${yamlSummary.value.kinds.length} 种`
  }
  return yamlSummary.value.association
})
const assistHintText = computed(() => yamlSummary.value.hints.join(' · '))
const editorThemeEffectiveDark = computed(() => {
  if (editorTheme.value === 'dark') return true
  if (editorTheme.value === 'light') return false
  return document.documentElement.classList.contains('dark')
})

function toggleEditorTheme() {
  editorTheme.value = editorThemeEffectiveDark.value ? 'light' : 'dark'
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(EDITOR_THEME_KEY, editorTheme.value)
  }
}

function openSearch() {
  viewerRef.value?.openSearch()
}

function foldAll() {
  viewerRef.value?.foldAll()
}

function unfoldAll() {
  viewerRef.value?.unfoldAll()
}

function formatYaml() {
  if (!editable.value || !hasText.value) return
  const formatted = formatK8sYaml(props.text)
  if (formatted.error) {
    notifyError(formatted.error)
    return
  }
  if (!formatted.changed) {
    notifySuccess('当前 YAML 已是规范格式')
    return
  }
  emit('update:text', formatted.text)
  notifySuccess('YAML 已格式化')
}

function applySmartFix() {
  if (!editable.value || !hasText.value) return
  const fixed = smartFixK8sYaml(props.text, props.yamlAssist ?? undefined)
  if (fixed.changed) {
    emit('update:text', fixed.text)
  }
  if (fixed.error) {
    notifyError(fixed.error)
    return
  }
  if (fixed.changed) {
    notifySuccess(fixed.notes.slice(0, 2).join('；') || '已完成智能修正')
    return
  }
  notifySuccess('当前 YAML 暂无可自动修正项')
}

async function copy() {
  if (!hasText.value) return
  try {
    await copyToClipboard(props.text)
    notifySuccess('已复制')
  } catch (error) {
    notifyError(error instanceof Error && error.message ? `复制失败：${error.message}` : '复制失败')
  }
}
</script>

<style scoped>
.yaml-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
  height: 100%;
}

.yaml-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 8px 10px;
  border-radius: 10px;
  border: 1px solid rgba(2, 6, 23, 0.06);
  background: rgba(255, 255, 255, 0.88);
  box-shadow: 0 10px 30px rgba(2, 6, 23, 0.06);
}

.yaml-toolbar__left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  min-width: 0;
}

.yaml-toolbar__status {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.yaml-toolbar__actions {
  margin-left: auto;
}

.yaml-panel :deep(.yaml-panel__editor.k8s-detail-box) {
  width: 100%;
  min-width: 0;
  min-height: 0;
  padding: 0;
  overflow: hidden;
  white-space: normal;
}

.yaml-panel :deep(.yaml-panel__editor .cm-host) {
  width: 100%;
  height: 100%;
  min-width: 0;
}

.yaml-panel :deep(.yaml-panel__editor .cm-editor) {
  width: 100%;
}

.meta {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

:global(html.dark) .yaml-toolbar {
  border-color: rgba(226, 232, 240, 0.08);
  background: rgba(2, 6, 23, 0.65);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.25);
}
</style>