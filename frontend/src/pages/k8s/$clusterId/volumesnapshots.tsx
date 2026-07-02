import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'
import { Tag } from 'antd'

export default function VolumeSnapshotsPage() {
  return (
    <GenericResourceList
      resource="volumesnapshots"
      title="VolumeSnapshots"
      namespaced
      extraColumns={[
        rawColumn('快照类', 'spec.volumeSnapshotClassName', { width: 150 }),
        rawColumn('PVC', 'spec.source.persistentVolumeClaimName', { width: 180 }),
        rawColumn('状态', 'status.readyToUse', {
          width: 80,
          render: (val) => <Tag color={val ? 'green' : 'orange'}>{val ? '就绪' : '处理中'}</Tag>,
        }),
      ]}
    />
  )
}
