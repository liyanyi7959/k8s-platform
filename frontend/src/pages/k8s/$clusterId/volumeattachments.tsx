import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'
import { Tag } from 'antd'

export default function VolumeAttachmentsPage() {
  return (
    <GenericResourceList
      resource="volumeattachments"
      title="VolumeAttachments"
      creatable={false}
      extraColumns={[
        rawColumn('PV', 'spec.source.persistentVolumeName', { width: 200 }),
        rawColumn('Node', 'spec.nodeName', { width: 150 }),
        rawColumn('已挂载', 'status.attached', {
          width: 80,
          render: (val) => <Tag color={val ? 'green' : 'default'}>{val ? '是' : '否'}</Tag>,
        }),
      ]}
    />
  )
}
