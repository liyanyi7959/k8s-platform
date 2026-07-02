import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function MutatingWebhooksPage() {
  return (
    <GenericResourceList
      resource="mutatingwebhooks"
      title="MutatingWebhooks"
      extraColumns={[
        rawColumn('Webhooks', 'webhooks', {
          width: 90,
          render: (val) => {
            const webhooks = val as Array<unknown>
            return webhooks?.length ?? 0
          },
        }),
      ]}
    />
  )
}
