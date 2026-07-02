import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function ValidatingAdmissionPoliciesPage() {
  return (
    <GenericResourceList
      resource="validatingadmissionpolicies"
      title="ValidatingAdmissionPolicies"
      extraColumns={[
        rawColumn('参数', 'spec.paramKind', { width: 150 }),
      ]}
    />
  )
}
