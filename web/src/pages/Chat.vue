<template>
  <div class="chat-root">
    <!-- 左侧会话面板 -->
    <aside class="session-panel">
      <div class="panel-header">
        <button class="new-chat-btn" @click="createNewSession">
          <PlusIcon :size="16" />
          新对话
        </button>
        <button class="icon-btn" title="搜索会话" @click="searchOpen = !searchOpen">
          <SearchIcon :size="16" />
        </button>
      </div>

      <!-- 搜索框 -->
      <div v-if="searchOpen" class="search-box">
        <SearchIcon :size="12" class="search-icon" />
        <input
          v-model="searchQuery"
          class="search-input"
          placeholder="搜索会话..."
          autofocus
        />
      </div>

      <!-- 会话列表 -->
      <div class="session-list">
        <template v-if="sessionsLoading">
          <Skeleton v-for="i in 5" :key="i" width="100%" height="40px" shape="round" style="margin-bottom: 8px;" />
        </template>
        <template v-else-if="filteredSessions.length === 0">
          <EmptyState
            :icon="MessageSquareIcon"
            title="暂无会话"
            description="点击新对话按钮开始聊天"
            :icon-size="32"
          >
            <button class="btn-primary btn-sm" @click="createNewSession">
              <PlusIcon :size="14" />
              新对话
            </button>
          </EmptyState>
        </template>
        <template v-else>
          <div
            v-for="session in filteredSessions"
            :key="session.id"
            class="session-item"
            :class="{ active: currentSessionId === session.id }"
            @click="selectSession(session.id)"
          >
            <MessageSquareIcon :size="16" class="session-icon" />
            <span class="session-name">{{ getSessionDisplayName(session) }}</span>
            <div class="session-actions">
              <button
                class="icon-btn-xs"
                title="重命名"
                @click.stop="startRenameSession(session)"
              >
                <PencilIcon :size="13" />
              </button>
              <button
                class="icon-btn-xs danger"
                title="删除"
                @click.stop="handleDeleteSession(session.id)"
              >
                <TrashIcon :size="13" />
              </button>
            </div>
          </div>
        </template>
      </div>
    </aside>

    <!-- 主聊天区 -->
    <div class="chat-main">
      <!-- Chat 顶栏 -->
      <div class="chat-header">
        <h3 class="chat-title">{{ currentSessionName || '聊天' }}</h3>
        <div class="header-actions">
          <AgentSelector
            v-model="currentAgentId"
            @change="onAgentChange"
          />
          <div v-if="sessionStats" class="stats-badge" :title="`消息: ${sessionStats.message_count} | 用户: ${formatTokens(sessionStats.user_tokens)} | 助手: ${formatTokens(sessionStats.assist_tokens)}`">
            <BarChartIcon :size="12" />
            <span>{{ formatTokens(sessionStats.total_tokens) }} tokens</span>
          </div>
        </div>
      </div>

      <!-- 消息区 -->
      <div ref="messagesRef" class="messages-area">
        <div v-if="messagesLoading" class="messages-loading">
          <Skeleton v-for="i in 3" :key="i" width="100%" height="60px" shape="round" style="margin-bottom: 12px;" />
        </div>
        <div v-else-if="messages.length === 0" class="empty-chat">
          <!-- 欢迎界面 -->
          <div class="welcome-container">
            <div class="welcome-header">
              <img src="/assets/logo.png" alt="GoPaw" class="welcome-logo" />
              <h2 class="welcome-title">有什么可以帮你的？</h2>
            </div>

            <!-- 快捷建议 -->
            <div class="suggestions-grid">
              <button
                v-for="suggestion in suggestions"
                :key="suggestion.text"
                class="suggestion-card"
                @click="handleSuggestionClick(suggestion.text)"
              >
                <component :is="suggestion.icon" :size="18" class="suggestion-icon" />
                <span class="suggestion-text">{{ suggestion.text }}</span>
                <ArrowRightIcon :size="14" class="suggestion-arrow" />
              </button>
            </div>

            <!-- 功能提示 -->
            <div class="feature-hints">
              <div class="hint-item">
                <SparklesIcon :size="14" />
                <span>支持多模型切换</span>
              </div>
              <div class="hint-item">
                <WrenchIcon :size="14" />
                <span>内置工具调用</span>
              </div>
              <div class="hint-item">
                <BrainIcon :size="14" />
                <span>智能记忆系统</span>
              </div>
            </div>
          </div>
        </div>

        <div
          v-for="msg in messages"
          :key="msg.id"
          class="message-row"
          :class="msg.role"
        >
          <!-- 用户消息 -->
          <template v-if="msg.role === 'user'">
            <div class="message-avatar user-avatar">
              <span class="avatar-initials">{{ getUserInitials() }}</span>
            </div>
            <div class="message-content user-content">
              <div class="message-text">{{ msg.content }}</div>
              <div class="message-meta">
                <span class="message-time">{{ msg.time }}</span>
                <button class="copy-btn" @click="copyMessage(msg.content)" title="复制">
                  <CopyIcon :size="11" />
                </button>
              </div>
            </div>
          </template>

          <!-- AI 消息 -->
          <template v-else>
            <div class="message-avatar ai-avatar">
              <img src="/assets/logo.png" alt="GoPaw" width="20" height="20" />
            </div>
            <div class="message-content ai-content">
              <div
                class="markdown-content"
                v-html="renderMarkdown(msg.content)"
              />
              <div class="message-meta">
                <span class="message-time">{{ msg.time }}</span>
                <button class="copy-btn" @click="copyMessage(msg.content)" title="复制">
                  <CopyIcon :size="11" />
                </button>
              </div>
            </div>
          </template>
        </div>

        <!-- 思考中 / 工具调用 -->
        <div v-if="isThinking" class="message-row assistant">
          <div class="assistant-avatar">
            <img src="/assets/logo.png" alt="GoPaw" width="24" height="24" style="border-radius: 6px;" />
          </div>
          <div class="assistant-content">
            <div v-if="toolProgress.length > 0" class="tool-list">
              <div
                v-for="(tool, i) in toolProgress"
                :key="i"
                class="tool-card"
                :class="tool.status"
                @click="tool.expanded = !tool.expanded"
              >
                <div class="tool-header">
                  <span class="tool-dot" :class="tool.status" />
                  <span class="tool-name">{{ tool.name }}</span>
                  <span v-if="tool.elapsedMs !== undefined" class="tool-elapsed">{{ tool.elapsedMs }}ms</span>
                  <LoaderIcon v-else :size="12" class="tool-spinner" />
                  <ChevronDownIcon v-if="tool.summary" :size="12" class="tool-chevron" :class="{ rotated: tool.expanded }" />
                </div>
                <div v-if="tool.expanded && tool.summary" class="tool-body">{{ tool.summary }}</div>
              </div>
            </div>
            <div v-else class="thinking-indicator">
              <span class="thinking-dot" /><span class="thinking-dot" /><span class="thinking-dot" />
              <span class="thinking-text">{{ t('chat.thinking') }}</span>
            </div>
          </div>
        </div>

        <!-- 流式回复 -->
        <div v-if="streamingContent" class="message-row assistant">
          <div class="assistant-avatar">
            <img src="/assets/logo.png" alt="GoPaw" width="24" height="24" style="border-radius: 6px;" />
          </div>
          <div class="assistant-content">
            <div class="markdown-content" v-html="renderMarkdown(streamingContent)" />
          </div>
        </div>
      </div>

      <!-- 输入区 -->
      <div class="input-area">
        <!-- 底部渐变遮罩 -->
        <div class="input-mask" />
        
        <div class="input-card" :class="{ focused: inputFocused }">
          <!-- 文件预览 -->
          <div v-if="pendingFile" class="file-preview">
            <FileTextIcon :size="12" />
            <span>{{ pendingFile.name }}</span>
            <button @click="pendingFile = null"><XIcon :size="11" /></button>
          </div>

          <textarea
            ref="textareaRef"
            v-model="inputMessage"
            class="chat-textarea"
            :placeholder="t('chat.placeholder')"
            :disabled="!appStore.isLLMConfigured || isStreaming"
            rows="1"
            @focus="inputFocused = true"
            @blur="inputFocused = false"
            @keydown="handleKeydown"
            @input="autoResize"
          />

          <div class="input-actions">
            <div class="left-actions">
              <!-- 上传按钮 -->
              <label v-if="!isStreaming" class="icon-btn" title="上传文件">
                <PaperclipIcon :size="14" />
                <input
                  type="file"
                  class="hidden"
                  @change="handleFileSelect"
                />
              </label>
              <!-- 停止按钮 -->
              <button v-else class="icon-btn danger" title="停止" @click="stopChatStream">
                <StopCircleIcon :size="14" />
              </button>
            </div>

            <div class="right-actions">
              <span class="input-hint">⏎ 发送 · ⇧⏎ 换行</span>
              <button
                class="send-btn"
                :disabled="(!inputMessage.trim() && !pendingFile) || !appStore.isLLMConfigured || isStreaming"
                @click="handleSend"
              >
                <ArrowUpIcon :size="14" />
              </button>
            </div>
          </div>
        </div>
        <p class="input-footer">GoPaw AI 助手可能会产生错误，请核实重要信息。</p>
      </div>
    </div>

    <!-- 重命名对话框 -->
    <div v-if="showRenameModal" class="modal-overlay" @click.self="showRenameModal = false">
      <div class="modal-card">
        <h4 class="modal-title">重命名会话</h4>
        <input
          v-model="renameValue"
          class="modal-input"
          placeholder="输入会话名称"
          @keydown.enter="confirmRename"
          autofocus
        />
        <div class="modal-actions">
          <button class="btn-secondary" @click="showRenameModal = false">取消</button>
          <button class="btn-primary" @click="confirmRename">确认</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, onUnmounted, watch, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  PlusIcon, SearchIcon, MessageSquareIcon, PencilIcon, TrashIcon,
  BarChartIcon, CopyIcon, PaperclipIcon, StopCircleIcon, ArrowUpIcon,
  LoaderIcon, ChevronDownIcon, FileTextIcon, XIcon,
  ArrowRightIcon, SparklesIcon, WrenchIcon, BrainIcon,
  CodeIcon, FileCodeIcon, LightbulbIcon, HelpCircleIcon
} from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAgentStore } from '@/stores/agent'
import { toast } from 'vue-sonner'
import { getSessionMessages, getSessionStats, getSessions, deleteSession, updateSessionName, sendChatStream } from '@/api/agent'
import type { ChatMessage, SessionStats, SessionInfo } from '@/types'
import type { Agent } from '@/api/agents'
import { default as markdownIt } from 'markdown-it'
import highlightjs from 'highlight.js'
import Skeleton from '@/components/Skeleton.vue'
import EmptyState from '@/components/EmptyState.vue'
import AgentSelector from '@/components/AgentSelector.vue'
import { saveCurrentSession, getCurrentSession } from '@/utils/storage'

