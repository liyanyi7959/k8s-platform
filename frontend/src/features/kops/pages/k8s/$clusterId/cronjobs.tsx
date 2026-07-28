import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Space, Tooltip, Drawer, Descriptions, Button, Input, Select, Typography, Table, Tabs } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, ThunderboltOutlined, PauseCircleOutlined, PlayCircleOutlined, SearchOutlined, ReloadOutlined, PlusOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listCronJobs, deleteCronJob, triggerCronJob, suspendCronJob, listJobs, getPodEvents } from '@/features/kops/api/k8s'
import { AppPage, EllipsisText } from '@/components'
import { NamespaceSelector, ManifestApplyDrawer } from '@/features/kops/components'
import YamlDrawer, { useYamlDrawer } from '@/features/kops/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { CronJob } from '@/features/kops/types'

const { Text } = Typography

const weekNames = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

/** 解析 cron 表达式为中文说明 */
function explainCron(cron: string): string {
  if (!cron) return ''
  const presets: Record<string, string> = {
    '@hourly': '每小时执行一次',
    '@daily': '每天 0 点执行一次',
    '@midnight': '每天 0 点执行一次',
    '@weekly': '每周日 0 点执行一次',
    '@monthly': '每月 1 号 0 点执行一次',
    '@yearly': '每年 1 月 1 日 0 点执行一次',
    '@annually': '每年 1 月 1 日 0 点执行一次',
  }
  if (presets[cron.toLowerCase()]) return presets[cron.toLowerCase()]

  const parts = cron.trim().split(/\s+/)
  if (parts.length !== 5) return cron
  const [min, hour, day, month, week] = parts

  // */N * * * * → 每 N 分钟
  if (min.startsWith('*/') && hour === '*' && day === '*' && month === '*' && week === '*')
    return `每 ${min.slice(2)} 分钟执行一次`
  // 0 */N * * * → 每 N 小时
  if (min === '0' && hour.startsWith('*/') && day === '*' && month === '*' && week === '*')
    return `每 ${hour.slice(2)} 小时执行一次`
  // 0 * * * * → 每小时
  if (min === '0' && hour === '*' && day === '*' && month === '*' && week === '*')
    return '每小时整点执行一次'
  // */N */M * * * → 每 M 小时的第 N 分钟
  if (min.startsWith('*/') && hour.startsWith('*/') && day === '*' && month === '*' && week === '*')
    return `每 ${hour.slice(2)} 小时的第 ${min.slice(2)} 分钟执行`
  // 0 0 * * * → 每天 0 点
  if (min === '0' && hour === '0' && day === '*' && month === '*' && week === '*')
    return '每天 0 点执行一次'
  // 0 N * * * → 每天 N 点
  if (min === '0' && /^\d+$/.test(hour) && day === '*' && month === '*' && week === '*')
    return `每天 ${hour} 点 ${min} 分执行一次`
  // N H * * * → 每天 H 点 N 分
  if (/^\d+$/.test(min) && /^\d+$/.test(hour) && day === '*' && month === '*' && week === '*')
    return `每天 ${hour}:${min.padStart(2, '0')} 执行一次`
  // 0 0 * * 0 → 每周日 0 点
  if (min === '0' && hour === '0' && day === '*' && month === '*' && /^\d$/.test(week))
    return `每${weekNames[parseInt(week)] || `周${week}`} 0 点执行一次`
  // 0 0 1 * * → 每月 1 号 0 点
  if (min === '0' && hour === '0' && /^\d+$/.test(day) && month === '*' && week === '*')
    return `每月 ${day} 号 0 点执行一次`
  // 0 H D * * → 每月 D 号 H 点
  if (min === '0' && /^\d+$/.test(hour) && /^\d+$/.test(day) && month === '*' && week === '*')
    return `每月 ${day} 号 ${hour} 点执行一次`

  return cron
}

const CronJobsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [searchType, setSearchType] = useState<'name' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [detailCronJob, setDetailCronJob] = useState<CronJob | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-cronjobs', clusterId, namespace],
    queryFn: ({ signal }) => listCronJobs(clusterId, namespace, signal),
    enabled: !!clusterId,
    refetchInterval: detailCronJob || createOpen ? false : 60_000,
    staleTime: 60_000,
  })

  const { data: jobsData, isLoading: jobsLoading } = useQuery({
    queryKey: ['k8s-cronjob-jobs', clusterId, detailCronJob?.namespace],
    queryFn: ({ signal }) => listJobs(clusterId, detailCronJob?.namespace, signal),
    enabled: !!clusterId && !!detailCronJob && !!detailCronJob.namespace,
  })

  const { data: eventsData, isLoading: eventsLoading } = useQuery({
    queryKey: ['k8s-cronjob-events', clusterId, detailCronJob?.namespace, detailCronJob?.name],
    queryFn: ({ signal }) => getPodEvents(clusterId, detailCronJob?.namespace || '', detailCronJob?.name || '', signal),
    enabled: !!clusterId && !!detailCronJob && !!detailCronJob.namespace && !!detailCronJob.name,
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

  const filteredData = (data?.items || []).filter((item: any) => {
    if (!searchValue) return true
    const v = searchValue.toLowerCase()
    if (searchType === 'name') return item.name.toLowerCase().includes(v)
    return item.labels && Object.entries(item.labels).some(([k, val]) =>
      `${k}=${val}`.toLowerCase().includes(v) || k.toLowerCase().includes(v) || String(val).toLowerCase().includes(v))
  })

  const relatedJobs = (jobsData?.items || []).filter((job: any) => {
    return job.ownerKind === 'CronJob' && job.ownerName === detailCronJob?.name
  })

  const columns: ProColumns<CronJob>[] = [
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
      title: '调度',
      dataIndex: 'schedule',
      width: 130,
      align: 'center' as const,
      render: (_, r) => (
        <Tooltip title={explainCron(r.schedule)}>
          <Tag color="blue">{r.schedule}</Tag>
        </Tooltip>
      ),
    },
    {
      title: '挂起',
      dataIndex: 'suspend',
      width: 80,
      align: 'center' as const,
      render: (_, r) =>
        r.suspend ? <Tag color="orange">是</Tag> : <Tag color="green">否</Tag>,
    },
    {
      title: '并发策略', dataIndex: 'concurrencyPolicy', width: 110, align: 'center' as const, search: false,
      render: (_, r) => <Tag>{r.concurrencyPolicy || 'Allow'}</Tag>,
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
      title: '活跃',
      dataIndex: 'active',
      width: 70,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (a.active || 0) - (b.active || 0),
    },
    {
      title: '最后调度',
      dataIndex: 'lastScheduleTime',
      width: 150,
      align: 'center' as const,
      search: false,
      ellipsis: true,
      render: (_, r) => (r.lastScheduleTime ? formatDate(r.lastScheduleTime) : '-'),
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
      width: 200,
      fixed: 'right',
      align: 'center' as const,
      render: (_, record) => (
        <Space size="small" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
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
              <a style={{ color: '#047857' }}>
                <ThunderboltOutlined />
              </a>
            </Tooltip>
          </Popconfirm>
          <Popconfirm
            title={record.suspend ? '恢复该 CronJob 调度？' : '暂停该 CronJob 调度？'}
            onConfirm={() => suspendMutation.mutate(record)}
          >
            <Tooltip title={record.suspend ? '恢复' : '暂停'}>
              <a style={{ color: '#b45309' }}>
                {record.suspend ? <PlayCircleOutlined /> : <PauseCircleOutlined />}
              </a>
            </Tooltip>
          </Popconfirm>
          <Popconfirm title="确定删除该 CronJob？" onConfirm={() => deleteMutation.mutate(record)}>
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
      <ProTable<CronJob>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1250 }}
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
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            创建 CronJob
          </Button>,
        ]}
        headerTitle={<Text strong>CronJob 列表</Text>}
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`CronJob 详情 - ${detailCronJob?.name}`}
        open={!!detailCronJob}
        onClose={() => setDetailCronJob(null)}
        width={720}
        destroyOnClose
      >
        {detailCronJob && (
          <Tabs
            defaultActiveKey="overview"
            items={[
              {
                key: 'overview',
                label: '概览',
                children: (
                  <>
                    <Descriptions bordered column={2} size="small">
                      <Descriptions.Item label="名称">{detailCronJob.name}</Descriptions.Item>
                      <Descriptions.Item label="Namespace"><Tag>{detailCronJob.namespace}</Tag></Descriptions.Item>
                      <Descriptions.Item label="调度" span={2}>
                        <Tag color="blue">{detailCronJob.schedule}</Tag>
                      </Descriptions.Item>
                      <Descriptions.Item label="挂起">
                        {detailCronJob.suspend ? <Tag color="orange">是</Tag> : <Tag color="green">否</Tag>}
                      </Descriptions.Item>
                      <Descriptions.Item label="活跃">{detailCronJob.active ?? 0}</Descriptions.Item>
                      <Descriptions.Item label="并发策略">
                        <Tag>{detailCronJob.concurrencyPolicy || 'Allow'}</Tag>
                      </Descriptions.Item>
                      <Descriptions.Item label="最后调度">
                        {detailCronJob.lastScheduleTime ? formatDate(detailCronJob.lastScheduleTime) : '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="历史保留">
                        成功 {detailCronJob.successfulJobsHistoryLimit ?? 3} / 失败 {detailCronJob.failedJobsHistoryLimit ?? 1}
                      </Descriptions.Item>
                      <Descriptions.Item label="创建时间">
                        {formatDate(detailCronJob.createdAt)}
                      </Descriptions.Item>
                    </Descriptions>
                    {detailCronJob.labels && Object.keys(detailCronJob.labels).length > 0 && (
                      <div style={{ marginTop: 16 }}>
                        <Text strong>标签</Text>
                        <div style={{ marginTop: 8 }}>
                          <Space size={4} wrap>
                            {Object.entries(detailCronJob.labels).map(([k, v]) => (
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
                key: 'jobs',
                label: '关联 Job',
                children: (
                  <Table
                    size="small"
                    rowKey="name"
                    loading={jobsLoading}
                    pagination={false}
                    dataSource={relatedJobs}
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
                            Running: 'processing',
                            Active: 'processing',
                            Succeeded: 'success',
                            Failed: 'error',
                            Pending: 'warning',
                          }
                          return <Tag color={colorMap[r.status] || 'default'}>{r.status}</Tag>
                        },
                      },
                      {
                        title: '完成数',
                        dataIndex: 'completions',
                        width: 90,
                        align: 'center' as const,
                      },
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
        resourceType="cronjobs"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />

      <ManifestApplyDrawer
        clusterId={clusterId}
        open={createOpen}
        onClose={() => { setCreateOpen(false); queryClient.invalidateQueries({ queryKey: ['k8s-cronjobs', clusterId] }) }}
        title="创建 CronJob"
        initialYaml={`apiVersion: batch/v1\nkind: CronJob\nmetadata:\n  name: my-cronjob\n  namespace: ${namespace || 'default'}\nspec:\n  schedule: "*/5 * * * *"\n  jobTemplate:\n    spec:\n      template:\n        spec:\n          containers:\n          - name: hello\n            image: busybox:1.28\n            command: ["echo", "hello"]\n          restartPolicy: OnFailure\n`}
      />
    </AppPage>
  )
}

export default CronJobsPage
