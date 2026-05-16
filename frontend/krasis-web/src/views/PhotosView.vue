<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import apiClient from '../api/client'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import {
  presignUpload,
  confirmUpload,
  deleteFile as deleteFileAPI,
  formatFileSize,
  isImageFile,
  triggerDownload,
  type FolderFile,
} from '../api/files'

const router = useRouter()

const photos = ref<FolderFile[]>([])
const loading = ref(true)
const uploading = ref(false)
const previewUrl = ref('')
const previewFile = ref<FolderFile | null>(null)
const showPreview = ref(false)
const previewLoading = ref(false)
const page = ref(1)
const total = ref(0)
const size = 50

async function loadPhotos() {
  loading.value = true
  try {
    const res = await apiClient.get('/files/images', { params: { page: page.value, size } })
    const d = res.data?.data || res.data || {}
    photos.value = d.items || []
    total.value = d.total || 0
  } catch {
    photos.value = []
  } finally {
    loading.value = false
  }
}

async function handleUpload() {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = 'image/*'
  input.multiple = true
  input.onchange = async () => {
    const selectedFiles = input.files
    if (!selectedFiles || selectedFiles.length === 0) return

    uploading.value = true
    for (let i = 0; i < selectedFiles.length; i++) {
      const file = selectedFiles[i]
      try {
        // Step 1: Get presigned URL
        const presignRes = await presignUpload({
          file_name: file.name,
          file_type: file.type || 'image/jpeg',
        })
        const pd = presignRes.data?.data || presignRes.data as { file_id: string; upload_url: string }

        // Step 2: Upload file directly to MinIO
        const uploadResp = await fetch(pd.upload_url, {
          method: 'PUT',
          body: file,
          headers: { 'Content-Type': file.type || 'image/jpeg' },
        })
        if (!uploadResp.ok) {
          throw new Error(`上传失败 (${uploadResp.status})`)
        }

        // Step 3: Confirm upload
        await confirmUpload({ file_id: pd.file_id })
      } catch (err: any) {
        MessagePlugin.error(`上传 ${file.name} 失败: ${err.message || '未知错误'}`)
      }
    }
    uploading.value = false
    loadPhotos()
  }
  input.click()
}

async function handlePreview(photo: FolderFile) {
  previewFile.value = photo
  showPreview.value = true
  previewLoading.value = true
  try {
    const res = await apiClient.get(`/files/${photo.id}/url`)
    previewUrl.value = res.data?.data?.url || ''
  } catch {
    MessagePlugin.error('获取预览地址失败')
  } finally {
    previewLoading.value = false
  }
}

function onPreviewError() {
  previewLoading.value = false
  MessagePlugin.error('图片加载失败')
}

function onPreviewLoaded() {
  previewLoading.value = false
}

async function handleDelete(photo: FolderFile) {
  DialogPlugin.confirm({
    header: '删除照片',
    body: `确定要删除「${photo.file_name}」吗？`,
    onConfirm: async () => {
      try {
        await deleteFileAPI(photo.id)
        MessagePlugin.success('已删除')
        loadPhotos()
      } catch {
        MessagePlugin.error('删除失败')
      }
    },
  })
}

async function handleDownload(photo: FolderFile) {
  try {
    await triggerDownload(photo.id, photo.file_name)
    MessagePlugin.success('开始下载')
  } catch {
    MessagePlugin.error('下载失败')
  }
}

onMounted(() => loadPhotos())
</script>

<template>
  <div class="photos-page" v-loading="loading">
    <div class="page-header">
      <h1 class="page-title">
        照片
        <span class="photo-count">{{ total }} 张</span>
      </h1>
      <t-button
        theme="primary"
        :loading="uploading"
        :disabled="uploading"
        @click="handleUpload"
      >
        <t-icon name="upload" /> 上传照片
      </t-button>
    </div>

    <div v-if="photos.length === 0 && !loading" class="empty-state">
      <t-icon name="image" size="48px" style="color: #c0c4cc" />
      <p>还没有照片，点击上传添加相册照片</p>
    </div>

    <div v-else class="photo-grid">
      <div
        v-for="photo in photos"
        :key="photo.id"
        class="photo-card"
        @click="handlePreview(photo)"
      >
        <div class="photo-thumb">
          <img
            v-if="photo.thumbnail_url"
            :src="photo.thumbnail_url"
            :alt="photo.file_name"
            class="thumb-img"
          />
          <t-icon v-else name="image" size="48px" class="thumb-placeholder" />
        </div>
        <div class="photo-info">
          <div class="photo-name" :title="photo.file_name">{{ photo.file_name }}</div>
          <div class="photo-meta">
            <span>{{ formatFileSize(photo.size_bytes) }}</span>
            <span class="photo-actions" @click.stop>
              <t-button variant="text" size="small" @click="handleDownload(photo)" title="下载">
                <t-icon name="download" />
              </t-button>
              <t-button variant="text" size="small" theme="danger" @click="handleDelete(photo)" title="删除">
                <t-icon name="delete" />
              </t-button>
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Preview Dialog -->
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
          <p>加载中...</p>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<style scoped>
.photos-page {
  max-width: 1400px;
  margin: 0 auto;
  padding: 16px;
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
  display: flex;
  align-items: center;
  gap: 8px;
}

.photo-count {
  font-size: 13px;
  font-weight: 400;
  color: #86909c;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 0;
  color: #86909c;
  font-size: 14px;
}

.photo-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
}

.photo-card {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  transition: box-shadow 0.2s, border-color 0.2s;
  background: #fff;
}

.photo-card:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border-color: #d0d5dd;
}

.photo-thumb {
  width: 100%;
  height: 140px;
  background: #f5f5f5;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.thumb-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.thumb-placeholder {
  color: #c0c4cc;
}

.photo-info {
  padding: 8px 10px;
}

.photo-name {
  font-size: 12px;
  color: #4e5969;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.photo-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 4px;
  font-size: 11px;
  color: #c0c4cc;
}

.photo-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

.preview-container {
  display: flex;
  justify-content: center;
  align-items: center;
  max-height: 70vh;
  overflow: auto;
  position: relative;
}

.preview-image {
  max-width: 100%;
  max-height: 65vh;
  object-fit: contain;
  border-radius: 4px;
}

.preview-loading {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: #86909c;
}
</style>
