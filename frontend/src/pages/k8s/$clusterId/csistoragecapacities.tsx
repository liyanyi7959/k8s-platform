import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function CSIStorageCapacitiesPage() {
  return (
    <GenericResourceList
      resource="csistoragecapacities"
      title="CSIStorageCapacities"
      namespaced
      creatable={false}
      deletable={false}
      extraColumns={[
        rawColumn('StorageClass', 'storageClassName', { width: 180 }),
        rawColumn('容量', 'capacity.storage', { width: 120 }),
      ]}
    />
  )
}
