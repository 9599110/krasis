<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { listFolders, createFolder, deleteFolder } from '../api/notes'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import AIChatPanel from '../components/AIChatPanel.vue'

const aiPanelRef = ref<InstanceType<typeof AIChatPanel> | null>(null)

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const sidebarCollapsed = ref(false)
const folders = ref<any[]>([])
const showNewFolder = ref(false)
const newFolderName = ref('')

const navItems = [
  { name: 'notes', label: '笔记', icon: 'file-copy' },
  { name: 'folders', label: '文件夹', icon: 'folder' },
  { name: 'ai-chat', label: 'AI 对话', icon: 'chat' },
  { name: 'search', label: '搜索', icon: 'search' },
]

onMounted(async () => {
  loadFolders()
})

async function loadFolders() {
  try {
    const res = await listFolders()
    const d = res.data?.data || res.data || {}
    folders.value = d.items || []
  } catch {
    // ignore
  }
}

function onFolderClick(folderId: string | null) {
  router.push({ name: 'notes', query: folderId ? { folder: folderId } : {} })
}

async function createNewFolder() {
  if (!newFolderName.value.trim()) return
  try {
    await createFolder({ name: newFolderName.value })
    showNewFolder.value = false
    newFolderName.value = ''
    loadFolders()
    MessagePlugin.success('文件夹已创建')
  } catch {
    MessagePlugin.error('创建文件夹失败')
  }
}

async function handleDeleteFolder(id: string) {
  DialogPlugin.confirm({
    header: '删除文件夹',
    body: '确定要删除此文件夹吗？',
    onConfirm: async () => {
      try {
        await deleteFolder(id)
        MessagePlugin.success('已删除')
        loadFolders()
      } catch {
        MessagePlugin.error('删除失败')
      }
    },
  })
}

function isActive(name: string) {
  return route.name === name
}

function openAIChat() {
  aiPanelRef.value?.toggle()
}

function isActiveFolder(folderId: string | null) {
  if (route.name !== 'notes') return false
  return String(route.query.folder || '') === (folderId || '')
}
</script>