const { t } = useI18n()
const appStore = useAppStore()
const agentStore = useAgentStore()
const route = useRoute()
const router = useRouter()

// Agent state
const currentAgentId = computed({
  get: () => agentStore.currentAgentId,
  set: (val: string | null) => agentStore.setCurrentAgent(val || '')
})

const sessions = ref<SessionInfo[]>([])
const currentSessionId = ref('')
const messages = ref<ChatMessage[]>([])
const sessionStats = ref<SessionStats | null>(null)
const inputMessage = ref('')
const messagesRef = ref<HTMLElement | null>(null)
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const isCreatingNew = ref(false)
const pendingFile = ref<{ name: string; content: string; type: string } | null>(null)
const inputFocused = ref(false)
const searchOpen = ref(false)
const searchQuery = ref('')
const sessionsLoading = ref(true)
const messagesLoading = ref(true)

const isThinking = ref(false)
const isStreaming = ref(false)
const abortController = ref<AbortController | null>(null)

interface ToolCallProgress {
  name: string
  status: 'calling' | 'done' | 'error'
  startMs: number
  elapsedMs?: number
  summary?: string
  expanded: boolean
}
const toolProgress = ref<ToolCallProgress[]>([])
const streamingContent = ref('')

// 欢迎界面建议
const suggestions = [
  { text: '帮我写一个 Python 脚本', icon: CodeIcon },
  { text: '解释这段代码的作用', icon: FileCodeIcon },
  { text: '给我一些项目建议', icon: LightbulbIcon },
  { text: '帮我调试一个问题', icon: HelpCircleIcon },
]

