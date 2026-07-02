import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function LimitRangesPage() {
  return (
    <GenericResourceList
      resource="limitranges"
      title="LimitRanges"
      namespaced
      creatable={false}
      extraColumns={[
        rawColumn('限制项数', 'spec.limits', {
          width: 90,
          render: (val) => {
            const limits = val as Array<unknown>
            return limits?.length || 0
          },
        }),
      ]}
    />
  )
}
