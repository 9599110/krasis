import React, { useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider, theme } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import MainLayout from './layouts/MainLayout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Users from './pages/Users'
import AiModels from './pages/AiModels'
import AiConfig from './pages/AiConfig'
import Shares from './pages/Shares'
import Settings from './pages/Settings'
import Logs from './pages/Logs'
import Groups from './pages/Groups'
import { useAuth } from './store/auth'

const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const token = localStorage.getItem('admin_token')
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

const App: React.FC = () => {
  const fetchMe = useAuth((s) => s.fetchMe)

  useEffect(() => {
    if (localStorage.getItem('admin_token')) {
      fetchMe()
    }
  }, [fetchMe])

  return (
    <ConfigProvider locale={zhCN} theme={{ algorithm: theme.defaultAlgorithm }}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <MainLayout />
              </ProtectedRoute>
            }
          >
            <Route index element={<Dashboard />} />
            <Route path="users" element={<Users />} />
            <Route path="groups" element={<Groups />} />
            <Route path="ai-models" element={<AiModels />} />
            <Route path="ai-config" element={<AiConfig />} />
            <Route path="shares" element={<Shares />} />
            <Route path="settings" element={<Settings />} />
            <Route path="logs" element={<Logs />} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  )
}

export default App
