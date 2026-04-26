import React, { useEffect, useState } from 'react'
import { Card, Col, Row, Statistic, Spin, Typography } from 'antd'
import { TeamOutlined, FileTextOutlined, ShareAltOutlined, CloudUploadOutlined } from '@ant-design/icons'
import { getStatsOverview } from '../../api'

const { Title } = Typography

const Dashboard: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [stats, setStats] = useState<Record<string, number> | null>(null)

  useEffect(() => {
    const fetchStats = async () => {
      setLoading(true)
      try {
        const res = await getStatsOverview()
        setStats(res.data.data)
      } catch {
        // ignore
      } finally {
        setLoading(false)
      }
    }
    fetchStats()
  }, [])

  if (loading || !stats) {
    return <Spin tip="加载中..." style={{ marginTop: 100 }} />
  }

  return (
    <div>
      <Title level={4}>系统概览</Title>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="总用户数" value={stats.total_users || 0} prefix={<TeamOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="笔记总数" value={stats.total_notes || 0} prefix={<FileTextOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="分享总数" value={stats.total_shares || 0} prefix={<ShareAltOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="待审核" value={stats.pending_shares || 0} prefix={<CloudUploadOutlined />} />
          </Card>
        </Col>
      </Row>
      <Row gutter={16} style={{ marginTop: 16 }}>
        <Col span={24}>
          <Card title="存储使用">
            <Statistic title="已使用 (GB)" value={stats.storage_used_gb || 0} precision={2} />
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard
