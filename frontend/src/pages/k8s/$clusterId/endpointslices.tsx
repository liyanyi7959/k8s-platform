import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function EndpointSlicesPage() {
  return (
    <GenericResourceList
      resource="endpointslices"
      title="EndpointSlices"
      namespaced
      creatable={false}
      extraColumns={[
        rawColumn('地址类型', 'addressType', { width: 120 }),
        rawColumn('端点数', 'endpoints', {
          width: 90,
          render: (val) => {
            const endpoints = val as unknown[]
            return endpoints?.length || 0
          },
        }),
      ]}
    />
  )
}
