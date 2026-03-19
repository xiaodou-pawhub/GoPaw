<template>
  <div class="agent-messages-page">
    <div class="page-header">
      <h1 class="text-h5">Agent 消息</h1>
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openSendDialog">
        发送消息
      </v-btn>
    </div>

    <v-row class="mt-4">
      <!-- 左侧：会话列表 -->
      <v-col cols="3">
        <v-card>
          <v-card-title>会话</v-card-title>
          <v-list density="compact">
            <v-list-item
              v-for="conv in conversations"
              :key="conv.id"
              :active="selectedConversation === conv.id"
              @click="selectConversation(conv)"
            >
              <v-list-item-title>{{ conv.title }}</v-list-item-title>
              <v-list-item-subtitle>
                {{ conv.message_count }} 条消息
                <span v-if="conv.last_message_at">
                  · {{ formatDate(conv.last_message_at) }}
                </span>
              </v-list-item-subtitle>
            </v-list-item>
          </v-list>
        </v-card>

        <!-- 统计信息 -->
        <v-card class="mt-4">
          <v-card-title>统计</v-card-title>
          <v-card-text>
            <div v-if="stats">
              <div class="d-flex justify-space-between">
                <span>发送:</span>
                <span>{{ stats.total_sent }}</span>
              </div>
              <div class="d-flex justify-space-between">
                <span>接收:</span>
                <span>{{ stats.total_received }}</span>
              </div>
              <div class="d-flex justify-space-between">
                <span>待处理:</span>
                <span class="text-warning">{{ stats.pending_count }}</span>
              </div>
              <div class="d-flex justify-space-between">
                <span>失败:</span>
                <span class="text-error">{{ stats.failed_count }}</span>
              </div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- 右侧：消息列表 -->
      <v-col cols="9">
        <v-card>
          <v-card-title class="d-flex align-center">
            <span>消息</span>
            <v-spacer />
            <v-btn-toggle v-model="messageFilter" density="compact">
              <v-btn value="received">接收</v-btn>
              <v-btn value="sent">发送</v-btn>
            </v-btn-toggle>
            <v-btn
              icon="mdi-refresh"
              variant="text"
              density="compact"
              @click="loadMessages"
            />
          </v-card-title>
          <v-card-text>
            <v-data-table
              :headers="headers"
              :items="filteredMessages"
              :loading="loading"
              item-value="id"
            >
              <template #item.type="{ item }">
                <v-chip
                  :color="getTypeColor(item.type)"
                  size="small"
                >
                  {{ item.type }}
                </v-chip>
              </template>

              <template #item.status="{ item }">
                <v-chip
                  :color="getStatusColor(item.status)"
                  size="small"
                >
                  {{ item.status }}
                </v-chip>
              </template>

              <template #item.from_agent="{ item }">
                <span v-if="messageFilter === 'sent'">
                  {{ item.to_agent }}
                </span>
                <span v-else>
                  {{ item.from_agent }}
                </span>
              </template>

              <template #item.content="{ item }">
                <div class="text-truncate" style="max-width: 300px;">
                  {{ item.content }}
                </div>
              </template>

              <template #item.created_at="{ item }">
                {{ formatDate(item.created_at) }}
              </template>

              <template #item.actions="{ item }">
                <v-btn
                  icon="mdi-eye"
                  size="small"
                  variant="text"
                  @click="viewMessage(item)"
                />
                <v-btn
                  v-if="item.status === 'pending'"
                  icon="mdi-check"
                  size="small"
                  variant="text"
                  color="success"
                  @click="markCompleted(item)"
                />
                <v-btn
                  v-if="item.status === 'pending'"
                  icon="mdi-close"
                  size="small"
                  variant="text"
                  color="error"
                  @click="markFailed(item)"
                />
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- 发送消息对话框 -->
    <v-dialog v-model="sendDialog.show" max-width="600">
      <v-card>
        <v-card-title>发送消息</v-card-title>
        <v-card-text>
          <v-form ref="form" v-model="sendDialog.valid">
            <v-select
              v-model="sendDialog.data.type"
              :items="messageTypes"
              label="消息类型"
              :rules="[(v: string) => !!v || '请选择类型']"
              required
            />
            <v-select
              v-model="sendDialog.data.from_agent"
              :items="agents"
              item-title="name"
              item-value="id"
              label="发送 Agent"
              :rules="[(v: string) => !!v || '请选择发送 Agent']"
              required
            />
            <v-select
              v-model="sendDialog.data.to_agent"
              :items="agents"
              item-title="name"
              item-value="id"
              label="接收 Agent"
              :rules="[(v: string) => !!v || '请选择接收 Agent']"
              required
            />
            <v-textarea
              v-model="sendDialog.data.content"
              label="内容"
              :rules="[(v: string) => !!v || '内容不能为空']"
              rows="3"
              required
            />
            <v-textarea
              v-model="sendDialog.data.payloadText"
              label="Payload (JSON)"
              rows="3"
              placeholder='{"key": "value"}'
            />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="sendDialog.show = false">取消</v-btn>
          <v-btn color="primary" :disabled="!sendDialog.valid" @click="sendMessage">
            发送
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 查看消息对话框 -->
    <v-dialog v-model="viewDialog.show" max-width="600">
      <v-card>
        <v-card-title>消息详情</v-card-title>
        <v-card-text v-if="viewDialog.message">
          <v-row>
            <v-col cols="6">
              <div class="text-caption text-grey">ID</div>
              <div>{{ viewDialog.message.id }}</div>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">类型</div>
              <v-chip
                :color="getTypeColor(viewDialog.message.type)"
                size="small"
              >
                {{ viewDialog.message.type }}
              </v-chip>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">发送者</div>
              <div>{{ viewDialog.message.from_agent }}</div>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">接收者</div>
              <div>{{ viewDialog.message.to_agent }}</div>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">状态</div>
              <v-chip
                :color="getStatusColor(viewDialog.message.status)"
                size="small"
              >
                {{ viewDialog.message.status }}
              </v-chip>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">时间</div>
              <div>{{ formatDate(viewDialog.message.created_at) }}</div>
            </v-col>
            <v-col cols="12">
              <div class="text-caption text-grey">内容</div>
              <div class="text-body-1">{{ viewDialog.message.content }}</div>
            </v-col>
            <v-col v-if="viewDialog.message.payload" cols="12">
              <div class="text-caption text-grey">Payload</div>
              <pre class="bg-grey-lighten-3 pa-2 rounded">{{ formatPayload(viewDialog.message.payload) }}</pre>
            </v-col>
            <v-col v-if="viewDialog.message.error" cols="12">
              <div class="text-caption text-grey">错误</div>
              <div class="text-error">{{ viewDialog.message.error }}</div>
            </v-col>
          </v-row>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="viewDialog.show = false">关闭</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue'
