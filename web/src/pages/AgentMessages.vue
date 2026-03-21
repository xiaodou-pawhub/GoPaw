<template>
  <div class="agent-messages-page">
    <div class="page-header">
      <h1 class="page-title">Agent 消息</h1>
      <button class="btn-primary" @click="openSendDialog">
        <SendIcon :size="16" />
        发送消息
      </button>
    </div>

    <div class="msg-layout">
      <!-- 左侧：会话列表 + 统计 -->
      <div class="left-panel">
        <div class="panel-card">
          <div class="panel-section-title">会话</div>
          <div
            v-for="conv in conversations"
            :key="conv.id"
            class="conv-item"
            :class="{ active: selectedConversation === conv.id }"
            @click="selectConversation(conv)"
          >
            <div class="conv-title">{{ conv.title }}</div>
            <div class="conv-meta">
              {{ conv.message_count }} 条消息
              <span v-if="conv.last_message_at">· {{ formatDate(conv.last_message_at) }}</span>
            </div>
          </div>
          <div v-if="conversations.length === 0" class="empty-state">暂无会话</div>
        </div>

        <div v-if="stats" class="panel-card">
          <div class="panel-section-title">统计</div>
          <div class="stats-list">
            <div class="stat-row">
              <span>发送</span><span>{{ stats.total_sent }}</span>
            </div>
            <div class="stat-row">
              <span>接收</span><span>{{ stats.total_received }}</span>
            </div>
            <div class="stat-row">
              <span>待处理</span>
              <span class="text-warning">{{ stats.pending_count }}</span>
            </div>
            <div class="stat-row">
              <span>失败</span>
              <span class="text-error">{{ stats.failed_count }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧：消息列表 -->
      <div class="right-panel">
        <div class="messages-header">
          <span class="panel-section-title" style="flex:1">消息</span>
          <div class="filter-tabs">
            <button class="filter-tab" :class="{ active: messageFilter === 'received' }" @click="messageFilter = 'received'; loadMessages()">接收</button>
            <button class="filter-tab" :class="{ active: messageFilter === 'sent' }" @click="messageFilter = 'sent'; loadMessages()">发送</button>
          </div>
          <button class="btn-icon" title="刷新" @click="loadMessages">
            <RefreshCwIcon :size="15" />
          </button>
        </div>

        <div class="msg-table">
          <div class="msg-thead">
            <span>类型</span>
            <span>{{ messageFilter === 'sent' ? '接收者' : '发送者' }}</span>
            <span>内容</span>
            <span>状态</span>
            <span>时间</span>
            <span>操作</span>
          </div>
          <div v-if="loading" class="empty-state">加载中...</div>
          <div v-else-if="filteredMessages.length === 0" class="empty-state">暂无消息</div>
          <div v-for="msg in filteredMessages" :key="msg.id" class="msg-row">
            <span>
              <span class="badge" :class="getTypeClass(msg.type)">{{ msg.type }}</span>
            </span>
            <span class="text-ellipsis">{{ messageFilter === 'sent' ? msg.to_agent : msg.from_agent }}</span>
            <span class="text-ellipsis">{{ msg.content }}</span>
            <span>
              <span class="badge" :class="getStatusClass(msg.status)">{{ msg.status }}</span>
            </span>
            <span class="text-sm">{{ formatDate(msg.created_at) }}</span>
            <span class="actions">
              <button class="action-btn" title="查看" @click="viewMessage(msg)">
                <EyeIcon :size="14" />
              </button>
              <button
                v-if="msg.status === 'pending'"
                class="action-btn action-success"
                title="标记完成"
                @click="markCompleted(msg)"
              >
                <CheckIcon :size="14" />
              </button>
              <button
                v-if="msg.status === 'pending'"
                class="action-btn action-danger"
                title="标记失败"
                @click="markFailed(msg)"
              >
                <XIcon :size="14" />
              </button>
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 发送消息弹窗 -->
    <div v-if="sendDialog.show" class="modal-overlay" @click.self="sendDialog.show = false">
      <div class="modal-card">
        <h2 class="modal-title">发送消息</h2>
        <div class="form-group">
          <label>消息类型</label>
          <select v-model="sendDialog.data.type">
            <option value="task">任务 (Task)</option>
            <option value="response">响应 (Response)</option>
            <option value="notify">通知 (Notify)</option>
            <option value="query">查询 (Query)</option>
            <option value="result">结果 (Result)</option>
          </select>
        </div>
        <div class="form-group">
          <label>发送 Agent</label>
          <select v-model="sendDialog.data.from_agent">
            <option value="">请选择...</option>
            <option v-for="agent in agents" :key="agent.id" :value="agent.id">{{ agent.name }}</option>
          </select>
        </div>
        <div class="form-group">
          <label>接收 Agent</label>
          <select v-model="sendDialog.data.to_agent">
            <option value="">请选择...</option>
            <option v-for="agent in agents" :key="agent.id" :value="agent.id">{{ agent.name }}</option>
          </select>
        </div>
        <div class="form-group">
          <label>内容</label>
          <textarea v-model="sendDialog.data.content" rows="3" placeholder="消息内容" />
        </div>
        <div class="form-group">
          <label>Payload (JSON)</label>
          <textarea v-model="sendDialog.data.payloadText" rows="3" placeholder='{"key": "value"}' />
        </div>
        <div class="modal-actions">
          <button class="btn-ghost" @click="sendDialog.show = false">取消</button>
          <button class="btn-primary" @click="sendMessage">发送</button>
        </div>
      </div>
    </div>

    <!-- 查看消息弹窗 -->
    <div v-if="viewDialog.show" class="modal-overlay" @click.self="viewDialog.show = false">
      <div class="modal-card" v-if="viewDialog.message">
        <h2 class="modal-title">消息详情</h2>
        <div class="detail-grid">
          <div class="detail-item">
            <div class="detail-label">ID</div>
            <div class="detail-value mono">{{ viewDialog.message.id }}</div>
          </div>
          <div class="detail-item">
            <div class="detail-label">类型</div>
            <div class="detail-value">
              <span class="badge" :class="getTypeClass(viewDialog.message.type)">{{ viewDialog.message.type }}</span>
            </div>
          </div>
          <div class="detail-item">
            <div class="detail-label">发送者</div>
            <div class="detail-value">{{ viewDialog.message.from_agent }}</div>
          </div>
          <div class="detail-item">
            <div class="detail-label">接收者</div>
            <div class="detail-value">{{ viewDialog.message.to_agent }}</div>
          </div>
          <div class="detail-item">
            <div class="detail-label">状态</div>
            <div class="detail-value">
              <span class="badge" :class="getStatusClass(viewDialog.message.status)">{{ viewDialog.message.status }}</span>
            </div>
          </div>
          <div class="detail-item">
            <div class="detail-label">时间</div>
            <div class="detail-value">{{ formatDate(viewDialog.message.created_at) }}</div>
          </div>
          <div class="detail-item full">
            <div class="detail-label">内容</div>
            <div class="detail-value">{{ viewDialog.message.content }}</div>
          </div>
          <div v-if="viewDialog.message.payload" class="detail-item full">
            <div class="detail-label">Payload</div>
            <pre class="code-block">{{ formatPayload(viewDialog.message.payload) }}</pre>
          </div>
          <div v-if="viewDialog.message.error" class="detail-item full">
            <div class="detail-label">错误</div>
            <div class="detail-value text-error">{{ viewDialog.message.error }}</div>
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn-ghost" @click="viewDialog.show = false">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue'
import { SendIcon, RefreshCwIcon, EyeIcon, CheckIcon, XIcon } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { agentMessagesApi, type AgentMessage, type Conversation, type MessageStats } from '@/api/agent_messages'
import { listAgents, type Agent } from '@/api/agents'

const loading = ref(false)
const messages = ref<AgentMessage[]>([])
const conversations = ref<Conversation[]>([])
const agents = ref<Agent[]>([])
const stats = ref<MessageStats | null>(null)
const messageFilter = ref<'received' | 'sent'>('received')
const selectedConversation = ref<string | null>(null)

const sendDialog = reactive({
  show: false,
  data: {
    type: 'task' as 'task' | 'response' | 'notify' | 'query' | 'result',
    from_agent: '',
    to_agent: '',
    content: '',
    payloadText: '',
  },
})

const viewDialog = reactive({
  show: false,
  message: null as AgentMessage | null,
})

const filteredMessages = computed(() => {
  if (selectedConversation.value) {
    return messages.value.filter(m =>
      m.id === selectedConversation.value || m.parent_id === selectedConversation.value,
    )
  }
  return messages.value
})

function getTypeClass(type: string) {
  const map: Record<string, string> = {
    task: 'badge-info',
    response: 'badge-success',
    notify: 'badge-warning',
    query: 'badge-purple',
    result: 'badge-neutral',
  }
  return map[type] || 'badge-neutral'
}

function getStatusClass(status: string) {
  const map: Record<string, string> = {
    completed: 'badge-success',
    processing: 'badge-info',
    pending: 'badge-warning',
    failed: 'badge-error',
  }
  return map[status] || 'badge-neutral'
}

function formatDate(date: string) {
  return new Date(date).toLocaleString('zh-CN')
}

function formatPayload(payload: unknown) {
  try { return JSON.stringify(payload, null, 2) } catch { return String(payload) }
}

async function loadMessages() {
  loading.value = true
  try {
    const currentAgent = agents.value[0]?.id
    if (!currentAgent) return
    if (messageFilter.value === 'sent') {
      messages.value = await agentMessagesApi.listSent(currentAgent)
    } else {
      messages.value = await agentMessagesApi.list(currentAgent)
    }
  } catch {
    toast.error('加载消息失败')
  } finally {
    loading.value = false
  }
}

async function loadConversations() {
  try {
    const currentAgent = agents.value[0]?.id
    if (!currentAgent) return
    conversations.value = await agentMessagesApi.listConversations(currentAgent)
  } catch {
    toast.error('加载会话失败')
  }
}

async function loadStats() {
  try {
    const currentAgent = agents.value[0]?.id
    if (!currentAgent) return
    stats.value = await agentMessagesApi.getStats(currentAgent)
  } catch {
    // 忽略
  }
}

async function loadAgents() {
  try {
    const response = await listAgents()
    agents.value = response.agents
  } catch {
    toast.error('加载 Agents 失败')
  }
}

function openSendDialog() {
  sendDialog.data = {
    type: 'task',
    from_agent: agents.value[0]?.id || '',
    to_agent: '',
    content: '',
    payloadText: '',
  }
  sendDialog.show = true
}

async function sendMessage() {
  let payload = {}
  if (sendDialog.data.payloadText) {
    try {
      payload = JSON.parse(sendDialog.data.payloadText)
    } catch {
      toast.error('Payload JSON 格式错误')
      return
    }
  }
  try {
    await agentMessagesApi.send({
      type: sendDialog.data.type,
      from_agent: sendDialog.data.from_agent,
      to_agent: sendDialog.data.to_agent,
      content: sendDialog.data.content,
      payload,
    })
    toast.success('消息发送成功')
    sendDialog.show = false
    loadMessages()
    loadStats()
  } catch (error: unknown) {
    const msg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
    toast.error(msg || '发送失败')
  }
}

function viewMessage(msg: AgentMessage) {
  viewDialog.message = msg
  viewDialog.show = true
}

async function markCompleted(msg: AgentMessage) {
  try {
    await agentMessagesApi.updateStatus(msg.id, 'completed')
    toast.success('已标记完成')
    loadMessages()
    loadStats()
  } catch {
    toast.error('操作失败')
  }
}

async function markFailed(msg: AgentMessage) {
  try {
    await agentMessagesApi.updateStatus(msg.id, 'failed', '手动标记失败')
    toast.success('已标记失败')
    loadMessages()
    loadStats()
  } catch {
    toast.error('操作失败')
  }
}

function selectConversation(conv: Conversation) {
  selectedConversation.value = conv.id
  agentMessagesApi.listConversation(conv.id).then(response => {
    messages.value = response
  })
}

onMounted(async () => {
  await loadAgents()
  loadMessages()
  loadConversations()
  loadStats()
})
</script>

<style scoped>
.agent-messages-page {
  padding: 24px 32px;
  height: 100%;
  overflow-y: auto;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.page-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-primary:hover { background: var(--accent-hover); }

.btn-ghost {
  padding: 8px 16px;
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
}

.btn-ghost:hover { background: var(--bg-overlay); }

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-secondary);
}

