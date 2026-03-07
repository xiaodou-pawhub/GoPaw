<template>
  <div class="chat-page">
    <div class="chat-layout">
      <!-- 左侧会话列表 -->
      <div class="session-sidebar">
        <n-button type="primary" block secondary @click="createNewSession" class="new-chat-btn">
          <template #icon><n-icon :component="AddOutline" /></template>
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
                  <span class="session-name">{{ getSessionDisplayName(session) }}</span>
                </div>
                <div class="session-actions">
                  <n-button
                    class="action-btn"
                    quaternary circle size="tiny"
                    @click.stop="startRenameSession(session)"
                  >
                    <template #icon><n-icon :component="CreateOutline" size="14" /></template>
                  </n-button>
                  <n-button
                    class="delete-session-btn"
                    quaternary circle size="tiny" type="error"
                    @click.stop="() => handleDeleteSession(session.id, resetCurrentSessionState)"
                  >
                    <template #icon><n-icon :component="TrashOutline" size="14" /></template>
                  </n-button>
                </div>
              </div>
            </n-list-item>
          </n-list>
        </n-scrollbar>
      </div>

          <!-- 右侧聊天窗口 -->
          <div class="chat-main">
            <div class="chat-header">
              <div class="header-left">
                <n-h3 class="session-title">{{ currentSessionName || t('chat.title') }}</n-h3>
              </div>
              
              <div v-if="sessionStats" class="header-right">
                <n-tooltip trigger="hover">
                  <template #trigger>
                    <div class="stats-badge">
                      <n-icon :component="BarChartOutline" />
                      <span>{{ formatTokens(sessionStats.total_tokens) }} tokens</span>
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

            <!-- 消息显示区 -->
            <div ref="messagesRef" class="messages-area">
              <div v-if="messages.length === 0" class="empty-state">
                <n-empty :description="t('chat.welcome')" />
              </div>
              <div v-for="msg in messages" :key="msg.id" class="message-row" :class="msg.role">
                <div class="avatar">
                   <n-avatar v-if="msg.role === 'user'" round :size="36" :class="{ 'user-avatar': msg.role === 'user' }">
                     <n-icon :component="PersonOutline" />
                   </n-avatar>
                   <img v-else src="/assets/logo.png" alt="GoPaw" class="avatar-img" />
                </div>
            <div class="message-content-wrapper">
              <div class="message-bubble" :class="msg.role">
                <div v-if="msg.role === 'assistant'" class="markdown-body" v-html="renderMarkdown(msg.content)"></div>
                <div v-else class="text-content">{{ msg.content }}</div>
              </div>
              <div class="message-meta">
                <span class="message-time">{{ msg.time }}</span>
                <n-button
                  quaternary size="tiny"
                  class="copy-btn"
                  @click="copyMessage(msg.content)"
                >
                  <template #icon><n-icon :component="CopyOutline" size="14" /></template>
                </n-button>
              </div>
            </div>
          </div>
          
          <!-- 思考中 / 工具调用进度 -->
          <div v-if="isThinking" class="message-row assistant">
            <div class="avatar">
              <img src="/assets/logo.png" alt="GoPaw" class="avatar-img" />
            </div>
            <div class="message-content-wrapper">
              <div class="message-bubble assistant thinking">
                <n-spin size="small" />
                <div v-if="toolProgress.length > 0" class="tool-progress-list">
                  <div
                    v-for="(tool, i) in toolProgress"
                    :key="i"
                    class="tool-card"
                    :class="tool.status"
                  >
                    <div class="tool-card-header" @click="tool.expanded = !tool.expanded">
                      <span class="tool-status-dot" :class="tool.status" />
                      <span class="tool-card-name">{{ tool.name }}</span>
                      <span v-if="tool.elapsedMs !== undefined" class="tool-elapsed">{{ tool.elapsedMs }}ms</span>
                      <span v-else class="tool-spinner" />
                      <span v-if="tool.summary" class="tool-expand-icon">{{ tool.expanded ? '▲' : '▼' }}</span>
                    </div>
                    <div v-if="tool.expanded && tool.summary" class="tool-card-body">
                      {{ tool.summary }}
                    </div>
                  </div>
                </div>
                <span v-else class="thinking-text">{{ t('chat.thinking') }}</span>
              </div>
            </div>
          </div>

          <!-- 打字机渲染中的回复 -->
          <div v-if="streamingContent" class="message-row assistant">
            <div class="avatar">
              <img src="/assets/logo.png" alt="GoPaw" class="avatar-img" />
            </div>
            <div class="message-content-wrapper">
              <div class="message-bubble assistant">
                <div class="markdown-body" v-html="renderMarkdown(streamingContent)"></div>
              </div>
            </div>
          </div>
        </div>

        <!-- 现代化底部输入区 -->
        <div class="input-container">
          <div class="input-card">
            <div v-if="pendingFile" class="file-preview-area">
              <n-tag type="success" closable @close="pendingFile = null" size="medium" round>
                <template #icon><n-icon :component="DocumentTextOutline" /></template>
                {{ pendingFile.name }}
              </n-tag>
            </div>
            
            <div class="input-inner">
              <n-input
                v-model:value="inputMessage"
                type="textarea"
                :placeholder="t('chat.placeholder')"
                :autosize="{ minRows: 2, maxRows: 12 }"
                :disabled="!appStore.isLLMConfigured || isStreaming"
                @keydown="handleKeydown"
                class="chat-textarea"
              />
              
              <div class="input-actions-row">
                <div class="left-actions">
                  <n-upload
                    v-if="!isStreaming"
                    action="/api/agent/upload"
                    :show-file-list="false"
                    @finish="handleUploadFinish"
                  >
                    <n-button quaternary circle size="medium">
                      <template #icon><n-icon :component="AttachOutline" /></template>
                    </n-button>
                  </n-upload>
                  <n-button v-else quaternary circle size="medium" type="error" @click="stopChatStream">
                    <template #icon><n-icon :component="StopCircleOutline" /></template>
                  </n-button>
                </div>
                
                <div class="right-actions">
                  <n-button
                    type="primary"
                    circle
                    size="medium"
                    :disabled="(!inputMessage.trim() && !pendingFile) || !appStore.isLLMConfigured || isStreaming"
                    :loading="isStreaming"
                    @click="handleSend"
                    class="send-btn"
                  >
                    <template #icon><n-icon :component="ArrowUpOutline" /></template>
                  </n-button>
                </div>
              </div>
            </div>
          </div>
          <div class="input-footer-tip">
            GoPaw AI 助手可能会产生错误，请核实重要信息。
          </div>
        </div>
      </div>
    </div>

    <!-- 重命名对话框 -->
    <n-modal v-model:show="showRenameModal" preset="dialog" :title="t('chat.renameSession')">
      <n-input v-model:value="renameValue" :placeholder="t('chat.sessionNamePlaceholder')" @keydown.enter="confirmRename" />
      <template #action>
        <n-button @click="showRenameModal = false">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" @click="confirmRename">{{ t('common.confirm') }}</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, onUnmounted, watch, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  NButton, NIcon, NInput, NList, NListItem, NScrollbar,
  NAvatar, NEmpty, NSpin, NTooltip, NUpload, NTag, NModal, NH3, useMessage
} from 'naive-ui'
import {
  AddOutline, ChatbubbleOutline, PersonOutline,
  ArrowUpOutline, TrashOutline, BarChartOutline, AttachOutline, StopCircleOutline,
  CreateOutline, CopyOutline, DocumentTextOutline
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { getSessionMessages, getSessionStats, getSessions, deleteSession, updateSessionName, sendChatStream } from '@/api/agent'
import type { ChatMessage, SessionStats, SessionInfo } from '@/types'
import { default as markdownIt } from 'markdown-it'
import highlightjs from 'highlight.js'
import 'highlight.js/styles/github.css'

const { t } = useI18n()
const message = useMessage()
const appStore = useAppStore()
const route = useRoute()
const router = useRouter()

const sessions = ref<SessionInfo[]>([])
const currentSessionId = ref('')
const messages = ref<ChatMessage[]>([])
const sessionStats = ref<SessionStats | null>(null)
const inputMessage = ref('')
const messagesRef = ref<HTMLElement | null>(null)
const isCreatingNew = ref(false)
const pendingFile = ref<{ name: string; content: string; type: string } | null>(null)

const isThinking = ref(false)
const isStreaming = ref(false)
const abortController = ref<AbortController | null>(null)

// 工具调用进度
interface ToolCallProgress {
  name: string
  status: 'calling' | 'done' | 'error'
  startMs: number
  elapsedMs?: number
  summary?: string
  expanded: boolean
}
const toolProgress = ref<ToolCallProgress[]>([])

// streamingContent：正在打字机渲染的回复文字
const streamingContent = ref('')

// 打字机内部状态
let typewriterQueue = ''
let typewriterTimer: ReturnType<typeof setInterval> | null = null

function startTypewriter() {
  if (typewriterTimer !== null) return
  typewriterTimer = setInterval(() => {
    if (typewriterQueue.length === 0) return
    streamingContent.value += typewriterQueue.slice(0, 6)
    typewriterQueue = typewriterQueue.slice(6)
    scrollToBottom()
  }, 20)
}

function stopTypewriter() {
  if (typewriterTimer !== null) {
    clearInterval(typewriterTimer)
    typewriterTimer = null
  }
  typewriterQueue = ''
}

async function flushTypewriter() {
  if (typewriterTimer !== null) {
    clearInterval(typewriterTimer)
    typewriterTimer = null
  }
  return new Promise<void>((resolve) => {
    typewriterTimer = setInterval(() => {
      if (typewriterQueue.length === 0) {
        stopTypewriter()
        resolve()
        return
      }
      streamingContent.value += typewriterQueue.slice(0, 100)
      typewriterQueue = typewriterQueue.slice(100)
      scrollToBottom()
    }, 10)
  })
}

// 重命名状态
const showRenameModal = ref(false)
const renameValue = ref('')
const renamingSession = ref<SessionInfo | null>(null)

// 当前会话名称
const currentSessionName = computed(() => {
  const session = sessions.value.find(s => s.id === currentSessionId.value)
  return session?.name || ''
})

// 获取会话显示名称
function getSessionDisplayName(session: SessionInfo): string {
  if (session.name) return session.name
  return session.id.substring(0, 8)
}

// 开始重命名
function startRenameSession(session: SessionInfo) {
  renamingSession.value = session
  renameValue.value = session.name || ''
  showRenameModal.value = true
}

// 确认重命名
async function confirmRename() {
  if (!renamingSession.value || !renameValue.value.trim()) return
  try {
    await updateSessionName(renamingSession.value.id, renameValue.value.trim())
    const idx = sessions.value.findIndex(s => s.id === renamingSession.value!.id)
    if (idx !== -1) sessions.value[idx].name = renameValue.value.trim()
    showRenameModal.value = false
    message.success(t('common.success'))
  } catch (e) {
    message.error(t('common.error'))
  }
}

// 复制消息
async function copyMessage(content: string) {
  try {
    await navigator.clipboard.writeText(content)
    message.success(t('chat.copied'))
  } catch {
    message.error(t('chat.copyFailed'))
  }
}

// 格式化 Token
const formatTokens = (n: number) => {
  if (!n) return '0'
  return n >= 1000000 ? `${(n / 1000000).toFixed(1)}M` : n >= 1000 ? `${(n / 1000).toFixed(1)}k` : n.toString()
}

// 处理上传完成
function handleUploadFinish({ file, event }: { file: any, event?: ProgressEvent }) {
  const response = (event?.target as any)?.response
  try {
    const res = JSON.parse(response)
    pendingFile.value = {
      name: res.filename || file.name,
      content: res.content || '',
      type: res.type || 'text/plain'
    }
    message.success(`文件上传成功: ${file.name}`)
  } catch (e) {
    message.error('文件解析失败，请重试')
    pendingFile.value = null
  }
}

// Markdown 渲染
const md: any = markdownIt({
  html: false, linkify: true, typographer: true,
  highlight: (str: string, lang: string): string => {
    if (lang && highlightjs.getLanguage(lang)) {
      try { return `<pre class="hljs"><code>${highlightjs.highlight(str, { language: lang, ignoreIllegals: true }).value}</code></pre>` } catch (__) {}
    }
    return `<pre class="hljs"><code>${md.utils.escapeHtml(str)}</code></pre>`
  }
})
const renderMarkdown = (c: string) => md.render(c)

// 停止流式对话并清理状态
function stopChatStream() {
  if (abortController.value) {
    abortController.value.abort()
    abortController.value = null
  }
  isStreaming.value = false
  isThinking.value = false
  toolProgress.value = []
  stopTypewriter()
  streamingContent.value = ''
}

// 加载会话列表
async function loadSessions(): Promise<SessionInfo[]> {
  try {
    const list = await getSessions()
    sessions.value = list
    return list
  } catch (e) {
    message.error('加载会话列表失败')
    return []
  }
}

// 选择会话
function selectSession(id: string) {
  if (currentSessionId.value === id) return
  router.push({ name: 'Chat', params: { id } })
}

// 处理会话切换
async function handleSessionSwitch(id: string) {
  if (currentSessionId.value === id && messages.value.length > 0) return
  if (isCreatingNew.value && currentSessionId.value === id) {
    isCreatingNew.value = false
    return
  }
  if (!sessions.value.some(s => s.id === id) && !isCreatingNew.value) {
    if (sessions.value.length > 0) selectSession(sessions.value[0].id)
    else createNewSession()
    return
  }
  stopChatStream()
  currentSessionId.value = id
  loadStats(id)
  try {
    const history = await getSessionMessages(id)
    messages.value = history
    scrollToBottom()
  } catch (error) {
    message.error('加载历史记录失败')
  }
}

// 创建新会话
function createNewSession() {
  stopChatStream()
  pendingFile.value = null
  const newId = generateUUID()
  isCreatingNew.value = true
  currentSessionId.value = newId
  messages.value = [{
    id: 'welcome-' + Date.now(), role: 'assistant',
    content: t('chat.welcome'), time: new Date().toLocaleTimeString()
  }]
  sessionStats.value = null
  router.push({ name: 'Chat', params: { id: newId } })
}

// 生成 UUID（兼容方案）
function generateUUID(): string {
  // 优先使用 crypto.randomUUID()（如果可用）
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  
  // 降级方案：使用 Math.random() 生成 UUID
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0
    const v = c === 'x' ? r : (r & 0x3 | 0x8)
    return v.toString(16)
  })
}

