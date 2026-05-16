import apiClient from './client'

export type PresignResult = {
  file_id: string
  upload_url: string
  expires_in: number
}

export type FolderFile = {
  id: string
  note_id: string | null
  folder_id: string | null
  user_id: string
  file_name: string
  file_type: string | null
  mime_type: string | null
  storage_path: string
  bucket: string
  size_bytes: number | null
  width: number | null
  height: number | null
  thumbnail_url: string | null
  metadata: Record<string, any> | null
  status: number
  is_hidden?: boolean
  created_at: string
}

export async function presignUpload(params: { file_name: string; file_type: string; note_id?: string; folder_id?: string }) {
  return apiClient.get('/files/presign', { params })
}

export async function confirmUpload(body: { file_id: string; note_id?: string; folder_id?: string; metadata?: Record<string, any> }) {
  return apiClient.post('/files/confirm', body)
}

export async function listFolderFiles(folderId: string) {
  return apiClient.get(`/files/folder/${folderId}`)
}

export async function deleteFile(fileId: string) {
  return apiClient.delete(`/files/${fileId}`)
}

/**
 * Get a download URL for a file (forces browser download)
 */
export async function downloadFile(fileId: string) {
  return apiClient.get(`/files/${fileId}/download`)
}

/** Trigger browser download by creating a temporary anchor element */
export async function triggerDownload(fileId: string, fileName: string) {
  try {
    const res = await downloadFile(fileId)
    const url = res.data?.data?.url
    if (!url) {
      throw new Error('获取下载链接失败')
    }
    const a = document.createElement('a')
    a.href = url
    a.download = fileName
    a.target = '_blank'
    a.rel = 'noopener noreferrer'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  } catch (err: any) {
    throw new Error(err.message || '下载失败')
  }
}

/** Format file size in human-readable form */
export function formatFileSize(bytes: number | null | undefined): string {
  if (!bytes || bytes <= 0) return ''
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

/** Determine if a mime type is an image (also checks extension for fallback) */
export function isImageFile(mimeType: string | null | undefined, fileName?: string): boolean {
  if (mimeType && mimeType.startsWith('image/')) return true
  // Fallback: check file extension for files with null mime_type
  if (fileName) {
    const ext = fileName.split('.').pop()?.toLowerCase() || ''
    const imgExts = ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico', 'avif', 'tiff', 'tif']
    return imgExts.includes(ext)
  }
  return false
}

/** Get a display-friendly file type icon name */
export function getFileIcon(fileName: string, mimeType: string | null | undefined): string {
  if (isImageFile(mimeType)) return 'image'
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  const iconMap: Record<string, string> = {
    pdf: 'file-pdf',
    doc: 'file-word',
    docx: 'file-word',
    xls: 'file-excel',
    xlsx: 'file-excel',
    ppt: 'file-ppt',
    pptx: 'file-ppt',
    zip: 'file-zip',
    rar: 'file-zip',
    '7z': 'file-zip',
    tar: 'file-zip',
    gz: 'file-zip',
    mp4: 'video',
    avi: 'video',
    mov: 'video',
    mp3: 'audio',
    wav: 'audio',
    flac: 'audio',
    txt: 'file-text',
    json: 'file-code',
    js: 'file-code',
    ts: 'file-code',
    py: 'file-code',
    go: 'file-code',
    html: 'file-code',
    css: 'file-code',
    md: 'file-text',
    csv: 'file-excel',
  }
  return iconMap[ext] || 'file'
}


