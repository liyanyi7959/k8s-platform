import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function ClusterRoleBindingsPage() {
  return (
    <GenericResourceList
      resource="clusterrolebindings"
      title="ClusterRoleBindings"
      extraColumns={[
        rawColumn('Role Ref', 'roleRef.name', { width: 200 }),
        rawColumn('Subjects', 'subjects', {
          width: 100,
          render: (val) => {
            const subjects = val as Array<unknown>
            return subjects?.length ?? 0
          },
        }),
      ]}
    />
  )
}