function handleSuggestionClick(text: string) {
  inputMessage.value = text
  handleSend()
}

// 打字机
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

// 重命名
const showRenameModal = ref(false)
const renameValue = ref('')
const renamingSession = ref<SessionInfo | null>(null)

const currentSessionName = computed(() => {
  const session = sessions.value.find(s => s.id === currentSessionId.value)
  return session?.name || ''
})

const filteredSessions = computed(() => {
  if (!searchQuery.value) return sessions.value
  const q = searchQuery.value.toLowerCase()
  return sessions.value.filter(s =>
    getSessionDisplayName(s).toLowerCase().includes(q)
  )
})

function getSessionDisplayName(session: SessionInfo): string {
  if (session.name) return session.name
  return session.id.substring(0, 8)
}

function startRenameSession(session: SessionInfo) {
  renamingSession.value = session
  renameValue.value = session.name || ''
  showRenameModal.value = true
}

async function confirmRename() {
  if (!renamingSession.value || !renameValue.value.trim()) return
  try {
    await updateSessionName(renamingSession.value.id, renameValue.value.trim())
    const idx = sessions.value.findIndex(s => s.id === renamingSession.value!.id)
    if (idx !== -1) sessions.value[idx].name = renameValue.value.trim()
    showRenameModal.value = false
    toast.success('已重命名')
  } catch {
    toast.error('操作失败')
  }
}

