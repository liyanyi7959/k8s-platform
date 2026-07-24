/**
 * 应用商店页
 * 展示 YAML 应用模板，支持分类筛选、搜索、变量替换与一键部署到集群
 */
import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from '@umijs/max'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import {
	Button,
  Card,
  Col,
  Drawer,
  Form,
  Input,
  Modal,
  Popconfirm,
  Row,
  Segmented,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd'
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  RocketOutlined,
  SearchOutlined,
  SettingOutlined,
} from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage, EmptyState, ManifestApplyDrawer, YamlEditor } from '@/components'
import AppAlert from '@/components/AppAlert'
import {
  createAppTemplate,
  deleteAppTemplate,
  listAppTemplates,
  updateAppTemplate,
  type AppTemplate,
} from '@/services/app-template'
import { listClusters } from '@/services/clusters'
import { listProjects, type Project } from '@/services/project'
import { helmPreflight, helmSearch, helmInstall, type HelmPreflightResult } from '@/services/k8s'
import { formatDate } from '@/utils'

const { Text, Paragraph } = Typography

/** 分类配置 */
const CATEGORIES = [
  { value: '', label: '全部' },
  { value: 'database', label: '数据库' },
  { value: 'middleware', label: '中间件' },
  { value: 'monitoring', label: '监控' },
  { value: 'devtool', label: '开发工具' },
  { value: 'networking', label: '网络' },
  { value: 'security', label: '安全' },
]

const CATEGORY_LABELS: Record<string, string> = {
  database: '数据库',
  middleware: '中间件',
  monitoring: '监控',
  devtool: '开发工具',
  networking: '网络',
  security: '安全',
}

const CATEGORY_COLORS: Record<string, string> = {
  database: 'blue',
  middleware: 'purple',
  monitoring: 'green',
  devtool: 'orange',
  networking: 'cyan',
  security: 'red',
}

const HELM_REPOSITORIES: Array<{ prefix: string; name: string; url: string }> = [
  { prefix: 'bitnami/', name: 'bitnami', url: 'https://charts.bitnami.com/bitnami' },
  {
    prefix: 'prometheus-community/',
    name: 'prometheus-community',
    url: 'https://prometheus-community.github.io/helm-charts',
  },
]

function getHelmRepository(chart?: string) {
  return HELM_REPOSITORIES.find((repository) => chart?.startsWith(repository.prefix))
}

/** 解析变量 JSON */
function parseVariables(variables: string): Array<{ name: string; default: string }> {
  try {
    return JSON.parse(variables || '[]')
  } catch {
    return []
  }
}

