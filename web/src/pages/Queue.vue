<template>
  <div class="queue-page">
    <div class="page-header">
      <h1 class="page-title">消息队列</h1>
      <button class="btn-primary" @click="showPublishDialog">
        <PlusIcon :size="16" />
        发布消息
      </button>
    </div>

    <div class="queue-layout">
      <!-- 队列列表 -->
      <div class="queue-sidebar">
        <div class="sidebar-title">队列</div>
        <div
          v-for="q in queues"
          :key="q.name"
          class="queue-item"
          :class="{ active: selectedQueue === q.name }"
          @click="selectQueue(q.name)"
        >
          <div class="queue-name">{{ q.name }}</div>
          <div class="queue-counts">
            <span class="count count-warning">{{ q.pending_count }}</span>
            <span class="count count-info">{{ q.processing_count }}</span>
            <span class="count count-error">{{ q.failed_count }}</span>
          </div>
        </div>
        <div v-if="queues.length === 0" class="empty-state">暂无队列</div>
      </div>

      <!-- 消息列表 -->
      <div class="messages-panel">
        <template v-if="selectedQueue">
          <div class="messages-header">
            <span class="messages-title">{{ selectedQueue }}</span>
            <div class="filter-tabs">
              <button
                v-for="f in statusFilters"
                :key="f.value"
                class="filter-tab"
                :class="{ active: statusFilter === f.value }"
                @click="statusFilter = f.value as any; loadMessages()"
              >{{ f.label }}</button>
            </div>
            <button class="btn-icon" title="刷新" @click="loadMessages">
              <RefreshCwIcon :size="15" />
            </button>
          </div>

          <div class="msg-table">
            <div class="msg-thead">
              <span>ID</span>
              <span>类型</span>
              <span>优先级</span>
              <span>状态</span>
              <span>重试</span>
              <span>创建时间</span>
              <span>操作</span>
            </div>
            <div v-if="loading" class="empty-state">加载中...</div>
            <div v-else-if="messages.length === 0" class="empty-state">暂无消息</div>
            <div v-for="msg in messages" :key="msg.id" class="msg-row">
              <span class="msg-id" :title="msg.id">{{ msg.id.substring(0, 8) }}...</span>
              <span>{{ msg.type }}</span>
              <span>
                <span class="badge" :class="getPriorityClass(msg.priority)">{{ msg.priority }}</span>
              </span>
              <span>
                <span class="badge" :class="getStatusClass(msg.status)">{{ msg.status }}</span>
              </span>
              <span>{{ msg.attempts }}</span>
              <span>{{ formatDate(msg.created_at) }}</span>
              <span class="actions">
                <button
                  v-if="msg.status === 'failed'"
                  class="action-btn"
                  title="重试"
                  @click="retryMessage(msg)"
                >
                  <RefreshCwIcon :size="14" />
                </button>
                <button
                  class="action-btn action-danger"
                  title="删除"
                  @click="deleteMessage(msg)"
                >
                  <Trash2Icon :size="14" />
                </button>
              </span>
            </div>
          </div>
        </template>
        <div v-else class="empty-state-full">
          <InboxIcon :size="40" />
          <p>请选择一个队列</p>
        </div>
      </div>
    </div>

    <!-- 发布消息弹窗 -->
    <div v-if="publishDialog.show" class="modal-overlay" @click.self="publishDialog.show = false">
      <div class="modal-card">
        <h2 class="modal-title">发布消息</h2>
        <div class="form-group">
          <label>队列</label>
          <select v-model="publishDialog.queue">
            <option v-for="name in queueNames" :key="name" :value="name">{{ name }}</option>
          </select>
        </div>
        <div class="form-group">
          <label>类型</label>
          <input v-model="publishDialog.type" type="text" placeholder="消息类型" required />
        </div>
        <div class="form-group">
          <label>Payload (JSON)</label>
          <textarea v-model="publishDialog.payload" rows="5" placeholder="{}" />
          <div v-if="payloadError" class="error-text">{{ payloadError }}</div>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label>优先级 (0-9)</label>
            <input v-model.number="publishDialog.priority" type="number" min="0" max="9" />
          </div>
          <div class="form-group">
            <label>最大重试次数</label>
            <input v-model.number="publishDialog.maxRetries" type="number" min="0" />
          </div>
          <div class="form-group">
            <label>延迟（秒）</label>
            <input v-model.number="publishDialog.delaySeconds" type="number" min="0" />
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn-ghost" @click="publishDialog.show = false">取消</button>
          <button class="btn-primary" @click="confirmPublish">发布</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue'
import { PlusIcon, RefreshCwIcon, Trash2Icon, InboxIcon } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { queueApi, type Message, type QueueInfo, type MessageStatus } from '@/api/queue'

const queues = ref<QueueInfo[]>([])
const selectedQueue = ref<string>('')
const messages = ref<Message[]>([])
const loading = ref(false)
const statusFilter = ref<MessageStatus | ''>('')
const payloadError = ref('')

