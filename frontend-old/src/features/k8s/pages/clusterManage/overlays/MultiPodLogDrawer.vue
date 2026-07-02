<template>
  <el-drawer
    v-model="visible"
    class="multi-pod-log-drawer"
    size="94%"
    destroy-on-close
    :with-header="false"
    :close-on-click-modal="false"
  >
    <MultiPodLogWorkbench
      ref="workbenchRef"
      :cluster-id="props.clusterId"
      show-close-button
      @request-close="close"
    />
  </el-drawer>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'

import MultiPodLogWorkbench from './MultiPodLogWorkbench.vue'

const props = defineProps<{ clusterId: number }>()

type MultiPodLogTarget = {
  ns: string
  name: string
  container?: string
  containers?: string[]
}

const visible = ref(false)
const workbenchRef = ref<InstanceType<typeof MultiPodLogWorkbench> | null>(null)

function open(nextTargets: MultiPodLogTarget[]) {
  visible.value = true
  void nextTick(() => {
    workbenchRef.value?.open(nextTargets)
  })
}

function close() {
  visible.value = false
}

defineExpose({ open, close })
</script>

<style scoped>
.multi-pod-log-drawer :deep(.el-drawer__body) {
  padding: 0;
}
</style>