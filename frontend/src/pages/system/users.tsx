import React, { useMemo, useState, useEffect } from 'react'
import {
  ModalForm,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProTable,
  type ProColumns,
} from '@ant-design/pro-components'
import {
  Button,
  Input,
  message,
  Popconfirm,
  Select,
  Space,
  Switch,
  Tag,
  Tooltip,
} from 'antd'
import {
  DeleteOutlined,
  EditOutlined,
  KeyOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import {
  createUser,
  deleteUser,
  listAllRoles,
  listUsers,
  resetPassword,
  updateUser,
} from '@/services/system'
import {
  resetPasswordSchema,
  userCreateSchema,
  userEditSchema,
} from '@/schemas/system'
import { formatDate } from '@/utils'
import type { Role, User } from '@/types'
import type { UserCreateInput, UserEditInput } from '@/schemas/system'

type StatusFilter = 'active' | 'disabled' | undefined

const getRoleIds = (user?: User | null): number[] =>
  user?.roles?.map((role) => role.id) || []

/** 用户管理页 */
const UsersPage: React.FC = () => {
  const queryClient = useQueryClient()
  const [createVisible, setCreateVisible] = useState(false)
  const [editVisible, setEditVisible] = useState(false)
  const [resetVisible, setResetVisible] = useState(false)
  const [currentUser, setCurrentUser] = useState<User | null>(null)

  const [searchText, setSearchText] = useState('')
  const [debouncedKeyword, setDebouncedKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>(undefined)
  const [roleFilter, setRoleFilter] = useState<number | undefined>(undefined)
  const [pagination, setPagination] = useState({ page: 1, pageSize: 10 })

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedKeyword(searchText.trim()), 300)
    return () => clearTimeout(timer)
  }, [searchText])

  const params = useMemo(
    () => ({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: debouncedKeyword || undefined,
      status: statusFilter,
      roleId: roleFilter,
    }),
    [debouncedKeyword, pagination.page, pagination.pageSize, roleFilter, statusFilter]
  )

  const { data, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ['users', params],
    queryFn: () => listUsers(params),
  })

  const { data: roleOptionsData = [] } = useQuery({
    queryKey: ['roles-all'],
    queryFn: async () => listAllRoles(),
  })

  const roleOptions = useMemo(
    () =>
      roleOptionsData.map((role: Role) => ({
        label: role.name,
        value: role.id,
      })),
    [roleOptionsData]
  )

  const createMutation = useMutation({
    mutationFn: (payload: UserCreateInput) =>
      createUser({
        username: payload.username,
        nickname: payload.nickname,
        email: payload.email,
        password: payload.password,
        roleIds: payload.roleIds,
        enabled: payload.enabled,
      }),
    onSuccess: () => {
      message.success('创建成功')
      setCreateVisible(false)
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
    onError: () => message.error('创建失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: UserEditInput }) =>
      updateUser(id, {
        nickname: payload.nickname,
        email: payload.email,
        enabled: payload.enabled,
        roleIds: payload.roleIds,
      }),
    onSuccess: () => {
      message.success('更新成功')
      setEditVisible(false)
      setCurrentUser(null)
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
    onError: () => message.error('更新失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteUser(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
    onError: () => message.error('删除失败'),
  })

  const resetMutation = useMutation({
    mutationFn: ({ id, password }: { id: number; password: string }) =>
      resetPassword(id, { password }),
    onSuccess: () => {
      message.success('密码重置成功')
      setResetVisible(false)
      setCurrentUser(null)
    },
    onError: () => message.error('密码重置失败'),
  })

  const handleToggleEnabled = (user: User, checked: boolean) => {
    updateMutation.mutate({
      id: user.id,
      payload: {
        id: user.id,
        nickname: user.nickname,
        email: user.email,
        enabled: checked,
        roleIds: getRoleIds(user),
      },
    })
  }

  const columns: ProColumns<User>[] = [
    {
      title: '用户',
      dataIndex: 'username',
      width: 260,
      render: (_, record) => (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4, minWidth: 0 }}>
          <span style={{ fontWeight: 500 }}>{record.nickname || record.username}</span>
          <span style={{ color: 'rgba(0, 0, 0, 0.45)', fontSize: 12 }}>
            {record.username}
            {record.email ? ` · ${record.email}` : ''}
          </span>
        </div>
      ),
    },
    {
      title: '角色',
      dataIndex: 'roles',
      width: 220,
      align: 'center',
      render: (_, record) => (
        <Space size={[6, 6]} wrap style={{ justifyContent: 'center' }}>
          {record.roles?.length ? (
            record.roles.map((role) => <Tag key={role.id}>{role.name}</Tag>)
          ) : (
            <span style={{ color: 'rgba(0, 0, 0, 0.25)' }}>未分配角色</span>
          )}
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 120,
      align: 'center',
      render: (_, record) => (
        <Switch
          size="small"
          checked={record.enabled}
          checkedChildren="启用"
          unCheckedChildren="禁用"
          onChange={(checked) => handleToggleEnabled(record, checked)}
          loading={updateMutation.isPending && currentUser?.id === record.id}
        />
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 180,
      align: 'center',
      render: (_, record) => formatDate(record.createdAt),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 132,
      align: 'center',
      render: (_, record) => (
        <Space size={4}>
          <Tooltip title="编辑">
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => {
                setCurrentUser(record)
                setEditVisible(true)
              }}
            />
          </Tooltip>
          <Tooltip title="重置密码">
            <Button
              type="text"
              size="small"
              icon={<KeyOutlined />}
              onClick={() => {
                setCurrentUser(record)
                setResetVisible(true)
              }}
            />
          </Tooltip>
          <Popconfirm
            title="确定删除该用户？"
            onConfirm={() => deleteMutation.mutate(record.id)}
          >
            <Tooltip title="删除">
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const hasFilters = Boolean(searchText || statusFilter || roleFilter)

  return (
    <AppPage>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <section
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: 12,
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Space size={12} wrap>
            <Input
              placeholder="搜索用户名、昵称或邮箱"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(event) => {
                setSearchText(event.target.value)
                setPagination((prev) => ({ ...prev, page: 1 }))
              }}
              style={{ width: 260 }}
              allowClear
            />
            <Select
              placeholder="账号状态"
              value={statusFilter}
              onChange={(value) => {
                setStatusFilter(value)
                setPagination((prev) => ({ ...prev, page: 1 }))
              }}
              style={{ width: 140 }}
              allowClear
              options={[
                { label: '启用', value: 'active' },
                { label: '禁用', value: 'disabled' },
              ]}
            />
            <Select
              placeholder="角色筛选"
              value={roleFilter}
              onChange={(value) => {
                setRoleFilter(value)
                setPagination((prev) => ({ ...prev, page: 1 }))
              }}
              style={{ width: 180 }}
              allowClear
              options={roleOptions}
            />
            <Button icon={<ReloadOutlined />} loading={isRefetching} onClick={() => refetch()}>
              刷新
            </Button>
            {hasFilters ? (
              <Button
                onClick={() => {
                  setSearchText('')
                  setStatusFilter(undefined)
                  setRoleFilter(undefined)
                  setPagination({ page: 1, pageSize: 10 })
                }}
              >
                重置
              </Button>
            ) : null}
          </Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
            创建用户
          </Button>
        </section>

        <ProTable<User>
          columns={columns}
          dataSource={data?.items || []}
          loading={isLoading}
          rowKey="id"
          search={false}
          options={false}
          cardBordered
          tableAlertRender={false}
          pagination={{
            current: pagination.page,
            pageSize: pagination.pageSize,
            total: data?.total || 0,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) => {
              setPagination({ page, pageSize: pageSize || 10 })
            },
          }}
          toolBarRender={false}
          scroll={{ x: 800 }}
        />
      </div>

      <ModalForm
        title="创建用户"
        open={createVisible}
        onOpenChange={setCreateVisible}
        onFinish={async (values) => {
          const result = userCreateSchema.safeParse(values)
          if (!result.success) {
            message.error(result.error.issues[0]?.message || '请检查表单输入')
            return false
          }
          createMutation.mutate(result.data)
          return true
        }}
        width={560}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText
          name="username"
          label="用户名"
          rules={[{ required: true }]}
          fieldProps={{ maxLength: 32 }}
        />
        <ProFormText name="nickname" label="昵称" fieldProps={{ maxLength: 80 }} />
        <ProFormText name="email" label="邮箱" fieldProps={{ maxLength: 120 }} />
        <ProFormSelect
          name="roleIds"
          label="角色"
          mode="multiple"
          rules={[{ required: true, message: '请选择角色' }]}
          options={roleOptions}
        />
        <ProFormSwitch name="enabled" label="启用状态" initialValue />
        <ProFormText.Password name="password" label="密码" rules={[{ required: true }]} />
        <ProFormText.Password
          name="confirmPassword"
          label="确认密码"
          rules={[{ required: true }]}
        />
      </ModalForm>

      <ModalForm
        title={`编辑用户${currentUser ? ` - ${currentUser.username}` : ''}`}
        open={editVisible}
        onOpenChange={(visible) => {
          setEditVisible(visible)
          if (!visible) {
            setCurrentUser(null)
          }
        }}
        onFinish={async (values) => {
          if (!currentUser) {
            return false
          }
          const payload: UserEditInput = {
            id: currentUser.id,
            nickname: values.nickname,
            email: values.email,
            enabled: values.enabled,
            roleIds: values.roleIds,
          }
          const result = userEditSchema.safeParse(payload)
          if (!result.success) {
            message.error(result.error.issues[0]?.message || '请检查表单输入')
            return false
          }
          updateMutation.mutate({ id: currentUser.id, payload: result.data })
          return true
        }}
        width={560}
        modalProps={{ destroyOnClose: true }}
        initialValues={
          currentUser
            ? {
                username: currentUser.username,
                nickname: currentUser.nickname,
                email: currentUser.email,
                enabled: currentUser.enabled,
                roleIds: getRoleIds(currentUser),
              }
            : undefined
        }
      >
        <ProFormText name="username" label="用户名" disabled />
        <ProFormText name="nickname" label="昵称" fieldProps={{ maxLength: 80 }} />
        <ProFormText name="email" label="邮箱" fieldProps={{ maxLength: 120 }} />
        <ProFormSelect
          name="roleIds"
          label="角色"
          mode="multiple"
          rules={[{ required: true, message: '请选择角色' }]}
          options={roleOptions}
        />
        <ProFormSwitch name="enabled" label="启用状态" />
      </ModalForm>

      <ModalForm
        title={`重置密码${currentUser ? ` - ${currentUser.username}` : ''}`}
        open={resetVisible}
        onOpenChange={(visible) => {
          setResetVisible(visible)
          if (!visible) {
            setCurrentUser(null)
          }
        }}
        onFinish={async (values) => {
          if (!currentUser) {
            return false
          }
          const result = resetPasswordSchema.safeParse(values)
          if (!result.success) {
            message.error(result.error.issues[0]?.message || '请检查表单输入')
            return false
          }
          resetMutation.mutate({
            id: currentUser.id,
            password: result.data.newPassword,
          })
          return true
        }}
        width={420}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText.Password
          name="newPassword"
          label="新密码"
          rules={[{ required: true, message: '请输入新密码' }]}
        />
        <ProFormText.Password
          name="confirmPassword"
          label="确认密码"
          rules={[{ required: true, message: '请再次输入密码' }]}
        />
      </ModalForm>
    </AppPage>
  )
}

export default UsersPage
