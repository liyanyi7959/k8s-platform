import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Space, Tooltip, Drawer, Descriptions, Button, Input, Select, Typography, Table, Tabs } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, ClearOutlined, SearchOutlined, ReloadOutlined, PlusOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listJobs, deleteJob, deleteCompletedJobs, listPods, getPodEvents } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText, ManifestApplyDrawer } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { Job } from '@/types'

const { Text } = Typography

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
  const [searchType, setSearchType] = useState<'name' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [detailJob, setDetailJob] = useState<Job | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-jobs', clusterId, namespace],
    queryFn: ({ signal }) => listJobs(clusterId, namespace, signal),
    enabled: !!clusterId,
    refetchInterval: detailJob || createOpen ? false : 60_000,
    staleTime: 60_000,
  })

  const { data: podsData, isLoading: podsLoading } = useQuery({
    queryKey: ['k8s-job-pods', clusterId, detailJob?.namespace],
    queryFn: ({ signal }) => listPods(clusterId, { namespace: detailJob?.namespace }, signal),
    enabled: !!clusterId && !!detailJob && !!detailJob.namespace,
  })

  const { data: eventsData, isLoading: eventsLoading } = useQuery({
    queryKey: ['k8s-job-events', clusterId, detailJob?.namespace, detailJob?.name],
    queryFn: ({ signal }) => getPodEvents(clusterId, detailJob?.namespace || '', detailJob?.name || '', signal),
    enabled: !!clusterId && !!detailJob && !!detailJob.namespace && !!detailJob.name,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: Job) => deleteJob(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('Job 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-jobs', clusterId] })
    },
  })

  const cleanupMutation = useMutation({
    mutationFn: () => deleteCompletedJobs(clusterId),
    onSuccess: () => {
      message.success('已清理所有已完成的 Job')
      queryClient.invalidateQueries({ queryKey: ['k8s-jobs', clusterId] })
    },
    onError: () => message.error('清理失败'),
  })

  const filteredData = (data?.items || []).filter((item: any) => {
    if (!searchValue) return true
    const v = searchValue.toLowerCase()
    if (searchType === 'name') return item.name.toLowerCase().includes(v)
    return item.labels && Object.entries(item.labels).some(([k, val]) =>
      `${k}=${val}`.toLowerCase().includes(v) || k.toLowerCase().includes(v) || String(val).toLowerCase().includes(v))
  })

  const relatedPods = (podsData?.items || []).filter((pod: any) => {
    return pod.ownerKind === 'Job' && pod.ownerName === detailJob?.name
  })

  const columns: ProColumns<Job>[] = [
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 160,
      ellipsis: true,
      copyable: true,
      render: (_, r) => <Text strong>{r.name}</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      align: 'center' as const,
      render: (_, r) => <Tag color={statusColor[r.status] || 'default'}>{r.status}</Tag>,
    },
    {
      title: '完成数',
      dataIndex: 'completions',
      width: 90,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (a.completions || 0) - (b.completions || 0),
    },
    {
      title: '并行度',
      dataIndex: 'parallelism',
      width: 80,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (a.parallelism || 0) - (b.parallelism || 0),
    },
    {
      title: '镜像', width: 200, ellipsis: true, search: false,
      render: (_, r) => {
        const images = r.images || []
        return images.length > 0 ? (
          <Tooltip title={images.join('\n')}>
            <Text style={{ fontSize: 11 }}>{images[0]}</Text>
          </Tooltip>
        ) : '-'
      },
    },
    {
      title: 'Age',
      dataIndex: 'createdAt',
      width: 110,
      align: 'center' as const,
      search: false,
      ellipsis: true,
      render: (_, r) => (r.createdAt ? formatDate(r.createdAt) : '-'),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 140,
      fixed: 'right',
      align: 'center' as const,
      render: (_, record) => (
        <Space size="small" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <Tooltip title="详情">
            <a onClick={() => setDetailJob(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm title="确定删除该 Job？" onConfirm={() => deleteMutation.mutate(record)}>
            <Tooltip title="删除">
              <a style={{ color: '#dc2626' }}>
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
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1030 }}
        toolBarRender={() => [
          <Input.Search
            key="search"
            placeholder={searchType === 'name' ? '按名称搜索' : '按标签搜索'}
            allowClear
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            style={{ width: 280 }}
            addonBefore={<Select value={searchType} onChange={(v) => setSearchType(v)} style={{ width: 70 }}
              options={[{ value: 'name', label: '名称' }, { value: 'label', label: '标签' }]} />}
            prefix={<SearchOutlined />}
          />,
          <NamespaceSelector
            key="ns"
            clusterId={clusterId}
            value={namespace}
            onChange={setNamespace}
            style={{ width: 180 }}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>,
          <Popconfirm
            key="cleanup"
            title="清理该命名空间下所有已完成的 Job？"
            onConfirm={() => cleanupMutation.mutate()}
          >
            <Button danger icon={<ClearOutlined />} loading={cleanupMutation.isPending}>
              清理已完成
            </Button>
          </Popconfirm>,
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            创建 Job
          </Button>,
        ]}
        headerTitle={<Text strong>Job 列表</Text>}
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`Job 详情 - ${detailJob?.name}`}
        open={!!detailJob}
        onClose={() => setDetailJob(null)}
        width={720}
        destroyOnClose
      >
        {detailJob && (
          <Tabs
            defaultActiveKey="overview"
            items={[
              {
                key: 'overview',
                label: '概览',
                children: (
                  <>
                    <Descriptions bordered column={2} size="small">
                      <Descriptions.Item label="名称">{detailJob.name}</Descriptions.Item>
                      <Descriptions.Item label="Namespace"><Tag>{detailJob.namespace}</Tag></Descriptions.Item>
                      <Descriptions.Item label="状态" span={2}>
                        <Tag color={statusColor[detailJob.status] || 'default'}>{detailJob.status}</Tag>
                      </Descriptions.Item>
                      <Descriptions.Item label="完成数">{detailJob.completions ?? '-'}</Descriptions.Item>
                      <Descriptions.Item label="并行数">{detailJob.parallelism ?? '-'}</Descriptions.Item>
                      <Descriptions.Item label="创建时间" span={2}>{formatDate(detailJob.createdAt)}</Descriptions.Item>
                    </Descriptions>
                    {detailJob.labels && Object.keys(detailJob.labels).length > 0 && (
                      <div style={{ marginTop: 16 }}>
                        <Text strong>标签</Text>
                        <div style={{ marginTop: 8 }}>
                          <Space size={4} wrap>
                            {Object.entries(detailJob.labels).map(([k, v]) => (
                              <Tag key={k}>{k}={v}</Tag>
                            ))}
                          </Space>
                        </div>
                      </div>
                    )}
                  </>
                ),
              },
              {
                key: 'pods',
                label: '关联 Pod',
                children: (
                  <Table
                    size="small"
                    rowKey="name"
                    loading={podsLoading}
                    pagination={false}
                    dataSource={relatedPods}
                    columns={[
                      {
                        title: '名称',
                        dataIndex: 'name',
                        ellipsis: true,
                        render: (_, r) => <Text strong>{r.name}</Text>,
                      },
                      {
                        title: '状态',
                        dataIndex: 'status',
                        width: 100,
                        render: (_, r) => {
                          const colorMap: Record<string, string> = {
                            Running: 'success',
                            Pending: 'processing',
                            Failed: 'error',
                            Succeeded: 'default',
                          }
                          return <Tag color={colorMap[r.status] || 'default'}>{r.status}</Tag>
                        },
                      },
                      { title: '节点', dataIndex: 'nodeName', ellipsis: true },
                      { title: '重启次数', dataIndex: 'restarts', width: 90, align: 'center' as const },
                      {
                        title: 'Age',
                        dataIndex: 'createdAt',
                        width: 110,
                        render: (_, r) => (r.createdAt ? formatDate(r.createdAt) : '-'),
                      },
                    ]}
                  />
                ),
              },
              {
                key: 'events',
                label: '事件',
                children: (
                  <Table
                    size="small"
                    rowKey={(_, i) => String(i)}
                    loading={eventsLoading}
                    pagination={{ defaultPageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
                    dataSource={eventsData || []}
                    columns={[
                      {
                        title: '类型',
                        dataIndex: 'type',
                        width: 90,
                        render: (_, r) => (
                          <Tag color={r.type === 'Warning' ? 'warning' : 'success'}>{r.type || 'Normal'}</Tag>
                        ),
                      },
                      { title: '原因', dataIndex: 'reason', width: 140, ellipsis: true },
                      { title: '消息', dataIndex: 'message', ellipsis: true },
                      {
                        title: '时间',
                        dataIndex: 'lastTimestamp',
                        width: 150,
                        render: (_, r) =>
                          formatDate(r.lastTimestamp || r.firstTimestamp || r.metadata?.creationTimestamp || ''),
                      },
                    ]}
                  />
                ),
              },
            ]}
          />
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

      <ManifestApplyDrawer
        clusterId={clusterId}
        open={createOpen}
        onClose={() => { setCreateOpen(false); queryClient.invalidateQueries({ queryKey: ['k8s-jobs', clusterId] }) }}
        title="创建 Job"
        initialYaml={`apiVersion: batch/v1\nkind: Job\nmetadata:\n  name: my-job\n  namespace: ${namespace || 'default'}\nspec:\n  template:\n    spec:\n      containers:\n      - name: hello\n        image: busybox:1.28\n        command: ["echo", "hello"]\n      restartPolicy: Never\n  backoffLimit: 4\n`}
      />
    </AppPage>
  )
}

export default JobsPage
