import React, { useDeferredValue, useMemo, useState } from 'react'
import { Alert, Button, Card, Col, Descriptions, Drawer, Dropdown, Form, Input, Modal, Row, Select, Space, Statistic, Table, Tabs, Tag, Tooltip, Typography, message } from 'antd'
import { DeleteOutlined, EyeOutlined, HistoryOutlined, MoreOutlined, PlusOutlined, ReloadOutlined, SearchOutlined, SyncOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage, NamespaceSelector, YamlEditor } from '@/components'
import { getHelmReleaseDetail, helmInstall, helmRollback, helmUninstall, helmUpgrade, listHelmReleases } from '@/services/k8s'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'

const { Text, Title } = Typography
const statusColorMap: Record<string, string> = { deployed: 'success', failed: 'error', superseded: 'default', uninstalled: 'default', 'pending-upgrade': 'warning', 'pending-rollback': 'warning', 'pending-install': 'processing' }

const HelmReleasesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState('')
  const [search, setSearch] = useState('')
  const deferredSearch = useDeferredValue(search.trim().toLowerCase())
  const [detailTarget, setDetailTarget] = useState<any>()
  const [installOpen, setInstallOpen] = useState(false)
  const [upgradeTarget, setUpgradeTarget] = useState<any>()
  const [rollbackTarget, setRollbackTarget] = useState<any>()
  const [valuesYaml, setValuesYaml] = useState('')
  const [installForm] = Form.useForm()
  const [upgradeForm] = Form.useForm()
  const [rollbackForm] = Form.useForm()

  const releasesQuery = useQuery({
    queryKey: ['helm-releases', clusterId, namespace],
    queryFn: ({ signal }) => listHelmReleases(clusterId, namespace || undefined, signal),
    enabled: !!clusterId,
    staleTime: 30_000,
    gcTime: 10 * 60_000,
    refetchOnWindowFocus: false,
  })
  const detailQuery = useQuery({
    queryKey: ['helm-release-detail', clusterId, detailTarget?.namespace, detailTarget?.name],
    queryFn: ({ signal }) => getHelmReleaseDetail(clusterId, detailTarget.namespace, detailTarget.name, signal),
    enabled: !!detailTarget,
    staleTime: 30_000,
  })

  const refresh = () => queryClient.invalidateQueries({ queryKey: ['helm-releases', clusterId] })
  const closeForms = () => { setInstallOpen(false); setUpgradeTarget(undefined); installForm.resetFields(); upgradeForm.resetFields(); setValuesYaml('') }
  const operationMutation = useMutation({
    mutationFn: async (operation: { type: 'install' | 'upgrade' | 'rollback' | 'uninstall'; payload: any }) => {
      if (operation.type === 'install') return helmInstall(clusterId, operation.payload)
      if (operation.type === 'upgrade') return helmUpgrade(clusterId, operation.payload.namespace, operation.payload.name, operation.payload)
      if (operation.type === 'rollback') return helmRollback(clusterId, operation.payload.namespace, operation.payload.name, operation.payload.revision)
      return helmUninstall(clusterId, operation.payload.namespace, operation.payload.name)
    },
    onSuccess: (_, operation) => { message.success({ install: '安装', upgrade: '升级', rollback: '回滚', uninstall: '卸载' }[operation.type] + '操作已完成'); closeForms(); setRollbackTarget(undefined); refresh() },
    onError: (error: any) => message.error(error?.message || 'Helm 操作失败'),
  })

  const releases = useMemo(() => (releasesQuery.data?.items || []).filter((item: any) => !deferredSearch || `${item.name} ${item.namespace} ${item.chart} ${item.status}`.toLowerCase().includes(deferredSearch)), [releasesQuery.data, deferredSearch])
  const stats = useMemo(() => ({ total: releases.length, deployed: releases.filter((item: any) => item.status === 'deployed').length, failed: releases.filter((item: any) => item.status === 'failed').length, pending: releases.filter((item: any) => String(item.status).startsWith('pending')).length }), [releases])

  const submitInstall = async () => {
    const values = await installForm.validateFields()
    operationMutation.mutate({ type: 'install', payload: { ...values, values_yaml: valuesYaml } })
  }
  const submitUpgrade = async () => {
    const values = await upgradeForm.validateFields()
    operationMutation.mutate({ type: 'upgrade', payload: { ...values, namespace: upgradeTarget.namespace, name: upgradeTarget.name, values_yaml: valuesYaml } })
  }
  const submitRollback = async () => {
    const values = await rollbackForm.validateFields()
    operationMutation.mutate({ type: 'rollback', payload: { ...values, namespace: rollbackTarget.namespace, name: rollbackTarget.name } })
  }

  const columns: any[] = [
    { title: 'Release', dataIndex: 'name', width: 210, fixed: 'left', ellipsis: true, render: (value: string, record: any) => <Button type="link" style={{ padding: 0 }} onClick={() => setDetailTarget(record)}>{value}</Button> },
    { title: '命名空间', dataIndex: 'namespace', width: 150, render: (value: string) => <Tag>{value}</Tag> },
    { title: 'Revision', dataIndex: 'revision', width: 90, align: 'center' },
    { title: '状态', dataIndex: 'status', width: 140, render: (value: string) => <Tag color={statusColorMap[value] || 'default'}>{value || 'unknown'}</Tag> },
    { title: 'Chart', dataIndex: 'chart', width: 240, ellipsis: true, render: (value: string) => value || '-' },
    { title: 'App 版本', dataIndex: 'app_version', width: 130, render: (value: string) => value || '-' },
    { title: '更新时间', dataIndex: 'updated', width: 200, render: (value: string) => value ? formatDate(value) : '-' },
    { title: '操作', width: 150, fixed: 'right', render: (_: any, record: any) => <Space>
      <Tooltip title="查看详情"><Button type="text" icon={<EyeOutlined />} onClick={() => setDetailTarget(record)} /></Tooltip>
      <Tooltip title="升级"><Button type="text" icon={<SyncOutlined />} onClick={() => { setUpgradeTarget(record); setValuesYaml(''); upgradeForm.setFieldsValue({ chart: record.chart?.replace(/-[^-]+$/, '') || '', atomic: true, wait: true, timeout: '5m' }) }} /></Tooltip>
      <Dropdown trigger={['click']} menu={{ items: [
        { key: 'rollback', icon: <HistoryOutlined />, label: '回滚到历史版本', onClick: () => { setRollbackTarget(record); rollbackForm.resetFields() } },
        { key: 'uninstall', danger: true, icon: <DeleteOutlined />, label: '卸载 Release', onClick: () => Modal.confirm({ title: `卸载 ${record.namespace}/${record.name}？`, content: 'Release 管理的资源将被删除，此操作不可撤销。', okText: '确认卸载', okButtonProps: { danger: true }, onOk: () => operationMutation.mutateAsync({ type: 'uninstall', payload: record }) }) },
      ] }}><Button type="text" icon={<MoreOutlined />} /></Dropdown>
    </Space> },
  ]

  return <AppPage>
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <Card size="small">
        <Row justify="space-between" align="middle" gutter={[12, 12]}>
          <Col><Title level={4} style={{ margin: 0 }}>Helm Releases</Title><Text type="secondary">Release 生命周期、配置、清单与版本历史统一管理</Text></Col>
          <Col><Space><Button icon={<ReloadOutlined />} loading={releasesQuery.isFetching} onClick={() => releasesQuery.refetch()}>刷新</Button><Button type="primary" icon={<PlusOutlined />} onClick={() => setInstallOpen(true)}>安装 Chart</Button></Space></Col>
        </Row>
        <Row gutter={[12, 12]} style={{ marginTop: 16 }}>
          <Col xs={24} lg={12}><Space.Compact block><NamespaceSelector clusterId={clusterId} value={namespace} onChange={setNamespace} style={{ width: 220 }} /><Input allowClear prefix={<SearchOutlined />} placeholder="搜索 Release、Chart 或状态" value={search} onChange={(event) => setSearch(event.target.value)} /></Space.Compact></Col>
          <Col xs={6} lg={3}><Statistic title="全部" value={stats.total} /></Col><Col xs={6} lg={3}><Statistic title="已部署" value={stats.deployed} valueStyle={{ color: '#16a34a' }} /></Col><Col xs={6} lg={3}><Statistic title="失败" value={stats.failed} valueStyle={{ color: '#dc2626' }} /></Col><Col xs={6} lg={3}><Statistic title="处理中" value={stats.pending} valueStyle={{ color: '#d97706' }} /></Col>
        </Row>
      </Card>
      {releasesQuery.data?.source === 'kubernetes-secrets' && <Alert type="warning" showIcon message="Helm CLI 数据源暂不可用，当前使用 Kubernetes Secret 降级数据；Chart 与应用版本字段可能不完整。" />}
      <Card size="small" styles={{ body: { padding: 0 } }}>
        <Table rowKey={(record) => `${record.namespace}/${record.name}`} loading={releasesQuery.isLoading} dataSource={releases} columns={columns} scroll={{ x: 1320 }} pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (total) => `共 ${total} 个 Release` }} />
      </Card>
    </Space>

    <Drawer width="min(960px, 92vw)" title={detailTarget ? `${detailTarget.namespace}/${detailTarget.name}` : 'Release 详情'} open={!!detailTarget} onClose={() => setDetailTarget(undefined)} loading={detailQuery.isLoading}>
      {detailQuery.data && <Tabs items={[
        { key: 'overview', label: '概览', children: <Descriptions bordered column={2} size="small"><Descriptions.Item label="名称">{detailQuery.data.name}</Descriptions.Item><Descriptions.Item label="命名空间">{detailQuery.data.namespace}</Descriptions.Item><Descriptions.Item label="Revision">{detailQuery.data.revision}</Descriptions.Item><Descriptions.Item label="状态"><Tag color={statusColorMap[detailQuery.data.status]}>{detailQuery.data.status}</Tag></Descriptions.Item><Descriptions.Item label="Chart">{detailQuery.data.chart || '-'}</Descriptions.Item><Descriptions.Item label="更新时间">{detailQuery.data.updated ? formatDate(detailQuery.data.updated) : '-'}</Descriptions.Item></Descriptions> },
        { key: 'values', label: 'Values', children: <YamlEditor readOnly value={detailQuery.data.values_yaml || '# 未配置自定义 values'} height={560} /> },
        { key: 'manifest', label: '渲染清单', children: <YamlEditor readOnly value={detailQuery.data.manifest || '# 暂无渲染清单'} height={560} /> },
        { key: 'history', label: `版本历史 (${detailQuery.data.history?.length || 0})`, children: <Table size="small" rowKey="revision" pagination={false} dataSource={detailQuery.data.history || []} columns={[{ title: 'Revision', dataIndex: 'revision' }, { title: '状态', dataIndex: 'status', render: (value) => <Tag>{value}</Tag> }, { title: 'Chart', dataIndex: 'chart' }, { title: '更新时间', dataIndex: 'updated' }, { title: '说明', dataIndex: 'description', ellipsis: true }]} /> },
      ]} />}
    </Drawer>

    <Modal title="安装 Helm Chart" width={760} open={installOpen} onCancel={closeForms} onOk={submitInstall} confirmLoading={operationMutation.isPending} okText="安装">
      <Form form={installForm} layout="vertical" initialValues={{ namespace: 'default' }}><Row gutter={12}><Col span={12}><Form.Item name="release_name" label="Release 名称" rules={[{ required: true }]}><Input placeholder="my-redis" /></Form.Item></Col><Col span={12}><Form.Item name="namespace" label="命名空间" rules={[{ required: true }]}><Input placeholder="default" /></Form.Item></Col></Row><Form.Item name="chart" label="Chart" rules={[{ required: true }]}><Input placeholder="bitnami/redis" /></Form.Item><Row gutter={12}><Col span={12}><Form.Item name="repo_name" label="仓库名称（首次添加时填写）"><Input placeholder="bitnami" /></Form.Item></Col><Col span={12}><Form.Item name="repo_url" label="仓库地址"><Input placeholder="https://charts.bitnami.com/bitnami" /></Form.Item></Col></Row><Form.Item label="values.yaml"><YamlEditor value={valuesYaml} onChange={setValuesYaml} height={280} /></Form.Item></Form>
    </Modal>
    <Modal title={`升级 ${upgradeTarget?.namespace}/${upgradeTarget?.name}`} width={760} open={!!upgradeTarget} onCancel={closeForms} onOk={submitUpgrade} confirmLoading={operationMutation.isPending} okText="执行升级">
      <Alert type="info" showIcon message="建议启用 Atomic：升级失败时 Helm 会自动回滚。" style={{ marginBottom: 16 }} />
      <Form form={upgradeForm} layout="vertical"><Row gutter={12}><Col span={16}><Form.Item name="chart" label="Chart" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={8}><Form.Item name="version" label="Chart 版本"><Input placeholder="留空使用最新版本" /></Form.Item></Col></Row><Row gutter={12}><Col span={8}><Form.Item name="atomic" label="失败自动回滚"><Select options={[{ label: '启用', value: true }, { label: '关闭', value: false }]} /></Form.Item></Col><Col span={8}><Form.Item name="wait" label="等待资源就绪"><Select options={[{ label: '启用', value: true }, { label: '关闭', value: false }]} /></Form.Item></Col><Col span={8}><Form.Item name="timeout" label="超时"><Input placeholder="5m" /></Form.Item></Col></Row><Form.Item label="values.yaml"><YamlEditor value={valuesYaml} onChange={setValuesYaml} height={280} /></Form.Item></Form>
    </Modal>
    <Modal title={`回滚 ${rollbackTarget?.namespace}/${rollbackTarget?.name}`} open={!!rollbackTarget} onCancel={() => setRollbackTarget(undefined)} onOk={submitRollback} confirmLoading={operationMutation.isPending} okButtonProps={{ danger: true }} okText="确认回滚"><Alert type="warning" showIcon message="回滚会重新应用目标 revision 的 Chart 与 values。" style={{ marginBottom: 16 }} /><Form form={rollbackForm} layout="vertical"><Form.Item name="revision" label="目标 Revision" rules={[{ required: true }]}><Input type="number" min={1} placeholder="例如 3" /></Form.Item></Form></Modal>
  </AppPage>
}

export default HelmReleasesPage
