import React, { useMemo, useState } from 'react'
import {
  ModalForm,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
  ProTable,
  type ProColumns,
} from '@ant-design/pro-components'
import { Button, Input, message, Popconfirm, Space, Tooltip } from 'antd'
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { createRole, deleteRole, listPermissions, listRoles, updateRole } from '@/services/system'
import { roleCreateSchema, roleEditSchema } from '@/schemas/system'
import { formatDate } from '@/utils'
import type { Permission, Role } from '@/types'
import type { RoleCreateInput, RoleEditInput } from '@/schemas/system'

/** 角色管理页 */
const RolesPage: React.FC = () => {
  const queryClient = useQueryClient()
  const [createVisible, setCreateVisible] = useState(false)
  const [editVisible, setEditVisible] = useState(false)
  const [currentRole, setCurrentRole] = useState<Role | null>(null)
  const [searchText, setSearchText] = useState('')

  const { data, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ['roles'],
    queryFn: () => listRoles(),
  })

  const { data: permissionsData = [] } = useQuery({
    queryKey: ['permissions-all'],
    queryFn: () => listPermissions(),
  })

  const roles = useMemo(() => {
    const keyword = searchText.trim().toLowerCase()
    return (data?.items || []).filter((role) => {
      if (!keyword) {
        return true
      }
      return (
        role.name.toLowerCase().includes(keyword) ||
        role.code.toLowerCase().includes(keyword) ||
        role.description?.toLowerCase().includes(keyword)
      )
    })
  }, [data?.items, searchText])

  const permissionOptions = useMemo(() => {
    const groups = new Map<string, Permission[]>()
    permissionsData.forEach((perm) => {
      const label = perm.categoryLabel || '其他'
      if (!groups.has(label)) {
        groups.set(label, [])
      }
      groups.get(label)?.push(perm)
    })
    return Array.from(groups.entries()).map(([label, items]) => ({
      label,
      options: items.map((perm) => ({
        label: `${perm.name} (${perm.code})`,
        value: perm.code,
      })),
    }))
  }, [permissionsData])

  const createMutation = useMutation({
    mutationFn: (payload: RoleCreateInput) =>
      createRole({
        name: payload.name,
        code: payload.code,
        description: payload.description,
        permissions: payload.permissions,
      }),
    onSuccess: () => {
      message.success('创建成功')
      setCreateVisible(false)
      queryClient.invalidateQueries({ queryKey: ['roles'] })
    },
    onError: () => message.error('创建失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: RoleEditInput }) =>
      updateRole(id, {
        name: payload.name,
        description: payload.description,
        permissions: payload.permissions,
      }),
    onSuccess: () => {
      message.success('更新成功')
      setEditVisible(false)
      setCurrentRole(null)
      queryClient.invalidateQueries({ queryKey: ['roles'] })
    },
    onError: () => message.error('更新失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteRole(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['roles'] })
    },
    onError: () => message.error('删除失败'),
  })

  const columns: ProColumns<Role>[] = [
    {
      title: '角色',
      dataIndex: 'name',
      width: 260,
      render: (_, record) => (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4, minWidth: 0 }}>
          <span style={{ fontWeight: 500 }}>{record.name}</span>
          <span style={{ color: 'rgba(0, 0, 0, 0.45)', fontSize: 12 }}>{record.code}</span>
        </div>
      ),
    },
    {
      title: '说明',
      dataIndex: 'description',
      ellipsis: true,
      render: (_, record) => record.description || '—',
    },
    {
      title: '权限数量',
      dataIndex: 'permissions',
      width: 120,
      align: 'center',
      render: (_, record) => record.permissions?.length || 0,
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
      width: 96,
      align: 'center',
      render: (_, record) => (
        <Space size={4}>
          <Tooltip title="编辑">
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => {
                setCurrentRole(record)
                setEditVisible(true)
              }}
            />
          </Tooltip>
          <Popconfirm title="确定删除该角色？" onConfirm={() => deleteMutation.mutate(record.id)}>
            <Tooltip title="删除">
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

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
              placeholder="搜索角色名称、编码或描述"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(event) => setSearchText(event.target.value)}
              style={{ width: 280 }}
              allowClear
            />
            <Button icon={<ReloadOutlined />} loading={isRefetching} onClick={() => refetch()}>
              刷新
            </Button>
            {searchText ? <Button onClick={() => setSearchText('')}>重置</Button> : null}
          </Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
            创建角色
          </Button>
        </section>

        <ProTable<Role>
          columns={columns}
          dataSource={roles}
          loading={isLoading}
          rowKey="id"
          search={false}
          options={false}
          cardBordered
          tableAlertRender={false}
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
          toolBarRender={false}
          scroll={{ x: 700 }}
        />
      </div>

      <ModalForm
        title="创建角色"
        open={createVisible}
        onOpenChange={setCreateVisible}
        onFinish={async (values) => {
          const result = roleCreateSchema.safeParse(values)
          if (!result.success) {
            message.error(result.error.issues[0]?.message || '请检查表单输入')
            return false
          }
          createMutation.mutate(result.data)
          return true
        }}
        width={600}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText
          name="name"
          label="角色名称"
          rules={[{ required: true }]}
          fieldProps={{ maxLength: 64 }}
        />
        <ProFormText
          name="code"
          label="角色编码"
          rules={[{ required: true }]}
          fieldProps={{ maxLength: 32 }}
        />
        <ProFormTextArea
          name="description"
          label="描述"
          fieldProps={{ rows: 3, maxLength: 200 }}
        />
        <ProFormSelect
          name="permissions"
          label="权限"
          mode="multiple"
          options={permissionOptions}
          fieldProps={{ placeholder: '请选择权限' }}
        />
      </ModalForm>

      <ModalForm
        title={`编辑角色${currentRole ? ` - ${currentRole.name}` : ''}`}
        open={editVisible}
        onOpenChange={(visible) => {
          setEditVisible(visible)
          if (!visible) {
            setCurrentRole(null)
          }
        }}
        onFinish={async (values) => {
          if (!currentRole) {
            return false
          }
          const payload: RoleEditInput = {
            id: currentRole.id,
            name: values.name,
            description: values.description,
            permissions: values.permissions,
          }
          const result = roleEditSchema.safeParse(payload)
          if (!result.success) {
            message.error(result.error.issues[0]?.message || '请检查表单输入')
            return false
          }
          updateMutation.mutate({ id: currentRole.id, payload: result.data })
          return true
        }}
        width={600}
        modalProps={{ destroyOnClose: true }}
        initialValues={
          currentRole
            ? {
                name: currentRole.name,
                code: currentRole.code,
                description: currentRole.description,
                permissions: currentRole.permissions || [],
              }
            : undefined
        }
      >
        <ProFormText
          name="name"
          label="角色名称"
          rules={[{ required: true }]}
          fieldProps={{ maxLength: 64 }}
        />
        <ProFormText name="code" label="角色编码" disabled fieldProps={{ maxLength: 32 }} />
        <ProFormTextArea
          name="description"
          label="描述"
          fieldProps={{ rows: 3, maxLength: 200 }}
        />
        <ProFormSelect
          name="permissions"
          label="权限"
          mode="multiple"
          options={permissionOptions}
          fieldProps={{ placeholder: '请选择权限' }}
        />
      </ModalForm>
    </AppPage>
  )
}

export default RolesPage
