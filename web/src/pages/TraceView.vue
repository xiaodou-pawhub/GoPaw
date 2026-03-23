<template>
  <div class="trace-view">
    <!-- 头部信息 -->
    <div class="trace-header">
      <div class="header-left">
        <h2>
          <i class="bi bi-diagram-3"></i>
          执行追踪
        </h2>
        <span class="trace-id" v-if="trace">{{ trace.id }}</span>
      </div>
      <div class="header-right">
        <button class="btn btn-outline-secondary" @click="loadTrace" :disabled="loading">
          <i class="bi bi-arrow-clockwise"></i>
          刷新
        </button>
        <router-link to="/flows/executions" class="btn btn-outline-secondary">
          <i class="bi bi-arrow-left"></i>
          返回
        </router-link>
      </div>
    </div>

    <!-- 加载状态 -->
    <div class="loading-container" v-if="loading && !trace">
      <div class="spinner-border text-primary" role="status">
        <span class="visually-hidden">加载中...</span>
      </div>
    </div>

    <!-- 错误状态 -->
    <div class="alert alert-danger" v-if="error">
      <i class="bi bi-exclamation-triangle"></i>
      {{ error }}
    </div>

    <!-- 追踪内容 -->
    <div class="trace-content" v-if="trace">
      <!-- 概览卡片 -->
      <div class="overview-cards">
        <div class="overview-card">
          <div class="card-icon status" :class="trace.status">
            <i :class="getStatusIcon(trace.status)"></i>
          </div>
          <div class="card-content">
            <div class="card-label">状态</div>
            <div class="card-value" :class="trace.status">{{ getStatusText(trace.status) }}</div>
          </div>
        </div>

        <div class="overview-card">
          <div class="card-icon duration">
            <i class="bi bi-clock"></i>
          </div>
          <div class="card-content">
            <div class="card-label">耗时</div>
            <div class="card-value">{{ formatDuration(trace.duration) }}</div>
          </div>
        </div>

        <div class="overview-card">
          <div class="card-icon tokens">
            <i class="bi bi-cpu"></i>
          </div>
          <div class="card-content">
            <div class="card-label">Token</div>
            <div class="card-value">{{ formatNumber(trace.total_tokens) }}</div>
          </div>
        </div>

        <div class="overview-card">
          <div class="card-icon cost">
            <i class="bi bi-currency-dollar"></i>
          </div>
          <div class="card-content">
            <div class="card-label">成本</div>
            <div class="card-value">${{ trace.total_cost.toFixed(4) }}</div>
          </div>
        </div>
      </div>

      <!-- 基本信息 -->
      <div class="info-section">
        <h5><i class="bi bi-info-circle"></i> 基本信息</h5>
        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">流程名称</span>
            <span class="info-value">{{ trace.flow_name || trace.flow_id }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">触发方式</span>
            <span class="info-value">
              <span class="badge bg-secondary">{{ getTriggerText(trace.trigger) }}</span>
            </span>
          </div>
          <div class="info-item">
            <span class="info-label">开始时间</span>
            <span class="info-value">{{ formatTime(trace.started_at) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">结束时间</span>
            <span class="info-value">{{ trace.completed_at ? formatTime(trace.completed_at) : '-' }}</span>
          </div>
        </div>
      </div>

      <!-- 输入输出 -->
      <div class="io-section" v-if="trace.metadata">
        <div class="io-card" v-if="trace.metadata.input">
          <h6><i class="bi bi-box-arrow-in-right"></i> 输入</h6>
          <pre class="io-content">{{ formatJSON(trace.metadata.input) }}</pre>
        </div>
        <div class="io-card" v-if="trace.metadata.output">
          <h6><i class="bi bi-box-arrow-right"></i> 输出</h6>
          <pre class="io-content">{{ formatJSON(trace.metadata.output) }}</pre>
        </div>
      </div>

      <!-- 错误信息 -->
      <div class="error-section" v-if="trace.error">
        <h5><i class="bi bi-exclamation-triangle text-danger"></i> 错误信息</h5>
        <div class="error-content">
          <pre>{{ trace.error }}</pre>
        </div>
      </div>

      <!-- Span 时间线 -->
      <div class="timeline-section">
        <h5><i class="bi bi-list-check"></i> 执行时间线</h5>
        <div class="timeline">
          <div
            class="timeline-item"
            v-for="span in spans"
            :key="span.id"
            :class="{ active: selectedSpan?.id === span.id }"
            @click="selectSpan(span)"
          >
            <div class="timeline-marker" :class="span.status">
              <i :class="getNodeIcon(span.node_type)"></i>
            </div>
            <div class="timeline-content">
              <div class="timeline-header">
                <span class="node-name">{{ span.node_name || span.node_id }}</span>
                <span class="node-type badge">{{ span.node_type }}</span>
                <span class="duration">{{ formatDuration(span.duration) }}</span>
              </div>
              <div class="timeline-details" v-if="span.agent_name">
                <i class="bi bi-robot"></i> {{ span.agent_name }}
                <span v-if="span.model" class="model">{{ span.model }}</span>
              </div>
              <div class="timeline-tokens" v-if="span.tokens_in || span.tokens_out">
                <span class="token-in">↑{{ span.tokens_in }}</span>
                <span class="token-out">↓{{ span.tokens_out }}</span>
                <span class="token-cost">${{ span.cost.toFixed(4) }}</span>
              </div>
              <div class="timeline-error" v-if="span.error">
                <i class="bi bi-exclamation-circle text-danger"></i>
                {{ span.error }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Span 详情 -->
      <div class="span-detail" v-if="selectedSpan">
        <h5><i class="bi bi-card-list"></i> Span 详情</h5>
        <div class="detail-header">
          <span class="span-id">{{ selectedSpan.id }}</span>
          <span class="badge" :class="getStatusClass(selectedSpan.status)">
            {{ getStatusText(selectedSpan.status) }}
          </span>
        </div>

        <div class="detail-tabs">
          <button
            :class="['tab-btn', { active: activeTab === 'info' }]"
            @click="activeTab = 'info'"
          >
            基本信息
          </button>
          <button
            :class="['tab-btn', { active: activeTab === 'events' }]"
            @click="activeTab = 'events'"
          >
            事件 ({{ selectedSpan.events?.length || 0 }})
          </button>
          <button
            :class="['tab-btn', { active: activeTab === 'io' }]"
            @click="activeTab = 'io'"
          >
            输入输出
          </button>
        </div>

        <div class="detail-content">
          <!-- 基本信息 -->
          <div class="tab-content" v-if="activeTab === 'info'">
            <div class="detail-grid">
              <div class="detail-item">
                <span class="detail-label">节点 ID</span>
                <span class="detail-value">{{ selectedSpan.node_id }}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">节点类型</span>
                <span class="detail-value">{{ selectedSpan.node_type }}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">开始时间</span>
                <span class="detail-value">{{ formatTime(selectedSpan.started_at) }}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">结束时间</span>
                <span class="detail-value">{{ selectedSpan.completed_at ? formatTime(selectedSpan.completed_at) : '-' }}</span>
              </div>
              <div class="detail-item" v-if="selectedSpan.agent_name">
                <span class="detail-label">Agent</span>
                <span class="detail-value">{{ selectedSpan.agent_name }}</span>
              </div>
              <div class="detail-item" v-if="selectedSpan.model">
                <span class="detail-label">模型</span>
                <span class="detail-value">{{ selectedSpan.model }}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">输入 Token</span>
                <span class="detail-value">{{ selectedSpan.tokens_in }}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">输出 Token</span>
                <span class="detail-value">{{ selectedSpan.tokens_out }}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">成本</span>
                <span class="detail-value">${{ selectedSpan.cost.toFixed(4) }}</span>
              </div>
            </div>

            <!-- 标签 -->
            <div class="tags-section" v-if="hasTags(selectedSpan.tags)">
              <h6>标签</h6>
              <div class="tags-list">
                <span class="tag" v-if="selectedSpan.tags?.loop_iteration">
                  <i class="bi bi-arrow-repeat"></i> 循环 #{{ selectedSpan.tags.loop_iteration }}
                </span>
                <span class="tag" v-if="selectedSpan.tags?.branch_name">
                  <i class="bi bi-diagram-2"></i> {{ selectedSpan.tags.branch_name }}
                </span>
                <span class="tag" v-if="selectedSpan.tags?.retry_attempt">
                  <i class="bi bi-arrow-clockwise"></i> 重试 #{{ selectedSpan.tags.retry_attempt }}
                </span>
                <span class="tag" v-if="selectedSpan.tags?.is_fallback">
                  <i class="bi bi-arrow-branch"></i> Fallback
                </span>
                <span class="tag" v-if="selectedSpan.tags?.cache_hit">
                  <i class="bi bi-lightning"></i> 缓存命中
                </span>
              </div>
            </div>
          </div>

          <!-- 事件列表 -->
          <div class="tab-content" v-if="activeTab === 'events'">
            <div class="events-list" v-if="selectedSpan.events?.length">
              <div
                class="event-item"
                v-for="event in selectedSpan.events"
                :key="event.id"
              >
                <div class="event-marker" :class="getEventClass(event.type)">
                  <i :class="getEventIcon(event.type)"></i>
                </div>
                <div class="event-content">
                  <div class="event-header">
                    <span class="event-name">{{ event.name }}</span>
                    <span class="event-type badge">{{ event.type }}</span>
                    <span class="event-time">{{ formatTime(event.timestamp) }}</span>
                  </div>
                  <div class="event-attrs" v-if="event.attributes && Object.keys(event.attributes).length">
                    <pre>{{ formatJSON(event.attributes) }}</pre>
                  </div>
                </div>
              </div>
            </div>
            <div class="empty-state" v-else>
              <i class="bi bi-inbox"></i>
              <p>暂无事件</p>
            </div>
          </div>

          <!-- 输入输出 -->
          <div class="tab-content" v-if="activeTab === 'io'">
            <div class="io-section">
              <div class="io-block" v-if="selectedSpan.input">
                <h6>输入</h6>
                <pre class="io-code">{{ formatJSON(selectedSpan.input) }}</pre>
              </div>
              <div class="io-block" v-if="selectedSpan.output">
                <h6>输出</h6>
                <pre class="io-code">{{ formatJSON(selectedSpan.output) }}</pre>
              </div>
              <div class="empty-state" v-if="!selectedSpan.input && !selectedSpan.output">
                <i class="bi bi-inbox"></i>
                <p>暂无输入输出数据</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'

interface TraceMetadata {
  input?: string
  output?: string
  variables?: Record<string, any>
  user_id?: string
  session_id?: string
}

interface Trace {
  id: string
  flow_id: string
  flow_name: string
  execution_id: string
  trigger: string
  status: string
  started_at: string
  completed_at?: string
  duration: number
  total_tokens: number
  total_cost: number
  root_span_id?: string
  error?: string
  metadata?: TraceMetadata
}

interface SpanTags {
  loop_iteration?: number
  branch_name?: string
  parallel_index?: number
  retry_attempt?: number
  is_fallback?: boolean
  cache_hit?: boolean
}

interface Event {
  id: string
  span_id: string
  name: string
  type: string
  timestamp: string
  attributes?: Record<string, any>
}

interface Span {
  id: string
  trace_id: string
  parent_span_id?: string
  node_id: string
  node_name: string
  node_type: string
  status: string
  started_at: string
  completed_at?: string
  duration: number
  tokens_in: number
  tokens_out: number
  cost: number
  agent_id?: string
  agent_name?: string
  model?: string
  input?: string
  output?: string
  error?: string
  events?: Event[]
  tags?: SpanTags
}

const route = useRoute()
const loading = ref(false)
const error = ref('')
const trace = ref<Trace | null>(null)
const spans = ref<Span[]>([])
const selectedSpan = ref<Span | null>(null)
const activeTab = ref('info')

// 加载追踪数据
async function loadTrace() {
  const execId = route.params.execId as string
  if (!execId) return

  loading.value = true
  error.value = ''

  try {
    // 加载追踪
    const traceRes = await fetch(`/api/flows/executions/${execId}/trace`)
    if (traceRes.ok) {
      trace.value = await traceRes.json()
    } else {
      throw new Error('加载追踪数据失败')
    }

    // 加载 Spans
    const spansRes = await fetch(`/api/flows/executions/${execId}/spans`)
    if (spansRes.ok) {
      spans.value = await spansRes.json() || []
    }

    // 默认选中第一个 Span
    if (spans.value.length > 0) {
      selectedSpan.value = spans.value[0]
    }
  } catch (e: any) {
    error.value = e.message || '加载追踪数据失败'
  } finally {
    loading.value = false
  }
}

// 选择 Span
function selectSpan(span: Span) {
  selectedSpan.value = span
  activeTab.value = 'info'
}

// 格式化时间
function formatTime(time: string) {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

// 格式化持续时间
function formatDuration(ms: number) {
  if (!ms) return '0ms'
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`
  return `${(ms / 60000).toFixed(2)}min`
}

// 格式化数字
function formatNumber(n: number) {
  if (!n) return '0'
  if (n >= 1000000) return `${(n / 1000000).toFixed(2)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(2)}K`
  return n.toString()
}

// 格式化 JSON
function formatJSON(data: any) {
  if (!data) return ''
  if (typeof data === 'string') {
    try {
      return JSON.stringify(JSON.parse(data), null, 2)
    } catch {
      return data
    }
  }
  return JSON.stringify(data, null, 2)
}

// 获取状态图标
function getStatusIcon(status: string) {
  switch (status) {
    case 'completed': return 'bi bi-check-circle-fill'
    case 'running': return 'bi bi-arrow-repeat'
    case 'failed': return 'bi bi-x-circle-fill'
    case 'waiting': return 'bi bi-pause-circle-fill'
    default: return 'bi bi-question-circle'
  }
}

// 获取状态文本
function getStatusText(status: string) {
  switch (status) {
    case 'completed': return '已完成'
    case 'running': return '运行中'
    case 'failed': return '失败'
    case 'waiting': return '等待中'
    case 'skipped': return '跳过'
    default: return status
  }
}

// 获取状态样式类
function getStatusClass(status: string) {
  switch (status) {
    case 'completed': return 'bg-success'
    case 'running': return 'bg-primary'
    case 'failed': return 'bg-danger'
    case 'waiting': return 'bg-warning'
    case 'skipped': return 'bg-secondary'
    default: return 'bg-secondary'
  }
}

// 获取触发方式文本
function getTriggerText(trigger: string) {
  switch (trigger) {
    case 'manual': return '手动触发'
    case 'webhook': return 'Webhook'
    case 'schedule': return '定时触发'
    case 'api': return 'API 调用'
    default: return trigger || '手动触发'
  }
}

// 获取节点图标
function getNodeIcon(type: string) {
  switch (type) {
    case 'start': return 'bi bi-play-circle'
    case 'end': return 'bi bi-stop-circle'
    case 'agent': return 'bi bi-robot'
    case 'human': return 'bi bi-person'
    case 'condition': return 'bi bi-signpost-split'
    case 'parallel': return 'bi bi-diagram-3'
    case 'loop': return 'bi bi-arrow-repeat'
    case 'subflow': return 'bi bi-diagram-2'
    case 'webhook': return 'bi bi-link-45deg'
    default: return 'bi bi-square'
  }
}

// 获取事件图标
function getEventIcon(type: string) {
  switch (type) {
    case 'node_start': return 'bi bi-play-fill'
    case 'node_complete': return 'bi bi-check-fill'
    case 'node_fail': return 'bi bi-x-fill'
    case 'node_retry': return 'bi bi-arrow-clockwise'
    case 'llm_call_start': return 'bi bi-cpu'
    case 'llm_call_end': return 'bi bi-cpu-fill'
    case 'tool_call_start': return 'bi bi-wrench'
    case 'tool_call_end': return 'bi bi-wrench-adjustable'
    case 'human_input': return 'bi bi-person-input'
    case 'human_output': return 'bi bi-person-output'
    case 'condition_eval': return 'bi bi-signpost-2'
    case 'loop_iterate': return 'bi bi-arrow-repeat'
    case 'parallel_start': return 'bi bi-diagram-3'
    case 'parallel_end': return 'bi bi-diagram-3-fill'
    case 'cache_hit': return 'bi bi-lightning-fill'
    case 'cache_miss': return 'bi bi-lightning'
    case 'error': return 'bi bi-exclamation-triangle-fill'
    case 'warning': return 'bi bi-exclamation-circle-fill'
    default: return 'bi bi-circle-fill'
  }
}

// 获取事件样式类
function getEventClass(type: string) {
  if (type.includes('fail') || type.includes('error')) return 'danger'
  if (type.includes('complete') || type.includes('end')) return 'success'
  if (type.includes('start') || type.includes('begin')) return 'primary'
  if (type.includes('warning')) return 'warning'
  return 'secondary'
}

// 检查是否有标签
function hasTags(tags?: SpanTags) {
  if (!tags) return false
  return tags.loop_iteration || tags.branch_name || tags.retry_attempt || tags.is_fallback || tags.cache_hit
}

onMounted(() => {
  loadTrace()
})
</script>

<style scoped>
.trace-view {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.trace-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-left h2 {
  margin: 0;
  font-size: 1.5rem;
  display: flex;
  align-items: center;
  gap: 8px;
}

.trace-id {
  font-size: 0.85rem;
  color: #6c757d;
  font-family: monospace;
}

.header-right {
  display: flex;
  gap: 8px;
}

.loading-container {
  display: flex;
  justify-content: center;
  padding: 60px;
}

/* 概览卡片 */
.overview-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.overview-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.card-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.5rem;
}

.card-icon.status {
  background: #e3f2fd;
  color: #1976d2;
}

.card-icon.status.completed {
  background: #e8f5e9;
  color: #2e7d32;
}

.card-icon.status.failed {
  background: #ffebee;
  color: #c62828;
}

.card-icon.status.running {
  background: #fff3e0;
  color: #f57c00;
}

.card-icon.duration {
  background: #f3e5f5;
  color: #7b1fa2;
}

.card-icon.tokens {
  background: #e0f7fa;
  color: #00838f;
}

.card-icon.cost {
  background: #fff8e1;
  color: #f9a825;
}

.card-label {
  font-size: 0.85rem;
  color: #6c757d;
}

.card-value {
  font-size: 1.25rem;
  font-weight: 600;
}

.card-value.completed { color: #2e7d32; }
.card-value.failed { color: #c62828; }
.card-value.running { color: #f57c00; }

/* 信息区域 */
.info-section, .io-section, .error-section, .timeline-section, .span-detail {
  background: white;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.info-section h5, .timeline-section h5, .span-detail h5 {
  margin: 0 0 16px;
  font-size: 1rem;
  display: flex;
  align-items: center;
  gap: 8px;
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
  font-size: 0.85rem;
  color: #6c757d;
}

.info-value {
  font-weight: 500;
}

/* 输入输出 */
.io-section {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

.io-card h6 {
  margin: 0 0 12px;
  font-size: 0.9rem;
  display: flex;
  align-items: center;
  gap: 6px;
}

.io-content {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 12px;
  font-size: 0.85rem;
  max-height: 200px;
  overflow: auto;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 错误区域 */
.error-section h5 {
  color: #c62828;
}

.error-content {
  background: #ffebee;
  border-radius: 8px;
  padding: 12px;
}

.error-content pre {
  margin: 0;
  color: #c62828;
  white-space: pre-wrap;
}

/* 时间线 */
.timeline {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.timeline-item {
  display: flex;
  gap: 16px;
  padding: 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
}

.timeline-item:hover {
  background: #f8f9fa;
}

.timeline-item.active {
  background: #e3f2fd;
  border: 1px solid #90caf9;
}

.timeline-marker {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: #e3f2fd;
  color: #1976d2;
}

.timeline-marker.completed {
  background: #e8f5e9;
  color: #2e7d32;
}

.timeline-marker.failed {
  background: #ffebee;
  color: #c62828;
}

.timeline-marker.running {
  background: #fff3e0;
  color: #f57c00;
}

.timeline-marker.skipped {
  background: #f5f5f5;
  color: #9e9e9e;
}

.timeline-content {
  flex: 1;
  min-width: 0;
}

.timeline-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.node-name {
  font-weight: 500;
}

.node-type {
  font-size: 0.75rem;
  background: #e9ecef;
}

.duration {
  margin-left: auto;
  font-size: 0.85rem;
  color: #6c757d;
  font-family: monospace;
}

.timeline-details {
  font-size: 0.85rem;
  color: #6c757d;
  display: flex;
  align-items: center;
  gap: 8px;
}

.timeline-details .model {
  background: #e9ecef;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.75rem;
}

.timeline-tokens {
  font-size: 0.8rem;
  color: #6c757d;
  display: flex;
  gap: 12px;
  margin-top: 4px;
}

.token-in { color: #1976d2; }
.token-out { color: #2e7d32; }
.token-cost { color: #f9a825; }

.timeline-error {
  font-size: 0.85rem;
  color: #c62828;
  margin-top: 4px;
}

/* Span 详情 */
.detail-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.span-id {
  font-family: monospace;
  font-size: 0.85rem;
  color: #6c757d;
}

.detail-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 16px;
  border-bottom: 1px solid #e9ecef;
  padding-bottom: 8px;
}

.tab-btn {
  padding: 8px 16px;
  border: none;
  background: none;
  cursor: pointer;
  border-radius: 6px 6px 0 0;
  font-size: 0.9rem;
  color: #6c757d;
  transition: all 0.2s;
}

.tab-btn:hover {
  background: #f8f9fa;
}

.tab-btn.active {
  background: #e3f2fd;
  color: #1976d2;
  font-weight: 500;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-label {
  font-size: 0.85rem;
  color: #6c757d;
}

.detail-value {
  font-weight: 500;
}

/* 标签 */
.tags-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e9ecef;
}

.tags-section h6 {
  margin: 0 0 8px;
  font-size: 0.85rem;
  color: #6c757d;
}

.tags-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: #e9ecef;
  border-radius: 16px;
  font-size: 0.8rem;
}

/* 事件列表 */
.events-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.event-item {
  display: flex;
  gap: 12px;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
}

.event-marker {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: #e9ecef;
  color: #6c757d;
}

.event-marker.success { background: #e8f5e9; color: #2e7d32; }
.event-marker.danger { background: #ffebee; color: #c62828; }
.event-marker.primary { background: #e3f2fd; color: #1976d2; }
.event-marker.warning { background: #fff3e0; color: #f57c00; }

.event-content {
  flex: 1;
  min-width: 0;
}

.event-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.event-name {
  font-weight: 500;
}

.event-type {
  font-size: 0.75rem;
  background: #e9ecef;
}

.event-time {
  margin-left: auto;
  font-size: 0.8rem;
  color: #6c757d;
}

.event-attrs pre {
  margin: 8px 0 0;
  padding: 8px;
  background: white;
  border-radius: 4px;
  font-size: 0.8rem;
  max-height: 150px;
  overflow: auto;
}

/* IO 区块 */
.io-block {
  margin-bottom: 16px;
}

.io-block h6 {
  margin: 0 0 8px;
  font-size: 0.85rem;
  color: #6c757d;
}

.io-code {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 12px;
  font-size: 0.85rem;
  max-height: 300px;
  overflow: auto;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 40px;
  color: #6c757d;
}

.empty-state i {
  font-size: 2rem;
  margin-bottom: 8px;
}

.empty-state p {
  margin: 0;
}

/* 响应式 */
@media (max-width: 768px) {
  .overview-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .info-grid, .detail-grid {
    grid-template-columns: 1fr;
  }

  .io-section {
    grid-template-columns: 1fr;
  }
}
</style>