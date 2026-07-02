import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function LeasesPage() {
  return (
    <GenericResourceList
      resource="leases"
      title="Leases"
      namespaced
      creatable={false}
      extraColumns={[
        rawColumn('Holder', 'spec.holderIdentity', { width: 200 }),
        rawColumn('租期(s)', 'spec.leaseDurationSeconds', { width: 90 }),
      ]}
    />
  )
}
