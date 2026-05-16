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
import {
  hasSessionKey,
  hasStoredKey,
  unlockKey,
  encryptNoteContent,
  decryptNoteContent,
  isEncryptedContent,
} from '../crypto/keyStore'

const route = useRoute()
const router = useRouter()
const id = computed(() => String(route.params.id ?? ''))

const title = ref('')
const loading = ref(false)
const saving = ref(false)
const version = ref(1)
const saveTimer = ref<any>(null)
const isNew = computed(() => id.value === 'new')
const folderId = computed(() => route.query.folder ? String(route.query.folder) : undefined)
// 内容快照，避免对未变更的内容触发的保存
let lastSavedContent = ''
let lastSavedTitle = ''

// Encryption state
const encryptionEnabled = ref(false)
const noteIsEncrypted = ref(false)
const encryptionUnlocked = ref(false)
const decrypting = ref(false)

// Password unlock dialog state
const showPasswordDialog = ref(false)
const passwordDialogMode = ref<'encrypt' | 'download'>('encrypt')
const passwordInput = ref('')
const passwordDialogLoading = ref(false)
let passwordDialogResolve: ((ok: boolean) => void) | null = null

// Download note state
const noteContentForDownload = ref('')

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
    noteIsEncrypted.value = d.is_encrypted === true
    encryptionUnlocked.value = hasSessionKey()
    // If note is already encrypted, sync the toggle state
    if (noteIsEncrypted.value) {
      encryptionEnabled.value = true
    }

    title.value = d.title || ''

    // Handle encrypted content
    let content = d.content || ''
    if (noteIsEncrypted.value && content) {
      if (hasSessionKey() && isEncryptedContent(content)) {
        decrypting.value = true
        try {
          content = await decryptNoteContent(content)
          encryptionUnlocked.value = true
        } catch {
          content = '🔒 **笔记已加密，无法自动解密。**\n\n请确保 ECC 密钥已解锁（设置 → 加密密钥管理）。'
          encryptionUnlocked.value = false
        } finally {
          decrypting.value = false
        }
      } else if (!hasSessionKey()) {
        content = '🔒 **笔记已加密。**\n\n请前往「加密密钥管理」页面解锁密钥后再查看。'
      }
    }

    editor.value?.commands.setContent(content)
    version.value = d.version || 1
    // 记录编辑器中显示的内容快照，用于对比变更（对加密笔记用解密后内容）
    lastSavedContent = content
    lastSavedTitle = d.title || ''
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

    // 如果内容和标题都未变更，跳过保存请求
    if (!isNew.value && markdown === lastSavedContent && title.value === lastSavedTitle) {
      return
    }

    // 新笔记且无内容则不存储（即使有标题）
    if (isNew.value && !markdown.trim()) {
      return
    }
    if (!title.value.trim()) {
      title.value = deriveTitleFromText(plainText)
    }

    // Determine if we should encrypt:
    // - For new notes: only if the encryption toggle is on AND has content
    // - For existing encrypted notes: ALWAYS re-encrypt (don't save plaintext to server)
    const shouldEncrypt = noteIsEncrypted.value
      || (encryptionEnabled.value && hasSessionKey() && markdown.trim().length > 0)
    let contentToSave = markdown

    // Guard: if note is encrypted but key is locked, don't save
    if (noteIsEncrypted.value && !hasSessionKey()) {
      MessagePlugin.warning('笔记已加密，请先解锁密钥后再保存')
      return
    }

    // Re-encrypt for already encrypted notes (even if empty content)
    if (shouldEncrypt) {
      contentToSave = await encryptNoteContent(markdown)
    }

    if (isNew.value) {
      const payload: Record<string, any> = { title: title.value, content: contentToSave }
      if (shouldEncrypt) payload.is_encrypted = true
      if (folderId.value) payload.folder_id = folderId.value
      const res = await createNote(payload)
      const d = res.data?.data || res.data || {}
      noteIsEncrypted.value = shouldEncrypt
      router.replace({ name: 'note-edit', params: { id: d.id } })
    } else {
      const payload: UpdateNoteRequest = {
        title: title.value,
        content: contentToSave,
      }
      if (shouldEncrypt) payload.is_encrypted = true
      const res = await updateNote(
        id.value,
        payload,
        version.value,
      )
      const d = res.data?.data || res.data || {}
      noteIsEncrypted.value = shouldEncrypt
      version.value = d.version || version.value
    }
    // 记录编辑器中显示的内容快照，用于下次对比变更
    lastSavedContent = markdown
    lastSavedTitle = title.value
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

