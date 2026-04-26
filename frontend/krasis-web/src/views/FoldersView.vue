<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listFolders, createFolder, updateFolder, deleteFolder } from '../api/notes'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'

const folders = ref<any[]>([])
const loading = ref(true)
const showDialog = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const formId = ref('')
const formName = ref('')
const formColor = ref('#4CAF50')

const colors = ['#4CAF50', '#2196F3', '#FF9800', '#9C27B0', '#F44336', '#607D8B', '#00BCD4', '#FFC107']

onMounted(() => loadFolders())

async function loadFolders() {
  loading.value = true
  try {
    const res = await listFolders()
    const d = res.data?.data || res.data || {}
    folders.value = d.items || []
  } catch {
    MessagePlugin.error('加载文件夹失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  dialogMode.value = 'create'
  formId.value = ''
  formName.value = ''
  formColor.value = '#4CAF50'
  showDialog.value = true
}

function openEdit(folder: any) {
  dialogMode.value = 'edit'
  formId.value = folder.id
  formName.value = folder.name
  formColor.value = folder.color || '#4CAF50'
  showDialog.value = true
}

async function submit() {
  if (!formName.value.trim()) {
    MessagePlugin.warning('请输入文件夹名称')
    return
  }
  try {
    if (dialogMode.value === 'create') {
      await createFolder({ name: formName.value.trim() })
      MessagePlugin.success('文件夹已创建')
    } else {
      await updateFolder(formId.value, { name: formName.value.trim() })
      MessagePlugin.success('文件夹已更新')
    }
    showDialog.value = false
    loadFolders()
  } catch {
    MessagePlugin.error('操作失败')
  }
}

function handleDelete(folder: any) {
  DialogPlugin.confirm({
    header: '删除文件夹',
    body: `确定要删除文件夹「${folder.name}」吗？`,
    onConfirm: async () => {
      try {
        await deleteFolder(folder.id)
        MessagePlugin.success('已删除')
        loadFolders()
      } catch {
        MessagePlugin.error('删除失败')
      }
    },
  })
}

function formatDate(dateStr: string) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString('zh-CN')
}
</script>

<template>
  <div class="folders-page" v-loading="loading">
    <div class="page-header">
      <h1 class="page-title">文件夹管理</h1>
      <t-button @click="openCreate">
        <t-icon name="add" /> 新建文件夹
      </t-button>
    </div>

    <div v-if="folders.length === 0" class="empty-state">
      <t-icon name="folder" size="48px" />
      <p>暂无文件夹</p>
      <t-button variant="outline" @click="openCreate">
        创建第一个文件夹
      </t-button>
    </div>

    <div class="folder-grid" v-else>
      <div v-for="folder in folders" :key="folder.id" class="folder-card">
        <div class="card-top">
          <t-icon name="folder" size="32px" :style="{ color: formColor === folder.color ? folder.color : '#4CAF50' }" />
          <div class="card-actions">
            <t-button variant="text" size="small" @click="openEdit(folder)">
              <t-icon name="edit" />
            </t-button>
            <t-button variant="text" size="small" theme="danger" @click="handleDelete(folder)">
              <t-icon name="delete" />
            </t-button>
          </div>
        </div>
        <div class="folder-name">{{ folder.name }}</div>
        <div class="folder-date">创建于 {{ formatDate(folder.created_at) }}</div>
      </div>
    </div>

    <t-dialog
      v-model:visible="showDialog"
      :header="dialogMode === 'create' ? '新建文件夹' : '编辑文件夹'"
      :footer="false"
      width="400px"
    >
      <div class="form">
        <div class="form-group">
          <label>名称</label>
          <t-input v-model="formName" placeholder="文件夹名称" />
        </div>
        <div class="form-group">
          <label>颜色</label>
          <div class="color-picker">
            <div
              v-for="c in colors"
              :key="c"
              class="color-swatch"
              :class="{ active: c === formColor }"
              :style="{ background: c }"
              @click="formColor = c"
            >
              <t-icon v-if="c === formColor" name="check" size="14px" style="color: #fff" />
            </div>
          </div>
        </div>
        <div class="form-actions">
          <t-button variant="outline" @click="showDialog = false">取消</t-button>
          <t-button @click="submit">{{ dialogMode === 'create' ? '创建' : '保存' }}</t-button>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<style scoped>
.folders-page {
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

.folder-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
}

.folder-card {
  background: #fff;
  border: 1px solid #e5e6eb;
  border-radius: 8px;
  padding: 16px;
  transition: box-shadow 0.2s;
}

.folder-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.card-actions {
  display: flex;
  gap: 4px;
}

.folder-name {
  font-size: 15px;
  font-weight: 600;
  color: #1d2129;
  margin-bottom: 6px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.folder-date {
  font-size: 12px;
  color: #c0c4cc;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 13px;
  font-weight: 500;
  color: #4e5969;
}

.color-picker {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.color-swatch {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 3px solid transparent;
  transition: border-color 0.15s;
}

.color-swatch.active {
  border-color: #fff;
  box-shadow: 0 0 0 2px #1677ff;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 8px;
}
</style>
