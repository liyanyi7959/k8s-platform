import React, { useEffect } from 'react'
import { ProForm, ProFormDigit, ProFormSwitch, ProFormText } from '@ant-design/pro-components'
import { Button, Card, Form, message } from 'antd'
import { useMutation, useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { getSystemSettings, updateSystemSettings } from '@/services/system'
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
      <div className="app-data-console">
        <Card className="app-data-console__card app-form-card" bordered={false}>
          <ProForm<SystemSettingsInput>
            form={form}
            onFinish={handleSubmit}
            loading={isLoading}
            submitter={{
              render: (_, dom) => (
                <div className="app-data-console__filters-right">
                  <Button loading={isRefetching} onClick={() => refetch()}>
                    重新加载
                  </Button>
                  {dom}
                </div>
              ),
              submitButtonProps: {
                type: 'primary',
                loading: updateMutation.isPending,
              },
            }}
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
        </Card>
      </div>
    </AppPage>
  )
}

export default SettingsPage
