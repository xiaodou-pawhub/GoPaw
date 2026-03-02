<template>
  <div class="chat-page">
    <div class="chat-layout">
      <!-- 左侧会话列表 -->
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
              class="session-list-item"
            >
              <div class="session-item">
                <div class="session-info">
                  <n-icon :component="ChatbubbleOutline" />
                  <span class="session-name">{{ session.id.substring(0, 8) }}...</span>
                </div>
                <!-- 会话删除按钮 -->
                <n-button
                  class="delete-session-btn"
                  quaternary
                  circle
                  size="small"
                  type="error"
                  @click="(e) => handleDeleteSession(session.id, e)"
                >
                  <template #icon>
                    <n-icon :component="TrashOutline" />
                  </template>
                </n-button>
              </div>
            </n-list-item>
          </n-list>
        </n-scrollbar>
      </div>

      <!-- 右侧聊天窗口 -->
      <div class="chat-main">
        <n-card :bordered="false" class="chat-card" content-style="padding: 0; display: flex; flex-direction: column; height: 100%;">
          <template #header>
            <div class="chat-header">
              <div class="header-left">
                <n-text strong size="large">{{ t('chat.title') }}</n-text>
                <n-text depth="3" small style="margin-left: 12px;">ID: {{ currentSessionId }}</n-text>
              </div>
              
              <!-- Token 统计展示 -->
              <div v-if="sessionStats" class="header-right stats-container">
                <n-tooltip trigger="hover">
                  <template #trigger>
                    <div class="stats-badge">
                      <n-icon :component="BarChartOutline" />
                      <span class="stats-text">{{ formatTokens(sessionStats.total_tokens) }} tokens</span>
                    </div>
                  </template>
                  <div class="stats-detail">
                    <div>消息: {{ sessionStats.message_count }}</div>
                    <div>用户: {{ formatTokens(sessionStats.user_tokens) }}</div>
                    <div>助手: {{ formatTokens(sessionStats.assist_tokens) }}</div>
                  </div>
                </n-tooltip>
              </div>
            </div>
          </template>

          <!-- 消息显示区 -->
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
            
            <!-- 思考中动画 -->
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

          <!-- 底部输入区 -->
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
import { ref, onMounted, nextTick, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  NCard, NButton, NIcon, NInput, NList, NListItem, NScrollbar,
  NAvatar, NText, NEmpty, NSpin, NTooltip, useMessage, useDialog
} from 'naive-ui'
import {
  AddOutline,
  ChatbubbleOutline,
  PersonOutline,
  PawOutline,
  SendOutline,
  TrashOutline,
  BarChartOutline
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { getSessions, getSessionMessages, getChatStreamUrl, deleteSession as apiDeleteSession, getSessionStats } from '@/api/agent'
import type { ChatMessage, SessionInfo, SessionStats } from '@/types'
import markdownIt from 'markdown-it'
import highlightjs from 'highlight.js'
import 'highlight.js/styles/github.css'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()
const router = useRouter()
const route = useRoute()

// 配置 Markdown 渲染引擎，显式禁用 HTML
const md = markdownIt({
  html: false,
  linkify: true,
  typographer: true,
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
const sessionStats = ref<SessionStats | null>(null)
const inputMessage = ref('')
const isThinking = ref(false)
const isStreaming = ref(false)
const messagesRef = ref<HTMLElement | null>(null)
let currentEventSource: EventSource | null = null

// 加载会话列表
async function loadSessions() {
  try {
    const list = await getSessions()
    sessions.value = list
    return list
  } catch (error) {
    console.error('Failed to load sessions:', error)
    return []
  }
}

// 安全关闭 SSE
function closeCurrentSSE() {
  if (currentEventSource) {
    currentEventSource.close()
    currentEventSource = null
    isStreaming.value = false
    isThinking.value = false
  }
}

// 切换会话（路由驱动）
function selectSession(id: string) {
  if (route.params.id === id) return
  router.push({ name: 'Chat', params: { id } })
}

// 加载特定会话数据
async function handleSessionSwitch(id: string) {
  if (currentSessionId.value === id && messages.value.length > 0) return
  
  closeCurrentSSE()
  currentSessionId.value = id
  loadStats(id)
  try {
    const history = await getSessionMessages(id)
    messages.value = history
    scrollToBottom()
  } catch (error) {
    console.error('Failed to load session history:', error)
    message.error('加载历史记录失败')
  }
}

// 删除会话
async function handleDeleteSession(id: string, e: MouseEvent) {
  e.stopPropagation()
  dialog.warning({
    title: t('common.confirm'),
    content: t('chat.deleteConfirm'),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        if (currentSessionId.value === id) {
          closeCurrentSSE()
          currentSessionId.value = ''
          messages.value = []
          sessionStats.value = null
        }
        
        await apiDeleteSession(id)
        const newList = await loadSessions()
        message.success(`${t('common.success')} (ID: ${id.substring(0, 8)})`)
        
        // 如果删除了当前正在看的会话，回退到主路径
        if (route.params.id === id) {
          router.push({ name: 'Chat' })
        }
      } catch (error) {
        message.error(t('common.error'))
      }
    }
  })
}

// 加载统计
async function loadStats(id: string) {
  try {
    sessionStats.value = await getSessionStats(id)
  } catch (error) {
    console.error('Failed to load stats:', error)
    message.warning('无法加载 Token 统计信息')
  }
}

// 格式化 Token
function formatTokens(n: number): string {
  if (!n) return '0'
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return n.toString()
}

// 创建新会话
function createNewSession() {
  closeCurrentSSE()
  const newId = crypto.randomUUID()
  router.push({ name: 'Chat', params: { id: newId } })
  
  messages.value = []
  sessionStats.value = null
  messages.value.push({
    id: 'welcome-' + Date.now(),
    role: 'assistant',
    content: t('chat.welcome'),
    time: new Date().toLocaleTimeString()
  })
}

// 渲染 Markdown
function renderMarkdown(content: string) {
  return md.render(content)
}

// 滚动到底部
async function scrollToBottom() {
  await nextTick()
  if (messagesRef.value) {
    messagesRef.value.scrollTo({
      top: messagesRef.value.scrollHeight,
      behavior: 'smooth'
    })
  }
}

// 键盘处理
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

// 发送消息
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

  const assistantMsgId = 'msg-' + (Date.now() + 1)
  const assistantMsg: ChatMessage = {
    id: assistantMsgId,
    role: 'assistant',
    content: '',
    time: new Date().toLocaleTimeString()
  }
  
  const url = getChatStreamUrl(currentSessionId.value, content)
  currentEventSource = new EventSource(url)

  currentEventSource.onmessage = (event) => {
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
        closeCurrentSSE()
        loadSessions()
        loadStats(currentSessionId.value)
      }
      if (data.error) {
        message.error(data.error)
        closeCurrentSSE()
      }
    } catch (e) {
      console.error('SSE parse error:', e)
    }
  }

  currentEventSource.onerror = (err) => {
    console.error('SSE error:', err)
    closeCurrentSSE()
    message.error(t('common.error'))
  }
}

