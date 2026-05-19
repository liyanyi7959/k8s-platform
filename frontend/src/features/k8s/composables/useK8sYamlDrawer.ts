import { computed, ref } from 'vue'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'
import { copyToClipboard, normalizeMultilineText } from '@/shared/utils/text'

export type K8sYamlLoader = () => Promise<{ text: string }>
export type K8sYamlSaver = (text: string) => Promise<void>

export function useK8sYamlDrawer() {
  const visible = ref(false)
  const meta = ref('')
  const text = ref('')
  const loader = ref<K8sYamlLoader | null>(null)
  const saver = ref<K8sYamlSaver | null>(null)
  const loading = ref(false)
  const saving = ref(false)

  const viewText = computed(() => normalizeMultilineText(text.value))
  const readOnly = computed(() => !saver.value)

  function open(nextMeta: string, nextLoader: K8sYamlLoader, nextSaver?: K8sYamlSaver) {
    meta.value = String(nextMeta ?? '')
    loader.value = nextLoader
    saver.value = nextSaver ?? null
    visible.value = true
    void load()
  }

  function close() {
    visible.value = false
  }

  async function load() {
    if (!loader.value) return
    loading.value = true
    try {
      const data = await loader.value()
      text.value = data.text
    } catch (e) {
      const err = e as ApiError
      notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
    } finally {
      loading.value = false
    }
  }

  async function save() {
    if (!saver.value) return
    saving.value = true
    try {
      await saver.value(text.value)
      notifySuccess('已保存')
      visible.value = false
    } catch (e) {
      const err = e as ApiError
      notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
    } finally {
      saving.value = false
    }
  }

  async function copy() {
    const v = viewText.value
    if (!v) return
    try {
      await copyToClipboard(v)
      notifySuccess('已复制')
    } catch (e) {
      const err = e as ApiError
      notifyError(err?.message ? `复制失败：${err.message}` : '复制失败')
    }
  }

  return { visible, meta, text, loader, saver, loading, saving, viewText, readOnly, open, close, load, save, copy }
}
