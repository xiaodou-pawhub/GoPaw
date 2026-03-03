<template>
  <div class="chat-page">
    <div class="chat-layout">
      <!-- 左侧会话列表 -->
      <div class="session-sidebar">
        <n-button type="primary" dashed block @click="createNewSession" class="new-chat-btn">
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
        <n-card :bordered="false" class="chat-card" content-style="padding: 0; display: flex; flex-direction: column; height: 100%;">
          <template #header>
            <div class="chat-header">
              <div class="header-left">
                <n-text strong size="large">{{ currentSessionName || t('chat.title') }}</n-text>
              </div>
              
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
                <n-avatar round :size="36" :style="{ backgroundColor: msg.role === 'user' ? '#18a058' : '#18a058' }">
                  <n-icon :component="msg.role === 'user' ? PersonOutline : PawOutline" />
                </n-avatar>
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
            
            <div v-if="isThinking" class="message-row assistant">
              <div class="avatar">
                <n-avatar round :size="36" style="background-color: #18a058">
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
            <div class="input-box-wrapper">
              <div class="input-actions-top">
                <div v-if="pendingFile" class="pending-file-tag">
                  <n-tag type="info" closable @close="pendingFile = null" size="small">
                    {{ pendingFile.name }}
                  </n-tag>
                </div>
                <n-upload
                  v-if="!isStreaming"
                  action="/api/agent/upload"
                  :show-file-list="false"
                  @finish="handleUploadFinish"
                >
                  <n-button quaternary circle size="small">
                    <template #icon><n-icon :component="AttachOutline" /></template>
                  </n-button>
                </n-upload>
                <n-button v-else quaternary circle size="small" type="error" @click="stopChatStream">
                  <template #icon><n-icon :component="StopCircleOutline" /></template>
                </n-button>
              </div>

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
                  type="primary" circle
                  :disabled="(!inputMessage.trim() && !pendingFile) || !appStore.isLLMConfigured || isStreaming"
                  :loading="isStreaming"
                  @click="handleSend"
                  class="send-btn"
                >
                  <template #icon><n-icon :component="SendOutline" /></template>
                </n-button>
              </div>
            </div>
          </div>
        </n-card>
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
  NCard, NButton, NIcon, NInput, NList, NListItem, NScrollbar,
  NAvatar, NText, NEmpty, NSpin, NTooltip, NUpload, NTag, NModal, useMessage
} from 'naive-ui'
import {
  AddOutline, ChatbubbleOutline, PersonOutline, PawOutline,
  SendOutline, TrashOutline, BarChartOutline, AttachOutline, StopCircleOutline,
  CreateOutline, CopyOutline
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

// 流式对话状态
const isThinking = ref(false)
const isStreaming = ref(false)
const abortController = ref<AbortController | null>(null)

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
    // 更新本地状态
    const idx = sessions.value.findIndex(s => s.id === renamingSession.value!.id)
    if (idx !== -1) {
      sessions.value[idx].name = renameValue.value.trim()
    }
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

// Markdown 渲染器
const md: any = markdownIt({
  html: false, linkify: true, typographer: true,
  highlight: (str: string, lang: string): string => {
    if (lang && highlightjs.getLanguage(lang)) {
      try { return `<pre class="hljs"><code>${highlightjs.highlight(str, { language: lang, ignoreIllegals: true }).value}</code></pre>` } catch (__) {}
    }
    return `<pre class="hljs"><code>${md.utils.escapeHtml(str)}</code></pre>`
  }
})

// 加载会话列表
async function loadSessions(): Promise<SessionInfo[]> {
  try {
    sessions.value = await getSessions()
    return sessions.value
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

// 加载统计数据
async function loadStats(id: string) {
  try {
    sessionStats.value = await getSessionStats(id)
  } catch (error) {
    // 静默处理
  }
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

// 处理会话切换
async function handleSessionSwitch(id: string) {
  if (currentSessionId.value === id && messages.value.length > 0) return
  if (isCreatingNew.value && currentSessionId.value === id) {
    isCreatingNew.value = false
    return
  }
  if (!sessions.value.some(s => s.id === id) && !isCreatingNew.value) {
    fallbackToValidSession()
    return
  }
  stopChatStream()
  currentSessionId.value = id
  loadStats(id)
  try {
    messages.value = await getSessionMessages(id)
    scrollToBottom()
  } catch (error) {
    message.error('加载历史记录失败')
  }
}

// 回退到有效会话
function fallbackToValidSession() {
  if (sessions.value.length > 0) {
    selectSession(sessions.value[0].id)
  } else {
    createNewSession()
  }
}

// 创建新会话
function createNewSession() {
  stopChatStream()
  pendingFile.value = null
  const newId = crypto.randomUUID()
  isCreatingNew.value = true
  currentSessionId.value = newId
  messages.value = [{
    id: 'welcome-' + Date.now(), role: 'assistant',
    content: t('chat.welcome'), time: new Date().toLocaleTimeString()
  }]
  sessionStats.value = null
  router.push({ name: 'Chat', params: { id: newId } })
}

// 停止流式对话
function stopChatStream() {
  if (abortController.value) {
    abortController.value.abort()
    abortController.value = null
  }
  isStreaming.value = false
  isThinking.value = false
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

  const assistantMsg: ChatMessage = {
    id: 'msg-' + (Date.now() + 1), role: 'assistant', content: '', time: new Date().toLocaleTimeString()
  }

  isThinking.value = true
  isStreaming.value = true
  abortController.value = new AbortController()

  try {
    await sendChatStream(
      currentSessionId.value,
      content,
      {
        onDelta: (delta) => {
          isThinking.value = false
          if (assistantMsg.content === '') messages.value.push(assistantMsg)
          assistantMsg.content += delta
          scrollToBottom()
        },
        onDone: () => {
          isStreaming.value = false
          isThinking.value = false
          loadSessions()
          loadStats(currentSessionId.value)
        },
        onError: (error) => {
          isStreaming.value = false
          isThinking.value = false
          message.error(error)
        }
      },
      { signal: abortController.value.signal }
    )
  } catch (e: any) {
    if (e.name !== 'AbortError') {
      message.error(t('common.error'))
    }
    isStreaming.value = false
    isThinking.value = false
  }
}

const formatTokens = (n: number) => {
  if (!n) return '0'
  return n >= 1000000 ? `${(n / 1000000).toFixed(1)}M` : n >= 1000 ? `${(n / 1000).toFixed(1)}k` : n.toString()
}

const renderMarkdown = (c: string) => md.render(c)

const scrollToBottom = async () => {
  await nextTick()
  messagesRef.value?.scrollTo({ top: messagesRef.value.scrollHeight, behavior: 'smooth' })
}

const handleKeydown = (e: KeyboardEvent) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() } }

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
.chat-page { height: calc(100vh - 112px); }
.chat-layout { display: flex; height: 100%; gap: 20px; }

.session-sidebar { 
  width: 260px; 
  display: flex; 
  flex-direction: column; 
  background: #fff; 
  border-radius: 12px; 
  padding: 16px; 
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.05); 
}

.new-chat-btn { margin-bottom: 16px; }
.session-list { flex: 1; }

.session-list-item { 
  position: relative; 
  border-radius: 8px;
  
  .session-actions {
    opacity: 0;
    transition: opacity 0.2s;
    display: flex;
    gap: 4px;
  }
  
  &:hover .session-actions { 
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
  flex: 1;
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

.chat-main { flex: 1; height: 100%; }
.chat-card { height: 100%; border-radius: 12px; box-shadow: 0 2px 12px rgba(0, 0, 0, 0.05); }
.chat-header { display: flex; align-items: center; justify-content: space-between; width: 100%; }
.stats-badge { display: flex; align-items: center; gap: 6px; padding: 4px 10px; background: #f3f4f6; border-radius: 6px; font-size: 13px; color: #666; cursor: default; }
.stats-detail { line-height: 1.6; }

.messages-area { 
  flex: 1; 
  overflow-y: auto; 
  padding: 24px; 
  background-color: #fafafa; 
}

.empty-state { height: 100%; display: flex; align-items: center; justify-content: center; }

.message-row { 
  display: flex; 
  gap: 16px; 
  margin-bottom: 20px; 
  max-width: 85%; 
  &.user { margin-left: auto; flex-direction: row-reverse; } 
}

.message-content-wrapper { 
  display: flex; 
  flex-direction: column; 
  max-width: 100%;
}

.message-bubble { 
  padding: 12px 16px; 
  border-radius: 16px; 
  font-size: 15px; 
  line-height: 1.6;
  
  &.user { 
    background: linear-gradient(135deg, #18a058 0%, #10b981 100%);
    color: #fff; 
    border-bottom-right-radius: 4px;
  } 
  
  &.assistant { 
    background-color: #fff; 
    color: #333; 
    border-bottom-left-radius: 4px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08); 
  } 
  
  &.thinking { 
    display: flex; 
    align-items: center; 
    gap: 8px; 
    background-color: #f5f5f5;
    color: #666;
  } 
}

.thinking-text { font-size: 14px; color: #666; }

.message-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  opacity: 0;
  transition: opacity 0.2s;
  
  .message-row:hover & {
    opacity: 1;
  }
}

.message-time { 
  font-size: 11px; 
  color: #999; 
}

.copy-btn {
  padding: 2px;
  color: #999;
  &:hover { color: #18a058; }
}

.user .message-meta { justify-content: flex-end; }

.input-container { 
  padding: 16px 20px; 
  background: #fff; 
  border-top: 1px solid #f0f0f0; 
}

.input-box-wrapper { display: flex; flex-direction: column; gap: 8px; }
.input-actions-top { display: flex; gap: 8px; align-items: center; }
.pending-file-tag { max-width: 200px; }

.input-box { 
  display: flex; 
  align-items: flex-end; 
  gap: 12px; 
  background: #f7f7f8; 
  padding: 10px 14px; 
  border-radius: 24px; 
  border: 1px solid transparent;
  transition: all 0.2s;
  
  &:focus-within { 
    border-color: #18a058; 
    box-shadow: 0 0 0 2px rgba(24, 160, 88, 0.1);
    background: #fff;
  } 
}

.chat-input { 
  flex: 1;
  :deep(.n-input__border), :deep(.n-input__state-border) { border: none !important; } 
  :deep(.n-input-wrapper) { padding: 0; } 
  :deep(.n-input__textarea-el) {
    font-size: 15px;
    line-height: 1.5;
  }
}

.send-btn { 
  flex-shrink: 0;
  width: 36px;
  height: 36px;
}

// Markdown 样式
:deep(.markdown-body) {
  line-height: 1.7;
  
  p { margin: 0 0 12px 0; &:last-child { margin-bottom: 0; } }
  
  code {
    background: #f5f5f5;
    padding: 2px 6px;
    border-radius: 4px;
    font-family: 'Fira Code', monospace;
    font-size: 13px;
  }
  
  pre {
    background: #1e1e1e;
    border-radius: 8px;
    padding: 12px;
    overflow-x: auto;
    margin: 12px 0;
    
    code {
      background: transparent;
      padding: 0;
      color: #d4d4d4;
    }
  }
  
  // 表格样式
  table {
    border-collapse: collapse;
    width: 100%;
    margin: 12px 0;
    font-size: 14px;
    
    th, td {
      border: 1px solid #e5e7eb;
      padding: 8px 12px;
      text-align: left;
    }
    
    th {
      background: #f9fafb;
      font-weight: 600;
    }
    
    tr:nth-child(even) {
      background: #f9fafb;
    }
  }
  
  ul, ol {
    padding-left: 20px;
    margin: 8px 0;
  }
  
  blockquote {
    border-left: 3px solid #18a058;
    padding-left: 12px;
    margin: 12px 0;
    color: #666;
  }
}

:deep(.hljs) { 
  padding: 12px; 
  border-radius: 8px; 
  font-family: 'Fira Code', 'Courier New', Courier, monospace; 
  font-size: 13px;
}
</style>