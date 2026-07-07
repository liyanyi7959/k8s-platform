/**
 * 应用商店页
 * 展示 YAML 应用模板，支持分类筛选、搜索、变量替换与一键部署到集群
 */
import { useEffect, useMemo, useState } from 'react'
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
  SettingOutlined,
} from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage, EmptyState, ManifestApplyDrawer, YamlEditor } from '@/components'
import {
  createAppTemplate,
  deleteAppTemplate,
  listAppTemplates,
  updateAppTemplate,
  type AppTemplate,
} from '@/services/app-template'
import { listClusters } from '@/services/clusters'
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

/** 解析变量 JSON */
function parseVariables(variables: string): Array<{ name: string; default: string }> {
  try {
    return JSON.parse(variables || '[]')
  } catch {
    return []
  }
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
  const [varValues, setVarValues] = useState<Record<string, string>>({})
  const [selectedCluster, setSelectedCluster] = useState<number>()
  const [deployOpen, setDeployOpen] = useState(false)

  const { data: clustersData } = useQuery({
    queryKey: ['clusters-for-deploy'],
    queryFn: () => listClusters(),
    enabled: open,
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
              style={{ width: 200 }}
              options={clusterOptions}
              value={selectedCluster}
              onChange={setSelectedCluster}
              showSearch
              optionFilterProp="label"
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

// ───────────────────────── 应用商店主页面 ─────────────────────────

const AppStorePage: React.FC = () => {
  const [adminMode, setAdminMode] = useState(false)
  const [category, setCategory] = useState('')
  const [keyword, setKeyword] = useState('')
  const [selectedTemplate, setSelectedTemplate] = useState<AppTemplate | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['app-templates', category],
    queryFn: () => listAppTemplates(category ? { category } : {}),
  })

  // 按关键词过滤
  const filteredList = useMemo(() => {
    const list = data?.list || []
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
