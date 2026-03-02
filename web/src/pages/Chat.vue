<template>
  <div class="chat-page">
    <div class="chat-layout">
      <!-- 中文：左侧会话列表 / English: Left session list -->
      <div class="session-sidebar">
        <n-button type="primary" dashed block @click="createNewSession" class="new-chat-btn">
          <template #icon>
            <n-icon :component="AddOutline" />
          </template>
          {{ t('chat.newChat') }}
        </n-button>
        
        <n-scrollbar class="session-list">
          <n-list hoverable clickable>
            <n-list-item
              v-for="session in sessions"
              :key="session.id"
              :class="{ active: currentSessionId === session.id }"
              @click="selectSession(session.id)"
            >
              <div class="session-item">
                <n-icon :component="ChatbubbleOutline" />
                <span class="session-name">{{ session.id.substring(0, 8) }}...</span>
              </div>
            </n-list-item>
          </n-list>
        </n-scrollbar>
      </div>

      <!-- 中文：右侧聊天窗口 / English: Right chat window -->
      <div class="chat-main">
        <n-card :bordered="false" class="chat-card" content-style="padding: 0; display: flex; flex-direction: column; height: 100%;">
          <template #header>
            <div class="chat-header">
              <n-text strong size="large">{{ t('chat.title') }}</n-text>
              <n-text depth="3" small style="margin-left: 12px;">ID: {{ currentSessionId }}</n-text>
            </div>
          </template>

          <!-- 中文：消息显示区 / English: Message display area -->
          <div ref="messagesRef" class="messages-area">
            <div v-if="messages.length === 0" class="empty-state">
              <n-empty :description="t('chat.welcome')" />
            </div>
            <div v-for="msg in messages" :key="msg.id" class="message-row" :class="msg.role">
              <div class="avatar">
                <n-avatar round :size="36" :style="{ backgroundColor: msg.role === 'user' ? '#18a058' : '#1a1a2e' }">
                  <n-icon :component="msg.role === 'user' ? PersonOutline : PawOutline" />
                </n-avatar>
              </div>
              <div class="message-content-wrapper">
                <div class="message-bubble" :class="msg.role">
                  <div v-if="msg.role === 'assistant'" class="markdown-body" v-html="renderMarkdown(msg.content)"></div>
                  <div v-else class="text-content">{{ msg.content }}</div>
                </div>
                <div class="message-time">{{ msg.time }}</div>
              </div>
            </div>
            
            <!-- 中文：思考中动画 / English: Thinking animation -->
            <div v-if="isThinking" class="message-row assistant">
              <div class="avatar">
                <n-avatar round :size="36" style="background-color: #1a1a2e">
                  <n-icon :component="PawOutline" />
                </n-avatar>
              </div>
              <div class="message-content-wrapper">
                <div class="message-bubble assistant thinking">
                  <n-spin size="small" />
                  <span class="thinking-text">{{ t('chat.thinking') }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 中文：底部输入区 / English: Bottom input area -->
          <div class="input-container">
            <div class="input-box">
              <n-input
                v-model:value="inputMessage"
                type="textarea"
                :placeholder="t('chat.placeholder')"
                :autosize="{ minRows: 1, maxRows: 6 }"
                :disabled="!appStore.isLLMConfigured || isStreaming"
                @keydown="handleKeydown"
                class="chat-input"
              />
              <n-button
                type="primary"
                circle
                :disabled="!inputMessage.trim() || !appStore.isLLMConfigured || isStreaming"
                :loading="isStreaming"
                @click="handleSend"
                class="send-btn"
              >
                <template #icon>
                  <n-icon :component="SendOutline" />
                </template>
              </n-button>
            </div>
          </div>
        </n-card>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, onMounted, nextTick, watch } from 'vue'
