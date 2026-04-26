<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getAIConfig, updateAIConfig } from '../../api/admin'
import { MessagePlugin } from 'tdesign-vue-next'
import { SettingIcon } from 'tdesign-icons-vue-next'

const loading = ref(false)
const config = ref({
  chunk_size: 500,
  chunk_overlap: 50,
  top_k: 5,
  score_threshold: 0.7,
  enable_rag: true,
  max_context_tokens: 8000,
  system_prompt: '',
  enable_streaming: true,
})

onMounted(() => { fetchConfig() })

async function fetchConfig() {
  loading.value = true
  try {
    const res = await getAIConfig()
    const d = res.data?.data || res.data || {}
    if (d && typeof d === 'object') {
      for (const key of Object.keys(config.value) as (keyof typeof config.value)[]) {
        if (d[key] !== undefined) {
          if (typeof d[key] === 'object' && 'value' in d[key]) {
            (config.value as any)[key] = d[key].value
          } else {
            (config.value as any)[key] = d[key]
          }
        }
      }
    }
  } catch {
    MessagePlugin.error('获取 AI 配置失败')
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  try {
    await updateAIConfig(config.value)
    MessagePlugin.success('配置已保存')
  } catch {
    MessagePlugin.error('保存失败')
  }
}
</script>

<template>
  <div class="ai-config-view">
    <div class="page-header">
      <SettingIcon class="page-icon" />
      <h1>AI 系统配置</h1>
      <span class="page-desc">配置 RAG 检索和模型参数</span>
    </div>

    <div class="form-card">
      <t-form :data="config" label-width="120px" v-loading="loading">
        <t-divider align="left" style="margin: 0 0 12px; font-size: 14px; font-weight: 600; color: #1d2129;">RAG 配置</t-divider>
        <t-form-item label="启用 RAG">
          <t-switch v-model="config.enable_rag" />
          <span class="help-text">开启后 AI 回答会基于笔记内容</span>
        </t-form-item>

        <t-form-item label="文本块大小">
          <t-input-number v-model="config.chunk_size" :min="100" :max="2000" style="width: 200px;" />
          <span class="help-text">每个文本块的 token 数量</span>
        </t-form-item>

        <t-form-item label="块重叠大小">
          <t-input-number v-model="config.chunk_overlap" :min="0" :max="200" style="width: 200px;" />
          <span class="help-text">相邻文本块间的重叠 token 数</span>
        </t-form-item>

        <t-form-item label="Top K">
          <t-input-number v-model="config.top_k" :min="1" :max="20" style="width: 200px;" />
          <span class="help-text">检索最相关文本块数量</span>
        </t-form-item>

        <t-form-item label="相似度阈值">
          <div style="display: flex; align-items: center; gap: 12px; width: 300px;">
            <t-slider v-model="config.score_threshold" :min="0" :max="1" :step="0.05" />
            <span class="help-text" style="margin: 0; min-width: 32px;">{{ config.score_threshold }}</span>
          </div>
        </t-form-item>

        <t-divider align="left" style="margin: 16px 0 12px; font-size: 14px; font-weight: 600; color: #1d2129;">性能设置</t-divider>

        <t-form-item label="启用流式响应">
          <t-switch v-model="config.enable_streaming" />
          <span class="help-text">开启后回答逐字输出</span>
        </t-form-item>

        <t-form-item label="最大上下文Token">
          <t-input-number v-model="config.max_context_tokens" :min="1000" :max="128000" :step="1000" style="width: 200px;" />
          <span class="help-text">对话最大上下文窗口</span>
        </t-form-item>

        <t-divider align="left" style="margin: 16px 0 12px; font-size: 14px; font-weight: 600; color: #1d2129;">提示词</t-divider>

        <t-form-item label="系统提示词">
          <t-textarea v-model="config.system_prompt" :autosize="{ minRows: 3, maxRows: 8 }" placeholder="系统提示词模板" />
        </t-form-item>

        <t-form-item>
          <t-button @click="handleSave" :loading="loading">保存配置</t-button>
        </t-form-item>
      </t-form>
    </div>
  </div>
</template>

<style scoped>
.form-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04), 0 1px 3px rgba(0, 0, 0, 0.06);
  padding: 24px;
  max-width: 720px;
}

.help-text {
  color: #86909c;
  font-size: 12px;
  margin-left: 8px;
}
</style>
