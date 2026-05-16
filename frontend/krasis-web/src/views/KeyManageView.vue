<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import QRCode from 'qrcode'
import jsQR from 'jsqr'
import {
  generateKeyPair,
  unlockKey,
  hasStoredKey,
  hasSessionKey,
  deleteStoredKey,
  clearSessionKey,
  getSessionPublicKey,
  publicKeyToPem,
  publicKeyToHex,
  syncToServer,
  syncFromServer,
  autoSyncKeys,
  getQRKeyPackage,
} from '../crypto/keyStore'
import { deleteKeyFromServer } from '../api/keys'

const keyExists = ref(false)
const keyUnlocked = ref(false)
const publicKeyHex = ref('')
const publicKeyPem = ref('')
const loading = ref(false)

// Sync status
const syncing = ref(false)
const syncError = ref(false)
const lastSyncTime = ref<string | null>(null)

// QR sharing
const qrCodeUrl = ref('')
const showQrCode = ref(false)
const qrLoading = ref(false)
const qrError = ref('')

// QR scan import
const showScanDialog = ref(false)
const scanLoading = ref(false)
const scanError = ref('')

// Generate dialog
const showGenerateDialog = ref(false)
const genPassword = ref('')
const genPasswordConfirm = ref('')

// Unlock dialog
const showUnlockDialog = ref(false)
const unlockPassword = ref('')

onMounted(async () => {
  // Try server auto-sync first
  syncing.value = true
  await autoSyncKeys()
  syncing.value = false
  await refreshStatus()
})

async function refreshStatus() {
  keyExists.value = await hasStoredKey()
  keyUnlocked.value = hasSessionKey()
  if (keyUnlocked.value) {
    const pub = getSessionPublicKey()
    if (pub) {
      publicKeyHex.value = publicKeyToHex(pub)
      publicKeyPem.value = publicKeyToPem(pub)
    }
  }
}

async function syncWithServer() {
  syncing.value = true
  syncError.value = false
  try {
    await syncToServer()
    lastSyncTime.value = new Date().toLocaleTimeString()
  } catch (e: any) {
    syncError.value = true
  } finally {
    syncing.value = false
  }
}

async function handleGenerate() {
  if (genPassword.value.length < 6) {
    MessagePlugin.warning('密码至少 6 位')
    return
  }
  if (genPassword.value !== genPasswordConfirm.value) {
    MessagePlugin.warning('两次输入的密码不一致')
    return
  }
  loading.value = true
  try {
    await generateKeyPair(genPassword.value)
    MessagePlugin.success('ECC 密钥对已生成并安全存储')
    showGenerateDialog.value = false
    genPassword.value = ''
    genPasswordConfirm.value = ''
    await refreshStatus()

    // Sync to server silently
    await syncToServer()
  } catch (e: any) {
    MessagePlugin.error('密钥生成失败: ' + (e?.message || ''))
  } finally {
    loading.value = false
  }
}

async function handleUnlock() {
  if (!unlockPassword.value) {
    MessagePlugin.warning('请输入密码')
    return
  }
  loading.value = true
  try {
    const ok = await unlockKey(unlockPassword.value)
    if (ok) {
      MessagePlugin.success('密钥已解锁，当前会话可用')
      showUnlockDialog.value = false
      unlockPassword.value = ''
      await refreshStatus()

      // Sync to server silently
      await syncToServer()
    } else {
      MessagePlugin.error('密码错误，无法解锁密钥')
    }
  } catch (e: any) {
    MessagePlugin.error('解锁失败: ' + (e?.message || ''))
  } finally {
    loading.value = false
  }
}

function handleLock() {
  clearSessionKey()
  keyUnlocked.value = false
  publicKeyHex.value = ''
  publicKeyPem.value = ''
  MessagePlugin.success('密钥已锁定')
}

async function handleDeleteKey() {
  DialogPlugin.confirm({
    header: '删除密钥',
    body: '确定要删除存储的 ECC 密钥吗？已加密的笔记将无法解密！同时将从云端移除同步的密钥。请确保已备份。',
    theme: 'danger',
    onConfirm: async () => {
      try {
        await deleteStoredKey()
        // Also delete from server
        await deleteKeyFromServer()
        MessagePlugin.success('密钥已删除（本地 + 云端）')
        await refreshStatus()
      } catch (e: any) {
        MessagePlugin.error('删除失败: ' + (e?.message || ''))
      }
    },
  })
}

