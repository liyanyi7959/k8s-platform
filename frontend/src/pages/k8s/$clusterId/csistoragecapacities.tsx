import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/services/k8s'
import { Descriptions, Tag, Space } from 'antd'
import { formatDate } from '@/utils'

const renderDetail = (record: GenericResourceItem) => {
  const sc = rawField(record, 'storageClassName') as string
  const capacity = (rawField(record, 'capacity') as any) || {}
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Namespace"><Tag>{record.namespace}</Tag></Descriptions.Item>
        <Descriptions.Item label="StorageClass"><Tag color="blue">{sc || '-'}</Tag></Descriptions.Item>
        <Descriptions.Item label="容量">{capacity.storage || '-'}</Descriptions.Item>
        {record.createdAt && <Descriptions.Item label="创建时间" span={2}>{formatDate(record.createdAt)}</Descriptions.Item>}
      </Descriptions>
    </Space>
  )
}

export default function CSIStorageCapacitiesPage() {
  return (
    <GenericResourceList
      resource="csistoragecapacities"
      title="CSIStorageCapacities"
      namespaced
      creatable={false}
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('StorageClass', 'storageClassName', { width: 180 }),
        rawColumn('容量', 'capacity.storage', { width: 120, align: 'center' }),
      ]}
    />
  )
}
