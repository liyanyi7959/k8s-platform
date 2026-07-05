import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/services/k8s'
import { Descriptions, Tag, Space, Table } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const limits = (rawField(record, 'spec.limits') as any[]) || []
  const limitData = limits.map((l, i) => ({
    key: i,
    type: l.type || '-',
    max: l.max ? Object.entries(l.max).map(([k, v]) => `${k}=${v}`).join(', ') : '-',
    min: l.min ? Object.entries(l.min).map(([k, v]) => `${k}=${v}`).join(', ') : '-',
    default: l.default ? Object.entries(l.default).map(([k, v]) => `${k}=${v}`).join(', ') : '-',
    defaultRequest: l.defaultRequest ? Object.entries(l.defaultRequest).map(([k, v]) => `${k}=${v}`).join(', ') : '-',
  }))
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Namespace"><Tag>{record.namespace}</Tag></Descriptions.Item>
      </Descriptions>
      <Table size="small" title={() => `限制项 (${limitData.length})`} rowKey="key" pagination={false}
        dataSource={limitData}
        columns={[
          { title: '类型', dataIndex: 'type', width: 100 },
          { title: 'Max', dataIndex: 'max', ellipsis: true },
          { title: 'Min', dataIndex: 'min', ellipsis: true },
          { title: 'Default', dataIndex: 'default', ellipsis: true },
          { title: 'DefaultRequest', dataIndex: 'defaultRequest', ellipsis: true },
        ]}
      />
    </Space>
  )
}

export default function LimitRangesPage() {
  return (
    <GenericResourceList
      resource="limitranges"
      title="LimitRanges"
      namespaced
      creatable={false}
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('限制项数', 'spec.limits', {
          width: 90,
          align: 'center',
          render: (val) => {
            const limits = val as Array<unknown>
            return limits?.length || 0
          },
        }),
      ]}
    />
  )
}
