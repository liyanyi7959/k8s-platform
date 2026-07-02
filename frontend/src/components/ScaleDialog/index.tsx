/**
 * 扩缩容对话框
 * 对标 frontend-old 的 K8sScaleDialog
 * 支持 Deployment/StatefulSet 的副本数调整
 */
import React, { useEffect } from 'react'
import { Modal, Form, InputNumber, message } from 'antd'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { scaleDeployment } from '@/services/k8s'

interface ScaleDialogProps {
  open: boolean
  onClose: () => void
  clusterId: number
  namespace: string
  name: string
  currentReplicas: number
  resourceType?: 'Deployment' | 'StatefulSet'
}

export const ScaleDialog: React.FC<ScaleDialogProps> = ({
  open,
  onClose,
  clusterId,
  namespace,
  name,
  currentReplicas,
  resourceType = 'Deployment',
}) => {
  const [form] = Form.useForm()
  const queryClient = useQueryClient()

  useEffect(() => {
    if (open) {
      form.setFieldsValue({ replicas: currentReplicas })
    }
  }, [open, currentReplicas, form])

  const scaleMutation = useMutation({
    mutationFn: (replicas: number) => scaleDeployment(clusterId, namespace, name, replicas),
    onSuccess: () => {
      message.success(`${resourceType} "${name}" 扩缩容成功`)
      queryClient.invalidateQueries({ queryKey: ['k8s-deployments', clusterId] })
      onClose()
    },
    onError: () => {
      message.error('扩缩容失败，请重试')
    },
  })

  const handleOk = async () => {
    try {
      const values = await form.validateFields()
      scaleMutation.mutate(values.replicas)
    } catch {
      // 表单校验失败
    }
  }

  return (
    <Modal
      title={`扩缩容 - ${resourceType}: ${name}`}
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      confirmLoading={scaleMutation.isPending}
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="replicas"
          label="副本数"
          rules={[{ required: true, message: '请输入副本数' }, { type: 'number', min: 0, max: 100, message: '副本数范围 0-100' }]}
        >
          <InputNumber style={{ width: '100%' }} min={0} max={100} placeholder="请输入目标副本数" />
        </Form.Item>
        <div style={{ color: '#999', fontSize: 12 }}>
          当前副本数: {currentReplicas}，调整为 0 将停止所有 Pod
        </div>
      </Form>
    </Modal>
  )
}
