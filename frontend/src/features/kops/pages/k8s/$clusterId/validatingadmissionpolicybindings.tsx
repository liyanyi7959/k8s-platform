import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Tag, Space } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const spec = rawField(record, 'spec') as any || {}
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Policy"><Tag color="blue">{spec.policyName || '-'}</Tag></Descriptions.Item>
        {spec.paramRef && <Descriptions.Item label="ParamRef" span={2}>{spec.paramRef.name || '-'}</Descriptions.Item>}
        {spec.validationActions && <Descriptions.Item label="Actions" span={2}>{spec.validationActions.join(', ')}</Descriptions.Item>}
      </Descriptions>
    </Space>
  )
}

export default function ValidatingAdmissionPolicyBindingsPage() {
  return (
    <GenericResourceList
      resource="validatingadmissionpolicybindings"
      title="ValidatingAdmissionPolicyBindings"
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('Policy', 'spec.policyName', { width: 200 }),
      ]}
    />
  )
}
