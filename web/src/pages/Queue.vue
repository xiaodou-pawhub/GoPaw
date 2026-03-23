<template>
  <div class="queue-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">任务队列</h1>
        <p class="page-desc">管理异步任务执行</p>
      </div>
      <div class="header-right">
        <button class="btn-secondary" @click="loadData">
          <RefreshCwIcon :size="16" /> 刷新
        </button>
        <button class="btn-primary" @click="showEnqueueDialog">
          <PlusIcon :size="16" /> 新建任务
        </button>
      </div>
    </div>

    <!-- 标签页切换 -->
    <div class="tab-nav">
      <button class="tab-btn" :class="{ active: activeTab === 'tasks' }" @click="activeTab = 'tasks'">
        任务队列
      </button>
      <button class="tab-btn" :class="{ active: activeTab === 'dead-letters' }" @click="activeTab = 'dead-letters'">
        死信队列
        <span v-if="deadLetters.length > 0" class="badge-count">{{ deadLetters.length }}</span>
      </button>
    </div>

    <!-- 任务队列 -->
    <template v-if="activeTab === 'tasks'">
    <!-- 统计卡片 -->
    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-icon pending">
          <ClockIcon :size="20" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.pending }}</div>
          <div class="stat-label">等待中</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon delayed">
          <TimerIcon :size="20" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.delayed }}</div>
          <div class="stat-label">延迟任务</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon workers">
          <CpuIcon :size="20" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.busy_workers }}/{{ stats.workers }}</div>
          <div class="stat-label">Worker</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon rate">
          <ActivityIcon :size="20" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ completionRate }}%</div>
          <div class="stat-label">完成率</div>
        </div>
      </div>
    </div>

    <!-- 任务列表 -->
    <div class="tasks-panel">
      <div class="tasks-header">
        <div class="filter-tabs">
          <button
            v-for="f in statusFilters"
            :key="f.value"
            class="filter-tab"
            :class="{ active: statusFilter === f.value }"
            @click="statusFilter = f.value as any; loadTasks()"
          >
            {{ f.label }}
            <span v-if="f.count" class="tab-count">{{ f.count }}</span>
          </button>
        </div>
        <div class="search-box">
          <SearchIcon :size="16" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索任务..."
            @input="filterTasks"
          />
        </div>
      </div>

      <div class="tasks-table">
        <div class="table-header">
          <span class="col-id">ID</span>
          <span class="col-type">类型</span>
          <span class="col-priority">优先级</span>
          <span class="col-status">状态</span>
          <span class="col-retry">重试</span>
          <span class="col-worker">Worker</span>
          <span class="col-time">创建时间</span>
          <span class="col-actions">操作</span>
        </div>

        <div v-if="loading" class="loading-state">
          <LoaderIcon :size="24" class="spin" />
          <span>加载中...</span>
        </div>

        <div v-else-if="filteredTasks.length === 0" class="empty-state">
          <InboxIcon :size="40" />
          <p>暂无任务</p>
        </div>

        <div
          v-else
          v-for="task in filteredTasks"
          :key="task.id"
          class="table-row"
          :class="{ clickable: true }"
          @click="showTaskDetail(task)"
        >
          <span class="col-id">
            <span class="task-id" :title="task.id">{{ task.id.substring(0, 8) }}</span>
          </span>
          <span class="col-type">
            <span class="type-badge">{{ getTypeLabel(task.type) }}</span>
          </span>
          <span class="col-priority">
            <span class="priority-badge" :class="getPriorityClass(task.priority)">
              {{ getPriorityLabel(task.priority) }}
            </span>
          </span>
          <span class="col-status">
            <span class="status-badge" :class="getStatusClass(task.status)">
              {{ getStatusLabel(task.status) }}
            </span>
          </span>
          <span class="col-retry">
            <span :class="{ 'text-warning': task.retry_count > 0 }">
              {{ task.retry_count }}/{{ task.max_retries }}
            </span>
          </span>
          <span class="col-worker">
            <span v-if="task.worker_id" class="worker-id">{{ task.worker_id }}</span>
            <span v-else class="text-muted">-</span>
          </span>
          <span class="col-time">
            <span class="time-text">{{ formatTime(task.created_at) }}</span>
            <span v-if="task.scheduled_at" class="scheduled-badge">
              定时 {{ formatTime(task.scheduled_at) }}
            </span>
          </span>
          <span class="col-actions" @click.stop>
            <button
              v-if="task.status === 'failed' || task.status === 'retry'"
              class="action-btn"
              title="重试"
              @click="retryTask(task)"
            >
              <RefreshCwIcon :size="14" />
            </button>
            <button
              v-if="task.status === 'pending' || task.status === 'queued'"
              class="action-btn action-danger"
              title="取消"
              @click="cancelTask(task)"
            >
              <XIcon :size="14" />
            </button>
            <button
              class="action-btn"
              title="详情"
              @click="showTaskDetail(task)"
            >
              <EyeIcon :size="14" />
            </button>
          </span>
        </div>
      </div>
    </div>
    </template>

    <!-- 死信队列 -->
    <template v-if="activeTab === 'dead-letters'">
      <div class="dead-letter-panel">
        <div class="panel-header">
          <h3>死信队列</h3>
          <p class="panel-desc">超过最大重试次数的失败任务</p>
          <button v-if="deadLetters.length > 0" class="btn-danger-outline" @click="purgeDeadLetters">
            <Trash2Icon :size="14" /> 清空全部
          </button>
        </div>

        <div v-if="deadLettersLoading" class="loading-state">
          <LoaderIcon :size="24" class="spin" />
          <span>加载中...</span>
        </div>

        <div v-else-if="deadLetters.length === 0" class="empty-state">
          <CheckCircleIcon :size="40" />
          <p>死信队列为空</p>
        </div>

        <div v-else class="dead-letter-list">
          <div v-for="dl in deadLetters" :key="dl.id" class="dead-letter-item">
            <div class="dl-header">
              <span class="dl-id">{{ dl.id.substring(0, 8) }}</span>
              <span class="type-badge">{{ getTypeLabel(dl.type) }}</span>
              <span class="dl-time">{{ formatTime(dl.moved_at) }}</span>
            </div>
            <div class="dl-error">
              <AlertTriangleIcon :size="14" />
              <span>{{ dl.error || dl.move_reason }}</span>
            </div>
            <div class="dl-meta">
              <span>重试 {{ dl.retry_count }}/{{ dl.max_retries }} 次</span>
              <span>移入时间: {{ formatDateTime(dl.moved_at) }}</span>
            </div>
            <div class="dl-actions">
              <button class="btn-sm btn-primary" @click="retryDeadLetter(dl)">
                <RefreshCwIcon :size="12" /> 重试
              </button>
              <button class="btn-sm btn-danger-outline" @click="deleteDeadLetter(dl)">
                <Trash2Icon :size="12" /> 删除
              </button>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 新建任务弹窗 -->
    <div v-if="enqueueDialog.show" class="modal-overlay" @click.self="enqueueDialog.show = false">
      <div class="modal-card">
        <h2 class="modal-title">新建任务</h2>

        <div class="form-group">
          <label>任务类型 *</label>
          <select v-model="enqueueDialog.type">
            <option value="">请选择</option>
            <option value="flow_execute">流程执行</option>
            <option value="webhook_callback">Webhook 回调</option>
            <option value="subflow_execute">子流程执行</option>
          </select>
        </div>

        <!-- 流程执行配置 -->
        <template v-if="enqueueDialog.type === 'flow_execute'">
          <div class="form-group">
            <label>流程 ID *</label>
            <input v-model="enqueueDialog.flowId" type="text" placeholder="输入流程 ID" />
          </div>
          <div class="form-group">
            <label>输入内容</label>
            <textarea v-model="enqueueDialog.input" rows="3" placeholder="输入内容" />
          </div>
        </template>

        <!-- Webhook 回调配置 -->
        <template v-if="enqueueDialog.type === 'webhook_callback'">
          <div class="form-group">
            <label>Webhook ID *</label>
            <input v-model="enqueueDialog.webhookId" type="text" placeholder="Webhook ID" />
          </div>
          <div class="form-group">
            <label>回调数据 (JSON)</label>
            <textarea v-model="enqueueDialog.payloadJson" rows="3" placeholder="{}" />
          </div>
        </template>

        <div class="form-row">
          <div class="form-group">
            <label>优先级</label>
            <select v-model.number="enqueueDialog.priority">
              <option :value="1">低</option>
              <option :value="5">普通</option>
              <option :value="10">高</option>
              <option :value="20">紧急</option>
            </select>
          </div>
          <div class="form-group">
            <label>延迟（秒）</label>
            <input v-model.number="enqueueDialog.delay" type="number" min="0" placeholder="0" />
          </div>
          <div class="form-group">
            <label>最大重试</label>
            <input v-model.number="enqueueDialog.maxRetries" type="number" min="0" max="10" placeholder="3" />
          </div>
        </div>

        <div v-if="enqueueDialog.error" class="error-text">{{ enqueueDialog.error }}</div>

        <div class="modal-actions">
          <button class="btn-ghost" @click="enqueueDialog.show = false">取消</button>
          <button class="btn-primary" @click="confirmEnqueue" :disabled="enqueueDialog.loading">
            <LoaderIcon v-if="enqueueDialog.loading" :size="14" class="spin" />
            创建任务
          </button>
        </div>
      </div>
    </div>

    <!-- 任务详情弹窗 -->
    <div v-if="detailDialog.show" class="modal-overlay" @click.self="detailDialog.show = false">
      <div class="modal-card modal-lg">
        <h2 class="modal-title">任务详情</h2>

        <div class="detail-grid">
          <div class="detail-item">
            <span class="detail-label">任务 ID</span>
            <span class="detail-value mono">{{ detailDialog.task?.id }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">类型</span>
            <span class="detail-value">{{ getTypeLabel(detailDialog.task?.type || '') }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">状态</span>
            <span class="status-badge" :class="getStatusClass(detailDialog.task?.status || '')">
              {{ getStatusLabel(detailDialog.task?.status || '') }}
            </span>
          </div>
          <div class="detail-item">
            <span class="detail-label">优先级</span>
            <span class="priority-badge" :class="getPriorityClass(detailDialog.task?.priority || 5)">
              {{ getPriorityLabel(detailDialog.task?.priority || 5) }}
            </span>
          </div>
          <div class="detail-item">
            <span class="detail-label">创建时间</span>
            <span class="detail-value">{{ formatDateTime(detailDialog.task?.created_at) }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">开始时间</span>
            <span class="detail-value">{{ detailDialog.task?.started_at ? formatDateTime(detailDialog.task?.started_at) : '-' }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">完成时间</span>
            <span class="detail-value">{{ detailDialog.task?.completed_at ? formatDateTime(detailDialog.task?.completed_at) : '-' }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">Worker</span>
            <span class="detail-value mono">{{ detailDialog.task?.worker_id || '-' }}</span>
          </div>
        </div>

        <div class="detail-section" v-if="detailDialog.task?.payload">
          <h4>负载</h4>
          <pre class="code-block">{{ JSON.stringify(detailDialog.task?.payload, null, 2) }}</pre>
        </div>

        <div class="detail-section" v-if="detailDialog.task?.result">
          <h4>结果</h4>
          <pre class="code-block">{{ JSON.stringify(detailDialog.task?.result, null, 2) }}</pre>
        </div>

        <div class="detail-section error-section" v-if="detailDialog.task?.error">
          <h4>错误信息</h4>
          <pre class="code-block error">{{ detailDialog.task?.error }}</pre>
        </div>

        <div class="modal-actions">
          <button
            v-if="detailDialog.task?.status === 'failed'"
            class="btn-secondary"
            @click="retryTask(detailDialog.task!); detailDialog.show = false"
          >
            <RefreshCwIcon :size="14" /> 重试
          </button>
          <button
            v-if="detailDialog.task?.status === 'pending' || detailDialog.task?.status === 'queued'"
            class="btn-danger"
            @click="cancelTask(detailDialog.task!); detailDialog.show = false"
          >
            <XIcon :size="14" /> 取消任务
          </button>
          <button class="btn-ghost" @click="detailDialog.show = false">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import {
  PlusIcon, RefreshCwIcon, ClockIcon, TimerIcon, CpuIcon, ActivityIcon,
  SearchIcon, InboxIcon, LoaderIcon, EyeIcon, XIcon, Trash2Icon, CheckCircleIcon, AlertTriangleIcon
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { taskQueueApi, type Task, type TaskStatus, type QueueStats, TaskTypes, PriorityLabels, StatusLabels } from '@/api/queue'

const activeTab = ref<'tasks' | 'dead-letters'>('tasks')
const stats = ref<QueueStats>({ pending: 0, delayed: 0, workers: 0, busy_workers: 0 })
const tasks = ref<Task[]>([])
const loading = ref(false)
const statusFilter = ref<TaskStatus | ''>('')
const searchQuery = ref('')
let refreshInterval: number | null = null

// 死信队列
interface DeadLetterTask extends Task {
  moved_at: string
  move_reason: string
}
const deadLetters = ref<DeadLetterTask[]>([])
const deadLettersLoading = ref(false)

const statusFilters = computed(() => [
  { value: '', label: '全部', count: null },
  { value: 'pending', label: '等待', count: null },
  { value: 'queued', label: '已入队', count: null },
  { value: 'running', label: '执行中', count: null },
  { value: 'completed', label: '已完成', count: null },
  { value: 'failed', label: '失败', count: null },
])

const filteredTasks = computed(() => {
  if (!searchQuery.value) return tasks.value
  const query = searchQuery.value.toLowerCase()
  return tasks.value.filter(t =>
    t.id.toLowerCase().includes(query) ||
    t.type.toLowerCase().includes(query) ||
    t.worker_id?.toLowerCase().includes(query)
  )
})

const completionRate = computed(() => {
  // 这里可以计算完成率，暂时返回占位
  return 0
})

const enqueueDialog = reactive({
  show: false,
  loading: false,
  type: '',
  flowId: '',
  input: '',
  webhookId: '',
  payloadJson: '{}',
  priority: 5 as number,
  delay: 0,
  maxRetries: 3,
  error: '',
})

const detailDialog = reactive({
  show: false,
  task: null as Task | null,
})

function getTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    [TaskTypes.FLOW_EXECUTE]: '流程执行',
    [TaskTypes.WEBHOOK_CALLBACK]: 'Webhook',
    [TaskTypes.SUBFLOW_EXECUTE]: '子流程',
    [TaskTypes.RETRY_EXECUTION]: '重试',
  }
  return labels[type] || type
}

function getPriorityLabel(priority: number): string {
  return PriorityLabels[priority]?.label || String(priority)
}

function getPriorityClass(priority: number): string {
  return PriorityLabels[priority]?.color || 'neutral'
}

function getStatusLabel(status: string): string {
  return StatusLabels[status as TaskStatus]?.label || status
}

function getStatusClass(status: string): string {
  return StatusLabels[status as TaskStatus]?.color || 'neutral'
}

function formatTime(dateStr?: string): string {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - d.getTime()

  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
  return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
}

function formatDateTime(dateStr?: string): string {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}

async function loadData() {
  await Promise.all([loadStats(), loadTasks(), loadDeadLetters()])
}

async function loadStats() {
  try {
    stats.value = await taskQueueApi.getStats()
  } catch (e) {
    console.error('Failed to load stats:', e)
  }
}

async function loadTasks() {
  loading.value = true
  try {
    tasks.value = await taskQueueApi.listTasks(statusFilter.value || undefined, 100)
  } catch (e) {
    console.error('Failed to load tasks:', e)
    toast.error('加载任务失败')
  } finally {
    loading.value = false
  }
}

async function loadDeadLetters() {
  deadLettersLoading.value = true
  try {
    const res = await fetch('/api/flows/queue/dead-letters')
    if (res.ok) {
      deadLetters.value = await res.json()
    }
  } catch (e) {
    console.error('Failed to load dead letters:', e)
  } finally {
    deadLettersLoading.value = false
  }
}

async function retryDeadLetter(dl: DeadLetterTask) {
  try {
    const res = await fetch(`/api/flows/queue/dead-letters/${dl.id}/retry`, { method: 'POST' })
    if (res.ok) {
      toast.success('任务已重新入队')
      loadDeadLetters()
      loadStats()
    } else {
      toast.error('重试失败')
    }
  } catch {
    toast.error('重试失败')
  }
}

async function deleteDeadLetter(dl: DeadLetterTask) {
  if (!confirm('确定要删除这个死信任务吗？')) return
  try {
    const res = await fetch(`/api/flows/queue/dead-letters/${dl.id}`, { method: 'DELETE' })
    if (res.ok) {
      toast.success('已删除')
      loadDeadLetters()
    } else {
      toast.error('删除失败')
    }
  } catch {
    toast.error('删除失败')
  }
}

async function purgeDeadLetters() {
  if (!confirm('确定要清空所有死信任务吗？此操作不可恢复。')) return
  try {
    const res = await fetch('/api/flows/queue/dead-letters', { method: 'DELETE' })
    if (res.ok) {
      toast.success('已清空死信队列')
      loadDeadLetters()
    } else {
      toast.error('清空失败')
    }
  } catch {
    toast.error('清空失败')
  }
}

function filterTasks() {
  // 过滤由 computed 处理
}

function showEnqueueDialog() {
  enqueueDialog.type = ''
  enqueueDialog.flowId = ''
  enqueueDialog.input = ''
  enqueueDialog.webhookId = ''
  enqueueDialog.payloadJson = '{}'
  enqueueDialog.priority = 5
  enqueueDialog.delay = 0
  enqueueDialog.maxRetries = 3
  enqueueDialog.error = ''
  enqueueDialog.show = true
}

async function confirmEnqueue() {
  enqueueDialog.error = ''

  if (!enqueueDialog.type) {
    enqueueDialog.error = '请选择任务类型'
    return
  }

  let payload: Record<string, any> = {}

  if (enqueueDialog.type === 'flow_execute') {
    if (!enqueueDialog.flowId) {
      enqueueDialog.error = '请输入流程 ID'
      return
    }
    payload = {
      flow_id: enqueueDialog.flowId,
      input: enqueueDialog.input,
    }
  } else if (enqueueDialog.type === 'webhook_callback') {
    if (!enqueueDialog.webhookId) {
      enqueueDialog.error = '请输入 Webhook ID'
      return
    }
    try {
      payload = JSON.parse(enqueueDialog.payloadJson || '{}')
      payload.webhook_id = enqueueDialog.webhookId
    } catch {
      enqueueDialog.error = '无效的 JSON 格式'
      return
    }
  }

  enqueueDialog.loading = true
  try {
    await taskQueueApi.enqueue({
      type: enqueueDialog.type,
      payload,
      priority: enqueueDialog.priority,
      delay: enqueueDialog.delay || undefined,
      max_retries: enqueueDialog.maxRetries,
    })
    toast.success('任务已创建')
    enqueueDialog.show = false
    loadData()
  } catch (e: any) {
    enqueueDialog.error = e.response?.data?.error || '创建失败'
  } finally {
    enqueueDialog.loading = false
  }
}

function showTaskDetail(task: Task) {
  detailDialog.task = task
  detailDialog.show = true
}

async function retryTask(task: Task) {
  try {
    // 重新入队
    await taskQueueApi.enqueue({
      type: task.type,
      payload: task.payload,
      priority: task.priority,
      max_retries: task.max_retries,
    })
    toast.success('任务已重新入队')
    loadData()
  } catch {
    toast.error('重试失败')
  }
}

async function cancelTask(task: Task) {
  if (!confirm('确定要取消这个任务吗？')) return

  try {
    await taskQueueApi.cancel(task.id)
    toast.success('任务已取消')
    loadData()
  } catch {
    toast.error('取消失败')
  }
}

onMounted(() => {
  loadData()
  // 每 5 秒刷新一次
  refreshInterval = window.setInterval(loadData, 5000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
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
  background: var(--bg-app);
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 24px;
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.page-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0;
}

.header-right {
  display: flex;
  gap: 8px;
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

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--bg-elevated);
  color: var(--text-primary);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
}

.btn-secondary:hover {
  background: var(--bg-overlay);
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

.btn-danger {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
}

.btn-danger:hover {
  background: rgba(239, 68, 68, 0.2);
}

/* 统计卡片 */
.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 14px;
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-icon.pending { background: rgba(234, 179, 8, 0.15); color: #ca8a04; }
.stat-icon.delayed { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.stat-icon.workers { background: rgba(139, 92, 246, 0.15); color: #8b5cf6; }
.stat-icon.rate { background: rgba(34, 197, 94, 0.15); color: #16a34a; }

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
}

.stat-label {
  font-size: 12px;
  color: var(--text-secondary);
}

/* 任务面板 */
.tasks-panel {
  flex: 1;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.tasks-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-overlay);
}

.filter-tabs {
  display: flex;
  gap: 4px;
}

.filter-tab {
  padding: 6px 12px;
  border-radius: 6px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--text-secondary);
  font-size: 13px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
}

.filter-tab:hover {
  background: var(--bg-overlay);
}

.filter-tab.active {
  background: var(--accent-dim);
  color: var(--accent);
  border-color: var(--accent);
}

.tab-count {
  background: var(--bg-overlay);
  padding: 1px 6px;
  border-radius: 10px;
  font-size: 11px;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
}

.search-box input {
  border: none;
  background: transparent;
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  width: 150px;
}

/* 任务表格 */
.tasks-table {
  flex: 1;
  overflow-y: auto;
}

.table-header, .table-row {
  display: grid;
  grid-template-columns: 80px 120px 80px 100px 70px 100px 150px 100px;
  padding: 10px 16px;
  align-items: center;
  gap: 8px;
}

.table-header {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--bg-overlay);
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
}

.table-row {
  font-size: 13px;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-subtle);
}

.table-row:hover {
  background: var(--bg-overlay);
}

.table-row.clickable {
  cursor: pointer;
}

.task-id {
  font-family: monospace;
  font-size: 12px;
  color: var(--text-secondary);
}

.type-badge {
  padding: 2px 8px;
  background: var(--bg-overlay);
  border-radius: 4px;
  font-size: 12px;
}

.priority-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.priority-badge.neutral { background: var(--bg-overlay); color: var(--text-secondary); }
.priority-badge.info { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.priority-badge.warning { background: rgba(234, 179, 8, 0.15); color: #ca8a04; }
.priority-badge.error { background: rgba(239, 68, 68, 0.15); color: #ef4444; }

.status-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.status-badge.neutral { background: var(--bg-overlay); color: var(--text-secondary); }
.status-badge.info { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.status-badge.primary { background: rgba(139, 92, 246, 0.15); color: #8b5cf6; }
.status-badge.success { background: rgba(34, 197, 94, 0.15); color: #16a34a; }
.status-badge.warning { background: rgba(234, 179, 8, 0.15); color: #ca8a04; }
.status-badge.error { background: rgba(239, 68, 68, 0.15); color: #ef4444; }

.worker-id {
  font-family: monospace;
  font-size: 11px;
  color: var(--text-secondary);
}

.text-muted { color: var(--text-tertiary); }
.text-warning { color: #ca8a04; }

.time-text {
  font-size: 12px;
}

.scheduled-badge {
  display: block;
  font-size: 10px;
  color: var(--text-tertiary);
  margin-top: 2px;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  color: var(--text-secondary);
  margin-right: 4px;
}

.action-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-primary);
}

.action-danger:hover {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border-color: rgba(239, 68, 68, 0.3);
}

.loading-state, .empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 60px;
  color: var(--text-tertiary);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 24px;
  width: 480px;
  max-width: 95vw;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-lg {
  width: 640px;
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
  margin-top: 8px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 20px;
}

/* 详情 */
.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  margin-bottom: 20px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-label {
  font-size: 11px;
  color: var(--text-tertiary);
  text-transform: uppercase;
}

.detail-value {
  font-size: 13px;
  color: var(--text-primary);
}

.detail-value.mono {
  font-family: monospace;
  font-size: 12px;
}

.detail-section {
  margin-bottom: 16px;
}

.detail-section h4 {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  margin: 0 0 8px;
}

.code-block {
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 12px;
  font-family: monospace;
  font-size: 12px;
  overflow-x: auto;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.code-block.error {
  background: rgba(239, 68, 68, 0.1);
  border-color: rgba(239, 68, 68, 0.3);
  color: #ef4444;
}

.error-section h4 {
  color: #ef4444;
}

/* 响应式 */
@media (max-width: 768px) {
  .stats-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .table-header, .table-row {
    grid-template-columns: 60px 100px 60px 80px 60px 80px;
  }

  .col-worker, .col-time {
    display: none;
  }
}

/* 标签页导航 */
.tab-nav {
  display: flex;
  gap: 4px;
  margin-bottom: 20px;
}

.tab-btn {
  padding: 8px 16px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--bg-elevated);
  color: var(--text-secondary);
  font-size: 13px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
}

.tab-btn:hover {
  background: var(--bg-overlay);
}

.tab-btn.active {
  background: var(--accent-dim);
  color: var(--accent);
  border-color: var(--accent);
}

.badge-count {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
  padding: 1px 6px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
}

/* 死信队列面板 */
.dead-letter-panel {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 20px;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}

.panel-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.panel-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0;
  flex: 1;
}

.btn-danger-outline {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: transparent;
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
}

.btn-danger-outline:hover {
  background: rgba(239, 68, 68, 0.1);
}

.dead-letter-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dead-letter-item {
  padding: 16px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 8px;
}

.dl-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.dl-id {
  font-family: monospace;
  font-size: 12px;
  color: var(--text-secondary);
}

.dl-time {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-left: auto;
}

.dl-error {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px;
  background: rgba(239, 68, 68, 0.1);
  border-radius: 6px;
  color: #ef4444;
  font-size: 13px;
  margin-bottom: 10px;
}

.dl-meta {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 12px;
}

.dl-actions {
  display: flex;
  gap: 8px;
}

.btn-sm {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
}

.btn-sm.btn-primary {
  background: var(--accent);
  color: #fff;
  border: none;
}

.btn-sm.btn-danger-outline {
  background: transparent;
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.btn-sm.btn-danger-outline:hover {
  background: rgba(239, 68, 68, 0.1);
}
</style>