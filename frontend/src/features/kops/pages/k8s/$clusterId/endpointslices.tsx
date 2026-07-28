import React from 'react'
import GenericResourceList, { rawColumn, rawField } from '@/features/kops/components/GenericResourceList'
import type { GenericResourceItem } from '@/features/kops/api/k8s'
import { Descriptions, Table, Tag, Space } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const endpoints = (rawField(record, 'endpoints') as any[]) || []
  const ports = (rawField(record, 'ports') as any[]) || []
  const addressType = rawField(record, 'addressType') as string

  const endpointData = endpoints.map((ep, i) => ({
    key: i,
    addresses: (ep.addresses || []).join(', ') || '-',
    ready: ep.conditions?.ready,
    serving: ep.conditions?.serving,
    terminating: ep.conditions?.terminating,
    targetRef: ep.targetRef ? `${ep.targetRef.kind}/${ep.targetRef.name}` : '-',
  }))

  const portData = ports.map((p, i) => ({
    key: i,
    port: p.port || 0,
    protocol: p.protocol || 'TCP',
    name: p.name || '-',
  }))

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Namespace"><Tag>{record.namespace}</Tag></Descriptions.Item>
        <Descriptions.Item label="地址类型"><Tag color="blue">{addressType || '-'}</Tag></Descriptions.Item>
        <Descriptions.Item label="端点数">{endpoints.length}</Descriptions.Item>
      </Descriptions>

      <Table
        size="small"
        title={() => `端点 (${endpoints.length})`}
        rowKey="key"
        pagination={{ defaultPageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        dataSource={endpointData}
        columns={[
          { title: '地址', dataIndex: 'addresses', ellipsis: true },
          { title: 'Ready', dataIndex: 'ready', width: 70, align: 'center', render: (v) => <Tag color={v ? 'green' : 'red'}>{v ? 'True' : 'False'}</Tag> },
          { title: 'Serving', dataIndex: 'serving', width: 70, align: 'center', render: (v) => <Tag color={v ? 'green' : 'red'}>{v ? 'True' : 'False'}</Tag> },
          { title: 'Terminating', dataIndex: 'terminating', width: 90, align: 'center', render: (v) => <Tag color={v ? 'orange' : 'default'}>{v ? 'True' : 'False'}</Tag> },
          { title: 'Target', dataIndex: 'targetRef', ellipsis: true },
        ]}
      />

      {portData.length > 0 && (
        <Table
          size="small"
          title={() => `端口 (${portData.length})`}
          rowKey="key"
          pagination={false}
          dataSource={portData}
          columns={[
            { title: '端口', dataIndex: 'port', width: 80, align: 'center' },
            { title: '协议', dataIndex: 'protocol', width: 80, align: 'center' },
            { title: '名称', dataIndex: 'name' },
          ]}
        />
      )}
    </Space>
  )
}

export default function EndpointSlicesPage() {
  return (
    <GenericResourceList
      resource="endpointslices"
      title="EndpointSlices"
      namespaced
      creatable={false}
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('地址类型', 'addressType', { width: 120, align: 'center' }),
        rawColumn('端点数', 'endpoints', {
          width: 90,
          align: 'center',
          sorter: (a, b) => {
            const ea = (rawField(a, 'endpoints') as any[]) || []
            const eb = (rawField(b, 'endpoints') as any[]) || []
            return ea.length - eb.length
          },
          render: (val) => {
            const endpoints = val as unknown[]
            return endpoints?.length || 0
          },
        }),
      ]}
    />
  )
}
