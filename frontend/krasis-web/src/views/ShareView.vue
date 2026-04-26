<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import apiClient from '../api/client'
import { MessagePlugin } from 'tdesign-vue-next'

const route = useRoute()
const router = useRouter()
const token = computed(() => String(route.params.token ?? ''))

const loading = ref(true)
const needPassword = ref(false)
const password = ref('')
const title = ref('')
const content = ref('')
const permission = ref<'view' | 'edit'>('view')
const error = ref('')
const showContent = ref(false)

onMounted(() => loadShare())

async function loadShare() {
  loading.value = true
  error.value = ''
  try {
    const res = await apiClient.get(`/share/${token.value}`, {
      headers: password.value ? { 'X-Share-Password': password.value } : {},
    })
    const d = res.data?.data || res.data || {}
    title.value = d.note?.title || '无标题'
    content.value = d.note?.content || ''
    permission.value = d.permission || 'view'
    showContent.value = true
  } catch (e: any) {
    const msg = e.response?.data?.message || ''
    if (msg.includes('需要密码') || e.response?.status === 401) {
      needPassword.value = true
    } else if (msg.includes('待审核')) {
      error.value = '该分享链接待审核，暂时无法访问'
    } else if (msg.includes('未通过')) {
      error.value = '该分享未通过审核'
    } else if (msg.includes('过期')) {
      error.value = '该分享已过期'
    } else if (msg.includes('不存在') || e.response?.status === 404) {
      error.value = '分享不存在或已被删除'
    } else {
      error.value = '加载分享失败'
    }
  } finally {
    loading.value = false
  }
}

async function submitPassword() {
  if (!password.value) return
  await loadShare()
}

function copyLink() {
  navigator.clipboard.writeText(window.location.href)
  MessagePlugin.success('链接已复制到剪贴板')
}


</script>

<template>
  <div class="share-page">
    <div class="share-header">
      <div class="header-left">
        <h1 class="page-title">{{ title || '加载中...' }}</h1>
        <span v-if="permission === 'edit'" class="perm-badge">可编辑</span>
        <span v-else class="perm-badge">只读</span>
      </div>
      <div class="header-right">
        <t-button variant="outline" size="small" @click="copyLink">
          <t-icon name="copy" /> 复制链接
        </t-button>
      </div>
    </div>

    <div v-if="loading" class="share-loading">
      <t-loading size="large" text="加载中..." />
    </div>

    <div v-else-if="needPassword" class="password-form">
      <t-icon name="lock-on" size="48px" style="color: #c0c4cc" />
      <p>该分享需要密码访问</p>
      <div class="password-input-wrap">
        <t-input
          v-model="password"
          type="password"
          placeholder="请输入分享密码"
          @enter="submitPassword"
          clearable
        />
        <t-button @click="submitPassword" style="margin-left: 8px">查看</t-button>
      </div>
      <p v-if="error" class="error-text">{{ error }}</p>
    </div>

    <div v-else-if="error" class="error-state">
      <t-icon name="error-circle" size="48px" style="color: #e34d59" />
      <p>{{ error }}</p>
      <t-button variant="outline" @click="router.push({ name: 'login' })">
        返回登录
      </t-button>
    </div>

    <div v-else-if="showContent" class="share-content">
      <div class="content-body" v-html="renderContent(content)"></div>
    </div>
  </div>
</template>

<script lang="ts">
function renderContent(text: string): string {
  if (!text) return '<p class="muted">暂无内容</p>'
  // Basic markdown-to-HTML rendering
  let html = text
    // Code blocks
    .replace(/```(\w*)\n([\s\S]*?)```/g, '<pre><code>$2</code></pre>')
    // Inline code
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    // Headers
    .replace(/^### (.+)$/gm, '<h3>$1</h3>')
    .replace(/^## (.+)$/gm, '<h2>$1</h2>')
    .replace(/^# (.+)$/gm, '<h1>$1</h1>')
    // Bold
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    // Italic
    .replace(/\*(.+?)\*/g, '<em>$1</em>')
    // Line breaks
    .replace(/\n/g, '<br>')
  return html
}
</script>

<style scoped>
.share-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 24px;
  min-height: 100vh;
}

.share-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 16px;
  border-bottom: 1px solid #e5e6eb;
  margin-bottom: 24px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-title {
  font-size: 22px;
  font-weight: 700;
  color: #1d2129;
  margin: 0;
}

.perm-badge {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
  background: #e8f3ff;
  color: #1677ff;
  font-weight: 500;
}

.share-loading {
  display: flex;
  justify-content: center;
  padding: 60px 0;
}

.password-form {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 60px 0;
  color: #86909c;
}

.password-input-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
}

.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: #86909c;
  padding: 60px 0;
}

.error-text {
  color: #e34d59;
}

.share-content {
  background: #fff;
  border-radius: 8px;
  padding: 24px;
  border: 1px solid #e5e6eb;
}

.content-body :deep(h1) {
  font-size: 22px;
  font-weight: 700;
  margin: 16px 0 8px;
  color: #1d2129;
}

.content-body :deep(h2) {
  font-size: 18px;
  font-weight: 600;
  margin: 14px 0 6px;
  color: #1d2129;
}

.content-body :deep(h3) {
  font-size: 16px;
  font-weight: 600;
  margin: 12px 0 6px;
  color: #1d2129;
}

.content-body :deep(strong) {
  font-weight: 600;
  color: #1d2129;
}

.content-body :deep(em) {
  font-style: italic;
  color: #4e5969;
}

.content-body :deep(code) {
  background: #f7f8fa;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
  font-family: 'SF Mono', 'Fira Code', monospace;
  color: #e34d59;
}

.content-body :deep(pre) {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 16px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 12px 0;
}

.content-body :deep(pre code) {
  background: none;
  color: inherit;
  padding: 0;
  font-size: 13px;
}

.content-body :deep(p) {
  line-height: 1.8;
  color: #1d2129;
  margin: 8px 0;
}

.muted {
  color: #c0c4cc;
  font-style: italic;
}
</style>