// 加载统计数据
async function loadStats(id: string) {
  try {
    sessionStats.value = await getSessionStats(id)
  } catch (error) {
    // 忽略加载失败
  }
}

// 输入框回车
const handleKeydown = (e: KeyboardEvent) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() } }

// 自动滚动
const scrollToBottom = async () => {
  await nextTick()
  messagesRef.value?.scrollTo({ top: messagesRef.value.scrollHeight, behavior: 'smooth' })
}

// 删除会话
async function handleDeleteSession(sessionId: string, onSuccess: () => void) {
  try {
    await deleteSession(sessionId)
    sessions.value = sessions.value.filter(s => s.id !== sessionId)
    message.success(t('common.success'))
    onSuccess()
  } catch (e) {
    message.error(t('common.error'))
  }
}

// 重置当前会话状态
function resetCurrentSessionState() {
  stopChatStream()
  currentSessionId.value = ''
  messages.value = []
  sessionStats.value = null
  pendingFile.value = null
}

// 发送消息
async function handleSend() {
  if ((!inputMessage.value.trim() && !pendingFile.value) || isStreaming.value) return
  if (!appStore.isLLMConfigured) return message.warning(t('setup.description'))

  let content = inputMessage.value
  if (pendingFile.value) {
    const f = pendingFile.value
    const fileDesc = f.type === 'image' ? `\n\n[图片附件：${f.name}]` : `\n\n[文件：${f.name}]\n${f.content}`
    content += fileDesc
    pendingFile.value = null
  }

  inputMessage.value = ''
  messages.value.push({
    id: 'msg-' + Date.now(), role: 'user', content, time: new Date().toLocaleTimeString()
  })
  scrollToBottom()

  isThinking.value = true
  isStreaming.value = true
  streamingContent.value = ''
  typewriterQueue = ''
  abortController.value = new AbortController()

  try {
    await sendChatStream(
      currentSessionId.value,
      content,
      {
        onToolCall: (toolName) => {
          toolProgress.value.push({ name: toolName, status: 'calling', startMs: Date.now(), expanded: false })
          scrollToBottom()
        },
        onToolResult: (toolName, result, isError) => {
          const item = [...toolProgress.value].reverse().find(t => t.name === toolName && t.status === 'calling')
          if (item) {
            item.status = isError ? 'error' : 'done'
            item.elapsedMs = Date.now() - item.startMs
            item.summary = result.length > 80 ? result.slice(0, 80) + '…' : result
          }
        },
        onDelta: (delta) => {
          isThinking.value = false
          toolProgress.value = []
          typewriterQueue += delta
          startTypewriter()
        },
        onDone: async () => {
          await flushTypewriter()
          if (streamingContent.value) {
            messages.value.push({
              id: 'msg-' + Date.now(),
              role: 'assistant',
              content: streamingContent.value,
              time: new Date().toLocaleTimeString()
            })
            streamingContent.value = ''
          }
          isStreaming.value = false
          isThinking.value = false
          toolProgress.value = []
          loadSessions()
          loadStats(currentSessionId.value)
        },
        onError: (error) => {
          stopTypewriter()
          streamingContent.value = ''
          isStreaming.value = false
          isThinking.value = false
          toolProgress.value = []
          message.error(error)
        }
      },
      { signal: abortController.value.signal }
    )
  } catch (e: any) {
    if (e.name !== 'AbortError') message.error(t('common.error'))
    isStreaming.value = false
    isThinking.value = false
    stopTypewriter()
  }
}

