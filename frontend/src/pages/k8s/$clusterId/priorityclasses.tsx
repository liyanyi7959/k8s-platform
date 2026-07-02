import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function PriorityClassesPage() {
  return (
    <GenericResourceList
      resource="priorityclasses"
      title="PriorityClasses"
      extraColumns={[
        rawColumn('优先级值', 'spec.value', { width: 120 }),
        rawColumn('全局默认', 'spec.globalDefault', {
          width: 100,
          render: (val) => (val ? '是' : '-'),
        }),
        rawColumn('描述', 'spec.description'),
      ]}
    />
  )
}