const statusFilters = [
  { value: '', label: '全部' },
  { value: 'pending', label: '待处理' },
  { value: 'processing', label: '处理中' },
  { value: 'completed', label: '已完成' },
  { value: 'failed', label: '失败' },
  { value: 'delayed', label: '延迟' },
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

function getPriorityClass(priority: number) {
  if (priority <= 2) return 'badge-error'
  if (priority <= 5) return 'badge-warning'
  return 'badge-success'
}

function getStatusClass(status: string) {
  const map: Record<string, string> = {
    pending: 'badge-warning',
    processing: 'badge-info',
    completed: 'badge-success',
    failed: 'badge-error',
    delayed: 'badge-neutral',
  }
  return map[status] || 'badge-neutral'
}

function formatDate(date: string) {
  return new Date(date).toLocaleString('zh-CN')
}

async function loadQueues() {
  try {
    queues.value = await queueApi.listQueues()
  } catch {
    toast.error('加载队列失败')
  }
}

async function loadMessages() {
  if (!selectedQueue.value) return
  loading.value = true
  try {
    messages.value = await queueApi.listMessages(
      selectedQueue.value,
      statusFilter.value || undefined,
      100,
    )
  } catch {
    toast.error('加载消息失败')
  } finally {
    loading.value = false
  }
}

function selectQueue(name: string) {
  selectedQueue.value = name
  loadMessages()
}

function showPublishDialog() {
  publishDialog.queue = selectedQueue.value || (queueNames.value[0] ?? '')
  publishDialog.type = ''
  publishDialog.payload = '{}'
  publishDialog.priority = 5
  publishDialog.maxRetries = 3
  publishDialog.delaySeconds = 0
  payloadError.value = ''
  publishDialog.show = true
}

async function confirmPublish() {
  let payload: Record<string, unknown>
  try {
    payload = JSON.parse(publishDialog.payload)
    payloadError.value = ''
  } catch {
    payloadError.value = '无效的 JSON'
    return
  }
  try {
    await queueApi.publishMessage(publishDialog.queue, {
      type: publishDialog.type,
      payload,
      priority: publishDialog.priority,
      max_retries: publishDialog.maxRetries,
      delay_seconds: publishDialog.delaySeconds || undefined,
    })
    toast.success('消息发布成功')
    publishDialog.show = false
    loadMessages()
    loadQueues()
  } catch {
    toast.error('发布失败')
  }
}

async function retryMessage(msg: Message) {
  try {
    await queueApi.retryMessage(msg.id)
    toast.success('消息已重试')
    loadMessages()
    loadQueues()
  } catch {
    toast.error('重试失败')
  }
}

async function deleteMessage(msg: Message) {
  try {
    await queueApi.deleteMessage(msg.id)
    toast.success('消息已删除')
    loadMessages()
    loadQueues()
  } catch {
    toast.error('删除失败')
  }
}

onMounted(() => {
  loadQueues()
})
</script>

<style scoped>
.queue-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 24px;
  height: 100%;
  overflow: hidden;
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

.btn-primary:hover {
  background: var(--accent-hover);
}

.btn-ghost {
  padding: 8px 16px;
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
}

.btn-ghost:hover {
  background: var(--bg-overlay);
}

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

.btn-icon:hover {
  background: var(--bg-overlay);
}

.queue-layout {
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 16px;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  width: 100%;
}

.queue-sidebar {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow-y: auto;
  overflow-x: hidden;
}

.sidebar-title {
  padding: 12px 16px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--border);
  background: var(--bg-overlay);
}

.queue-item {
  padding: 10px 16px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-subtle);
  transition: background 0.1s;
}

.queue-item:hover {
  background: var(--bg-overlay);
}

.queue-item.active {
  background: var(--accent-dim);
  color: var(--accent);
}

.queue-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.queue-item.active .queue-name {
  color: var(--accent);
}

.queue-counts {
  display: flex;
  gap: 4px;
}

.count {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
}

.count-warning { background: rgba(234,179,8,0.15); color: #ca8a04; }
.count-info    { background: rgba(59,130,246,0.15); color: #3b82f6; }
.count-error   { background: rgba(239,68,68,0.15);  color: #ef4444; }

.messages-panel {
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
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-overlay);
  flex-wrap: wrap;
}

.messages-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  flex: 1;
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
  transition: all 0.1s;
}

.filter-tab:hover {
  background: var(--bg-overlay);
}

.filter-tab.active {
  background: var(--accent-dim);
  color: var(--accent);
  border-color: var(--accent);
}

.msg-table {
  flex: 1;
  overflow-y: auto;
}

.msg-thead,
.msg-row {
  display: grid;
  grid-template-columns: 110px 1fr 80px 100px 50px 150px 80px;
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

.msg-row:last-child {
  border-bottom: none;
}

.msg-id {
  font-family: monospace;
  font-size: 12px;
  color: var(--text-secondary);
}

.badge {
  display: inline-block;
  padding: 2px 7px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.badge-warning { background: rgba(234,179,8,0.15);  color: #ca8a04; }
.badge-info    { background: rgba(59,130,246,0.15);  color: #3b82f6; }
.badge-success { background: rgba(34,197,94,0.15);   color: #16a34a; }
.badge-error   { background: rgba(239,68,68,0.15);   color: #ef4444; }
.badge-neutral { background: var(--bg-overlay);      color: var(--text-secondary); border: 1px solid var(--border); }

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

.action-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-primary);
}

.action-danger:hover {
  background: rgba(239,68,68,0.1);
  color: #ef4444;
  border-color: rgba(239,68,68,0.3);
}

.empty-state {
  padding: 24px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 13px;
  grid-column: 1 / -1;
}

.empty-state-full {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-tertiary);
}

.empty-state-full p {
  font-size: 14px;
  margin: 0;
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
  width: 500px;
  max-width: 95vw;
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

.form-group textarea {
  resize: vertical;
  font-family: monospace;
}

.form-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.error-text {
  color: #ef4444;
  font-size: 12px;
  margin-top: 4px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 20px;
}
</style>
