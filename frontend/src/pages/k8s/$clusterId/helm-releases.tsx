import React, { useDeferredValue, useEffect, useMemo, useState } from 'react'
import { history, useModel, useSearchParams } from '@umijs/max'
import {
  Button,
  Col,
  Descriptions,
  Drawer,
  Dropdown,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd'
import {
  DeleteOutlined,
  EyeOutlined,
  HistoryOutlined,
  MoreOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  SyncOutlined,
} from '@ant-design/icons'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage, NamespaceSelector, YamlEditor } from '@/components'
import AppAlert from '@/components/AppAlert'
import {
  getHelmReleaseDetail,
  helmInstall,
  helmRollback,
  helmUninstall,
  helmUpgrade,
  listHelmReleases,
  listHelmRepos,
  listNamespaces,
} from '@/services/k8s'
import { getClusterById } from '@/services/clusters'
import { listAppTemplates, type AppTemplate } from '@/services/app-template'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'

const { Text } = Typography

const statusColorMap: Record<string, string> = {
  deployed: 'success',
  failed: 'error',
  superseded: 'default',
  uninstalled: 'default',
  'pending-upgrade': 'warning',
  'pending-rollback': 'warning',
  'pending-install': 'processing',
}

type InstallFormValues = {
  release_name: string
  namespace: string
  chart: string
  version?: string
  repo_name?: string
  repo_url?: string
}

function releaseNameFromChart(chart: string) {
  const candidate = chart.split('/').pop() || ''
  return candidate
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 53)
}

