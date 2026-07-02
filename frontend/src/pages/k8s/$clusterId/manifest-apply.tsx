import React, { useState } from 'react'
import { Alert, Button, Card, Space, Typography } from 'antd'
import { AppPage, ManifestApplyDrawer } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'

const { Paragraph, Title, Text } = Typography

const starterYaml = `apiVersion: v1
kind: Namespace
metadata:
  name: demo
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: demo-config
  namespace: demo
data:
  APP_ENV: production
  FEATURE_FLAG: "true"
`

const ManifestApplyPage: React.FC = () => {
  const clusterId = useClusterId()
  const [drawerOpen, setDrawerOpen] = useState(false)

  return (
    <AppPage>
      <Card>
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Title level={4} style={{ margin: 0 }}>YAML 部署</Title>
          <Alert
            type="info"
            showIcon
            message="直接提交 Kubernetes Manifest"
            description="适用于快速创建 ConfigMap、Secret、RBAC、存储资源以及后端暂未提供结构化表单的对象。"
          />
          <Paragraph style={{ marginBottom: 0 }}>
            这里复用了当前前端已经接通的 manifest apply 接口。你可以直接粘贴多文档 YAML，提交到后端，再由后端应用到目标集群。
          </Paragraph>
          <Space wrap>
            <Button type="primary" onClick={() => setDrawerOpen(true)}>
              打开 YAML 工作台
            </Button>
          </Space>
          <Card size="small" title="示例清单">
            <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{starterYaml}</pre>
          </Card>
          <Text type="secondary">对于 ConfigMap、Secret、RBAC 等资源，优先在这里下发比继续保留空白表单更稳。</Text>
        </Space>
      </Card>

      <ManifestApplyDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} clusterId={clusterId} />
    </AppPage>
  )
}

export default ManifestApplyPage