import GenericResourceList, { rawColumn, rawField } from '@/features/kops/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Table, Tag, Space } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const driver = rawField(record, 'driver') as string
  const deletionPolicy = rawField(record, 'deletionPolicy') as string
  const parameters = (rawField(record, 'parameters') as Record<string, string>) || {}

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Driver"><Tag color="blue">{driver || '-'}</Tag></Descriptions.Item>
        <Descriptions.Item label="Deletion Policy">
          <Tag color={deletionPolicy === 'Delete' ? 'red' : 'blue'}>{deletionPolicy || '-'}</Tag>
        </Descriptions.Item>
      </Descriptions>

      {Object.keys(parameters).length > 0 && (
        <Table
          size="small"
          title={() => `参数 (${Object.keys(parameters).length})`}
          rowKey="key"
          pagination={false}
          dataSource={Object.entries(parameters).map(([k, v]) => ({ key: k, value: v }))}
          columns={[
            { title: '键', dataIndex: 'key', width: 200, ellipsis: true },
            { title: '值', dataIndex: 'value', ellipsis: true },
          ]}
        />
      )}
    </Space>
  )
}

export default function VolumeSnapshotClassesPage() {
  return (
    <GenericResourceList
      resource="volumesnapshotclasses"
      title="VolumeSnapshotClasses"
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('Driver', 'driver', { width: 200 }),
        rawColumn('Deletion', 'deletionPolicy', { width: 120, align: 'center' }),
      ]}
    />
  )
}
