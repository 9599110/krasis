<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listGroups, createGroup, updateGroup, deleteGroup, getGroupFeatures, updateGroupFeatures } from '../../api/admin'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import { UsergroupIcon } from 'tdesign-icons-vue-next'

const loading = ref(false)
const groups = ref<any[]>([])
const showForm = ref(false)
const showFeatures = ref(false)
const isEdit = ref(false)
const editId = ref('')
const currentGroupId = ref('')

const form = ref({
  name: '',
  description: '',
  max_users: -1,
  is_default: false,
})

const features = ref({
  max_notes: -1,
  max_storage_bytes: 10737418240,
  max_file_size_bytes: 104857600,
  enable_ai: true,
  enable_sharing: true,
  enable_full_text_search: true,
  enable_versions: true,
  rate_limit_per_minute: 60,
  rate_limit_per_hour: 1000,
})

onMounted(() => { fetchGroups() })

async function fetchGroups() {
  loading.value = true
  try {
    const res = await listGroups()
    const d = res.data?.data || res.data || {}
    groups.value = Array.isArray(d) ? d : d.items || []
  } catch {
    MessagePlugin.error('获取用户组列表失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  isEdit.value = false
  form.value = { name: '', description: '', max_users: -1, is_default: false }
  showForm.value = true
}

function openEdit(group: any) {
  isEdit.value = true
  editId.value = group.id
  form.value = {
    name: group.name || '',
    description: group.description || '',
    max_users: group.max_users ?? -1,
    is_default: group.is_default ?? false,
  }
  showForm.value = true
}

async function handleSubmit() {
  if (!form.value.name) {
    MessagePlugin.warning('请填写组名称')
    return
  }
  try {
    if (isEdit.value) {
      await updateGroup(editId.value, form.value)
      MessagePlugin.success('用户组已更新')
    } else {
      await createGroup(form.value)
      MessagePlugin.success('用户组已创建')
    }
    showForm.value = false
    fetchGroups()
  } catch {
    MessagePlugin.error('操作失败')
  }
}

async function handleDelete(group: any) {
  DialogPlugin.confirm({
    header: '确认删除',
    body: `确定要删除用户组 "${group.name}" 吗？`,
    onConfirm: async () => {
      try {
        await deleteGroup(group.id)
        MessagePlugin.success('已删除')
        fetchGroups()
      } catch {
        MessagePlugin.error('删除失败')
      }
    },
  })
}

async function openFeatureEdit(group: any) {
  currentGroupId.value = group.id
  try {
    const res = await getGroupFeatures(group.id)
    const d = res.data?.data || res.data || {}
    if (d && typeof d === 'object') {
      for (const key of Object.keys(features.value) as (keyof typeof features.value)[]) {
        if (d[key] !== undefined) {
          if (typeof d[key] === 'object' && 'value' in d[key]) {
            (features.value as any)[key] = d[key].value
          } else {
            (features.value as any)[key] = d[key]
          }
        }
      }
    }
  } catch {
    // use defaults
  }
  showFeatures.value = true
}

async function saveFeatures() {
  try {
    await updateGroupFeatures(currentGroupId.value, features.value)
    MessagePlugin.success('功能配置已保存')
    showFeatures.value = false
  } catch {
    MessagePlugin.error('保存失败')
  }
}

function formatBytes(bytes: number) {
  if (bytes === -1) return '无限制'
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(0)} GB`
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(0)} MB`
  return `${bytes} B`
}
</script>

<template>
  <div class="groups-view">
    <div class="page-header">
      <UsergroupIcon class="page-icon" />
      <h1>用户组管理</h1>
      <span class="page-desc">管理用户组和权限配置</span>
    </div>

    <div class="admin-toolbar">
      <div class="toolbar-left">
        <span class="toolbar-hint">共 {{ groups.length }} 个用户组</span>
      </div>
      <div class="toolbar-right">
        <t-button @click="openCreate">添加用户组</t-button>
      </div>
    </div>

    <div class="table-wrapper">
      <t-table
        v-if="groups.length > 0 || !loading"
        :data="groups"
        :loading="loading"
        row-key="id"
        :columns="[
          { colKey: 'name', title: '名称', width: 120 },
          { colKey: 'description', title: '描述', ellipsis: true },
          { colKey: 'user_count', title: '用户数', width: 80 },
          { colKey: 'is_default', title: '默认', width: 70 },
          { colKey: 'created_at', title: '创建时间', width: 180 },
          { colKey: 'operations', title: '操作', width: 200 },
        ]"
      >
        <template #is_default="{ row }">
          <t-tag v-if="row.is_default" theme="warning" variant="light-outline" size="small">是</t-tag>
        </template>
        <template #operations="{ row }">
          <t-button variant="text" size="small" @click="openFeatureEdit(row)">功能配置</t-button>
          <t-button variant="text" size="small" @click="openEdit(row)">编辑</t-button>
          <t-button variant="text" size="small" theme="danger" @click="handleDelete(row)">删除</t-button>
        </template>
      </t-table>
    </div>

    <!-- Create/Edit Dialog -->
    <t-dialog v-model:visible="showForm" :header="isEdit ? '编辑用户组' : '添加用户组'" width="500px" :on-confirm="handleSubmit">
      <t-form :data="form" label-width="80px">
        <t-form-item label="名称">
          <t-input v-model="form.name" placeholder="用户组名称" />
        </t-form-item>
        <t-form-item label="描述">
          <t-textarea v-model="form.description" placeholder="描述" :autosize="{ minRows: 2, maxRows: 4 }" />
        </t-form-item>
        <t-form-item label="最大用户">
          <t-input-number v-model="form.max_users" :min="-1" />
          <span class="help-text">-1 表示无限制</span>
        </t-form-item>
        <t-form-item label="默认组">
          <t-switch v-model="form.is_default" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <!-- Features Dialog -->
    <t-dialog v-model:visible="showFeatures" header="功能配置" width="560px" :on-confirm="saveFeatures">
      <t-form :data="features" label-width="160px">
        <t-divider align="left" style="margin: 0 0 12px; font-size: 14px; font-weight: 600; color: #1d2129;">限制配置</t-divider>
        <t-form-item label="最大笔记数">
          <t-input-number v-model="features.max_notes" :min="-1" />
          <span class="help-text">-1 表示无限制</span>
        </t-form-item>
        <t-form-item label="存储空间">
          <t-input-number v-model="features.max_storage_bytes" :step="1073741824" />
          <span class="help-text">{{ formatBytes(features.max_storage_bytes) }}</span>
        </t-form-item>
        <t-form-item label="最大文件大小">
          <t-input-number v-model="features.max_file_size_bytes" :step="10485760" />
          <span class="help-text">{{ formatBytes(features.max_file_size_bytes) }}</span>
        </t-form-item>

        <t-divider align="left" style="margin: 16px 0 12px; font-size: 14px; font-weight: 600; color: #1d2129;">功能开关</t-divider>
        <t-form-item label="AI 功能">
          <t-switch v-model="features.enable_ai" />
        </t-form-item>
        <t-form-item label="分享功能">
          <t-switch v-model="features.enable_sharing" />
        </t-form-item>
        <t-form-item label="全文搜索">
          <t-switch v-model="features.enable_full_text_search" />
        </t-form-item>
        <t-form-item label="版本历史">
          <t-switch v-model="features.enable_versions" />
        </t-form-item>

        <t-divider align="left" style="margin: 16px 0 12px; font-size: 14px; font-weight: 600; color: #1d2129;">频率限制</t-divider>
        <t-form-item label="每分钟请求数">
          <t-input-number v-model="features.rate_limit_per_minute" :min="1" />
        </t-form-item>
        <t-form-item label="每小时请求数">
          <t-input-number v-model="features.rate_limit_per_hour" :min="1" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<style scoped>
.toolbar-hint {
  font-size: 13px;
  color: #86909c;
  padding: 0 8px;
}
</style>