.btn-icon:hover { background: var(--bg-overlay); }

.msg-layout {
  display: grid;
  grid-template-columns: 240px 1fr;
  gap: 16px;
  min-height: 400px;
}

.left-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.panel-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}

.panel-section-title {
  padding: 10px 16px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--bg-overlay);
  border-bottom: 1px solid var(--border);
}

.conv-item {
  padding: 10px 16px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-subtle);
  transition: background 0.1s;
}

.conv-item:hover { background: var(--bg-overlay); }
.conv-item.active { background: var(--accent-dim); }

.conv-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conv-meta {
  font-size: 11px;
  color: var(--text-tertiary);
  margin-top: 2px;
}

.stats-list { padding: 8px 0; }

.stat-row {
  display: flex;
  justify-content: space-between;
  padding: 6px 16px;
  font-size: 13px;
  color: var(--text-secondary);
}

.text-warning { color: #ca8a04; }
.text-error   { color: #ef4444; }
.text-sm      { font-size: 12px; color: var(--text-secondary); }

.right-panel {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.messages-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-overlay);
}

.filter-tabs {
  display: flex;
  gap: 4px;
}

.filter-tab {
  padding: 4px 10px;
  border-radius: 4px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
}

.filter-tab:hover { background: var(--bg-overlay); }
.filter-tab.active {
  background: var(--accent-dim);
  color: var(--accent);
  border-color: var(--accent);
}

.msg-table { flex: 1; overflow-y: auto; }

.msg-thead, .msg-row {
  display: grid;
  grid-template-columns: 90px 120px 1fr 100px 130px 90px;
  padding: 10px 16px;
  align-items: center;
  gap: 8px;
}

.msg-thead {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--bg-overlay);
  border-bottom: 1px solid var(--border);
}

