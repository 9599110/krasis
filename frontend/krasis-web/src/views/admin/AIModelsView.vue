<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { listModels, createModel, updateModel, deleteModel, testModel } from '../../api/admin'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import { Ai1Icon } from 'tdesign-icons-vue-next'

const loading = ref(false)
const models = ref<any[]>([])
const isEdit = ref(false)
const editId = ref('')

const columns = [
  { colKey: 'name', title: '名称', width: 120 },
  { colKey: 'provider', title: '提供商', width: 100 },
  { colKey: 'type', title: '类型', width: 100 },
  { colKey: 'model_name', title: '模型名', width: 180 },
  { colKey: 'endpoint', title: '端点', ellipsis: true },
  { colKey: 'is_enabled', title: '状态', width: 70 },
  { colKey: 'is_default', title: '默认', width: 70 },
  { colKey: 'operations', title: '操作', width: 200 },
]

const form = ref({
  name: '',
  provider: 'openai',
  type: 'llm',
  endpoint: '',
  api_key: '',
  model_name: '',
  api_version: '',
  max_tokens: 4096,
  temperature: 0.7,
  top_p: 0.9,
  dimensions: 0,
  is_enabled: true,
  is_default: false,
  priority: 100,
  config: '{}',
})

onMounted(() => { fetchModels() })

