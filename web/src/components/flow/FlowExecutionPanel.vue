<template>
  <div class="execution-panel">
    <!-- 面板头部 -->
    <div class="panel-header">
      <div class="header-left">
        <span class="panel-title">{{ flowName || '流程执行' }}</span>
        <span class="exec-id">{{ executionId }}</span>
      </div>
      <button class="close-btn" @click="emit('close')">
        <XIcon :size="16" />
      </button>
    </div>

    <!-- 加载中 -->
    <div v-if="loading && !exec" class="panel-loading">
      <div class="spinner" />
      <span>加载中...</span>
    </div>

    <!-- 执行内容 -->
    <template v-else-if="exec">
      <!-- 状态概览 -->
      <div class="status-bar" :class="`status-bar--${exec.status}`">
        <!-- running -->
        <template v-if="exec.status === 'running'">
          <div class="spinner spinner--sm" />
          <span>执行中...</span>
        </template>
        <!-- waiting -->
        <template v-else-if="exec.status === 'waiting'">
          <ClockIcon :size="16" class="status-icon" />
          <span>等待人工输入</span>
        </template>
        <!-- completed -->
        <template v-else-if="exec.status === 'completed'">
          <CheckCircleIcon :size="16" class="status-icon" />
          <span>已完成</span>
          <span v-if="exec.duration_ms" class="duration">
            耗时 {{ formatDuration(exec.duration_ms) }}
          </span>
        </template>
        <!-- failed -->
        <template v-else-if="exec.status === 'failed'">
          <XCircleIcon :size="16" class="status-icon" />
          <span>执行失败</span>
        </template>
      </div>

      <!-- 滚动内容区 -->
      <div class="panel-body">

        <!-- 最终输出（completed） -->
        <div v-if="exec.status === 'completed' && exec.output" class="section">
          <div class="section-title">最终输出</div>
          <textarea
            class="output-textarea"
            :value="exec.output"
            readonly
            rows="5"
          />
        </div>

        <!-- 错误信息（failed） -->
        <div v-if="exec.status === 'failed' && exec.error" class="section">
          <div class="section-title">错误信息</div>
          <div class="error-box">{{ exec.error }}</div>
        </div>

        <!-- 人工交互区（waiting） -->
        <div v-if="exec.status === 'waiting'" class="section human-section">
          <div class="section-title">需要您的输入</div>

          <!-- 节点提示文本 -->
          <p v-if="waitingPrompt" class="waiting-prompt">{{ waitingPrompt }}</p>

          <!-- 快捷选项 -->
          <div v-if="waitingOptions.length" class="options-row">
            <button
              v-for="opt in waitingOptions"
              :key="opt"
              class="option-pill"
              :disabled="submitting"
              @click="submitInput(opt)"
            >
              {{ opt }}
            </button>
          </div>

          <!-- 手动输入 -->
          <div class="input-row">
            <textarea
              v-model="humanInput"
              class="human-textarea"
              placeholder="输入您的回复..."
              rows="3"
              :disabled="submitting"
              @keydown.ctrl.enter.prevent="submitInput(humanInput)"
            />
            <button
              class="submit-btn"
              :disabled="submitting || !humanInput.trim()"
              @click="submitInput(humanInput)"
            >
              <div v-if="submitting" class="spinner spinner--sm" />
              <span v-else>提交</span>
            </button>
          </div>
          <div class="input-hint">Ctrl+Enter 快速提交</div>
        </div>

        <!-- 步骤历史 -->
        <div v-if="exec.history && exec.history.length" class="section">
          <div class="section-title">执行步骤</div>
          <div class="history-list">
            <div
              v-for="(step, index) in exec.history"
              :key="index"
              class="history-item"
            >
              <div class="step-dot" :class="`step-dot--${step.status}`" />
              <div class="step-info">
                <div class="step-header">
                  <span class="step-name">{{ formatNodeId(step.node_id) }}</span>
                  <span class="step-type">{{ step.node_type }}</span>
                  <span class="step-status" :class="`step-status--${step.status}`">
                    {{ statusLabel(step.status) }}
                  </span>
                </div>
                <div v-if="step.output" class="step-output">{{ step.output }}</div>
              </div>
            </div>
          </div>
        </div>

      </div>
    </template>

    <!-- 加载失败 -->
    <div v-else-if="fetchError" class="panel-error">
      <XCircleIcon :size="24" />
      <span>{{ fetchError }}</span>
      <button class="retry-btn" @click="fetchExecution">重试</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  XIcon,
  CheckCircleIcon,
  XCircleIcon,
  ClockIcon,
} from 'lucide-vue-next'