// 监听 ID 变化实现刷新恢复
watch(
  () => route.params.id,
  (newId) => {
    if (newId) {
      handleSessionSwitch(newId as string)
    } else {
      if (sessions.value.length > 0) {
        selectSession(sessions.value[0].id)
      } else {
        createNewSession()
      }
    }
  }
)

onMounted(async () => {
  const list = await loadSessions()
  const routeId = route.params.id as string
  
  if (routeId) {
    handleSessionSwitch(routeId)
  } else if (list.length > 0) {
    selectSession(list[0].id)
  } else {
    createNewSession()
  }
})

onUnmounted(() => {
  closeCurrentSSE()
})
</script>

<style scoped lang="scss">
.chat-page {
  height: calc(100vh - 112px);
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

.session-list-item {
  position: relative;
  
  .delete-session-btn {
    opacity: 0;
    transition: opacity 0.2s;
  }
  
  &:hover .delete-session-btn {
    opacity: 1;
  }
}

.session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.session-info {
  display: flex;
  align-items: center;
  gap: 10px;
  overflow: hidden;
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
  justify-content: space-between;
  width: 100%;
}

.stats-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: #f3f4f6;
  border-radius: 6px;
  font-size: 13px;
  color: #666;
  cursor: default;
}

.stats-detail {
  line-height: 1.6;
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

/* Markdown 代码高亮样式适配 */
:deep(.hljs) {
  padding: 12px;
  border-radius: 8px;
  margin: 8px 0;
  font-family: 'Fira Code', 'Courier New', Courier, monospace;
}
</style>
