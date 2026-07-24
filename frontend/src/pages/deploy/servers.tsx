import { useMemo, useState, type Key } from 'react'
import { history } from '@umijs/max'
import {
  Alert,
  Badge,
  Button,
  Card,
  Descriptions,
  Drawer,
  Dropdown,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd'
import {
  ApiOutlined,
  CloudServerOutlined,
  CodeOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  EllipsisOutlined,
  EyeOutlined,
  HddOutlined,
  KeyOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  SafetyCertificateOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import { AppPage } from '@/components'
import {
  createServer,
  deleteServer,
  getCredentials,
  getServers,
  getServerSummary,
  testServerSSH,
  updateServer,
} from '@/services/deploy'
import type { CreateServerRequest, DeployServer } from '@/types/deploy'
import ServerTerminalDrawer from './ServerTerminalDrawer'

const { Paragraph, Text, Title } = Typography

type ServerFormValues = CreateServerRequest & { labelsText?: string }

const statusMeta: Record<string, { badge: 'success' | 'error' | 'processing' | 'default'; text: string; color: string }> = {
  available: { badge: 'success', text: '在线', color: 'green' },
  registered: { badge: 'processing', text: '待巡检', color: 'gold' },
  unavailable: { badge: 'error', text: '不可达', color: 'red' },
}

function StatusBadge({ status }: { status: string }) {
  const meta = statusMeta[status] || { badge: 'default' as const, text: status || '未知', color: 'default' }
  return <Badge status={meta.badge} text={meta.text} />
}

function parseLabels(value?: string): Record<string, string> | undefined {
  const text = value?.trim()
  if (!text) return undefined
  const labels: Record<string, string> = {}
  for (const segment of text.split(/[,，\n]/)) {
    const item = segment.trim()
    if (!item) continue
    const separator = item.indexOf('=')
    if (separator <= 0 || separator === item.length - 1) {
      throw new Error(`标签“${item}”格式错误，请使用 key=value`)
    }
    labels[item.slice(0, separator).trim()] = item.slice(separator + 1).trim()
  }
  return Object.keys(labels).length ? labels : undefined
}

function labelsToText(labels?: Record<string, any>) {
  return Object.entries(labels || {})
    .map(([key, value]) => `${key}=${String(value)}`)
    .join(', ')
}

function formatMemory(memoryMb?: number) {
  if (!memoryMb) return '-'
  return memoryMb >= 1024 ? `${(memoryMb / 1024).toFixed(memoryMb % 1024 === 0 ? 0 : 1)} GiB` : `${memoryMb} MiB`
}

function escapeCsv(value: unknown) {
  return `"${String(value ?? '').replaceAll('"', '""')}"`
}

export default function ServersPage() {
  const queryClient = useQueryClient()
  const [form] = Form.useForm<ServerFormValues>()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [searchDraft, setSearchDraft] = useState('')
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>()
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<DeployServer | null>(null)
  const [detailServer, setDetailServer] = useState<DeployServer | null>(null)
  const [terminalServer, setTerminalServer] = useState<DeployServer | null>(null)
  const [testingId, setTestingId] = useState<number | null>(null)
  const [batchTesting, setBatchTesting] = useState(false)
  const [batchDeleting, setBatchDeleting] = useState(false)

  const serverQuery = useQuery({
    queryKey: ['deploy-servers', page, pageSize, keyword, statusFilter],
    queryFn: () => getServers({ page, pageSize, keyword: keyword || undefined, status: statusFilter }),
  })
  const summaryQuery = useQuery({
    queryKey: ['deploy-server-summary'],
    queryFn: getServerSummary,
  })
  const credentialsQuery = useQuery({
    queryKey: ['deploy-credentials-for-select'],
    queryFn: () => getCredentials({ page: 1, pageSize: 200 }),
  })

  const refreshServers = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['deploy-servers'] }),
      queryClient.invalidateQueries({ queryKey: ['deploy-server-summary'] }),
    ])
  }

  const createMutation = useMutation({
    mutationFn: createServer,
    onSuccess: async () => {
      message.success('服务器已加入资源池')
      setModalOpen(false)
      form.resetFields()
      await refreshServers()
    },
    onError: (error: any) => message.error(error?.message || '添加服务器失败'),
  })
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<CreateServerRequest> }) => updateServer(id, data),
    onSuccess: async () => {
      message.success('服务器信息已更新')
      setModalOpen(false)
      setEditing(null)
      form.resetFields()
      await refreshServers()
    },
    onError: (error: any) => message.error(error?.message || '更新服务器失败'),
  })
  const deleteMutation = useMutation({
    mutationFn: deleteServer,
    onSuccess: async () => {
      message.success('服务器已移出资源池')
      setSelectedRowKeys([])
      await refreshServers()
    },
    onError: (error: any) => message.error(error?.message || '删除服务器失败'),
  })
  const testMutation = useMutation({
    mutationFn: async (id: number) => {
      setTestingId(id)
      return testServerSSH(id)
    },
    onSuccess: async (result) => {
      message.success(result.message || 'SSH 连接成功，主机信息已刷新')
      setTestingId(null)
      await refreshServers()
    },
    onError: async (error: any) => {
      message.error(error?.message || 'SSH 连通性检测失败')
      setTestingId(null)
      await refreshServers()
    },
  })

  const openCreateModal = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ sshPort: 22, user: 'root', authType: 'key' })
    setModalOpen(true)
  }

  const openEditModal = (server: DeployServer) => {
    setEditing(server)
    form.setFieldsValue({
      name: server.name,
      ip: server.ip,
      sshPort: server.sshPort,
      user: server.user,
      authType: server.authType,
      credentialId: server.credentialId,
      labelsText: labelsToText(server.labels),
      remark: server.remark,
    })
    setModalOpen(true)
  }

  const submitServer = async () => {
    try {
      const values = await form.validateFields()
      const payload: CreateServerRequest = {
        ...values,
        labels: parseLabels(values.labelsText),
      }
      delete (payload as ServerFormValues).labelsText
      if (editing) updateMutation.mutate({ id: editing.id, data: payload })
      else createMutation.mutate(payload)
    } catch (error: any) {
      if (error instanceof Error) message.error(error.message)
    }
  }

  const selectedServers = useMemo(() => {
    const selected = new Set(selectedRowKeys.map(Number))
    return (serverQuery.data?.items || []).filter((server) => selected.has(server.id))
  }, [selectedRowKeys, serverQuery.data?.items])

  const runBatchInspection = async () => {
    const targets = selectedServers.length ? selectedServers : serverQuery.data?.items || []
    if (!targets.length) return
    setBatchTesting(true)
    const results = await Promise.allSettled(targets.map((server) => testServerSSH(server.id)))
    const successCount = results.filter((result) => result.status === 'fulfilled').length
    const failedCount = results.length - successCount
    if (failedCount > 0) message.warning(`巡检完成：${successCount} 台可达，${failedCount} 台异常`)
    else message.success(`巡检完成：${successCount} 台服务器均可达`)
    setBatchTesting(false)
    await refreshServers()
  }

  const deleteSelectedServers = async () => {
    if (!selectedServers.length) return
    setBatchDeleting(true)
    const results = await Promise.allSettled(selectedServers.map((server) => deleteServer(server.id)))
    const successCount = results.filter((result) => result.status === 'fulfilled').length
    const failedCount = results.length - successCount
    if (failedCount) message.warning(`已移除 ${successCount} 台，${failedCount} 台移除失败`)
    else message.success(`已移除 ${successCount} 台服务器`)
    setBatchDeleting(false)
    setSelectedRowKeys([])
    await refreshServers()
  }

  const exportInventory = () => {
    const items = serverQuery.data?.items || []
    if (!items.length) {
      message.info('当前筛选条件下没有可导出的服务器')
      return
    }
    const rows = [
      ['名称', '主机地址', 'SSH端口', '用户', '认证方式', '操作系统', '内核', 'CPU核数', '内存MiB', '磁盘GiB', '状态', '标签', '备注'],
      ...items.map((server) => [
        server.name,
        server.ip,
        server.sshPort,
        server.user,
        server.authType,
        [server.os, server.osVersion].filter(Boolean).join(' '),
        server.kernel,
        server.cpuCores,
        server.memoryMb,
        server.diskGb,
        statusMeta[server.status]?.text || server.status,
        labelsToText(server.labels),
        server.remark,
      ]),
    ]
    const csv = `\uFEFF${rows.map((row) => row.map(escapeCsv).join(',')).join('\r\n')}`
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }))
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `server-inventory-${dayjs().format('YYYYMMDD-HHmm')}.csv`
    anchor.click()
    URL.revokeObjectURL(url)
  }

  const columns: ColumnsType<DeployServer> = [
    {
      title: '服务器',
      dataIndex: 'name',
      width: 210,
      fixed: 'left',
      render: (name: string, server) => (
        <div className="app-server-cell">
          <span className={`app-server-cell__icon app-server-cell__icon--${server.status}`}>
            <CloudServerOutlined />
          </span>
          <div>
            <Text strong>{name}</Text>
            <Text type="secondary" copyable={{ text: server.ip }}>{server.ip}</Text>
          </div>
        </div>
      ),
    },
    {
      title: '访问方式',
      width: 185,
      render: (_, server) => (
        <div className="app-server-stack">
          <Text>{server.user}@{server.ip}</Text>
          <Space size={6}>
            <Text type="secondary">SSH :{server.sshPort}</Text>
            <Tag bordered={false} color="blue">{server.authType === 'key' ? '密钥' : '密码'}</Tag>
          </Space>
        </div>
      ),
    },
    {
      title: '系统与内核',
      width: 230,
      render: (_, server) => (
        <div className="app-server-stack">
          <Text>{[server.os, server.osVersion].filter(Boolean).join(' ') || '等待首次巡检'}</Text>
          <Text type="secondary" ellipsis={{ tooltip: server.kernel }}>{server.kernel || '内核信息未采集'}</Text>
        </div>
      ),
    },
    {
      title: '资源容量',
      width: 205,
      render: (_, server) => (
        <Space size={4} wrap>
          <Tag>{server.cpuCores ? `${server.cpuCores}C` : 'CPU -'}</Tag>
          <Tag>{formatMemory(server.memoryMb)}</Tag>
          <Tag>{server.diskGb ? `${server.diskGb} GiB` : '磁盘 -'}</Tag>
        </Space>
      ),
    },
    {
      title: '资产标签',
      width: 180,
      render: (_, server) => {
        const labels = Object.entries(server.labels || {})
        return labels.length ? (
          <Space size={[4, 4]} wrap>
            {labels.slice(0, 2).map(([key, value]) => <Tag key={key}>{key}:{String(value)}</Tag>)}
            {labels.length > 2 ? <Tag>+{labels.length - 2}</Tag> : null}
          </Space>
        ) : <Text type="secondary">未设置</Text>
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 105,
      render: (status: string) => <StatusBadge status={status} />,
    },
    {
      title: '最近更新',
      dataIndex: 'updatedAt',
      width: 150,
      render: (value: string) => value ? <Tooltip title={dayjs(value).format('YYYY-MM-DD HH:mm:ss')}><Text type="secondary">{dayjs(value).format('MM-DD HH:mm')}</Text></Tooltip> : '-',
    },
    {
      title: '操作',
      width: 160,
      align: 'right',
      fixed: 'right',
      render: (_, server) => (
        <Space size={4}>
          <Tooltip title="打开终端">
            <Button
              type="primary"
              ghost
              size="small"
              icon={<CodeOutlined />}
              aria-label="打开终端"
              onClick={() => setTerminalServer(server)}
            />
          </Tooltip>
          <Tooltip title="查看详情">
            <Button
              size="small"
              icon={<EyeOutlined />}
              aria-label="查看服务器详情"
              onClick={() => setDetailServer(server)}
            />
          </Tooltip>
          <Dropdown
            trigger={['click']}
            menu={{
              items: [
                { key: 'inspect', icon: <ApiOutlined />, label: 'SSH 巡检' },
                { key: 'edit', icon: <EditOutlined />, label: '编辑资产' },
              ],
              onClick: ({ key }) => {
                if (key === 'inspect') testMutation.mutate(server.id)
                if (key === 'edit') openEditModal(server)
              },
            }}
          >
            <Button size="small" icon={<EllipsisOutlined />} loading={testingId === server.id} aria-label="更多操作" />
          </Dropdown>
          <Popconfirm
            title="移除服务器"
            description="仅移除平台资产记录，不会关闭或重装目标服务器。"
            okText="确认移除"
            okButtonProps={{ danger: true }}
            onConfirm={() => deleteMutation.mutate(server.id)}
          >
            <Button size="small" danger type="text" icon={<DeleteOutlined />} aria-label="移除服务器" />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const summary = summaryQuery.data || {
    total: serverQuery.data?.total || 0,
    available: (serverQuery.data?.items || []).filter((server) => server.status === 'available').length,
    registered: (serverQuery.data?.items || []).filter((server) => server.status === 'registered').length,
    unavailable: (serverQuery.data?.items || []).filter((server) => server.status === 'unavailable').length,
    cpuCores: (serverQuery.data?.items || []).reduce((total, server) => total + (server.cpuCores || 0), 0),
    memoryMb: (serverQuery.data?.items || []).reduce((total, server) => total + (server.memoryMb || 0), 0),
    diskGb: (serverQuery.data?.items || []).reduce((total, server) => total + (server.diskGb || 0), 0),
  }
  const onlineRate = summary?.total ? Math.round((summary.available / summary.total) * 100) : 0
  const credentialOptions = (credentialsQuery.data?.items || []).map((credential) => ({
    label: `${credential.name} · ${credential.username} · ${credential.authType === 'key' ? '密钥' : '密码'}`,
    value: credential.id,
  }))

  return (
    <AppPage className="app-server-pool-page">
      <section className="app-server-pool-hero">
        <div>
          <Space size={10} align="center">
            <span className="app-server-pool-hero__mark"><HddOutlined /></span>
            <div>
              <Title level={3}>主机资源池</Title>
              <Paragraph>统一维护 SSH 资产、连通状态与硬件画像，并从浏览器直接进入受控终端。</Paragraph>
            </div>
          </Space>
        </div>
        <Space wrap>
          <Button icon={<KeyOutlined />} onClick={() => history.push('/config/credentials')}>凭据库</Button>
          <Button icon={<DownloadOutlined />} onClick={exportInventory}>导出清单</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>添加服务器</Button>
        </Space>
      </section>

      <div className="app-server-pool-summary">
        <div className="app-server-pool-stat">
          <span>纳管主机</span>
          <strong>{summary?.total ?? '-'}</strong>
          <small>统一资产台账</small>
        </div>
        <div className="app-server-pool-stat app-server-pool-stat--healthy">
          <span>在线可用</span>
          <strong>{summary?.available ?? '-'}</strong>
          <small>在线率 {onlineRate}%</small>
        </div>
        <div className="app-server-pool-stat app-server-pool-stat--warning">
          <span>待处理</span>
          <strong>{(summary?.registered || 0) + (summary?.unavailable || 0)}</strong>
          <small>{summary?.registered || 0} 待巡检 · {summary?.unavailable || 0} 不可达</small>
        </div>
        <div className="app-server-pool-stat app-server-pool-stat--capacity">
          <span>资源总容量</span>
          <strong>{summary?.cpuCores || 0}C</strong>
          <small>{formatMemory(summary?.memoryMb)} 内存 · {summary?.diskGb || 0} GiB 磁盘</small>
        </div>
      </div>

      <Card className="app-server-pool-card" bordered={false}>
        <div className="app-server-pool-toolbar">
          <Space wrap>
            <Input
              value={searchDraft}
              allowClear
              prefix={<SearchOutlined />}
              placeholder="搜索名称或 IP"
              style={{ width: 240 }}
              onChange={(event) => setSearchDraft(event.target.value)}
              onPressEnter={() => { setPage(1); setKeyword(searchDraft.trim()) }}
              onClear={() => { setPage(1); setKeyword('') }}
            />
            <Select
              allowClear
              placeholder="全部状态"
              value={statusFilter}
              style={{ width: 140 }}
              options={Object.entries(statusMeta).map(([value, meta]) => ({ value, label: meta.text }))}
              onChange={(value) => { setPage(1); setStatusFilter(value) }}
            />
            <Button onClick={() => { setPage(1); setKeyword(searchDraft.trim()) }}>查询</Button>
          </Space>
          <Space wrap>
            <Button
              icon={<ThunderboltOutlined />}
              loading={batchTesting}
              onClick={() => void runBatchInspection()}
            >
              {selectedRowKeys.length ? `巡检已选 ${selectedRowKeys.length} 台` : '巡检当前页'}
            </Button>
            <Button icon={<ReloadOutlined />} loading={serverQuery.isFetching} onClick={() => void refreshServers()}>刷新</Button>
          </Space>
        </div>

        {selectedRowKeys.length ? (
          <div className="app-server-pool-selection">
            <Space wrap>
              <SafetyCertificateOutlined />
              <Text strong>已选择 {selectedRowKeys.length} 台服务器</Text>
              <Text type="secondary">可执行批量连通巡检或从资源池移除</Text>
            </Space>
            <Popconfirm
              title={`确认移除选中的 ${selectedRowKeys.length} 台服务器？`}
              description="该操作不会影响目标服务器本身。"
              okButtonProps={{ danger: true, loading: batchDeleting }}
              onConfirm={() => void deleteSelectedServers()}
            >
              <Button danger size="small" loading={batchDeleting}>批量移除</Button>
            </Popconfirm>
          </div>
        ) : null}

        {serverQuery.isError ? (
          <Alert type="error" showIcon message="服务器列表加载失败" description={(serverQuery.error as Error)?.message} style={{ marginBottom: 16 }} />
        ) : null}

        <Table
          rowKey="id"
          columns={columns}
          dataSource={serverQuery.data?.items || []}
          loading={serverQuery.isLoading}
          scroll={{ x: 1425 }}
          rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys, preserveSelectedRowKeys: false }}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无服务器，添加第一台主机开始纳管" /> }}
          pagination={{
            current: page,
            pageSize,
            total: serverQuery.data?.total || 0,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 台`,
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPageSize !== pageSize ? 1 : nextPage)
              setPageSize(nextPageSize)
              setSelectedRowKeys([])
            },
          }}
        />
      </Card>

      <Modal
        title={editing ? '编辑服务器资产' : '添加服务器'}
        open={modalOpen}
        width={640}
        destroyOnClose
        onCancel={() => { setModalOpen(false); setEditing(null); form.resetFields() }}
        onOk={() => void submitServer()}
        okText={editing ? '保存修改' : '添加并纳管'}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
      >
        <Alert
          type="info"
          showIcon
          message="平台使用已保存的凭据进行连通巡检、信息采集和 Web Shell 登录。"
          style={{ marginBottom: 18 }}
        />
        <Form form={form} layout="vertical" requiredMark="optional">
          <div className="app-server-form-grid">
            <Form.Item name="name" label="资产名称" rules={[{ required: true, message: '请输入服务器名称' }]}>
              <Input placeholder="例如：prod-master-01" />
            </Form.Item>
            <Form.Item name="ip" label="主机地址" rules={[{ required: true, message: '请输入 IP 或主机名' }]}>
              <Input placeholder="192.168.1.10" />
            </Form.Item>
            <Form.Item name="sshPort" label="SSH 端口" rules={[{ required: true, message: '请输入 SSH 端口' }]}>
              <InputNumber min={1} max={65535} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item name="user" label="登录用户" rules={[{ required: true, message: '请输入登录用户' }]}>
              <Input placeholder="root" />
            </Form.Item>
            <Form.Item name="authType" label="认证方式" rules={[{ required: true }]}>
              <Select options={[{ label: 'SSH 密钥', value: 'key' }, { label: '密码', value: 'password' }]} />
            </Form.Item>
            <Form.Item
              name="credentialId"
              label="SSH 凭据"
              rules={editing ? [] : [{ required: true, message: '请选择 SSH 凭据' }]}
            >
              <Select
                showSearch
                optionFilterProp="label"
                placeholder="选择凭据库中的凭据"
                loading={credentialsQuery.isLoading}
                options={credentialOptions}
                notFoundContent="暂无可用凭据，请先前往凭据库创建"
              />
            </Form.Item>
          </div>
          <Form.Item name="labelsText" label="资产标签" extra="多个标签使用逗号分隔，例如 env=prod, zone=cn-east-1">
            <Input placeholder="env=prod, role=worker" />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={3} maxLength={500} showCount placeholder="记录机房、用途、责任团队或维护窗口" />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={detailServer ? `服务器详情 · ${detailServer.name}` : '服务器详情'}
        open={Boolean(detailServer)}
        onClose={() => setDetailServer(null)}
        width={560}
        extra={detailServer ? (
          <Space>
            <Button icon={<ApiOutlined />} loading={testingId === detailServer.id} onClick={() => testMutation.mutate(detailServer.id)}>重新巡检</Button>
            <Button type="primary" icon={<CodeOutlined />} onClick={() => setTerminalServer(detailServer)}>打开终端</Button>
          </Space>
        ) : null}
      >
        {detailServer ? (
          <div className="app-server-detail">
            <div className="app-server-detail__identity">
              <span className={`app-server-cell__icon app-server-cell__icon--${detailServer.status}`}><CloudServerOutlined /></span>
              <div><Title level={4}>{detailServer.name}</Title><Text type="secondary">{detailServer.user}@{detailServer.ip}:{detailServer.sshPort}</Text></div>
              <StatusBadge status={detailServer.status} />
            </div>
            <Descriptions title="访问与系统" bordered size="small" column={1}>
              <Descriptions.Item label="主机地址"><Text copyable>{detailServer.ip}</Text></Descriptions.Item>
              <Descriptions.Item label="认证方式">{detailServer.authType === 'key' ? 'SSH 密钥' : '密码'}</Descriptions.Item>
              <Descriptions.Item label="操作系统">{[detailServer.os, detailServer.osVersion].filter(Boolean).join(' ') || '未采集'}</Descriptions.Item>
              <Descriptions.Item label="内核版本">{detailServer.kernel || '未采集'}</Descriptions.Item>
            </Descriptions>
            <Descriptions title="资源容量" bordered size="small" column={1}>
              <Descriptions.Item label="CPU">{detailServer.cpuCores ? `${detailServer.cpuCores} 核` : '未采集'}</Descriptions.Item>
              <Descriptions.Item label="内存">{formatMemory(detailServer.memoryMb)}</Descriptions.Item>
              <Descriptions.Item label="根磁盘">{detailServer.diskGb ? `${detailServer.diskGb} GiB` : '未采集'}</Descriptions.Item>
            </Descriptions>
            <Descriptions title="资产治理" bordered size="small" column={1}>
              <Descriptions.Item label="标签">
                {Object.entries(detailServer.labels || {}).length ? Object.entries(detailServer.labels || {}).map(([key, value]) => <Tag key={key}>{key}={String(value)}</Tag>) : '未设置'}
              </Descriptions.Item>
              <Descriptions.Item label="备注">{detailServer.remark || '无'}</Descriptions.Item>
              <Descriptions.Item label="创建时间">{dayjs(detailServer.createdAt).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
              <Descriptions.Item label="最近更新">{dayjs(detailServer.updatedAt).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
            </Descriptions>
            <Button block icon={<EditOutlined />} onClick={() => openEditModal(detailServer)}>编辑服务器信息</Button>
          </div>
        ) : null}
      </Drawer>

      <ServerTerminalDrawer open={Boolean(terminalServer)} server={terminalServer} onClose={() => setTerminalServer(null)} />
    </AppPage>
  )
}
