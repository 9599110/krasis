<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { listNotes, deleteNote } from '../api/notes'
import apiClient from '../api/client'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import {
  listFolderFiles,
  deleteFile as deleteFileAPI,
  presignUpload,
  confirmUpload,
  formatFileSize,
  isImageFile,
  getFileIcon,
  triggerDownload,
  type FolderFile,
} from '../api/files'

const router = useRouter()
const route = useRoute()

const notes = ref<any[]>([])
const loading = ref(true)

// File browsing state
const files = ref<FolderFile[]>([])
const filesLoading = ref(false)
const uploading = ref(false)
const previewFile = ref<FolderFile | null>(null)
const previewUrl = ref('')
const showPreview = ref(false)
const deleting = ref(false)
const previewLoading = ref(false)

const folderId = computed(() => String(route.query.folder || ''))

const isFolderView = computed(() => !!folderId.value)

onMounted(() => {
  loadNotes()
  if (folderId.value) loadFiles()
})

watch(() => route.query.folder, () => {
  loadNotes()
  if (folderId.value) loadFiles()
})

async function loadNotes() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: 1, size: 100 }
    if (folderId.value) params.folder_id = folderId.value
    const res = await listNotes(params)
    const responseData = res.data?.data
    const d = responseData && typeof responseData === 'object' && !Array.isArray(responseData) ? responseData : {}
    notes.value = d.items || []
  } catch {
    MessagePlugin.error('加载笔记失败')
  } finally {
    loading.value = false
  }
}

