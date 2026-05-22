<template>
  <div ref="hostEl" :class="['cm-host', !lineNumbers ? 'cm-host--no-gutter' : '']" :style="hostStyle" />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { EditorState, Compartment, RangeSetBuilder, Transaction } from '@codemirror/state'
import { Decoration, EditorView, ViewPlugin, type DecorationSet, keymap } from '@codemirror/view'
import { basicSetup } from 'codemirror'
import { openSearchPanel, searchKeymap, highlightSelectionMatches } from '@codemirror/search'
import { foldAll, unfoldAll, foldKeymap, syntaxHighlighting, defaultHighlightStyle } from '@codemirror/language'
import { yaml } from '@codemirror/lang-yaml'
import { json } from '@codemirror/lang-json'
import { oneDarkHighlightStyle } from '@codemirror/theme-one-dark'
import { MergeView } from '@codemirror/merge'
import { buildK8sYamlAssistExtensions, type K8sYamlAssistContext } from '@/shared/components/codeMirrorYamlAssist'

const emit = defineEmits<{
  (e: 'update:text', v: string): void
}>()

const props = withDefaults(
  defineProps<{
    text: string
    compareText?: string | null
    showDiff?: boolean
    language?: 'yaml' | 'json' | 'text' | 'log'
    yamlAssist?: K8sYamlAssistContext | null
    theme?: 'auto' | 'light' | 'dark'
    readOnly?: boolean
    wrap?: boolean
    lineNumbers?: boolean
    height?: string
  }>(),
  { language: 'text', showDiff: false, compareText: null, yamlAssist: null, theme: 'auto', readOnly: true, wrap: true, lineNumbers: true, height: '100%' }
)

const hostEl = ref<HTMLElement | null>(null)
let view: EditorView | null = null
let mergeView: MergeView | null = null
let applyingExternalChange = false

const langCompartment = new Compartment()
const wrapCompartment = new Compartment()
const roCompartment = new Compartment()
const themeCompartment = new Compartment()
const yamlAssistCompartment = new Compartment()

const hostStyle = computed(() => ({ height: props.height }))

function languageExtension(lang: string) {
  if (lang === 'yaml') return yaml()
  if (lang === 'json') return json()
  if (lang === 'log') return logHighlightExtension
  return []
}

function isDark(): boolean {
  if (props.theme === 'dark') return true
  if (props.theme === 'light') return false
  return document.documentElement.classList.contains('dark')
}

const lightTheme = EditorView.theme({
  '&': {
    fontSize: '12px',
    color: '#0f172a',
    backgroundColor: '#ffffff'
  },
  '.cm-content': {
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace'
  },
  '.cm-scroller': {
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace'
  },
  '.cm-gutters': {
    backgroundColor: '#f8fafc',
    color: '#475569',
    borderRight: '1px solid rgba(15, 23, 42, 0.10)'
  },
  '.cm-activeLine': {
    backgroundColor: 'rgba(2, 6, 23, 0.04)'
  },
  '.cm-activeLineGutter': {
    backgroundColor: 'rgba(2, 6, 23, 0.06)',
    color: '#0f172a'
  },
  '.cm-selectionBackground': {
    backgroundColor: 'rgba(14, 165, 233, 0.20)'
  },
  '&.cm-focused .cm-selectionBackground': {
    backgroundColor: 'rgba(14, 165, 233, 0.28)'
  },
  '.cm-cursor': {
    borderLeftColor: '#0f172a'
  }
}, { dark: false })

const darkTheme = EditorView.theme({
  '&': {
    fontSize: '12px',
    color: '#e2e8f0',
    backgroundColor: '#0b1220'
  },
  '.cm-content': {
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace'
  },
  '.cm-scroller': {
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace'
  },
  '.cm-gutters': {
    backgroundColor: '#0a1020',
    color: '#94a3b8',
    borderRight: '1px solid rgba(148, 163, 184, 0.14)'
  },
  '.cm-activeLine': {
    backgroundColor: 'rgba(148, 163, 184, 0.10)'
  },
  '.cm-activeLineGutter': {
    backgroundColor: 'rgba(148, 163, 184, 0.14)',
    color: '#e2e8f0'
  },
  '.cm-selectionBackground': {
    backgroundColor: 'rgba(56, 189, 248, 0.22)'
  },
  '&.cm-focused .cm-selectionBackground': {
    backgroundColor: 'rgba(56, 189, 248, 0.30)'
  },
  '.cm-cursor': {
    borderLeftColor: '#e2e8f0'
  }
}, { dark: true })

function buildTheme() {
  if (isDark()) return [darkTheme, syntaxHighlighting(oneDarkHighlightStyle, { fallback: true })]
  return [lightTheme, syntaxHighlighting(defaultHighlightStyle, { fallback: true })]
}

const logPatterns = [
  { re: /\b(?:FATAL|ERROR|ERR)\b/g, cls: 'cm-log-token cm-log-token--error' },
  { re: /\b(?:WARN|WARNING)\b/g, cls: 'cm-log-token cm-log-token--warn' },
  { re: /\bINFO\b/g, cls: 'cm-log-token cm-log-token--info' },
  { re: /\bDEBUG\b/g, cls: 'cm-log-token cm-log-token--debug' },
  { re: /\bTRACE\b/g, cls: 'cm-log-token cm-log-token--trace' },
  { re: /\b\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?\b/g, cls: 'cm-log-token cm-log-token--time' }
]

function buildLogDecorations(view: EditorView): DecorationSet {
  const builder = new RangeSetBuilder<Decoration>()
  for (const { from, to } of view.visibleRanges) {
    const chunk = view.state.doc.sliceString(from, to)
    for (const p of logPatterns) {
      p.re.lastIndex = 0
      let m: RegExpExecArray | null
      while ((m = p.re.exec(chunk)) != null) {
        const start = from + m.index
        const end = start + m[0].length
        builder.add(start, end, Decoration.mark({ class: p.cls }))
        if (m.index === p.re.lastIndex) p.re.lastIndex++
      }
    }
  }
  return builder.finish()
}

const logHighlightExtension = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet
    constructor(view: EditorView) {
      this.decorations = buildLogDecorations(view)
    }
    update(u: { view: EditorView; docChanged: boolean; viewportChanged: boolean }) {
      if (u.docChanged || u.viewportChanged) this.decorations = buildLogDecorations(u.view)
    }
  },
  { decorations: (v) => v.decorations }
)

function buildReadOnlyExt() {
  return props.readOnly ? [EditorState.readOnly.of(true), EditorView.editable.of(false)] : [EditorState.readOnly.of(false), EditorView.editable.of(true)]
}

function buildWrapExt() {
  return props.wrap ? EditorView.lineWrapping : []
}

function buildYamlAssistExt() {
  if (props.language !== 'yaml') return []
  return buildK8sYamlAssistExtensions(props.yamlAssist ?? undefined)
}

function getActiveView(): EditorView | null {
  if (mergeView) return mergeView.b
  return view
}

function getScrollElement(): HTMLElement | null {
  return getActiveView()?.scrollDOM ?? null
}

function buildBaseExtensions() {
  return [
    basicSetup,
    keymap.of([...searchKeymap, ...foldKeymap]),
    highlightSelectionMatches(),
    EditorView.updateListener.of((u) => {
      if (!u.docChanged) return
      if (applyingExternalChange) return
      const isUser = u.transactions.some((tr) => tr.isUserEvent('input') || tr.isUserEvent('delete') || tr.isUserEvent('move') || tr.isUserEvent('select'))
      if (!isUser) return
      emit('update:text', u.state.doc.toString())
    })
  ]
}

function buildExtensions() {
  return [
    ...buildBaseExtensions(),
    langCompartment.of(languageExtension(props.language)),
    yamlAssistCompartment.of(buildYamlAssistExt()),
    wrapCompartment.of(buildWrapExt()),
    roCompartment.of(buildReadOnlyExt()),
    themeCompartment.of(buildTheme())
  ]
}

function init() {
  if (!hostEl.value) return
  const diff = props.showDiff && props.compareText != null
  if (diff) {
    mergeView = new MergeView({
      a: {
        doc: String(props.compareText ?? ''),
        extensions: [
          ...buildBaseExtensions(),
          languageExtension(props.language),
          buildWrapExt(),
          EditorState.readOnly.of(true),
          EditorView.editable.of(false),
          buildTheme()
        ]
      },
      b: {
        doc: String(props.text ?? ''),
        extensions: [
          ...buildBaseExtensions(),
          languageExtension(props.language),
          buildWrapExt(),
          ...buildReadOnlyExt(),
          buildTheme()
        ]
      },
      parent: hostEl.value
    })
    return
  }

  const state = EditorState.create({ doc: String(props.text ?? ''), extensions: buildExtensions() })
  view = new EditorView({ state, parent: hostEl.value })
}

function destroy() {
  mergeView?.destroy()
  mergeView = null
  view?.destroy()
  view = null
}

