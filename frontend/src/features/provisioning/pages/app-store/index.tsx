/**
 * 应用商店：维护可复用的 YAML / Helm 应用目录。
 * 实际 Helm 仓库、Release 与安装动作始终在目标集群页面完成。
 */
import { useEffect, useMemo, useState } from 'react'
import { useLocation, useSearchParams } from '@umijs/max'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import {
  Button,
  Col,
  Drawer,
  Form,
  Input,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd'
import { DeleteOutlined, EditOutlined, EyeOutlined, PlusOutlined, ReloadOutlined, RocketOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage, YamlEditor } from '@/components'
import { ManifestApplyDrawer } from '@/features/kops'
import {
  createAppTemplate,
  deleteAppTemplate,
  listAppTemplates,
  updateAppTemplate,
  type AppTemplate,
} from '@/features/provisioning/api/app-template'
import { listClusters } from '@/features/fleet'
import { listProjects } from '@/features/workspace'
import { formatDate } from '@/utils'

const { Text, Paragraph } = Typography

const CATEGORIES = [
  { value: 'database', label: '数据库' },
  { value: 'middleware', label: '中间件' },
  { value: 'monitoring', label: '监控' },
  { value: 'devtool', label: '开发工具' },
  { value: 'networking', label: '网络' },
  { value: 'security', label: '安全' },
]

const CATEGORY_LABELS: Record<string, string> = Object.fromEntries(CATEGORIES.map((item) => [item.value, item.label]))
const CATEGORY_COLORS: Record<string, string> = {
  database: 'blue', middleware: 'purple', monitoring: 'green', devtool: 'orange', networking: 'cyan', security: 'red',
}

function parseVariables(variables: string): Array<{ name: string; default: string }> {
  try {
    const parsed = JSON.parse(variables || '[]')
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function splitNamespaces(value: string) {
  return value.split(',').map((item) => item.trim()).filter(Boolean)
}

function resolveTemplate(template: string, variables: string, overrides: Record<string, string>) {
  return parseVariables(variables).reduce(
    (result, variable) => result.split(`\${${variable.name}}`).join(overrides[variable.name] ?? variable.default ?? ''),
    template,
  )
}

function TemplateDrawer({ template, open, onClose }: { template: AppTemplate | null; open: boolean; onClose: () => void }) {
  const [searchParams] = useSearchParams()
  const [values, setValues] = useState<Record<string, string>>({})
  const [clusterID, setClusterID] = useState<number>()
  const [projectID, setProjectID] = useState<number>()
  const [namespace, setNamespace] = useState<string>()
  const [deployOpen, setDeployOpen] = useState(false)
  const clustersQuery = useQuery({ queryKey: ['clusters-for-template'], queryFn: () => listClusters(), enabled: open })
  const projectsQuery = useQuery({ queryKey: ['projects-for-template'], queryFn: () => listProjects(), enabled: open && !!clusterID })

  useEffect(() => {
    if (!open || !template) return
    const defaults: Record<string, string> = {}
    parseVariables(template.variables).forEach((item) => { defaults[item.name] = item.default || '' })
    setValues(defaults)
    const id = Number(searchParams.get('cluster_id'))
    setClusterID(Number.isFinite(id) && id > 0 ? id : undefined)
    setProjectID(undefined)
    setNamespace(undefined)
  }, [open, searchParams, template])

  const projects = (projectsQuery.data?.list || []).filter((item) => item.cluster_id === clusterID)
  const currentProject = projects.find((item) => item.id === projectID)
  const namespaces = currentProject ? splitNamespaces(currentProject.namespaces) : []
  const variables = template ? parseVariables(template.variables) : []
  const resolvedYaml = template ? resolveTemplate(template.template, template.variables, values) : ''

  useEffect(() => {
    if (namespace && variables.some((item) => item.name === 'NAMESPACE')) {
      setValues((current) => ({ ...current, NAMESPACE: namespace }))
    }
  }, [namespace, variables])

  if (!template) return null
  if (template.deploy_type === 'helm') {
    return <Drawer title={template.display_name || template.name} width="min(760px, 94vw)" open={open} onClose={onClose}>
      <Paragraph type="secondary">{template.description || '暂无说明'}</Paragraph>
      <Row gutter={[16, 16]} style={{ marginBottom: 20 }}>
        <Col span={12}><Text type="secondary">Chart</Text><div><Text strong>{template.template || '-'}</Text></div></Col>
        <Col span={12}><Text type="secondary">Chart 版本</Text><div><Text strong>{template.helm_chart_version || '仓库最新版本'}</Text></div></Col>
        <Col span={12}><Text type="secondary">默认仓库</Text><div><Text strong>{template.helm_repo_name || '安装时选择'}</Text></div></Col>
        <Col span={12}><Text type="secondary">仓库地址</Text><div><Text strong>{template.helm_repo_url || '-'}</Text></div></Col>
      </Row>
      <Text strong>默认 values.yaml</Text>
      <div style={{ marginTop: 8 }}><YamlEditor readOnly value={template.helm_values_yaml || '# 未配置默认 values.yaml'} height={400} /></div>
    </Drawer>
  }
  return <>
    <Drawer
      title={template.display_name || template.name}
      width="min(880px, 94vw)"
      open={open}
      onClose={onClose}
      extra={<Space>
        <Select allowClear showSearch optionFilterProp="label" style={{ width: 160 }} placeholder="选择集群" value={clusterID}
          options={(clustersQuery.data?.items || []).map((item) => ({ label: item.name, value: item.id }))}
          onChange={(value) => { setClusterID(value); setProjectID(undefined); setNamespace(undefined) }} />
        <Select allowClear showSearch optionFilterProp="label" style={{ width: 150 }} placeholder="选择项目" disabled={!clusterID} value={projectID}
          options={projects.map((item) => ({ label: item.name, value: item.id }))}
          onChange={(value) => { setProjectID(value); setNamespace(undefined) }} />
        <Button type="primary" icon={<RocketOutlined />} disabled={!clusterID} onClick={() => setDeployOpen(true)}>部署到集群</Button>
      </Space>}
    >
      <Paragraph type="secondary">{template.description || '暂无说明'}</Paragraph>
      {projectID && namespaces.length > 0 && <Form.Item label="部署命名空间"><Select allowClear value={namespace} placeholder="选择命名空间" options={namespaces.map((item) => ({ label: item, value: item }))} onChange={setNamespace} /></Form.Item>}
      {variables.length > 0 && <div style={{ marginBottom: 16 }}>
        <Text strong>模板变量</Text>
        <Row gutter={[12, 12]} style={{ marginTop: 8 }}>{variables.map((item) => <Col span={12} key={item.name}>
          <Text type="secondary" style={{ fontSize: 12 }}>{item.name}</Text>
          <Input value={values[item.name] ?? ''} placeholder={item.default} onChange={(event) => setValues((current) => ({ ...current, [item.name]: event.target.value }))} />
        </Col>)}</Row>
      </div>}
      <Text strong>YAML 预览</Text>
      <div style={{ marginTop: 8 }}><YamlEditor readOnly value={resolvedYaml} height={440} /></div>
    </Drawer>
    {clusterID && <ManifestApplyDrawer open={deployOpen} onClose={() => setDeployOpen(false)} clusterId={clusterID} initialYaml={resolvedYaml} title={`部署 ${template.display_name || template.name}`} />}
  </>
}

function TemplateCatalogTable({ deployType, onPreview }: { deployType: 'yaml' | 'helm'; onPreview: (template: AppTemplate) => void }) {
  const queryClient = useQueryClient()
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState<string>()
  const [editing, setEditing] = useState<AppTemplate | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm<AppTemplate>()
  const templateValue = Form.useWatch('template', form) ?? ''
  const helmValues = Form.useWatch('helm_values_yaml', form) ?? ''
  const templatesQuery = useQuery({ queryKey: ['app-templates'], queryFn: () => listAppTemplates() })
  const createMutation = useMutation({
    mutationFn: (payload: Partial<AppTemplate>) => createAppTemplate(payload),
    onSuccess: () => { message.success('已创建'); closeModal(); queryClient.invalidateQueries({ queryKey: ['app-templates'] }) },
    onError: (error: any) => message.error(error?.message || '创建失败'),
  })
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<AppTemplate> }) => updateAppTemplate(id, data),
    onSuccess: () => { message.success('已保存'); closeModal(); queryClient.invalidateQueries({ queryKey: ['app-templates'] }) },
    onError: (error: any) => message.error(error?.message || '保存失败'),
  })
  const deleteMutation = useMutation({
    mutationFn: deleteAppTemplate,
    onSuccess: () => { message.success('已删除'); queryClient.invalidateQueries({ queryKey: ['app-templates'] }) },
    onError: (error: any) => message.error(error?.message || '删除失败'),
  })
  const closeModal = () => { setModalOpen(false); setEditing(null); form.resetFields() }
  const openCreate = () => { setEditing(null); form.resetFields(); form.setFieldsValue({ variables: '[]', category: '', deploy_type: deployType }); setModalOpen(true) }
  const openEdit = (record: AppTemplate) => { setEditing(record); form.setFieldsValue({ ...record, deploy_type: deployType }); setModalOpen(true) }
  const submit = async () => {
    try {
      const values = await form.validateFields()
      const payload = { ...values, deploy_type: deployType }
      if (editing) updateMutation.mutate({ id: editing.id, data: payload })
      else createMutation.mutate(payload)
    } catch {
      // 表单字段会显示校验结果。
    }
  }
  const data = useMemo(() => (templatesQuery.data?.list || []).filter((item) => {
    if ((item.deploy_type || 'yaml') !== deployType) return false
    if (category && item.category !== category) return false
    const term = keyword.trim().toLowerCase()
    return !term || `${item.name} ${item.display_name} ${item.description} ${item.template}`.toLowerCase().includes(term)
  }), [category, deployType, keyword, templatesQuery.data?.list])
  const columns: ProColumns<AppTemplate>[] = [
    { title: '应用', width: 220, render: (_, record) => <Space><span style={{ fontSize: 20 }}>{record.icon || '📦'}</span><Space direction="vertical" size={0}><Text strong>{record.display_name || record.name}</Text><Text type="secondary" style={{ fontSize: 12 }}>{record.name}</Text></Space></Space> },
    { title: '分类', dataIndex: 'category', width: 110, render: (_, record) => {
      const value = record.category
      return value ? <Tag color={CATEGORY_COLORS[value]}>{CATEGORY_LABELS[value] || value}</Tag> : '-'
    } },
    ...(deployType === 'helm' ? [
      { title: 'Chart', dataIndex: 'template', width: 220, ellipsis: true },
      { title: '默认来源', width: 250, ellipsis: true, render: (_: unknown, record: AppTemplate) => record.helm_repo_url ? <Space direction="vertical" size={0}><Text>{record.helm_repo_name || '-'}</Text><Text type="secondary" style={{ fontSize: 12 }}>{record.helm_repo_url}</Text></Space> : <Text type="secondary">安装时选择</Text> },
      { title: 'values.yaml', width: 110, render: (_: unknown, record: AppTemplate) => record.helm_values_yaml?.trim() ? <Tag color="green">已配置</Tag> : <Tag>未配置</Tag> },
    ] as ProColumns<AppTemplate>[] : [
      { title: '说明', dataIndex: 'description', ellipsis: true },
      { title: '变量', width: 90, render: (_: unknown, record: AppTemplate) => parseVariables(record.variables).length || '-' },
    ] as ProColumns<AppTemplate>[]),
    { title: '更新于', dataIndex: 'updated_at', width: 170, render: (_, record) => record.updated_at ? formatDate(record.updated_at, 'YYYY-MM-DD HH:mm') : '-' },
    { title: '操作', width: 132, fixed: 'right', align: 'center', render: (_: unknown, record: AppTemplate) => <Space size={2}>
      <Tooltip title="查看"><Button type="text" shape="circle" icon={<EyeOutlined />} onClick={() => onPreview(record)} /></Tooltip>
      <Tooltip title="编辑"><Button type="text" shape="circle" icon={<EditOutlined />} onClick={() => openEdit(record)} /></Tooltip>
      {record.is_builtin ? <Tooltip title="内置应用不可删除"><span><Button type="text" shape="circle" disabled icon={<DeleteOutlined />} /></span></Tooltip> :
        <Popconfirm title="确认删除此应用目录？" description="不会影响任何已安装的集群应用。" onConfirm={() => deleteMutation.mutate(record.id)}><Tooltip title="删除"><Button type="text" danger shape="circle" icon={<DeleteOutlined />} /></Tooltip></Popconfirm>}
    </Space> },
  ]
  const title = deployType === 'helm' ? 'Helm Chart 应用目录' : 'YAML 模板'
  const subtitle = deployType === 'helm'
    ? '目录维护 Chart、可选默认仓库和 values.yaml；目标集群的仓库、Release 状态以其 Master 实际数据为准，并在“Helm 发布”中安装。'
    : '模板内容与变量在这里维护；选择模板后可在详情中预览并部署到目标集群。'
  return <>
    <ProTable<AppTemplate>
      headerTitle={<Space direction="vertical" size={0}><Text strong>{title}</Text><Text type="secondary" style={{ fontSize: 12 }}>{subtitle}</Text></Space>}
      rowKey="id" loading={templatesQuery.isLoading} dataSource={data} columns={columns} search={false} options={false}
      scroll={{ x: deployType === 'helm' ? 1240 : 1000 }} pagination={{ defaultPageSize: 12, showSizeChanger: true, showTotal: (total) => `共 ${total} 项` }}
      toolBarRender={() => [
        <Select key="category" allowClear placeholder="全部分类" value={category} onChange={setCategory} style={{ width: 130 }} options={CATEGORIES} />,
        <Input.Search key="search" allowClear placeholder={`搜索${deployType === 'helm' ? '应用或 Chart' : '模板'}`} value={keyword} onChange={(event) => setKeyword(event.target.value)} style={{ width: 240 }} />,
        <Button key="refresh" icon={<ReloadOutlined />} loading={templatesQuery.isFetching} onClick={() => templatesQuery.refetch()}>刷新</Button>,
        <Button key="create" type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建{deployType === 'helm' ? ' Helm Chart' : ' YAML 模板'}</Button>,
      ]}
    />
    <Modal title={editing ? `编辑${deployType === 'helm' ? ' Helm Chart' : ' YAML 模板'}` : `新建${deployType === 'helm' ? ' Helm Chart' : ' YAML 模板'}`} open={modalOpen} onCancel={closeModal} onOk={submit} confirmLoading={createMutation.isPending || updateMutation.isPending} okText="保存" width={860} destroyOnClose>
      <Form form={form} layout="vertical">
        <Row gutter={16}>
          <Col span={12}><Form.Item name="name" label="标识" rules={[{ required: true, message: '请输入唯一标识' }]}><Input placeholder={deployType === 'helm' ? '例如 redis' : '例如 nginx-deploy'} disabled={editing?.is_builtin} /></Form.Item></Col>
          <Col span={12}><Form.Item name="display_name" label="显示名称" rules={[{ required: true, message: '请输入显示名称' }]}><Input placeholder="例如 Redis" /></Form.Item></Col>
        </Row>
        <Row gutter={16}>
          <Col span={12}><Form.Item name="category" label="分类"><Select allowClear options={CATEGORIES} placeholder="选择分类" /></Form.Item></Col>
          <Col span={12}><Form.Item name="icon" label="图标"><Input placeholder="emoji 或图标 URL" /></Form.Item></Col>
        </Row>
        <Form.Item name="description" label="说明"><Input.TextArea rows={2} placeholder="描述此应用的用途和注意事项" /></Form.Item>
        {deployType === 'yaml' ? <>
          <Form.Item name="template" label="YAML 模板" rules={[{ required: true, message: '请输入 YAML 模板内容' }]}><YamlEditor value={templateValue} onChange={(value) => form.setFieldValue('template', value)} height={300} /></Form.Item>
          <Form.Item name="variables" label="变量定义（JSON）"><Input.TextArea rows={3} placeholder='[{"name":"NAMESPACE","default":"default"}]' /></Form.Item>
        </> : <>
          <Row gutter={16}>
            <Col span={16}><Form.Item name="template" label="Chart" rules={[{ required: true, message: '请输入 Chart，例如 bitnami/redis' }]}><Input placeholder="例如 bitnami/redis" /></Form.Item></Col>
            <Col span={8}><Form.Item name="helm_chart_version" label="Chart 版本"><Input placeholder="留空使用仓库最新版本" /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col span={10}><Form.Item name="helm_repo_name" label="默认仓库名称（可选）"><Input placeholder="例如 bitnami" /></Form.Item></Col>
            <Col span={14}><Form.Item name="helm_repo_url" label="默认仓库地址（可选）"><Input placeholder="https://charts.example.com/repo" /></Form.Item></Col>
          </Row>
          <Form.Item name="helm_values_yaml" label="默认 values.yaml"><YamlEditor value={helmValues} onChange={(value) => form.setFieldValue('helm_values_yaml', value)} height={280} /></Form.Item>
        </>}
      </Form>
    </Modal>
  </>
}

const AppStorePage: React.FC = () => {
  const location = useLocation()
  const deployType: 'yaml' | 'helm' = location.pathname.endsWith('/helm') ? 'helm' : 'yaml'
  const [preview, setPreview] = useState<AppTemplate | null>(null)
  return <AppPage>
    <TemplateCatalogTable deployType={deployType} onPreview={setPreview} />
    <TemplateDrawer template={preview} open={!!preview} onClose={() => setPreview(null)} />
  </AppPage>
}

export default AppStorePage
