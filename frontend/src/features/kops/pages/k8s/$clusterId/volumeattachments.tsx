import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Tag, Space } from 'antd'
import { formatDate } from '@/utils'

const renderDetail = (record: GenericResourceItem) => {
  const spec = (rawField(record, 'spec') as any) || {}
  const status = (rawField(record, 'status') as any) || {}
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="PV"><Tag color="blue">{spec.source?.persistentVolumeName || '-'}</Tag></Descriptions.Item>
        <Descriptions.Item label="Node">{spec.nodeName || '-'}</Descriptions.Item>
        <Descriptions.Item label="已挂载">
          <Tag color={status.attached ? 'green' : 'red'}>{status.attached ? '是' : '否'}</Tag>
        </Descriptions.Item>
        {status.attachError && (
          <Descriptions.Item label="挂载错误" span={2}>
            <Tag color="red">{typeof status.attachError.message === 'string' ? status.attachError.message : JSON.stringify(status.attachError)}</Tag>
          </Descriptions.Item>
        )}
        {record.createdAt && <Descriptions.Item label="创建时间" span={2}>{formatDate(record.createdAt)}</Descriptions.Item>}
      </Descriptions>
    </Space>
  )
}

export default function VolumeAttachmentsPage() {
  return (
    <GenericResourceList
      resource="volumeattachments"
      title="VolumeAttachments"
      creatable={false}
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('PV', 'spec.source.persistentVolumeName', { width: 200 }),
        rawColumn('Node', 'spec.nodeName', { width: 150 }),
        rawColumn('已挂载', 'status.attached', {
          width: 80,
          align: 'center',
          render: (val) => <Tag color={val ? 'green' : 'default'}>{val ? '是' : '否'}</Tag>,
        }),
      ]}
    />
  )
}
