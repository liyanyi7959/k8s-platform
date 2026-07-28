import React from 'react'
import GenericResourceList, { rawColumn, rawField } from '@/features/kops/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Tag, Space } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const spec = (rawField(record, 'spec') as any) || {}
  const status = (rawField(record, 'status') as any) || {}
  const driver = spec.driver || '-'
  const ready = status.readyToUse
  const vsName = spec.volumeSnapshotRef?.name || '-'
  const vsNamespace = spec.volumeSnapshotRef?.namespace || '-'
  const volumeHandle = spec.source?.volumeHandle || '-'

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Driver"><Tag color="blue">{driver}</Tag></Descriptions.Item>
        <Descriptions.Item label="Ready">
          <Tag color={ready ? 'green' : 'orange'}>{ready ? '是' : '否'}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="Volume Handle">{volumeHandle}</Descriptions.Item>
        <Descriptions.Item label="VolumeSnapshot" span={2}>{vsNamespace}/{vsName}</Descriptions.Item>
      </Descriptions>
    </Space>
  )
}

export default function VolumeSnapshotContentsPage() {
  return (
    <GenericResourceList
      resource="volumesnapshotcontents"
      title="VolumeSnapshotContents"
      creatable={false}
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('Driver', 'spec.driver', { width: 200 }),
        rawColumn('Ready', 'status.readyToUse', {
          width: 80,
          align: 'center',
          render: (val) => <Tag color={val ? 'green' : 'orange'}>{val ? '是' : '否'}</Tag>,
        }),
      ]}
    />
  )
}
