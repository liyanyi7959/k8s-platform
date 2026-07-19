/**
 * 部署配置页 - Ansible 流水线概览 + 仓库配置
 * 基于 Ansible Playbook 的 K8s 部署流程说明
 */
import React, { useEffect, useState } from 'react'
import { Card, Tabs, Table, Button, Space, Tag, Modal, Form, Input, InputNumber, Switch, Select, Popconfirm, message, Badge, Typography, Alert, Spin } from 'antd'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { AppPage, YamlEditor } from '@/components'
import {
  listRepositories,
  createRepository,
  updateRepository,
  deleteRepository,
  getAnsiblePlaybook,
  getAnsibleInventoryTemplate,
  checkAnsibleEnv,
} from '@/services/deploy'
import type { RepositoryConfig } from '@/types'
import {
  PlusOutlined,
  DesktopOutlined,
  CheckCircleOutlined,
  ContainerOutlined,
  ClusterOutlined,
  ApartmentOutlined,
  CloudUploadOutlined,
  RightOutlined,
} from '@ant-design/icons'

const { Text } = Typography

type AnsiblePhase = 'preflight' | 'install' | 'init' | 'join' | 'addon' | 'finalize'

/** Ansible 部署流水线步骤定义 */
interface AnsibleStep {
  key: string
  title: string
  description: string
  icon: React.ReactNode
  appliesTo: string
  phase: AnsiblePhase
  tasks: string[]
}

const phaseConfig: Record<
  AnsiblePhase,
  { color: string; label: string; borderColor: string; bgColor: string }
> = {
  preflight: { color: 'default', label: '环境预检', borderColor: '#d9d9d9', bgColor: '#f5f5f5' },
  install: { color: 'blue', label: '软件安装', borderColor: '#1677ff', bgColor: '#e6f4ff' },
  init: { color: 'gold', label: '集群初始化', borderColor: '#faad14', bgColor: '#fffbe6' },
  join: { color: 'cyan', label: '节点加入', borderColor: '#13c2c2', bgColor: '#e6fffb' },
  addon: { color: 'green', label: '网络插件', borderColor: '#52c41a', bgColor: '#f6ffed' },
  finalize: { color: 'purple', label: '平台注册', borderColor: '#722ed1', bgColor: '#f9f0ff' },
}

