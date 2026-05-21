<template>
  <div class="yaml-panel">
    <div class="yaml-toolbar">
      <el-space wrap>
        <div v-if="meta" class="meta">{{ meta }}</div>
        <el-tooltip content="复制" placement="top">
          <el-button :disabled="!hasText" :icon="CopyDocument" circle @click="copy" />
        </el-tooltip>
        <el-tooltip content="搜索" placement="top">
          <el-button :disabled="!hasText" :icon="Search" circle @click="openSearch" />
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
      :height="height"
      class="k8s-detail-box k8s-detail-box--fill"
      @update:text="(value) => emit('update:text', value)"
    />
  </div>
</template>

<script setup lang="ts">
import { Check, CopyDocument, Expand, Fold, Moon, RefreshRight, Search, Sunny } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
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
  }>(),
  {
    meta: '',
    loading: false,
    saving: false,
    height: '100%',
    readOnly: true,
    refreshable: true,
    saveable: false
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
  padding: 8px 10px;
  border-radius: 10px;
  border: 1px solid rgba(2, 6, 23, 0.06);
  background: rgba(255, 255, 255, 0.88);
  box-shadow: 0 10px 30px rgba(2, 6, 23, 0.06);
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