<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { askStream, listConversations, getMessages } from '../api/ai'
import type { Conversation, Message } from '../api/types'

const conversations = ref<Conversation[]>([])
const currentConv = ref<string | null>(null)
const messages = ref<Message[]>([])
const input = ref('')
const loading = ref(false)
const streaming = ref(false)
const sidebarOpen = ref(true)

const chatRef = ref<HTMLElement | null>(null)

onMounted(() => { loadConversations() })

async function loadConversations() {
  try {
    const res = await listConversations()
    const d = res.data?.data || res.data || {}
    conversations.value = d || []
  } catch {
    // ignore
  }
}

async function openConversation(id: string) {
  currentConv.value = id
  try {
    const res = await getMessages(id)
    const d = res.data?.data || res.data || []
    messages.value = d || []
  } catch {
    messages.value = []
  }
  await nextTick()
  scrollToBottom()
}

async function createAndSend() {
  if (!input.value.trim() || streaming.value) return

  const text = input.value.trim()
  input.value = ''

  // Add user message optimistically
  messages.value.push({
    id: 'tmp-' + Date.now(),
    conversation_id: '',
    role: 'user',
    content: text,
    created_at: new Date().toISOString(),
  })
  await nextTick()
  scrollToBottom()

  // Build assistant placeholder
  const assistantIdx = messages.value.length
  messages.value.push({
    id: 'tmp-ai-' + Date.now(),
    conversation_id: '',
    role: 'assistant',
    content: '',
    created_at: new Date().toISOString(),
  })
  await nextTick()
  scrollToBottom()

  loading.value = true
  streaming.value = true

  let fullAnswer = ''

  const abort = askStream(
    { question: text, conversation_id: currentConv.value || undefined },
    (token: string) => {
      fullAnswer += token
      messages.value[assistantIdx].content = fullAnswer
      scrollToBottom()
    },
    () => {
      loading.value = false
      streaming.value = false
      loadConversations()
    },
    (err: Error) => {
      loading.value = false
      streaming.value = false
      MessagePlugin.error(err.message || '请求失败')
      // Remove the empty assistant message on error
      if (!fullAnswer) messages.value.pop()
    },
  )

  // Store abort on message for potential cancellation
  ;(messages.value[assistantIdx] as any)._abort = abort
}

function newConversation() {
  currentConv.value = null
  messages.value = []
}

function scrollToBottom() {
  nextTick(() => {
    if (chatRef.value) {
      chatRef.value.scrollTop = chatRef.value.scrollHeight
    }
  })
}

function formatTime(dateStr: string) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div class="ai-chat">
    <!-- Sidebar: conversation list -->
    <aside class="chat-sidebar" :class="{ collapsed: !sidebarOpen }">
      <div class="sidebar-header">
        <h2 class="sidebar-title">对话</h2>
        <t-button variant="text" size="small" @click="sidebarOpen = !sidebarOpen">
          <t-icon :name="sidebarOpen ? 'view-list' : 'view-in-ar'" />
        </t-button>
      </div>
      <t-button variant="outline" block size="small" class="new-btn" @click="newConversation">
        <t-icon name="add" /> 新对话
      </t-button>
      <div class="conv-list">
        <div
          v-for="conv in conversations"
          :key="conv.id"
          class="conv-item"
          :class="{ active: currentConv === conv.id }"
          @click="openConversation(conv.id)"
        >
          <span class="conv-title">{{ conv.title }}</span>
          <span class="conv-time">{{ formatTime(conv.updated_at) }}</span>
        </div>
      </div>
    </aside>

    <!-- Main chat area -->
    <main class="chat-main">
      <div class="chat-header">
        <t-button v-if="!sidebarOpen" variant="text" size="small" @click="sidebarOpen = true">
          <t-icon name="view-list" />
        </t-button>
        <span class="chat-title-text">{{ currentConv ? conversations.find(c => c.id === currentConv)?.title || '对话' : '新对话' }}</span>
      </div>

      <div class="chat-messages" ref="chatRef">
        <div v-if="messages.length === 0 && !loading" class="welcome">
          <t-icon name="chat" size="64px" style="color: #c0c4cc" />
          <p class="welcome-text">有什么可以帮你的？</p>
        </div>

        <div
          v-for="msg in messages"
          :key="msg.id"
          class="message"
          :class="msg.role"
        >
          <div class="avatar">
            <t-icon :name="msg.role === 'user' ? 'user' : 'bulb'" />
          </div>
          <div class="bubble">
            <pre class="msg-text">{{ msg.content || (streaming && msg.role === 'assistant' ? '正在生成...' : '') }}</pre>
          </div>
        </div>
      </div>

      <div class="chat-input">
        <textarea
          v-model="input"
          class="input-box"
          placeholder="输入你的问题..."
          rows="1"
          @keydown.enter.exact.prevent="createAndSend"
          :disabled="streaming"
        />
        <t-button
          @click="createAndSend"
          :loading="streaming"
          :disabled="!input.trim()"
          shape="circle"
          variant="base"
        >
          <t-icon name="send" />
        </t-button>
      </div>
    </main>
  </div>
