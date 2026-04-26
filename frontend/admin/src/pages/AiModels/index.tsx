import React, { useRef, useState } from 'react'
import { ActionType, ProColumns, ProTable, ModalForm, ProFormText, ProFormSelect, ProFormDigit, ProFormSwitch } from '@ant-design/pro-components'
import { Button, Tag, message, Popconfirm } from 'antd'
import { PlusOutlined, DeleteOutlined, ExperimentOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { getModels, createModel, deleteModel, testModel } from '../../api'

const AiModels: React.FC = () => {
  const actionRef = useRef<ActionType>()
  const [createVisible, setCreateVisible] = useState(false)
  const [testingId, setTestingId] = useState<string | null>(null)

  const columns: ProColumns<API.Model>[] = [
    { title: '名称', dataIndex: 'name', width: 150 },
    { title: '提供商', dataIndex: 'provider', valueEnum: { openai: { text: 'OpenAI' }, azure: { text: 'Azure' }, ollama: { text: 'Ollama' }, anthropic: { text: 'Anthropic' } } },
    { title: '类型', dataIndex: 'type', valueEnum: { llm: { text: 'LLM' }, embedding: { text: 'Embedding' } }, render: (_, r) => <Tag color={r.type === 'llm' ? 'blue' : 'purple'}>{r.type}</Tag> },
    { title: '模型名', dataIndex: 'model_name', ellipsis: true },
    { title: '默认', dataIndex: 'is_default', width: 60, render: (_, r) => r.is_default ? <Tag color="green">是</Tag> : '-' },
    { title: '状态', dataIndex: 'is_enabled', width: 80, valueEnum: { true: { text: '启用', status: 'Success' }, false: { text: '禁用', status: 'Error' } } },
    { title: '创建时间', dataIndex: 'created_at', valueType: 'dateTime', render: (_, r) => dayjs(r.created_at).format('YYYY-MM-DD HH:mm'), hideInSearch: true },
    {
      title: '操作',
      valueType: 'option',
      width: 200,
      render: (_, record) => [
        <Button key="test" type="link" size="small" icon={<ExperimentOutlined />} loading={testingId === record.id} onClick={async () => {
          setTestingId(record.id)
          try {
            const res = await testModel(record.id)
            const data = res.data.data
            message[data.status === 'ok' ? 'success' : 'error'](`测试${data.status === 'ok' ? '成功' : '失败'}: ${data.message}`)
          } catch {
            message.error('测试失败')
          } finally {
            setTestingId(null)
          }
        }}>测试</Button>,
        <Popconfirm key="del" title="确定删除？" onConfirm={async () => {
          await deleteModel(record.id)
          message.success('已删除')
          actionRef.current?.reload()
        }}>
          <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
        </Popconfirm>,
      ],
    },
  ]

  return (
    <div>
      <ProTable<API.Model>
        actionRef={actionRef}
        headerTitle="AI 模型管理"
        rowKey="id"
        request={async (params) => {
          const res = await getModels(params.type)
          return { data: res.data.data || [], total: res.data.data?.length || 0, success: true }
        }}
        columns={columns}
        scroll={{ x: 800 }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
            添加模型
          </Button>,
        ]}
        search={false}
      />

      <ModalForm
        open={createVisible}
        onOpenChange={setCreateVisible}
        title="添加 AI 模型"
        modalProps={{ destroyOnClose: true }}
        onFinish={async (values) => {
          await createModel({
            name: values.name,
            provider: values.provider,
            type: values.type,
            endpoint: values.endpoint,
            api_key: values.api_key,
            model_name: values.model_name,
            is_enabled: values.is_enabled ?? true,
            is_default: values.is_default ?? false,
            max_tokens: values.max_tokens || 4096,
            temperature: values.temperature || 0.7,
          })
          message.success('创建成功')
          setCreateVisible(false)
          actionRef.current?.reload()
          return true
        }}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true }]} />
        <ProFormSelect name="provider" label="提供商" rules={[{ required: true }]}
          options={[{ label: 'OpenAI', value: 'openai' }, { label: 'Azure', value: 'azure' }, { label: 'Ollama', value: 'ollama' }, { label: 'Anthropic', value: 'anthropic' }]} />
        <ProFormSelect name="type" label="类型" rules={[{ required: true }]}
          options={[{ label: 'LLM', value: 'llm' }, { label: 'Embedding', value: 'embedding' }]} />
        <ProFormText name="endpoint" label="API 地址" />
        <ProFormText name="api_key" label="API Key" />
        <ProFormText name="model_name" label="模型名称" rules={[{ required: true }]} />
        <ProFormDigit name="max_tokens" label="最大 Token" initialValue={4096} />
        <ProFormDigit name="temperature" label="Temperature" initialValue={0.7} min={0} max={2} fieldProps={{ step: 0.1 }} />
        <ProFormSwitch name="is_enabled" label="启用" initialValue={true} />
        <ProFormSwitch name="is_default" label="设为默认" />
      </ModalForm>
    </div>
  )
}

export default AiModels
