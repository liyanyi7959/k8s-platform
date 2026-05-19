<template>
  <el-drawer v-model="visible" class="workload-detail-drawer" :with-header="false" size="72%">
    <div class="workload-detail-root">
      <el-card shadow="never" class="k8s-summary-card">
        <div class="k8s-summary-top">
          <div class="k8s-summary-title">{{ title }}</div>
          <div class="k8s-summary-actions">
            <el-space :size="8">
              <slot name="actions" />
              <el-tooltip content="复制引用" placement="top">
                <el-button :icon="CopyDocument" circle :disabled="!copyTarget" @click="copyRef" />
              </el-tooltip>
              <el-tooltip content="刷新" placement="top">
                <el-button :loading="loading" :icon="RefreshRight" circle @click="$emit('refresh')" />
              </el-tooltip>
              <el-tooltip content="关闭" placement="top">
                <el-button :icon="Close" circle @click="visible = false" />
              </el-tooltip>
            </el-space>
          </div>
        </div>

        <slot name="summary" />
      </el-card>

      <slot />
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Close, CopyDocument, RefreshRight } from '@element-plus/icons-vue'
import { notifyError, notifySuccess } from '@/shared/utils/notify'

const props = defineProps<{
  modelValue: boolean
  title: string
  loading?: boolean
  ns?: string
  name?: string
  refText?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'refresh'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const copyTarget = computed(() => {
  const refText = String(props.refText ?? '').trim()
  if (refText) return refText
  const ns = String(props.ns ?? '').trim()
  const name = String(props.name ?? '').trim()
  if (!name) return ''
  if (!ns) return name
  return `${ns}/${name}`
})

async function copyText(text: string) {
  const v = String(text ?? '')
  if (!v) return
  try {
    await navigator.clipboard.writeText(v)
    notifySuccess('已复制')
  } catch {
    try {
      const ta = document.createElement('textarea')
      ta.value = v
      ta.style.position = 'fixed'
      ta.style.left = '-9999px'
      document.body.appendChild(ta)
      ta.focus()
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
      notifySuccess('已复制')
    } catch {
      notifyError('复制失败')
    }
  }
}

async function copyRef() {
  if (!copyTarget.value) return
  await copyText(copyTarget.value)
}
</script>

<style scoped>
.workload-detail-root {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
  height: 100%;
}
</style>