// 标题变更单独触发保存（不影响编辑器内容的 AutoSave 计时器）
watch([title], () => {
  if (saveTimer.value) clearTimeout(saveTimer.value)
  saveTimer.value = setTimeout(doSave, 3000)
})

// ─── Password unlock dialog ─────────────────────────────────────────────────
async function handlePasswordUnlock() {
  if (!passwordInput.value) {
    MessagePlugin.warning('请输入密码')
    return
  }
  passwordDialogLoading.value = true
  try {
    const ok = await unlockKey(passwordInput.value)
    if (ok) {
      passwordDialogLoading.value = false
      showPasswordDialog.value = false
      passwordInput.value = ''
      MessagePlugin.success('密钥已解锁')

      // If mode was encrypt, toggle encryption on after unlock
      if (passwordDialogMode.value === 'encrypt') {
        encryptionEnabled.value = !encryptionEnabled.value
        if (encryptionEnabled.value) {
          MessagePlugin.success('已开启加密，保存时将自动加密笔记内容')
        } else {
          MessagePlugin.info('已关闭加密')
        }
      } else if (passwordDialogMode.value === 'download') {
        // Perform the download now that key is unlocked
        await doDownloadNote()
      }
    } else {
      passwordDialogLoading.value = false
      MessagePlugin.error('密码错误，无法解锁密钥')
    }
  } catch {
    passwordDialogLoading.value = false
    MessagePlugin.error('解锁失败')
  }
}

// ─── Download note ──────────────────────────────────────────────────────────
async function downloadNote() {
  if (isNew.value) {
    MessagePlugin.warning('请先保存笔记后再下载')
    return
  }
  if (noteIsEncrypted.value && !hasSessionKey()) {
    // Need to unlock first
    const stored = await hasStoredKey()
    if (!stored) {
      MessagePlugin.warning('无法下载加密笔记：请先在「加密密钥管理」页面生成并解锁密钥')
      return
    }
    showPasswordDialog.value = true
    passwordDialogMode.value = 'download'
    return
  }
  await doDownloadNote()
}

