<template>
  <el-drawer v-model="visible" :title="drawerTitle" :size="size">
    <K8sYamlPanel
      :meta="meta"
      :text="viewText"
      :loading="loading"
      :saving="saving"
      :read-only="readOnly"
      :refreshable="Boolean(loader)"
      :saveable="Boolean(saver)"
      height="calc(100vh - 170px)"
      @refresh="load"
      @save="save"
      @update:text="(value) => (text = value)"
    />
  </el-drawer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import K8sYamlPanel from '@/features/k8s/components/K8sYamlPanel.vue'
import { useK8sYamlDrawer, type K8sYamlLoader, type K8sYamlSaver } from '@/features/k8s/composables/useK8sYamlDrawer'

const props = withDefaults(
  defineProps<{
    title?: string
    size?: string
  }>(),
  { title: '', size: '56%' }
)

const { visible, meta, loader, saver, loading, saving, text, viewText, readOnly, open, load, save } = useK8sYamlDrawer()
const drawerTitle = computed(() => props.title || (readOnly.value ? '查看 YAML' : '编辑 YAML'))

defineExpose<{ open: (meta: string, loader: K8sYamlLoader, saver?: K8sYamlSaver) => void }>({ open })
</script>
