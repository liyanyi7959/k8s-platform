import React, { useMemo, useState } from 'react'
import {
  ModalForm,
  ProForm,
  ProFormDependency,
  ProFormDigit,
  ProFormSelect,
  ProFormText,
  ProTable,
  type ProColumns,
} from '@ant-design/pro-components'
import { Alert, Button, Input, message, Popconfirm, Select, Space, Switch, Tag, Typography } from 'antd'
import { BellOutlined, DeleteOutlined, EditOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { AppPage } from '@/components'
import { listClusters } from '@/services/clusters'
import { createAlertRule, deleteAlertRule, listAlertRules, toggleAlertRule, updateAlertRule } from '@/services/monitor'
import { formatDate } from '@/utils'
import type { AlertRule, CreateAlertRuleRequest } from '@/types'

const { Text } = Typography

const METRICS = [
  { value: 'cpu_usage', label: 'CPU 使用率', unit: '%' },
  { value: 'memory_usage', label: '内存使用率', unit: '%' },
  { value: 'pod_restarts', label: 'Pod 重启次数', unit: ' 次' },
  { value: 'pod_failed', label: '失败 Pod 数量', unit: ' 个' },
  { value: 'node_not_ready', label: '未就绪节点数量', unit: ' 个' },
] as const

const severityMeta = {
  info: { label: '提示', color: 'blue' },
  warning: { label: '警告', color: 'orange' },
  critical: { label: '严重', color: 'red' },
}

const metricMeta = (metric?: string) => METRICS.find((item) => item.value === metric)
const buildCondition = (values: Record<string, any>) => {
  const unit = metricMeta(values.metric)?.unit?.trim() || ''
  return `${values.metric} ${values.operator} ${values.threshold}${unit === '%' ? '%' : ''}`
}

const parseRuleValues = (rule?: AlertRule | null) => {
  const match = rule?.condition?.match(/^([a-z_]+)\s*(>=|<=|==|>|<)\s*([\d.]+)/i)
  return {
    name: rule?.name,
    clusterId: rule?.clusterId,
    severity: rule?.severity || 'warning',
    metric: match?.[1] || 'cpu_usage',
    operator: match?.[2] || '>',
    threshold: Number(match?.[3] || 80),
    duration: rule?.duration || '5m',
    receivers: rule?.receivers || [],
  }
}

const getErrorMessage = (error: unknown, fallback: string) =>
  error instanceof Error && error.message ? error.message : fallback

const RuleFields: React.FC<{ clusters: Array<{ label: string; value: number }> }> = ({ clusters }) => (
  <>
    <ProFormText name="name" label="规则名称" placeholder="例如：生产集群 CPU 持续过高" rules={[{ required: true, message: '请输入规则名称' }]} />
    <ProForm.Group>
      <ProFormSelect name="clusterId" label="监控集群" width="md" options={clusters} rules={[{ required: true, message: '请选择集群' }]} />
      <ProFormSelect
        name="severity"
        label="告警级别"
        width="sm"
        options={Object.entries(severityMeta).map(([value, meta]) => ({ value, label: meta.label }))}
        rules={[{ required: true }]}
      />
    </ProForm.Group>
    <ProForm.Group>
      <ProFormSelect name="metric" label="指标" width="md" options={METRICS.map(({ value, label }) => ({ value, label }))} rules={[{ required: true }]} />
      <ProFormSelect
        name="operator"
        label="运算符"
        width="xs"
        options={[
          { value: '>', label: '大于' },
          { value: '>=', label: '大于等于' },
          { value: '<', label: '小于' },
          { value: '<=', label: '小于等于' },
          { value: '==', label: '等于' },
        ]}
        rules={[{ required: true }]}
      />
      <ProFormDigit name="threshold" label="阈值" width="xs" min={0} fieldProps={{ precision: 2 }} rules={[{ required: true }]} />
    </ProForm.Group>
    <ProFormSelect
      name="duration"
      label="持续时间"
      width="sm"
      options={['1m', '5m', '10m', '15m', '30m', '1h'].map((value) => ({ value, label: value }))}
      rules={[{ required: true }]}
    />
    <ProFormSelect name="receivers" label="通知渠道" mode="tags" placeholder="输入邮件、群组或通知渠道后回车" />
    <ProFormDependency name={['metric', 'operator', 'threshold', 'duration']}>
      {(values) => {
        const metric = metricMeta(values.metric)
        const operator = { '>': '大于', '>=': '大于等于', '<': '小于', '<=': '小于等于', '==': '等于' }[values.operator as string]
        return (
          <Alert
            type="info"
            showIcon
            message="规则影响预览"
            description={metric && values.threshold !== undefined
              ? `当 ${metric.label}${operator || values.operator}${values.threshold}${metric.unit}，并持续 ${values.duration || '—'} 时触发告警。`
              : '选择指标和阈值后，这里会显示触发范围。'}
          />
        )
      }}
    </ProFormDependency>
  </>
)

const AlertRulesPage: React.FC = () => {
  const queryClient = useQueryClient()
  const [createVisible, setCreateVisible] = useState(false)
  const [currentRule, setCurrentRule] = useState<AlertRule | null>(null)
  const [keyword, setKeyword] = useState('')
  const [severity, setSeverity] = useState<string>()
  const [enabled, setEnabled] = useState<boolean>()

  const rulesQuery = useQuery({
    queryKey: ['alert-rules'],
    queryFn: ({ signal }) => listAlertRules({ page: 1, pageSize: 100 }, signal),
  })
  const clustersQuery = useQuery({
    queryKey: ['alert-rule-clusters'],
    queryFn: ({ signal }) => listClusters({ page: 1, pageSize: 100 }, signal),
  })

  const clusterOptions = (clustersQuery.data?.items || []).map((cluster) => ({ label: cluster.name, value: cluster.id }))
  const filteredRules = useMemo(
    () => (rulesQuery.data?.items || []).filter((rule) => {
      const matchesKeyword = !keyword || `${rule.name} ${rule.clusterName}`.toLowerCase().includes(keyword.trim().toLowerCase())
      return matchesKeyword && (!severity || rule.severity === severity) && (enabled === undefined || rule.enabled === enabled)
    }),
    [enabled, keyword, rulesQuery.data, severity],
  )

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ['alert-rules'] })
  const createMutation = useMutation({
    mutationFn: (data: CreateAlertRuleRequest) => createAlertRule(data),
    onSuccess: () => { message.success('告警规则已创建'); setCreateVisible(false); invalidate() },
    onError: (error) => message.error(getErrorMessage(error, '告警规则创建失败')),
  })
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: CreateAlertRuleRequest }) => updateAlertRule(id, data),
    onSuccess: () => { message.success('告警规则已更新'); setCurrentRule(null); invalidate() },
    onError: (error) => message.error(getErrorMessage(error, '告警规则更新失败')),
  })
  const deleteMutation = useMutation({
    mutationFn: deleteAlertRule,
    onSuccess: () => { message.success('告警规则已删除'); invalidate() },
    onError: (error) => message.error(getErrorMessage(error, '告警规则删除失败')),
  })
  const toggleMutation = useMutation({
    mutationFn: ({ id, nextEnabled }: { id: number; nextEnabled: boolean }) => toggleAlertRule(id, nextEnabled),
    onSuccess: (_, variables) => { message.success(variables.nextEnabled ? '规则已启用' : '规则已停用'); invalidate() },
    onError: (error) => message.error(getErrorMessage(error, '规则状态更新失败')),
  })

  const submitValues = (values: Record<string, any>): CreateAlertRuleRequest => ({
    name: values.name,
    clusterId: values.clusterId,
    severity: values.severity,
    condition: buildCondition(values),
    duration: values.duration,
    receivers: values.receivers || [],
  })

  const columns: ProColumns<AlertRule>[] = [
    {
      title: '规则',
      dataIndex: 'name',
      render: (_, rule) => <Space direction="vertical" size={2}><Text strong>{rule.name}</Text><Text type="secondary">{rule.clusterName || `集群 ${rule.clusterId}`}</Text></Space>,
    },
    { title: '级别', dataIndex: 'severity', width: 90, render: (_, rule) => <Tag color={severityMeta[rule.severity].color}>{severityMeta[rule.severity].label}</Tag> },
    {
      title: '触发条件',
      dataIndex: 'condition',
      render: (_, rule) => {
        const parsed = parseRuleValues(rule)
        const metric = metricMeta(parsed.metric)
        return <Space direction="vertical" size={2}><Text>{metric?.label || rule.condition} {parsed.operator} {parsed.threshold}{metric?.unit}</Text><Text type="secondary">持续 {rule.duration}</Text></Space>
      },
    },
    { title: '通知渠道', dataIndex: 'receivers', width: 180, render: (_, rule) => rule.receivers?.length ? <Space size={[4, 4]} wrap>{rule.receivers.map((receiver) => <Tag key={receiver}>{receiver}</Tag>)}</Space> : <Text type="secondary">未配置</Text> },
    {
      title: '状态', dataIndex: 'enabled', width: 120,
      render: (_, rule) => <Switch checked={rule.enabled} checkedChildren="已启用" unCheckedChildren="已停用" loading={toggleMutation.isPending && toggleMutation.variables?.id === rule.id} onChange={(nextEnabled) => toggleMutation.mutate({ id: rule.id, nextEnabled })} />,
    },
    { title: '最近更新', dataIndex: 'updatedAt', width: 170, render: (_, rule) => formatDate(rule.updatedAt) },
    {
      title: '操作', valueType: 'option', width: 130,
      render: (_, rule) => <Space>
        <Button type="link" size="small" icon={<EditOutlined />} onClick={() => setCurrentRule(rule)}>编辑</Button>
        <Popconfirm title="删除后将停止维护这条规则，确定删除？" onConfirm={() => deleteMutation.mutate(rule.id)}>
          <Button type="link" danger size="small" icon={<DeleteOutlined />}>删除</Button>
        </Popconfirm>
      </Space>,
    },
  ]

  return (
    <AppPage keepHeaderTitle title="告警规则">
      <div className="app-page-shell">
        <div className="app-data-provenance">
          <span><BellOutlined />规则数据来自平台告警规则库</span>
          <span>当前 {rulesQuery.data?.total || 0} 条</span>
          <span>创建规则前可预览指标、阈值和持续时间</span>
        </div>
        <section className="app-console-filters">
          <div className="app-console-filters__left">
            <Input allowClear prefix={<SearchOutlined />} placeholder="搜索规则或集群" value={keyword} onChange={(event) => setKeyword(event.target.value)} className="app-console-filters__search" />
            <Select allowClear placeholder="告警级别" value={severity} onChange={setSeverity} options={Object.entries(severityMeta).map(([value, meta]) => ({ value, label: meta.label }))} className="app-console-filters__select" />
            <Select allowClear placeholder="启用状态" value={enabled} onChange={setEnabled} options={[{ value: true, label: '已启用' }, { value: false, label: '已停用' }]} className="app-console-filters__select" />
          </div>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>创建规则</Button>
        </section>
        <ProTable<AlertRule>
          className="app-console-table"
          headerTitle="规则清单"
          columns={columns}
          dataSource={filteredRules}
          loading={rulesQuery.isLoading}
          rowKey="id"
          search={false}
          options={false}
          toolBarRender={false}
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => `共 ${total} 条规则` }}
        />
      </div>

      <ModalForm
        title="创建告警规则"
        open={createVisible}
        onOpenChange={setCreateVisible}
        initialValues={parseRuleValues()}
        width={720}
        modalProps={{ destroyOnClose: true }}
        submitter={{ searchConfig: { submitText: '创建规则', resetText: '取消' }, submitButtonProps: { loading: createMutation.isPending } }}
        onFinish={async (values) => { await createMutation.mutateAsync(submitValues(values)); return true }}
      >
        <RuleFields clusters={clusterOptions} />
      </ModalForm>

      <ModalForm
        key={currentRule?.id || 'edit-rule'}
        title="编辑告警规则"
        open={Boolean(currentRule)}
        onOpenChange={(open) => { if (!open) setCurrentRule(null) }}
        initialValues={parseRuleValues(currentRule)}
        width={720}
        modalProps={{ destroyOnClose: true }}
        submitter={{ searchConfig: { submitText: '保存修改', resetText: '取消' }, submitButtonProps: { loading: updateMutation.isPending } }}
        onFinish={async (values) => { if (!currentRule) return false; await updateMutation.mutateAsync({ id: currentRule.id, data: submitValues(values) }); return true }}
      >
        <RuleFields clusters={clusterOptions} />
      </ModalForm>
    </AppPage>
  )
}

export default AlertRulesPage
