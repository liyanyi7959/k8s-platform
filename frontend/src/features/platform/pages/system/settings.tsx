import React, { useEffect } from 'react'
import {
  ProForm,
  ProFormDigit,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components'
import { Button, Form, message, Space, Spin } from 'antd'
import { useMutation, useQuery } from '@tanstack/react-query'
import { ReloadOutlined, RotateLeftOutlined, SaveOutlined } from '@ant-design/icons'
import { AppPage, FormWorkspace } from '@/components'
import { getSystemSettings, updateSystemSettings } from '@/features/platform/api'
import { systemSettingsSchema } from '@/schemas/system'
import type { SystemSettingsInput } from '@/schemas/system'

/** 系统设置页 */
const SettingsPage: React.FC = () => {
  const [form] = Form.useForm<SystemSettingsInput>()
  const {
    data: settings,
    isLoading,
    refetch,
    isRefetching,
  } = useQuery({
    queryKey: ['system-settings'],
    queryFn: () => getSystemSettings(),
  })

  useEffect(() => {
    if (settings) {
      form.setFieldsValue(settings)
    }
  }, [form, settings])

  const updateMutation = useMutation({
    mutationFn: (payload: SystemSettingsInput) => updateSystemSettings(payload),
    onSuccess: () => {
      message.success('保存成功')
    },
    onError: () => {
      message.error('保存失败')
    },
  })

  const handleSubmit = async (values: SystemSettingsInput) => {
    const result = systemSettingsSchema.safeParse(values)
    if (!result.success) {
      message.error(result.error.issues[0]?.message || '表单校验失败')
      return false
    }
    updateMutation.mutate(result.data)
    return true
  }

  return (
    <AppPage>
      <FormWorkspace
        title="系统设置"
        actions={(
          <Space>
            <Button
              loading={isRefetching}
              icon={<ReloadOutlined />}
              onClick={() => refetch()}
            >
              重新加载
            </Button>
            <Button
              icon={<RotateLeftOutlined />}
              onClick={() => {
                if (settings) {
                  form.setFieldsValue(settings)
                } else {
                  form.resetFields()
                }
              }}
            >
              重置
            </Button>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={updateMutation.isPending}
              onClick={() => form.submit()}
            >
              保存设置
            </Button>
          </Space>
        )}
      >
        <Spin spinning={isLoading}>
          <ProForm<SystemSettingsInput>
            form={form}
            onFinish={handleSubmit}
            submitter={false}
            layout="vertical"
          >
            <ProFormText
              name="siteName"
              label="站点名称"
              rules={[{ required: true }]}
              fieldProps={{ maxLength: 64 }}
            />
            <ProFormDigit
              name="sessionTimeout"
              label="会话超时（秒）"
              rules={[{ required: true }]}
              min={300}
              max={86400}
              fieldProps={{ precision: 0 }}
            />
            <ProFormDigit
              name="maxLoginAttempts"
              label="最大登录尝试次数"
              rules={[{ required: true }]}
              min={3}
              max={10}
              fieldProps={{ precision: 0 }}
            />
            <ProFormDigit
              name="passwordExpirationDays"
              label="密码过期天数"
              rules={[{ required: true }]}
              min={30}
              max={365}
              fieldProps={{ precision: 0 }}
            />
            <ProFormSwitch name="enableAuditLog" label="启用审计日志" />
            <ProFormSwitch name="enableTwoFactorAuth" label="启用双因素认证" />
          </ProForm>
        </Spin>
      </FormWorkspace>
    </AppPage>
  )
}

export default SettingsPage
