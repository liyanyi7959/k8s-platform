import React, { useRef, useState } from 'react'
import { history } from '@umijs/max'
import { ProForm, ProFormText, ProFormTextArea } from '@ant-design/pro-components'
import { Card, Form, Input, message, Upload } from 'antd'
import { InboxOutlined } from '@ant-design/icons'
import type { UploadChangeParam } from 'antd/es/upload'
import { useMutation } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { importCluster } from '@/services/clusters'
import { clusterImportSchema } from '@/schemas/cluster'
import type { ClusterImportInput } from '@/schemas/cluster'
import type { ProFormInstance } from '@ant-design/pro-components'

const { Dragger } = Upload

/** 导入集群页 */
const ClusterImportPage: React.FC = () => {
  const formRef = useRef<ProFormInstance>()
  const [fileName, setFileName] = useState<string>()
  const [kubeconfigFile, setKubeconfigFile] = useState<File>()

  const handleKubeconfigFile = (info: UploadChangeParam) => {
    const rawFile = info.file.originFileObj || info.file
    if (!rawFile || !(rawFile instanceof File)) return

    setFileName(rawFile.name)
    setKubeconfigFile(rawFile)
    const reader = new FileReader()
    reader.onload = (e) => {
      const content = e.target?.result as string
      formRef.current?.setFieldValue('kubeconfig', content)
      message.success(`${rawFile.name} 读取成功`)
    }
    reader.onerror = () => {
      message.error('文件读取失败，请重试')
    }
    reader.readAsText(rawFile)
  }

  const importMutation = useMutation({
    mutationFn: (data: ClusterImportInput) => importCluster(data, kubeconfigFile),
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
          <div style={{ marginBottom: 24 }}>
            <Dragger
              accept=".yaml,.yml,.json,.txt,.kubeconfig,.config"
              maxCount={1}
              beforeUpload={() => false}
              onChange={handleKubeconfigFile}
              showUploadList={false}
            >
              <p className="ant-upload-drag-icon">
                <InboxOutlined />
              </p>
              <p className="ant-upload-text">点击或拖拽 kubeconfig 文件到此处</p>
              <p className="ant-upload-hint">
                {fileName
                  ? `已选择：${fileName}，内容已填充到下方文本框`
                  : '支持 .yaml / .yml / .json / .txt / .kubeconfig'}
              </p>
            </Dragger>
          </div>
          <Form.Item
            name="kubeconfig"
            label="Kubeconfig 内容"
            rules={[{ required: true, message: '请上传或粘贴 kubeconfig' }]}
          >
            <Input.TextArea
              rows={8}
              placeholder="或在此直接粘贴 kubeconfig 内容"
              style={{
                fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                fontSize: 13,
                lineHeight: 1.6,
              }}
            />
          </Form.Item>
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