const ansibleSteps: AnsibleStep[] = [
  {
    key: 'pre_check',
    title: '环境预检',
    description: '检查目标节点的 CPU、内存、磁盘、端口、hostname 等环境要求',
    icon: <CheckCircleOutlined />,
    appliesTo: '所有节点',
    phase: 'preflight',
    tasks: [
      'Ping 测试 SSH 连通性',
      '检查 OS 发行版和版本号',
      '检查 CPU 核数 >= 2',
      '检查内存 >= 2048MB',
      '检查磁盘可用空间 >= 20GB',
      '检查 hostname 是否设置且唯一',
      'Master 节点检查端口 6443/2379/2380 等未被占用',
    ],
  },
  {
    key: 'bootstrap',
    title: '基础环境初始化',
    description: '关闭 swap、加载内核模块、设置 sysctl、禁用防火墙、安装基础依赖',
    icon: <DesktopOutlined />,
    appliesTo: '所有节点',
    phase: 'preflight',
    tasks: [
      '关闭 swap 并注释 /etc/fstab',
      '加载内核模块 br_netfilter, overlay, nf_conntrack',
      '设置 sysctl 参数（ip_forward 等）',
      '停止并禁用 firewalld',
      '设置 SELinux 为 permissive/disabled',
      '安装基础包: lvm2, wget, curl, vim, chrony',
      '配置 Kubernetes yum/apt 仓库',
    ],
  },
  {
    key: 'container_runtime',
    title: '容器运行时安装',
    description: '安装 containerd，配置 SystemdCgroup 和 sandbox 镜像',
    icon: <ContainerOutlined />,
    appliesTo: '所有节点',
    phase: 'install',
    tasks: [
      '安装 Docker CE 仓库（RedHat/Debian 兼容）',
      '安装 containerd.io',
      '生成 containerd 默认配置',
      '设置 SystemdCgroup=true',
      '设置 sandbox_image=registry.k8s.io/pause:3.9',
      '启动 containerd 并设置开机自启',
    ],
  },
  {
    key: 'kubeadm_init',
    title: 'Master 初始化',
    description: '安装 kubeadm/kubelet/kubectl，执行 kubeadm init 初始化集群',
    icon: <ClusterOutlined />,
    appliesTo: 'Master 节点',
    phase: 'init',
    tasks: [
      '安装 kubeadm, kubelet, kubectl',
      '锁定版本不自动更新',
      '生成 kubeadm-init.yaml 配置文件',
      '执行 kubeadm init --config /tmp/kubeadm-init.yaml',
      '配置 kubeconfig (~/.kube/config)',
      '生成 worker join 命令',
    ],
  },
  {
    key: 'join_workers',
    title: 'Worker 加入集群',
    description: '在 Worker 节点安装 kubeadm/kubelet，执行 kubeadm join',
    icon: <ApartmentOutlined />,
    appliesTo: 'Worker 节点',
    phase: 'join',
    tasks: [
      '安装 kubeadm, kubelet',
      '锁定版本不自动更新',
      '执行 kubeadm join <master_ip>:6443',
    ],
  },
  {
    key: 'install_cni',
    title: '安装 CNI 网络插件',
    description: '安装 Flannel/Calico/Cilium 网络插件，等待 Pods 就绪',
    icon: <ApartmentOutlined />,
    appliesTo: 'Master 节点',
    phase: 'addon',
    tasks: [
      '根据 cni_type 变量选择 CNI 插件',
      'Flannel: 下载并应用 kube-flannel.yml',
      'Calico: 下载并应用 calico.yaml',
      'Cilium: 安装 cilium CLI 并部署',
      '等待 CNI Pods 就绪',
      '检查所有节点状态为 Ready',
    ],
  },
  {
    key: 'register',
    title: '注册到管理平台',
    description: '提取 kubeconfig，注册集群到 K8s 管理平台',
    icon: <CloudUploadOutlined />,
    appliesTo: 'Master 节点',
    phase: 'finalize',
    tasks: [
      '读取 /etc/kubernetes/admin.conf',
      '替换 API Server 地址为 Master 实际 IP',
      '写入 kubeconfig 到临时文件',
      '后端读取 kubeconfig 并注册集群',
    ],
  },
]


const DeployConfig: React.FC = () => {
  const [activeTab, setActiveTab] = useState('pipeline')

  return (
    <AppPage>
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            { key: 'pipeline', label: 'Ansible 部署流水线', children: <AnsiblePipelinePanel /> },
            { key: 'playbook', label: 'Playbook 源码', children: <AnsiblePlaybookPanel /> },
            { key: 'inventory', label: 'Inventory 模板', children: <AnsibleInventoryPanel /> },
            { key: 'env', label: '环境检查', children: <AnsibleEnvPanel /> },
            { key: 'repos', label: '仓库配置', children: <RepoConfigPanel /> },
          ]}
        />
      </Card>
    </AppPage>
  )
}

// ==================== Ansible 部署流水线概览 ====================

