import GenericResourceList, { rawColumn, rawField } from '@/features/kops/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Space, Table } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const webhooks = (rawField(record, 'webhooks') as any[]) || []
  const whData = webhooks.map((wh, i) => ({
    key: i,
    name: wh.name || '-',
    service: wh.clientConfig?.service ? `${wh.clientConfig.service.name}:${wh.clientConfig.service.port || 443}` : '-',
    operations: (wh.rules?.flatMap((r: any) => r.operations || []) || []).join(', ') || '-',
    resources: (wh.rules?.flatMap((r: any) => r.resources || []) || []).join(', ') || '-',
  }))
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Webhook 数">{webhooks.length}</Descriptions.Item>
      </Descriptions>
      <Table size="small" title={() => `Webhooks (${webhooks.length})`} rowKey="key" pagination={false}
        dataSource={whData}
        columns={[
          { title: '名称', dataIndex: 'name', width: 200, ellipsis: true },
          { title: 'Service', dataIndex: 'service', ellipsis: true },
          { title: 'Operations', dataIndex: 'operations', ellipsis: true },
          { title: 'Resources', dataIndex: 'resources', ellipsis: true },
        ]}
      />
    </Space>
  )
}

export default function ValidatingWebhooksPage() {
  return (
    <GenericResourceList
      resource="validatingwebhooks"
      title="ValidatingWebhooks"
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('Webhooks', 'webhooks', {
          width: 90,
          align: 'center',
          render: (val) => {
            const webhooks = val as Array<unknown>
            return webhooks?.length ?? 0
          },
        }),
      ]}
    />
  )
}