interface HistoryStep {
  node_id: string
  node_type: string
  status: 'running' | 'completed' | 'failed' | string
  output?: string
}

interface ExecutionContext {
  waiting_node_prompt?: string
  waiting_node_options?: string[]
  [key: string]: unknown
}

interface Execution {
  id: string
  flow_id: string
  status: 'running' | 'waiting' | 'completed' | 'failed' | string
  output?: string
  error?: string
  duration_ms?: number
  history?: HistoryStep[]
  context?: ExecutionContext
}

const props = defineProps<{
  flowId: string
  executionId: string
  flowName?: string
}>()

const emit = defineEmits<{
  close: []
  completed: [output: string]
}>()

const exec = ref<Execution | null>(null)
const loading = ref(false)
const fetchError = ref<string | null>(null)
const humanInput = ref('')
const submitting = ref(false)

let pollTimer: ReturnType<typeof setInterval> | null = null

const waitingPrompt = computed(
  () => exec.value?.context?.waiting_node_prompt ?? ''
)

const waitingOptions = computed(
  () => exec.value?.context?.waiting_node_options ?? []
)

async function fetchExecution() {
  loading.value = true
  fetchError.value = null
  try {
    const resp = await fetch(`/api/flows/executions/${props.executionId}`)
    if (!resp.ok) throw new Error(`请求失败：${resp.status}`)
    const data: Execution = await resp.json()
    exec.value = data

    if (data.status === 'completed') {
      stopPoll()
      if (data.output) emit('completed', data.output)
    } else if (data.status === 'failed') {
      stopPoll()
    }
  } catch (e: unknown) {
    fetchError.value = e instanceof Error ? e.message : '未知错误'
  } finally {
    loading.value = false
  }
}

function startPoll() {
  if (pollTimer !== null) return
  pollTimer = setInterval(fetchExecution, 2000)
}

function stopPoll() {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function submitInput(input: string) {
  if (!input.trim() || submitting.value) return
  submitting.value = true
  try {
    const resp = await fetch(`/api/flows/executions/${props.executionId}/continue`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ input: input.trim() }),
    })
    if (!resp.ok) throw new Error(`提交失败：${resp.status}`)
    humanInput.value = ''
    // 立即刷新一次，然后继续轮询
    await fetchExecution()
    if (!pollTimer && exec.value?.status === 'running') {
      startPoll()
    }
  } catch (e: unknown) {
    alert(e instanceof Error ? e.message : '提交失败')
  } finally {
    submitting.value = false
  }
}

function formatNodeId(nodeId: string): string {
  if (!nodeId) return nodeId
  // 将 snake_case / kebab-case 转换为可读形式
  return nodeId
    .replace(/[-_]/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase())
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  const min = Math.floor(ms / 60000)
  const sec = Math.round((ms % 60000) / 1000)
  return `${min}m ${sec}s`
}

function statusLabel(status: string): string {
  const map: Record<string, string> = {
    running: '执行中',
    completed: '已完成',
    failed: '失败',
    pending: '等待',
    skipped: '跳过',
  }
  return map[status] ?? status
}

onMounted(async () => {
  await fetchExecution()
  if (exec.value && !['completed', 'failed'].includes(exec.value.status)) {
    startPoll()
  }
})

onUnmounted(() => {
  stopPoll()
})
</script>

<style scoped>
.execution-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}

/* 头部 */
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}

.panel-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.exec-id {
  font-size: 11px;
  color: var(--text-muted, #9ca3af);
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 160px;
}

.close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--text-secondary);
  cursor: pointer;
  flex-shrink: 0;
}
.close-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-primary);
}

