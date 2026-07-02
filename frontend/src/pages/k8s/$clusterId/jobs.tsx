import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Space, Tooltip, Drawer, Descriptions } from 'antd'
import { DeleteOutlined, CodeOutlined, EyeOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listJobs, deleteJob } from '@/services/k8s'
import { AppPage, NamespaceSelector } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { Job } from '@/types'

const statusColor: Record<string, string> = {
  Running: 'processing',
  Succeeded: 'success',
  Failed: 'error',
  Pending: 'warning',
}

const JobsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [detailJob, setDetailJob] = useState<Job | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-jobs', clusterId, namespace],
    queryFn: () => listJobs(clusterId, namespace),
    enabled: !!clusterId,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: Job) => deleteJob(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('Job 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-jobs', clusterId] })
    },
  })

  const columns: ProColumns<Job>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 120,
      render: (_, r) => r.namespace || namespace,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (_, record) => (
        <Tag color={statusColor[record.status] || 'default'}>{record.status}</Tag>
      ),
    },
    { title: '完成数', dataIndex: 'completions', width: 100, search: false },
    { title: '并行度', dataIndex: 'parallelism', width: 80, search: false },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 180,
      search: false,
      render: (_, record) => formatDate(record.createdAt),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 140,
      fixed: 'right',
      render: (_, record) => (
        <Space
          size="small"
          style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}
        >
          <Tooltip title="详情">
            <a onClick={() => setDetailJob(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <CodeOutlined />
            </a>
          </Tooltip>
          <Popconfirm title="确定删除该 Job？" onConfirm={() => deleteMutation.mutate(record)}>
            <Tooltip title="删除">
              <a style={{ color: '#ff4d4f' }}>
                <DeleteOutlined />
              </a>
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <AppPage>
      <ProTable<Job>
        columns={columns}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 800 }}
        headerTitle={
          <NamespaceSelector
            clusterId={clusterId}
            value={namespace}
            onChange={setNamespace}
            style={{ width: 200 }}
          />
        }
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`Job 详情 - ${detailJob?.name}`}
        open={!!detailJob}
        onClose={() => setDetailJob(null)}
        width={640}
        destroyOnClose
      >
        {detailJob && (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="名称">{detailJob.name}</Descriptions.Item>
            <Descriptions.Item label="命名空间">
              <Tag>{detailJob.namespace}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={statusColor[detailJob.status] || 'default'}>{detailJob.status}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="完成数">{detailJob.completions ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="并行数">{detailJob.parallelism ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {formatDate(detailJob.createdAt)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>

      <YamlDrawer
        clusterId={clusterId}
        resourceType="jobs"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default JobsPage
