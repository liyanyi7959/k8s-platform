/**
 * SSH 凭证管理页
 * 管理 SSH 密码/密钥凭证，支持批量删除
 */
import { useState } from 'react'
import {
  Card, Table, Button, Space, Tag, Modal, Form, Input, Radio, Popconfirm,
  message, Typography, Tooltip, Alert,
} from 'antd'
import {
  PlusOutlined, DeleteOutlined, EditOutlined, ReloadOutlined,
  KeyOutlined, LockOutlined,
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { ColumnsType } from 'antd/es/table'
import { AppPage } from '@/components'
import { getCredentials, createCredential, updateCredential, deleteCredential, batchDeleteCredentials } from '@/services/deploy'
import type { Credential, CreateCredentialRequest } from '@/types/deploy'

const { Text } = Typography
const { TextArea } = Input

export default function CredentialsPage() {
  const queryClient = useQueryClient()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Credential | null>(null)
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([])
  const [form] = Form.useForm()

  const credType = Form.useWatch('authType', form) || 'key'

  const { data, isLoading } = useQuery({
    queryKey: ['deploy-credentials'],
    queryFn: () => getCredentials(),
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateCredentialRequest) => createCredential(data),
    onSuccess: () => {
      message.success('创建成功')
      setModalOpen(false)
      form.resetFields()
      queryClient.invalidateQueries({ queryKey: ['deploy-credentials'] })
    },
    onError: (err: any) => message.error(err?.message || '创建失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<CreateCredentialRequest> }) => updateCredential(id, data),
    onSuccess: () => {
      message.success('更新成功')
      setModalOpen(false)
      setEditing(null)
      form.resetFields()
      queryClient.invalidateQueries({ queryKey: ['deploy-credentials'] })
    },
    onError: (err: any) => message.error(err?.message || '更新失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteCredential(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['deploy-credentials'] })
    },
    onError: () => message.error('删除失败'),
  })

  const batchDeleteMutation = useMutation({
    mutationFn: (ids: number[]) => batchDeleteCredentials(ids),
    onSuccess: () => {
      message.success(`成功删除 ${selectedRowKeys.length} 条`)
      setSelectedRowKeys([])
      queryClient.invalidateQueries({ queryKey: ['deploy-credentials'] })
    },
    onError: () => message.error('批量删除失败'),
  })

  const handleAdd = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (record: Credential) => {
    setEditing(record)
    form.setFieldsValue({
      name: record.name,
      authType: record.authType,
      username: record.username,
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

  const columns: ColumnsType<Credential> = [
    {
      title: '凭证名称',
      dataIndex: 'name',
      width: 180,
      render: (name: string) => <Text strong>{name}</Text>,
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 120,
      align: 'center',
      render: (type: string) => {
        const map: Record<string, { color: string; text: string; icon: React.ReactNode }> = {
          key: { color: 'blue', text: '密钥', icon: <KeyOutlined /> },
          password: { color: 'orange', text: '密码', icon: <LockOutlined /> },
        }
        const item = map[type] || { color: 'default', text: type, icon: null }
        return <Tag color={item.color} icon={item.icon}>{item.text}</Tag>
      },
    },
    {
      title: '用户名',
      dataIndex: 'username',
      width: 140,
    },
    {
      title: '使用情况',
      width: 140,
      align: 'center',
      render: (_: unknown, record: Credential) => {
        return <Text type="secondary">{record.serverCount ? `${record.serverCount} 台服务器引用` : '未被引用'}</Text>
      },
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 170,
      render: (v: string) => v ? new Date(v).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      width: 120,
      align: 'center',
      fixed: 'right',
      render: (_: unknown, record: Credential) => (
        <Space>
          <Tooltip title="编辑">
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          </Tooltip>
          <Popconfirm title="确认删除该凭证？" onConfirm={() => deleteMutation.mutate(record.id)}>
            <Tooltip title="删除">
              <Button type="link" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <AppPage>
      <Card
        title={
          <Space>
            <Text type="secondary">已管理 {data?.items?.length || 0} 份 SSH 凭证</Text>
            {selectedRowKeys.length > 0 && (
              <Text type="secondary" style={{ fontWeight: 'normal', fontSize: 13 }}>
                已选 {selectedRowKeys.length} 项
              </Text>
            )}
          </Space>
        }
        extra={
          <Space>
            {selectedRowKeys.length > 0 && (
              <Popconfirm
                title={`确认删除选中的 ${selectedRowKeys.length} 条凭证？`}
                onConfirm={() => batchDeleteMutation.mutate(selectedRowKeys)}
              >
                <Button danger icon={<DeleteOutlined />}>批量删除</Button>
              </Popconfirm>
            )}
            <Button icon={<ReloadOutlined />} onClick={() => queryClient.invalidateQueries({ queryKey: ['deploy-credentials'] })}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              添加凭证
            </Button>
          </Space>
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={data?.items || []}
          loading={isLoading}
          scroll={{ x: 900 }}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys) => setSelectedRowKeys(keys as number[]),
          }}
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        />
      </Card>

      <Modal
        title={editing ? '编辑凭证' : '添加凭证'}
        open={modalOpen}
        onCancel={() => { setModalOpen(false); setEditing(null); form.resetFields() }}
        onOk={handleSubmit}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        destroyOnClose
        width={520}
      >
        <Form form={form} layout="vertical" initialValues={{ authType: 'key' }}>
          <Form.Item name="name" label="凭证名称" rules={[{ required: true, message: '请输入凭证名称' }]}>
            <Input placeholder="例如：生产环境 root 密钥" />
          </Form.Item>

          <Form.Item name="authType" label="认证方式" rules={[{ required: true }]}> 
            <Radio.Group>
              <Radio.Button value="key"><KeyOutlined /> SSH 密钥</Radio.Button>
              <Radio.Button value="password"><LockOutlined /> 密码</Radio.Button>
            </Radio.Group>
          </Form.Item>

          <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input placeholder="root" />
          </Form.Item>

          {credType === 'key' && (
            <>
              <Form.Item
                name="privateKey"
                label="私钥内容"
                rules={[{ required: !editing, message: '请输入私钥' }]}
              >
                <TextArea
                  rows={6}
                  placeholder="-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"
                  style={{ fontFamily: 'monospace', fontSize: 12 }}
                />
              </Form.Item>
              <Form.Item name="passphrase" label="密钥密码（可选）">
                <Input.Password disabled placeholder="当前后端不单独存储 passphrase，如有需要请直接使用未加密或预处理后的私钥" />
              </Form.Item>
            </>
          )}

          {credType === 'password' && (
            <Form.Item
              name="password"
              label="登录密码"
              rules={[{ required: !editing, message: '请输入密码' }]}
            >
              <Input.Password placeholder="SSH 登录密码" />
            </Form.Item>
          )}

          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={2} placeholder="可选备注信息" />
          </Form.Item>

          {editing && (
            <Alert
              type="info"
              showIcon
              message="编辑模式下，敏感字段（密码/私钥）留空则保持不变。"
              style={{ marginBottom: 16 }}
            />
          )}
        </Form>
      </Modal>
    </AppPage>
  )
}
