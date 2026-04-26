import React, { useEffect, useState } from 'react'
import { Card, Form, Input, InputNumber, Switch, Button, message, Tabs, Row, Col } from 'antd'
import { getSystemConfig, updateSystemConfig, getOAuthConfig, updateOAuthConfig } from '../../api'

const Settings: React.FC = () => {
  const [, setSysLoading] = useState(false)
  const [, setOauthLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [sysForm] = Form.useForm()
  const [oauthForm] = Form.useForm()

  useEffect(() => {
    const fetch = async () => {
      setSysLoading(true)
      try {
        const res = await getSystemConfig()
        sysForm.setFieldsValue(res.data.data)
      } catch { /* ignore */ } finally { setSysLoading(false) }
    }
    fetch()
  }, [sysForm])

  useEffect(() => {
    const fetch = async () => {
      setOauthLoading(true)
      try {
        const res = await getOAuthConfig()
        const data = res.data.data || {}
        const github = data.find((p: { provider: string }) => p.provider === 'github') || {}
        const google = data.find((p: { provider: string }) => p.provider === 'google') || {}
        oauthForm.setFieldsValue({ github, google })
      } catch { /* ignore */ } finally { setOauthLoading(false) }
    }
    fetch()
  }, [oauthForm])

  const handleSysSave = async (values: Record<string, unknown>) => {
    setSaving(true)
    try {
      await updateSystemConfig(values)
      message.success('系统配置已保存')
    } catch {
      message.error('保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleOauthSave = async (values: { github: Record<string, unknown>; google: Record<string, unknown> }) => {
    setSaving(true)
    try {
      await updateOAuthConfig(values)
      message.success('OAuth 配置已保存')
    } catch {
      message.error('保存失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card title="系统配置">
      <Tabs
        items={[
          {
            key: 'system',
            label: '系统参数',
            children: (
              <Form form={sysForm} layout="vertical" onFinish={handleSysSave} style={{ maxWidth: 600 }}>
                <Form.Item name="site_name" label="站点名称">
                  <Input />
                </Form.Item>
                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Item name="default_role" label="默认角色">
                      <Input />
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item name="session_duration_days" label="会话有效期（天）">
                      <InputNumber style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                </Row>
                <Row gutter={16}>
                  <Col span={8}>
                    <Form.Item name="allow_signup" label="允许注册" valuePropName="checked">
                      <Switch />
                    </Form.Item>
                  </Col>
                  <Col span={8}>
                    <Form.Item name="enable_sharing" label="允许分享" valuePropName="checked">
                      <Switch />
                    </Form.Item>
                  </Col>
                  <Col span={8}>
                    <Form.Item name="enable_ai" label="启用 AI" valuePropName="checked">
                      <Switch />
                    </Form.Item>
                  </Col>
                </Row>
                <Form.Item>
                  <Button type="primary" htmlType="submit" loading={saving}>保存</Button>
                </Form.Item>
              </Form>
            ),
          },
          {
            key: 'oauth',
            label: 'OAuth 配置',
            children: (
              <Form form={oauthForm} layout="vertical" onFinish={handleOauthSave} style={{ maxWidth: 600 }}>
                <Form.Item label="GitHub OAuth">
                  <Form.Item name={['github', 'enabled']} label="启用" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item name={['github', 'client_id']} label="Client ID">
                    <Input />
                  </Form.Item>
                  <Form.Item name={['github', 'client_secret']} label="Client Secret">
                    <Input.Password />
                  </Form.Item>
                  <Form.Item name={['github', 'redirect_uri']} label="Redirect URI">
                    <Input />
                  </Form.Item>
                </Form.Item>
                <Form.Item label="Google OAuth">
                  <Form.Item name={['google', 'enabled']} label="启用" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item name={['google', 'client_id']} label="Client ID">
                    <Input />
                  </Form.Item>
                  <Form.Item name={['google', 'client_secret']} label="Client Secret">
                    <Input.Password />
                  </Form.Item>
                  <Form.Item name={['google', 'redirect_uri']} label="Redirect URI">
                    <Input />
                  </Form.Item>
                </Form.Item>
                <Form.Item>
                  <Button type="primary" htmlType="submit" loading={saving}>保存</Button>
                </Form.Item>
              </Form>
            ),
          },
        ]}
      />
    </Card>
  )
}

export default Settings