async function loadFiles() {
  if (!folderId.value) return
  filesLoading.value = true
  try {
    const res = await listFolderFiles(folderId.value)
    const responseData = res.data?.data
    const list = Array.isArray(responseData) ? responseData : []
    files.value = list
  } catch {
    files.value = []
  } finally {
    filesLoading.value = false
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

// File upload
async function handleUpload() {
  // Create a hidden file input
  const input = document.createElement('input')
  input.type = 'file'
  input.multiple = true
  input.onchange = async () => {
    const selectedFiles = input.files
    if (!selectedFiles || selectedFiles.length === 0) return

    uploading.value = true
    for (let i = 0; i < selectedFiles.length; i++) {
      const file = selectedFiles[i]
      try {
        await uploadSingleFile(file)
      } catch (err: any) {
        MessagePlugin.error(`上传 ${file.name} 失败: ${err.message || '未知错误'}`)
      }
    }
    uploading.value = false
    loadFiles()
  }
  input.click()
}

async function uploadSingleFile(file: File) {
  // Step 1: Get presigned URL
  const presignRes = await presignUpload({
    file_name: file.name,
    file_type: file.type || 'application/octet-stream',
    folder_id: folderId.value,
  })
  const presignData = presignRes.data?.data || presignRes.data as { file_id: string; upload_url: string }

  // Step 2: Upload file directly to MinIO
  const uploadResp = await fetch(presignData.upload_url, {
    method: 'PUT',
    body: file,
    headers: { 'Content-Type': file.type || 'application/octet-stream' },
  })
  if (!uploadResp.ok) {
    throw new Error(`上传到存储失败 (${uploadResp.status})`)
  }

  // Step 3: Confirm upload
  await confirmUpload({ file_id: presignData.file_id, folder_id: folderId.value })
}

// File preview
async function handleFileClick(file: FolderFile) {
  if (isImageFile(file.mime_type, file.file_name)) {
    // Preview image
    try {
      previewLoading.value = true
      const urlRes = await fetchFileUrl(file.id)
      if (urlRes) {
        previewUrl.value = urlRes
        previewFile.value = file
        showPreview.value = true
      } else {
        previewLoading.value = false
        MessagePlugin.error('获取预览地址失败')
      }
    } catch {
      previewLoading.value = false
      MessagePlugin.error('加载预览失败')
    }
  }
}

async function fetchFileUrl(fileId: string): Promise<string | null> {
  try {
    const res = await apiClient.get(`/files/${fileId}/url`)
    return res.data?.data?.url || null
  } catch {
    return null
  }
}

function onPreviewError() {
  previewLoading.value = false
  MessagePlugin.error('图片加载失败')
}

function onPreviewLoaded() {
  previewLoading.value = false
}

async function handleDeleteFile(file: FolderFile) {
  DialogPlugin.confirm({
    header: '删除文件',
    body: `确定要删除「${file.file_name}」吗？`,
    onConfirm: async () => {
      deleting.value = true
      try {
        await deleteFileAPI(file.id)
        MessagePlugin.success('文件已删除')
        loadFiles()
      } catch {
        MessagePlugin.error('删除文件失败')
      } finally {
        deleting.value = false
      }
    },
  })
}

async function handleDownloadFile(file: FolderFile) {
  try {
    await triggerDownload(file.id, file.file_name)
    MessagePlugin.success('开始下载文件')
  } catch {
    MessagePlugin.error('下载失败')
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString('zh-CN')
}
</script>

<template>
  <div class="note-list-page" v-loading="loading">
    <!-- Folder File Browser -->
    <div v-if="isFolderView" class="folder-file-section">
      <div class="section-header">
        <h2 class="section-title">
          <t-icon name="file" /> 附件
          <span class="file-count">{{ files.length }} 个文件</span>
        </h2>
        <div class="section-actions">
          <t-button
            variant="outline"
            size="small"
            :loading="uploading"
            :disabled="uploading"
            @click="handleUpload"
          >
            <t-icon name="upload" /> 上传
          </t-button>
        </div>
      </div>

      <div v-if="filesLoading" class="file-grid-loading">
        <t-loading />
      </div>

      <div v-else-if="files.length === 0" class="file-empty">
        <t-icon name="file" size="36px" style="color: #c0c4cc" />
        <p>文件夹暂无文件，点击上传添加文件或图片</p>
      </div>

      <div v-else class="file-grid">
        <div
          v-for="file in files"
          :key="file.id"
          class="file-item"
          @click="handleFileClick(file)"
          :title="file.file_name"
        >
          <!-- Preview thumbnail for images -->
          <div class="file-icon-wrapper">
            <template v-if="isImageFile(file.mime_type)">
              <img
                v-if="file.thumbnail_url"
                :src="file.thumbnail_url"
                class="file-thumb"
                alt=""
              />
              <t-icon v-else :name="getFileIcon(file.file_name, file.mime_type)" size="32px" class="file-icon" />
            </template>
            <t-icon v-else :name="getFileIcon(file.file_name, file.mime_type)" size="32px" class="file-icon" />
          </div>
          <div class="file-name" :title="file.file_name">{{ file.file_name }}</div>
          <div class="file-meta">
            <span class="file-size">{{ formatFileSize(file.size_bytes) }}</span>
          </div>
          <div class="file-actions" @click.stop>
            <t-button
              variant="text"
              size="small"
              @click="handleDownloadFile(file)"
              :title="'下载' + file.file_name"
            >
              <t-icon name="download" />
            </t-button>
            <t-button
              variant="text"
              size="small"
              theme="danger"
              @click="handleDeleteFile(file)"
            >
              <t-icon name="delete" />
            </t-button>
          </div>
        </div>
      </div>
    </div>

    <!-- Page Header -->
    <div class="page-header">
      <h1 class="page-title">
        {{ notes.length }} 篇笔记
      </h1>
      <t-button @click="router.push({ name: 'note-edit', params: { id: 'new' }, query: route.query.folder ? { folder: route.query.folder } : {} })">
        <t-icon name="add" />
        新建笔记
      </t-button>
    </div>

    <div v-if="notes.length === 0" class="empty-state">
      <t-icon name="file-copy" size="48px" />
      <p>还没有笔记</p>
      <t-button variant="outline" @click="router.push({ name: 'note-edit', params: { id: 'new' }, query: route.query.folder ? { folder: route.query.folder } : {} })">
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
        <div class="note-preview">
          <template v-if="note.is_encrypted">
            <t-icon name="lock-on" size="12px" style="margin-right: 4px" />
            <span style="color: #00a870">笔记已加密</span>
          </template>
          <template v-else>{{ note.content?.substring(0, 120).replace(/[#*`_~\[\]()]/g, '') || '暂无内容' }}</template>
        </div>
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

    <!-- Image Preview Dialog -->
    <t-dialog
      v-model:visible="showPreview"
      :header="previewFile?.file_name || '图片预览'"
      :footer="false"
      width="80vw"
      :close-btn="true"
      destroy-on-close
    >
      <div class="preview-container" v-if="previewUrl">
        <img
          :src="previewUrl"
          class="preview-image"
          :alt="previewFile?.file_name"
          @error="onPreviewError"
          @load="onPreviewLoaded"
        />
        <div v-if="previewLoading" class="preview-loading">
          <t-loading size="32px" />
          <p>图片加载中...</p>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<style scoped>
.note-list-page {
  padding: 24px;
  max-width: 1200px;
}

/* File section */
.folder-file-section {
  background: #fff;
  border: 1px solid #e5e6eb;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 24px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: #1d2129;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.file-count {
  font-size: 12px;
  font-weight: 400;
  color: #86909c;
  margin-left: 4px;
}

.section-actions {
  display: flex;
  gap: 8px;
}

.file-grid-loading {
  display: flex;
  justify-content: center;
  padding: 24px;
}

.file-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: #86909c;
  padding: 24px 0;
  font-size: 13px;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 12px;
}

.file-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px 8px 8px;
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
  position: relative;
  min-height: 110px;
}

.file-item:hover {
  background: #f5f7fa;
  border-color: #d0d5dd;
}

.file-item:hover .file-actions {
  opacity: 1;
}

.file-icon-wrapper {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 6px;
}

.file-icon {
  color: #86909c;
}

.file-thumb {
  width: 48px;
  height: 48px;
  object-fit: cover;
  border-radius: 4px;
}

.file-name {
  font-size: 12px;
  color: #4e5969;
  text-align: center;
  word-break: break-all;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.3;
  max-width: 100%;
}

.file-meta {
  font-size: 11px;
  color: #c0c4cc;
  margin-top: 2px;
}

.file-actions {
  position: absolute;
  top: 2px;
  right: 2px;
  opacity: 0;
  transition: opacity 0.15s;
}

/* Page header */
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
  contain: layout style;
  content-visibility: auto;
  contain-intrinsic-size: 120px;
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

/* Preview dialog */
.preview-container {
  display: flex;
  justify-content: center;
  align-items: center;
  max-height: 70vh;
  overflow: auto;
}

.preview-image {
  max-width: 100%;
  max-height: 65vh;
  object-fit: contain;
  border-radius: 4px;
}
</style>
