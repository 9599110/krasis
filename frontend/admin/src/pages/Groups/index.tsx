import React, { useRef, useState } from 'react'
import { ActionType, ProColumns, ProTable, ModalForm, ProFormText, ProFormSwitch } from '@ant-design/pro-components'
import { Button, Tag, message, Popconfirm, Space, Modal } from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined, SettingOutlined } from '@ant-design/icons'
import { Input } from 'antd'
import dayjs from 'dayjs'
import { getGroups, createGroup, updateGroup, deleteGroup, getGroupFeatures, updateGroupFeatures } from '../../api'

const Groups: React.FC = () => {
  const actionRef = useRef<ActionType>()
  const [createVisible, setCreateVisible] = useState(false)
  const [editingGroup, setEditingGroup] = useState<API.Group | null>(null)
  const [featuresVisible, setFeaturesVisible] = useState(false)
  const [featuresGroup, setFeaturesGroup] = useState<API.Group | null>(null)
  const [features, setFeatures] = useState<Record<string, unknown>>({})

  const columns: ProColumns<API.Group>[] = [
    { title: 'ID', dataIndex: 'id', width: 280, ellipsis: true, copyable: true },
    { title: '名称', dataIndex: 'name', width: 150 },
    { title: '描述', dataIndex: 'description', ellipsis: true },
    {
      title: '默认',
      dataIndex: 'is_default',
      width: 60,
      render: (_, r) => r.is_default ? <Tag color="green">是</Tag> : '-',
    },
    {
      title: '用户数',
      dataIndex: 'user_count',
      width: 80,
      render: (_, r) => r.user_count || 0,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      valueType: 'dateTime',
      render: (_, r) => dayjs(r.created_at).format('YYYY-MM-DD HH:mm'),
      hideInSearch: true,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 220,
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => setEditingGroup(record)}
          >
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            icon={<SettingOutlined />}
            onClick={() => openFeaturesModal(record)}
          >
            功能
          </Button>
          {!record.is_default && (
            <Popconfirm title="确定删除此用户组？" onConfirm={async () => {
              await deleteGroup(record.id)
              message.success('已删除')
              actionRef.current?.reload()
            }}>
              <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const openFeaturesModal = async (group: API.Group) => {
    setFeaturesGroup(group)
    setFeaturesVisible(true)
    try {
      const res = await getGroupFeatures(group.id)
      const list = res.data.data || []
      const map: Record<string, unknown> = {}
      for (const item of list) {
        map[item.feature_key] = parseFeatureValue(item.feature_value)
      }
      setFeatures(map)
    } catch {
      setFeatures({})
    }
  }

  const parseFeatureValue = (raw: unknown): unknown => {
    if (typeof raw === 'string') {
      try { return JSON.parse(raw) } catch { return raw }
    }
    return raw
  }

  const handleSaveFeatures = async () => {
    if (!featuresGroup) return
    const raw: Record<string, string> = {}
    for (const [key, val] of Object.entries(features)) {
      raw[key] = JSON.stringify(val)
    }
    await updateGroupFeatures(featuresGroup.id, raw as Record<string, unknown>)
    message.success('功能配置已保存')
    setFeaturesVisible(false)
  }

  return (
    <div>
      <ProTable<API.Group>
        actionRef={actionRef}
        headerTitle="用户组管理"
        rowKey="id"
        request={async () => {
          const res = await getGroups()
          return { data: res.data.data || [], total: res.data.data?.length || 0, success: true }
        }}
        columns={columns}
        scroll={{ x: 800 }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
            添加用户组
          </Button>,
        ]}
        search={false}
      />

      {/* Create Modal */}
      <ModalForm
        open={createVisible}
        onOpenChange={setCreateVisible}
        title="添加用户组"
        modalProps={{ destroyOnClose: true }}
        onFinish={async (values) => {
          await createGroup({
            name: values.name,
            description: values.description,
            is_default: values.is_default ?? false,
          })
          message.success('创建成功')
          setCreateVisible(false)
          actionRef.current?.reload()
          return true
        }}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true }]} />
        <ProFormText name="description" label="描述" />
        <ProFormSwitch name="is_default" label="设为默认组" initialValue={false} />
      </ModalForm>

      {/* Edit Modal */}
      <ModalForm
        open={!!editingGroup}
        onOpenChange={(v) => { if (!v) setEditingGroup(null) }}
        title="编辑用户组"
        modalProps={{ destroyOnClose: true }}
        initialValues={editingGroup || undefined}
        onFinish={async (values) => {
          if (!editingGroup) return false
          await updateGroup(editingGroup.id, {
            name: values.name,
            description: values.description,
          })
          message.success('更新成功')
          setEditingGroup(null)
          actionRef.current?.reload()
          return true
        }}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true }]} />
        <ProFormText name="description" label="描述" />
      </ModalForm>

      {/* Features Modal */}
      <Modal
        open={featuresVisible}
        onCancel={() => setFeaturesVisible(false)}
        onOk={handleSaveFeatures}
        title={featuresGroup ? `功能配置 - ${featuresGroup.name}` : '功能配置'}
        width={500}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginTop: 8 }}>
          {Object.entries(features).map(([key, val]) => (
            <div key={key} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ minWidth: 120, fontWeight: 500 }}>{key}</span>
              <Input
                value={typeof val === 'object' ? JSON.stringify(val) : String(val)}
                onChange={(e) => {
                  try {
                    setFeatures((prev) => ({ ...prev, [key]: JSON.parse(e.target.value) }))
                  } catch {
                    setFeatures((prev) => ({ ...prev, [key]: e.target.value }))
                  }
                }}
              />
            </div>
          ))}
          {Object.keys(features).length === 0 && (
            <div style={{ color: '#999', textAlign: 'center', padding: '24px 0' }}>暂无功能配置</div>
          )}
        </div>
      </Modal>
    </div>
  )
}

export default Groups
