import React, { useRef } from 'react'
import { ActionType, ProColumns, ProTable } from '@ant-design/pro-components'
import { Tag } from 'antd'
import dayjs from 'dayjs'
import { getAuditLogs } from '../../api'

const Logs: React.FC = () => {
  const actionRef = useRef<ActionType>()

  const columns: ProColumns<API.Log>[] = [
    { title: '操作', dataIndex: 'action', width: 180 },
    { title: '目标类型', dataIndex: 'target_type', render: (_, r) => r.target_type ? <Tag>{r.target_type}</Tag> : '-' },
    { title: '管理员', dataIndex: 'admin_id', width: 280, ellipsis: true, copyable: true },
    { title: 'IP', dataIndex: 'ip_address', width: 150, hideInSearch: true },
    { title: '时间', dataIndex: 'created_at', valueType: 'dateTime', render: (_, r) => dayjs(r.created_at).format('YYYY-MM-DD HH:mm:ss'), hideInSearch: true },
    {
      title: '日期范围',
      dataIndex: 'date_range',
      valueType: 'dateRange',
      hideInTable: true,
      fieldProps: {
        format: 'YYYY-MM-DD',
      },
      search: {
        transform: (value) => ({ start_date: value[0], end_date: value[1] }),
      },
    },
  ]

  return (
    <ProTable<API.Log>
      actionRef={actionRef}
      headerTitle="操作日志"
      rowKey="id"
      request={async (params) => {
        const res = await getAuditLogs({
          page: params.current || 1,
          size: params.pageSize || 20,
          action: params.action,
          user_id: params.user_id,
          start_date: params.start_date,
          end_date: params.end_date,
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

export default Logs