function setViewText(v: EditorView, next: string) {
  const current = v.state.doc.toString()
  const target = String(next ?? '')
  if (current === target) return
  applyingExternalChange = true
  v.dispatch({ changes: { from: 0, to: current.length, insert: target }, annotations: Transaction.userEvent.of('external') })
  applyingExternalChange = false
}

function reconfigureLanguage() {
  if (!view) return
  view.dispatch({ effects: langCompartment.reconfigure(languageExtension(props.language)) })
}

function reconfigureWrap() {
  if (!view) return
  view.dispatch({ effects: wrapCompartment.reconfigure(buildWrapExt()) })
}

function reconfigureYamlAssist() {
  if (!view) return
  view.dispatch({ effects: yamlAssistCompartment.reconfigure(buildYamlAssistExt()) })
}

function reconfigureReadOnly() {
  if (!view) return
  view.dispatch({ effects: roCompartment.reconfigure(buildReadOnlyExt()) })
}

function reconfigureTheme() {
  if (!view) return
  view.dispatch({ effects: themeCompartment.reconfigure(buildTheme()) })
}

function openSearch() {
  const v = getActiveView()
  if (!v) return
  openSearchPanel(v)
  v.focus()
}

function doFoldAll() {
  const v = getActiveView()
  if (!v) return
  foldAll(v)
}

function doUnfoldAll() {
  const v = getActiveView()
  if (!v) return
  unfoldAll(v)
}

defineExpose({
  openSearch,
  foldAll: doFoldAll,
  unfoldAll: doUnfoldAll,
  focus: () => getActiveView()?.focus(),
  getScrollElement
})

onMounted(() => init())
onBeforeUnmount(() => destroy())

function rebuild() {
  destroy()
  init()
}

watch(
  () => props.text,
  (v) => {
    if (mergeView) {
      setViewText(mergeView.b, String(v ?? ''))
      return
    }
    if (view) setViewText(view, String(v ?? ''))
  }
)
watch(
  () => props.compareText,
  (v) => {
    if (!mergeView) return
    setViewText(mergeView.a, String(v ?? ''))
  }
)
watch(
  () => props.language,
  () => {
    if (mergeView) {
      rebuild()
      return
    }
    reconfigureLanguage()
    reconfigureYamlAssist()
  }
)
watch(
  () => props.wrap,
  () => {
    if (mergeView) {
      rebuild()
      return
    }
    reconfigureWrap()
  }
)
watch(
  () => props.readOnly,
  () => {
    if (mergeView) {
      rebuild()
      return
    }
    reconfigureReadOnly()
  }
)
watch(
  () => props.showDiff,
  () => rebuild()
)

let themeObs: MutationObserver | null = null
function setupThemeObserver() {
  themeObs?.disconnect()
  themeObs = null
  if (props.theme !== 'auto') return
  themeObs = new MutationObserver(() => {
    if (mergeView) {
      rebuild()
      return
    }
    reconfigureTheme()
  })
  themeObs.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
}

onMounted(() => setupThemeObserver())
onBeforeUnmount(() => {
  themeObs?.disconnect()
  themeObs = null
})

watch(
  () => props.theme,
  () => {
    if (mergeView) rebuild()
    else reconfigureTheme()
    setupThemeObserver()
  }
)

watch(
  () => [props.language, props.yamlAssist?.defaultNamespace, props.yamlAssist?.sourceResource, props.yamlAssist?.workloadKind],
  () => {
    if (mergeView) rebuild()
    else reconfigureYamlAssist()
  }
)
</script>

<style scoped>
.cm-host {
  width: 100%;
}

.cm-host :deep(.cm-editor) {
  height: 100%;
  border-radius: 10px;
  overflow: hidden;
}

.cm-host :deep(.cm-scroller) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
}

.cm-host :deep(.cm-gutters) {
  user-select: none;
}

.cm-host :deep(.cm-lineNumbers .cm-gutterElement) {
  padding-left: 10px;
  padding-right: 10px;
}

.cm-host--no-gutter :deep(.cm-gutters) {
  display: none;
}

.cm-host :deep(.cm-log-token--error) {
  color: var(--c-red-500);
  font-weight: 600;
}

.cm-host :deep(.cm-log-token--warn) {
  color: var(--c-amber-400);
  font-weight: 600;
}

.cm-host :deep(.cm-log-token--info) {
  color: var(--c-cyan-500);
  font-weight: 600;
}

.cm-host :deep(.cm-log-token--debug) {
  color: var(--c-violet-400);
  font-weight: 600;
}

.cm-host :deep(.cm-log-token--trace) {
  color: var(--c-emerald-500);
  font-weight: 600;
}

.cm-host :deep(.cm-log-token--time) {
  opacity: 0.85;
}
</style>
