<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getSystemConfig, updateSystemConfig } from '../../api/admin'
import { MessagePlugin } from 'tdesign-vue-next'
import { ServerIcon } from 'tdesign-icons-vue-next'

const loading = ref(false)
const config = ref({
  site_name: 'Krasis',
  allow_signup: true,
  require_email_verification: true,
  default_role: 'member',
  max_notes_per_user: -1,
  max_storage_per_user_bytes: 10737418240,
  max_file_size_bytes: 104857600,
  session_duration_days: 7,
  max_devices_per_user: 10,
  enable_sharing: true,
  enable_ai: true,
  maintenance_mode: false,
})

onMounted(() => { fetchConfig() })

async function fetchConfig() {
  loading.value = true
  try {
    const res = await getSystemConfig()
    const d = res.data?.data || res.data || {}
    if (d && typeof d === 'object') {
      Object.assign(config.value, d)
    }
  } catch {
    MessagePlugin.error('获取系统配置失败')
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  try {
    await updateSystemConfig(config.value)
    MessagePlugin.success('配置已保存')
  } catch {
    MessagePlugin.error('保存失败')
  }
}

function formatBytes(bytes: number) {
  if (bytes === -1) return '无限制'
  const gb = bytes / 1073741824
  return `${gb} GB`
}
</script>

<template>
  <div class="system-config-view">
    <div class="page-header">
      <ServerIcon class="page-icon" />
      <h1>系统配置</h1>
      <span class="page-desc">管理注册、存储和功能开关</span>
    </div>

    <div class="form-card">
      <t-form :data="config" label-width="160px" v-loading="loading">
        <t-divider align="left" style="margin: 0 0 12px; font-size: 14px; font-weight: 600; color: #1d2129;">基本设置</t-divider>

        <t-form-item label="站点名称">
          <t-input v-model="config.site_name" style="width: 300px;" />
        </t-form-item>

        <t-form-item label="允许注册">
          <t-switch v-model="config.allow_signup" />
        </t-form-item>

        <t-form-item label="邮箱验证">
          <t-switch v-model="config.require_email_verification" />
          <span class="help-text">新注册用户需验证邮箱</span>
        </t-form-item>

        <t-form-item label="默认角色">
          <t-select v-model="config.default_role" style="width: 160px;">
            <t-option value="member" label="普通用户" />
            <t-option value="viewer" label="查看者" />
          </t-select>
        </t-form-item>

        <t-divider align="left" style="margin: 16px 0 12px; font-size: 14px; font-weight: 600; color: #1d2129;">限制设置</t-divider>

        <t-form-item label="每用户最大笔记数">
          <t-input-number v-model="config.max_notes_per_user" style="width: 200px;" />
          <span class="help-text">-1 表示无限制</span>
        </t-form-item>

        <t-form-item label="每用户存储空间">
          <t-input-number v-model="config.max_storage_per_user_bytes" :step="1073741824" style="width: 200px;" />
          <span class="help-text">{{ formatBytes(config.max_storage_per_user_bytes) }}</span>
        </t-form-item>

        <t-form-item label="最大文件大小">
          <t-input-number v-model="config.max_file_size_bytes" :step="10485760" style="width: 200px;" />
          <span class="help-text">{{ (config.max_file_size_bytes / 1048576).toFixed(0) }} MB</span>
        </t-form-item>

        <t-form-item label="会话有效期(天)">
          <t-input-number v-model="config.session_duration_days" :min="1" :max="365" style="width: 200px;" />
        </t-form-item>

        <t-form-item label="每用户最大设备数">
          <t-input-number v-model="config.max_devices_per_user" :min="1" :max="50" style="width: 200px;" />
        </t-form-item>

        <t-divider align="left" style="margin: 16px 0 12px; font-size: 14px; font-weight: 600; color: #1d2129;">功能开关</t-divider>

        <t-form-item label="分享功能">
          <t-switch v-model="config.enable_sharing" />
        </t-form-item>

        <t-form-item label="AI 功能">
          <t-switch v-model="config.enable_ai" />
        </t-form-item>

        <t-form-item label="维护模式">
          <t-switch v-model="config.maintenance_mode" />
          <span class="help-text" style="color: #e34d59;">开启后用户无法访问</span>
        </t-form-item>

        <t-form-item>
          <t-button @click="handleSave" :loading="loading">保存配置</t-button>
        </t-form-item>
      </t-form>
    </div>
  </div>
</template>

<style scoped>
.form-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04), 0 1px 3px rgba(0, 0, 0, 0.06);
  padding: 24px;
  max-width: 720px;
}

.help-text {
  color: #86909c;
  font-size: 12px;
  margin-left: 8px;
}
</style>