import {
  NCard, NButton, NIcon, NInput, NList, NListItem, NScrollbar,
  NAvatar, NText, NEmpty, NSpin, useMessage
} from 'naive-ui'
import {
  AddOutline,
  ChatbubbleOutline,
  PersonOutline,
  PawOutline,
  SendOutline
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { getSessions, getSessionMessages, getChatStreamUrl } from '@/api/agent'
import type { ChatMessage, SessionInfo } from '@/types'
import markdownIt from 'markdown-it'
import highlightjs from 'highlight.js'
import 'highlight.js/styles/github.css'

const { t } = useI18n()
const message = useMessage()
const appStore = useAppStore()

// 中文：配置 Markdown 渲染引擎
// English: Configure Markdown rendering engine
const md = markdownIt({
  highlight: function (str, lang) {
    if (lang && highlightjs.getLanguage(lang)) {
      try {
        return '<pre class="hljs"><code>' +
               highlightjs.highlight(str, { language: lang, ignoreIllegals: true }).value +
               '</code></pre>';
      } catch (__) {}
    }
    return '<pre class="hljs"><code>' + md.utils.escapeHtml(str) + '</code></pre>';
  }
})

const sessions = ref<SessionInfo[]>([])
const currentSessionId = ref('')
const messages = ref<ChatMessage[]>([])
const inputMessage = ref('')
const isThinking = ref(false)
const isStreaming = ref(false)
const messagesRef = ref<HTMLElement | null>(null)

// 中文：加载所有会话
// English: Load all sessions
async function loadSessions() {
  try {
    sessions.value = await getSessions()
    if (sessions.value.length > 0 && !currentSessionId.value) {
      selectSession(sessions.value[0].id)
    }
  } catch (error) {
    console.error('Failed to load sessions:', error)
  }
}

// 中文：创建新会话
// English: Create a new session
function createNewSession() {
  const newId = crypto.randomUUID()
  currentSessionId.value = newId
  messages.value = []
  // 中文：添加欢迎语
  // English: Add welcome message
  messages.value.push({
    id: 'welcome-' + Date.now(),
    role: 'assistant',
    content: t('chat.welcome'),
    time: new Date().toLocaleTimeString()
  })
}

// 中文：选择会话并加载历史记录
// English: Select a session and load history
async function selectSession(id: string) {
  currentSessionId.value = id
  try {
    const history = await getSessionMessages(id)
    messages.value = history
    scrollToBottom()
  } catch (error) {
    console.error('Failed to load session history:', error)
    message.error('加载历史记录失败 / Failed to load history')
  }
}

// 中文：渲染 Markdown
// English: Render Markdown
function renderMarkdown(content: string) {
  return md.render(content)
}

// 中文：平滑滚动到底部
// English: Smooth scroll to bottom
async function scrollToBottom() {
  await nextTick()
  if (messagesRef.value) {
    messagesRef.value.scrollTo({
      top: messagesRef.value.scrollHeight,
      behavior: 'smooth'
    })
  }
}

// 中文：键盘事件处理
// English: Keyboard event handler
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

// 中文：发送消息并处理流式响应
// English: Send message and handle streaming response
async function handleSend() {
  if (!inputMessage.value.trim() || isStreaming.value) return
  
  if (!appStore.isLLMConfigured) {
    message.warning(t('setup.description'))
    return
  }

  const content = inputMessage.value
  inputMessage.value = ''
  
  const userMsg: ChatMessage = {
    id: 'msg-' + Date.now(),
    role: 'user',
    content: content,
    time: new Date().toLocaleTimeString()
  }
  
  messages.value.push(userMsg)
  await scrollToBottom()
  
  isThinking.value = true
  isStreaming.value = true

  // 中文：准备助手消息占位
  // English: Prepare assistant message placeholder
  const assistantMsgId = 'msg-' + (Date.now() + 1)
  const assistantMsg: ChatMessage = {
    id: assistantMsgId,
    role: 'assistant',
    content: '',
    time: new Date().toLocaleTimeString()
  }
  
  // 中文：使用 SSE (Server-Sent Events) 获取流式响应
  // English: Use SSE (Server-Sent Events) for streaming response
  const url = getChatStreamUrl(currentSessionId.value, content)
  const eventSource = new EventSource(url)

  eventSource.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      
      if (isThinking.value) {
        isThinking.value = false
        messages.value.push(assistantMsg)
      }

      if (data.delta) {
        assistantMsg.content += data.delta
        scrollToBottom()
      }

      if (data.done) {
        eventSource.close()
        isStreaming.value = false
        loadSessions() // 中文：刷新会话列表 / Refresh session list
      }

      if (data.error) {
        message.error(data.error)
        eventSource.close()
        isStreaming.value = false
        isThinking.value = false
      }
    } catch (e) {
      console.error('SSE parse error:', e)
    }
  }

  eventSource.onerror = (err) => {
    console.error('SSE error:', err)
    eventSource.close()
    isStreaming.value = false
    isThinking.value = false
    message.error(t('common.error'))
  }
}

