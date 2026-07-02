import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function ValidatingWebhooksPage() {
  return (
    <GenericResourceList
      resource="validatingwebhooks"
      title="ValidatingWebhooks"
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
