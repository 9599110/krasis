import React, { useRef, useState } from 'react'
import { ActionType, ProColumns, ProTable } from '@ant-design/pro-components'
import { Tag, Button, message, Modal, Popconfirm } from 'antd'
import { DeleteOutlined, LockOutlined, ExportOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { getUsers, deleteUser, batchDisableUsers, exportUsers, updateUserStatus } from '../../api'

const Users: React.FC = () => {
  const actionRef = useRef<ActionType>()
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])

  const columns: ProColumns<API.User>[] = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 280,
      copyable: true,
      ellipsis: true,
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      copyable: true,
    },
    {
      title: '用户名',
      dataIndex: 'username',
    },
    {
      title: '角色',
      dataIndex: 'role',
      valueEnum: { admin: { text: '管理员', status: 'Success' }, member: { text: '普通用户', status: 'Default' }, viewer: { text: '只读', status: 'Warning' } },
      render: (_, record) => {
        const color = record.role === 'admin' ? 'green' : 'blue'
        return <Tag color={color}>{record.role}</Tag>
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: { 1: { text: '正常', status: 'Success' }, 0: { text: '已禁用', status: 'Error' } },
    },
    {
      title: '注册时间',
      dataIndex: 'created_at',
      valueType: 'dateTime',
      render: (_, record) => dayjs(record.created_at).format('YYYY-MM-DD HH:mm'),
      hideInSearch: true,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 200,
      render: (_, record) => [
        record.status === 1 ? (
          <Popconfirm title="确定禁用此用户？" onConfirm={async () => {
            await updateUserStatus(record.id, 0)
            message.success('已禁用')
            actionRef.current?.reload()
          }}>
            <Button type="link" size="small" icon={<LockOutlined />} danger>禁用</Button>
          </Popconfirm>
        ) : (
          <Popconfirm title="确定启用此用户？" onConfirm={async () => {
            await updateUserStatus(record.id, 1)
            message.success('已启用')
            actionRef.current?.reload()
          }}>
            <Button type="link" size="small">启用</Button>
          </Popconfirm>
        ),
        <Popconfirm title="确定删除此用户？" onConfirm={async () => {
          await deleteUser(record.id)
          message.success('已删除')
          actionRef.current?.reload()
        }}>
          <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
        </Popconfirm>,
      ],
    },
  ]

  return (
    <ProTable<API.User>
      actionRef={actionRef}
      headerTitle="用户管理"
      rowKey="id"
      request={async (params) => {
        const res = await getUsers({
          page: params.current || 1,
          size: params.pageSize || 20,
          keyword: params.keyword,
          role: params.role,
        })
        return { data: res.data.data?.items || res.data.data || [], total: res.data.data?.total || 0, success: true }
      }}
      columns={columns}
      rowSelection={{
        selectedRowKeys,
        onChange: (keys) => setSelectedRowKeys(keys),
      }}
      scroll={{ x: 800 }}
      toolBarRender={() => [
        selectedRowKeys.length > 0 && (
          <Button danger onClick={async () => {
            Modal.confirm({
              title: '确定批量禁用选中用户？',
              onOk: async () => {
                await batchDisableUsers(selectedRowKeys as string[])
                message.success('已批量禁用')
                setSelectedRowKeys([])
                actionRef.current?.reload()
              },
            })
          }}>
            批量禁用 ({selectedRowKeys.length})
          </Button>
        ),
        <Button icon={<ExportOutlined />} onClick={async () => {
          const res = await exportUsers()
          const url = window.URL.createObjectURL(new Blob([res.data]))
          const a = document.createElement('a')
          a.href = url
          a.download = 'users.csv'
          a.click()
        }}>
          导出 CSV
        </Button>,
      ]}
      search={{
        labelWidth: 'auto',
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}

export default Users
