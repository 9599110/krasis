<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getPendingShares, getShareStats, approveShare, rejectShare, reReviewShare, revokeShare, batchReview, getShareDetail } from '../../api/admin'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import { CheckCircleIcon } from 'tdesign-icons-vue-next'

const loading = ref(false)
const shares = ref<any[]>([])
const stats = ref<any>({ pending: 0, approved: 0, rejected: 0, revoked: 0 })
const page = ref(1)
const size = ref(20)
const statusFilter = ref('pending')
const keyword = ref('')
const rejectReason = ref('')
const showRejectDialog = ref(false)
const currentShareId = ref('')
const selectedRows = ref<any[]>([])
const showContentDialog = ref(false)
const shareDetail = ref<any>(null)

onMounted(() => {
  fetchShares()
  fetchStats()
})

async function fetchShares() {
  loading.value = true
  try {
    const res = await getPendingShares({
      page: page.value,
      size: size.value,
      status: statusFilter.value,
      keyword: keyword.value,
    })
    const d = res.data?.data || res.data || {}
    shares.value = d.items || []
  } catch (e: any) {
    MessagePlugin.error('获取分享列表失败')
  } finally {
    loading.value = false
  }
}

async function fetchStats() {
  try {
    const res = await getShareStats()
    const d = res.data?.data || res.data || {}
    stats.value = d
  } catch {
    // ignore
  }
}

function onPageChange(p: number) {
  page.value = p
  fetchShares()
}

function onFilterChange() {
  page.value = 1
  fetchShares()
}

function onSelectionChange(rows: any[]) {
  selectedRows.value = rows
}

async function handleApprove(id: string) {
  try {
    await approveShare(id)
    MessagePlugin.success('审核通过')
    fetchShares()
    fetchStats()
  } catch {
    MessagePlugin.error('审核失败')
  }
}

function openRejectDialog(id: string) {
  currentShareId.value = id
  rejectReason.value = ''
  showRejectDialog.value = true
}

async function confirmReject() {
  if (!rejectReason.value.trim()) {
    MessagePlugin.warning('请填写拒绝原因')
    return
  }
  try {
    await rejectShare(currentShareId.value, rejectReason.value)
    MessagePlugin.success('已拒绝')
    showRejectDialog.value = false
    fetchShares()
    fetchStats()
  } catch {
    MessagePlugin.error('操作失败')
  }
}

async function handleReReview(id: string) {
  try {
    await reReviewShare(id)
    MessagePlugin.success('已重新设为待审核')
    fetchShares()
    fetchStats()
  } catch {
    MessagePlugin.error('操作失败')
  }
}

async function handleRevoke(id: string) {
  DialogPlugin.confirm({
    header: '确认撤回',
    body: '确定要撤回此分享吗？撤回后链接将失效。',
    onConfirm: async () => {
      try {
        await revokeShare(id)
        MessagePlugin.success('已撤回')
        fetchShares()
        fetchStats()
      } catch {
        MessagePlugin.error('撤回失败')
      }
    },
  })
}

async function handleBatchReview(action: 'approve' | 'reject') {
  if (selectedRows.value.length === 0) {
    MessagePlugin.warning('请选择要操作的分享')
    return
  }
  const ids = selectedRows.value.map((s: any) => s.id)
  try {
    await batchReview(ids, action)
    MessagePlugin.success(`已批量${action === 'approve' ? '通过' : '拒绝'} ${ids.length} 个分享`)
    selectedRows.value = []
    fetchShares()
    fetchStats()
  } catch {
    MessagePlugin.error('批量操作失败')
  }
}

async function viewContent(item: any) {
  try {
    const res = await getShareDetail(item.id)
    shareDetail.value = res.data?.data || res.data || {}
    showContentDialog.value = true
  } catch {
    // fallback: use inline snapshot
    shareDetail.value = item
    showContentDialog.value = true
  }
}

function statusTag(status: string) {
  const map: Record<string, { theme: string; label: string }> = {
    pending: { theme: 'warning', label: '待审核' },
    approved: { theme: 'success', label: '已通过' },
    rejected: { theme: 'danger', label: '已拒绝' },
    revoked: { theme: 'default', label: '已撤回' },
  }
  return map[status] || { theme: 'default', label: status }
}
</script>

