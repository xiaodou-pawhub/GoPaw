<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">执行历史</h1>
        <p class="page-desc">查看流程执行记录和详细日志</p>
      </div>
      <div class="header-right">
        <select v-model="statusFilter" class="filter-select" @change="loadExecutions">
          <option value="">全部状态</option>
          <option value="running">执行中</option>
          <option value="waiting">等待中</option>
          <option value="completed">已完成</option>
          <option value="failed">失败</option>
        </select>
        <button class="btn-secondary" @click="loadExecutions">
          <RefreshIcon :size="16" /> 刷新
        </button>
      </div>
    </div>

    <div class="content-layout">
      <!-- 执行列表 -->
      <div class="execution-list">
        <div v-if="loading" class="loading-state">
          <LoaderIcon :size="24" class="spin" />
          <span>加载中...</span>
        </div>
        <div v-else-if="!executions || executions.length === 0" class="empty-state">
          <HistoryIcon :size="48" />
          <p>暂无执行记录</p>
        </div>
        <div v-else class="execution-items">
          <div
            v-for="exec in executions"
            :key="exec.id"
            class="execution-item"
            :class="{ active: selectedExecution?.id === exec.id }"
            @click="selectExecution(exec)"
          >
            <div class="item-header">
              <span class="item-status" :class="exec.status">
                {{ getStatusLabel(exec.status) }}
              </span>
              <span class="item-time">{{ formatTime(exec.started_at) }}</span>
            </div>
            <div class="item-title">{{ exec.flow_name || '流程' }}</div>
            <div class="item-meta">
              <span>{{ exec.id.substring(0, 8) }}...</span>
              <span v-if="exec.completed_at">耗时 {{ getDuration(exec.started_at, exec.completed_at) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 执行详情 -->
      <div class="execution-detail">
        <div v-if="!selectedExecution" class="no-selection">
          <FileSearchIcon :size="48" />
          <p>选择一条执行记录查看详情</p>
        </div>
        <div v-else class="detail-content">
          <div class="detail-header">
            <div class="detail-title">
              <h2>{{ selectedExecution.flow_name || '流程执行' }}</h2>
              <span class="detail-status" :class="selectedExecution.status">
                {{ getStatusLabel(selectedExecution.status) }}
              </span>
            </div>
            <div class="detail-actions">
              <button
                v-if="selectedExecution.status === 'waiting'"
                class="btn-primary"
                @click="goToPendingTasks"
              >
                去处理
              </button>
              <button class="btn-secondary" @click="retryExecution" :disabled="selectedExecution.status === 'running'">
                <RotateCcwIcon :size="14" /> 重新执行
              </button>
            </div>
          </div>

          <!-- 基本信息 -->
          <div class="detail-section">
            <h3>基本信息</h3>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">执行 ID</span>
                <span class="info-value">{{ selectedExecution.id }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">流程 ID</span>
                <span class="info-value">{{ selectedExecution.flow_id }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">开始时间</span>
                <span class="info-value">{{ formatDateTime(selectedExecution.started_at) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">结束时间</span>
                <span class="info-value">{{ selectedExecution.completed_at ? formatDateTime(selectedExecution.completed_at) : '-' }}</span>
              </div>
            </div>
          </div>

          <!-- 输入输出 -->
          <div class="detail-section">
            <h3>输入</h3>
            <div class="code-block">
              <pre>{{ selectedExecution.input || '(无输入)' }}</pre>
            </div>
          </div>

          <div class="detail-section" v-if="selectedExecution.output">
            <h3>输出</h3>
            <div class="code-block">
              <pre>{{ selectedExecution.output }}</pre>
            </div>
          </div>

          <div class="detail-section" v-if="selectedExecution.error">
            <h3>错误信息</h3>
            <div class="code-block error">
              <pre>{{ selectedExecution.error }}</pre>
            </div>
          </div>

          <!-- 执行步骤 -->
          <div class="detail-section">
            <h3>执行步骤</h3>
            <div v-if="!selectedExecution.history || selectedExecution.history.length === 0" class="no-steps">
              暂无执行记录
            </div>
            <div v-else class="step-list">
              <div
                v-for="(step, index) in selectedExecution.history"
                :key="index"
                class="step-item"
                :class="step.status"
              >
                <div class="step-indicator">
                  <span class="step-index">{{ index + 1 }}</span>
                  <div class="step-line" v-if="index < selectedExecution.history.length - 1" />
                </div>
                <div class="step-content">
                  <div class="step-header">
                    <span class="step-name">{{ step.node_id }}</span>
                    <span class="step-type">{{ step.node_type }}</span>
                    <span class="step-status" :class="step.status">
                      {{ getStatusLabel(step.status) }}
                    </span>
                    <span class="step-time" v-if="step.started_at">
                      {{ getDuration(step.started_at, step.ended_at) }}
                    </span>
                  </div>
                  <div class="step-output" v-if="step.output">
                    <pre>{{ JSON.stringify(step.output, null, 2) }}</pre>
                  </div>
                  <div class="step-error" v-if="step.error">
                    <span>{{ step.error }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  RefreshIcon, LoaderIcon, HistoryIcon, FileSearchIcon, RotateCcwIcon
} from 'lucide-vue-next'

interface ExecutionStep {
  node_id: string
  node_type: string
  status: string
  started_at?: string
  ended_at?: string
  output?: any
  error?: string
}

interface Execution {
  id: string
  flow_id: string
  flow_name?: string
  status: string
  input: string
  output?: string
  error?: string
  started_at: string
  completed_at?: string
  history?: ExecutionStep[]
  context?: Record<string, any>
}

const router = useRouter()
const loading = ref(true)
const executions = ref<Execution[]>([])
const selectedExecution = ref<Execution | null>(null)
const statusFilter = ref('')

let refreshInterval: number | null = null

async function loadExecutions() {
  loading.value = true
  try {
    const url = statusFilter.value
      ? `/api/flows/executions?status=${statusFilter.value}`
      : '/api/flows/executions'
    const res = await fetch(url)
    if (res.ok) {
      const data = await res.json()
      executions.value = data.executions || []
    }
  } catch (e) {
    console.error('Failed to load executions:', e)
  } finally {
    loading.value = false
  }
}

function selectExecution(exec: Execution) {
  selectedExecution.value = exec
}

function getStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    running: '执行中',
    waiting: '等待中',
    completed: '已完成',
    failed: '失败',
    pending: '等待'
  }
  return labels[status] || status
}

function formatTime(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - d.getTime()

  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`

  return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
}

function formatDateTime(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}

function getDuration(start: string, end?: string): string {
  if (!start) return '-'
  const startTime = new Date(start).getTime()
  const endTime = end ? new Date(end).getTime() : Date.now()
  const diff = endTime - startTime

  if (diff < 1000) return `${diff}ms`
  if (diff < 60000) return `${(diff / 1000).toFixed(1)}s`
  return `${Math.floor(diff / 60000)}m ${Math.floor((diff % 60000) / 1000)}s`
}

function goToPendingTasks() {
  router.push('/pending-tasks')
}

async function retryExecution() {
  if (!selectedExecution.value) return

  try {
    const res = await fetch(`/api/flows/${selectedExecution.value.flow_id}/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ input: selectedExecution.value.input })
    })
    if (res.ok) {
      await loadExecutions()
    }
  } catch (e) {
    console.error('Failed to retry:', e)
  }
}

