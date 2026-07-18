/**
 * YAML 编辑器组件
 * 懒加载 Monaco Editor
 */
import React, { lazy, Suspense, useCallback, useRef } from 'react'
import { Spin } from 'antd'
import type { Monaco, OnMount } from '@monaco-editor/react'
import { registerKubernetesYamlLanguage, validateKubernetesYaml, type YamlDiagnosticSummary } from './kubernetesYamlLanguage'

const MonacoEditor = lazy(() => import('@monaco-editor/react'))

interface YamlEditorProps {
  value: string
  onChange?: (value: string) => void
  readOnly?: boolean
  height?: number
  language?: string
  kubernetes?: boolean
  onDiagnosticsChange?: (summary: YamlDiagnosticSummary) => void
}

export const YamlEditor: React.FC<YamlEditorProps> = ({
  value,
  onChange,
  readOnly = false,
  height = 400,
  language = 'yaml',
  kubernetes = false,
  onDiagnosticsChange,
}) => {
  const validationTimer = useRef<number>()
  const beforeMount = useCallback((monaco: Monaco) => {
    if (kubernetes) registerKubernetesYamlLanguage(monaco)
  }, [kubernetes])
  const handleMount = useCallback<OnMount>((editor, monaco) => {
    if (!kubernetes) return
    const validate = () => {
      window.clearTimeout(validationTimer.current)
      validationTimer.current = window.setTimeout(() => {
        const model = editor.getModel()
        if (model) onDiagnosticsChange?.(validateKubernetesYaml(monaco, model))
      }, 180)
    }
    validate()
    editor.onDidChangeModelContent(validate)
  }, [kubernetes, onDiagnosticsChange])
  return (
    <Suspense fallback={<Spin tip="编辑器加载中..." />}>
      <MonacoEditor
        height={height}
        language={language}
        value={value}
        beforeMount={beforeMount}
        onMount={handleMount}
        onChange={(val) => onChange?.(val || '')}
        options={{
          readOnly,
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          fontSize: 14,
          wordWrap: 'on',
          automaticLayout: true,
          quickSuggestions: kubernetes ? { other: true, comments: false, strings: true } : undefined,
          suggestOnTriggerCharacters: kubernetes,
          tabCompletion: kubernetes ? 'on' : 'off',
          wordBasedSuggestions: kubernetes ? 'off' : 'currentDocument',
          parameterHints: { enabled: kubernetes },
          glyphMargin: kubernetes,
        }}
      />
    </Suspense>
  )
}