<template>
  <div class="shares-view">
    <div class="page-header">
      <CheckCircleIcon class="page-icon" />
      <h1>分享审核</h1>
      <span class="page-desc">审核用户分享链接和内容</span>
    </div>

    <!-- Stats -->
    <div class="stats-bar">
      <div class="stat-mini">
        <span class="stat-mini-count total">{{ stats.total || 0 }}</span>
        <span class="stat-mini-label">全部</span>
      </div>
      <div class="stat-mini">
        <span class="stat-mini-count pending">{{ stats.pending || 0 }}</span>
        <span class="stat-mini-label">待审核</span>
      </div>
      <div class="stat-mini">
        <span class="stat-mini-count approved">{{ stats.approved || 0 }}</span>
        <span class="stat-mini-label">已通过</span>
      </div>
      <div class="stat-mini">
        <span class="stat-mini-count rejected">{{ stats.rejected || 0 }}</span>
        <span class="stat-mini-label">已拒绝</span>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="admin-toolbar">
      <div class="toolbar-left">
        <t-radio-group v-model="statusFilter" variant="primary-filled" @change="onFilterChange">
          <t-radio-button value="pending">待审核</t-radio-button>
          <t-radio-button value="approved">已通过</t-radio-button>
          <t-radio-button value="rejected">已拒绝</t-radio-button>
          <t-radio-button value="all">全部</t-radio-button>
        </t-radio-group>
      </div>
      <div class="toolbar-right">
        <t-input v-model="keyword" placeholder="搜索笔记标题/用户" style="width: 200px;" @enter="onFilterChange" />
        <t-button variant="outline" :disabled="selectedRows.length === 0" @click="handleBatchReview('approve')">
          批量通过
        </t-button>
        <t-button variant="outline" theme="danger" :disabled="selectedRows.length === 0" @click="handleBatchReview('reject')">
          批量拒绝
        </t-button>
      </div>
    </div>

    <!-- Table -->
    <div class="table-wrapper">
      <t-table
        v-if="shares.length > 0 || !loading"
        :data="shares"
        :loading="loading"
        row-key="id"
        :columns="[
          { colKey: 'row-select', type: 'multiple', width: 50 },
          { colKey: 'note_title', title: '笔记标题', width: 200 },
          { colKey: 'owner_username', title: '用户', width: 100 },
          { colKey: 'share_type', title: '类型', width: 80, cell: ({ row }: any) => row.share_type === 'link' ? '链接' : '邮箱' },
          { colKey: 'permission', title: '权限', width: 80, cell: ({ row }: any) => row.permission === 'read' ? '仅阅读' : '可编辑' },
          { colKey: 'status', title: '状态', width: 90 },
          { colKey: 'created_at', title: '创建时间', width: 180 },
          { colKey: 'operations', title: '操作', width: 320 },
        ]"
        :pagination="{ current: page, pageSize: size, total: shares.length, onChange: onPageChange }"
        @select-change="onSelectionChange"
      >
        <template #status="{ row }">
          <t-tag :theme="statusTag(row.status).theme" variant="light" size="small">{{ statusTag(row.status).label }}</t-tag>
        </template>
        <template #note_title="{ row }">
          <t-link theme="primary" @click="viewContent(row)" hover="color" style="cursor: pointer;">
            {{ row.note_title || '无标题' }}
          </t-link>
        </template>
        <template #operations="{ row }">
          <t-button v-if="row.status === 'pending'" variant="text" size="small" theme="success" @click="handleApprove(row.id)">通过</t-button>
          <t-button v-if="row.status === 'pending'" variant="text" size="small" theme="danger" @click="openRejectDialog(row.id)">拒绝</t-button>
          <t-button v-if="row.status === 'approved'" variant="text" size="small" @click="handleReReview(row.id)">复审</t-button>
          <t-button v-if="row.status === 'approved'" variant="text" size="small" theme="danger" @click="handleRevoke(row.id)">撤回</t-button>
          <t-button v-if="row.status === 'rejected'" variant="text" size="small" @click="handleReReview(row.id)">重新审核</t-button>
        </template>
      </t-table>
    </div>

    <!-- Reject Dialog -->
    <t-dialog v-model:visible="showRejectDialog" header="拒绝分享" width="480px" :on-confirm="confirmReject">
      <t-textarea v-model="rejectReason" placeholder="请填写拒绝原因（必填）" :autosize="{ minRows: 3, maxRows: 6 }" />
    </t-dialog>

    <!-- Content Preview Dialog -->
    <t-dialog v-model:visible="showContentDialog" :header="shareDetail?.note_title || '内容预览'" width="720px" :footer="false">
      <div v-if="shareDetail" class="content-preview">
        <div class="meta-row">
          <span>用户: {{ shareDetail.owner_username }}</span>
          <span>类型: {{ shareDetail.share_type === 'link' ? '链接分享' : '邮箱分享' }}</span>
          <span>权限: {{ shareDetail.permission === 'read' ? '仅阅读' : '可编辑' }}</span>
        </div>
        <t-divider />
        <pre class="content-text">{{ shareDetail.content_snapshot || shareDetail.note_preview || '无内容快照' }}</pre>
      </div>
    </t-dialog>
  </div>
</template>

<style scoped>
.content-preview .meta-row {
  display: flex;
  gap: 16px;
  color: #86909c;
  font-size: 13px;
  margin-bottom: 12px;
}

.content-text {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 14px;
  line-height: 1.6;
  color: #1d2129;
  max-height: 400px;
  overflow-y: auto;
  padding: 12px;
  background: #f7f8fa;
  border-radius: 4px;
}
</style>
