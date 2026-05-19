<template>
  <el-tooltip :content="tooltip" placement="top" :show-after="showAfter" :disabled="!tooltip">
    <button
      type="button"
      :class="buttonClass"
      :disabled="disabled || loading"
      :aria-label="tooltip"
      @click="emit('click')"
    >
      <el-icon v-if="loading"><Loading /></el-icon>
      <component :is="icon" v-else />
    </button>
  </el-tooltip>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Component } from 'vue'
import { Loading } from '@element-plus/icons-vue'

const props = withDefaults(defineProps<{
  icon: Component
  tooltip: string
  variant?: 'info' | 'edit' | 'cyan' | 'danger' | 'warn' | 'success' | 'violet' | 'more' | 'copy'
  size?: 'default' | 'toolbar'
  disabled?: boolean
  loading?: boolean
  showAfter?: number
}>(), {
  variant: 'info',
  size: 'default',
  disabled: false,
  loading: false,
  showAfter: 300
})

const emit = defineEmits<{
  (e: 'click'): void
}>()

const buttonClass = computed(() => [
  'k8s-act-btn',
  `k8s-act-btn--${props.variant}`,
  props.size === 'toolbar' ? 'k8s-act-btn--toolbar' : ''
])
</script>
