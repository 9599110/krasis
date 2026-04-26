<script setup lang="ts">
import { ref, computed, onMounted, watch, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getNote, updateNote, createNote } from '../api/notes'
import apiClient from '../api/client'
import { MessagePlugin } from 'tdesign-vue-next'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import { Markdown } from 'tiptap-markdown'
import Underline from '@tiptap/extension-underline'
import { TextStyle } from '@tiptap/extension-text-style'
import Color from '@tiptap/extension-color'
import Highlight from '@tiptap/extension-highlight'
import TextAlign from '@tiptap/extension-text-align'
import Image from '@tiptap/extension-image'
import FontFamily from '@tiptap/extension-font-family'
import { presignUpload, confirmUpload } from '../api/files'

const route = useRoute()
const router = useRouter()
const id = computed(() => String(route.params.id ?? ''))

const title = ref('')
const loading = ref(false)
const saving = ref(false)
const version = ref(1)
const saveTimer = ref<any>(null)
const isNew = computed(() => id.value === 'new')

// Share dialog state
const showShareDialog = ref(false)
const shareLoading = ref(false)
const shareLink = ref('')
const sharePassword = ref('')
const shareExpiresDays = ref<number | null>(null)

const TITLE_MAX_LEN = 80

function deriveTitleFromText(text: string): string {
  const t = (text || '').replace(/\u0000/g, '').trim()
  if (!t) return '无标题笔记'
  const firstNonEmptyLine = t
    .split(/\r?\n/)
    .map((s) => s.trim())
    .find((s) => !!s) || ''
  const firstSentence = firstNonEmptyLine.split(/[。！？!?]/)[0]?.trim() || firstNonEmptyLine
  const normalized = firstSentence.replace(/\s+/g, ' ').trim()
  return normalized.length > TITLE_MAX_LEN ? normalized.slice(0, TITLE_MAX_LEN) : normalized
}

onMounted(() => {
  isNew.value ? (title.value = '无标题笔记') : loadNote()
})

const editor = useEditor({
  extensions: [
    StarterKit,
    Underline,
    TextStyle,
    Color,
    Highlight,
    FontFamily,
    TextAlign.configure({ types: ['heading', 'paragraph'] }),
    Image.configure({ inline: false }),
    Markdown.configure({
      // Requirement: pasted markdown should be stored as-is (no transform)
      transformPastedText: false,
      transformCopiedText: false,
    }),
  ],
  content: '',
  editorProps: {
    attributes: {
      class: 'wysiwyg-editor',
    },
  },
  onUpdate: () => {
    scheduleAutoSave()
  },
})

const fontFamily = ref('默认字体')
const fontSize = ref(14)
const paragraphStyle = ref<'正文' | 'H1' | 'H2' | 'H3'>('正文')

function applyFontFamily(val: string) {
  fontFamily.value = val
  if (val === '默认字体') {
    editor.value?.chain().focus().unsetFontFamily().run()
    return
  }
  editor.value?.chain().focus().setFontFamily(val).run()
}

function applyFontSize(val: number) {
  fontSize.value = val
  editor.value?.chain().focus().setMark('textStyle', { fontSize: `${val}px` }).run()
}

function applyParagraphStyle(val: '正文' | 'H1' | 'H2' | 'H3') {
  paragraphStyle.value = val
  const e = editor.value
  if (!e) return
  if (val === '正文') e.chain().focus().setParagraph().run()
  if (val === 'H1') e.chain().focus().toggleHeading({ level: 1 }).run()
  if (val === 'H2') e.chain().focus().toggleHeading({ level: 2 }).run()
  if (val === 'H3') e.chain().focus().toggleHeading({ level: 3 }).run()
}

function pickTextColor() {
  const c = window.prompt('输入文字颜色（如 #1677ff）')
  if (!c) return
  editor.value?.chain().focus().setColor(c).run()
}

