import React from 'react'
import GenericResourceList, { rawColumn, rawField } from '@/features/kops/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Tag, Space } from 'antd'
import { formatDate } from '@/utils'

const renderDetail = (record: GenericResourceItem) => {
  const spec = (rawField(record, 'spec') as any) || {}
  const status = (rawField(record, 'status') as any) || {}
  const vscName = spec.volumeSnapshotClassName || '-'
  const pvcName = spec.source?.persistentVolumeClaimName || '-'
  const ready = status.readyToUse
  const size = status.restoreSize ? `${status.restoreSize} bytes` : '-'
  const createdAt = record.createdAt

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Namespace"><Tag>{record.namespace}</Tag></Descriptions.Item>
        <Descriptions.Item label="快照类">{vscName}</Descriptions.Item>
        <Descriptions.Item label="PVC 来源">{pvcName}</Descriptions.Item>
        <Descriptions.Item label="状态">
          <Tag color={ready ? 'green' : 'orange'}>{ready ? '就绪' : '处理中'}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="大小">{size}</Descriptions.Item>
        {createdAt && <Descriptions.Item label="创建时间" span={2}>{formatDate(createdAt)}</Descriptions.Item>}
      </Descriptions>
    </Space>
  )
}

export default function VolumeSnapshotsPage() {
  return (
    <GenericResourceList
      resource="volumesnapshots"
      title="VolumeSnapshots"
      namespaced
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('快照类', 'spec.volumeSnapshotClassName', { width: 150 }),
        rawColumn('PVC', 'spec.source.persistentVolumeClaimName', { width: 180 }),
        rawColumn('状态', 'status.readyToUse', {
          width: 80,
          align: 'center',
          render: (val) => <Tag color={val ? 'green' : 'orange'}>{val ? '就绪' : '处理中'}</Tag>,
        }),
      ]}
    />
  )
}
