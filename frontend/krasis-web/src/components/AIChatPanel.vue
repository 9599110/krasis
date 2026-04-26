<script setup lang="ts">
import { ref, nextTick, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { askStream } from '../api/ai'

const visible = ref(false)
const messages = ref<Array<{ role: string; content: string }>>([])
const input = ref('')
const loading = ref(false)

const chatRef = ref<HTMLElement | null>(null)

watch(visible, () => {
  if (visible.value) {
    nextTick(() => scrollToBottom())
  }
})

function toggle() {
  visible.value = !visible.value
}

async function send() {
  if (!input.value.trim() || loading.value) return

  const text = input.value.trim()
  input.value = ''

  messages.value.push({ role: 'user', content: text })
  const aiIdx = messages.value.length
  messages.value.push({ role: 'assistant', content: '' })
  await nextTick()
  scrollToBottom()

  loading.value = true

  let fullAnswer = ''

  askStream(
    { question: text },
    (token: string) => {
      fullAnswer += token
      messages.value[aiIdx].content = fullAnswer
      scrollToBottom()
    },
    () => {
      loading.value = false
    },
    (err: Error) => {
      loading.value = false
      if (!fullAnswer) messages.value.pop()
      MessagePlugin.error(err.message || '请求失败')
    },
  )
}

function scrollToBottom() {
  nextTick(() => {
    if (chatRef.value) {
      chatRef.value.scrollTop = chatRef.value.scrollHeight
    }
  })
}

function clearChat() {
  messages.value = []
}

defineExpose({ toggle, visible })
</script>

<template>
  <teleport to="body">
    <!-- Floating button -->
    <t-button
      v-show="!visible"
      class="ai-float-btn"
      shape="circle"
      variant="base"
      @click="toggle"
    >
      <t-icon name="chat" />
    </t-button>

    <!-- Chat panel -->
    <div v-show="visible" class="ai-panel">
      <div class="ai-panel-header">
        <h3 class="ai-panel-title">AI 对话</h3>
        <div class="ai-panel-actions">
          <t-button variant="text" size="small" @click="clearChat">
            <t-icon name="refresh" />
          </t-button>
          <t-button variant="text" size="small" @click="toggle">
            <t-icon name="close" />
          </t-button>
        </div>
      </div>

      <div class="ai-panel-messages" ref="chatRef">
        <div v-if="messages.length === 0 && !loading" class="welcome">
          <t-icon name="chat" size="40px" style="color: #c0c4cc" />
          <p>有什么可以帮你的？</p>
        </div>

        <div
          v-for="(msg, idx) in messages"
          :key="idx"
          class="ai-message"
          :class="msg.role"
        >
          <div class="ai-avatar">
            <t-icon :name="msg.role === 'user' ? 'user' : 'bulb'" />
          </div>
          <div class="ai-bubble">
            <pre class="ai-msg-text">{{ msg.content || (loading && msg.role === 'assistant' ? '正在生成...' : '') }}</pre>
          </div>
        </div>
      </div>

      <div class="ai-panel-input">
        <textarea
          v-model="input"
          class="ai-input-box"
          placeholder="输入你的问题..."
          rows="1"
          @keydown.enter.exact.prevent="send"
          :disabled="loading"
        />
        <t-button
          @click="send"
          :loading="loading"
          :disabled="!input.trim()"
          shape="circle"
          variant="base"
        >
          <t-icon name="send" />
        </t-button>
      </div>
    </div>
  </teleport>
</template>

<style scoped>
/* Floating button */
.ai-float-btn {
  position: fixed;
  bottom: 24px;
  right: 24px;
  width: 56px;
  height: 56px;
  z-index: 999;
  box-shadow: 0 4px 16px rgba(22, 119, 255, 0.3);
}

/* Panel */
.ai-panel {
  position: fixed;
  bottom: 0;
  right: 0;
  width: 420px;
  height: 100vh;
  background: #fff;
  border-left: 1px solid #e5e6eb;
  display: flex;
  flex-direction: column;
  z-index: 1000;
  box-shadow: -4px 0 24px rgba(0, 0, 0, 0.08);
}

/* Header */
.ai-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid #e5e6eb;
  background: #fff;
}

.ai-panel-title {
  font-size: 16px;
  font-weight: 600;
  color: #1d2129;
  margin: 0;
}

.ai-panel-actions {
  display: flex;
  gap: 4px;
}

/* Messages */
.ai-panel-messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  margin-top: 40vh;
  color: #86909c;
}

.ai-message {
  display: flex;
  gap: 10px;
  max-width: 360px;
  align-self: flex-start;
}

.ai-message.user {
  align-self: flex-end;
  flex-direction: row-reverse;
}

.ai-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #f0f1f3;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #4e5969;
  font-size: 14px;
}

.ai-message.user .ai-avatar {
  background: #e8f3ff;
  color: #1677ff;
}

.ai-bubble {
  background: #f7f8fa;
  border-radius: 10px;
  padding: 10px 14px;
  max-width: 300px;
}

.ai-message.user .ai-bubble {
  background: #1677ff;
}

.ai-msg-text {
  margin: 0;
  font-size: 13px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
  color: #1d2129;
  background: transparent;
  font-family: inherit;
}

.ai-message.user .ai-msg-text {
  color: #fff;
}

/* Input */
.ai-panel-input {
  display: flex;
  align-items: flex-end;
  gap: 10px;
  padding: 14px 16px;
  border-top: 1px solid #e5e6eb;
  background: #fff;
}

.ai-input-box {
  flex: 1;
  border: 1px solid #e5e6eb;
  border-radius: 10px;
  padding: 10px 14px;
  font-size: 13px;
  resize: none;
  outline: none;
  line-height: 1.5;
  font-family: inherit;
  color: #1d2129;
  background: #f7f8fa;
  transition: border-color 0.2s;
  max-height: 100px;
}

.ai-input-box:focus {
  border-color: #1677ff;
  background: #fff;
}

.ai-input-box:disabled {
  opacity: 0.6;
}

/* Responsive */
@media (max-width: 768px) {
  .ai-panel {
    width: 100vw;
  }

  .ai-message {
    max-width: 280px;
  }

  .ai-bubble {
    max-width: 240px;
  }
}
</style>