async function copyMessage(content: string) {
  try {
    await navigator.clipboard.writeText(content)
    toast.success(t('chat.copied'))
  } catch {
    toast.error(t('chat.copyFailed'))
  }
}

const formatTokens = (n: number) => {
  if (!n) return '0'
  return n >= 1000000 ? `${(n / 1000000).toFixed(1)}M` : n >= 1000 ? `${(n / 1000).toFixed(1)}k` : n.toString()
}

// 文件选择
function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = (e) => {
    pendingFile.value = {
      name: file.name,
      content: e.target?.result as string || '',
      type: file.type || 'text/plain'
    }
  }
  reader.readAsText(file)
  input.value = ''
}

// Markdown
const md: any = markdownIt({
  html: false, linkify: true, typographer: true,
  highlight: (str: string, lang: string): string => {
    if (lang && highlightjs.getLanguage(lang)) {
      try {
        return `<pre class="hljs"><code>${highlightjs.highlight(str, { language: lang, ignoreIllegals: true }).value}</code></pre>`
      } catch (__) {}
    }
    return `<pre class="hljs"><code>${md.utils.escapeHtml(str)}</code></pre>`
  }
})
const renderMarkdown = (c: string) => md.render(c)

// textarea 自动高度
function autoResize() {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 200) + 'px'
}

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

async function loadSessions(): Promise<SessionInfo[]> {
  sessionsLoading.value = true
  try {
    const list = await getSessions()
    sessions.value = list
    return list
  } catch {
    toast.error('加载会话列表失败')
    return []
  } finally {
    sessionsLoading.value = false
  }
}

