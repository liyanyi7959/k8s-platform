/**
 * 服务器管理页
 * 管理部署目标服务器，支持 SSH 连通性测试
 */
import { useState } from 'react'
import {
  Card, Table, Button, Space, Modal, Form, Input, InputNumber, Select,
  Popconfirm, message, Typography, Tooltip, Badge,
} from 'antd'
import {
  PlusOutlined, DeleteOutlined, EditOutlined, ReloadOutlined,
  ApiOutlined,
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { ColumnsType } from 'antd/es/table'
import { AppPage } from '@/components'
import { getServers, createServer, updateServer, deleteServer, testServerSSH } from '@/services/deploy'
import { getCredentials } from '@/services/deploy'
import type { DeployServer, CreateServerRequest } from '@/types/deploy'

const { Text } = Typography

/** 状态标签 */
function StatusTag({ status }: { status: string }) {
  const map: Record<string, { color: string; text: string }> = {
    available: { color: 'green', text: '可用' },
    registered: { color: 'gold', text: '已注册' },
    unavailable: { color: 'red', text: '不可用' },
  }
  const item = map[status as keyof typeof map] ?? { color: 'default', text: status || '未知' }
  return <Badge status={status === 'available' ? 'success' : status === 'unavailable' ? 'error' : 'processing'} text={item.text} />
}

export default function ServersPage() {
  const queryClient = useQueryClient()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<DeployServer | null>(null)
  const [testingId, setTestingId] = useState<number | null>(null)
  const [form] = Form.useForm()

  const { data, isLoading } = useQuery({
    queryKey: ['deploy-servers'],
    queryFn: () => getServers(),
  })

  const { data: credentialsData } = useQuery({
    queryKey: ['deploy-credentials-for-select'],
    queryFn: () => getCredentials(),
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateServerRequest) => createServer(data),
    onSuccess: () => {
      message.success('创建成功')
      setModalOpen(false)
      form.resetFields()
      queryClient.invalidateQueries({ queryKey: ['deploy-servers'] })
    },
    onError: (err: any) => message.error(err?.message || '创建失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<CreateServerRequest> }) => updateServer(id, data),
    onSuccess: () => {
      message.success('更新成功')
      setModalOpen(false)
      setEditing(null)
      form.resetFields()
      queryClient.invalidateQueries({ queryKey: ['deploy-servers'] })
    },
    onError: (err: any) => message.error(err?.message || '更新失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteServer(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['deploy-servers'] })
    },
    onError: () => message.error('删除失败'),
  })

  const testSSHMutation = useMutation({
    mutationFn: (id: number) => {
      setTestingId(id)
      return testServerSSH(id)
    },
    onSuccess: (res) => {
      if (res.success) {
        message.success(res.message || 'SSH 连接成功')
      } else {
        message.warning(res.message || 'SSH 连接失败')
      }
      setTestingId(null)
      queryClient.invalidateQueries({ queryKey: ['deploy-servers'] })
    },
    onError: (err: any) => {
      message.error(err?.message || 'SSH 测试失败')
      setTestingId(null)
    },
  })

  const handleAdd = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (record: DeployServer) => {
    setEditing(record)
    form.setFieldsValue({
      name: record.name,
      ip: record.ip,
      sshPort: record.sshPort,
      user: record.user,
      authType: record.authType,
      credentialId: record.credentialId,
      remark: record.remark,
    })
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

  const columns: ColumnsType<DeployServer> = [
    {
      title: '服务器名称',
      dataIndex: 'name',
      width: 140,
      render: (name: string) => <Text strong>{name || '-'}</Text>,
    },
    {
      title: '主机地址',
      dataIndex: 'ip',
      width: 180,
      render: (ip: string, record) => (
        <Space>
          <Text strong copyable={{ text: ip }}>{ip}</Text>
          <Text type="secondary">:{record.sshPort}</Text>
        </Space>
      ),
    },
    {
      title: '登录用户',
      dataIndex: 'user',
      width: 100,
    },
    {
      title: '系统信息',
      width: 200,
      render: (_: unknown, record: DeployServer) => {
        const parts = [record.os, record.osVersion].filter(Boolean)
        return parts.length ? parts.join(' ') : <Text type="secondary">待检测</Text>
      },
    },
    {
      title: '资源',
      width: 160,
      render: (_: unknown, record: DeployServer) => {
        if (!record.cpuCores && !record.memoryMb && !record.diskGb) {
          return <Text type="secondary">待检测</Text>
        }
        return (
          <Space split={<span style={{ color: '#d9d9d9' }}>|</span>}>
            <Tooltip title="CPU 核数"><Text>{record.cpuCores || '-'}C</Text></Tooltip>
            <Tooltip title="内存"><Text>{record.memoryMb ? `${(record.memoryMb / 1024).toFixed(0)}G` : '-'}</Text></Tooltip>
            <Tooltip title="磁盘"><Text>{record.diskGb ? `${record.diskGb}G` : '-'}</Text></Tooltip>
          </Space>
        )
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      align: 'center',
      render: (status: string) => <StatusTag status={status} />,
    },
    {
      title: '操作',
      width: 180,
      align: 'center',
      fixed: 'right',
      render: (_: unknown, record: DeployServer) => (
        <Space>
          <Tooltip title="测试 SSH 连接">
            <Button
              type="link"
              size="small"
              icon={<ApiOutlined />}
              loading={testingId === record.id}
              onClick={() => testSSHMutation.mutate(record.id)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          </Tooltip>
          <Popconfirm title="确认删除该服务器？" onConfirm={() => deleteMutation.mutate(record.id)}>
            <Tooltip title="删除">
              <Button type="link" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const credentialOptions = (credentialsData?.items || []).map((c) => ({
    label: `${c.name} (${c.username})`,
    value: c.id,
  }))

  return (
    <AppPage>
      <Card
        title={
          <Space>
            {data?.items && (
              <Text type="secondary" style={{ fontWeight: 'normal', fontSize: 13 }}>
                共 {data.items.length} 台服务器
                {data.items.filter((s) => s.status === 'available').length > 0 && (
                  <>, <Text style={{ color: '#52c41a' }}>{data.items.filter((s) => s.status === 'available').length} 可用</Text></>
                )}
              </Text>
            )}
          </Space>
        }
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => queryClient.invalidateQueries({ queryKey: ['deploy-servers'] })}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              添加服务器
            </Button>
          </Space>
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={data?.items || []}
          loading={isLoading}
          scroll={{ x: 1000 }}
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        />
      </Card>

      <Modal
        title={editing ? '编辑服务器' : '添加服务器'}
        open={modalOpen}
        onCancel={() => { setModalOpen(false); setEditing(null); form.resetFields() }}
        onOk={handleSubmit}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        destroyOnClose
        width={520}
      >
        <Form form={form} layout="vertical" initialValues={{ sshPort: 22, user: 'root', authType: 'key' }}>
          <Form.Item name="name" label="服务器名称" rules={[{ required: true, message: '请输入服务器名称' }]}>
            <Input placeholder="例如：web-server-01" />
          </Form.Item>
          <Form.Item name="ip" label="主机 IP" rules={[{ required: true, message: '请输入主机 IP' }]}>
            <Input placeholder="192.168.1.10" />
          </Form.Item>
          <Form.Item name="sshPort" label="SSH 端口" rules={[{ required: true }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="user" label="登录用户" rules={[{ required: true }]}>
            <Input placeholder="root" />
          </Form.Item>
          <Form.Item name="authType" label="认证方式" rules={[{ required: true }]}>
            <Select
              options={[
                { label: 'SSH 密钥', value: 'key' },
                { label: '密码', value: 'password' },
              ]}
            />
          </Form.Item>
          <Form.Item name="credentialId" label="SSH 凭证" rules={editing ? [] : [{ required: true, message: '请选择 SSH 凭证' }]}>
            <Select
              placeholder="选择凭证"
              options={credentialOptions}
              showSearch
              optionFilterProp="label"
              notFoundContent="暂无凭证，请先在「SSH 凭证」页面创建"
            />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={2} placeholder="可选备注信息" />
          </Form.Item>
        </Form>
      </Modal>
    </AppPage>
  )
}
