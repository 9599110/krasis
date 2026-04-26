<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getNoteVersions, restoreVersion } from '../api/notes'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'

const route = useRoute()
const router = useRouter()
const id = String(route.params.id ?? '')

const versions = ref<any[]>([])
const loading = ref(true)
const selectedVersion = ref<any>(null)

onMounted(() => { loadVersions() })

async function loadVersions() {
  loading.value = true
  try {
    const res = await getNoteVersions(id)
    const d = res.data?.data || res.data || {}
    versions.value = d.items || []
  } catch {
    MessagePlugin.error('加载版本历史失败')
  } finally {
    loading.value = false
  }
}

function showVersion(v: any) {
  selectedVersion.value = v
}

async function handleRestore(v: any) {
  DialogPlugin.confirm({
    header: '恢复版本',
    body: `确定要恢复到版本 ${v.version} 吗？当前内容将被覆盖。`,
    onConfirm: async () => {
      try {
        await restoreVersion(id, v.version, {
          title: v.title || '',
          content: v.content || '',
        })
        MessagePlugin.success('已恢复到该版本')
        router.push({ name: 'note-edit', params: { id } })
      } catch {
        MessagePlugin.error('恢复失败')
      }
    },
  })
}

function formatDate(dateStr: string) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleString('zh-CN')
}
</script>

<template>
  <div class="versions-page">
    <div class="header">
      <t-button variant="text" @click="router.push({ name: 'note-edit', params: { id } })">
        <t-icon name="arrow-left" />
        返回编辑
      </t-button>
      <h1 class="page-title">版本历史</h1>
    </div>

    <div class="version-list" v-loading="loading">
      <div v-if="versions.length === 0" class="empty-state">
        <t-icon name="history" size="48px" />
        <p>暂无版本历史</p>
      </div>
      <div
        v-for="v in versions"
        :key="v.version"
        class="version-card"
        :class="{ selected: selectedVersion?.version === v.version }"
        @click="showVersion(v)"
      >
        <div class="version-header">
          <span class="version-num">版本 {{ v.version }}</span>
          <span class="version-date">{{ formatDate(v.created_at) }}</span>
        </div>
        <div class="version-title">{{ v.title || '无标题' }}</div>
        <div class="version-preview">{{ v.content?.substring(0, 100) || '暂无内容' }}</div>
        <div class="version-actions" v-if="selectedVersion?.version === v.version" @click.stop>
          <t-button size="small" @click="handleRestore(v)">恢复此版本</t-button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.versions-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 24px;
}

.header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #1d2129;
  margin: 0;
}

.version-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: #86909c;
  padding: 60px 0;
}

.version-card {
  background: #fff;
  border: 1px solid #e5e6eb;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.version-card:hover {
  border-color: #1677ff;
}

.version-card.selected {
  border-color: #1677ff;
  box-shadow: 0 2px 8px rgba(22, 119, 255, 0.1);
}

.version-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.version-num {
  font-size: 14px;
  font-weight: 600;
  color: #1677ff;
}

.version-date {
  font-size: 12px;
  color: #c0c4cc;
}

.version-title {
  font-size: 15px;
  font-weight: 600;
  color: #1d2129;
  margin-bottom: 4px;
}

.version-preview {
  font-size: 13px;
  color: #86909c;
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.version-actions {
  margin-top: 12px;
}
</style>

