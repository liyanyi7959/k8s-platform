import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Space, Table, Tag } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const spec = rawField(record, 'spec') as any || {}
  const status = rawField(record, 'status') as any || {}
  const conditions = (status.conditions || []) as any[]
  const avail = conditions.find(c => c.type === 'Available')
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Group"><Tag color="blue">{spec.group || '-'}</Tag></Descriptions.Item>
        <Descriptions.Item label="Version">{spec.version || '-'}</Descriptions.Item>
        <Descriptions.Item label="可用">
          <Tag color={avail?.status === 'True' ? 'green' : 'red'}>{avail?.status === 'True' ? '是' : '否'}</Tag>
        </Descriptions.Item>
      </Descriptions>
      {conditions.length > 0 && (
        <Table size="small" title={() => `条件 (${conditions.length})`} rowKey={(_, i) => String(i)} pagination={false}
          dataSource={conditions}
          columns={[
            { title: '类型', dataIndex: 'type', width: 120 },
            { title: '状态', dataIndex: 'status', width: 80, align: 'center' as const, render: (v) => <Tag color={v === 'True' ? 'green' : 'red'}>{v}</Tag> },
            { title: '原因', dataIndex: 'reason', ellipsis: true },
            { title: '消息', dataIndex: 'message', ellipsis: true },
          ]}
        />
      )}
    </Space>
  )
}

export default function APIServicesPage() {
  return (
    <GenericResourceList
      resource="apiservices"
      title="APIServices"
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('Group', 'spec.group', { width: 150 }),
        rawColumn('Version', 'spec.version', { width: 100 }),
        rawColumn('可用', 'status.conditions', {
          width: 80,
          align: 'center',
          render: (val) => {
            const conditions = val as Array<{ type: string; status: string }>
            const avail = conditions?.find((c) => c.type === 'Available')
            return avail ? <Tag color={avail.status === 'True' ? 'green' : 'red'}>{avail.status === 'True' ? '是' : '否'}</Tag> : '-'
          },
        }),
      ]}
    />
  )
}
