import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'
import { Tag } from 'antd'

export default function CRDsPage() {
  return (
    <GenericResourceList
      resource="crds"
      title="CustomResourceDefinitions"
      extraColumns={[
        rawColumn('Group', 'spec.group', { width: 180 }),
        rawColumn('Scope', 'spec.scope', {
          width: 100,
          render: (val) => <Tag>{String(val)}</Tag>,
        }),
        rawColumn('版本', 'spec.versions', {
          width: 120,
          render: (val) => {
            const versions = val as Array<{ name: string; served?: boolean }>
            const served = versions?.filter((v) => v.served).map((v) => v.name)
            return served?.length ? served.join(', ') : '-'
          },
        }),
      ]}
    />
  )
}
