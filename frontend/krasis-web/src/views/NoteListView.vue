<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { listNotes, deleteNote } from '../api/notes'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'

const router = useRouter()
const route = useRoute()

const notes = ref<any[]>([])
const loading = ref(true)

const folderId = computed(() => String(route.query.folder || ''))

onMounted(() => loadNotes())

async function loadNotes() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: 1, size: 100 }
    if (folderId.value) params.folder_id = folderId.value
    const res = await listNotes(params)
    const d = res.data?.data || res.data || {}
    notes.value = d.items || []
  } catch {
    MessagePlugin.error('加载笔记失败')
  } finally {
    loading.value = false
  }
}

async function handleDelete(note: any) {
  DialogPlugin.confirm({
    header: '删除笔记',
    body: `确定要删除「${note.title}」吗？`,
    onConfirm: async () => {
      try {
        await deleteNote(note.id)
        MessagePlugin.success('已删除')
        loadNotes()
      } catch {
        MessagePlugin.error('删除失败')
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
  <div class="note-list-page" v-loading="loading">
    <div class="page-header">
      <h1 class="page-title">
        {{ notes.length }} 篇笔记
      </h1>
      <t-button @click="router.push({ name: 'note-edit', params: { id: 'new' } })">
        <t-icon name="add" />
        新建笔记
      </t-button>
    </div>

    <div v-if="notes.length === 0" class="empty-state">
      <t-icon name="file-copy" size="48px" />
      <p>还没有笔记</p>
      <t-button variant="outline" @click="router.push({ name: 'note-edit', params: { id: 'new' } })">
        创建第一篇笔记
      </t-button>
    </div>

    <div class="note-grid">
      <div
        v-for="note in notes"
        :key="note.id"
        class="note-card"
        @click="router.push({ name: 'note-edit', params: { id: note.id } })"
      >
        <div class="note-title">{{ note.title || '无标题' }}</div>
        <div class="note-preview">{{ note.content?.substring(0, 120).replace(/[#*`_~\[\]()]/g, '') || '暂无内容' }}</div>
        <div class="note-footer">
          <span class="note-date">{{ formatDate(note.updated_at) }}</span>
          <span class="note-actions" @click.stop>
            <t-dropdown>
              <t-button variant="text" shape="square" size="small">
                <t-icon name="ellipsis" />
              </t-button>
              <t-dropdown-menu>
                <t-dropdown-item @click="router.push({ name: 'note-versions', params: { id: note.id } })">
                  <t-icon name="history" /> 版本历史
                </t-dropdown-item>
                <t-dropdown-item @click="handleDelete(note)" theme="error">
                  <t-icon name="delete" /> 删除
                </t-dropdown-item>
              </t-dropdown-menu>
            </t-dropdown>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.note-list-page {
  padding: 24px;
  max-width: 1200px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #1d2129;
  margin: 0;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: #86909c;
  padding: 60px 0;
}

.note-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.note-card {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  border: 1px solid #e5e6eb;
  transition: box-shadow 0.2s;
}

.note-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.note-title {
  font-size: 15px;
  font-weight: 600;
  color: #1d2129;
  margin-bottom: 8px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.note-preview {
  font-size: 13px;
  color: #86909c;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  margin-bottom: 12px;
}

.note-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.note-date {
  font-size: 12px;
  color: #c0c4cc;
}
</style>