const AnsiblePipelinePanel: React.FC = () => {
  const [selectedKey, setSelectedKey] = useState<string | null>(null)
  const [expandedKeys, setExpandedKeys] = useState<string[]>([])

  // 点击顶部卡片后：高亮并展开对应表格行，再滚动定位
  useEffect(() => {
    if (!selectedKey) return
    setExpandedKeys((prev) => (prev.includes(selectedKey) ? prev : [...prev, selectedKey]))
    const row = document.querySelector(`.deploy-step-row-${selectedKey}`)
    row?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
  }, [selectedKey])

  return (
    <>
      {/* 流程可视化 */}
      <div style={{ display: 'flex', gap: 0, overflowX: 'auto', paddingBottom: 16, marginBottom: 24 }}>
        {ansibleSteps.map((step, idx) => {
          const phase = phaseConfig[step.phase]
          const isSelected = selectedKey === step.key
          return (
            <div key={step.key} style={{ display: 'flex', alignItems: 'center', flexShrink: 0 }}>
              <div
                onClick={() => setSelectedKey(step.key)}
                style={{
                  width: 220,
                  minHeight: 150,
                  border: `${isSelected ? 2 : 1}px solid ${phase.borderColor}`,
                  borderRadius: 10,
                  padding: 14,
                  background: isSelected ? phase.bgColor : '#fff',
                  boxShadow: isSelected ? `0 0 0 2px ${phase.borderColor}33, 0 8px 20px rgba(0, 0, 0, 0.1)` : '0 2px 8px rgba(0, 0, 0, 0.06)',
                  transition: 'all 0.2s ease',
                  position: 'relative',
                  overflow: 'hidden',
                  cursor: 'pointer',
                }}
                onMouseEnter={(e) => {
                  if (selectedKey === step.key) return
                  e.currentTarget.style.transform = 'translateY(-4px)'
                  e.currentTarget.style.boxShadow = '0 8px 20px rgba(0, 0, 0, 0.1)'
                }}
                onMouseLeave={(e) => {
                  if (selectedKey === step.key) return
                  e.currentTarget.style.transform = 'translateY(0)'
                  e.currentTarget.style.boxShadow = '0 2px 8px rgba(0, 0, 0, 0.06)'
                }}
              >
                {/* 阶段色顶部条 */}
                <div
                  style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    right: 0,
                    height: 3,
                    background: phase.borderColor,
                  }}
                />

                {/* 步骤图标 + 标题 */}
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10, marginBottom: 10, marginTop: 4 }}>
                  <div
                    style={{
                      width: 36,
                      height: 36,
                      borderRadius: '50%',
                      background: isSelected ? '#fff' : phase.bgColor,
                      color: phase.color === 'default' ? '#595959' : phase.borderColor,
                      border: `1px solid ${phase.borderColor}`,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 16,
                      flexShrink: 0,
                    }}
                  >
                    {step.icon}
                  </div>
                  <div style={{ minWidth: 0 }}>
                    <Text strong style={{ fontSize: 14, display: 'block' }}>{step.title}</Text>
                    <Text code style={{ fontSize: 11 }}>{step.key}</Text>
                  </div>
                </div>

                {/* 阶段 + 适用范围 */}
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 8 }}>
                  <Tag color={phase.color} style={{ fontSize: 11, margin: 0 }}>{phase.label}</Tag>
                  <Tag color={step.appliesTo.includes('所有') ? 'blue' : step.appliesTo.includes('Master') ? 'volcano' : 'cyan'} style={{ fontSize: 11, margin: 0 }}>
                    {step.appliesTo}
                  </Tag>
                </div>

                {/* 描述 */}
                <div style={{ fontSize: 12, color: '#595959', minHeight: 36, lineHeight: '18px', overflow: 'hidden', marginBottom: 8 }}>
                  {step.description}
                </div>

                {/* 任务数量 */}
                <div style={{ fontSize: 12, color: '#8c8c8c', display: 'flex', alignItems: 'center', gap: 6 }}>
                  <Badge count={step.tasks.length} style={{ backgroundColor: '#1677ff' }} />
                  <span>个 Ansible 任务</span>
                </div>
              </div>

              {/* 连接箭头 */}
              {idx < ansibleSteps.length - 1 && (
                <div style={{ padding: '0 10px', color: '#bfbfbf', fontSize: 16, display: 'flex', alignItems: 'center' }}>
                  <RightOutlined />
                </div>
              )}
            </div>
          )
        })}
      </div>

      {/* 详细步骤列表 */}
      <Card bodyStyle={{ padding: '0 16px' }} title="步骤详情" size="small">
        <Table
          rowKey="key"
          dataSource={ansibleSteps}
          pagination={false}
          size="small"
          scroll={{ x: 700 }}
          rowClassName={(record) => `deploy-step-row-${record.key}`}
          expandable={{
            expandRowByClick: true,
            expandedRowKeys: expandedKeys,
            onExpandedRowsChange: (keys) => setExpandedKeys(keys as string[]),
            expandedRowRender: (record: AnsibleStep) => (
              <div style={{ padding: '8px 16px' }}>
                <Text strong style={{ fontSize: 12, display: 'block', marginBottom: 8 }}>Ansible 任务</Text>
                <ul style={{ margin: 0, paddingLeft: 16, fontSize: 12, lineHeight: '1.8', color: '#595959' }}>
                  {record.tasks.map((task, i) => <li key={i}>{task}</li>)}
                </ul>
              </div>
            ),
          }}
          onRow={(record) => ({
            onClick: () => setSelectedKey(record.key),
            style: {
              cursor: 'pointer',
              backgroundColor: record.key === selectedKey ? phaseConfig[record.phase].bgColor : undefined,
              transition: 'background-color 0.2s ease',
            },
          })}
          columns={[
            {
              title: '步骤',
              dataIndex: 'key',
              key: 'key',
              width: 200,
              render: (key: string, record: AnsibleStep) => (
                <Space>
                  <span style={{ color: phaseConfig[record.phase].color === 'default' ? '#595959' : phaseConfig[record.phase].color, fontSize: 16 }}>
                    {record.icon}
                  </span>
                  <div>
                    <Text strong>{record.title}</Text>
                    <div><Text code style={{ fontSize: 11 }}>{key}</Text></div>
                  </div>
                </Space>
              ),
            },
            {
              title: '阶段',
              dataIndex: 'phase',
              key: 'phase',
              width: 100,
              render: (phase: AnsiblePhase) => {
                const cfg = phaseConfig[phase]
                return <Tag color={cfg.color} style={{ margin: 0 }}>{cfg.label}</Tag>
              },
            },
            {
              title: '适用范围',
              dataIndex: 'appliesTo',
              key: 'appliesTo',
              width: 110,
              render: (text: string) => (
                <Tag color={text.includes('所有') ? 'blue' : text.includes('Master') ? 'volcano' : 'cyan'} style={{ margin: 0 }}>{text}</Tag>
              ),
            },
            {
              title: '说明',
              dataIndex: 'description',
              key: 'description',
              ellipsis: true,
            },
          ]}
        />
      </Card>
    </>
  )
}