onMounted(() => {
  loadSessions()
  if (!currentSessionId.value) {
    createNewSession()
  }
})
</script>

<style scoped lang="scss">
.chat-page {
  height: calc(100vh - 112px); // Header + Content padding
}

.chat-layout {
  display: flex;
  height: 100%;
  gap: 20px;
}

.session-sidebar {
  width: 260px;
  display: flex;
  flex-direction: column;
  background: #fff;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.05);
}

.new-chat-btn {
  margin-bottom: 16px;
}

.session-list {
  flex: 1;
}

.session-item {
  display: flex;
  align-items: center;
  gap: 10px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.active {
  background-color: #f3f4f6;
  border-radius: 8px;
}

.chat-main {
  flex: 1;
  height: 100%;
}

.chat-card {
  height: 100%;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.05);
}

.chat-header {
  display: flex;
  align-items: center;
}

.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  background-color: #fafafa;
}

.empty-state {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.message-row {
  display: flex;
  gap: 16px;
  margin-bottom: 24px;
  max-width: 85%;

  &.user {
    margin-left: auto;
    flex-direction: row-reverse;
  }
}

.message-content-wrapper {
  display: flex;
  flex-direction: column;
}

.message-bubble {
  padding: 12px 16px;
  border-radius: 12px;
  position: relative;
  font-size: 15px;
  line-height: 1.6;

  &.user {
    background-color: #18a058;
    color: #fff;
    border-top-right-radius: 2px;
  }

  &.assistant {
    background-color: #fff;
    color: #333;
    border-top-left-radius: 2px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  }
  
  &.thinking {
    display: flex;
    align-items: center;
    gap: 8px;
    background-color: #f0f0f0;
  }
}

.thinking-text {
  font-size: 14px;
  color: #666;
}

.message-time {
  margin-top: 6px;
  font-size: 12px;
  color: #999;
  text-align: inherit;
}

.user .message-time {
  text-align: right;
}

.input-container {
  padding: 20px 24px;
  background: #fff;
  border-top: 1px solid #eee;
}

.input-box {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  background: #f9fafb;
  padding: 8px 12px;
  border-radius: 12px;
  border: 1px solid #e5e7eb;
  transition: all 0.3s;

  &:focus-within {
    border-color: #18a058;
    box-shadow: 0 0 0 2px rgba(24, 160, 88, 0.1);
  }
}

.chat-input {
  :deep(.n-input__border), :deep(.n-input__state-border) {
    border: none !important;
  }
  :deep(.n-input-wrapper) {
    padding: 0;
  }
}

.send-btn {
  margin-bottom: 4px;
}

/* 中文：Markdown 代码高亮样式适配 / English: Markdown code highlight styles */
:deep(.hljs) {
  padding: 12px;
  border-radius: 8px;
  margin: 8px 0;
  font-family: 'Fira Code', 'Courier New', Courier, monospace;
}
</style>
