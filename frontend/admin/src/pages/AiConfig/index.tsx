import React, { useEffect, useState } from 'react'
import { Card, Form, InputNumber, Input, Switch, Button, message, Row, Col } from 'antd'
import { getAIConfig, updateAIConfig } from '../../api'

const AiConfig: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [form] = Form.useForm()
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    const fetch = async () => {
      setLoading(true)
      try {
        const res = await getAIConfig()
        form.setFieldsValue(res.data.data)
      } catch {
        message.error('获取配置失败')
      } finally {
        setLoading(false)
      }
    }
    fetch()
  }, [form])

  const onFinish = async (values: Record<string, unknown>) => {
    setSaving(true)
    try {
      await updateAIConfig(values)
      message.success('保存成功')
    } catch {
      message.error('保存失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card loading={loading} title="AI 系统配置">
      <Form form={form} layout="vertical" onFinish={onFinish} style={{ maxWidth: 600 }}>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="chunk_size" label="文档切片大小 (tokens)">
              <InputNumber min={100} max={4000} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="chunk_overlap" label="切片重叠 (tokens)">
              <InputNumber min={0} max={500} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="top_k" label="RAG 检索 Top K">
              <InputNumber min={1} max={20} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="score_threshold" label="相似度阈值">
              <InputNumber min={0} max={1} step={0.05} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="max_context_tokens" label="最大上下文 Token">
              <InputNumber min={1000} max={32000} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="enable_streaming" label="流式响应" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
        </Row>
        <Form.Item name="enable_rag" label="启用 RAG" valuePropName="checked">
          <Switch />
        </Form.Item>
        <Form.Item name="system_prompt" label="系统提示词">
          <Input.TextArea rows={4} />
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit" loading={saving}>
            保存配置
          </Button>
        </Form.Item>
      </Form>
    </Card>
  )
}

export default AiConfig