function selectSession(id: string) {
  if (currentSessionId.value === id) return
  // 保存当前会话 ID 到存储
  saveCurrentSession(id)
  router.push({ name: 'Chat', params: { id } })
}

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
  messagesLoading.value = true
  currentSessionId.value = id
  loadStats(id)
  try {
    const history = await getSessionMessages(id)
    messages.value = history
    scrollToBottom()
  } catch {
    toast.error('加载历史记录失败')
  } finally {
    messagesLoading.value = false
  }
}

function getUserInitials(): string {
  // 简单实现：返回 "U" 作为用户首字母
  // 后续可以从用户配置中读取
  return "U"
}

function createNewSession() {
  stopChatStream()
  pendingFile.value = null
  const newId = generateUUID()
  isCreatingNew.value = true
  currentSessionId.value = newId
  messages.value = [] // 清空消息，让欢迎界面显示
  messagesLoading.value = false // 关闭加载状态
  sessionStats.value = null
  router.push({ name: 'Chat', params: { id: newId } })
}

function onAgentChange(agent: Agent) {
  // Show toast notification
  toast.success(`已切换到 Agent: ${agent.name}`)
  // Note: Agent ID is passed in the next chat request via sendChatStream
}

function generateUUID(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) return crypto.randomUUID()
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0
    return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16)
  })
}

async function loadStats(id: string) {
  try {
    sessionStats.value = await getSessionStats(id)
  } catch {}
}

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

const scrollToBottom = async () => {
  await nextTick()
  messagesRef.value?.scrollTo({ top: messagesRef.value.scrollHeight, behavior: 'smooth' })
}

async function handleDeleteSession(sessionId: string) {
  if (!confirm('删除后无法恢复，是否继续？')) return
  try {
    await deleteSession(sessionId)
    sessions.value = sessions.value.filter(s => s.id !== sessionId)
    toast.success('已删除')
    if (currentSessionId.value === sessionId) {
      stopChatStream()
      currentSessionId.value = ''
      messages.value = []
      sessionStats.value = null
      pendingFile.value = null
      if (sessions.value.length > 0) selectSession(sessions.value[0].id)
      else createNewSession()
    }
  } catch {
    toast.error('删除失败')
  }
}

async function handleSend() {
  if ((!inputMessage.value.trim() && !pendingFile.value) || isStreaming.value) return
  if (!appStore.isLLMConfigured) {
    toast.warning('请先配置 LLM 提供商')
    return
  }

  let content = inputMessage.value
  if (pendingFile.value) {
    const f = pendingFile.value
    const fileDesc = f.type.startsWith('image/') ? `\n\n[图片附件：${f.name}]` : `\n\n[文件：${f.name}]\n${f.content}`
    content += fileDesc
    pendingFile.value = null
  }

  inputMessage.value = ''
  if (textareaRef.value) {
    textareaRef.value.style.height = 'auto'
  }
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
          toast.error(error)
        }
      },
      { signal: abortController.value.signal },
      currentAgentId.value || undefined
    )
  } catch (e: any) {
    if (e.name !== 'AbortError') toast.error(t('common.error'))
    isStreaming.value = false
    isThinking.value = false
    stopTypewriter()
  }
}

watch(() => route.params.id, (id) => {
  if (id) handleSessionSwitch(id as string)
  else if (sessions.value.length > 0) selectSession(sessions.value[0].id)
  else createNewSession()
})

function handleChatKey(e: KeyboardEvent) {
  const target = e.target as HTMLElement
  const inInput = target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable
  if ((e.metaKey || e.ctrlKey) && e.key === 'n' && !inInput) {
    e.preventDefault()
    createNewSession()
  }
}

