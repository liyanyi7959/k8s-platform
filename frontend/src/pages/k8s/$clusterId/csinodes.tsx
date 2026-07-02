import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function CSINodesPage() {
  return (
    <GenericResourceList
      resource="csinodes"
      title="CSINodes"
      creatable={false}
      extraColumns={[
        rawColumn('Node', 'spec.nodeID', { width: 200 }),
        rawColumn('Drivers', 'spec.drivers', {
          width: 100,
          render: (val) => {
            const drivers = val as Array<unknown>
            return drivers?.length || 0
          },
        }),
      ]}
    />
  )
}
