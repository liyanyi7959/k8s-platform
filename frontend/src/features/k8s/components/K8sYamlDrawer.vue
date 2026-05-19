<template>
  <el-drawer v-model="visible" :title="title" :size="size">
    <div class="yaml-toolbar">
      <el-space>
        <div class="meta">{{ meta }}</div>
        <el-tooltip content="复制" placement="top">
          <el-button :disabled="!viewText" :icon="CopyDocument" circle @click="copy" />
        </el-tooltip>
        <el-tooltip content="搜索" placement="top">
          <el-button :disabled="!viewText" :icon="Search" circle @click="openSearch" />
        </el-tooltip>
        <el-tooltip content="折叠全部" placement="top">
          <el-button :disabled="!viewText" :icon="Fold" circle @click="foldAll" />
        </el-tooltip>
        <el-tooltip content="展开全部" placement="top">
          <el-button :disabled="!viewText" :icon="Expand" circle @click="unfoldAll" />
        </el-tooltip>
        <el-switch v-model="wrap" inline-prompt active-text="换行" inactive-text="单行" />
        <el-switch v-model="lineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
        <el-tooltip :content="editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
          <el-button :icon="editorThemeEffectiveDark ? Sunny : Moon" circle @click="toggleEditorTheme" />
        </el-tooltip>
        <el-tooltip content="刷新" placement="top">
          <el-button :disabled="!loader" :loading="loading" :icon="RefreshRight" circle @click="load" />
        </el-tooltip>
        <el-tooltip v-if="saver" content="保存" placement="top">
          <el-button type="primary" :loading="saving" :icon="Check" circle @click="save" />
        </el-tooltip>
      </el-space>
    </div>
    <CodeMirrorViewer
      ref="viewerRef"
      :text="viewText"
      language="yaml"
      :theme="editorTheme"
      :wrap="wrap"
      :line-numbers="lineNumbers"
      :read-only="readOnly"
      height="calc(100vh - 170px)"
      class="k8s-detail-box k8s-detail-box--fill"
      @update:text="(v) => (text = v)"
    />
  </el-drawer>
</template>

<script setup lang="ts">
import { CopyDocument, Expand, Fold, Moon, RefreshRight, Search, Sunny, Check } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'
import { useK8sYamlDrawer, type K8sYamlLoader, type K8sYamlSaver } from '@/features/k8s/composables/useK8sYamlDrawer'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'

withDefaults(
  defineProps<{
    title?: string
    size?: string
  }>(),
  { title: 'YAML', size: '56%' }
)

const { visible, meta, loader, saver, loading, saving, text, viewText, readOnly, open, load, save, copy } = useK8sYamlDrawer()
const wrap = ref(true)
const lineNumbers = ref(true)
const viewerRef = ref<{ openSearch: () => void; foldAll: () => void; unfoldAll: () => void } | null>(null)

type EditorTheme = 'auto' | 'light' | 'dark'
const EDITOR_THEME_KEY = 'k8s-platform:viewer-theme:yaml'
const editorTheme = ref<EditorTheme>((localStorage.getItem(EDITOR_THEME_KEY) as EditorTheme) || 'auto')

const editorThemeEffectiveDark = computed(() => {
  if (editorTheme.value === 'dark') return true
  if (editorTheme.value === 'light') return false
  return document.documentElement.classList.contains('dark')
})

function toggleEditorTheme() {
  editorTheme.value = editorThemeEffectiveDark.value ? 'light' : 'dark'
  localStorage.setItem(EDITOR_THEME_KEY, editorTheme.value)
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

defineExpose<{ open: (meta: string, loader: K8sYamlLoader, saver?: K8sYamlSaver) => void }>({ open })
</script>

<style scoped>
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

:global(html.dark) .yaml-toolbar {
  border-color: rgba(226, 232, 240, 0.08);
  background: rgba(2, 6, 23, 0.65);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.25);
}
</style>
