import React, { useState } from 'react'
import { useParams, history, useModel } from '@umijs/max'
import { Card, Descriptions, Button, Form, Input, Space, Spin, Table, Tag, Typography, Upload, message } from 'antd'
import { ModalForm, ProFormText } from '@ant-design/pro-components'
import { InboxOutlined } from '@ant-design/icons'
import type { UploadChangeParam } from 'antd/es/upload'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getClusterById, updateCluster } from '@/features/fleet/api/clusters'
import { listProjects, type Project } from '@/features/workspace/api'
import { AppPage, StatusTag } from '@/components'
import { enterClusterWorkspace, formatDate } from '@/utils'
import type { Cluster } from '@/shared/types'

const { Text } = Typography

/** 将逗号分隔的命名空间字符串拆分为数组 */
function splitNamespaces(ns: string): string[] {
  return ns
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

/** 集群下的项目列表 Tab */
const ClusterProjectsTab: React.FC<{
  cluster: Pick<Cluster, 'id' | 'name' | 'status' | 'k8sVersion'>
  onEnterCluster: (targetPath?: string) => void
}> = ({ cluster, onEnterCluster }) => {
  const clusterId = cluster.id
  const { data, isLoading } = useQuery({
    queryKey: ['cluster-projects', clusterId],
    queryFn: () => listProjects(),
  })

  // 前端过滤出属于当前集群的项目
  const projects = (data?.list || []).filter((p) => p.cluster_id === clusterId)

  const columns = [
    {
      title: '项目名称',
      dataIndex: 'name',
      width: 160,
      render: (name: string) => <Text strong>{name || '-'}</Text>,
    },
    {
      title: '描述',
      dataIndex: 'description',
      width: 200,
      ellipsis: true,
      render: (v: string) => v || <Text type="secondary">-</Text>,
    },
    {
      title: '命名空间数',
      dataIndex: 'namespaces',
      width: 100,
      align: 'center' as const,
      render: (ns: string) => splitNamespaces(ns).length,
    },
    {
      title: '配额',
      width: 240,
      render: (_: unknown, record: Project) => (
        <Space size={4} wrap>
          {record.quota_cpu && <Tag color="blue">CPU: {record.quota_cpu}</Tag>}
          {record.quota_memory && <Tag color="green">内存: {record.quota_memory}</Tag>}
          {record.quota_pods && <Tag color="orange">Pod: {record.quota_pods}</Tag>}
          {!record.quota_cpu && !record.quota_memory && !record.quota_pods && (
            <Text type="secondary">-</Text>
          )}
        </Space>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) =>
        v ? formatDate(v, 'YYYY-MM-DD HH:mm') : <Text type="secondary">-</Text>,
    },
    {
      title: '操作',
      width: 100,
      align: 'center' as const,
      render: () => (
        <Button type="link" size="small" onClick={() => history.push('/projects')}>
          查看详情
        </Button>
      ),
    },
  ]

  return (
    <Card
      title="集群项目"
      extra={
        <Space>
          <Button type="primary" onClick={() => history.push('/projects')}>
            创建项目
          </Button>
          <Button onClick={() => history.push('/app-store/yaml')}>应用商店</Button>
          <Button onClick={() => onEnterCluster(`/k8s/${clusterId}/helm-releases`)}>
            Helm 管理
          </Button>
        </Space>
      }
    >
      <Table
        dataSource={projects}
        columns={columns}
        rowKey="id"
        loading={isLoading}
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }}
        scroll={{ x: 1000 }}
      />
    </Card>
  )
}

