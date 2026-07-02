import React, { useState } from 'react'
import {
  ProTable,
  type ProColumns,
  ModalForm,
  ProFormText,
  ProFormSelect,
} from '@ant-design/pro-components'
import { Button, message, Popconfirm, Tag, Space } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import {
  listAlertRules,
  createAlertRule,
  updateAlertRule,
  deleteAlertRule,
  toggleAlertRule,
} from '@/services/monitor'
import { formatDate } from '@/utils'
import type { AlertRule, CreateAlertRuleRequest } from '@/types'

/** 告警规则页 */
const AlertRulesPage: React.FC = () => {
  const queryClient = useQueryClient()
  const [createVisible, setCreateVisible] = useState(false)
  const [editVisible, setEditVisible] = useState(false)
  const [currentRule, setCurrentRule] = useState<AlertRule | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['alert-rules'],
    queryFn: () => listAlertRules(),
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateAlertRuleRequest) => createAlertRule(data),
    onSuccess: () => {
      message.success('创建成功')
      setCreateVisible(false)
      queryClient.invalidateQueries({ queryKey: ['alert-rules'] })
    },
    onError: () => message.error('创建失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<CreateAlertRuleRequest> }) =>
      updateAlertRule(id, data),
    onSuccess: () => {
      message.success('更新成功')
      setEditVisible(false)
      setCurrentRule(null)
      queryClient.invalidateQueries({ queryKey: ['alert-rules'] })
    },
    onError: () => message.error('更新失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteAlertRule(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['alert-rules'] })
    },
    onError: () => message.error('删除失败'),
  })

  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) => toggleAlertRule(id, enabled),
    onSuccess: () => {
      message.success('状态已更新')
      queryClient.invalidateQueries({ queryKey: ['alert-rules'] })
    },
  })

  const handleEdit = (record: AlertRule) => {
    setCurrentRule(record)
    setEditVisible(true)
  }

  const columns: ProColumns<AlertRule>[] = [
    { title: '规则名称', dataIndex: 'name' },
    { title: '集群 ID', dataIndex: 'clusterId', width: 80 },
    {
      title: '告警级别',
      dataIndex: 'severity',
      width: 100,
      render: (_, record) => {
        const colorMap: Record<string, string> = {
          info: 'blue',
          warning: 'orange',
          critical: 'red',
        }
        return <Tag color={colorMap[record.severity]}>{record.severity}</Tag>
      },
    },
    { title: '触发条件', dataIndex: 'condition', width: 150 },
    { title: '持续时间', dataIndex: 'duration', width: 100 },
    {
      title: '通知方式',
      dataIndex: 'receivers',
      width: 150,
      render: (_, record) =>
        record.receivers?.length ? record.receivers.map((r) => <Tag key={r}>{r}</Tag>) : '-',
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 80,
      render: (_, record) => (
        <Tag
          color={record.enabled ? 'green' : 'default'}
          style={{ cursor: 'pointer' }}
          onClick={() => toggleMutation.mutate({ id: record.id, enabled: !record.enabled })}
        >
          {record.enabled ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 180,
      render: (_, record) => formatDate(record.createdAt),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 120,
      render: (_, record) => (
        <Space>
          <a onClick={() => handleEdit(record)}>编辑</a>
          <Popconfirm title="确定删除该告警规则？" onConfirm={() => deleteMutation.mutate(record.id)}>
            <a style={{ color: '#ff4d4f' }}>删除</a>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <AppPage>
      <ProTable<AlertRule>
        headerTitle="告警规则"
        columns={columns}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey="id"
        search={false}
        pagination={{ pageSize: 10, showTotal: (total) => `共 ${total} 条` }}
        toolBarRender={() => [
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateVisible(true)}
          >
            创建规则
          </Button>,
        ]}
      />

      {/* 创建告警规则 */}
      <ModalForm
        title="创建告警规则"
        open={createVisible}
        onOpenChange={setCreateVisible}
        onFinish={async (values) => {
          createMutation.mutate({
            name: values.name,
            clusterId: values.clusterId || 1,
            severity: values.severity,
            condition: values.condition,
            duration: values.duration || '5m',
            receivers: values.receivers || [],
          })
          return true
        }}
        width={520}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="规则名称" rules={[{ required: true }]} />
        <ProFormSelect
          name="severity"
          label="告警级别"
          rules={[{ required: true }]}
          options={[
            { label: '信息', value: 'info' },
            { label: '警告', value: 'warning' },
            { label: '严重', value: 'critical' },
          ]}
        />
        <ProFormText name="condition" label="触发条件" rules={[{ required: true }]} placeholder="如: cpu > 80%" />
        <ProFormText name="duration" label="持续时间" placeholder="如: 5m" initialValue="5m" />
        <ProFormSelect
          name="receivers"
          label="通知方式"
          mode="tags"
          placeholder="输入通知渠道"
        />
      </ModalForm>

      {/* 编辑告警规则 */}
      <ModalForm
        title="编辑告警规则"
        open={editVisible}
        onOpenChange={(v) => {
          setEditVisible(v)
          if (!v) setCurrentRule(null)
        }}
        onFinish={async (values) => {
          if (currentRule) {
            updateMutation.mutate({ id: currentRule.id, data: values })
          }
          return true
        }}
        width={520}
        modalProps={{ destroyOnClose: true }}
        initialValues={currentRule || {}}
      >
        <ProFormText name="name" label="规则名称" rules={[{ required: true }]} />
        <ProFormSelect
          name="severity"
          label="告警级别"
          rules={[{ required: true }]}
          options={[
            { label: '信息', value: 'info' },
            { label: '警告', value: 'warning' },
            { label: '严重', value: 'critical' },
          ]}
        />
        <ProFormText name="condition" label="触发条件" rules={[{ required: true }]} placeholder="如: cpu > 80%" />
        <ProFormText name="duration" label="持续时间" placeholder="如: 5m" />
        <ProFormSelect
          name="receivers"
          label="通知方式"
          mode="tags"
          placeholder="输入通知渠道"
        />
      </ModalForm>
    </AppPage>
  )
}

export default AlertRulesPage
