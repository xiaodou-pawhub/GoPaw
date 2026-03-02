<template>
  <div class="chat-page">
    <n-card :title="t('chat.title')" class="chat-card">
      <div class="chat-container">
        <!-- 中文：消息列表 / English: Message list -->
        <div ref="messagesRef" class="messages-container">
          <div v-for="msg in messages" :key="msg.id" class="message" :class="msg.role">
            <div class="message-avatar">
              <n-icon v-if="msg.role === 'user'" :component="PersonOutline" :size="24" />
              <n-icon v-else :component="Logo" :size="24" color="#18a058" />
            </div>
            <div class="message-content">
              <div v-if="msg.role === 'assistant'" class="markdown-body" v-html="renderMarkdown(msg.content)"></div>
              <div v-else>{{ msg.content }}</div>
              <div class="message-time">{{ msg.time }}</div>
            </div>
          </div>
          
          <!-- 中文：思考中状态 / English: Thinking status -->
          <div v-if="isThinking" class="message assistant">
            <div class="message-avatar">
              <n-icon :component="Logo" :size="24" color="#18a058" />
            </div>
            <div class="message-content">
              <n-spin size="small" />
              <n-text depth="3">{{ t('chat.thinking') }}</n-text>
            </div>
          </div>
        </div>
        
        <!-- 中文：输入区域 / English: Input area -->
        <div class="input-area">
          <n-input
            v-model:value="inputMessage"
            type="textarea"
            :placeholder="t('chat.placeholder')"
            :rows="3"
            :disabled="!appStore.isLLMConfigured"
            @keydown="handleKeydown"
          />
          <n-button
            type="primary"
            :disabled="!inputMessage.trim() || !appStore.isLLMConfigured || isThinking"
            :loading="isThinking"
            @click="sendMessage"
          >
            {{ t('chat.send') }}
          </n-button>
        </div>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, nextTick, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  NCard, NIcon, NButton, NInput, NSpin, NText, useMessage
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { Logo, PersonOutline } from '@vicons/ionicons5'
import { useAppStore } from '@/stores/app'
import markdownIt from 'markdown-it'

const router = useRouter()
const { t } = useI18n()
const message = useMessage()
const appStore = useAppStore()

const md = markdownIt()

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  time: string
}

const messages = ref<Message[]>([])
const inputMessage = ref('')
const isThinking = ref(false)
const messagesRef = ref<HTMLElement | null>(null)

// 中文：渲染 Markdown
// English: Render Markdown
function renderMarkdown(content: string) {
  return md.render(content)
}

// 中文：滚动到底部
// English: Scroll to bottom
async function scrollToBottom() {
  await nextTick()
  if (messagesRef.value) {
    messagesRef.value.scrollTop = messagesRef.value.scrollHeight
  }
}

// 中文：处理键盘事件
// English: Handle keyboard event
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

// 中文：发送消息
// English: Send message
async function sendMessage() {
  if (!inputMessage.value.trim() || isThinking.value) return
  
  const userMsg: Message = {
    id: Date.now().toString(),
    role: 'user',
    content: inputMessage.value,
    time: new Date().toLocaleTimeString()
  }
  
  messages.value.push(userMsg)
  inputMessage.value = ''
  isThinking.value = true
  await scrollToBottom()
  
  // TODO: 调用后端 API / Call backend API
  setTimeout(() => {
    const assistantMsg: Message = {
      id: (Date.now() + 1).toString(),
      role: 'assistant',
      content: '这是一个测试回复。后端 API 集成开发中... This is a test reply. Backend API integration coming soon...',
      time: new Date().toLocaleTimeString()
    }
    messages.value.push(assistantMsg)
    isThinking.value = false
    scrollToBottom()
  }, 1000)
}

onMounted(() => {
  if (!appStore.isLLMConfigured) {
    message.info('请先配置 LLM 提供商 / Please configure LLM provider first')
    router.push('/setup')
  }
  
  // 中文：添加欢迎消息
  // English: Add welcome message
  messages.value.push({
    id: 'welcome',
    role: 'assistant',
    content: t('chat.welcome'),
    time: new Date().toLocaleTimeString()
  })
})
</script>

<style scoped>
.chat-page {
  padding: 24px;
  height: 100%;
}

.chat-card {
  height: calc(100% - 48px);
}

.chat-container {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 200px);
}

.messages-container {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.message {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.message.user {
  flex-direction: row-reverse;
}

.message-content {
  max-width: 70%;
  padding: 12px 16px;
  border-radius: 12px;
  background: #f0f0f0;
}

.message.user .message-content {
  background: #18a058;
  color: white;
}

.message-time {
  margin-top: 4px;
  font-size: 12px;
  opacity: 0.6;
}

.input-area {
  display: flex;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid #eee;
}

.input-area textarea {
  resize: none;
}
</style>
