<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import apiClient from '../api/client'
import { MessagePlugin } from 'tdesign-vue-next'

const router = useRouter()

const keyword = ref('')
const results = ref<any[]>([])
const loading = ref(false)
const searched = ref(false)

async function handleSearch() {
  if (!keyword.value.trim()) return
  loading.value = true
  searched.value = true
  try {
    const res = await apiClient.get('/search', { params: { q: keyword.value } })
    const d = res.data?.data || res.data || {}
    results.value = d.results || []
  } catch {
    MessagePlugin.error('搜索失败')
  } finally {
    loading.value = false
  }
}

function openNote(id: string) {
  router.push({ name: 'note-edit', params: { id } })
}

function highlightText(text: string) {
  if (!keyword.value) return text
  const escaped = keyword.value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const re = new RegExp(`(${escaped})`, 'gi')
  return text.replace(re, '<mark>$1</mark>')
}

function formatDate(dateStr: string) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString('zh-CN')
}
</script>

<template>
  <div class="search-page">
    <div class="search-bar">
      <t-input
        v-model="keyword"
        placeholder="搜索笔记标题或内容..."
        size="large"
        @enter="handleSearch"
        clearable
      >
        <template #suffix>
          <t-button @click="handleSearch" shape="square">
            <t-icon name="search" />
          </t-button>
        </template>
      </t-input>
    </div>

    <div class="results" v-loading="loading">
      <div v-if="searched && results.length === 0" class="empty-state">
        <t-icon name="search" size="48px" />
        <p>没有找到相关结果</p>
      </div>
      <div v-for="item in results" :key="item.id" class="result-item" @click="openNote(item.id)">
        <div class="result-title" v-html="highlightText(item.title)"></div>
        <div class="result-content" v-html="highlightText(item.content?.substring(0, 200) || '')"></div>
        <div class="result-meta">
          <span>{{ formatDate(item.updated_at) }}</span>
          <span v-if="item.score" class="score">相关度: {{ (item.score * 100).toFixed(0) }}%</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.search-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 24px;
}

.search-bar {
  margin-bottom: 24px;
}

.results {
  min-height: 200px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: #86909c;
  padding: 60px 0;
}

.result-item {
  padding: 16px;
  border-radius: 8px;
  cursor: pointer;
  border: 1px solid #e5e6eb;
  margin-bottom: 12px;
  transition: box-shadow 0.2s;
}

.result-item:hover {
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
}

.result-title {
  font-size: 16px;
  font-weight: 600;
  color: #1d2129;
  margin-bottom: 8px;
}

.result-title :deep(mark) {
  background: #fff3cd;
  padding: 0 2px;
  border-radius: 2px;
}

.result-content {
  font-size: 14px;
  color: #4e5969;
  line-height: 1.6;
  margin-bottom: 8px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.result-content :deep(mark) {
  background: #fff3cd;
  padding: 0 2px;
  border-radius: 2px;
}

.result-meta {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: #c0c4cc;
}

.score {
  color: #1677ff;
}
</style>