function copyPublicKey() {
  navigator.clipboard.writeText(publicKeyPem.value)
  MessagePlugin.success('公钥已复制到剪贴板')
}

// ─── QR code sharing ───────────────────────────────────────────────────────

async function showQRShare() {
  qrLoading.value = true
  qrError.value = ''
  qrCodeUrl.value = ''
  showQrCode.value = true

  try {
    const json = await getQRKeyPackage()
    if (!json) {
      qrError.value = '未找到密钥，请先生成密钥'
      return
    }

    // Generate QR code as data URL
    qrCodeUrl.value = await QRCode.toDataURL(json, {
      width: 400,
      margin: 2,
      color: { dark: '#000000', light: '#ffffff' },
    })
  } catch (e: any) {
    qrError.value = '生成二维码失败: ' + (e?.message || '')
  } finally {
    qrLoading.value = false
  }
}

async function handleScanQR() {
  showScanDialog.value = true
  scanError.value = ''
}

function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  scanLoading.value = true
  scanError.value = ''

  const reader = new FileReader()
  reader.onload = async (e) => {
    try {
      const img = new Image()
      img.onload = async () => {
        try {
          // Draw image to canvas to get pixel data
          const canvas = document.createElement('canvas')
          canvas.width = img.naturalWidth
          canvas.height = img.naturalHeight
          const ctx = canvas.getContext('2d')
          if (!ctx) {
            scanError.value = '无法读取图片'
            scanLoading.value = false
            return
          }
          ctx.drawImage(img, 0, 0)
          const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height)

          // Decode QR code
          const code = jsQR(imageData.data, imageData.width, imageData.height)
          if (!code) {
            scanError.value = '未在图片中找到二维码'
            scanLoading.value = false
            return
          }

          // Parse the QR content as key package
          let pkgData: any
          try {
            pkgData = JSON.parse(code.data)
          } catch {
            scanError.value = '二维码内容不是有效的密钥包格式'
            scanLoading.value = false
            return
          }

          // Validate compact format: { v, t, ek, s, i, pk }
          if (!pkgData.ek || !pkgData.s || !pkgData.i || !pkgData.pk) {
            scanError.value = '二维码密钥包格式无效'
            scanLoading.value = false
            return
          }

          // Validate base64
          try {
            atob(pkgData.ek)
            atob(pkgData.s)
            atob(pkgData.i)
            atob(pkgData.pk)
          } catch {
            scanError.value = '密钥数据编码无效'
            scanLoading.value = false
            return
          }

          // Remove existing keys then write via import
          const { importKeyPackage } = await import('../crypto/keyStore')
          const fullPackage = JSON.stringify({
            encryptedPrivateKey: pkgData.ek,
            salt: pkgData.s,
            iv: pkgData.i,
            publicKeyRaw: pkgData.pk,
          })
          const ok = await importKeyPackage(fullPackage)
          if (ok) {
            MessagePlugin.success('通过二维码导入密钥成功')
            showScanDialog.value = false
            // Sync back to server
            await syncToServer()
            await refreshStatus()
          } else {
            scanError.value = '密钥导入失败'
          }
        } catch (e: any) {
          scanError.value = '解析二维码失败: ' + (e?.message || '')
        }
        scanLoading.value = false
      }
      img.src = e.target?.result as string
    } catch {
      scanError.value = '读取文件失败'
      scanLoading.value = false
    }
  }
  reader.readAsDataURL(file)

  // Reset input so same file can be selected again
  input.value = ''
}
</script>

