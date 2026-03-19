<template>
  <div class="queue-page">
    <div class="page-header">
      <h1 class="text-h5">消息队列</h1>
      <v-btn color="primary" prepend-icon="mdi-plus" @click="showPublishDialog">
        发布消息
      </v-btn>
    </div>

    <v-row class="mt-4">
      <!-- 队列列表 -->
      <v-col cols="3">
        <v-card>
          <v-card-title>队列</v-card-title>
          <v-list density="compact">
            <v-list-item
              v-for="q in queues"
              :key="q.name"
              :active="selectedQueue === q.name"
              @click="selectQueue(q.name)"
            >
              <v-list-item-title>{{ q.name }}</v-list-item-title>
              <v-list-item-subtitle>
                <v-chip size="x-small" color="warning" class="mr-1">
                  {{ q.pending_count }}
                </v-chip>
                <v-chip size="x-small" color="info" class="mr-1">
                  {{ q.processing_count }}
                </v-chip>
                <v-chip size="x-small" color="error">
                  {{ q.failed_count }}
                </v-chip>
              </v-list-item-subtitle>
            </v-list-item>
          </v-list>
        </v-card>
      </v-col>

      <!-- 消息列表 -->
      <v-col cols="9">
        <v-card v-if="selectedQueue">
          <v-card-title class="d-flex align-center">
            <span>{{ selectedQueue }}</span>
            <v-spacer />
            <v-btn-toggle v-model="statusFilter" mandatory density="compact">
              <v-btn value="">全部</v-btn>
              <v-btn value="pending">待处理</v-btn>
              <v-btn value="processing">处理中</v-btn>
              <v-btn value="completed">已完成</v-btn>
              <v-btn value="failed">失败</v-btn>
              <v-btn value="delayed">延迟</v-btn>
            </v-btn-toggle>
            <v-btn
              icon="mdi-refresh"
              size="small"
              variant="text"
              @click="loadMessages"
              class="ml-2"
            />
          </v-card-title>
          <v-card-text>
            <v-data-table
              :headers="headers"
              :items="messages"
              :loading="loading"
              density="compact"
            >
              <template #item.priority="{ item }">
                <v-chip
                  :color="getPriorityColor(item.priority)"
                  size="small"
                >
                  {{ item.priority }}
                </v-chip>
              </template>
              <template #item.status="{ item }">
                <v-chip :color="getStatusColor(item.status)" size="small">
                  {{ item.status }}
                </v-chip>
              </template>
              <template #item.created_at="{ item }">
                {{ formatDate(item.created_at) }}
              </template>
              <template #item.actions="{ item }">
                <v-btn
                  v-if="item.status === 'failed'"
                  icon="mdi-refresh"
                  size="x-small"
                  variant="text"
                  @click="retryMessage(item)"
                />
                <v-btn
                  icon="mdi-delete"
                  size="x-small"
                  variant="text"
                  color="error"
                  @click="deleteMessage(item)"
                />
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- 发布消息对话框 -->
    <v-dialog v-model="publishDialog.show" max-width="600">
      <v-card>
        <v-card-title>发布消息</v-card-title>
        <v-card-text>
          <v-select
            v-model="publishDialog.queue"
            :items="queueNames"
            label="队列"
            required
          />
          <v-text-field
            v-model="publishDialog.type"
            label="类型"
            required
          />
          <v-textarea
            v-model="publishDialog.payload"
            label="Payload (JSON)"
            rows="5"
            :rules="[(v: string) => {
              if (!v) return 'Payload 不能为空'
              try {
                JSON.parse(v)
                return true
              } catch {
                return '无效的 JSON'
              }
            }]"
          />
          <v-slider
            v-model="publishDialog.priority"
            label="优先级"
            min="0"
            max="9"
            step="1"
            thumb-label
          />
          <v-text-field
            v-model="publishDialog.maxRetries"
            label="最大重试次数"
            type="number"
          />
          <v-text-field
            v-model="publishDialog.delaySeconds"
            label="延迟（秒）"
            type="number"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="publishDialog.show = false">取消</v-btn>
          <v-btn color="primary" @click="confirmPublish">发布</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, watch, computed } from 'vue'
