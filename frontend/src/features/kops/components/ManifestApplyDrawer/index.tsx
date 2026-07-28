/**
 * YAML 应用抽屉
 * 对标 frontend-old 的 ManifestApplyDrawer
 * 支持粘贴 YAML 并应用到集群
 */
import React, { useState } from 'react'
import { Drawer, Button, Space, message } from 'antd'
import { SendOutlined, ClearOutlined } from '@ant-design/icons'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { applyYaml } from '@/features/kops/api/k8s'
import { YamlEditor } from '@/components/YamlEditor'
import AppAlert from '@/components/AppAlert'

interface ManifestApplyDrawerProps {
  open: boolean
  onClose: () => void
  clusterId: number
  /** 打开时预填充的 YAML 模板 */
  initialYaml?: string
  /** 抽屉标题，默认“应用 YAML” */
  title?: string
}

export const ManifestApplyDrawer: React.FC<ManifestApplyDrawerProps> = ({
  open,
  onClose,
  clusterId,
  initialYaml = '',
  title = '应用 YAML',
}) => {
  const [yaml, setYaml] = useState(initialYaml)
  const [result, setResult] = useState<{ success: boolean; message: string } | null>(null)
  const queryClient = useQueryClient()

  // 打开或模板变化时同步初始 YAML
  React.useEffect(() => {
    if (open) {
      setYaml(initialYaml)
      setResult(null)
    }
  }, [open, initialYaml])

  const applyMutation = useMutation({
    mutationFn: () => applyYaml(clusterId, yaml),
    onSuccess: (res) => {
      setResult(res)
      if (res.success) {
        message.success('YAML 应用成功')
        queryClient.invalidateQueries()
      } else {
        message.error(res.message)
      }
    },
    onError: () => {
      message.error('YAML 应用失败，请重试')
    },
  })

  const handleApply = () => {
    if (!yaml.trim()) {
      message.warning('请输入 YAML 内容')
      return
    }
    applyMutation.mutate()
  }

  const handleClear = () => {
    setYaml('')
    setResult(null)
  }

  return (
    <Drawer
      title={title}
      open={open}
      onClose={onClose}
      width={800}
      extra={
        <Space>
          <Button icon={<ClearOutlined />} onClick={handleClear}>清空</Button>
          <Button type="primary" icon={<SendOutlined />} onClick={handleApply} loading={applyMutation.isPending}>
            应用
          </Button>
        </Space>
      }
    >
      {result && (
        <AppAlert
          type={result.success ? 'success' : 'error'}
          message={result.success ? '应用成功' : '应用失败'}
          description={result.message}
          closable
          style={{ marginBottom: 16 }}
        />
      )}
      <YamlEditor value={yaml} onChange={setYaml} height={500} />
    </Drawer>
  )
}
