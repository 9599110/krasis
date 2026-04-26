<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getAuditLogs } from '../../api/admin'
import { MessagePlugin } from 'tdesign-vue-next'
import { HistoryIcon } from 'tdesign-icons-vue-next'

const loading = ref(false)
const logs = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)

const filters = ref({
  action: '',
  target_type: '',
  admin_username: '',
})

const actionOptions = [
  { value: '', label: '全部' },
  { value: 'user.create', label: '创建用户' },
  { value: 'user.update', label: '更新用户' },
  { value: 'user.delete', label: '删除用户' },
  { value: 'share.approve', label: '通过分享' },
  { value: 'share.reject', label: '拒绝分享' },
  { value: 'share.revoke', label: '撤回分享' },
  { value: 'config.update', label: '修改配置' },
  { value: 'group.create', label: '创建用户组' },
  { value: 'group.update', label: '更新用户组' },
  { value: 'group.delete', label: '删除用户组' },
]

const targetOptions = [
  { value: '', label: '全部' },
  { value: 'user', label: '用户' },
  { value: 'share', label: '分享' },
  { value: 'config', label: '配置' },
  { value: 'group', label: '用户组' },
]

onMounted(() => { fetchLogs() })

async function fetchLogs() {
  loading.value = true
  try {
    const params: Record<string, any> = {
      page: page.value,
      size: size.value,
    }
    if (filters.value.action) params.action = filters.value.action
    if (filters.value.target_type) params.target_type = filters.value.target_type
    if (filters.value.admin_username) params.admin_username = filters.value.admin_username

    const res = await getAuditLogs(params)
    const d = res.data?.data || res.data || {}
    logs.value = d.items || []
    total.value = d.total || 0
  } catch {
    MessagePlugin.error('获取操作日志失败')
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) {
  page.value = p
  fetchLogs()
}

function handleFilter() {
  page.value = 1
  fetchLogs()
}

function formatChanges(changes: any) {
  if (!changes) return '-'
  if (typeof changes === 'string') return changes
  return JSON.stringify(changes, null, 2)
}

function actionTagTheme(action: string) {
  const prefix = action?.split('.')[0] || ''
  const map: Record<string, string> = {
    user: 'primary',
    share: 'warning',
    config: 'success',
    group: 'default',
    ai_model: 'info',
  }
  return map[prefix] || 'default'
}

function actionLabel(action: string) {
  const opt = actionOptions.find((o) => o.value === action)
  return opt?.label || action
}
</script>

<template>
  <div class="logs-view">
    <div class="page-header">
      <HistoryIcon class="page-icon" />
      <h1>操作日志</h1>
      <span class="page-desc">查看管理员操作记录</span>
    </div>

    <div class="admin-toolbar">
      <div class="toolbar-left">
        <t-select v-model="filters.action" placeholder="操作类型" clearable style="width: 150px;" @change="handleFilter">
          <t-option v-for="opt in actionOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </t-select>
        <t-select v-model="filters.target_type" placeholder="目标类型" clearable style="width: 140px;" @change="handleFilter">
          <t-option v-for="opt in targetOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </t-select>
        <t-input v-model="filters.admin_username" placeholder="操作用户" style="width: 160px;" @enter="handleFilter" />
        <t-button @click="handleFilter">搜索</t-button>
      </div>
      <div class="toolbar-right">
        <span class="toolbar-hint">共 {{ total }} 条记录</span>
      </div>
    </div>

    <div class="table-wrapper">
      <t-table
        v-if="logs.length > 0 || !loading"
        :data="logs"
        :loading="loading"
        row-key="id"
        :columns="[
          { colKey: 'created_at', title: '时间', width: 180 },
          { colKey: 'admin_username', title: '操作用户', width: 120 },
          { colKey: 'action', title: '操作', width: 140 },
          { colKey: 'target_type', title: '目标类型', width: 100 },
          { colKey: 'target_id', title: '目标ID', width: 100, ellipsis: true },
          { colKey: 'changes', title: '变更内容', ellipsis: true },
        ]"
        :pagination="{ current: page, pageSize: size, total, onChange: onPageChange }"
      >
        <template #action="{ row }">
          <t-tag :theme="actionTagTheme(row.action)" variant="light" size="small">{{ actionLabel(row.action) }}</t-tag>
        </template>
        <template #target_type="{ row }">
          <t-tag variant="light-outline" size="small">{{ row.target_type || '-' }}</t-tag>
        </template>
        <template #changes="{ row }">
          <span class="changes-code">{{ formatChanges(row.changes) }}</span>
        </template>
      </t-table>
    </div>
  </div>
</template>

<style scoped>
.toolbar-hint {
  font-size: 13px;
  color: #86909c;
  padding: 0 8px;
}

.changes-code {
  font-family: ui-monospace, 'SF Mono', Consolas, monospace;
  font-size: 12px;
  color: #4e5969;
  background: #f7f8fa;
  padding: 4px 8px;
  border-radius: 6px;
  word-break: break-all;
  max-width: 400px;
  display: inline-block;
  line-height: 1.5;
}
</style>
