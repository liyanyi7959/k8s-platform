import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function IngressClassesPage() {
  return (
    <GenericResourceList
      resource="ingressclasses"
      title="IngressClasses"
      extraColumns={[
        rawColumn('Controller', 'spec.controller', { width: 200 }),
        rawColumn('默认', 'spec.defaultBackend', {
          width: 80,
          render: (val) => (val ? '是' : '-'),
        }),
      ]}
    />
  )
}