/** 将逗号分隔的命名空间字符串拆分为数组 */
function splitNamespaces(ns: string): string[] {
  return ns
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

/** 用变量值替换模板中的 ${VAR} 占位符 */
function resolveTemplate(
  template: string,
  variables: string,
  overrides?: Record<string, string>,
): string {
  const vars = parseVariables(variables)
  let result = template
  vars.forEach((v) => {
    const val = overrides?.[v.name] ?? v.default ?? ''
    result = result.split('${' + v.name + '}').join(val)
  })
  return result
}

// ───────────────────────── 模板详情抽屉 ─────────────────────────

function TemplateDrawer({
  template,
  open,
  onClose,
}: {
  template: AppTemplate | null
  open: boolean
  onClose: () => void
}) {
  const [searchParams] = useSearchParams()
  const [varValues, setVarValues] = useState<Record<string, string>>({})
  const [selectedCluster, setSelectedCluster] = useState<number>()
  const [selectedProject, setSelectedProject] = useState<number>()
  const [selectedNamespace, setSelectedNamespace] = useState<string>()
  const [deployOpen, setDeployOpen] = useState(false)

  const { data: clustersData } = useQuery({
    queryKey: ['clusters-for-deploy'],
    queryFn: () => listClusters(),
    enabled: open,
  })

  // 获取项目列表（基于选中集群过滤）
  const { data: projectsData } = useQuery({
    queryKey: ['projects-for-deploy', selectedCluster],
    queryFn: () => listProjects(),
    enabled: open && !!selectedCluster,
  })

  const variables = useMemo(
    () => (template ? parseVariables(template.variables) : []),
    [template],
  )

  const resolvedYaml = useMemo(() => {
    if (!template) return ''
    return resolveTemplate(template.template, template.variables, varValues)
  }, [template, varValues])

  // 模板切换时重置变量为默认值
  useEffect(() => {
    if (template) {
      const defaults: Record<string, string> = {}
      parseVariables(template.variables).forEach((v) => {
        defaults[v.name] = v.default ?? ''
      })
      setVarValues(defaults)
    }
  }, [template])

  // URL 参数预选集群
  useEffect(() => {
    if (open) {
      const clusterIdParam = searchParams.get('cluster_id')
      if (clusterIdParam && !selectedCluster) {
        setSelectedCluster(Number(clusterIdParam))
      }
    }
  }, [open, searchParams, selectedCluster])

  // 当前集群下的项目列表
  const projects = (projectsData?.list || []).filter(
    (p) => p.cluster_id === selectedCluster,
  )
  const projectOptions = projects.map((p) => ({ label: p.name, value: p.id }))

  // 选中项目后获取命名空间列表
  const selectedProjectData = projects.find((p) => p.id === selectedProject)
  const nsList = selectedProjectData
    ? splitNamespaces(selectedProjectData.namespaces)
    : []
  const nsOptions = nsList.map((ns) => ({ label: ns, value: ns }))

  // URL 参数预选命名空间
  useEffect(() => {
    if (open && nsList.length > 0 && !selectedNamespace) {
      const nsParam = searchParams.get('namespaces')
      if (nsParam) {
        const firstNs = splitNamespaces(nsParam)[0]
        if (firstNs && nsList.includes(firstNs)) {
          setSelectedNamespace(firstNs)
        }
      }
    }
  }, [open, searchParams, nsList, selectedNamespace])

  // 选择命名空间后，自动填充模板中的 NAMESPACE 变量
  useEffect(() => {
    if (selectedNamespace && variables.some((v) => v.name === 'NAMESPACE')) {
      setVarValues((prev) => ({ ...prev, NAMESPACE: selectedNamespace }))
    }
  }, [selectedNamespace, variables])

  const clusterOptions = (clustersData?.items || []).map((c) => ({
    label: c.name,
    value: c.id,
  }))

  const handleDeploy = () => {
    if (!selectedCluster) {
      message.warning('请先选择集群')
      return
    }
    setDeployOpen(true)
  }

  if (!template) return null

  return (
    <>
      <Drawer
        title={
          <Space>
            <span style={{ fontSize: 20 }}>{template.icon || '📦'}</span>
            <span>{template.display_name || template.name}</span>
          </Space>
        }
        open={open}
        onClose={onClose}
        width={720}
        extra={
          <Space>
            <Select
              placeholder="选择集群"
              style={{ width: 160 }}
              options={clusterOptions}
              value={selectedCluster}
              onChange={(v) => {
                setSelectedCluster(v)
                setSelectedProject(undefined)
                setSelectedNamespace(undefined)
              }}
              showSearch
              optionFilterProp="label"
            />
            <Select
              placeholder="选择项目"
              style={{ width: 140 }}
              options={projectOptions}
              value={selectedProject}
              onChange={(v) => {
                setSelectedProject(v)
                setSelectedNamespace(undefined)
              }}
              showSearch
              optionFilterProp="label"
              disabled={!selectedCluster}
            />
            <Button
              type="primary"
              icon={<RocketOutlined />}
              onClick={handleDeploy}
              disabled={!selectedCluster}
            >
              部署到集群
            </Button>
          </Space>
        }
      >
        <Paragraph type="secondary">{template.description || '暂无描述'}</Paragraph>

        {/* 命名空间选择（选择项目后出现） */}
        {selectedProject && nsList.length > 0 && (
          <div style={{ marginBottom: 16 }}>
            <Text strong>部署命名空间</Text>
            <div style={{ marginTop: 8 }}>
              <Select
                placeholder="选择命名空间"
                style={{ width: '100%' }}
                options={nsOptions}
                value={selectedNamespace}
                onChange={setSelectedNamespace}
                showSearch
                optionFilterProp="label"
              />
            </div>
          </div>
        )}

        {variables.length > 0 && (
          <div style={{ marginBottom: 16 }}>
            <Text strong>模板变量</Text>
            <Row gutter={[12, 12]} style={{ marginTop: 8 }}>
              {variables.map((v) => (
                <Col key={v.name} span={12}>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {v.name}
                  </Text>
                  <Input
                    value={varValues[v.name] ?? ''}
                    onChange={(e) =>
                      setVarValues((prev) => ({ ...prev, [v.name]: e.target.value }))
                    }
                    placeholder={v.default}
                  />
                </Col>
              ))}
            </Row>
          </div>
        )}

        <Text strong>YAML 预览</Text>
        <div style={{ marginTop: 8 }}>
          <YamlEditor value={resolvedYaml} height={400} readOnly />
        </div>
      </Drawer>

      {selectedCluster && (
        <ManifestApplyDrawer
          open={deployOpen}
          onClose={() => setDeployOpen(false)}
          clusterId={selectedCluster}
          initialYaml={resolvedYaml}
          title={`部署 ${template.display_name || template.name}`}
        />
      )}
    </>
  )
}

// ───────────────────────── 管理模式表格 ─────────────────────────

function AdminTable({ onBack }: { onBack: () => void }) {
  const queryClient = useQueryClient()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<AppTemplate | null>(null)
  const [form] = Form.useForm()

  const { data, isLoading } = useQuery({
    queryKey: ['app-templates'],
    queryFn: () => listAppTemplates(),
  })

  const createMutation = useMutation({
    mutationFn: (payload: Partial<AppTemplate>) => createAppTemplate(payload),
    onSuccess: () => {
      message.success('创建成功')
      setModalOpen(false)
      form.resetFields()
      queryClient.invalidateQueries({ queryKey: ['app-templates'] })
    },
    onError: (err: any) => message.error(err?.message || '创建失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<AppTemplate> }) =>
      updateAppTemplate(id, data),
    onSuccess: () => {
      message.success('更新成功')
      setModalOpen(false)
      setEditing(null)
      form.resetFields()
      queryClient.invalidateQueries({ queryKey: ['app-templates'] })
    },
    onError: (err: any) => message.error(err?.message || '更新失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteAppTemplate(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['app-templates'] })
    },
    onError: (err: any) => message.error(err?.message || '删除失败'),
  })

  const handleAdd = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (record: AppTemplate) => {
    setEditing(record)
    form.setFieldsValue(record)
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editing) {
        updateMutation.mutate({ id: editing.id, data: values })
      } else {
        createMutation.mutate(values)
      }
    } catch {
      // 校验失败
    }
  }

  const columns: ProColumns<AppTemplate>[] = [
    {
      title: '图标',
      dataIndex: 'icon',
      width: 60,
      render: (v: string) => <span style={{ fontSize: 20 }}>{v || '📦'}</span>,
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 150,
      render: (_, r) => (
        <Space direction="vertical" size={0}>
          <Text strong>{r.name}</Text>
          {r.display_name && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              {r.display_name}
            </Text>
          )}
        </Space>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      width: 100,
      render: (v: string) =>
        v ? <Tag color={CATEGORY_COLORS[v]}>{CATEGORY_LABELS[v] || v}</Tag> : '-',
    },
    {
      title: '描述',
      dataIndex: 'description',
      ellipsis: true,
      render: (v: string) => v || <Text type="secondary">-</Text>,
    },
    {
      title: '内置',
      dataIndex: 'is_builtin',
      width: 70,
      render: (v: boolean) => (v ? <Tag color="gold">内置</Tag> : '-'),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => (v ? formatDate(v, 'YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: '操作',
      width: 120,
      align: 'center',
      fixed: 'right',
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          />
          {!record.is_builtin && (
            <Popconfirm
              title="确认删除该模板？"
              onConfirm={() => deleteMutation.mutate(record.id)}
            >
              <Button type="link" size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const categoryOptions = CATEGORIES.filter((c) => c.value).map((c) => ({
    label: c.label,
    value: c.value,
  }))

  return (
    <>
      <ProTable<AppTemplate>
        headerTitle="模板管理"
        columns={columns}
        dataSource={data?.list || []}
        loading={isLoading}
        rowKey="id"
        search={false}
        options={false}
        scroll={{ x: 1000 }}
        pagination={{ pageSize: 10, showSizeChanger: true }}
        toolBarRender={() => [
          <Button key="back" onClick={onBack}>
            返回商店
          </Button>,
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() => queryClient.invalidateQueries({ queryKey: ['app-templates'] })}
          >
            刷新
          </Button>,
          <Button key="add" type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            新建模板
          </Button>,
        ]}
      />

      <Modal
        title={editing ? '编辑模板' : '新建模板'}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false)
          setEditing(null)
          form.resetFields()
        }}
        onOk={handleSubmit}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        destroyOnClose
        width={640}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="模板名称"
            rules={[{ required: true, message: '请输入模板名称' }]}
          >
            <Input placeholder="例如：nginx-deploy" disabled={editing?.is_builtin} />
          </Form.Item>
          <Form.Item name="display_name" label="展示名称">
            <Input placeholder="例如：Nginx" />
          </Form.Item>
          <Form.Item name="category" label="分类">
            <Select options={categoryOptions} placeholder="选择分类" />
          </Form.Item>
          <Form.Item name="icon" label="图标">
            <Input placeholder="emoji 或 URL，如 🌐" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="模板用途说明" />
          </Form.Item>
          <Form.Item name="template" label="YAML 模板">
            <Input.TextArea rows={8} placeholder="YAML 模板内容，支持 ${VAR} 变量" />
          </Form.Item>
          <Form.Item name="variables" label="变量定义（JSON）">
            <Input.TextArea
              rows={3}
              placeholder='[{"name":"NAMESPACE","default":"default"}]'
            />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

// ───────────────────────── Helm Chart 面板 ─────────────────────────

function HelmChartPanel() {
  const [searchParams] = useSearchParams()
  const [selectedCluster, setSelectedCluster] = useState<number>()
  const [selectedProject, setSelectedProject] = useState<number>()
  const [keyword, setKeyword] = useState('')
  const [searchResults, setSearchResults] = useState<any[]>([])
  const [searching, setSearching] = useState(false)
  const [installOpen, setInstallOpen] = useState(false)
  const [selectedChart, setSelectedChart] = useState<any>(null)
	const [valuesYaml, setValuesYaml] = useState('')
	const [preparing, setPreparing] = useState(false)
	const [preflightResult, setPreflightResult] = useState<HelmPreflightResult | null>(null)
  const [installForm] = Form.useForm()

  const { data: clustersData } = useQuery({
    queryKey: ['clusters-for-helm-search'],
    queryFn: () => listClusters(),
  })

  // 获取项目列表（基于选中集群过滤）
  const { data: projectsData } = useQuery({
    queryKey: ['projects-for-helm', selectedCluster],
    queryFn: () => listProjects(),
    enabled: !!selectedCluster,
  })

  // URL 参数预选集群
  useEffect(() => {
    const clusterIdParam = searchParams.get('cluster_id')
    if (clusterIdParam && !selectedCluster) {
      setSelectedCluster(Number(clusterIdParam))
    }
  }, [searchParams, selectedCluster])

  // 获取预置 Helm 模板
  const { data: helmTemplatesData } = useQuery({
    queryKey: ['app-templates', 'helm'],
    queryFn: () => listAppTemplates({ category: '' }),
  })

  const helmTemplates = (helmTemplatesData?.list || []).filter((t) => t.deploy_type === 'helm')

  const clusterOptions = (clustersData?.items || []).filter((c) => c.status === 'active').map((c) => ({
    label: c.name,
    value: c.id,
  }))

  // 当前集群下的项目列表
  const projects = (projectsData?.list || []).filter(
    (p) => p.cluster_id === selectedCluster,
  )
  const projectOptions = projects.map((p) => ({ label: p.name, value: p.id }))

  // 选中项目后获取命名空间列表
  const selectedProjectData = projects.find((p) => p.id === selectedProject)
  const nsList = selectedProjectData
    ? splitNamespaces(selectedProjectData.namespaces)
    : []
  const nsOptions = nsList.map((ns) => ({ label: ns, value: ns }))

  // 搜索 Helm Chart
  const handleSearch = async () => {
    if (!selectedCluster) {
      message.warning('请先选择集群')
      return
    }
    if (!keyword.trim()) {
      message.warning('请输入搜索关键词')
      return
    }
    setSearching(true)
    try {
      const results = await helmSearch(selectedCluster, keyword)
      setSearchResults(results)
    } catch (err: any) {
      message.error(err?.message || '搜索失败')
    } finally {
      setSearching(false)
    }
  }

  const installMutation = useMutation({
    mutationFn: (data: any) => helmInstall(selectedCluster!, data),
    onSuccess: (result: any) => {
      const masterMessage = result?.master?.message ? `；${result.master.message}` : ''
      message.success(`安装成功：已等待 Release 就绪${masterMessage}`)
      setInstallOpen(false)
      installForm.resetFields()
      setValuesYaml('')
    },
    onError: (err: any) => message.error(err?.message || '安装失败'),
  })

	const handleInstall = async () => {
		try {
			const values = await installForm.validateFields()
			if (!selectedCluster) {
				message.warning('请先选择可用集群')
				return
			}
			setPreparing(true)
			const result = await helmPreflight(selectedCluster)
			setPreflightResult(result)
			installMutation.mutate({
				...values,
				chart: selectedChart?.name || selectedChart?.template,
				version: selectedChart?.version,
				repo_url: values.repo_url || selectedChart?.repoUrl,
        repo_name: selectedChart?.repoName,
        values_yaml: valuesYaml,
      })
		} catch (err: any) {
			message.error(err?.message || '部署前置校验失败，已阻止安装')
		} finally {
			setPreparing(false)
		}
  }

  // 点击预置模板卡片 → 直接打开安装 Modal
  const handlePresetClick = (template: any) => {
    const repository = getHelmRepository(template.template)
    setSelectedChart({
      name: template.template,
      description: template.description,
      isPreset: true,
      repoUrl: repository?.url,
      repoName: repository?.name,
    })
    installForm.setFieldsValue({ repo_url: repository?.url })
    setInstallOpen(true)
  }

  // 点击搜索结果 → 打开安装 Modal
	const handleSearchResultClick = (chart: any) => {
		setSelectedChart({
			...chart,
			repoName: chart.repo_name,
			repoUrl: chart.repo_url,
		})
    setInstallOpen(true)
  }

  const searchColumns = [
    { title: 'Chart 名称', dataIndex: 'name', width: 200, ellipsis: true },
    { title: '版本', dataIndex: 'version', width: 100 },
    { title: 'App Version', dataIndex: 'app_version', width: 120 },
    { title: '描述', dataIndex: 'description', ellipsis: true },
    {
      title: '操作', width: 100, align: 'center' as const,
      render: (_: any, record: any) => (
        <Button type="link" size="small" icon={<RocketOutlined />} onClick={() => handleSearchResultClick(record)}>
          安装
        </Button>
      ),
    },
  ]

  return (
    <>
      {/* 集群选择 + 项目选择 + 搜索 */}
      <Space style={{ marginBottom: 16 }}>
        <Select
          placeholder="选择集群"
          style={{ width: 200 }}
          options={clusterOptions}
          value={selectedCluster}
			onChange={(v) => {
				setSelectedCluster(v)
				setSelectedProject(undefined)
				setPreflightResult(null)
          }}
          showSearch
          optionFilterProp="label"
        />
        <Select
          placeholder="选择项目"
          style={{ width: 160 }}
          options={projectOptions}
          value={selectedProject}
          onChange={setSelectedProject}
          showSearch
          optionFilterProp="label"
          disabled={!selectedCluster}
        />
        <Input
          placeholder="搜索远程仓库 Chart，如 redis"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          style={{ width: 280 }}
          onPressEnter={handleSearch}
          allowClear
        />
        <Button type="primary" icon={<SearchOutlined />} loading={searching} onClick={handleSearch}>
          搜索
        </Button>
      </Space>

      {/* 预置 Helm 模板 */}
      {helmTemplates.length > 0 && (
        <div style={{ marginBottom: 24 }}>
          <div style={{ marginBottom: 12 }}>
            <Text strong>推荐应用</Text>
            <Text type="secondary" style={{ fontSize: 12, marginLeft: 8 }}>
              预置 Helm Chart，点击直接安装
            </Text>
          </div>
          <Row gutter={[16, 16]}>
            {helmTemplates.map((template) => (
              <Col key={template.id} xs={24} sm={12} md={8} lg={6}>
                <Card hoverable onClick={() => handlePresetClick(template)}>
                  <div style={{ textAlign: 'center', marginBottom: 12 }}>
                    <span style={{ fontSize: 40 }}>{template.icon || '📦'}</span>
                  </div>
                  <div style={{ textAlign: 'center', marginBottom: 8 }}>
                    <Text strong style={{ fontSize: 16 }}>
                      {template.display_name || template.name}
                    </Text>
                  </div>
                  <Paragraph
                    type="secondary"
                    ellipsis={{ rows: 2 }}
                    style={{ textAlign: 'center', minHeight: 40, marginBottom: 8 }}
                  >
                    {template.description || '暂无描述'}
                  </Paragraph>
                  <div style={{ textAlign: 'center' }}>
                    {template.category && (
                      <Tag color={CATEGORY_COLORS[template.category]}>
                        {CATEGORY_LABELS[template.category] || template.category}
                      </Tag>
                    )}
                    <Tag color="geekblue">Helm</Tag>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        </div>
      )}

      {/* 远程搜索结果 */}
      {searchResults.length > 0 && (
        <div>
          <div style={{ marginBottom: 12 }}>
            <Text strong>搜索结果</Text>
          </div>
          <Table
            dataSource={searchResults}
            columns={searchColumns}
            rowKey={(r) => `${r.name}/${r.version}`}
            loading={searching}
            pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
            scroll={{ x: 800 }}
          />
        </div>
      )}

      {/* 安装 Modal */}
      <Modal
        title="Helm 安装"
        open={installOpen}
			onCancel={() => {
          setInstallOpen(false)
          installForm.resetFields()
          setValuesYaml('')
			setSelectedChart(null)
			setPreflightResult(null)
        }}
        onOk={handleInstall}
			confirmLoading={preparing || installMutation.isPending}
        width={640}
        destroyOnClose
		>
		{preflightResult ? (
			<AppAlert
				showIcon
				type="success"
				style={{ marginBottom: 16 }}
				message={`部署环境已就绪 · Kubernetes ${preflightResult.cluster_version}`}
				description={`${preflightResult.master.master_name} (${preflightResult.master.master_ip}) · ${preflightResult.master.message} · ${preflightResult.master.helm_version}`}
			/>
		) : (
			<AppAlert
				showIcon
				type="info"
				style={{ marginBottom: 16 }}
				message="部署前会严格校验运行环境"
				description="系统会验证 Kubernetes API 与 Ready 节点，连接关联 Master；如未安装 Helm，将从官方 HTTPS 源下载、校验 SHA-256 后安装。任一步失败均不会创建 Release。"
			/>
		)}
		<Form form={installForm} layout="vertical" initialValues={{ namespace: 'default' }}>
          <Form.Item name="release_name" label="Release 名称" rules={[{ required: true, message: '请输入 Release 名称' }]}>
            <Input placeholder="例如：my-redis" />
          </Form.Item>
          <Form.Item name="namespace" label="命名空间" rules={[{ required: true, message: '请选择命名空间' }]}>
            {selectedProject ? (
              <Select
                placeholder="选择命名空间"
                options={nsOptions}
                showSearch
                optionFilterProp="label"
              />
            ) : (
              <Input placeholder="default" />
            )}
          </Form.Item>
          <Form.Item label="Chart">
            <Input value={selectedChart?.name} disabled />
          </Form.Item>
          {selectedChart?.isPreset && (
            <Form.Item name="repo_url" label="仓库地址（可选）">
              <Input placeholder="如 https://charts.bitnami.com/bitnami" />
            </Form.Item>
          )}
          <Form.Item label="values.yaml（可选）">
            <YamlEditor value={valuesYaml} onChange={setValuesYaml} height={300} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

// ───────────────────────── 应用商店主页面 ─────────────────────────

const AppStorePage: React.FC = () => {
  const [adminMode, setAdminMode] = useState(false)
  const [mode, setMode] = useState<'yaml' | 'helm'>('yaml')
  const [category, setCategory] = useState('')
  const [keyword, setKeyword] = useState('')
  const [selectedTemplate, setSelectedTemplate] = useState<AppTemplate | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['app-templates', category],
    queryFn: () => listAppTemplates(category ? { category } : {}),
  })

  // 按关键词过滤（仅展示 YAML 类型模板）
  const filteredList = useMemo(() => {
    const list = (data?.list || []).filter((t) => t.deploy_type !== 'helm')
    if (!keyword.trim()) return list
    const kw = keyword.toLowerCase()
    return list.filter(
      (t) =>
        t.name.toLowerCase().includes(kw) ||
        t.display_name.toLowerCase().includes(kw) ||
        t.description.toLowerCase().includes(kw),
    )
  }, [data?.list, keyword])

  const handleCardClick = (template: AppTemplate) => {
    setSelectedTemplate(template)
    setDrawerOpen(true)
  }

  if (adminMode) {
    return (
      <AppPage>
        <AdminTable onBack={() => setAdminMode(false)} />
      </AppPage>
    )
  }

  return (
    <AppPage>
      <Tabs
        activeKey={mode}
        onChange={(k) => setMode(k as 'yaml' | 'helm')}
        items={[
          {
            key: 'yaml',
            label: 'YAML 模板',
            children: (
              <>
                {/* 顶部工具栏：分类筛选 + 搜索 + 管理入口 */}
                <div style={{ marginBottom: 24 }}>
                  <Row gutter={[12, 12]} align="middle">
                    <Col flex="auto">
                      <Segmented
                        options={CATEGORIES}
                        value={category}
                        onChange={(v) => setCategory(v as string)}
                      />
                    </Col>
                    <Col>
                      <Space>
                        <Input.Search
                          placeholder="搜索模板..."
                          allowClear
                          style={{ width: 200 }}
                          onChange={(e) => setKeyword(e.target.value)}
                        />
                        <Button icon={<SettingOutlined />} onClick={() => setAdminMode(true)}>
                          管理
                        </Button>
                      </Space>
                    </Col>
                  </Row>
                </div>

                {/* 模板卡片网格 */}
                {isLoading ? (
                  <div style={{ textAlign: 'center', padding: 48 }}>
                    <Text type="secondary">加载中...</Text>
                  </div>
                ) : filteredList.length === 0 ? (
                  <EmptyState description="暂无应用模板" />
                ) : (
                  <Row gutter={[16, 16]}>
                    {filteredList.map((template) => (
                      <Col key={template.id} xs={24} sm={12} md={8} lg={6}>
                        <Card hoverable onClick={() => handleCardClick(template)}>
                          <div style={{ textAlign: 'center', marginBottom: 12 }}>
                            <span style={{ fontSize: 40 }}>{template.icon || '📦'}</span>
                          </div>
                          <div style={{ textAlign: 'center', marginBottom: 8 }}>
                            <Text strong style={{ fontSize: 16 }}>
                              {template.display_name || template.name}
                            </Text>
                          </div>
                          <Paragraph
                            type="secondary"
                            ellipsis={{ rows: 2 }}
                            style={{ textAlign: 'center', minHeight: 40, marginBottom: 8 }}
                          >
                            {template.description || '暂无描述'}
                          </Paragraph>
                          <div style={{ textAlign: 'center' }}>
                            {template.category && (
                              <Tag color={CATEGORY_COLORS[template.category]}>
                                {CATEGORY_LABELS[template.category] || template.category}
                              </Tag>
                            )}
                            {template.is_builtin && <Tag color="gold">内置</Tag>}
                          </div>
                        </Card>
                      </Col>
                    ))}
                  </Row>
                )}
              </>
            ),
          },
          {
            key: 'helm',
            label: 'Helm Chart',
            children: <HelmChartPanel />,
          },
        ]}
      />

      {/* 详情抽屉 */}
      <TemplateDrawer
        template={selectedTemplate}
        open={drawerOpen}
        onClose={() => {
          setDrawerOpen(false)
          setSelectedTemplate(null)
        }}
      />
    </AppPage>
  )
}

export default AppStorePage
