import React, { useState } from 'react'
import { useParams, history } from '@umijs/max'
import { Card, Descriptions, Button, Space, Spin, message } from 'antd'
import { ModalForm, ProFormText } from '@ant-design/pro-components'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getClusterById, updateCluster } from '@/services/clusters'
import { AppPage, StatusTag } from '@/components'
import { formatDate } from '@/utils'
import type { Cluster } from '@/types'

/** 集群详情页 */
const ClusterDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const queryClient = useQueryClient()
  const [editVisible, setEditVisible] = useState(false)

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

  return (
    <AppPage
      header={{
        extra: (
          <Space>
            <Button onClick={() => setEditVisible(true)}>编辑</Button>
            <Button type="primary" onClick={() => history.push(`/k8s/${cluster.id}/dashboard`)}>进入管理</Button>
            <Button onClick={() => history.push('/clusters')}>返回列表</Button>
          </Space>
        ),
      }}
    >
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

      {/* 编辑集群弹窗 */}
      <ModalForm
        title="编辑集群"
        open={editVisible}
        onOpenChange={setEditVisible}
        onFinish={async (values) => {
          updateMutation.mutate(values)
          return true
        }}
        width={520}
        modalProps={{ destroyOnClose: true }}
        initialValues={cluster}
      >
        <ProFormText
          name="name"
          label="集群名称"
          rules={[{ required: true, message: '请输入集群名称' }]}
        />
      </ModalForm>
    </AppPage>
  )
}

export default ClusterDetailPage