.msg-row {
  font-size: 13px;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-subtle);
}

.msg-row:last-child { border-bottom: none; }

.text-ellipsis {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.badge {
  display: inline-block;
  padding: 2px 7px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.badge-info    { background: rgba(59,130,246,0.15);  color: #3b82f6; }
.badge-success { background: rgba(34,197,94,0.15);   color: #16a34a; }
.badge-warning { background: rgba(234,179,8,0.15);   color: #ca8a04; }
.badge-error   { background: rgba(239,68,68,0.15);   color: #ef4444; }
.badge-purple  { background: rgba(168,85,247,0.15);  color: #a855f7; }
.badge-neutral { background: var(--bg-overlay); color: var(--text-secondary); border: 1px solid var(--border); }

.actions {
  display: flex;
  gap: 4px;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all 0.1s;
}

.action-btn:hover { background: var(--bg-overlay); }
.action-success:hover { background: rgba(34,197,94,0.1); color: #16a34a; border-color: rgba(34,197,94,0.3); }
.action-danger:hover  { background: rgba(239,68,68,0.1); color: #ef4444; border-color: rgba(239,68,68,0.3); }

.empty-state {
  padding: 24px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 13px;
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 24px;
  width: 520px;
  max-width: 95vw;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 20px;
}

.form-group {
  margin-bottom: 14px;
}

.form-group label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 8px 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  box-sizing: border-box;
}

.form-group textarea { resize: vertical; }

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 20px;
}

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.detail-item.full { grid-column: 1 / -1; }

.detail-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 4px;
}

.detail-value {
  font-size: 13px;
  color: var(--text-primary);
}

.detail-value.mono { font-family: monospace; font-size: 12px; }

.code-block {
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 10px;
  font-size: 12px;
  overflow-x: auto;
  white-space: pre;
  margin: 0;
  color: var(--text-primary);
}
</style>
