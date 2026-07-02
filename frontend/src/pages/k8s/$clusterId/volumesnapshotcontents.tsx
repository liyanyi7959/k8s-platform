import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'
import { Tag } from 'antd'

export default function VolumeSnapshotContentsPage() {
  return (
    <GenericResourceList
      resource="volumesnapshotcontents"
      title="VolumeSnapshotContents"
      creatable={false}
      extraColumns={[
        rawColumn('Driver', 'spec.driver', { width: 200 }),
        rawColumn('Ready', 'status.readyToUse', {
          width: 80,
          render: (val) => <Tag color={val ? 'green' : 'orange'}>{val ? '是' : '否'}</Tag>,
        }),
      ]}
    />
  )
}
