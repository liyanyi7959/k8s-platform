/**
 * 修改密码弹窗
 * 调用 /api/v1/auth/change-password，成功后提示并要求重新登录
 */
import React from 'react'
import { Modal, Form, Input, message } from 'antd'
import { useMutation } from '@tanstack/react-query'
import { changePassword } from '@/services/auth'

export interface ChangePasswordModalProps {
  open: boolean
  onClose: () => void
  /** 修改成功后的回调（通常用于登出重登） */
  onSuccess?: () => void
}

interface FormValues {
  oldPassword: string
  newPassword: string
  confirmPassword: string
}

const ChangePasswordModal: React.FC<ChangePasswordModalProps> = ({ open, onClose, onSuccess }) => {
  const [form] = Form.useForm<FormValues>()

  const mutation = useMutation({
    mutationFn: (values: FormValues) =>
      changePassword({ oldPassword: values.oldPassword, newPassword: values.newPassword }),
    onSuccess: () => {
      message.success('密码修改成功，请重新登录')
      form.resetFields()
      onClose()
      onSuccess?.()
    },
    onError: (err: any) => {
      message.error(err?.message || '密码修改失败')
    },
  })

  const handleOk = async () => {
    const values = await form.validateFields()
    mutation.mutate(values)
  }

  const handleCancel = () => {
    form.resetFields()
    onClose()
  }

  return (
    <Modal
      title="修改密码"
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      confirmLoading={mutation.isPending}
      okText="确认修改"
      cancelText="取消"
      destroyOnClose
      maskClosable={false}
    >
      <Form form={form} layout="vertical" autoComplete="off" style={{ marginTop: 12 }}>
        <Form.Item
          label="当前密码"
          name="oldPassword"
          rules={[{ required: true, message: '请输入当前密码' }]}
        >
          <Input.Password placeholder="请输入当前密码" autoComplete="current-password" />
        </Form.Item>
        <Form.Item
          label="新密码"
          name="newPassword"
          rules={[
            { required: true, message: '请输入新密码' },
            { min: 6, message: '密码长度至少 6 位' },
          ]}
        >
          <Input.Password placeholder="请输入新密码" autoComplete="new-password" />
        </Form.Item>
        <Form.Item
          label="确认新密码"
          name="confirmPassword"
          dependencies={['newPassword']}
          rules={[
            { required: true, message: '请再次输入新密码' },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('newPassword') === value) {
                  return Promise.resolve()
                }
                return Promise.reject(new Error('两次输入的密码不一致'))
              },
            }),
          ]}
        >
          <Input.Password placeholder="请再次输入新密码" autoComplete="new-password" />
        </Form.Item>
      </Form>
    </Modal>
  )
}

export default ChangePasswordModal