import { queueApi, type Message, type QueueInfo, type MessageStatus } from '@/api/queue'

const showSnackbar = (message: string, type: 'success' | 'error' | 'info' = 'info') => {
  console.log(`[${type}] ${message}`)
}

const queues = ref<QueueInfo[]>([])
const selectedQueue = ref<string>('')
const messages = ref<Message[]>([])
const loading = ref(false)
const statusFilter = ref<MessageStatus | ''>('')

const headers = [
  { title: 'ID', key: 'id' },
  { title: '类型', key: 'type' },
  { title: '优先级', key: 'priority' },
  { title: '状态', key: 'status' },
  { title: '重试', key: 'attempts' },
  { title: '创建时间', key: 'created_at' },
  { title: '操作', key: 'actions', sortable: false },
]

const publishDialog = reactive({
  show: false,
  queue: '',
  type: '',
  payload: '{}',
  priority: 5,
  maxRetries: 3,
  delaySeconds: 0,
})

const queueNames = computed(() => queues.value.map(q => q.name))

function getPriorityColor(priority: number) {
  if (priority <= 2) return 'error'
  if (priority <= 5) return 'warning'
  return 'success'
}

function getStatusColor(status: string) {
  const colors: Record<string, string> = {
    pending: 'warning',
    processing: 'info',
    completed: 'success',
    failed: 'error',
    delayed: 'grey',
  }
  return colors[status] || 'grey'
}

function formatDate(date: string) {
  return new Date(date).toLocaleString('zh-CN')
}

async function loadQueues() {
  try {
    const response = await queueApi.listQueues()
    queues.value = response
  } catch (error) {
    showSnackbar('加载队列失败', 'error')
  }
}

async function loadMessages() {
  if (!selectedQueue.value) return
  loading.value = true
  try {
    const response = await queueApi.listMessages(
      selectedQueue.value,
      statusFilter.value || undefined,
      100
    )
    messages.value = response
  } catch (error) {
    showSnackbar('加载消息失败', 'error')
  } finally {
    loading.value = false
  }
}

function selectQueue(name: string) {
  selectedQueue.value = name
  loadMessages()
}

function showPublishDialog() {
  publishDialog.queue = selectedQueue.value || 'workflow'
  publishDialog.type = ''
  publishDialog.payload = '{}'
  publishDialog.priority = 5
  publishDialog.maxRetries = 3
  publishDialog.delaySeconds = 0
  publishDialog.show = true
}

async function confirmPublish() {
  try {
    let payload: Record<string, any>
    try {
      payload = JSON.parse(publishDialog.payload)
    } catch {
      showSnackbar('无效的 JSON', 'error')
      return
    }

    await queueApi.publishMessage(publishDialog.queue, {
      type: publishDialog.type,
      payload,
      priority: publishDialog.priority,
      max_retries: publishDialog.maxRetries,
      delay_seconds: publishDialog.delaySeconds || undefined,
    })

    showSnackbar('消息发布成功', 'success')
    publishDialog.show = false
    loadMessages()
    loadQueues()
  } catch (error) {
    showSnackbar('发布失败', 'error')
  }
}

async function retryMessage(msg: Message) {
  try {
    await queueApi.retryMessage(msg.id)
    showSnackbar('消息已重试', 'success')
    loadMessages()
    loadQueues()
  } catch (error) {
    showSnackbar('重试失败', 'error')
  }
}

async function deleteMessage(msg: Message) {
  try {
    await queueApi.deleteMessage(msg.id)
    showSnackbar('消息已删除', 'success')
    loadMessages()
    loadQueues()
  } catch (error) {
    showSnackbar('删除失败', 'error')
  }
}

watch(statusFilter, () => {
  loadMessages()
})

onMounted(() => {
  loadQueues()
})
</script>

<style scoped>
.queue-page {
  padding: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
