import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/services/k8s'
import { Descriptions, Space, Table, Tag } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const roleRef = rawField(record, 'roleRef') as any || {}
  const subjects = (rawField(record, 'subjects') as any[]) || []
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="RoleRef"><Tag color="blue">{roleRef.kind}/{roleRef.name}</Tag></Descriptions.Item>
      </Descriptions>
      {subjects.length > 0 && (
        <Table size="small" title={() => `Subjects (${subjects.length})`} rowKey={(_, i) => String(i)} pagination={false}
          dataSource={subjects}
          columns={[
            { title: 'Kind', dataIndex: 'kind', width: 100 },
            { title: 'Name', dataIndex: 'name', ellipsis: true },
            { title: 'APIGroup', dataIndex: 'apiGroup', width: 120 },
          ]}
        />
      )}
    </Space>
  )
}

export default function ClusterRoleBindingsPage() {
  return (
    <GenericResourceList
      resource="clusterrolebindings"
      title="ClusterRoleBindings"
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('Role Ref', 'roleRef.name', { width: 200 }),
        rawColumn('Subjects', 'subjects', {
          width: 100,
          align: 'center',
          render: (val) => {
            const subjects = val as Array<unknown>
            return subjects?.length ?? 0
          },
        }),
      ]}
    />
  )
}