// ==================== Playbook 源码 ====================

const AnsiblePlaybookPanel: React.FC = () => {
  const { data, isLoading } = useQuery({
    queryKey: ['ansible-playbook'],
    queryFn: () => getAnsiblePlaybook(),
  })

  return (
    <Spin spinning={isLoading}>
      {data?.content ? (
        <>
          <Alert
            type="info"
            showIcon
            message={`Playbook 路径：${data.path || 'ansible/site.yml'}`}
            style={{ marginBottom: 12 }}
          />
          <YamlEditor readOnly value={data.content} height={640} />
        </>
      ) : (
        <Alert type="warning" showIcon message="未找到 Playbook 文件，请确认后端 ansible/site.yml 是否存在" />
      )}
    </Spin>
  )
}

// ==================== Inventory 模板 ====================

const AnsibleInventoryPanel: React.FC = () => {
  const { data, isLoading } = useQuery({
    queryKey: ['ansible-inventory-template'],
    queryFn: () => getAnsibleInventoryTemplate(),
  })

  return (
    <Spin spinning={isLoading}>
      {data?.content ? (
        <>
          <Alert
            type="info"
            showIcon
            message={`Inventory 模板路径：${data.path || 'ansible/inventory.ini'}`}
            style={{ marginBottom: 12 }}
          />
          <YamlEditor readOnly value={data.content} height={640} />
        </>
      ) : (
        <Alert type="warning" showIcon message="未找到 Inventory 模板文件" />
      )}
    </Spin>
  )
}

// ==================== Ansible 环境检查 ====================

const AnsibleEnvPanel: React.FC = () => {
  const { data, isLoading } = useQuery({
    queryKey: ['ansible-env-check'],
    queryFn: () => checkAnsibleEnv(),
  })

  return (
    <Spin spinning={isLoading}>
      {data?.installed ? (
        <Alert
          type="success"
          showIcon
          message="Ansible 已安装"
          description={data.version}
        />
      ) : (
        <Alert
          type="error"
          showIcon
          message="Ansible 未安装"
          description={
            <div>
              <p>后端执行 Ansible 部署需要系统安装 ansible-playbook 命令。</p>
              <p>安装方式：</p>
              <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4 }}>
{`# Ubuntu/Debian
apt install ansible

# CentOS/RHEL
yum install ansible

# 或 pip
pip install ansible`}
              </pre>
            </div>
          }
        />
      )}
    </Spin>
  )
}

// ==================== 仓库配置 ====================

const REPO_TYPE_OPTIONS = [
  { value: 'container_mirror', label: '容器镜像加速' },
  { value: 'registry', label: '容器镜像仓库' },
  { value: 'yum', label: 'YUM 源' },
  { value: 'apt', label: 'APT 源' },
]

