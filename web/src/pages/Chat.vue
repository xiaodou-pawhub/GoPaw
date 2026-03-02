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
            <!-- 待发送文件预览 -->
            <div v-if="pendingFile" class="pending-file">
              <n-tag closable @close="clearPendingFile" type="info">
                <template #icon>
                  <n-icon :component="AttachOutline" />
                </template>
                {{ pendingFile.filename }}
              </n-tag>
            </div>
            <div class="input-box">
              <!-- 隐藏的文件输入 -->
              <input
                ref="fileInputRef"
                type="file"
                accept=".txt,.md,.csv,.json,.yaml,.yml,.png,.jpg,.jpeg,.gif"
                style="display: none"
                @change="handleFileUpload"
              />
              <!-- 文件上传按钮 -->
              <n-button
                quaternary
                circle
                :loading="uploadingFile"
                :disabled="!appStore.isLLMConfigured || isStreaming"
                @click="triggerFileUpload"
                class="upload-btn"
              >
                <template #icon>
                  <n-icon :component="AttachOutline" />
                </template>
              </n-button>
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
                :disabled="(!inputMessage.trim() && !pendingFile) || !appStore.isLLMConfigured || isStreaming"
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
  BarChartOutline,
  AttachOutline
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { getSessions, getSessionMessages, sendChatStream, deleteSession as apiDeleteSession, getSessionStats } from '@/api/agent'
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
const fileInputRef = ref<HTMLInputElement | null>(null)
const pendingFile = ref<{ filename: string; content: string; type: string } | null>(null)
const uploadingFile = ref(false)
// 中文：当前流式请求的控制器，用于取消请求
// English: Current streaming request controller for cancellation
const streamController = ref<AbortController | null>(null)

// 用于标记当前是否正在执行新建会话过程，避免欢迎语被覆盖
const isCreatingNew = ref(false)

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

// 中文：取消流式请求
// English: Cancel streaming request
function cancelStreaming() {
  isStreaming.value = false
  isThinking.value = false
  
  // 中断正在进行的流请求
  if (streamController.value) {
    streamController.value.abort('会话切换或取消')
    streamController.value = null
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
  
  // 校验 ID 合法性：如果 ID 不存在于会话列表且不是正在新建，则立即执行降级
  const exists = sessions.value.some(s => s.id === id)
  if (!exists && !isCreatingNew.value) {
    console.warn('会话 ID 不存在，正在执行降级恢复...')
    fallbackToValidSession()
    return
  }

  // 如果是新建会话过程，保持当前生成的欢迎语，不进行后端同步
  if (isCreatingNew.value && currentSessionId.value === id) {
    isCreatingNew.value = false
    return
  }

  cancelStreaming()
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

// 降级恢复逻辑
function fallbackToValidSession() {
  if (sessions.value.length > 0) {
    router.replace({ name: 'Chat', params: { id: sessions.value[0].id } })
  } else {
    createNewSession()
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
          cancelStreaming()
          currentSessionId.value = ''
          messages.value = []
          sessionStats.value = null
        }
        
        await apiDeleteSession(id)
        await loadSessions()
        message.success(`${t('common.success')} (ID: ${id.substring(0, 8)})`)
        
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
    // 仅针对已知会话显示统计加载失败警告
    if (sessions.value.some(s => s.id === id)) {
      message.warning('无法加载 Token 统计信息')
    }
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
  cancelStreaming()
  const newId = crypto.randomUUID()
  
  // 设置状态标记
  isCreatingNew.value = true
  currentSessionId.value = newId
  messages.value = []
  sessionStats.value = null
  
  // 生成欢迎语
  messages.value.push({
    id: 'welcome-' + Date.now(),
    role: 'assistant',
    content: t('chat.welcome'),
    time: new Date().toLocaleTimeString()
  })

  // 跳转路由（watcher 会触发，但会被 isCreatingNew 拦截同步逻辑）
  router.push({ name: 'Chat', params: { id: newId } })
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

// 触发文件选择
function triggerFileUpload() {
  fileInputRef.value?.click()
}

// 处理文件上传
async function handleFileUpload(event: Event) {
  const input = event.target as HTMLInputElement
  if (!input.files?.length) return

  const file = input.files[0]
  uploadingFile.value = true

  try {
    const formData = new FormData()
    formData.append('file', file)

    const res = await fetch('/api/agent/upload', {
      method: 'POST',
      body: formData
    })

    if (!res.ok) {
      const errData = await res.json()
      throw new Error(errData.error || 'Upload failed')
    }

    const data = await res.json()
    pendingFile.value = {
      filename: data.filename,
      content: data.content,
      type: data.type
    }
    message.success(`文件 "${data.filename}" 已准备，发送消息时将附带文件内容`)
  } catch (error: any) {
    message.error(error.message || t('common.error'))
  } finally {
    uploadingFile.value = false
    // 重置 input，允许重复选择同一文件
    input.value = ''
  }
}

// 清除待发送文件
function clearPendingFile() {
  pendingFile.value = null
}

// 发送消息
async function handleSend() {
  if (!inputMessage.value.trim() && !pendingFile.value) return
  if (isStreaming.value) return
  if (!appStore.isLLMConfigured) {
    message.warning(t('setup.description'))
    return
  }

  // 构建消息内容，附加文件
  let content = inputMessage.value
  if (pendingFile.value) {
    content = `[文件: ${pendingFile.value.filename}]\n${pendingFile.value.content}\n\n${content}`
  }
  
  inputMessage.value = ''
  const fileToSend = pendingFile.value
  pendingFile.value = null
  
  const userMsg: ChatMessage = {
    id: 'msg-' + Date.now(),
    role: 'user',
    content: fileToSend ? `[附件: ${fileToSend.filename}]\n${content.replace(/\[文件:.*?\]\n[\s\S]*?\n\n/, '')}` : content,
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
  
  // 中文：使用 POST 流式请求，支持大内容
  // English: Use POST streaming request, supports large content
  // 创建新的 AbortController 用于取消请求
  const controller = new AbortController()
  streamController.value = controller
  
  try {
    await sendChatStream(currentSessionId.value, content, {
      onDelta: (delta) => {
        if (isThinking.value) {
          isThinking.value = false
          messages.value.push(assistantMsg)
        }
        assistantMsg.content += delta
        scrollToBottom()
      },
      onDone: () => {
        isStreaming.value = false
        streamController.value = null
        loadSessions()
        loadStats(currentSessionId.value)
      },
      onError: (error) => {
        isThinking.value = false
        isStreaming.value = false
        streamController.value = null
        message.error(error)
      }
    }, { signal: controller.signal })
  } catch (error) {
    isThinking.value = false
    isStreaming.value = false
    streamController.value = null
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
  cancelStreaming()
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

.pending-file {
  margin-bottom: 12px;
  padding: 0 4px;
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

.upload-btn {
  margin-bottom: 4px;
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
