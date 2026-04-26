<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listUsers, updateUserRole, updateUserStatus, deleteUser, batchDisableUsers, createUser } from '../../api/admin'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import { UserListIcon } from 'tdesign-icons-vue-next'

const loading = ref(false)
const users = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const keyword = ref('')
const roleFilter = ref('')
const showCreate = ref(false)

const newUser = ref({ email: '', username: '', password: '', role: 'member' })
const selectedRows = ref<any[]>([])

onMounted(() => { fetchUsers() })

async function fetchUsers() {
  loading.value = true
  try {
    const res = await listUsers({
      page: page.value,
      size: size.value,
      keyword: keyword.value,
      role: roleFilter.value,
    })
    const d = res.data?.data || res.data || {}
    users.value = d.items || []
    total.value = d.total || 0
  } catch (e: any) {
    MessagePlugin.error('获取用户列表失败: ' + (e.response?.data?.message || e.message))
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) {
  page.value = p
  fetchUsers()
}

async function handleSearch() {
  page.value = 1
  fetchUsers()
}

function onSelectionChange(rows: any[]) {
  selectedRows.value = rows
}

async function toggleRole(user: any) {
  const newRole = user.role === 'admin' ? 'member' : 'admin'
  try {
    await updateUserRole(user.id, newRole)
    MessagePlugin.success('角色已更新')
    fetchUsers()
  } catch (e: any) {
    MessagePlugin.error('更新角色失败')
  }
}

async function toggleStatus(user: any) {
  const newStatus = user.status === 1 ? 0 : 1
  try {
    await updateUserStatus(user.id, newStatus)
    MessagePlugin.success('状态已更新')
    fetchUsers()
  } catch (e: any) {
    MessagePlugin.error('更新状态失败')
  }
}

async function handleDelete(user: any) {
  DialogPlugin.confirm({
    header: '确认删除',
    body: `确定要删除用户 "${user.username}" 吗？此操作不可撤销。`,
    onConfirm: async () => {
      try {
        await deleteUser(user.id)
        MessagePlugin.success('用户已删除')
        fetchUsers()
      } catch {
        MessagePlugin.error('删除失败')
      }
    },
  })
}

async function handleBatchDisable() {
  if (selectedRows.value.length === 0) {
    MessagePlugin.warning('请选择要禁用的用户')
    return
  }
  try {
    await batchDisableUsers(selectedRows.value.map((u: any) => u.id))
    MessagePlugin.success(`已批量禁用 ${selectedRows.value.length} 个用户`)
    selectedRows.value = []
    fetchUsers()
  } catch {
    MessagePlugin.error('批量禁用失败')
  }
}

async function handleCreate() {
  if (!newUser.value.email || !newUser.value.username || !newUser.value.password) {
    MessagePlugin.warning('请填写必填字段')
    return
  }
  try {
    await createUser(newUser.value)
    MessagePlugin.success('用户创建成功')
    showCreate.value = false
    newUser.value = { email: '', username: '', password: '', role: 'member' }
    fetchUsers()
  } catch {
    MessagePlugin.error('创建用户失败')
  }
}

function roleTheme(role: string) {
  const map: Record<string, string> = { admin: 'warning', member: 'primary', viewer: 'default' }
  return map[role] || 'default'
}

function roleLabel(role: string) {
  const map: Record<string, string> = { admin: '管理员', member: '用户', viewer: '查看者' }
  return map[role] || role
}
</script>

<template>
  <div class="users-view">
    <div class="page-header">
      <UserListIcon class="page-icon" />
      <h1>用户管理</h1>
      <span class="page-desc">管理系统用户、角色和状态</span>
    </div>

    <div class="admin-toolbar">
      <div class="toolbar-left">
        <t-input v-model="keyword" placeholder="搜索邮箱/用户名" style="width: 240px;" @enter="handleSearch" />
        <t-select v-model="roleFilter" placeholder="角色筛选" clearable style="width: 140px;" @change="handleSearch">
          <t-option value="admin" label="admin" />
          <t-option value="member" label="member" />
          <t-option value="viewer" label="viewer" />
        </t-select>
        <t-button @click="handleSearch">搜索</t-button>
      </div>
      <div class="toolbar-right">
        <t-button variant="outline" :disabled="selectedRows.length === 0" @click="handleBatchDisable">
          批量禁用
        </t-button>
        <t-button @click="showCreate = true">创建用户</t-button>
      </div>
    </div>

    <div class="table-wrapper">
      <t-table
        v-if="users.length > 0 || !loading"
        :data="users"
        :loading="loading"
        row-key="id"
        :columns="[
          { colKey: 'row-select', type: 'multiple', width: 50 },
          { colKey: 'email', title: '邮箱', width: 200 },
          { colKey: 'username', title: '用户名', width: 120 },
          { colKey: 'role', title: '角色', width: 100 },
          { colKey: 'status', title: '状态', width: 80 },
          { colKey: 'created_at', title: '注册时间', width: 180 },
          { colKey: 'operations', title: '操作', width: 250 },
        ]"
        :pagination="{ current: page, pageSize: size, total, onChange: onPageChange }"
        @select-change="onSelectionChange"
      >
        <template #role="{ row }">
          <t-tag :theme="roleTheme(row.role)" variant="light" size="small">{{ roleLabel(row.role) }}</t-tag>
        </template>
        <template #status="{ row }">
          <t-tag :theme="row.status === 1 ? 'success' : 'danger'" variant="light" size="small">
            {{ row.status === 1 ? '启用' : '禁用' }}
          </t-tag>
        </template>
        <template #operations="{ row }">
          <t-button variant="text" size="small" @click="toggleRole(row)">
            {{ row.role === 'admin' ? '降为普通用户' : '设为管理员' }}
          </t-button>
          <t-button variant="text" size="small" :theme="row.status === 1 ? 'warning' : 'success'" @click="toggleStatus(row)">
            {{ row.status === 1 ? '禁用' : '启用' }}
          </t-button>
          <t-button variant="text" size="small" theme="danger" @click="handleDelete(row)">删除</t-button>
        </template>
      </t-table>
    </div>

    <!-- Create User Dialog -->
    <t-dialog v-model:visible="showCreate" header="创建用户" width="560px" :on-confirm="handleCreate">
      <t-form :data="newUser" label-width="80px">
        <t-form-item label="邮箱">
          <t-input v-model="newUser.email" placeholder="邮箱地址" />
        </t-form-item>
        <t-form-item label="用户名">
          <t-input v-model="newUser.username" placeholder="用户名" />
        </t-form-item>
        <t-form-item label="密码">
          <t-input v-model="newUser.password" type="password" placeholder="密码" />
        </t-form-item>
        <t-form-item label="角色">
          <t-select v-model="newUser.role">
            <t-option value="member" label="普通用户" />
            <t-option value="admin" label="管理员" />
            <t-option value="viewer" label="查看者" />
          </t-select>
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>