const AUTH_TYPE_OPTIONS = [
  { value: 'none', label: '无需认证' },
  { value: 'basic', label: '用户名/密码' },
  { value: 'token', label: 'Token' },
]

const RepoConfigPanel: React.FC = () => {
  const [modalOpen, setModalOpen] = useState(false)
  const [editRecord, setEditRecord] = useState<RepositoryConfig | null>(null)
  const [form] = Form.useForm()
  const queryClient = useQueryClient()

  const { data: repos, isLoading } = useQuery({
    queryKey: ['repositories'],
    queryFn: ({ signal }) => listRepositories(undefined, signal),
  })

  const createMutation = useMutation({
    mutationFn: createRepository,
    onSuccess: () => {
      message.success('仓库创建成功')
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
      setModalOpen(false)
      form.resetFields()
    },
  })

  const updateMutation = useMutation({
    mutationFn: (values: Partial<RepositoryConfig>) => updateRepository(editRecord!.id, values),
    onSuccess: () => {
      message.success('仓库更新成功')
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
      setModalOpen(false)
      setEditRecord(null)
      form.resetFields()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteRepository,
    onSuccess: () => {
      message.success('仓库已删除')
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
    },
  })

  const handleAdd = () => {
    setEditRecord(null)
    form.resetFields()
    form.setFieldsValue({ authType: 'none', priority: 1, enabled: true })
    setModalOpen(true)
  }

  const handleEdit = (record: RepositoryConfig) => {
    setEditRecord(record)
    form.setFieldsValue(record)
    setModalOpen(true)
  }

  const columns = [
    {
      title: '仓库名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: RepositoryConfig) => (
        <Space>
          <strong>{text}</strong>
          {record.isDefault && <Tag color="blue">默认</Tag>}
        </Space>
      ),
    },
    {
      title: '类型',
      dataIndex: 'repoType',
      key: 'repoType',
      width: 120,
      render: (t: string) => {
        const opt = REPO_TYPE_OPTIONS.find((o) => o.value === t)
        return <Tag>{opt?.label || t}</Tag>
      },
    },
    {
      title: '地址',
      dataIndex: 'url',
      key: 'url',
      ellipsis: true,
    },
    {
      title: '认证',
      dataIndex: 'authType',
      key: 'authType',
      width: 100,
      render: (t: string) => {
        const opt = AUTH_TYPE_OPTIONS.find((o) => o.value === t)
        return opt?.label || t
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      sorter: (a: RepositoryConfig, b: RepositoryConfig) => (a.priority ?? 0) - (b.priority ?? 0),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean) => (
        <Badge status={enabled ? 'success' : 'default'} text={enabled ? '启用' : '禁用'} />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: RepositoryConfig) => (
        <Space>
          <a onClick={() => handleEdit(record)}>编辑</a>
          <Popconfirm title="确认删除该仓库配置？" onConfirm={() => deleteMutation.mutate(record.id)}>
            <a style={{ color: '#ff4d4f' }}>删除</a>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          添加仓库
        </Button>
      </div>

      <Table
        rowKey="id"
        columns={columns}
        dataSource={repos}
        loading={isLoading}
        pagination={false}
      />

      <Modal
        title={editRecord ? '编辑仓库' : '添加仓库'}
        open={modalOpen}
        onCancel={() => { setModalOpen(false); setEditRecord(null); form.resetFields() }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        width={560}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => {
            if (editRecord) {
              updateMutation.mutate(values)
            } else {
              createMutation.mutate(values)
            }
          }}
        >
          <Form.Item name="name" label="仓库名称" rules={[{ required: true, message: '请输入仓库名称' }]}>
            <Input placeholder="如 Harbor 私有仓库" />
          </Form.Item>
          <Form.Item name="repoType" label="仓库类型" rules={[{ required: true, message: '请选择仓库类型' }]}>
            <Select options={REPO_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="url" label="仓库地址" rules={[{ required: true, message: '请输入仓库地址' }]}>
            <Input placeholder="https://harbor.example.com" />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="authType" label="认证方式">
            <Select options={AUTH_TYPE_OPTIONS} />
          </Form.Item>
          <Space>
            <Form.Item name="priority" label="优先级">
              <InputNumber min={1} max={100} />
            </Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </>
  )
}

export default DeployConfig
