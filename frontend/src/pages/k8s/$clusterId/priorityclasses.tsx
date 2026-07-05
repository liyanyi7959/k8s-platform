import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/services/k8s'
import { Descriptions, Space, Tag } from 'antd'
import { formatDate } from '@/utils'

const renderDetail = (record: GenericResourceItem) => {
  const spec = rawField(record, 'spec') as any || {}
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="优先级值"><Tag color="blue">{spec.value ?? '-'}</Tag></Descriptions.Item>
        <Descriptions.Item label="全局默认">{spec.globalDefault ? <Tag color="green">是</Tag> : '否'}</Descriptions.Item>
        <Descriptions.Item label="Preemption">{spec.preemptionPolicy || '-'}</Descriptions.Item>
        <Descriptions.Item label="描述" span={2}>{spec.description || '-'}</Descriptions.Item>
        {record.createdAt && <Descriptions.Item label="创建时间" span={2}>{formatDate(record.createdAt)}</Descriptions.Item>}
      </Descriptions>
    </Space>
  )
}

export default function PriorityClassesPage() {
  return (
    <GenericResourceList
      resource="priorityclasses"
      title="PriorityClasses"
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('优先级值', 'spec.value', { width: 120, align: 'center' }),
        rawColumn('全局默认', 'spec.globalDefault', {
          width: 100,
          align: 'center',
          render: (val) => (val ? '是' : '-'),
        }),
        rawColumn('描述', 'spec.description'),
      ]}
    />
  )
}
