import React from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu, theme, Button, Avatar, Dropdown } from 'antd'
import {
  DashboardOutlined,
  TeamOutlined,
  RobotOutlined,
  SettingOutlined,
  ShareAltOutlined,
  FileTextOutlined,
  LogoutOutlined,
  UserOutlined,
  ApartmentOutlined,
} from '@ant-design/icons'
import { useAuth } from '../store/auth'

const { Sider, Header, Content } = Layout

const menuItems = [
  { key: '/', icon: <DashboardOutlined />, label: '仪表盘' },
  { key: '/users', icon: <TeamOutlined />, label: '用户管理' },
  { key: '/groups', icon: <ApartmentOutlined />, label: '用户组' },
  { key: '/ai-models', icon: <RobotOutlined />, label: 'AI 模型' },
  { key: '/ai-config', icon: <SettingOutlined />, label: 'AI 配置' },
  { key: '/shares', icon: <ShareAltOutlined />, label: '分享审核' },
  { key: '/settings', icon: <SettingOutlined />, label: '系统配置' },
  { key: '/logs', icon: <FileTextOutlined />, label: '操作日志' },
]

const MainLayout: React.FC = () => {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const {
    token: { colorBgContainer },
  } = theme.useToken()

  const userMenu = {
    items: [
      { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: logout },
    ],
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider theme="dark" breakpoint="lg" collapsedWidth={80}>
        <div style={{ padding: '16px', color: '#fff', fontSize: 18, fontWeight: 'bold', textAlign: 'center' }}>
          Krasis Admin
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{ padding: '0 24px', background: colorBgContainer, display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 12 }}>
          <Dropdown menu={userMenu} placement="bottomRight">
            <Button type="text" icon={<Avatar size="small" icon={<UserOutlined />} />}>
              {user?.username || user?.email || '管理员'}
            </Button>
          </Dropdown>
        </Header>
        <Content style={{ margin: 24 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}

export default MainLayout
