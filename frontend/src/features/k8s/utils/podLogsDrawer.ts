import { nextTick, onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { lockPageScroll, unlockPageScroll } from '@/shared/utils/scrollLock'

export type PodLogsTarget = { ns: string; name: string; container?: string }

export function usePodLogsDrawer(opts: {
  clusterId: Ref<number | undefined>
  fetchLogs: (args: { clusterId: number; target: PodLogsTarget; tailLines: number; signal: AbortSignal }) => Promise<{ text: string }>
  onError: (e: unknown) => void
}) {
  const visible = ref(false)
  const loading = ref(false)
  const text = ref('')
  const tailLines = ref(200)
  const target = ref<PodLogsTarget | null>(null)
  const boxRef = ref<HTMLElement | null>(null)
  const live = ref(true)
  const wrap = ref(false)

  let timer: number | null = null
  let controller: AbortController | null = null
  let reqSeq = 0

  function stopTimer() {
    if (timer != null) {
      window.clearInterval(timer)
      timer = null
    }
  }

  function startTimer() {
    stopTimer()
    if (!visible.value || !live.value) return
    timer = window.setInterval(() => {
      if (!visible.value || !live.value) return
      void refresh({ force: false })
    }, 2000)
  }

  function scrollToBottom() {
    void nextTick(() => {
      const el = boxRef.value
      if (!el) return
      el.scrollTop = el.scrollHeight
    })
  }

  async function refresh(refreshOpts?: { force?: boolean }) {
    const cid = opts.clusterId.value
    const t = target.value
    if (!cid || !t) return

    const force = refreshOpts?.force ?? true
    if (!force && loading.value) return

    const seq = (reqSeq += 1)
    if (force) controller?.abort()
    controller = new AbortController()
    const signal = controller.signal

    loading.value = true
    try {
      const data = await opts.fetchLogs({ clusterId: cid, target: t, tailLines: tailLines.value, signal })
      if (seq !== reqSeq) return
      text.value = data.text ?? ''
      if (live.value) scrollToBottom()
    } catch (e) {
      if ((e as any)?.name === 'CanceledError' || (e as any)?.code === 'ERR_CANCELED') return
      if ((e as any)?.name === 'AbortError') return
      if (seq !== reqSeq) return
      opts.onError(e)
    } finally {
      if (seq === reqSeq) loading.value = false
    }
  }

  function open(next: PodLogsTarget) {
    target.value = next
    visible.value = true
    live.value = true
    void refresh({ force: true })
  }

  function abortInFlight() {
    controller?.abort()
    controller = null
  }

  watch(visible, (v) => {
    if (v) {
      lockPageScroll()
      startTimer()
    } else {
      stopTimer()
      abortInFlight()
      unlockPageScroll()
    }
  })

  watch(live, (v) => {
    if (!visible.value) return
    if (v) {
      void refresh({ force: true })
      startTimer()
    } else {
      stopTimer()
    }
  })

  watch([tailLines, () => target.value?.ns, () => target.value?.name, () => target.value?.container, () => opts.clusterId.value], () => {
    if (!visible.value) return
    void refresh({ force: true })
  })

  onBeforeUnmount(() => {
    stopTimer()
    abortInFlight()
    unlockPageScroll()
  })

  return {
    visible,
    loading,
    text,
    tailLines,
    target,
    boxRef,
    live,
    wrap,
    open,
    refresh,
    scrollToBottom
  }
}