async function doDownloadNote() {
  try {
    // Get current content from editor
    let content = ''
    if (noteIsEncrypted.value) {
      if (!hasSessionKey()) return
      // Load the full content from server for download
      const res = await getNote(id.value)
      const d = res.data?.data || res.data || {}
      const encryptedContent = d.content || ''
      if (encryptedContent && isEncryptedContent(encryptedContent)) {
        content = await decryptNoteContent(encryptedContent)
      }
    } else {
      content = (editor.value as any)?.storage?.markdown?.getMarkdown?.() ?? editor.value?.getText?.() ?? ''
    }

    // Create download blob
    const header = `# ${title.value}\n\n`
    const fullContent = header + content
    const blob = new Blob([fullContent], { type: 'text/markdown;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${title.value || '笔记'}.md`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    MessagePlugin.success('笔记已下载')
  } catch {
    MessagePlugin.error('下载笔记失败')
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

// Encryption toggle
async function toggleEncryption() {
  const stored = await hasStoredKey()
  if (!stored) {
    MessagePlugin.warning('请先在「加密密钥管理」页面生成并解锁密钥')
    return
  }
  if (!hasSessionKey()) {
    // Key exists but not unlocked — show unlock dialog instead of toggle
    showPasswordDialog.value = true
    passwordDialogMode.value = 'encrypt'
    passwordDialogResolve = null
    return
  }
  if (noteIsEncrypted.value) {
    // Already encrypted note - can't disable (would lose data)
    MessagePlugin.info('此笔记已加密保存，新建笔记时如需关闭加密请取消勾选')
    return
  }
  encryptionEnabled.value = !encryptionEnabled.value
  if (encryptionEnabled.value) {
    MessagePlugin.success('已开启加密，保存时将自动加密笔记内容')
  } else {
    MessagePlugin.info('已关闭加密')
  }
}

function getEncryptionTooltip(): string {
  if (hasSessionKey()) return encryptionEnabled.value ? '关闭加密' : '开启加密'
  // Key may exist in storage but not in session
  return '点击输入密码解锁密钥'
}

const moreMenuOptions = [
  { value: 'code', content: '代码块' },
  { value: 'blockquote', content: '引用' },
  { value: 'horizontal-rule', content: '分割线' },
  { value: 'undo', content: '撤销' },
  { value: 'redo', content: '重做' },
]

function handleMoreMenuClick(option: Record<string, any>) {
  const val = option.value
  if (val === 'code') editor.value?.chain().focus().toggleCodeBlock().run()
  else if (val === 'blockquote') editor.value?.chain().focus().toggleBlockquote().run()
  else if (val === 'horizontal-rule') editor.value?.chain().focus().setHorizontalRule().run()
  else if (val === 'undo') editor.value?.chain().focus().undo().run()
  else if (val === 'redo') editor.value?.chain().focus().redo().run()
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

        <!-- Encryption badge/toggle -->
        <div class="encryption-badge" v-if="noteIsEncrypted" title="此笔记已加密">
          <t-icon name="lock-on" size="14px" style="color: #00a870" />
          <span style="color: #00a870; font-size: 12px;">已加密</span>
        </div>
        <t-tooltip :content="getEncryptionTooltip()">
          <t-button size="small" variant="text" @click="toggleEncryption">
            <t-icon :name="noteIsEncrypted || encryptionEnabled ? 'lock-on' : 'lock-off'" size="16px" />
          </t-button>
        </t-tooltip>

        <t-button size="small" @click="handleSave" :loading="saving">保存</t-button>
        <t-button
          size="small"
          variant="text"
          @click="downloadNote"
          :disabled="isNew"
        >
          <t-icon name="download" /> 下载
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
          <t-icon :name="isRecording ? 'stop-circle' : 'microphone'" size="16px" />
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
          <t-icon name="textformat-bold" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip content="斜体">
        <t-button size="small" variant="text" @click="toggleItalic" :disabled="!editor">
          <t-icon name="textformat-italic" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip content="下划线">
        <t-button size="small" variant="text" @click="editor?.chain().focus().toggleUnderline().run()" :disabled="!editor">
          <t-icon name="textformat-underline" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip content="删除线">
        <t-button size="small" variant="text" @click="toggleStrike" :disabled="!editor">
          <t-icon name="textformat-strikethrough" size="16px" />
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
          <t-icon name="highlight" size="16px" />
        </t-button>
      </t-tooltip>

      <t-divider layout="vertical" style="margin: 0 6px" />

      <t-tooltip content="有序列表">
        <t-button size="small" variant="text" @click="toggleOrderedList" :disabled="!editor">
          <t-icon name="order-list" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip content="无序列表">
        <t-button size="small" variant="text" @click="toggleBulletList" :disabled="!editor">
          <t-icon name="bulletpoint" size="16px" />
        </t-button>
      </t-tooltip>

      <t-divider layout="vertical" style="margin: 0 6px" />

      <t-tooltip content="左对齐">
        <t-button size="small" variant="text" @click="setAlign('left')" :disabled="!editor">
          <t-icon name="format-vertical-align-left" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip content="居中">
        <t-button size="small" variant="text" @click="setAlign('center')" :disabled="!editor">
          <t-icon name="format-vertical-align-center" size="16px" />
        </t-button>
      </t-tooltip>
      <t-tooltip content="右对齐">
        <t-button size="small" variant="text" @click="setAlign('right')" :disabled="!editor">
          <t-icon name="format-vertical-align-right" size="16px" />
        </t-button>
      </t-tooltip>

      <t-divider layout="vertical" style="margin: 0 6px" />

      <t-dropdown
        trigger="click"
        placement="bottom-right"
        :options="moreMenuOptions"
        @click="handleMoreMenuClick"
        :disabled="!editor"
      >
        <t-button size="small" variant="text" :disabled="!editor">
          <t-icon name="more" size="16px" />
        </t-button>
      </t-dropdown>
    </div>

    <!-- Editor body -->
    <div class="editor-body">
      <div class="pane edit-pane">
        <EditorContent v-if="editor" :editor="editor" class="editor-surface" />
      </div>
    </div>

    <!-- Password unlock dialog (for encryption toggle and download) -->
    <t-dialog
      v-model:visible="showPasswordDialog"
      :header="passwordDialogMode === 'download' ? '下载加密笔记 - 请输入密码' : '开启加密 - 请输入密码'"
      :footer="false"
      width="380px"
      :close-on-overlay-click="false"
    >
      <div class="dialog-form">
        <div class="form-group">
          <label>私钥加密密码</label>
          <p class="form-hint" v-if="passwordDialogMode === 'download'">此笔记已加密，需要解锁密钥后解密下载。</p>
          <p class="form-hint" v-else>需要先解锁密钥才能开启加密。</p>
          <t-input
            v-model="passwordInput"
            type="password"
            placeholder="输入生成密钥时设置的密码"
            autocomplete="current-password"
            @enter="handlePasswordUnlock"
          />
        </div>
        <div class="dialog-actions">
          <t-button @click="showPasswordDialog = false; passwordInput = ''" variant="text">取消</t-button>
          <t-button theme="primary" @click="handlePasswordUnlock" :loading="passwordDialogLoading">
            解锁
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

.dialog-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-hint {
  font-size: 12px;
  color: #86909c;
  margin: 0;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 8px;
}
</style>
