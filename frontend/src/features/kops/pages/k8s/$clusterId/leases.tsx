import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Space, Tag } from 'antd'
import { formatDate } from '@/utils'

const renderDetail = (record: GenericResourceItem) => {
  const spec = rawField(record, 'spec') as any || {}
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Namespace">{record.namespace || '-'}</Descriptions.Item>
        <Descriptions.Item label="Holder">{spec.holderIdentity || '-'}</Descriptions.Item>
        <Descriptions.Item label="租期(秒)">{spec.leaseDurationSeconds || '-'}</Descriptions.Item>
        <Descriptions.Item label="获取时间" span={2}>{spec.acquireTime || '-'}</Descriptions.Item>
        <Descriptions.Item label="续约时间" span={2}>{spec.renewTime || '-'}</Descriptions.Item>
        {record.createdAt && <Descriptions.Item label="创建时间" span={2}>{formatDate(record.createdAt)}</Descriptions.Item>}
      </Descriptions>
    </Space>
  )
}

export default function LeasesPage() {
  return (
    <GenericResourceList
      resource="leases"
      title="Leases"
      namespaced
      creatable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('Holder', 'spec.holderIdentity', { width: 200 }),
        rawColumn('租期(s)', 'spec.leaseDurationSeconds', { width: 90, align: 'center' }),
      ]}
    />
  )
}