async function fetchModels() {
  loading.value = true
  try {
    const res = await listModels()
    const d = res.data?.data || res.data || {}
    models.value = Array.isArray(d) ? d : d.items || []
  } catch (e: any) {
    MessagePlugin.error('获取模型列表失败: ' + (e.response?.data?.message || e.message || ''))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  isEdit.value = false
  editId.value = ''
  form.value = { name: '', provider: 'openai', type: 'llm', endpoint: '', api_key: '', model_name: '', api_version: '', max_tokens: 4096, temperature: 0.7, top_p: 0.9, dimensions: 0, is_enabled: true, is_default: false, priority: 100, config: '{}' }
  showDialog()
}

function openEdit(model: any) {
  isEdit.value = true
  editId.value = model.id
  form.value = {
    name: model.name || '',
    provider: model.provider || 'openai',
    type: model.type || 'llm',
    endpoint: model.endpoint || '',
    api_key: '',
    model_name: model.model_name || '',
    api_version: model.api_version || '',
    max_tokens: model.max_tokens || 4096,
    temperature: model.temperature || 0.7,
    top_p: model.top_p || 0.9,
    dimensions: model.dimensions || 0,
    is_enabled: model.is_enabled ?? true,
    is_default: model.is_default ?? false,
    priority: model.priority || 100,
    config: typeof model.config === 'string' ? model.config : JSON.stringify(model.config || {}),
  }
  showDialog()
}

function showDialog() {
  DialogPlugin({
    header: isEdit.value ? '编辑模型' : '添加模型',
    width: '700px',
    body: () => h('div', { style: 'max-height: 60vh; overflow-y: auto; padding: 16px 0;' }, [
      h('div', { style: 'margin-bottom: 12px; font-size: 14px; font-weight: 600; color: #1d2129;' }, '基本信息'),
      h('div', { style: 'display: flex; gap: 16px; margin-bottom: 12px;' }, [
        h('div', { style: 'flex: 1;' }, [
          h('label', { style: 'display: block; font-size: 13px; margin-bottom: 4px; color: #4e5969;' }, '名称'),
          h('input', {
            type: 'text',
            value: form.value.name,
            onInput: (e: Event) => { form.value.name = (e.target as HTMLInputElement).value },
            placeholder: '配置名称',
            style: 'width: 100%; padding: 8px 12px; border: 1px solid #e5e6eb; border-radius: 6px; font-size: 14px;',
          }),
        ]),
      ]),
      h('div', { style: 'display: flex; gap: 16px; margin-bottom: 12px;' }, [
        h('div', { style: 'flex: 1;' }, [
          h('label', { style: 'display: block; font-size: 13px; margin-bottom: 4px; color: #4e5969;' }, '提供商'),
          h('select', {
            value: form.value.provider,
            onChange: (e: Event) => { form.value.provider = (e.target as HTMLSelectElement).value },
            style: 'width: 100%; padding: 8px 12px; border: 1px solid #e5e6eb; border-radius: 6px; font-size: 14px;',
          }, [
            h('option', { value: 'openai' }, 'OpenAI'),
            h('option', { value: 'azure' }, 'Azure OpenAI'),
            h('option', { value: 'ollama' }, 'Ollama'),
            h('option', { value: 'anthropic' }, 'Anthropic'),
          ]),
        ]),
        h('div', { style: 'flex: 1;' }, [
          h('label', { style: 'display: block; font-size: 13px; margin-bottom: 4px; color: #4e5969;' }, '类型'),
          h('select', {
            value: form.value.type,
            onChange: (e: Event) => { form.value.type = (e.target as HTMLSelectElement).value },
            style: 'width: 100%; padding: 8px 12px; border: 1px solid #e5e6eb; border-radius: 6px; font-size: 14px;',
          }, [
            h('option', { value: 'llm' }, 'LLM (对话)'),
            h('option', { value: 'embedding' }, 'Embedding (向量)'),
          ]),
        ]),
      ]),
      h('div', { style: 'margin: 16px 0 12px; font-size: 14px; font-weight: 600; color: #1d2129;' }, '连接配置'),
      h('div', { style: 'display: flex; flex-direction: column; gap: 12px;' }, [
        h('div', {}, [
          h('label', { style: 'display: block; font-size: 13px; margin-bottom: 4px; color: #4e5969;' }, '端点'),
          h('input', {
            type: 'text',
            value: form.value.endpoint,
            onInput: (e: Event) => { form.value.endpoint = (e.target as HTMLInputElement).value },
            placeholder: 'https://api.openai.com/v1',
            style: 'width: 100%; padding: 8px 12px; border: 1px solid #e5e6eb; border-radius: 6px; font-size: 14px;',
          }),
        ]),
        h('div', {}, [
          h('label', { style: 'display: block; font-size: 13px; margin-bottom: 4px; color: #4e5969;' }, 'API Key'),
          h('input', {
            type: 'password',
            value: form.value.api_key,
            onInput: (e: Event) => { form.value.api_key = (e.target as HTMLInputElement).value },
            placeholder: 'sk-...',
            style: 'width: 100%; padding: 8px 12px; border: 1px solid #e5e6eb; border-radius: 6px; font-size: 14px;',
          }),
        ]),
        h('div', {}, [
          h('label', { style: 'display: block; font-size: 13px; margin-bottom: 4px; color: #4e5969;' }, '模型名'),
          h('input', {
            type: 'text',
            value: form.value.model_name,
            onInput: (e: Event) => { form.value.model_name = (e.target as HTMLInputElement).value },
            placeholder: 'gpt-4 / deepseek-chat',
            style: 'width: 100%; padding: 8px 12px; border: 1px solid #e5e6eb; border-radius: 6px; font-size: 14px;',
          }),
        ]),
      ]),
      h('div', { style: 'margin: 16px 0 12px; font-size: 14px; font-weight: 600; color: #1d2129;' }, '高级设置'),
      h('div', { style: 'display: flex; gap: 16px; margin-bottom: 12px;' }, [
        h('div', { style: 'flex: 1;' }, [
          h('label', { style: 'display: block; font-size: 13px; margin-bottom: 4px; color: #4e5969;' }, '最大 Token'),
          h('input', {
            type: 'number',
            value: form.value.max_tokens,
            onInput: (e: Event) => { form.value.max_tokens = parseInt((e.target as HTMLInputElement).value) || 4096 },
            min: '1',
            style: 'width: 100%; padding: 8px 12px; border: 1px solid #e5e6eb; border-radius: 6px; font-size: 14px;',
          }),
        ]),
        h('div', { style: 'flex: 1;' }, [
          h('label', { style: 'display: block; font-size: 13px; margin-bottom: 4px; color: #4e5969;' }, 'Temperature'),
          h('input', {
            type: 'number',
            value: form.value.temperature,
            onInput: (e: Event) => { form.value.temperature = parseFloat((e.target as HTMLInputElement).value) || 0.7 },
            min: '0',
            max: '2',
            step: '0.1',
            style: 'width: 100%; padding: 8px 12px; border: 1px solid #e5e6eb; border-radius: 6px; font-size: 14px;',
          }),
        ]),
      ]),
    ]),
    onConfirm: async () => {
      if (!form.value.name || !form.value.model_name) {
        MessagePlugin.warning('请填写名称和模型名')
        return false
      }
      try {
        const payload: Record<string, any> = {
          ...form.value,
          config: typeof form.value.config === 'string' ? JSON.parse(form.value.config) : form.value.config,
        }
        // Avoid accidentally clearing secret by sending empty api_key
        if (!payload.api_key) delete payload.api_key
        if (isEdit.value) {
          await updateModel(editId.value, payload)
          MessagePlugin.success('模型已更新')
        } else {
          await createModel(payload)
          MessagePlugin.success('模型已创建')
        }
        fetchModels()
        return true
      } catch (e: any) {
        const msg = e.response?.data?.message || e.message || '操作失败'
        MessagePlugin.error(`操作失败: ${msg}`)
        return false
      }
    },
  })
}

async function handleDelete(model: any) {
  let dlg: any
  dlg = DialogPlugin.confirm({
    header: '确认删除',
    body: `确定要删除模型 "${model.name}" 吗？`,
    onConfirm: async () => {
      try {
        await deleteModel(model.id)
        MessagePlugin.success('已删除')
        fetchModels()
        dlg?.hide?.()
        return true
      } catch {
        MessagePlugin.error('删除失败')
        // Still close dialog to avoid "stuck" confirm modal UX
        dlg?.hide?.()
        return true
      }
    },
  })
}

async function handleTest(model: any) {
  try {
    const res = await testModel(model.id)
    const d = res.data?.data || res.data || {}
    if (d.status === 'ok') {
      MessagePlugin.success('连接正常')
    } else {
      MessagePlugin.warning(`连接异常: ${d.message}`)
    }
  } catch {
    MessagePlugin.error('连接测试失败')
  }
}
</script>

<template>
  <div class="ai-models-view">
    <div class="page-header">
      <Ai1Icon class="page-icon" />
      <h1>AI 模型配置</h1>
      <span class="page-desc">管理 LLM 和 Embedding 模型提供商</span>
    </div>

    <div class="admin-toolbar">
      <div class="toolbar-left">
        <span class="toolbar-hint">共 {{ models.length }} 个模型</span>
      </div>
      <div class="toolbar-right">
        <t-button @click="openCreate">添加模型</t-button>
      </div>
    </div>

    <div class="table-wrapper">
      <t-table
        v-if="models.length > 0 || !loading"
        :data="models"
        :loading="loading"
        row-key="id"
        :columns="columns"
      >
        <template #type="{ row }">
          <t-tag :theme="row.type === 'llm' ? 'primary' : 'success'" variant="light" size="small">
            {{ row.type === 'llm' ? 'LLM' : 'Embedding' }}
          </t-tag>
        </template>
        <template #is_enabled="{ row }">
          <t-tag :theme="row.is_enabled ? 'success' : 'default'" variant="light" size="small">
            {{ row.is_enabled ? '启用' : '禁用' }}
          </t-tag>
        </template>
        <template #is_default="{ row }">
          <t-tag v-if="row.is_default" theme="warning" variant="light-outline" size="small">默认</t-tag>
        </template>
        <template #operations="{ row }">
          <t-button variant="text" size="small" @click="handleTest(row)">测试</t-button>
          <t-button variant="text" size="small" @click="openEdit(row)">编辑</t-button>
          <t-button variant="text" size="small" theme="danger" @click="handleDelete(row)">删除</t-button>
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
</style>
