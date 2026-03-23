<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">待办任务</h1>
        <p class="page-desc">处理流程中等待人工介入的任务</p>
      </div>
      <div class="header-right">
        <button class="btn-secondary" @click="loadTasks">
          <RefreshCwIcon :size="16" /> 刷新
        </button>
      </div>
    </div>

    <!-- 任务列表 -->
    <div class="task-list">
      <div v-if="loading" class="loading-state">
        <LoaderIcon :size="24" class="spin" />
        <span>加载中...</span>
      </div>
      <div v-else-if="!tasks || tasks.length === 0" class="empty-state">
        <CheckCircleIcon :size="48" />
        <p>暂无待办任务</p>
        <span class="hint">所有流程都在正常运行中</span>
      </div>
      <div v-else class="task-grid">
        <div
          v-for="task in tasks"
          :key="task.execution_id"
          class="task-card"
        >
          <div class="card-header">
            <div class="card-type">
              <ClockIcon :size="14" /> 等待中
            </div>
            <div class="card-time">
              {{ formatTime(task.started_at) }}
            </div>
          </div>
          <h3 class="card-title">{{ task.flow_name || '流程任务' }}</h3>
          <div class="card-prompt" v-if="task.prompt">
            <MessageSquareIcon :size="14" />
            <span>{{ task.prompt }}</span>
          </div>
          <div class="card-meta">
            <span>执行 ID: {{ task.execution_id.substring(0, 8) }}...</span>
          </div>

          <!-- 快捷选项 -->
          <div v-if="task.options && task.options.length > 0" class="quick-options">
            <button
              v-for="(opt, idx) in task.options"
              :key="idx"
              class="option-btn"
              @click="submitTask(task.execution_id, opt)"
            >
              {{ opt }}
            </button>
          </div>

          <!-- 自定义输入 -->
          <div class="custom-input">
            <textarea
              v-model="inputMap[task.execution_id]"
              rows="2"
              placeholder="输入回复内容..."
            />
            <button
              class="btn-primary btn-submit"
              :disabled="submitting === task.execution_id"
              @click="submitTask(task.execution_id, inputMap[task.execution_id])"
            >
              <SendIcon :size="14" />
              {{ submitting === task.execution_id ? '提交中...' : '提交' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import {
  ClockIcon,
  CheckCircleIcon,
  LoaderIcon,
  RefreshCwIcon,
  MessageSquareIcon,
  SendIcon
} from 'lucide-vue-next'

interface PendingTask {
  execution_id: string
  flow_id: string
  flow_name: string
  status: string
  started_at: string
  waiting_for: string
  prompt: string
  options: string[]
  context: Record<string, any>
}

const tasks = ref<PendingTask[]>([])
const loading = ref(true)
const submitting = ref<string | null>(null)
const inputMap = ref<Record<string, string>>({})

let refreshInterval: number | null = null

async function loadTasks() {
  loading.value = true
  try {
    // 获取所有等待中的执行实例
    const res = await fetch('/api/flows/executions?status=waiting')
    if (res.ok) {
      const data = await res.json()
      // 转换为待办任务格式
      tasks.value = (data.executions || []).map((exec: any) => ({
        execution_id: exec.id,
        flow_id: exec.flow_id,
        flow_name: exec.flow_name || '流程任务',
        status: exec.status,
        started_at: exec.started_at,
        waiting_for: exec.context?.waiting_for || '',
        prompt: exec.context?.waiting_node_prompt || '请提供输入',
        options: exec.context?.waiting_node_options || [],
        context: exec.context || {}
      }))
    }
  } catch (e) {
    console.error('Failed to load tasks:', e)
  } finally {
    loading.value = false
  }
}

async function submitTask(executionId: string, input: string) {
  if (!input || !input.trim()) {
    alert('请输入回复内容')
    return
  }

  submitting.value = executionId
  try {
    const res = await fetch(`/api/flows/executions/${executionId}/continue`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ input: input.trim() })
    })

    if (res.ok) {
      // 从列表中移除已处理的任务
      tasks.value = tasks.value.filter(t => t.execution_id !== executionId)
      delete inputMap.value[executionId]
    } else {
      const err = await res.json()
      alert(err.error || '提交失败')
    }
  } catch (e) {
    console.error('Failed to submit task:', e)
    alert('提交失败')
  } finally {
    submitting.value = null
  }
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

onMounted(() => {
  loadTasks()
  // 每 30 秒自动刷新
  refreshInterval = window.setInterval(loadTasks, 30000)
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
  max-width: 1200px;
  margin: 0 auto;
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

.task-list {
  margin-top: 16px;
}

.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: var(--text-secondary);
  gap: 12px;
}

.empty-state .hint {
  font-size: 12px;
  color: var(--text-muted);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.task-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 16px;
}

.task-card {
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 16px;
  transition: box-shadow 0.2s;
}

.task-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.card-type {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #f59e0b;
  background: #fef3c7;
  padding: 2px 8px;
  border-radius: 4px;
}

.card-time {
  font-size: 12px;
  color: var(--text-muted);
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 8px 0;
}

.card-prompt {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px;
  background: var(--bg-app);
  border-radius: 8px;
  margin-bottom: 12px;
  font-size: 14px;
  color: var(--text-primary);
  line-height: 1.5;
}

.card-prompt svg {
  flex-shrink: 0;
  margin-top: 2px;
  color: var(--accent);
}

.card-meta {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 12px;
}

.quick-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.option-btn {
  padding: 6px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.option-btn:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.custom-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.custom-input textarea {
  width: 100%;
  padding: 10px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 14px;
  resize: vertical;
  min-height: 60px;
}

.custom-input textarea:focus {
  outline: none;
  border-color: var(--accent);
}

.btn-primary {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 16px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.2s;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-submit {
  width: 100%;
}
</style>