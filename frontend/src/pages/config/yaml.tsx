import React from 'react'
import Editor from '@monaco-editor/react'
import { ProCard } from '@ant-design/pro-components'
import { Alert } from 'antd'
import { AppPage } from '@/components'

const deploymentYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: aiops-demo
  namespace: production
  labels:
    app: aiops-demo
spec:
  replicas: 3
  selector:
    matchLabels:
      app: aiops-demo
  template:
    metadata:
      labels:
        app: aiops-demo
    spec:
      containers:
        - name: web
          image: registry.example.com/aiops/demo:1.0.0
          ports:
            - containerPort: 8080
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi
          readinessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 5
`

const YamlConfigPage: React.FC = () => {
  return (
    <AppPage breadcrumbRender={false}>
      <div className="app-yaml-page">
        <Alert
          showIcon
          type="info"
          message="YAML 配置查看"
          description="当前示例以只读模式展示 Kubernetes Deployment，后续可接入配置审计、差异对比和发布前校验。"
        />
        <ProCard bordered className="app-yaml-page__editor">
          <Editor
            height="calc(100vh - 260px)"
            defaultLanguage="yaml"
            value={deploymentYaml}
            theme="vs"
            options={{
              readOnly: true,
              minimap: { enabled: false },
              scrollBeyondLastLine: false,
              wordWrap: 'on',
              automaticLayout: true,
              fontSize: 13,
              lineNumbersMinChars: 3,
            }}
          />
        </ProCard>
      </div>
    </AppPage>
  )
}

export default YamlConfigPage
