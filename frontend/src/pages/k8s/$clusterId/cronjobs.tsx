import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Space, Tooltip, Drawer, Descriptions } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, ThunderboltOutlined, PauseCircleOutlined, PlayCircleOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listCronJobs, deleteCronJob, triggerCronJob, suspendCronJob } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { CronJob } from '@/types'

const CronJobsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [detailCronJob, setDetailCronJob] = useState<CronJob | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-cronjobs', clusterId, namespace],
    queryFn: () => listCronJobs(clusterId, namespace),
    enabled: !!clusterId,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: CronJob) =>
      deleteCronJob(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('CronJob 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-cronjobs', clusterId] })
    },
  })

  const triggerMutation = useMutation({
    mutationFn: (record: CronJob) =>
      triggerCronJob(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('已手动触发一次 Job')
      queryClient.invalidateQueries({ queryKey: ['k8s-cronjobs', clusterId] })
    },
    onError: () => message.error('触发失败'),
  })

  const suspendMutation = useMutation({
    mutationFn: (record: CronJob) =>
      suspendCronJob(clusterId, record.namespace || namespace, record.name, !record.suspend),
    onSuccess: (_, record) => {
      message.success(record.suspend ? '已恢复调度' : '已暂停调度')
      queryClient.invalidateQueries({ queryKey: ['k8s-cronjobs', clusterId] })
    },
    onError: () => message.error('操作失败'),
  })

  const columns: ProColumns<CronJob>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
    },
    {
      title: '调度',
      dataIndex: 'schedule',
      width: 130,
      render: (_, record) => <Tag color="blue">{record.schedule}</Tag>,
    },
    {
      title: '挂起',
      dataIndex: 'suspend',
      width: 80,
      render: (_, record) =>
        record.suspend ? <Tag color="orange">是</Tag> : <Tag color="green">否</Tag>,
    },
    { title: '活跃', dataIndex: 'active', width: 60, search: false },
    {
      title: '最后调度',
      dataIndex: 'lastScheduleTime',
      width: 180,
      search: false,
      render: (_, record) => (record.lastScheduleTime ? formatDate(record.lastScheduleTime) : '-'),
    },
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
      width: 200,
      fixed: 'right',
      render: (_, record) => (
        <Space
          size="small"
          style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}
        >
          <Tooltip title="详情">
            <a onClick={() => setDetailCronJob(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm title="立即手动触发一次该 CronJob？" onConfirm={() => triggerMutation.mutate(record)}>
            <Tooltip title="手动触发">
              <a style={{ color: '#52c41a' }}>
                <ThunderboltOutlined />
              </a>
            </Tooltip>
          </Popconfirm>
          <Popconfirm
            title={record.suspend ? '恢复该 CronJob 调度？' : '暂停该 CronJob 调度？'}
            onConfirm={() => suspendMutation.mutate(record)}
          >
            <Tooltip title={record.suspend ? '恢复' : '暂停'}>
              <a style={{ color: '#faad14' }}>
                {record.suspend ? <PlayCircleOutlined /> : <PauseCircleOutlined />}
              </a>
            </Tooltip>
          </Popconfirm>
          <Popconfirm title="确定删除该 CronJob？" onConfirm={() => deleteMutation.mutate(record)}>
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
      <ProTable<CronJob>
        columns={columns}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 900 }}
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
        title={`CronJob 详情 - ${detailCronJob?.name}`}
        open={!!detailCronJob}
        onClose={() => setDetailCronJob(null)}
        width={640}
        destroyOnClose
      >
        {detailCronJob && (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="名称">{detailCronJob.name}</Descriptions.Item>
            <Descriptions.Item label="命名空间">
              <Tag>{detailCronJob.namespace}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="调度">
              <Tag color="blue">{detailCronJob.schedule}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="挂起">
              {detailCronJob.suspend ? <Tag color="orange">是</Tag> : <Tag color="green">否</Tag>}
            </Descriptions.Item>
            <Descriptions.Item label="活跃">{detailCronJob.active ?? 0}</Descriptions.Item>
            <Descriptions.Item label="最后调度">
              {detailCronJob.lastScheduleTime ? formatDate(detailCronJob.lastScheduleTime) : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {formatDate(detailCronJob.createdAt)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>

      <YamlDrawer
        clusterId={clusterId}
        resourceType="cronjobs"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default CronJobsPage
