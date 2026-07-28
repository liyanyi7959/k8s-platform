import React from 'react'
import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Table, Tag, Space } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const controller = rawField(record, 'spec.controller') as string
  const defaultBackend = rawField(record, 'spec.defaultBackend')

  const backendData = defaultBackend ? [{
    key: 0,
    service: defaultBackend.service?.name
      ? `${defaultBackend.service.name}:${defaultBackend.service.port?.number || defaultBackend.service.port?.name || '*'}`
      : '-',
    resource: defaultBackend.resource ? `${defaultBackend.resource.kind}/${defaultBackend.resource.name}` : '-',
  }] : []

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Controller"><Tag color="blue">{controller || '-'}</Tag></Descriptions.Item>
      </Descriptions>

      {backendData.length > 0 && (
        <Table
          size="small"
          title={() => '默认后端'}
          rowKey="key"
          pagination={false}
          dataSource={backendData}
          columns={[
            { title: 'Service', dataIndex: 'service' },
            { title: 'Resource', dataIndex: 'resource' },
          ]}
        />
      )}
    </Space>
  )
}

export default function IngressClassesPage() {
  return (
    <GenericResourceList
      resource="ingressclasses"
      title="IngressClasses"
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('Controller', 'spec.controller', {
          width: 200,
          align: 'center',
          sorter: (a, b) => {
            const ca = (rawField(a, 'spec.controller') as string) || ''
            const cb = (rawField(b, 'spec.controller') as string) || ''
            return ca.localeCompare(cb)
          },
        }),
        rawColumn('默认', 'spec.defaultBackend', {
          width: 80,
          align: 'center',
          render: (val) => (val ? <Tag color="green">是</Tag> : '-'),
        }),
      ]}
    />
  )
}
