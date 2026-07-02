import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function EndpointsPage() {
  return (
    <GenericResourceList
      resource="endpoints"
      title="Endpoints"
      namespaced
      creatable={false}
      extraColumns={[
        rawColumn('地址数', 'subsets', {
          width: 100,
          render: (val) => {
            const subsets = val as Array<{ addresses?: unknown[] }>
            const count = subsets?.reduce((sum, s) => sum + (s.addresses?.length || 0), 0) || 0
            return count > 0 ? count : '-'
          },
        }),
        rawColumn('端口', 'subsets', {
          width: 120,
          render: (val) => {
            const subsets = val as Array<{ ports?: Array<{ port: number }> }>
            const ports = subsets?.flatMap((s) => s.ports?.map((p) => p.port) || []) || []
            return ports.length > 0 ? ports.join(', ') : '-'
          },
        }),
      ]}
    />
  )
}
