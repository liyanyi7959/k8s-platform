import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function RuntimeClassesPage() {
  return (
    <GenericResourceList
      resource="runtimeclasses"
      title="RuntimeClasses"
      extraColumns={[
        rawColumn('Handler', 'spec.handler', { width: 150 }),
      ]}
    />
  )
}