onMounted(async () => {
  // 从存储恢复当前会话
  const storedSessionId = getCurrentSession()
  
  const list = await loadSessions()
  const routeId = route.params.id as string
  
  if (routeId) {
    handleSessionSwitch(routeId)
  } else if (storedSessionId && list.some(s => s.id === storedSessionId)) {
    // 恢复存储的会话
    selectSession(storedSessionId)
  } else if (list.length > 0) {
    selectSession(list[0].id)
  } else {
    createNewSession()
  }
  
  window.addEventListener('keydown', handleChatKey)
})

onUnmounted(() => {
  stopChatStream()
  window.removeEventListener('keydown', handleChatKey)
})
</script>

<style scoped>
.chat-root {
  flex: 1;
  display: flex;
  overflow: hidden;
  height: 100%;
}

/* ===== 会话面板 ===== */
.session-panel {
  width: 280px;
  background: var(--bg-panel);
  border-right: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 12px 10px 8px;
  flex-shrink: 0;
}

.new-chat-btn {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: var(--accent-dim);
  border: 1px solid rgba(124, 106, 247, 0.2);
  border-radius: 6px;
  color: var(--accent);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}

.new-chat-btn:hover {
  background: rgba(124, 106, 247, 0.15);
}

.icon-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
  flex-shrink: 0;
}

.icon-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-secondary);
}

.icon-btn.danger:hover {
  background: rgba(239, 68, 68, 0.1);
  color: var(--red);
}

.search-box {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 10px 6px;
  padding: 6px 10px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 6px;
}

.search-icon {
  color: var(--text-tertiary);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  background: none;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-size: 12px;
}

.search-input::placeholder {
  color: var(--text-tertiary);
}

.session-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 6px 8px;
}

.empty-sessions {
  text-align: center;
  color: var(--text-tertiary);
  font-size: 12px;
  padding: 24px 0;
}

/* EmptyState 覆盖样式 */
.session-list :deep(.empty-state) {
  padding: 32px 16px;
}

.session-list :deep(.empty-title) {
  font-size: 14px;
}

.session-list :deep(.empty-description) {
  font-size: 12px;
  margin-bottom: 16px;
}

.session-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.12s;
  position: relative;
}

.session-item:hover {
  background: var(--bg-overlay);
}

.session-item.active {
  background: var(--accent-dim);
}

.session-item.active .session-name {
  color: var(--accent);
}

.session-icon {
  color: var(--text-tertiary);
  flex-shrink: 0;
}

.session-name {
  flex: 1;
  font-size: 12px;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-actions {
  display: none;
  gap: 2px;
  flex-shrink: 0;
}

.session-item:hover .session-actions {
  display: flex;
}

.icon-btn-xs {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}

.icon-btn-xs:hover {
  background: var(--bg-elevated);
  color: var(--text-secondary);
}

.icon-btn-xs.danger:hover {
  color: var(--red);
}

/* ===== 主聊天区 ===== */
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--bg-app);
  overflow: hidden;
}

.chat-header {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.chat-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.stats-badge {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  background: var(--bg-elevated);
  border-radius: 20px;
  font-size: 11px;
  color: var(--text-secondary);
  cursor: default;
}

.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 20px 0;
  display: flex;
  flex-direction: column;
}

.messages-loading {
  padding: 20px 24px;
  max-width: 800px;
  width: 100%;
  margin: 0 auto;
}

.empty-chat {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  overflow-y: auto;
}

/* ===== 欢迎界面 ===== */
.welcome-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  max-width: 600px;
  width: 100%;
}

.welcome-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 32px;
}

.welcome-logo {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  margin-bottom: 16px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.welcome-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

/* 快捷建议网格 */
.suggestions-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  width: 100%;
  margin-bottom: 32px;
}

.suggestion-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: var(--bg-panel);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
}

.suggestion-card:hover {
  background: var(--bg-overlay);
  border-color: var(--border);
  transform: translateY(-1px);
}

.suggestion-icon {
  color: var(--accent);
  flex-shrink: 0;
}

