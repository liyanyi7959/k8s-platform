import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/services/k8s'
import { Descriptions, Tag, Space } from 'antd'
import { formatDate } from '@/utils'

const renderDetail = (record: GenericResourceItem) => {
  const spec = (rawField(record, 'spec') as any) || {}
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="CSI Driver"><Tag color="blue">{spec.csiDriverName || '-'}</Tag></Descriptions.Item>
        <Descriptions.Item label="拓扑要求">{spec.requiresRepublish ? '是' : '否'}</Descriptions.Item>
        <Descriptions.Item label="存储容量">{spec.attachRequired ? '需要' : '不需要'}</Descriptions.Item>
        {record.createdAt && <Descriptions.Item label="创建时间" span={2}>{formatDate(record.createdAt)}</Descriptions.Item>}
      </Descriptions>
    </Space>
  )
}

export default function CSIDriversPage() {
  return (
    <GenericResourceList
      resource="csidrivers"
      title="CSIDrivers"
      creatable={false}
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('CSI Driver', 'spec.csiDriverName', { width: 200 }),
        rawColumn('拓扑', 'spec.requiresRepublish', {
          width: 80,
          align: 'center',
          render: (val) => (val ? '是' : '-'),
        }),
      ]}
    />
  )
}
