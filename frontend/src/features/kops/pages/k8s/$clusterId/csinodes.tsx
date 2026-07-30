import GenericResourceList, { rawColumn, rawField } from '@/features/kops/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Space, Table } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const spec = (rawField(record, 'spec') as any) || {}
  const drivers = (spec.drivers || []) as any[]
  const driverData = drivers.map((d, i) => ({
    key: i,
    name: d.name || '-',
    nodeID: d.nodeID || '-',
    allocatable: d.allocatable ? Object.entries(d.allocatable).map(([k, v]) => `${k}=${v}`).join(', ') : '-',
    capacity: d.capacity ? Object.entries(d.capacity).map(([k, v]) => `${k}=${v}`).join(', ') : '-',
  }))
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="驱动数">{drivers.length}</Descriptions.Item>
      </Descriptions>
      <Table size="small" title={() => `CSI 驱动 (${driverData.length})`} rowKey="key" pagination={false}
        dataSource={driverData}
        columns={[
          { title: '驱动名称', dataIndex: 'name', ellipsis: true },
          { title: 'Node ID', dataIndex: 'nodeID', ellipsis: true },
          { title: '可分配', dataIndex: 'allocatable', ellipsis: true },
          { title: '容量', dataIndex: 'capacity', ellipsis: true },
        ]}
      />
    </Space>
  )
}

export default function CSINodesPage() {
  return (
    <GenericResourceList
      resource="csinodes"
      title="CSINodes"
      creatable={false}
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('Node', 'metadata.name', { width: 200 }),
        rawColumn('Drivers', 'spec.drivers', {
          width: 100,
          align: 'center',
          render: (val) => {
            const drivers = val as Array<unknown>
            return drivers?.length || 0
          },
        }),
      ]}
    />
  )
}