.suggestion-text {
  flex: 1;
  font-size: 13px;
  color: var(--text-primary);
  line-height: 1.4;
}

.suggestion-arrow {
  color: var(--text-tertiary);
  opacity: 0;
  transition: opacity 0.15s;
}

.suggestion-card:hover .suggestion-arrow {
  opacity: 1;
}

/* 功能提示 */
.feature-hints {
  display: flex;
  gap: 24px;
  padding: 16px 24px;
  background: var(--bg-panel);
  border-radius: 12px;
  border: 1px solid var(--border-subtle);
}

.hint-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-secondary);
}

.hint-item svg {
  color: var(--accent);
}

/* 响应式 */
@media (max-width: 600px) {
  .suggestions-grid {
    grid-template-columns: 1fr;
  }

  .feature-hints {
    flex-direction: column;
    gap: 12px;
  }
}

/* ===== 消息行 ===== */
.message-row {
  display: flex;
  padding: 12px 24px;
  max-width: 800px;
  width: 100%;
  margin: 0 auto;
  gap: 12px;
}

.message-row.user {
  flex-direction: row;
  align-items: flex-start;
}

.message-row.assistant {
  flex-direction: row;
  align-items: flex-start;
}

/* ===== 头像 ===== */
.message-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
}

.user-avatar {
  background: var(--accent-dim);
  color: var(--accent);
}

.avatar-initials {
  display: flex;
  align-items: center;
  justify-content: center;
}

.ai-avatar {
  background: var(--bg-sidebar);
  border: 1px solid var(--border);
  overflow: hidden;
}

.ai-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 50%;
}

/* ===== 消息内容 ===== */
.message-content {
  flex: 1;
  min-width: 0;
  padding: 12px 16px;
  border-radius: 12px;
}

.user-content {
  background: rgba(218, 119, 86, 0.05);
  border: 1px solid rgba(218, 119, 86, 0.1);
}

.ai-content {
  background: transparent;
  border: none;
  padding: 12px 0;
}

.message-text {
  font-size: 14px;
  line-height: 1.7;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-break: break-word;
}

/* 消息元数据 */
.message-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  opacity: 0;
  transition: opacity 0.15s;
}

.message-row:hover .message-meta {
  opacity: 1;
}

.message-time {
  font-size: 11px;
  color: var(--text-tertiary);
}

/* ===== 思考中 ===== */
.thinking-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 0;
}

.thinking-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--text-tertiary);
  animation: thinking-bounce 1.2s infinite;
}

.thinking-dot:nth-child(2) { animation-delay: 0.2s; }
.thinking-dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes thinking-bounce {
  0%, 60%, 100% { transform: translateY(0); opacity: 0.4; }
  30% { transform: translateY(-4px); opacity: 1; }
}

.thinking-text {
  font-size: 12px;
  color: var(--text-tertiary);
  font-style: italic;
}

/* ===== 工具调用卡片 ===== */
.tool-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 4px 0;
}

.tool-card {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg-elevated);
  cursor: pointer;
  transition: border-color 0.15s;
}

.tool-card.calling { border-color: rgba(124, 106, 247, 0.4); background: rgba(124, 106, 247, 0.05); }
.tool-card.done { border-color: rgba(34, 197, 94, 0.3); background: rgba(34, 197, 94, 0.04); }
.tool-card.error { border-color: rgba(239, 68, 68, 0.3); background: rgba(239, 68, 68, 0.04); }

.tool-header {
  padding: 7px 10px;
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 12px;
  font-weight: 500;
}

.tool-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-tertiary);
  flex-shrink: 0;
}

.tool-dot.calling { background: var(--accent); animation: pulse-dot 1.5s infinite; }
.tool-dot.done { background: var(--green); }
.tool-dot.error { background: var(--red); }

