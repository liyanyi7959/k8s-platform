import React, { useMemo, useState } from 'react'
import { history, useLocation } from '@umijs/max'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Popconfirm, Space, Tag, message } from 'antd'
import { CodeOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { deleteGenericResource, listGenericResources, listPermissionAudits } from '@/services/k8s'
import { formatDate } from '@/utils'

const resourceNameMap: Record<string, string> = {
  'log-workbench': '日志工作台',
  'manifest-apply': 'YAML 部署',
  podmetrics: 'PodMetrics',
  endpoints: 'Endpoints',
  endpointslices: 'EndpointSlices',
  'network-policies': 'NetworkPolicies',
  'ingress-classes': 'IngressClasses',
  'volume-snapshots': 'VolumeSnapshots',
  'volume-snapshot-classes': 'VolumeSnapshotClasses',
  'volume-snapshot-contents': 'VolumeSnapshotContents',
  'csi-drivers': 'CSIDrivers',
  'csi-nodes': 'CSINodes',
  'csi-storage-capacities': 'CSIStorageCapacities',
  'volume-attachments': 'VolumeAttachments',
  'limit-ranges': 'LimitRanges',
  'cluster-roles': 'ClusterRoles',
  'cluster-role-bindings': 'ClusterRoleBindings',
  leases: 'Leases',
  events: 'Events',
  crds: 'CRDs',
  'api-services': 'APIServices',
  'priority-classes': 'PriorityClasses',
  'runtime-classes': 'RuntimeClasses',
  'validating-webhooks': 'ValidatingWebhooks',
  'mutating-webhooks': 'MutatingWebhooks',
  'validating-admission-policies': 'ValidatingAdmissionPolicies',
  'validating-admission-policy-bindings': 'ValidatingAdmissionPolicyBindings',
  'permission-audits': '权限分析',
}

const namespacedResources = new Set([
  'podmetrics',
  'endpoints',
  'endpointslices',
  'volume-snapshots',
  'csi-storage-capacities',
  'limit-ranges',
  'leases',
])

const deletableResources = new Set([
  'endpoints',
  'endpointslices',
  'ingress-classes',
  'volume-snapshots',
  'volume-snapshot-classes',
  'volume-snapshot-contents',
  'csi-drivers',
  'csi-nodes',
  'csi-storage-capacities',
  'volume-attachments',
  'limit-ranges',
  'cluster-roles',
  'cluster-role-bindings',
  'leases',
  'crds',
  'api-services',
  'priority-classes',
  'runtime-classes',
  'validating-webhooks',
  'mutating-webhooks',
  'validating-admission-policies',
  'validating-admission-policy-bindings',
])

type AuditRow = {
  id: number
  display_name?: string
  cluster_name?: string
  source_type?: string
  status?: string
  summary?: Record<string, any>
  created_at?: string
}

const K8sResourcePlaceholder: React.FC = () => {
  const location = useLocation()
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const resourceKey = location.pathname.split('/').filter(Boolean).pop() || ''
  const resourceName = resourceNameMap[resourceKey] || resourceKey
  const [namespace, setNamespace] = useState<string>('')
  const yamlDrawer = useYamlDrawer()

  const isPermissionAudit = resourceKey === 'permission-audits'
  const canFilterNamespace = namespacedResources.has(resourceKey)
  const canDelete = deletableResources.has(resourceKey)

  const { data, isLoading } = useQuery({
    queryKey: ['generic-resource-page', clusterId, resourceKey, namespace],
    queryFn: () => listGenericResources(clusterId, resourceKey, canFilterNamespace ? namespace || undefined : undefined),
    enabled: !!clusterId && !isPermissionAudit,
    refetchInterval: 30_000,
  })

  const { data: audits, isLoading: auditsLoading } = useQuery({
    queryKey: ['permission-audits', clusterId],
    queryFn: () => listPermissionAudits(clusterId),
    enabled: !!clusterId && isPermissionAudit,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: { name: string; namespace?: string }) =>
      deleteGenericResource(clusterId, resourceKey, record.name, record.namespace),
    onSuccess: () => {
      message.success('资源已删除')
      queryClient.invalidateQueries({ queryKey: ['generic-resource-page', clusterId, resourceKey] })
    },
  })

  const genericColumns: ProColumns<any>[] = useMemo(() => [
    {
      title: '名称',
      dataIndex: 'name',
      ellipsis: true,
      render: (_: unknown, record: any) => <span style={{ fontWeight: 600 }}>{record.name || '-'}</span>,
    },
    canFilterNamespace ? {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_: unknown, record: any) => <EllipsisText text={record.namespace} tag />,
    } : undefined,
    {
      title: '类型',
      dataIndex: 'kind',
      width: 180,
      render: (_: unknown, record: any) => record.kind || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 140,
      render: (_: unknown, record: any) => record.status || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 180,
      render: (_: unknown, record: any) => (record.createdAt ? formatDate(record.createdAt) : '-'),
    },
    {
      title: '操作',
      valueType: 'option',
      width: canDelete ? 120 : 70,
      render: (_: unknown, record: any) => (
        <Space>
          <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace)}>
            <CodeOutlined />
          </a>
          {canDelete ? (
            <Popconfirm title="确定删除该资源？" onConfirm={() => deleteMutation.mutate(record)}>
              <a style={{ color: '#ff4d4f' }}>
                <DeleteOutlined />
              </a>
            </Popconfirm>
          ) : null}
        </Space>
      ),
    },
  ].filter(Boolean) as ProColumns<any>[], [canDelete, canFilterNamespace, deleteMutation, yamlDrawer])

  const auditColumns: ProColumns<AuditRow>[] = [
    { title: 'ID', dataIndex: 'id', width: 90 },
    { title: '名称', dataIndex: 'display_name', ellipsis: true },
    { title: '集群', dataIndex: 'cluster_name', width: 160 },
    {
      title: '来源',
      dataIndex: 'source_type',
      width: 120,
      render: (_, record) => <Tag>{record.source_type || '-'}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      render: (_, record) => {
        const color = record.status === 'success' ? 'green' : record.status === 'failed' ? 'red' : 'blue'
        return <Tag color={color}>{record.status || '-'}</Tag>
      },
    },
    {
      title: '摘要',
      dataIndex: 'summary',
      render: (_, record) => {
        const summary = record.summary || {}
        const parts = Object.entries(summary)
          .slice(0, 3)
          .map(([key, value]) => `${key}: ${String(value)}`)
        return parts.length > 0 ? parts.join(' | ') : '-'
      },
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 180,
      render: (_, record) => (record.created_at ? formatDate(record.created_at) : '-'),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 110,
      render: (_, record) => (
        <a onClick={() => history.push(`/permission-audits/${record.id}`)}>查看详情</a>
      ),
    },
  ]

  return (
    <AppPage>
      <ProTable<any>
        headerTitle={resourceName}
        columns={isPermissionAudit ? (auditColumns as ProColumns<any>[]) : genericColumns}
        dataSource={isPermissionAudit ? (audits?.items || []) : (data?.items || [])}
        loading={isPermissionAudit ? auditsLoading : isLoading}
        rowKey={(record) => record.id || `${record.namespace || ''}/${record.name}`}
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        toolBarRender={() => [
          canFilterNamespace ? (
            <NamespaceSelector
              key="namespace"
              clusterId={clusterId}
              value={namespace}
              onChange={setNamespace}
              style={{ width: 220 }}
            />
          ) : null,
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() => {
              if (isPermissionAudit) {
                queryClient.invalidateQueries({ queryKey: ['permission-audits', clusterId] })
              } else {
                queryClient.invalidateQueries({ queryKey: ['generic-resource-page', clusterId, resourceKey] })
              }
            }}
          >
            刷新
          </Button>,
        ].filter(Boolean)}
      />

      <Alert
        style={{ marginTop: 16 }}
        type={isPermissionAudit ? 'info' : 'warning'}
        showIcon
        message={isPermissionAudit ? '权限分析已接入真实数据' : '当前页面使用通用资源浏览模式'}
        description={
          isPermissionAudit
            ? '这里先展示当前集群的权限分析任务列表。更深的创建、比对和日志能力仍走专门的权限分析详情链路。'
            : `资源标识：${resourceKey}。当前已接入列表、YAML 查看和部分资源删除能力；如需结构化创建，优先使用 YAML 部署页。`
        }
      />

      {!isPermissionAudit ? (
        <YamlDrawer
          clusterId={clusterId}
          resourceType={resourceKey}
          name={yamlDrawer.name}
          namespace={yamlDrawer.namespace}
          open={yamlDrawer.open}
          onClose={yamlDrawer.closeYaml}
        />
      ) : null}
    </AppPage>
  )
}

export default K8sResourcePlaceholder