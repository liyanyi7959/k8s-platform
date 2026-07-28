import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Space, Table } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const rules = (rawField(record, 'rules') as any[]) || []
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="规则数">{rules.length}</Descriptions.Item>
      </Descriptions>
      <Table size="small" title={() => `规则 (${rules.length})`} rowKey={(_, i) => String(i)} pagination={false}
        dataSource={rules}
        columns={[
          { title: 'API Groups', dataIndex: 'apiGroups', render: (v: string[]) => v?.join(', ') || '-', ellipsis: true },
          { title: 'Resources', dataIndex: 'resources', render: (v: string[]) => v?.join(', ') || '-', ellipsis: true },
          { title: 'Verbs', dataIndex: 'verbs', render: (v: string[]) => v?.join(', ') || '-', ellipsis: true },
        ]}
      />
    </Space>
  )
}

export default function ClusterRolesPage() {
  return (
    <GenericResourceList
      resource="clusterroles"
      title="ClusterRoles"
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('规则数', 'rules', {
          width: 80,
          align: 'center',
          render: (val) => {
            const rules = val as Array<unknown>
            return rules?.length ?? 0
          },
        }),
      ]}
    />
  )
}
