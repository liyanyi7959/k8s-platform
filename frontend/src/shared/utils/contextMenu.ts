import { computed, reactive, ref } from 'vue'

export function useContextMenu<T>(options?: { width?: number; height?: number; margin?: number }) {
  const width = options?.width ?? 220
  const height = options?.height ?? 260
  const margin = options?.margin ?? 8

  const visible = ref(false)
  const data = ref<T | null>(null)
  const point = reactive<{ x: number; y: number }>({ x: 0, y: 0 })
  const viewportTick = ref(0)

  const style = computed(() => {
    viewportTick.value
    const x = Math.max(margin, Math.min(point.x, window.innerWidth - width - margin))
    const y = Math.max(margin, Math.min(point.y, window.innerHeight - height - margin))
    return { left: `${x}px`, top: `${y}px` }
  })

  function open(e: MouseEvent, payload: T) {
    e.preventDefault()
    data.value = payload
    point.x = e.clientX
    point.y = e.clientY
    visible.value = true
  }

  function close() {
    visible.value = false
    data.value = null
  }

  function bumpViewport() {
    viewportTick.value += 1
  }

  return { visible, data, style, open, close, bumpViewport }
}