function pickHighlight() {
  const c = window.prompt('输入高亮颜色（如 #fff1b8），留空取消高亮')
  if (!c) {
    editor.value?.chain().focus().unsetHighlight().run()
    return
  }
  editor.value?.chain().focus().setHighlight({ color: c }).run()
}

function insertImage() {
  const url = window.prompt('输入图片 URL')
  if (!url) return
  editor.value?.chain().focus().setImage({ src: url }).run()
}

function setAlign(align: 'left' | 'center' | 'right' | 'justify') {
  editor.value?.chain().focus().setTextAlign(align).run()
}

function pad2(n: number) {
  return String(n).padStart(2, '0')
}

function buildVoiceFileName(ext: string) {
  const d = new Date()
  const yyyy = d.getFullYear()
  const mm = pad2(d.getMonth() + 1)
  const dd = pad2(d.getDate())
  const hh = pad2(d.getHours())
  const mi = pad2(d.getMinutes())
  const ss = pad2(d.getSeconds())
  return `${yyyy}${mm}${dd}_${hh}${mi}${ss}.${ext}`
}

const isRecording = ref(false)
let mediaRecorder: MediaRecorder | null = null
let recordedChunks: BlobPart[] = []

async function toggleVoiceInput() {
  if (isRecording.value) {
    mediaRecorder?.stop()
    return
  }

  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    recordedChunks = []
    mediaRecorder = new MediaRecorder(stream)
    mediaRecorder.ondataavailable = (e) => {
      if (e.data && e.data.size > 0) recordedChunks.push(e.data)
    }
    mediaRecorder.onstop = async () => {
      try {
        const blob = new Blob(recordedChunks, { type: mediaRecorder?.mimeType || 'audio/webm' })
        stream.getTracks().forEach((t) => t.stop())
        isRecording.value = false

        const ext = blob.type.includes('ogg') ? 'ogg' : 'webm'
        const fileName = buildVoiceFileName(ext)
        const noteId = isNew.value ? undefined : id.value
        const presignRes = await presignUpload({ file_name: fileName, file_type: 'audio', note_id: noteId })
        const presign = presignRes.data?.data || presignRes.data

        await fetch(presign.upload_url, { method: 'PUT', body: blob })
        await confirmUpload({ file_id: presign.file_id, note_id: noteId })

        editor.value?.chain().focus().insertContent(`\n[语音 ${fileName}](file:${presign.file_id})\n`).run()
        MessagePlugin.success('语音已上传')
      } catch (e: any) {
        isRecording.value = false
        MessagePlugin.error('语音上传失败: ' + (e?.message || ''))
      }
    }
    mediaRecorder.start()
    isRecording.value = true
    MessagePlugin.success('开始录音，再次点击停止')
  } catch (e: any) {
    MessagePlugin.error('无法开始录音: ' + (e?.message || ''))
  }
}

onBeforeUnmount(() => {
  editor.value?.destroy()
})

async function loadNote() {
  loading.value = true
  try {
    const res = await getNote(id.value)
    const d = res.data?.data || res.data || {}
    title.value = d.title || ''
    editor.value?.commands.setContent(d.content || '')
    version.value = d.version || 1
  } catch {
    MessagePlugin.error('加载笔记失败')
    router.push({ name: 'notes' })
  } finally {
    loading.value = false
  }
}

function scheduleAutoSave() {
  if (saveTimer.value) clearTimeout(saveTimer.value)
  saveTimer.value = setTimeout(doSave, 2000)
}