watch(() => route.params.id, (id) => id ? handleSessionSwitch(id as string) : (sessions.value.length > 0 ? selectSession(sessions.value[0].id) : createNewSession()))

onMounted(async () => {
  const list = await loadSessions()
  const routeId = route.params.id as string
  if (routeId) handleSessionSwitch(routeId)
  else if (list.length > 0) selectSession(list[0].id)
  else createNewSession()
})

onUnmounted(() => stopChatStream())
</script>

<style scoped lang="scss">
.chat-page { 
  height: 100vh;
  overflow: hidden;
}

.chat-layout { 
  display: flex; 
  height: 100%; 
  background-color: #f9fafb;
}

// 会话侧边栏
.session-sidebar { 
  width: 280px; 
  display: flex; 
  flex-direction: column; 
  background: #ffffff; 
  border-right: 1px solid #e5e7eb;
  padding: 20px 12px;
}

.new-chat-btn { 
  margin-bottom: 20px; 
  height: 42px;
  border-radius: 10px;
  font-weight: 600;
}

.session-list { flex: 1; }

.session-list-item { 
  border-radius: 10px;
  margin-bottom: 4px;
  transition: all 0.2s;
  
  .session-actions {
    opacity: 0;
    transition: opacity 0.2s;
    display: flex;
    gap: 4px;
  }
  
  &:hover {
    background-color: #f3f4f6;
    .session-actions { opacity: 1; }
  }

  &.active { 
    background-color: #f0fdf4; 
    .session-name { color: #16a34a; font-weight: 600; }
  }
}

.session-item { 
  display: flex; 
  align-items: center; 
  justify-content: space-between; 
  width: 100%; 
  padding: 4px 8px;
}

.session-info { 
  display: flex; 
  align-items: center; 
  gap: 12px; 
  overflow: hidden; 
  flex: 1;
  color: #4b5563;
}

.session-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
}

