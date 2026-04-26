import React, { useRef } from 'react'
import { ActionType, ProColumns, ProTable } from '@ant-design/pro-components'
import { Button, Tag, message, Popconfirm, Space } from 'antd'
import { CheckOutlined, CloseOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { getPendingShares, approveShare, rejectShare, revokeShare } from '../../api'

const Shares: React.FC = () => {
  const actionRef = useRef<ActionType>()

  const columns: ProColumns<API.Share>[] = [
    { title: 'ID', dataIndex: 'id', width: 280, ellipsis: true, copyable: true },
    { title: '笔记标题', dataIndex: 'note_title', ellipsis: true },
    { title: '分享人', dataIndex: 'creator_email', ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: {
        pending: { text: '待审核', status: 'Warning' },
        approved: { text: '已通过', status: 'Success' },
        rejected: { text: '已拒绝', status: 'Error' },
        revoked: { text: '已撤回', status: 'Default' },
      },
      render: (_, r) => {
        const map: Record<string, { color: string; text: string }> = {
          pending: { color: 'orange', text: '待审核' },
          approved: { color: 'green', text: '已通过' },
          rejected: { color: 'red', text: '已拒绝' },
          revoked: { color: 'default', text: '已撤回' },
        }
        const s = map[r.status] || { color: 'default', text: r.status }
        return <Tag color={s.color}>{s.text}</Tag>
      },
    },
    { title: '过期时间', dataIndex: 'expires_at', valueType: 'dateTime', render: (_, r) => r.expires_at ? dayjs(r.expires_at).format('YYYY-MM-DD HH:mm') : '永不过期', hideInSearch: true },
    { title: '创建时间', dataIndex: 'created_at', valueType: 'dateTime', render: (_, r) => dayjs(r.created_at).format('YYYY-MM-DD HH:mm'), hideInSearch: true },
    {
      title: '操作',
      valueType: 'option',
      width: 250,
      render: (_, record) => (
        <Space>
          {record.status === 'pending' && (
            <>
              <Popconfirm title="确定通过？" onConfirm={async () => {
                await approveShare(record.id)
                message.success('已通过')
                actionRef.current?.reload()
              }}>
                <Button type="link" size="small" icon={<CheckOutlined />} style={{ color: '#52c41a' }}>通过</Button>
              </Popconfirm>
              <Popconfirm title="确定拒绝？" onConfirm={async () => {
                await rejectShare(record.id, '不符合分享规范')
                message.success('已拒绝')
                actionRef.current?.reload()
              }}>
                <Button type="link" size="small" danger icon={<CloseOutlined />}>拒绝</Button>
              </Popconfirm>
            </>
          )}
          {record.status === 'approved' && (
            <Popconfirm title="确定撤回此分享？" onConfirm={async () => {
              await revokeShare(record.id)
              message.success('已撤回')
              actionRef.current?.reload()
            }}>
              <Button type="link" size="small" danger>撤回</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  return (
    <ProTable<API.Share>
      actionRef={actionRef}
      headerTitle="分享审核"
      rowKey="id"
      request={async (params) => {
        const res = await getPendingShares({
          page: params.current || 1,
          size: params.pageSize || 20,
          status: params.status,
          keyword: params.keyword,
        })
        return { data: res.data.data?.items || res.data.data || [], total: res.data.data?.total || 0, success: true }
      }}
      columns={columns}
      scroll={{ x: 800 }}
      search={{ labelWidth: 'auto' }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}

export default Shares