<template>
  <div class="key-manage-page">
    <div class="page-header">
      <h1 class="page-title">
        <t-icon name="lock-on" /> ECC 加密密钥管理
      </h1>
      <p class="page-desc">使用 ECDH P-256 + AES-256-GCM 对笔记进行端到端加密。密钥自动同步到云端，可在不同设备使用同一密钥。</p>
    </div>

    <!-- Status card -->
    <div class="status-card">
      <div class="status-icon" :class="{ active: keyUnlocked, exists: keyExists && !keyUnlocked }">
        <t-icon :name="keyUnlocked ? 'lock-on' : 'lock-off'" size="32px" />
      </div>
      <div class="status-info">
        <div class="status-title">
          {{ keyUnlocked ? '密钥已解锁' : keyExists ? '密钥已存储，未解锁' : '未创建密钥' }}
        </div>
        <div class="status-desc">
          {{ keyUnlocked ? '当前会话中私钥可用，可加密/解密笔记' : keyExists ? '请输入密码解锁密钥以加密笔记' : '首次使用请生成 ECC 密钥对' }}
        </div>
        <div v-if="keyUnlocked" class="key-fingerprint">
          <span class="fingerprint-label">公钥指纹：</span>
          <code class="fingerprint-value">{{ publicKeyHex?.substring(0, 32) || '' }}...</code>
        </div>
        <!-- Sync status -->
        <div v-if="keyExists" class="sync-status">
          <t-icon v-if="syncing" name="loading" class="sync-icon spinning" />
          <t-icon v-else-if="syncError" name="close-circle" class="sync-icon error" />
          <t-icon v-else name="check-circle" class="sync-icon ok" />
          <span class="sync-text">
            {{ syncing ? '同步中...' : syncError ? '同步失败' : lastSyncTime ? '已同步: ' + lastSyncTime : '点击同步到云端' }}
          </span>
        </div>
      </div>
      <div class="status-actions">
        <t-button v-if="!keyExists" theme="primary" @click="showGenerateDialog = true">
          <t-icon name="add" /> 生成密钥
        </t-button>
        <t-button v-else-if="!keyUnlocked" theme="primary" @click="showUnlockDialog = true">
          <t-icon name="unlock" /> 解锁密钥
        </t-button>
        <t-button v-else variant="outline" @click="handleLock">
          <t-icon name="lock-off" /> 锁定密钥
        </t-button>
      </div>
    </div>

    <!-- Key details (when unlocked) -->
    <div v-if="keyUnlocked" class="key-details-card">
      <h3>公钥信息</h3>
      <div class="key-detail-section">
        <label>算法</label>
        <span>ECDH P-256 (X9.63)</span>
      </div>
      <div class="key-detail-section">
        <label>指纹 (SHA-256)</label>
        <code class="key-hex">{{ publicKeyHex }}</code>
      </div>
      <div class="key-detail-section">
        <label>公钥 (PEM)</label>
        <pre class="key-pem">{{ publicKeyPem }}</pre>
        <t-button size="small" variant="outline" @click="copyPublicKey">
          <t-icon name="copy" /> 复制公钥
        </t-button>
      </div>
    </div>

    <!-- Server sync card -->
    <div v-if="keyExists" class="sync-card">
      <h3><t-icon name="cloud-upload" /> 云端同步</h3>
      <p class="sync-desc">密钥经密码加密后存储到服务端，登录同一账号后自动同步到其他设备，无需手动导入导出。</p>
      <div class="sync-actions">
        <t-button variant="outline" theme="default" @click="syncWithServer" :loading="syncing">
          <t-icon name="refresh" /> 同步到云端
        </t-button>
      </div>
    </div>

    <!-- QR sharing card -->
    <div v-if="keyExists" class="qr-card">
      <h3><t-icon name="qr-code" /> 二维码分享私钥（离线）</h3>
      <p class="qr-desc">生成二维码，另一设备可扫码直接导入私钥。注意：二维码包含加密私钥，请确保在可信环境中扫描。</p>
      <div class="qr-actions">
        <t-button variant="outline" theme="default" @click="showQRShare">
          <t-icon name="qr-code" /> 显示二维码
        </t-button>
        <t-button variant="outline" theme="default" @click="handleScanQR">
          <t-icon name="scan" /> 扫码导入
        </t-button>
      </div>

      <!-- QR code display -->
      <div v-if="showQrCode" class="qr-preview">
        <div v-if="qrLoading" class="qr-placeholder">生成二维码中...</div>
        <div v-else-if="qrError" class="qr-error">{{ qrError }}</div>
        <img v-else-if="qrCodeUrl" :src="qrCodeUrl" alt="密钥二维码" class="qr-image" />
      </div>
    </div>

    <!-- Danger zone -->
    <div v-if="keyExists" class="danger-card">
      <h3>危险操作</h3>
      <p class="danger-desc">删除密钥后，所有已加密的笔记将无法解密。同时将从云端移除同步的密钥。请确保已备份密钥或解密所有笔记后再执行。</p>
      <t-button theme="danger" variant="outline" @click="handleDeleteKey">
        <t-icon name="delete" /> 删除密钥（本地 + 云端）
      </t-button>
    </div>

    <!-- Generate dialog -->
    <t-dialog
      v-model:visible="showGenerateDialog"
      header="生成 ECC 密钥对"
      :footer="false"
      width="420px"
    >
      <div class="dialog-form">
        <div class="form-group">
          <label>加密密码 *</label>
          <p class="form-hint">此密码用于加密保护私钥，每次解锁密钥时需要。建议使用强密码（至少 8 位，含大小写字母和数字）。</p>
          <t-input
            v-model="genPassword"
            type="password"
            placeholder="私钥加密密码"
            autocomplete="new-password"
          />
        </div>
        <div class="form-group">
          <label>确认密码 *</label>
          <t-input
            v-model="genPasswordConfirm"
            type="password"
            placeholder="再次输入密码"
            autocomplete="new-password"
          />
        </div>
        <div class="dialog-actions">
          <t-button @click="showGenerateDialog = false" variant="text">取消</t-button>
          <t-button theme="primary" @click="handleGenerate" :loading="loading">
            生成密钥
          </t-button>
        </div>
      </div>
    </t-dialog>

    <!-- Unlock dialog -->
    <t-dialog
      v-model:visible="showUnlockDialog"
      header="解锁密钥"
      :footer="false"
      width="380px"
    >
      <div class="dialog-form">
        <div class="form-group">
          <label>私钥加密密码</label>
          <t-input
            v-model="unlockPassword"
            type="password"
            placeholder="输入生成密钥时设置的密码"
            autocomplete="current-password"
            @enter="handleUnlock"
          />
        </div>
        <div class="dialog-actions">
          <t-button @click="showUnlockDialog = false" variant="text">取消</t-button>
          <t-button theme="primary" @click="handleUnlock" :loading="loading">
            解锁
          </t-button>
        </div>
      </div>
    </t-dialog>

    <!-- QR scan import dialog -->
    <t-dialog
      v-model:visible="showScanDialog"
      header="扫码导入私钥"
      :footer="false"
      width="480px"
      @closed="scanError = ''"
    >
      <div class="dialog-form">
        <div class="form-group">
          <label>选择包含密钥二维码的图片</label>
          <p class="form-hint">在其他设备上生成二维码后截图保存，在此上传图片即可扫描导入。</p>
          <input
            type="file"
            accept="image/*"
            @change="onFileSelected"
            class="file-input"
          />
        </div>
        <div v-if="scanLoading" class="scan-status">正在扫描二维码...</div>
        <div v-if="scanError" class="scan-error">{{ scanError }}</div>
        <div class="dialog-actions">
          <t-button @click="showScanDialog = false; scanError = ''" variant="text">关闭</t-button>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<style scoped>