async function doSave() {
  if (saving.value) return
  saving.value = true
  try {
    const markdown = (editor.value as any)?.storage?.markdown?.getMarkdown?.() ?? ''
    const plainText = editor.value?.getText?.() ?? ''
    if (!title.value.trim()) {
      title.value = deriveTitleFromText(plainText)
    }
    if (isNew.value) {
      const res = await createNote({ title: title.value, content: markdown })
      const d = res.data?.data || res.data || {}
      router.replace({ name: 'note-edit', params: { id: d.id } })
    } else {
      const res = await updateNote(
        id.value,
        {
          title: title.value,
          content: markdown,
        },
        version.value,
      )
      const d = res.data?.data || res.data || {}
      version.value = d.version || version.value
    }
    MessagePlugin.success({ content: '已保存', duration: 1000 })
  } catch (e: any) {
    if (e.isVersionConflict) {
      MessagePlugin.warning('内容已被其他设备修改，请刷新后重试')
    } else {
      MessagePlugin.error('保存失败')
    }
  } finally {
    saving.value = false
  }
}

async function handleSave() {
  if (saveTimer.value) clearTimeout(saveTimer.value)
  await doSave()
}

watch([title], () => {
  scheduleAutoSave()
})

async function openShareDialog() {
  showShareDialog.value = true
  shareLink.value = ''
  sharePassword.value = ''
  shareExpiresDays.value = null
  await checkExistingShare()
}

async function createShare() {
  if (isNew.value) {
    MessagePlugin.warning('请先保存笔记后再分享')
    return
  }
  shareLoading.value = true
  try {
    const body: Record<string, any> = {}
    if (sharePassword.value) body.password = sharePassword.value
    if (shareExpiresDays.value) {
      const d = new Date()
      d.setDate(d.getDate() + shareExpiresDays.value)
      body.expires_at = d.toISOString()
    }
    const res = await apiClient.post(`/notes/${id.value}/share`, body)
    const d = res.data?.data || res.data || {}
    shareLink.value = `${window.location.origin}/share/${d.token}`
    MessagePlugin.success('分享链接已创建')
  } catch {
    MessagePlugin.error('创建分享链接失败')
  } finally {
    shareLoading.value = false
  }
}

function copyShareLink() {
  navigator.clipboard.writeText(shareLink.value)
  MessagePlugin.success('链接已复制到剪贴板')
}

async function checkExistingShare() {
  try {
    const res = await apiClient.get(`/notes/${id.value}/share`)
    const d = res.data?.data || res.data || {}
    if (d.token) {
      shareLink.value = `${window.location.origin}/share/${d.token}`
    }
  } catch {
    // no share exists
  }
}

async function deleteShare() {
  try {
    await apiClient.delete(`/notes/${id.value}/share`)
    shareLink.value = ''
    MessagePlugin.success('分享链接已取消')
  } catch {
    MessagePlugin.error('取消分享失败')
  }
}

function toggleBold() {
  editor.value?.chain().focus().toggleBold().run()
}
function toggleItalic() {
  editor.value?.chain().focus().toggleItalic().run()
}
function toggleStrike() {
  editor.value?.chain().focus().toggleStrike().run()
}
function toggleBulletList() {
  editor.value?.chain().focus().toggleBulletList().run()
}
function toggleOrderedList() {
  editor.value?.chain().focus().toggleOrderedList().run()
}
function toggleCodeBlock() {
  editor.value?.chain().focus().toggleCodeBlock().run()
}
</script>