/* 状态栏 */
.status-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  font-size: 13px;
  font-weight: 500;
  flex-shrink: 0;
  border-bottom: 1px solid var(--border);
}

.status-bar--running {
  background: color-mix(in srgb, var(--accent) 8%, transparent);
  color: var(--accent);
}

.status-bar--waiting {
  background: color-mix(in srgb, #f59e0b 8%, transparent);
  color: #d97706;
}

.status-bar--completed {
  background: color-mix(in srgb, #22c55e 8%, transparent);
  color: #16a34a;
}

.status-bar--failed {
  background: color-mix(in srgb, #ef4444 8%, transparent);
  color: #dc2626;
}

.status-icon {
  flex-shrink: 0;
}

.duration {
  margin-left: auto;
  font-size: 12px;
  opacity: 0.8;
}

/* 主体滚动区 */
.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 分节 */
.section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

/* 最终输出文本框 */
.output-textarea {
  width: 100%;
  padding: 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  line-height: 1.5;
  resize: vertical;
  font-family: inherit;
  box-sizing: border-box;
}
.output-textarea:focus {
  outline: 2px solid var(--accent);
  outline-offset: -1px;
}

/* 错误框 */
.error-box {
  padding: 10px 12px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 6px;
  color: #dc2626;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}

/* 人工交互 */
.human-section {
  padding: 14px;
  background: color-mix(in srgb, #f59e0b 6%, var(--bg-elevated));
  border: 1px solid color-mix(in srgb, #f59e0b 30%, transparent);
  border-radius: 8px;
  gap: 10px;
}

.waiting-prompt {
  font-size: 13px;
  color: var(--text-primary);
  line-height: 1.6;
  margin: 0;
}

.options-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.option-pill {
  padding: 5px 12px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 12px;
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}
.option-pill:hover:not(:disabled) {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}
.option-pill:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.input-row {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.human-textarea {
  flex: 1;
  padding: 8px 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  line-height: 1.5;
  resize: none;
  font-family: inherit;
  box-sizing: border-box;
}
.human-textarea:focus {
  outline: 2px solid var(--accent);
  outline-offset: -1px;
}
.human-textarea:disabled {
  opacity: 0.6;
}

.submit-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 64px;
  height: 36px;
  padding: 0 14px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
  flex-shrink: 0;
}
.submit-btn:hover:not(:disabled) {
  opacity: 0.85;
}
.submit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.input-hint {
  font-size: 11px;
  color: var(--text-muted, #9ca3af);
}

/* 步骤历史 */
.history-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.history-item {
  display: flex;
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px solid var(--border);
  position: relative;
}
.history-item:last-child {
  border-bottom: none;
}

.step-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-top: 4px;
  flex-shrink: 0;
}
.step-dot--running { background: var(--accent); }
.step-dot--completed { background: #22c55e; }
.step-dot--failed { background: #ef4444; }
.step-dot--pending { background: #d1d5db; }
.step-dot--skipped { background: #9ca3af; }

.step-info {
  flex: 1;
  min-width: 0;
}

.step-header {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.step-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.step-type {
  font-size: 11px;
  color: var(--text-muted, #9ca3af);
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 3px;
  padding: 1px 5px;
}

.step-status {
  font-size: 11px;
  margin-left: auto;
}
.step-status--running { color: var(--accent); }
.step-status--completed { color: #16a34a; }
.step-status--failed { color: #dc2626; }
.step-status--pending { color: #9ca3af; }
.step-status--skipped { color: #9ca3af; }

.step-output {
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 加载 / 错误占位 */
.panel-loading,
.panel-error {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--text-secondary);
  font-size: 14px;
}

.panel-error {
  color: #dc2626;
}

.retry-btn {
  padding: 6px 14px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 4px;
  font-size: 13px;
  color: var(--text-primary);
  cursor: pointer;
}
.retry-btn:hover {
  background: var(--bg-overlay);
}

/* Spinner */
.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
.spinner--sm {
  width: 14px;
  height: 14px;
  border-width: 2px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