.key-manage-page {
  max-width: 720px;
  padding: 32px;
}

.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 22px;
  font-weight: 700;
  color: #1d2129;
  margin: 0 0 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.page-desc {
  font-size: 13px;
  color: #86909c;
  line-height: 1.6;
}

/* Status card */
.status-card {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 20px;
  border: 1px solid #e5e6eb;
  border-radius: 8px;
  background: #fff;
  margin-bottom: 16px;
}

.status-icon {
  width: 48px;
  height: 48px;
  background: #f2f3f5;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #86909c;
}

.status-icon.active {
  background: #e8f8e8;
  color: #00a870;
}

.status-icon.exists {
  background: #fff7e6;
  color: #ed7b2f;
}

.status-info {
  flex: 1;
}

.status-title {
  font-size: 15px;
  font-weight: 600;
  color: #1d2129;
  margin-bottom: 4px;
}

.status-desc {
  font-size: 13px;
  color: #86909c;
}

.key-fingerprint {
  margin-top: 8px;
  font-size: 12px;
}

.fingerprint-label {
  color: #86909c;
}

.fingerprint-value {
  font-family: ui-monospace, monospace;
  font-size: 12px;
  color: #4e5969;
  background: #f7f8fa;
  padding: 2px 6px;
  border-radius: 4px;
}