<template>
  <div class="note-editor" v-loading="loading">
    <!-- Header bar -->
    <div class="editor-header">
      <div class="header-left">
        <t-button variant="text" @click="router.push({ name: 'notes' })">
          <t-icon name="arrow-left" />
        </t-button>
        <t-input
          v-model="title"
          class="title-input-inline"
          placeholder="笔记标题"
          clearable
        />
      </div>
      <div class="header-right">
        <span class="save-status" v-if="saving">保存中...</span>
        <span class="save-status" v-else-if="!isNew">已保存</span>
        <t-button size="small" @click="handleSave" :loading="saving">保存</t-button>
        <t-button
          size="small"
          variant="text"
          @click="openShareDialog"
          :disabled="isNew"
        >
          <t-icon name="share" /> 分享
        </t-button>
        <t-button
          size="small"
          variant="text"
          @click="router.push({ name: 'note-versions', params: { id } })"
        >
          <t-icon name="history" />
        </t-button>
      </div>
    </div>

    <!-- WYSIWYG toolbar (match screenshot layout) -->
    <div class="format-toolbar">
      <t-tooltip content="插入图片">
        <t-button size="small" variant="text" @click="insertImage" :disabled="!editor">
          <t-icon name="image" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip :content="isRecording ? '停止语音输入' : '语音输入'">
        <t-button size="small" variant="text" @click="toggleVoiceInput" :disabled="!editor">
          <t-icon :name="isRecording ? 'stop-circle' : 'mic'" size="16px" />
        </t-button>
      </t-tooltip>
      <t-button size="small" variant="text" @click="insertImage" :disabled="!editor">
        插入 <t-icon name="chevron-down" size="16px" />
      </t-button>

      <t-divider layout="vertical" style="margin: 0 6px" />

      <t-select
        v-model="paragraphStyle"
        size="small"
        style="width: 86px"
        :disabled="!editor"
        @change="(v: any) => applyParagraphStyle(v as any)"
      >
        <t-option value="正文" label="正文" />
        <t-option value="H1" label="标题 1" />
        <t-option value="H2" label="标题 2" />
        <t-option value="H3" label="标题 3" />
      </t-select>

      <t-select
        v-model="fontFamily"
        size="small"
        style="width: 110px"
        :disabled="!editor"
        @change="(v: any) => applyFontFamily(String(v))"
      >
        <t-option value="默认字体" label="默认字体" />
        <t-option value="system-ui" label="系统字体" />
        <t-option value="serif" label="Serif" />
        <t-option value="monospace" label="Monospace" />
      </t-select>

      <t-select
        v-model="fontSize"
        size="small"
        style="width: 70px"
        :disabled="!editor"
        @change="(v: any) => applyFontSize(Number(v))"
      >
        <t-option :value="12" label="12" />
        <t-option :value="14" label="14" />
        <t-option :value="16" label="16" />
        <t-option :value="18" label="18" />
        <t-option :value="20" label="20" />
        <t-option :value="24" label="24" />
      </t-select>

      <t-divider layout="vertical" style="margin: 0 6px" />

      <t-tooltip content="加粗">
        <t-button size="small" variant="text" @click="toggleBold" :disabled="!editor">
          <t-icon name="format-bold" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip content="斜体">
        <t-button size="small" variant="text" @click="toggleItalic" :disabled="!editor">
          <t-icon name="format-italic" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip content="下划线">
        <t-button size="small" variant="text" @click="editor?.chain().focus().toggleUnderline().run()" :disabled="!editor">
          <t-icon name="underline" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip content="删除线">
        <t-button size="small" variant="text" @click="toggleStrike" :disabled="!editor">
          <t-icon name="format-strikethrough" size="16px" />
        </t-button>
      </t-tooltip>

      <t-divider layout="vertical" style="margin: 0 6px" />

      <t-tooltip content="文字颜色">
        <t-button size="small" variant="text" @click="pickTextColor" :disabled="!editor">
          <span style="font-weight: 700; line-height: 1">A</span>
        </t-button>
      </t-tooltip>
      <t-tooltip content="高亮">
        <t-button size="small" variant="text" @click="pickHighlight" :disabled="!editor">
          <t-icon name="highlight-1" size="16px" />
        </t-button>
      </t-tooltip>

      <t-divider layout="vertical" style="margin: 0 6px" />

      <t-tooltip content="有序列表">
        <t-button size="small" variant="text" @click="toggleOrderedList" :disabled="!editor">
          <t-icon name="ordered-list" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip content="无序列表">
        <t-button size="small" variant="text" @click="toggleBulletList" :disabled="!editor">
          <t-icon name="list" size="16px" />
        </t-button>
      </t-tooltip>

      <t-divider layout="vertical" style="margin: 0 6px" />

      <t-tooltip content="左对齐">
        <t-button size="small" variant="text" @click="setAlign('left')" :disabled="!editor">
          <t-icon name="text-align-left" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip content="居中">
        <t-button size="small" variant="text" @click="setAlign('center')" :disabled="!editor">
          <t-icon name="text-align-center" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip content="右对齐">
        <t-button size="small" variant="text" @click="setAlign('right')" :disabled="!editor">
          <t-icon name="text-align-right" size="16px" />
        </t-button>
      </t-tooltip>

      <t-divider layout="vertical" style="margin: 0 6px" />

      <t-tooltip content="更多">
        <t-button size="small" variant="text" @click="toggleCodeBlock" :disabled="!editor">
          <t-icon name="more" size="16px" />
        </t-button>
      </t-tooltip>
    </div>

    <!-- Editor body -->
    <div class="editor-body">
      <div class="pane edit-pane">
        <EditorContent v-if="editor" :editor="editor" class="editor-surface" />
      </div>
    </div>

    <!-- Share dialog -->
    <t-dialog
      v-model:visible="showShareDialog"
      header="分享笔记"
      :footer="false"
      width="480px"
    >
      <div class="share-dialog-content">
        <!-- Existing share status -->
        <div v-if="shareLink" class="share-result">
          <p class="label">分享链接</p>
          <div class="link-row">
            <t-input :model-value="shareLink" readonly />
            <t-button size="small" @click="copyShareLink">
              <t-icon name="copy" />
            </t-button>
          </div>
          <t-button
            size="small"
            variant="outline"
            theme="danger"
            @click="deleteShare"
            style="margin-top: 12px"
          >
            取消分享
          </t-button>
        </div>

        <!-- Create new share -->
        <div v-else class="share-form">
          <div class="form-group">
            <label>访问密码（可选）</label>
            <t-input
              v-model="sharePassword"
              type="password"
              placeholder="留空则无需密码"
              clearable
            />
          </div>
          <div class="form-group">
            <label>过期时间（可选）</label>
            <t-select v-model="shareExpiresDays" placeholder="永不过期" clearable>
              <t-option :value="1" label="1 天" />
              <t-option :value="7" label="7 天" />
              <t-option :value="30" label="30 天" />
              <t-option :value="90" label="90 天" />
            </t-select>
          </div>
          <t-button
            block
            @click="createShare"
            :loading="shareLoading"
          >
            创建分享链接
          </t-button>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<style scoped>