.action-btn {
  color: #666;
  &:hover { color: #18a058; }
}

.active { 
  background-color: #f0f7f1; 
  border-radius: 8px; 
}

.chat-main { 
  flex: 1; 
  height: 100%; 
  display: flex; 
  flex-direction: column;
  background: #ffffff;
}

.chat-header { 
  height: 64px;
  display: flex; 
  align-items: center; 
  justify-content: space-between; 
  padding: 0 32px;
  border-bottom: 1px solid #f3f4f6;
  flex-shrink: 0;

  .session-title {
    margin: 0;
    font-size: 18px;
    font-weight: 700;
    color: #1f2937;
  }
}

.stats-badge { 
  display: flex; 
  align-items: center; 
  gap: 6px; 
  padding: 6px 12px; 
  background: #f3f4f6; 
  border-radius: 20px; 
  font-size: 12px; 
  color: #6b7280; 
  font-weight: 500;
}

.messages-area { 
  flex: 1; 
  overflow-y: auto; 
  padding: 32px 0;
  display: flex;
  flex-direction: column;
}

.message-row { 
  display: flex; 
  gap: 16px; 
  margin-bottom: 32px; 
  max-width: 800px;
  width: 100%;
  margin-left: auto;
  margin-right: auto;
  padding: 0 24px;

  &.user { 
    flex-direction: row-reverse; 
  } 
}

.message-content-wrapper { 
  display: flex; 
  flex-direction: column; 
  max-width: calc(100% - 52px);
}

.message-bubble { 
  padding: 14px 20px; 
  border-radius: 20px; 
  font-size: 15px; 
  line-height: 1.6;
  
  &.user { 
    background-color: #18a058;
    color: #fff; 
    border-bottom-right-radius: 4px;
  } 
  
  &.assistant { 
    background-color: #f3f4f6; 
    color: #1f2937; 
    border-bottom-left-radius: 4px;
  } 
  
  &.thinking { 
    display: flex; 
    align-items: center; 
    gap: 12px; 
    background-color: #f9fafb;
    border: 1px solid #e5e7eb;
  } 
}

// 现代化输入框
.input-container { 
  padding: 0 24px 24px;
  flex-shrink: 0;
  max-width: 848px;
  width: 100%;
  margin: 0 auto;
}

.input-card {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 24px;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.05);
  transition: all 0.2s;
  overflow: hidden;

  &:focus-within {
    border-color: #18a058;
    box-shadow: 0 10px 15px -3px rgba(24, 160, 88, 0.1);
  }
}

