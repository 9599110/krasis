<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'

const authStore = useAuthStore()
const activeTab = ref('profile')

const profileForm = ref({
  username: authStore.user?.name || '',
})
const saving = ref(false)

const sessions = ref<any[]>([])
const sessionsLoading = ref(false)

onMounted(() => { loadSessions() })

async function loadSessions() {
  sessionsLoading.value = true
  try {
    sessions.value = await authStore.getSessions()
  } catch {
    // ignore
  } finally {
    sessionsLoading.value = false
  }
}

async function handleSaveProfile() {
  if (!profileForm.value.username.trim()) {
    MessagePlugin.warning('请输入用户名')
    return
  }
  saving.value = true
  try {
    await authStore.updateProfile({ name: profileForm.value.username })
    MessagePlugin.success('资料已更新')
  } catch {
    MessagePlugin.error('更新失败')
  } finally {
    saving.value = false
  }
}

async function handleRevokeSession(id: string) {
  try {
    await authStore.revokeSession(id)
    MessagePlugin.success('设备已下线')
    loadSessions()
  } catch {
    MessagePlugin.error('操作失败')
  }
}

async function handleRevokeAll() {
  DialogPlugin.confirm({
    header: '确认',
    body: '确定要让所有设备下线吗？当前设备也会断开连接。',
    onConfirm: async () => {
      try {
        await authStore.revokeAllSessions()
        MessagePlugin.success('所有设备已下线')
        loadSessions()
      } catch {
        MessagePlugin.error('操作失败')
      }
    },
  })
}

function formatDate(dateStr: string) {
  if (!dateStr) return '未知'
  return new Date(dateStr).toLocaleString('zh-CN')
}
</script>

<template>
  <section class="profile-page">
    <div class="profile-header">
      <div class="avatar">{{ (authStore.user?.name || '?')[0].toUpperCase() }}</div>
      <div class="info">
        <h2>{{ authStore.user?.name || '未登录' }}</h2>
        <p class="email">{{ authStore.user?.email || '' }}</p>
      </div>
    </div>

    <t-tabs v-model="activeTab" class="profile-tabs">
      <t-tab-panel value="profile" label="个人资料">
        <t-form :data="profileForm" label-width="80px" class="profile-form">
          <t-form-item label="用户名">
            <t-input v-model="profileForm.username" style="width: 300px;" />
          </t-form-item>
          <t-form-item label="邮箱">
            <t-input :value="authStore.user?.email" disabled style="width: 300px;" />
          </t-form-item>
          <t-form-item label="角色">
            <t-tag>{{ authStore.user?.role || 'member' }}</t-tag>
          </t-form-item>
          <t-form-item>
            <t-button @click="handleSaveProfile" :loading="saving">保存修改</t-button>
          </t-form-item>
        </t-form>
      </t-tab-panel>

      <t-tab-panel value="devices" label="设备管理">
        <div class="devices-toolbar">
          <t-button variant="outline" theme="danger" @click="handleRevokeAll">全部下线</t-button>
        </div>
        <t-table
          :data="sessions"
          :loading="sessionsLoading"
          row-key="id"
          :columns="[
            { colKey: 'device_name', title: '设备', width: 160 },
            { colKey: 'ip_address', title: 'IP 地址', width: 160 },
            { colKey: 'created_at', title: '登录时间', width: 200 },
            { colKey: 'operations', title: '操作', width: 100 },
          ]"
        >
          <template #device_name="{ row }">
            <t-tag v-if="row.is_current" theme="success" variant="light-outline">当前设备</t-tag>
            <span v-else>{{ row.device_name || '未知设备' }}</span>
          </template>
          <template #created_at="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
          <template #operations="{ row }">
            <t-button
              v-if="!row.is_current"
              variant="text"
              size="small"
              theme="danger"
              @click="handleRevokeSession(row.id)"
            >
              下线
            </t-button>
          </template>
        </t-table>
      </t-tab-panel>

      <t-tab-panel value="settings" label="设置">
        <div class="settings-section">
          <h3>外观</h3>
          <div class="setting-item">
            <span>深色模式</span>
            <!-- TODO: wire up theme toggle -->
            <t-switch disabled />
          </div>
        </div>
        <div class="settings-section">
          <h3>关于</h3>
          <div class="setting-item">
            <span>版本</span>
            <t-tag variant="light-outline">v0.1.0</t-tag>
          </div>
        </div>
      </t-tab-panel>
    </t-tabs>
  </section>
</template>

<style scoped>
.profile-page {
  max-width: 720px;
  margin: 0 auto;
  padding: 24px;
}

.profile-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}

.avatar {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: #1677ff;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: 700;
}

.profile-header .info h2 {
  margin: 0;
  font-size: 20px;
}

.email {
  margin: 4px 0 0;
  color: #86909c;
  font-size: 14px;
}

.profile-tabs {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
}

.profile-form {
  margin-top: 16px;
}

.devices-toolbar {
  margin-bottom: 12px;
}

.settings-section {
  margin-bottom: 24px;
}

.settings-section h3 {
  margin: 0 0 12px;
  font-size: 16px;
  color: #1d2129;
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid #f2f3f5;
}
</style>
