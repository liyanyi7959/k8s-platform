import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Tag, Space, Table } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const spec = rawField(record, 'spec') as any || {}
  const validations = (spec.validations || []) as any[]
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="ParamKind"><Tag color="blue">{spec.paramKind?.kind || '-'}</Tag></Descriptions.Item>
      </Descriptions>
      {validations.length > 0 && (
        <Table size="small" title={() => `Validations (${validations.length})`} rowKey={(_, i) => String(i)} pagination={false}
          dataSource={validations}
          columns={[
            { title: 'Expression', dataIndex: 'expression', ellipsis: true },
            { title: 'Message', dataIndex: 'message', ellipsis: true },
          ]}
        />
      )}
    </Space>
  )
}

export default function ValidatingAdmissionPoliciesPage() {
  return (
    <GenericResourceList
      resource="validatingadmissionpolicies"
      title="ValidatingAdmissionPolicies"
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('参数', 'spec.paramKind', { width: 150 }),
      ]}
    />
  )
}