@keyframes pulse-dot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.tool-name { flex: 1; color: var(--text-primary); }
.tool-elapsed { font-size: 11px; color: var(--text-tertiary); font-family: monospace; }

.tool-spinner {
  color: var(--accent);
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.tool-chevron {
  color: var(--text-tertiary);
  transition: transform 0.2s;
}

.tool-chevron.rotated { transform: rotate(180deg); }

.tool-body {
  padding: 0 10px 8px;
  font-size: 11px;
  color: var(--text-secondary);
  word-break: break-all;
  border-top: 1px solid var(--border-subtle);
  padding-top: 6px;
  margin: 0 4px;
}

/* ===== 输入区 ===== */
.input-area {
  padding: 0 24px 20px;
  flex-shrink: 0;
  max-width: 800px;
  width: 100%;
  margin: 0 auto;
  position: relative;
}

/* 底部渐变遮罩 */
.input-mask {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 120px;
  background: linear-gradient(to bottom, transparent, var(--bg-app));
  pointer-events: none;
  z-index: 1;
}

.input-card {
  background: #FFFFFF;
  border: 1px solid var(--border);
  border-radius: 16px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.04);
  transition: border-color 0.15s, box-shadow 0.15s;
  overflow: hidden;
  position: relative;
  z-index: 2;
}

.input-card.focused {
  border-color: var(--accent);
  box-shadow: 0 6px 28px rgba(218, 119, 86, 0.08);
}

.file-preview {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px 0;
  color: var(--text-secondary);
  font-size: 12px;
}

.file-preview button {
  background: transparent;
  border: none;
  cursor: pointer;
  color: var(--text-tertiary);
  display: flex;
  align-items: center;
  padding: 2px;
  border-radius: 3px;
}

.file-preview button:hover {
  color: var(--red);
}

.chat-textarea {
  width: 100%;
  padding: 12px 14px 6px;
  background: transparent;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-size: 13px;
  line-height: 1.6;
  resize: none;
  min-height: 44px;
  max-height: 200px;
  font-family: inherit;
  box-sizing: border-box;
}

.chat-textarea::placeholder {
  color: var(--text-tertiary);
}

.chat-textarea:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.input-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 8px 8px;
}

.left-actions,
.right-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.input-hint {
  font-size: 11px;
  color: var(--text-disabled);
}

.icon-btn label {
  cursor: pointer;
}

.hidden {
  display: none;
}

.send-btn {
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent);
  border: none;
  border-radius: 8px;
  color: #fff;
  cursor: pointer;
  transition: background 0.15s, transform 0.1s;
  flex-shrink: 0;
}

.send-btn:hover:not(:disabled) {
  background: var(--accent-hover);
  transform: scale(1.05);
}

.send-btn:disabled {
  background: var(--bg-overlay);
  color: var(--text-disabled);
  cursor: not-allowed;
  transform: none;
}

.input-footer {
  text-align: center;
  font-size: 11px;
  color: var(--text-disabled);
  margin-top: 8px;
}

/* ===== 重命名对话框 ===== */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-card {
  width: 360px;
  padding: 24px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.modal-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.modal-input {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  box-sizing: border-box;
  transition: border-color 0.15s;
}

.modal-input:focus {
  border-color: var(--accent);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.btn-secondary {
  padding: 6px 14px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-secondary:hover {
  background: var(--bg-elevated);
}

.btn-primary {
  padding: 6px 14px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-primary:hover {
  background: var(--accent-hover);
}

/* ===== 响应式布局 ===== */
@media (max-width: 768px) {
  .chat-root {
    flex-direction: column;
  }
  
  .session-panel {
    width: 100%;
    height: auto;
    max-height: 300px;
    border-right: none;
    border-bottom: 1px solid var(--border-subtle);
  }
  
  .chat-main {
    flex: 1;
    min-height: 0;
  }
  
  .message-row {
    padding: 8px 16px;
  }
  
  .input-area {
    padding: 0 16px 16px;
  }
}
</style>