const HelmReleasesPage: React.FC = () => {
  const clusterId = useClusterId()
  const [searchParams] = useSearchParams()
  const { currentCluster } = useModel('cluster')
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState('')
  const [search, setSearch] = useState('')
  const deferredSearch = useDeferredValue(search.trim().toLowerCase())
  const [detailTarget, setDetailTarget] = useState<any>()
  const [installOpen, setInstallOpen] = useState(false)
  const [sourceMode, setSourceMode] = useState<'catalog' | 'cluster' | 'internet'>('catalog')
  const [selectedCatalogID, setSelectedCatalogID] = useState<number>()
  const [upgradeTarget, setUpgradeTarget] = useState<any>()
  const [rollbackTarget, setRollbackTarget] = useState<any>()
  const [valuesYaml, setValuesYaml] = useState('')
  const [installForm] = Form.useForm<InstallFormValues>()
  const [upgradeForm] = Form.useForm()
  const [rollbackForm] = Form.useForm()

  const clusterQuery = useQuery({
    queryKey: ['cluster', clusterId],
    queryFn: ({ signal }) => getClusterById(clusterId, signal),
    enabled: !!clusterId,
    staleTime: 60_000,
  })
  const namespacesQuery = useQuery({
    queryKey: ['helm-install-namespaces', clusterId],
    queryFn: ({ signal }) => listNamespaces(clusterId, signal),
    enabled: !!clusterId && installOpen,
    staleTime: 30_000,
  })
  const repositoriesQuery = useQuery({
    queryKey: ['helm-repos', clusterId],
    queryFn: ({ signal }) => listHelmRepos(clusterId, signal),
    enabled: !!clusterId && installOpen,
    staleTime: 30_000,
  })
  const catalogQuery = useQuery({
    queryKey: ['app-templates', 'helm-catalog'],
    queryFn: () => listAppTemplates(),
    enabled: installOpen,
    staleTime: 60_000,
  })
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

  const namespaces = useMemo(
    () => (namespacesQuery.data || []).map((item) => item.name).filter(Boolean),
    [namespacesQuery.data],
  )
  const repositories = repositoriesQuery.data || []
  const catalogTemplates = useMemo(
    () => (catalogQuery.data?.list || []).filter((item) => item.deploy_type === 'helm'),
    [catalogQuery.data?.list],
  )
  const clusterName = clusterQuery.data?.name || currentCluster?.name || `集群 #${clusterId}`
  const clusterVersion = clusterQuery.data?.k8sVersion || currentCluster?.version

  const invalidateHelmData = () => {
    queryClient.invalidateQueries({ queryKey: ['helm-releases', clusterId] })
    queryClient.invalidateQueries({ queryKey: ['helm-repos', clusterId] })
  }

  const closeForms = () => {
    setInstallOpen(false)
    setUpgradeTarget(undefined)
    installForm.resetFields()
    upgradeForm.resetFields()
    setValuesYaml('')
    setSelectedCatalogID(undefined)
  }

  useEffect(() => {
    if (!installOpen || installForm.getFieldValue('namespace')) return
    const defaultNamespace = namespaces.includes('default') ? 'default' : namespaces[0] || 'default'
    installForm.setFieldValue('namespace', defaultNamespace)
  }, [installForm, installOpen, namespaces])

  const applyCatalogTemplate = (template?: AppTemplate) => {
    if (!template) return
    const chart = template.template || ''
    installForm.setFieldsValue({
      chart,
      version: template.helm_chart_version || '',
      repo_name: template.helm_repo_name || '',
      repo_url: template.helm_repo_url || '',
      release_name: releaseNameFromChart(chart),
    })
    setValuesYaml(template.helm_values_yaml || '')
  }

  const changeSourceMode = (mode: 'catalog' | 'cluster' | 'internet') => {
    setSourceMode(mode)
    if (mode === 'catalog') {
      setSelectedCatalogID(undefined)
      installForm.setFieldsValue({ chart: '', version: '', repo_name: '', repo_url: '', release_name: '' })
      setValuesYaml('')
      return
    }
    setSelectedCatalogID(undefined)
    installForm.setFieldsValue({ repo_name: '', repo_url: '' })
  }

  useEffect(() => {
    if (!installOpen || selectedCatalogID || !catalogTemplates.length) return
    const catalogID = Number(searchParams.get('app_template_id'))
    const template = catalogTemplates.find((item) => item.id === catalogID)
    if (template) {
      setSourceMode('catalog')
      setSelectedCatalogID(template.id)
      applyCatalogTemplate(template)
    }
  }, [catalogTemplates, installOpen, searchParams, selectedCatalogID])

  const operationMutation = useMutation({
    mutationFn: async (operation: { type: 'install' | 'upgrade' | 'rollback' | 'uninstall'; payload: any }) => {
      if (operation.type === 'install') return helmInstall(clusterId, operation.payload)
      if (operation.type === 'upgrade') return helmUpgrade(clusterId, operation.payload.namespace, operation.payload.name, operation.payload)
      if (operation.type === 'rollback') return helmRollback(clusterId, operation.payload.namespace, operation.payload.name, operation.payload.revision)
      return helmUninstall(clusterId, operation.payload.namespace, operation.payload.name)
    },
    onSuccess: (result: any, operation) => {
      const actionName = { install: '安装', upgrade: '升级', rollback: '回滚', uninstall: '卸载' }[operation.type]
      const autoInstalled = operation.type === 'install' && result?.master?.installed_now
      message.success(`${actionName}完成${autoInstalled ? '，Master 上的 Helm 已自动准备' : ''}`)
      closeForms()
      setRollbackTarget(undefined)
      invalidateHelmData()
    },
    onError: (error: any) => message.error(error?.message || 'Helm 操作失败'),
  })

  const releases = useMemo(
    () => (releasesQuery.data?.items || []).filter((item: any) => !deferredSearch || `${item.name} ${item.namespace} ${item.chart} ${item.status}`.toLowerCase().includes(deferredSearch)),
    [releasesQuery.data, deferredSearch],
  )

  const openInstall = () => {
    const chart = searchParams.get('chart')?.trim() || ''
    const repoName = searchParams.get('repo_name')?.trim() || ''
    const repoURL = searchParams.get('repo_url')?.trim() || ''
    const version = searchParams.get('version')?.trim() || ''
    const preferredNamespace = namespaces.includes('default') ? 'default' : namespaces[0] || 'default'
    installForm.resetFields()
    installForm.setFieldsValue({
      release_name: releaseNameFromChart(chart),
      namespace: preferredNamespace,
      chart,
      version,
      repo_name: repoName,
      repo_url: repoURL,
    })
    setValuesYaml('')
    setSelectedCatalogID(undefined)
    setSourceMode(repoURL ? 'internet' : 'catalog')
    setInstallOpen(true)
  }

  const submitInstall = async () => {
    const values = await installForm.validateFields()
    if (sourceMode === 'catalog' && !selectedCatalogID) {
      message.warning('请选择 AIOPS 应用目录中的 Helm Chart')
      return
    }
    const matchedRepository = repositories.find((repository) => repository.name === values.repo_name)
    const repoName = values.repo_name?.trim() || ''
    const repoURL = values.repo_url?.trim() || matchedRepository?.url || ''
    let chart = values.chart.trim()
    if (repoName && !chart.includes('/')) chart = `${repoName}/${chart}`
    operationMutation.mutate({
      type: 'install',
      payload: { ...values, chart, repo_name: repoName, repo_url: repoURL, values_yaml: valuesYaml },
    })
  }

  const submitUpgrade = async () => {
    const values = await upgradeForm.validateFields()
    operationMutation.mutate({ type: 'upgrade', payload: { ...values, namespace: upgradeTarget.namespace, name: upgradeTarget.name, values_yaml: valuesYaml } })
  }
  const submitRollback = async () => {
    const values = await rollbackForm.validateFields()
    operationMutation.mutate({ type: 'rollback', payload: { ...values, namespace: rollbackTarget.namespace, name: rollbackTarget.name } })
  }

  const columns: ProColumns<any>[] = [
    { title: 'Release', dataIndex: 'name', width: 210, fixed: 'left', ellipsis: true, render: (value: string, record: any) => <Button type="link" style={{ padding: 0 }} onClick={() => setDetailTarget(record)}>{value}</Button> },
    { title: '命名空间', dataIndex: 'namespace', width: 150, render: (value: string) => <Tag>{value}</Tag> },
    { title: 'Revision', dataIndex: 'revision', width: 90, align: 'center' },
    { title: '状态', dataIndex: 'status', width: 140, render: (value: string) => <Tag color={statusColorMap[value] || 'default'}>{value || 'unknown'}</Tag> },
    { title: 'Chart', dataIndex: 'chart', width: 240, ellipsis: true, render: (value: string) => value || '-' },
    { title: 'App 版本', dataIndex: 'app_version', width: 130, render: (value: string) => value || '-' },
    { title: '更新时间', dataIndex: 'updated', width: 200, render: (value: string) => value ? formatDate(value) : '-' },
    {
      title: '操作', width: 150, fixed: 'right', render: (_: any, record: any) => <Space>
        <Tooltip title="查看详情"><Button type="text" icon={<EyeOutlined />} onClick={() => setDetailTarget(record)} /></Tooltip>
        <Tooltip title="升级"><Button type="text" icon={<SyncOutlined />} onClick={() => {
          setUpgradeTarget(record)
          setValuesYaml('')
          upgradeForm.setFieldsValue({ chart: record.chart?.replace(/-[^-]+$/, '') || '', atomic: true, wait: true, timeout: '5m' })
        }} /></Tooltip>
        <Dropdown trigger={['click']} menu={{ items: [
          { key: 'rollback', icon: <HistoryOutlined />, label: '回滚到历史版本', onClick: () => { setRollbackTarget(record); rollbackForm.resetFields() } },
          { key: 'uninstall', danger: true, icon: <DeleteOutlined />, label: '卸载 Release', onClick: () => Modal.confirm({ title: `卸载 ${record.namespace}/${record.name}`, content: 'Release 管理的资源将被删除，此操作不可撤销。', okText: '确认卸载', okButtonProps: { danger: true }, onOk: () => operationMutation.mutateAsync({ type: 'uninstall', payload: record }) }) },
        ] }}><Button type="text" icon={<MoreOutlined />} /></Dropdown>
      </Space>,
    },
  ]

  return <AppPage>
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      {releasesQuery.isError && <AppAlert type="error" showIcon message="无法读取当前集群的 Helm Release" description={(releasesQuery.error as Error)?.message || '请检查集群 API 连接与访问权限。'} />}
      <ProTable<any>
        headerTitle={<Space size={8}><span>Helm 发布</span><Tag color="blue">{clusterName}</Tag>{clusterVersion ? <Text type="secondary">Kubernetes {clusterVersion}</Text> : null}</Space>}
        rowKey={(record) => `${record.namespace}/${record.name}`}
        loading={releasesQuery.isLoading}
        dataSource={releases}
        columns={columns}
        search={false}
        options={{ reload: false }}
        scroll={{ x: 1320 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (total) => `共 ${total} 个 Release` }}
        toolBarRender={() => [
          <Input.Search
            key="search"
            allowClear
            placeholder="搜索 Release、Chart 或状态"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            style={{ width: 280 }}
            prefix={<SearchOutlined />}
          />,
          <NamespaceSelector key="ns" clusterId={clusterId} value={namespace} onChange={setNamespace} />,
          <Button key="refresh" icon={<ReloadOutlined />} loading={releasesQuery.isFetching} onClick={() => releasesQuery.refetch()}>刷新</Button>,
          <Button key="install" type="primary" icon={<PlusOutlined />} onClick={openInstall}>安装 Chart</Button>,
        ]}
      />
    </Space>

    <Drawer width="min(960px, 92vw)" title={detailTarget ? `${detailTarget.namespace}/${detailTarget.name}` : 'Release 详情'} open={!!detailTarget} onClose={() => setDetailTarget(undefined)} loading={detailQuery.isLoading}>
      {detailQuery.data && <Descriptions bordered column={2} size="small" title="概览">
        <Descriptions.Item label="名称">{detailQuery.data.name}</Descriptions.Item><Descriptions.Item label="命名空间">{detailQuery.data.namespace}</Descriptions.Item>
        <Descriptions.Item label="Revision">{detailQuery.data.revision}</Descriptions.Item><Descriptions.Item label="状态"><Tag color={statusColorMap[detailQuery.data.status]}>{detailQuery.data.status}</Tag></Descriptions.Item>
        <Descriptions.Item label="Chart">{detailQuery.data.chart || '-'}</Descriptions.Item><Descriptions.Item label="更新时间">{detailQuery.data.updated ? formatDate(detailQuery.data.updated) : '-'}</Descriptions.Item>
      </Descriptions>}
      {detailQuery.data && <Table style={{ marginTop: 16 }} size="small" rowKey="revision" pagination={false} dataSource={detailQuery.data.history || []} columns={[{ title: 'Revision', dataIndex: 'revision' }, { title: '状态', dataIndex: 'status', render: (value) => <Tag>{value}</Tag> }, { title: 'Chart', dataIndex: 'chart' }, { title: '更新时间', dataIndex: 'updated' }, { title: '说明', dataIndex: 'description', ellipsis: true }]} />}
      {detailQuery.data && <div style={{ marginTop: 16 }}><Text strong>Values</Text><YamlEditor readOnly value={detailQuery.data.values_yaml || '# 未配置自定义 values'} height={360} /></div>}
      {detailQuery.data && <div style={{ marginTop: 16 }}><Text strong>渲染清单</Text><YamlEditor readOnly value={detailQuery.data.manifest || '# 暂无渲染清单'} height={480} /></div>}
    </Drawer>

    <Modal title="安装 Helm Chart" width={720} open={installOpen} onCancel={closeForms} onOk={submitInstall} confirmLoading={operationMutation.isPending} okText="开始安装" destroyOnClose>
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 24, padding: '12px 16px', marginBottom: 20, border: '1px solid #d9e8ff', borderRadius: 8, background: '#f6faff' }}>
        <div><Text type="secondary">目标集群</Text><div><Text strong>{clusterName}</Text>{clusterVersion ? <Tag color="blue" style={{ marginLeft: 8 }}>{clusterVersion}</Tag> : null}</div></div>
        <div><Text type="secondary">执行节点</Text><div><Text strong>Master</Text><Text type="secondary"> · Helm 自动校验</Text></div></div>
      </div>
      <Form form={installForm} layout="vertical">
        <Row gutter={12}>
          <Col span={12}><Form.Item name="release_name" label="Release 名称" rules={[{ required: true, message: '请输入 Release 名称' }]}><Input placeholder="例如 my-redis" /></Form.Item></Col>
          <Col span={12}><Form.Item name="namespace" label="命名空间" rules={[{ required: true, message: '请选择或输入命名空间' }]}><Select mode="tags" tokenSeparators={[',']} loading={namespacesQuery.isFetching} placeholder="选择已有命名空间或输入新名称" options={namespaces.map((item) => ({ label: item, value: item }))} /></Form.Item></Col>
        </Row>
        <Row gutter={12}>
          <Col span={16}><Form.Item name="chart" label="Chart" rules={[{ required: true, message: '请输入 Chart，例如 bitnami/redis' }]}><Input placeholder="例如 bitnami/redis" /></Form.Item></Col>
          <Col span={8}><Form.Item name="version" label="Chart 版本"><Input placeholder="留空使用仓库最新版本" /></Form.Item></Col>
        </Row>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
          <Text strong>Chart 来源</Text>
          <Space size={4}>
            <Button type={sourceMode === 'catalog' ? 'link' : 'text'} size="small" onClick={() => changeSourceMode('catalog')}>AIOPS 应用</Button>
            <Button type={sourceMode === 'cluster' ? 'link' : 'text'} size="small" onClick={() => changeSourceMode('cluster')}>集群仓库</Button>
            <Button type={sourceMode === 'internet' ? 'link' : 'text'} size="small" onClick={() => changeSourceMode('internet')}>互联网仓库</Button>
          </Space>
        </div>
        {sourceMode === 'catalog' ? <>
          <Select
            allowClear
            value={selectedCatalogID}
            loading={catalogQuery.isFetching}
            placeholder="选择 AIOPS 应用目录中的 Helm Chart"
            options={catalogTemplates.map((item) => ({ label: `${item.display_name || item.name} · ${item.template}`, value: item.id }))}
            onChange={(value) => {
              setSelectedCatalogID(value)
              applyCatalogTemplate(catalogTemplates.find((item) => item.id === value))
            }}
          />
          <Text type="secondary" style={{ fontSize: 12 }}>目录提供可编辑的默认 values.yaml；本次修改只作用于当前 Release。</Text>
        </> : sourceMode === 'cluster' ? <>
          <Form.Item name="repo_name" style={{ marginBottom: 6 }} rules={[{ required: true, message: '请选择当前集群仓库' }]}>
            <Select loading={repositoriesQuery.isFetching} placeholder="选择当前 Master 已同步的仓库" options={repositories.map((item) => ({ label: `${item.name} · ${item.url}`, value: item.name }))} onChange={(value) => installForm.setFieldValue('repo_url', repositories.find((item) => item.name === value)?.url || '')} />
          </Form.Item>
          <Text type="secondary" style={{ fontSize: 12 }}>仓库数据实时来自当前集群 Master。</Text>
        </> : <Row gutter={12}>
          <Col span={9}><Form.Item name="repo_name" rules={[{ required: true, message: '请输入仓库名称' }]}><Input placeholder="例如 bitnami" /></Form.Item></Col>
          <Col span={15}><Form.Item name="repo_url" rules={[{ required: true, type: 'url', message: '请输入 HTTPS 仓库地址' }]}><Input placeholder="https://charts.example.com/repo" /></Form.Item></Col>
        </Row>}
        <div style={{ marginBottom: 16 }}><Button type="link" size="small" style={{ paddingLeft: 0 }} onClick={() => history.push(`/k8s/${clusterId}/helm-repos`)}>管理当前集群 Helm 仓库</Button></div>
        <Form.Item label="values.yaml（可选）"><YamlEditor value={valuesYaml} onChange={setValuesYaml} height={220} /></Form.Item>
      </Form>
    </Modal>
    <Modal title={`升级 ${upgradeTarget?.namespace}/${upgradeTarget?.name}`} width={760} open={!!upgradeTarget} onCancel={closeForms} onOk={submitUpgrade} confirmLoading={operationMutation.isPending} okText="执行升级">
      <Form form={upgradeForm} layout="vertical"><Row gutter={12}><Col span={16}><Form.Item name="chart" label="Chart" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={8}><Form.Item name="version" label="Chart 版本"><Input placeholder="留空使用最新版本" /></Form.Item></Col></Row><Row gutter={12}><Col span={8}><Form.Item name="atomic" label="失败自动回滚"><Select options={[{ label: '启用', value: true }, { label: '关闭', value: false }]} /></Form.Item></Col><Col span={8}><Form.Item name="wait" label="等待资源就绪"><Select options={[{ label: '启用', value: true }, { label: '关闭', value: false }]} /></Form.Item></Col><Col span={8}><Form.Item name="timeout" label="超时"><Input placeholder="5m" /></Form.Item></Col></Row><Form.Item label="values.yaml"><YamlEditor value={valuesYaml} onChange={setValuesYaml} height={280} /></Form.Item></Form>
    </Modal>
    <Modal title={`回滚 ${rollbackTarget?.namespace}/${rollbackTarget?.name}`} open={!!rollbackTarget} onCancel={() => setRollbackTarget(undefined)} onOk={submitRollback} confirmLoading={operationMutation.isPending} okButtonProps={{ danger: true }} okText="确认回滚"><Form form={rollbackForm} layout="vertical"><Form.Item name="revision" label="目标 Revision" rules={[{ required: true }]}><Input type="number" min={1} placeholder="例如 3" /></Form.Item></Form></Modal>
  </AppPage>
}

export default HelmReleasesPage