import { agentMessagesApi, type AgentMessage, type Conversation, type MessageStats } from '@/api/agent_messages'
import { listAgents, type Agent } from '@/api/agents'

const showSnackbar = (message: string, type: 'success' | 'error' | 'info' = 'info') => {
  console.log(`[${type}] ${message}`)
}

const loading = ref(false)
const messages = ref<AgentMessage[]>([])
const conversations = ref<Conversation[]>([])
const agents = ref<Agent[]>([])
const stats = ref<MessageStats | null>(null)
const messageFilter = ref<'received' | 'sent'>('received')
const selectedConversation = ref<string | null>(null)

const headers = computed(() => [
  { title: '类型', key: 'type' },
  { title: messageFilter.value === 'sent' ? '接收者' : '发送者', key: 'from_agent' },
  { title: '内容', key: 'content' },
  { title: '状态', key: 'status' },
  { title: '时间', key: 'created_at' },
  { title: '操作', key: 'actions', sortable: false },
])

const messageTypes = [
  { title: '任务 (Task)', value: 'task' },
  { title: '响应 (Response)', value: 'response' },
  { title: '通知 (Notify)', value: 'notify' },
  { title: '查询 (Query)', value: 'query' },
  { title: '结果 (Result)', value: 'result' },
]

const sendDialog = reactive({
  show: false,
  valid: false,
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
      m.id === selectedConversation.value ||
      m.parent_id === selectedConversation.value
    )
  }
  return messages.value
})