.file-preview-area {
  padding: 12px 16px 4px;
  display: flex;
  gap: 8px;
}

.input-inner {
  padding: 8px 12px 12px;
  display: flex;
  flex-direction: column;
}

.chat-textarea {
  :deep(.n-input__border), :deep(.n-input__state-border) { border: none !important; }
  :deep(.n-input-wrapper) { padding: 4px 8px; }
  :deep(textarea) {
    font-size: 16px;
    line-height: 1.6;
    color: #374151;
  }
}

.input-actions-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 4px;
  padding: 0 4px;
}

.send-btn {
  width: 36px;
  height: 36px;
  transition: transform 0.2s;
  
  &:not(:disabled):hover {
    transform: scale(1.05);
  }
}

.input-footer-tip {
  text-align: center;
  font-size: 12px;
  color: #9ca3af;
  margin-top: 12px;
}

// 工具卡片
.tool-progress-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 8px;
}

.tool-card {
  border-radius: 12px;
  border: 1px solid #e5e7eb;
  background: #fff;
  
  &.calling { border-color: #93c5fd; background: #eff6ff; }
  &.done { border-color: #86efac; background: #f0fdf4; }
}

.tool-card-header {
  padding: 8px 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
}

.tool-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #94a3b8;
  &.calling { background: #3b82f6; animation: pulse 2s infinite; }
  &.done { background: #22c55e; }
}

@keyframes pulse {
  0% { opacity: 1; }
  50% { opacity: 0.5; }
  100% { opacity: 1; }
}

.tool-card-body {
  padding: 0 12px 10px;
  font-size: 12px;
  color: #6b7280;
  word-break: break-all;
}

.message-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  padding: 0 4px;
  opacity: 0;
  transition: opacity 0.2s;
  .message-row:hover & { opacity: 1; }
}

.message-time { font-size: 11px; color: #9ca3af; }

// Markdown 内容美化 (由 github-markdown-css 提供基础样式)
:deep(.markdown-body) {
  font-size: 15px;
  line-height: 1.6;
  background-color: transparent !important; // 保持气泡背景
  color: inherit;

  // 微调：移除自带的 padding 和背景，适配气泡
  &::before, &::after { display: none; }
  padding: 0 !important;
  
  // 代码块适配气泡圆角
  pre {
    border-radius: 12px;
    background-color: #1f2937 !important;
    code { color: #e5e7eb; }
  }
}

.avatar-img {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
}
</style>
