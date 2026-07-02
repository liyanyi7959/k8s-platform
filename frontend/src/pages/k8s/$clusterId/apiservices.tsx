import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'
import { Tag } from 'antd'

export default function APIServicesPage() {
  return (
    <GenericResourceList
      resource="apiservices"
      title="APIServices"
      extraColumns={[
        rawColumn('Group', 'spec.group', { width: 150 }),
        rawColumn('Version', 'spec.version', { width: 100 }),
        rawColumn('可用', 'status.conditions', {
          width: 80,
          render: (val) => {
            const conditions = val as Array<{ type: string; status: string }>
            const avail = conditions?.find((c) => c.type === 'Available')
            return avail ? <Tag color={avail.status === 'True' ? 'green' : 'red'}>{avail.status === 'True' ? '是' : '否'}</Tag> : '-'
          },
        }),
      ]}
    />
  )
}
