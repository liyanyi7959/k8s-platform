/**
 * YAML 编辑器组件
 * 懒加载 Monaco Editor
 */
import React, { lazy, Suspense } from 'react'
import { Spin } from 'antd'

const MonacoEditor = lazy(() => import('@monaco-editor/react'))

interface YamlEditorProps {
  value: string
  onChange?: (value: string) => void
  readOnly?: boolean
  height?: number
  language?: string
}

export const YamlEditor: React.FC<YamlEditorProps> = ({
  value,
  onChange,
  readOnly = false,
  height = 400,
  language = 'yaml',
}) => {
  return (
    <Suspense fallback={<Spin tip="编辑器加载中..." />}>
      <MonacoEditor
        height={height}
        language={language}
        value={value}
        onChange={(val) => onChange?.(val || '')}
        options={{
          readOnly,
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          fontSize: 14,
          wordWrap: 'on',
          automaticLayout: true,
        }}
      />
    </Suspense>
  )
}