</template>

<style scoped>
.ai-chat {
  display: flex;
  height: calc(100vh - 60px);
  overflow: hidden;
}

/* Sidebar */
.chat-sidebar {
  width: 260px;
  background: #f7f8fa;
  border-right: 1px solid #e5e6eb;
  display: flex;
  flex-direction: column;
  transition: width 0.2s;
  overflow: hidden;
}

.chat-sidebar.collapsed {
  width: 0;
  border-right: none;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
}

.sidebar-title {
  font-size: 16px;
  font-weight: 600;
  color: #1d2129;
  margin: 0;
}

.new-btn {
  margin: 0 12px 8px;
}

.conv-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 8px;
}

.conv-item {
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 2px;
  transition: background 0.15s;
}

.conv-item:hover {
  background: #e5e6eb;
}

.conv-item.active {
  background: #e8f3ff;
}

.conv-title {
  font-size: 13px;
  color: #1d2129;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conv-time {
  font-size: 11px;
  color: #c0c4cc;
}

/* Main */
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.chat-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  border-bottom: 1px solid #e5e6eb;
  background: #fff;
}

.chat-title-text {
  font-size: 15px;
  font-weight: 600;
  color: #1d2129;
}

/* Messages */
.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  margin-top: 20vh;
}

.welcome-text {
  font-size: 16px;
  color: #86909c;
}

.message {
  display: flex;
  gap: 12px;
  max-width: 720px;
  align-self: flex-start;
}

.message.user {
  align-self: flex-end;
  flex-direction: row-reverse;
}

.avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: #f0f1f3;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #4e5969;
}

.message.user .avatar {
  background: #e8f3ff;
  color: #1677ff;
}

.bubble {
  background: #f7f8fa;
  border-radius: 12px;
  padding: 12px 16px;
  max-width: 560px;
}

.message.user .bubble {
  background: #1677ff;
}

.msg-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
  color: #1d2129;
  background: transparent;
  font-family: inherit;
}

.message.user .msg-text {
  color: #fff;
}

/* Input */
.chat-input {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid #e5e6eb;
  background: #fff;
}

.input-box {
  flex: 1;
  border: 1px solid #e5e6eb;
  border-radius: 12px;
  padding: 12px 16px;
  font-size: 14px;
  resize: none;
  outline: none;
  line-height: 1.5;
  font-family: inherit;
  color: #1d2129;
  background: #f7f8fa;
  transition: border-color 0.2s;
  max-height: 120px;
}

.input-box:focus {
  border-color: #1677ff;
  background: #fff;
}

.input-box:disabled {
  opacity: 0.6;
}
</style>
