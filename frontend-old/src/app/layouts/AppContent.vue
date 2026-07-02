<template>
  <main
    ref="mainEl"
    :class="['min-h-0 flex-1 overflow-y-auto overflow-x-hidden', full ? 'p-0' : 'px-3 py-3 md:px-4 md:py-4 xl:px-5 xl:py-4']"
  >
    <div :class="full ? 'h-full w-full' : 'content-constrained w-full'">
      <slot />
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'

withDefaults(defineProps<{ full?: boolean }>(), { full: false })

const mainEl = ref<HTMLElement | null>(null)
const route = useRoute()

/* 路由切换时自动回到顶部 */
watch(() => route.path, () => {
  mainEl.value?.scrollTo({ top: 0 })
})
</script>

<style scoped>
.content-constrained {
  max-width: var(--content-max-width, 1440px);
  margin-inline: auto;
}
</style>
