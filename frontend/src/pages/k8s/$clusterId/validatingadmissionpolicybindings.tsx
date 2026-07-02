import GenericResourceList, { rawColumn } from '@/components/GenericResourceList'

export default function ValidatingAdmissionPolicyBindingsPage() {
  return (
    <GenericResourceList
      resource="validatingadmissionpolicybindings"
      title="ValidatingAdmissionPolicyBindings"
      extraColumns={[
        rawColumn('Policy', 'spec.policyName', { width: 200 }),
      ]}
    />
  )
}
