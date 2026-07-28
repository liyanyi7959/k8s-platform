import GenericResourceList, { rawColumn, rawField } from '@/features/kops/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Space, Table, Tag } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const spec = rawField(record, 'spec') as any || {}
  const versions = (spec.versions || []) as any[]
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Group"><Tag color="blue">{spec.group || '-'}</Tag></Descriptions.Item>
        <Descriptions.Item label="Scope"><Tag>{spec.scope || '-'}</Tag></Descriptions.Item>
      </Descriptions>
      <Table size="small" title={() => `版本 (${versions.length})`} rowKey="name" pagination={false}
        dataSource={versions}
        columns={[
          { title: '名称', dataIndex: 'name', width: 100 },
          { title: 'Served', dataIndex: 'served', width: 80, align: 'center' as const, render: (v) => <Tag color={v ? 'green' : 'default'}>{v ? '是' : '否'}</Tag> },
          { title: 'Storage', dataIndex: 'storage', width: 80, align: 'center' as const, render: (v) => <Tag color={v ? 'blue' : 'default'}>{v ? '是' : '否'}</Tag> },
        ]}
      />
    </Space>
  )
}

export default function CRDsPage() {
  return (
    <GenericResourceList
      resource="crds"
      title="CustomResourceDefinitions"
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('Group', 'spec.group', { width: 180 }),
        rawColumn('Scope', 'spec.scope', {
          width: 100,
          align: 'center',
          render: (val) => <Tag>{String(val)}</Tag>,
        }),
        rawColumn('版本', 'spec.versions', {
          width: 120,
          align: 'center',
          render: (val) => {
            const versions = val as Array<{ name: string; served?: boolean }>
            const served = versions?.filter((v) => v.served).map((v) => v.name)
            return served?.length ? served.join(', ') : '-'
          },
        }),
      ]}
    />
  )
}