.sync-status {
  margin-top: 6px;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
}

.sync-icon {
  font-size: 14px;
}

.sync-icon.spinning {
  animation: spin 1s linear infinite;
  color: #0052d9;
}

.sync-icon.ok {
  color: #00a870;
}

.sync-icon.error {
  color: #e34d59;
}

.sync-text {
  color: #86909c;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.status-actions {
  flex-shrink: 0;
}

/* Key details */
.key-details-card {
  padding: 20px;
  border: 1px solid #e5e6eb;
  border-radius: 8px;
  background: #fff;
  margin-bottom: 16px;
}

.key-details-card h3 {
  font-size: 15px;
  font-weight: 600;
  color: #1d2129;
  margin: 0 0 16px;
}

.key-detail-section {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 12px;
}

.key-detail-section label {
  min-width: 100px;
  font-size: 13px;
  font-weight: 500;
  color: #86909c;
  flex-shrink: 0;
  padding-top: 2px;
}

.key-hex {
  font-family: ui-monospace, monospace;
  font-size: 11px;
  color: #4e5969;
  word-break: break-all;
  line-height: 1.6;
  background: #f7f8fa;
  padding: 4px 8px;
  border-radius: 4px;
  flex: 1;
}

.key-pem {
  font-family: ui-monospace, monospace;
  font-size: 11px;
  color: #4e5969;
  background: #f7f8fa;
  padding: 8px 12px;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-all;
  line-height: 1.5;
  margin: 0;
  flex: 1;
}

/* Danger card */
.danger-card {
  padding: 20px;
  border: 1px solid #fde3e3;
  border-radius: 8px;
  background: #fff8f8;
  margin-bottom: 16px;
}

.danger-card h3 {
  font-size: 15px;
  font-weight: 600;
  color: #e34d59;
  margin: 0 0 8px;
}

.danger-desc {
  font-size: 13px;
  color: #86909c;
  margin-bottom: 12px;
}

/* Sync card */
.sync-card {
  padding: 20px;
  border: 1px solid #bce3ff;
  border-radius: 8px;
  background: #f2f9ff;
  margin-bottom: 16px;
}

.sync-card h3 {
  font-size: 15px;
  font-weight: 600;
  color: #0052d9;
  margin: 0 0 8px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.sync-desc {
  font-size: 13px;
  color: #86909c;
  margin-bottom: 12px;
}

.sync-actions {
  display: flex;
  gap: 12px;
}

/* QR card */
.qr-card {
  padding: 20px;
  border: 1px solid #d4c5f9;
  border-radius: 8px;
  background: #f8f5ff;
  margin-bottom: 16px;
}

.qr-card h3 {
  font-size: 15px;
  font-weight: 600;
  color: #722ed1;
  margin: 0 0 8px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.qr-desc {
  font-size: 13px;
  color: #86909c;
  margin-bottom: 12px;
}

.qr-actions {
  display: flex;
  gap: 12px;
}

.qr-preview {
  margin-top: 16px;
  display: flex;
  justify-content: center;
  padding: 16px;
  background: #fff;
  border-radius: 8px;
  border: 1px solid #e5e6eb;
}

.qr-image {
  width: 240px;
  height: 240px;
  image-rendering: pixelated;
}

.qr-placeholder,
.qr-error {
  color: #86909c;
  font-size: 13px;
  padding: 24px;
}

.qr-error {
  color: #e34d59;
}

/* Scan dialog */
.file-input {
  font-size: 14px;
  padding: 8px 0;
}

.scan-status,
.scan-error {
  font-size: 13px;
  padding: 8px 0;
}

.scan-error {
  color: #e34d59;
}

/* Dialog */
.dialog-form {
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
