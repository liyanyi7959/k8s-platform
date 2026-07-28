import GenericResourceList, { rawColumn, rawField } from '@/features/kops/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Tag, Space } from 'antd'
import { formatDate } from '@/utils'

const renderDetail = (record: GenericResourceItem) => {
  const spec = rawField(record, 'spec') as any || {}
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Handler"><Tag color="blue">{spec.handler || '-'}</Tag></Descriptions.Item>
        {spec.overhead && <Descriptions.Item label="Overhead" span={2}>CPU: {spec.overhead?.pod?.cpu || '-'}, Memory: {spec.overhead?.pod?.memory || '-'}</Descriptions.Item>}
        {spec.scheduling && <Descriptions.Item label="Scheduling" span={2}>{JSON.stringify(spec.scheduling)}</Descriptions.Item>}
        {record.createdAt && <Descriptions.Item label="创建时间" span={2}>{formatDate(record.createdAt)}</Descriptions.Item>}
      </Descriptions>
    </Space>
  )
}

export default function RuntimeClassesPage() {
  return (
    <GenericResourceList
      resource="runtimeclasses"
      title="RuntimeClasses"
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('Handler', 'spec.handler', { width: 150 }),
      ]}
    />
  )
}