.note-editor {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 60px);
}

.editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  border-bottom: 1px solid #e5e6eb;
  background: #fff;
  min-height: 48px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.title-input-inline {
  font-size: 18px;
  font-weight: 600;
  border: none;
  outline: none;
  padding: 4px 0;
  color: #1d2129;
  background: transparent;
  flex: 1;
  min-width: 0;
}

:deep(.title-input-inline .t-input__inner) {
  font-size: 18px;
  font-weight: 600;
  padding: 4px 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.save-status {
  font-size: 12px;
  color: #86909c;
}

/* Formatting toolbar */
.format-toolbar {
  display: flex;
  align-items: center;
  padding: 4px 16px;
  border-bottom: 1px solid #e5e6eb;
  background: #fafbfc;
  gap: 2px;
  flex-wrap: wrap;
}

/* Editor body */
.editor-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.pane {
  overflow-y: auto;
}

.edit-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.editor-surface {
  flex: 1;
  padding: 24px;
}

:deep(.wysiwyg-editor) {
  min-height: 100%;
  font-size: 15px;
  line-height: 1.8;
  color: #1d2129;
  background: transparent;
}

:deep(.wysiwyg-editor:focus) {
  outline: none;
}

/* Share dialog */
.share-dialog-content {
  padding: 8px 0;
}

.share-result {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.share-result .label {
  font-size: 13px;
  font-weight: 600;
  color: #4e5969;
  margin: 0;
}

.link-row {
  display: flex;
  gap: 8px;
}

.share-form {
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
</style>
