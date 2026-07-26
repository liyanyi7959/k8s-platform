/**
 * 命名空间选择器
 * 跨 K8s 页面复用，统一样式和行为
 */
import React from 'react'
import { Select } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { listNamespaces } from '@/services/k8s'

interface NamespaceSelectorProps {
  clusterId: number
  value?: string
  onChange?: (value: string) => void
  allowClear?: boolean
  placeholder?: string
  style?: React.CSSProperties
  showAll?: boolean
}

export const NamespaceSelector: React.FC<NamespaceSelectorProps> = ({
  clusterId,
  value,
  onChange,
  allowClear = true,
  placeholder = '选择命名空间',
  style = { width: 180 },
  showAll = true,
}) => {
  const { data } = useQuery({
    queryKey: ['namespaces-selector', clusterId],
    queryFn: () => listNamespaces(clusterId),
    enabled: !!clusterId,
    staleTime: 60_000,
  })

  const options = ((Array.isArray(data) ? data : (data as any)?.items) || []).map((ns: any) => ({
    label: ns.name,
    value: ns.name,
  }))

  return (
    <Select
      placeholder={placeholder}
      value={value}
      onChange={onChange}
      allowClear={allowClear}
      style={style}
      options={showAll ? [{ label: '全部命名空间', value: '' }, ...options] : options}
      showSearch
      filterOption={(input, option) => (option?.label as string)?.toLowerCase().includes(input.toLowerCase())}
    />
  )
}
