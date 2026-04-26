<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getOAuthConfig, updateOAuthConfig } from '../../api/admin'
import { MessagePlugin } from 'tdesign-vue-next'
import { LogoGithubIcon } from 'tdesign-icons-vue-next'

const loading = ref(false)
const config = ref({
  github_enabled: false,
  github_client_id: '',
  github_client_secret: '',
  google_enabled: false,
  google_client_id: '',
  google_client_secret: '',
  microsoft_enabled: false,
  microsoft_client_id: '',
  microsoft_client_secret: '',
})

onMounted(() => { fetchConfig() })

async function fetchConfig() {
  loading.value = true
  try {
    const res = await getOAuthConfig()
    const d = res.data?.data || res.data || {}
    if (d && typeof d === 'object') {
      for (const key of Object.keys(config.value) as (keyof typeof config.value)[]) {
        if (d[key] !== undefined) {
          if (typeof d[key] === 'object' && 'value' in d[key]) {
            (config.value as any)[key] = d[key].value
          } else {
            (config.value as any)[key] = d[key]
          }
        }
      }
    }
  } catch {
    MessagePlugin.error('获取 OAuth 配置失败')
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  try {
    await updateOAuthConfig(config.value)
    MessagePlugin.success('配置已保存')
  } catch {
    MessagePlugin.error('保存失败')
  }
}
</script>

<template>
  <div class="oauth-config-view">
    <div class="page-header">
      <LogoGithubIcon class="page-icon" />
      <h1>OAuth 配置</h1>
      <span class="page-desc">配置第三方登录提供商</span>
    </div>

    <div class="form-card">
      <t-form :data="config" label-width="140px" v-loading="loading">
        <!-- GitHub -->
        <div class="provider-card" :class="{ 'provider-enabled': config.github_enabled }">
          <div class="provider-header">
            <span class="provider-name">GitHub</span>
            <t-switch v-model="config.github_enabled" size="small" />
          </div>
          <t-form-item label="Client ID">
            <t-input v-model="config.github_client_id" placeholder="GitHub OAuth Client ID" style="width: 350px;" />
          </t-form-item>
          <t-form-item label="Client Secret">
            <t-input v-model="config.github_client_secret" type="password" placeholder="GitHub OAuth Client Secret" style="width: 350px;" />
          </t-form-item>
        </div>

        <!-- Google -->
        <div class="provider-card" :class="{ 'provider-enabled': config.google_enabled }">
          <div class="provider-header">
            <span class="provider-name">Google</span>
            <t-switch v-model="config.google_enabled" size="small" />
          </div>
          <t-form-item label="Client ID">
            <t-input v-model="config.google_client_id" placeholder="Google OAuth Client ID" style="width: 350px;" />
          </t-form-item>
          <t-form-item label="Client Secret">
            <t-input v-model="config.google_client_secret" type="password" placeholder="Google OAuth Client Secret" style="width: 350px;" />
          </t-form-item>
        </div>

        <!-- Microsoft -->
        <div class="provider-card" :class="{ 'provider-enabled': config.microsoft_enabled }">
          <div class="provider-header">
            <span class="provider-name">Microsoft</span>
            <t-switch v-model="config.microsoft_enabled" size="small" />
          </div>
          <t-form-item label="Client ID">
            <t-input v-model="config.microsoft_client_id" placeholder="Microsoft OAuth Client ID" style="width: 350px;" />
          </t-form-item>
          <t-form-item label="Client Secret">
            <t-input v-model="config.microsoft_client_secret" type="password" placeholder="Microsoft OAuth Client Secret" style="width: 350px;" />
          </t-form-item>
        </div>

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

.provider-card {
  background: #f7f8fa;
  border-radius: 8px;
  padding: 16px 20px;
  margin-bottom: 16px;
  border-left: 3px solid #e3e6e8;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.provider-card.provider-enabled {
  border-left-color: #00a870;
  background: #f0faf5;
}

.provider-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.provider-name {
  font-size: 14px;
  font-weight: 600;
  color: #1d2129;
}
</style>
