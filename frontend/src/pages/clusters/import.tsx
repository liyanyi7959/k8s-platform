import React, { useRef } from 'react'
import { history } from '@umijs/max'
import { ProForm, ProFormText, ProFormTextArea } from '@ant-design/pro-components'
import { Card, message } from 'antd'
import { useMutation } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { importCluster } from '@/services/clusters'
import { clusterImportSchema } from '@/schemas/cluster'
import type { ClusterImportInput } from '@/schemas/cluster'
import type { ProFormInstance } from '@ant-design/pro-components'

/** 导入集群页 */
const ClusterImportPage: React.FC = () => {
  const formRef = useRef<ProFormInstance>()

  const importMutation = useMutation({
    mutationFn: (data: ClusterImportInput) => importCluster(data),
    onSuccess: () => {
      message.success('导入成功')
      history.push('/clusters')
    },
    onError: (error: any) => {
      if (error?.data?.field) {
        formRef.current?.setFields([
          { name: error.data.field, errors: [error.data.message] },
        ])
      }
    },
  })

  const handleSubmit = async (values: ClusterImportInput) => {
    const result = clusterImportSchema.safeParse(values)
    if (!result.success) {
      message.error('表单校验失败')
      return
    }
    importMutation.mutate(result.data)
  }

  return (
    <AppPage>
      <Card>
        <ProForm<ClusterImportInput>
          formRef={formRef}
          onFinish={handleSubmit}
          layout="vertical"
          submitter={{
            submitButtonProps: { loading: importMutation.isPending },
          }}
        >
          <ProFormText
            name="name"
            label="集群名称"
            placeholder="请输入集群名称"
            rules={[{ required: true }]}
            fieldProps={{ maxLength: 64 }}
          />
          <ProFormTextArea
            name="kubeconfig"
            label="Kubeconfig"
            placeholder="请粘贴 kubeconfig 内容"
            rules={[{ required: true }]}
            fieldProps={{ rows: 10 }}
          />
          <ProFormTextArea
            name="description"
            label="备注"
            placeholder="请输入备注信息"
            fieldProps={{ maxLength: 200 }}
          />
        </ProForm>
      </Card>
    </AppPage>
  )
}

export default ClusterImportPage