function getTypeColor(type: string) {
  switch (type) {
    case 'task': return 'primary'
    case 'response': return 'success'
    case 'notify': return 'warning'
    case 'query': return 'info'
    case 'result': return 'secondary'
    default: return 'grey'
  }
}

function getStatusColor(status: string) {
  switch (status) {
    case 'completed': return 'success'
    case 'processing': return 'info'
    case 'pending': return 'warning'
    case 'failed': return 'error'
    default: return 'grey'
  }
}

function formatDate(date: string) {
  return new Date(date).toLocaleString('zh-CN')
}

function formatPayload(payload: any) {
  try {
    return JSON.stringify(payload, null, 2)
  } catch {
    return String(payload)
  }
}

async function loadMessages() {
  loading.value = true
  try {
    // 这里需要知道当前选中的 agent
    const currentAgent = agents.value[0]?.id
    if (!currentAgent) return

    if (messageFilter.value === 'sent') {
      const response = await agentMessagesApi.listSent(currentAgent)
      messages.value = response
    } else {
      const response = await agentMessagesApi.list(currentAgent)
      messages.value = response
    }
  } catch (error) {
    showSnackbar('加载消息失败', 'error')
  } finally {
    loading.value = false
  }
}

async function loadConversations() {
  try {
    const currentAgent = agents.value[0]?.id
    if (!currentAgent) return

    const response = await agentMessagesApi.listConversations(currentAgent)
    conversations.value = response
  } catch (error) {
    showSnackbar('加载会话失败', 'error')
  }
}

async function loadStats() {
  try {
    const currentAgent = agents.value[0]?.id
    if (!currentAgent) return

    const response = await agentMessagesApi.getStats(currentAgent)
    stats.value = response
  } catch (error) {
    showSnackbar('加载统计失败', 'error')
  }
}

async function loadAgents() {
  try {
    const response = await listAgents()
    agents.value = response.agents
  } catch (error) {
    showSnackbar('加载 Agents 失败', 'error')
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
  try {
    let payload = {}
    if (sendDialog.data.payloadText) {
      try {
        payload = JSON.parse(sendDialog.data.payloadText)
      } catch {
        showSnackbar('Payload JSON 格式错误', 'error')
        return
      }
    }

    await agentMessagesApi.send({
      type: sendDialog.data.type,
      from_agent: sendDialog.data.from_agent,
      to_agent: sendDialog.data.to_agent,
      content: sendDialog.data.content,
      payload,
    })

    showSnackbar('消息发送成功', 'success')
    sendDialog.show = false
    loadMessages()
    loadStats()
  } catch (error: any) {
    showSnackbar(error.response?.data?.error || '发送失败', 'error')
  }
}

function viewMessage(msg: AgentMessage) {
  viewDialog.message = msg
  viewDialog.show = true
}

async function markCompleted(msg: AgentMessage) {
  try {
    await agentMessagesApi.updateStatus(msg.id, 'completed')
    showSnackbar('标记完成', 'success')
    loadMessages()
    loadStats()
  } catch (error) {
    showSnackbar('操作失败', 'error')
  }
}

async function markFailed(msg: AgentMessage) {
  try {
    await agentMessagesApi.updateStatus(msg.id, 'failed', '手动标记失败')
    showSnackbar('标记失败', 'success')
    loadMessages()
    loadStats()
  } catch (error) {
    showSnackbar('操作失败', 'error')
  }
}

function selectConversation(conv: Conversation) {
  selectedConversation.value = conv.id
  // Load conversation messages
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
  padding: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