onMounted(() => {
  loadExecutions()
  refreshInterval = window.setInterval(loadExecutions, 30000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<style scoped>
.page-container {
  padding: 24px;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.page-desc {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.header-right {
  display: flex;
  gap: 8px;
}

.filter-select {
  padding: 8px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
}

.btn-secondary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
  cursor: pointer;
}

.btn-secondary:hover {
  background: var(--bg-overlay);
}

.btn-secondary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 14px;
  cursor: pointer;
}

.content-layout {
  flex: 1;
  display: flex;
  gap: 24px;
  min-height: 0;
}

.execution-list {
  width: 320px;
  flex-shrink: 0;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.execution-items {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.execution-item {
  padding: 12px;
  border-radius: 8px;
  cursor: pointer;
  margin-bottom: 4px;
  transition: background 0.15s;
}

.execution-item:hover {
  background: var(--bg-app);
}

.execution-item.active {
  background: var(--accent-dim);
}

.item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.item-status {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
}

.item-status.running { background: #dbeafe; color: #2563eb; }
.item-status.waiting { background: #fef3c7; color: #d97706; }
.item-status.completed { background: #dcfce7; color: #16a34a; }
.item-status.failed { background: #fee2e2; color: #dc2626; }

.item-time {
  font-size: 11px;
  color: var(--text-muted);
}

.item-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.item-meta {
  display: flex;
  gap: 12px;
  font-size: 11px;
  color: var(--text-muted);
}

.execution-detail {
  flex: 1;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.no-selection {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  gap: 12px;
}

.detail-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
}

.detail-title h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 8px 0;
}

.detail-status {
  font-size: 12px;
  padding: 4px 8px;
  border-radius: 4px;
}

.detail-actions {
  display: flex;
  gap: 8px;
}

.detail-section {
  margin-bottom: 20px;
}

.detail-section h3 {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 12px 0;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-label {
  font-size: 12px;
  color: var(--text-muted);
}

.info-value {
  font-size: 13px;
  color: var(--text-primary);
  font-family: monospace;
}

.code-block {
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px;
  overflow-x: auto;
}

.code-block pre {
  margin: 0;
  font-size: 12px;
  font-family: monospace;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-break: break-all;
}

.code-block.error {
  border-color: #fecaca;
  background: #fef2f2;
}

.code-block.error pre {
  color: #dc2626;
}

.step-list {
  display: flex;
  flex-direction: column;
}

.step-item {
  display: flex;
  gap: 12px;
  padding-bottom: 16px;
}

.step-item:last-child {
  padding-bottom: 0;
}

.step-indicator {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.step-index {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--bg-app);
  border: 2px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
}

.step-item.completed .step-index {
  background: #dcfce7;
  border-color: #22c55e;
  color: #16a34a;
}

.step-item.failed .step-index {
  background: #fee2e2;
  border-color: #ef4444;
  color: #dc2626;
}

.step-line {
  flex: 1;
  width: 2px;
  background: var(--border);
  margin-top: 4px;
}

.step-content {
  flex: 1;
  min-width: 0;
}

.step-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.step-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.step-type {
  font-size: 11px;
  padding: 2px 6px;
  background: var(--bg-app);
  border-radius: 4px;
  color: var(--text-secondary);
}

.step-status {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
}

.step-time {
  font-size: 11px;
  color: var(--text-muted);
  margin-left: auto;
}

.step-output {
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 8px;
  margin-top: 8px;
}

.step-output pre {
  margin: 0;
  font-size: 11px;
  font-family: monospace;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-break: break-all;
}

.step-error {
  font-size: 12px;
  color: #dc2626;
  margin-top: 8px;
}

.no-steps {
  color: var(--text-muted);
  font-size: 14px;
}

.loading-state, .empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  gap: 12px;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>