<template>
  <div class="app-layout">
    <aside class="sidebar" :class="{ collapsed: sidebarCollapsed }">
      <div class="sidebar-header">
        <div class="logo-row">
          <t-icon name="layers" size="24px" style="color: #1677ff" />
          <span v-show="!sidebarCollapsed" class="logo-text">Krasis</span>
        </div>
        <t-button variant="text" size="small" @click="sidebarCollapsed = !sidebarCollapsed">
          <t-icon :name="sidebarCollapsed ? 'menu-unfold' : 'menu-fold'" />
        </t-button>
      </div>

      <nav class="nav-list">
        <router-link
          v-for="item in navItems.filter(i => i.name !== 'ai-chat')"
          :key="item.name"
          :to="{ name: item.name }"
          class="nav-item"
          :class="{ active: isActive(item.name) }"
        >
          <t-icon :name="item.icon" />
          <span v-show="!sidebarCollapsed">{{ item.label }}</span>
        </router-link>
        <div
          class="nav-item ai-nav-item"
          @click="openAIChat"
        >
          <t-icon name="chat" />
          <span v-show="!sidebarCollapsed">AI 对话</span>
        </div>
      </nav>

      <div v-show="!sidebarCollapsed" class="folder-section">
        <div class="section-header">
          <span>文件夹</span>
          <t-button variant="text" size="small" @click="showNewFolder = true">
            <t-icon name="add" />
          </t-button>
        </div>

        <div
          class="folder-item"
          :class="{ active: isActiveFolder(null) }"
          @click="onFolderClick(null)"
        >
          <t-icon name="file" />
          <span>全部笔记</span>
        </div>

        <div
          v-for="folder in folders"
          :key="folder.id"
          class="folder-item"
          :class="{ active: isActiveFolder(folder.id) }"
          @click="onFolderClick(folder.id)"
        >
          <t-icon name="folder" />
          <span class="folder-name">{{ folder.name }}</span>
          <t-button
            variant="text"
            size="small"
            class="folder-delete"
            @click.stop="handleDeleteFolder(folder.id)"
          >
            <t-icon name="close" size="12px" />
          </t-button>
        </div>

        <div v-if="showNewFolder" class="new-folder-form">
          <t-input
            v-model="newFolderName"
            size="small"
            placeholder="文件夹名称"
            @enter="createNewFolder"
            @blur="!newFolderName && (showNewFolder = false)"
          />
          <div class="new-folder-actions">
            <t-button size="small" @click="createNewFolder">创建</t-button>
            <t-button size="small" variant="text" @click="showNewFolder = false">取消</t-button>
          </div>
        </div>
      </div>

      <div class="sidebar-footer" v-show="!sidebarCollapsed">
        <div class="user-info">
          <t-avatar size="32px">{{ authStore.user?.name?.charAt(0)?.toUpperCase() || '?' }}</t-avatar>
          <div class="user-details">
            <span class="user-name">{{ authStore.user?.name || '用户' }}</span>
          </div>
        </div>
        <div class="footer-links">
          <router-link :to="{ name: 'profile' }">
            <t-icon name="user" /> 个人资料
          </router-link>
          <a v-if="authStore.user?.role === 'admin'" @click="router.push({ path: '/admin' })">
            <t-icon name="setting" /> 管理后台
          </a>
          <a @click="authStore.logout(); router.push({ name: 'login' })">
            <t-icon name="poweroff" /> 退出登录
          </a>
        </div>
      </div>
    </aside>

    <main class="main-content">
      <router-view />
    </main>

    <AIChatPanel ref="aiPanelRef" />
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.sidebar {
  width: 240px;
  min-width: 240px;
  background: #f7f8fa;
  border-right: 1px solid #e5e6eb;
  display: flex;
  flex-direction: column;
  transition: width 0.2s, min-width 0.2s;
  overflow: hidden;
}

.sidebar.collapsed {
  width: 60px;
  min-width: 60px;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 12px;
  border-bottom: 1px solid #e5e6eb;
}

.logo-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.logo-text {
  font-size: 17px;
  font-weight: 700;
  color: #1d2129;
}

.nav-list {
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: 6px;
  color: #4e5969;
  text-decoration: none;
  font-size: 14px;
  font-weight: 500;
  transition: background 0.15s, color 0.15s;
}

.nav-item:hover {
  background: #e5e6eb;
  color: #1d2129;
}

.nav-item.active {
  background: #e8f3ff;
  color: #1677ff;
}

.ai-nav-item {
  cursor: pointer;
}

.ai-nav-item:hover {
  background: #e5e6eb;
  color: #1d2129;
}

.folder-section {
  flex: 1;
  padding: 8px;
  overflow-y: auto;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 12px 8px;
  font-size: 12px;
  font-weight: 600;
  color: #86909c;
}

.folder-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  color: #4e5969;
  transition: background 0.15s;
}

.folder-item:hover {
  background: #e5e6eb;
}

.folder-item.active {
  background: #e8f3ff;
  color: #1677ff;
}

.folder-name {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.folder-delete {
  display: none;
  padding: 0;
  min-width: 0;
  height: auto;
}

.folder-item:hover .folder-delete {
  display: inline-flex;
}

.new-folder-form {
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.new-folder-actions {
  display: flex;
  gap: 4px;
}

.sidebar-footer {
  padding: 12px;
  border-top: 1px solid #e5e6eb;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.user-details {
  min-width: 0;
}

.user-name {
  font-size: 13px;
  font-weight: 600;
  color: #1d2129;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.footer-links {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.footer-links a,
.footer-links .router-link-active,
.footer-links :deep(a) {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #86909c;
  text-decoration: none;
  padding: 4px 0;
  cursor: pointer;
  transition: color 0.15s;
}

.footer-links a:hover {
  color: #1677ff;
}

.main-content {
  flex: 1;
  overflow: auto;
  background: #fff;
}
</style>
