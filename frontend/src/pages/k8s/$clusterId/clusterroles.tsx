import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function ClusterRolesPage() {
  return (
    <GenericResourceList
      resource="clusterroles"
      title="ClusterRoles"
      extraColumns={[
        rawColumn('规则数', 'rules', {
          width: 80,
          render: (val) => {
            const rules = val as Array<unknown>
            return rules?.length ?? 0
          },
        }),
      ]}
    />
  )
}