/** 集群详情页 */
const ClusterDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const { setCurrentCluster } = useModel('cluster')
  const queryClient = useQueryClient()
  const [editVisible, setEditVisible] = useState(false)
  const [activeTab, setActiveTab] = useState('info')
  const [editForm] = Form.useForm()
  const [kubeconfigFileName, setKubeconfigFileName] = useState<string>()

  const handleKubeconfigFile = (info: UploadChangeParam) => {
    const rawFile = info.file.originFileObj || info.file
    if (!rawFile || !(rawFile instanceof File)) return
    if (rawFile.size > 1024 * 1024) {
      setKubeconfigFileName(undefined)
      message.error('kubeconfig 文件不能超过 1MB')
      return
    }
    setKubeconfigFileName(rawFile.name)
    const reader = new FileReader()
    reader.onload = (e) => {
      editForm.setFieldValue('kubeconfig', e.target?.result as string)
      message.success(`${rawFile.name} 读取成功`)
    }
    reader.onerror = () => message.error('文件读取失败，请重试')
    reader.readAsText(rawFile)
  }

  const { data: cluster, isLoading } = useQuery({
    queryKey: ['cluster', id],
    queryFn: () => getClusterById(Number(id)),
    enabled: !!id,
  })

  const updateMutation = useMutation({
    mutationFn: (data: Partial<Cluster>) => updateCluster(Number(id), data),
    onSuccess: () => {
      message.success('更新成功')
      setEditVisible(false)
      queryClient.invalidateQueries({ queryKey: ['cluster', id] })
      queryClient.invalidateQueries({ queryKey: ['clusters'] })
    },
    onError: () => message.error('更新失败'),
  })

  if (isLoading) {
    return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />
  }

  if (!cluster) {
    return <div>集群不存在</div>
  }

  const handleEnterCluster = (targetPath?: string) => {
    enterClusterWorkspace(cluster, {
      setCurrentCluster,
      targetPath,
    })
  }

  return (
    <AppPage
      header={{
        extra: (
          <Space>
            <Button onClick={() => setEditVisible(true)}>编辑</Button>
            <Button type="primary" onClick={() => handleEnterCluster()}>
              进入管理
            </Button>
            <Button onClick={() => history.push('/clusters')}>返回列表</Button>
          </Space>
        ),
      }}
      tabActiveKey={activeTab}
      onTabChange={setActiveTab}
      tabList={[
        { key: 'info', tab: '基本信息' },
        { key: 'projects', tab: '项目' },
      ]}
    >
      {activeTab === 'info' && (
        <Card>
          <Descriptions column={2} bordered>
            <Descriptions.Item label="集群名称">{cluster.name}</Descriptions.Item>
            <Descriptions.Item label="类型">{cluster.type || '-'}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <StatusTag status={cluster.status} />
            </Descriptions.Item>
            <Descriptions.Item label="K8s 版本">{cluster.k8sVersion || '-'}</Descriptions.Item>
            <Descriptions.Item label="节点数">{cluster.nodeCount || '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{cluster.createdAt ? formatDate(cluster.createdAt) : '-'}</Descriptions.Item>
            <Descriptions.Item label="最后健康检查">{cluster.lastHealthAt ? formatDate(cluster.lastHealthAt) : '-'}</Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      {activeTab === 'projects' && (
        <ClusterProjectsTab cluster={cluster} onEnterCluster={handleEnterCluster} />
      )}

      {/* 编辑集群弹窗 */}
      <ModalForm
        title="编辑集群"
        open={editVisible}
        onOpenChange={(visible) => {
          setEditVisible(visible)
          if (!visible) {
            setKubeconfigFileName(undefined)
          }
        }}
        form={editForm}
        onFinish={async (values) => {
          const payload: Record<string, string> = { name: values.name }
          if (values.kubeconfig) {
            payload.kubeconfig = values.kubeconfig
          }
          updateMutation.mutate(payload)
          return true
        }}
        width={560}
        modalProps={{ destroyOnClose: true }}
        initialValues={cluster}
      >
        <ProFormText
          name="name"
          label="集群名称"
          rules={[{ required: true, message: '请输入集群名称' }]}
        />
        <Form.Item label="更新 Kubeconfig">
          <Upload.Dragger
            accept=".yaml,.yml,.json,.txt,.kubeconfig,.config"
            maxCount={1}
            beforeUpload={() => false}
            onChange={handleKubeconfigFile}
            showUploadList={false}
          >
            <p className="ant-upload-drag-icon">
              <InboxOutlined />
            </p>
            <p className="ant-upload-text">
              {kubeconfigFileName
                ? `已选择：${kubeconfigFileName}`
                : '点击或拖拽 kubeconfig 文件到此处'}
            </p>
            <p className="ant-upload-hint">留空则不更新凭据</p>
          </Upload.Dragger>
        </Form.Item>
        <Form.Item
          name="kubeconfig"
          label="Kubeconfig 内容"
          extra="粘贴或上传新的 kubeconfig 以更新集群凭据，留空则保持不变"
        >
          <Input.TextArea
            rows={6}
            placeholder="或在此直接粘贴新的 kubeconfig 内容"
            style={{
              fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
              fontSize: 13,
              lineHeight: 1.6,
            }}
          />
        </Form.Item>
      </ModalForm>
    </AppPage>
  )
}

export default ClusterDetailPage
