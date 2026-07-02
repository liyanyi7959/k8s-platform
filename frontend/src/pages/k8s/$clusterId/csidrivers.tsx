import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function CSIDriversPage() {
  return (
    <GenericResourceList
      resource="csidrivers"
      title="CSIDrivers"
      creatable={false}
      extraColumns={[
        rawColumn('CSI Driver', 'spec.csiDriverName', { width: 200 }),
        rawColumn('拓扑', 'spec.requiresRepublish', {
          width: 80,
          render: (val) => (val ? '是' : '-'),
        }),
      ]}
    />
  )
}
