import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function VolumeSnapshotClassesPage() {
  return (
    <GenericResourceList
      resource="volumesnapshotclasses"
      title="VolumeSnapshotClasses"
      extraColumns={[
        rawColumn('Driver', 'driver', { width: 200 }),
        rawColumn('Deletion', 'deletionPolicy', { width: 120 }),
      ]}
    />
  )
}
