<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import {
  HomeIcon,
  UserListIcon,
  CheckCircleIcon,
  Ai1Icon,
  SettingIcon,
  UsergroupIcon,
  ServerIcon,
  LogoGithubIcon,
  HistoryIcon,
  BrowseGalleryIcon,
} from 'tdesign-icons-vue-next'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const collapsed = ref(false)

const menuItems = [
  { name: 'admin-dashboard', label: '仪表盘', icon: HomeIcon },
  { name: 'admin-users', label: '用户管理', icon: UserListIcon },
  { name: 'admin-shares', label: '分享审核', icon: CheckCircleIcon },
  { name: 'admin-ai-models', label: 'AI 模型', icon: Ai1Icon },
  { name: 'admin-ai-config', label: 'AI 配置', icon: SettingIcon },
  { name: 'admin-groups', label: '用户组', icon: UsergroupIcon },
  { name: 'admin-system-config', label: '系统配置', icon: ServerIcon },
  { name: 'admin-oauth-config', label: 'OAuth 配置', icon: LogoGithubIcon },
  { name: 'admin-logs', label: '操作日志', icon: HistoryIcon },
]

const pageTitles: Record<string, string> = {
  'admin-dashboard': '系统概览',
  'admin-users': '用户管理',
  'admin-shares': '分享审核',
  'admin-ai-models': 'AI 模型',
  'admin-ai-config': 'AI 配置',
  'admin-groups': '用户组',
  'admin-system-config': '系统配置',
  'admin-oauth-config': 'OAuth 配置',
  'admin-logs': '操作日志',
}

const activeMenu = computed(() => route.name as string)
const currentPageTitle = computed(() => pageTitles[route.name as string] || '后台管理')

function handleLogout() {
  authStore.logout()
  router.push({ name: 'login' })
}

function goBack() {
  router.push({ name: 'notes' })
}
</script>

<template>
  <div class="admin-layout admin-theme">
    <aside class="admin-sidebar" :class="{ collapsed }">
      <div class="sidebar-header">
        <span v-show="!collapsed" class="title">Krasis Admin</span>
        <button class="collapse-btn" @click="collapsed = !collapsed">
          <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2">
            <polyline :points="collapsed ? '9 18 15 12 9 6' : '15 18 9 12 15 6'" />
          </svg>
        </button>
      </div>

      <nav class="sidebar-nav">
        <router-link
          v-for="item in menuItems"
          :key="item.name"
          :to="{ name: item.name }"
          class="nav-item"
          :class="{ active: activeMenu === item.name }"
        >
          <component :is="item.icon" class="nav-icon" />
          <span v-show="!collapsed">{{ item.label }}</span>
        </router-link>
      </nav>

      <div class="sidebar-footer" v-show="!collapsed">
        <div class="user-info">
          <span class="username">{{ authStore.user?.name || 'Admin' }}</span>
          <t-button variant="text" size="small" @click="goBack">
            <BrowseGalleryIcon class="btn-icon" /> 返回应用
          </t-button>
          <t-button variant="text" size="small" theme="danger" @click="handleLogout">退出</t-button>
        </div>
      </div>
    </aside>

    <main class="admin-main">
      <div class="admin-topbar">
        <span class="breadcrumb-label">{{ currentPageTitle }}</span>
        <div class="topbar-right">
          <t-button variant="text" size="small" @click="goBack">
            返回应用
          </t-button>
        </div>
      </div>
      <div class="admin-content">
        <router-view :key="$route.fullPath" />
      </div>
    </main>
  </div>
</template>

<style scoped>
.admin-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.admin-sidebar {
  width: 240px;
  min-width: 240px;
  background: linear-gradient(180deg, #001529 0%, #002140 100%);
  color: #fff;
  display: flex;
  flex-direction: column;
  transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1), min-width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.15);
  z-index: 10;
}

.admin-sidebar.collapsed {
  width: 64px;
  min-width: 64px;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  min-height: 56px;
}

.sidebar-header .title {
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.3px;
  background: linear-gradient(135deg, #1890ff 0%, #36cfc9 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.collapse-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: rgba(255, 255, 255, 0.5);
  cursor: pointer;
  transition: all 0.2s ease;
}

.collapse-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.sidebar-nav {
  flex: 1;
  padding: 12px;
  overflow-y: auto;
}

.sidebar-nav::-webkit-scrollbar {
  width: 4px;
}
.sidebar-nav::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 2px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.65);
  text-decoration: none;
  font-size: 14px;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  margin-bottom: 2px;
}

.nav-item:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}

.nav-item:hover .nav-icon {
  opacity: 1;
}

.nav-item.active {
  background: rgba(24, 144, 255, 0.2);
  color: #fff;
  box-shadow: inset 3px 0 0 #1890ff;
}

.nav-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  opacity: 0.65;
  transition: opacity 0.2s ease;
}

.nav-item.active .nav-icon {
  opacity: 1;
}

.sidebar-footer {
  padding: 12px 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

.user-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.username {
  font-size: 13px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.85);
}

.btn-icon {
  width: 14px;
  height: 14px;
}

.admin-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: linear-gradient(135deg, #f5f7fa 0%, #e8ebf0 100%);
}

.admin-topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 24px;
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid #f0f0f0;
  min-height: 48px;
}

.breadcrumb-label {
  font-size: 15px;
  font-weight: 600;
  color: #1d2129;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.admin-content {
  flex: 1;
  overflow: auto;
  padding: 24px;
}
</style>
