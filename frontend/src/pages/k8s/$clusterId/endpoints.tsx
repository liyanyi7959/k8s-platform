import React from 'react'
import GenericResourceList, { rawColumn, rawField } from '@/components/GenericResourceList'
import type { GenericResourceItem } from '@/services/k8s'
import { Descriptions, Table, Tag, Space } from 'antd'

const renderDetail = (record: GenericResourceItem) => {
  const subsets = (rawField(record, 'subsets') as any[]) || []
  const addresses = subsets.flatMap((s) =>
    (s.addresses || []).map((a: any) => ({
      ip: a.ip || '-',
      nodeName: a.nodeName || '-',
      targetRef: a.targetRef ? `${a.targetRef.kind}/${a.targetRef.name}` : '-',
    }))
  )
  const notReady = subsets.flatMap((s) =>
    (s.notReadyAddresses || []).map((a: any) => ({
      ip: a.ip || '-',
      nodeName: a.nodeName || '-',
      targetRef: a.targetRef ? `${a.targetRef.kind}/${a.targetRef.name}` : '-',
    }))
  )
  const ports = subsets.flatMap((s) =>
    (s.ports || []).map((p: any) => ({
      port: p.port || 0,
      protocol: p.protocol || 'TCP',
      name: p.name || '-',
    }))
  )

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{record.name}</Descriptions.Item>
        <Descriptions.Item label="Namespace"><Tag>{record.namespace}</Tag></Descriptions.Item>
      </Descriptions>

      <Table
        size="small"
        title={() => `就绪地址 (${addresses.length})`}
        rowKey={(_, i) => String(i)}
        pagination={{ defaultPageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        dataSource={addresses}
        columns={[
          { title: 'IP', dataIndex: 'ip', render: (v) => <Tag color="blue">{v}</Tag> },
          { title: '节点', dataIndex: 'nodeName', ellipsis: true },
          { title: 'Target', dataIndex: 'targetRef', ellipsis: true },
        ]}
      />

      {notReady.length > 0 && (
        <Table
          size="small"
          title={() => `未就绪地址 (${notReady.length})`}
          rowKey={(_, i) => String(i)}
          pagination={false}
          dataSource={notReady}
          columns={[
            { title: 'IP', dataIndex: 'ip', render: (v) => <Tag color="red">{v}</Tag> },
            { title: '节点', dataIndex: 'nodeName', ellipsis: true },
            { title: 'Target', dataIndex: 'targetRef', ellipsis: true },
          ]}
        />
      )}

      {ports.length > 0 && (
        <Table
          size="small"
          title={() => `端口 (${ports.length})`}
          rowKey={(_, i) => String(i)}
          pagination={false}
          dataSource={ports}
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

export default function EndpointsPage() {
  return (
    <GenericResourceList
      resource="endpoints"
      title="Endpoints"
      namespaced
      creatable={false}
      deletable={false}
      renderDetail={renderDetail}
      extraColumns={[
        rawColumn('地址数', 'subsets', {
          width: 100,
          align: 'center',
          sorter: (a, b) => {
            const sa = (rawField(a, 'subsets') as any[]) || []
            const sb = (rawField(b, 'subsets') as any[]) || []
            const ca = sa.reduce((sum, s) => sum + (s.addresses?.length || 0), 0)
            const cb = sb.reduce((sum, s) => sum + (s.addresses?.length || 0), 0)
            return ca - cb
          },
          render: (val) => {
            const subsets = val as Array<{ addresses?: unknown[] }>
            const count = subsets?.reduce((sum, s) => sum + (s.addresses?.length || 0), 0) || 0
            return count > 0 ? count : '-'
          },
        }),
        rawColumn('端口', 'subsets', {
          width: 120,
          render: (val) => {
            const subsets = val as Array<{ ports?: Array<{ port: number }> }>
            const ports = subsets?.flatMap((s) => s.ports?.map((p) => p.port) || []) || []
            return ports.length > 0 ? ports.join(', ') : '-'
          },
        }),
      ]}
    />
  )
}
