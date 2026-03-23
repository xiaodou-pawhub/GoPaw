<template>
  <div class="traces-root">
    <!-- 头部工具栏 -->
    <div class="traces-header">
      <div class="header-left">
        <h2 class="page-title">执行轨迹</h2>
        <span class="page-subtitle">查看 Agent 执行详情</span>
      </div>
      <div class="header-right">
        <div class="filter-group">
          <select v-model="filterStatus" class="filter-select">
            <option value="">全部状态</option>
            <option value="completed">已完成</option>
            <option value="error">错误</option>
            <option value="running">运行中</option>
          </select>
          <button class="refresh-btn" @click="loadTraces" :disabled="loading">
            <RefreshCwIcon :size="14" :class="{ spinning: loading }" />
            <span>刷新</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div v-if="stats" class="stats-row">
      <div class="stat-card">
        <span class="stat-value">{{ stats.total_traces }}</span>
        <span class="stat-label">总轨迹</span>
      </div>
      <div class="stat-card">
        <span class="stat-value" style="color: var(--green)">{{ stats.completed }}</span>
        <span class="stat-label">已完成</span>
      </div>
      <div class="stat-card">
        <span class="stat-value" style="color: var(--red)">{{ stats.errors }}</span>
        <span class="stat-label">错误</span>
      </div>
      <div class="stat-card">
        <span class="stat-value">{{ formatDuration(stats.avg_duration_ms) }}</span>
        <span class="stat-label">平均耗时</span>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading && traces.length === 0" class="traces-loading">
      <div class="loading-spinner" />
      <span>加载执行轨迹...</span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="traces.length === 0" class="traces-empty">
      <ActivityIcon :size="48" class="empty-icon" />
      <p class="empty-text">暂无执行轨迹</p>
      <p class="empty-hint">Agent 执行后会自动生成轨迹记录</p>
    </div>

    <!-- 轨迹列表 -->
    <div v-else class="traces-list">
      <div
        v-for="trace in filteredTraces"
        :key="trace.id"
        class="trace-item"
        :class="{ expanded: expandedTrace === trace.id }"
        @click="toggleExpand(trace.id)"
      >
        <div class="trace-header">
          <div class="trace-status" :style="{ background: getStatusColor(trace.status) }">
            <component :is="getStatusIcon(trace.status)" :size="12" />
          </div>
          <div class="trace-info">
            <div class="trace-title">
              <span class="trace-id">{{ trace.id.slice(0, 8) }}</span>
              <span class="trace-session">{{ trace.session_id }}</span>
            </div>
            <div class="trace-meta">
              <span class="meta-item">
                <ClockIcon :size="12" />
                {{ formatTimestamp(trace.started_at) }}
              </span>
              <span class="meta-item">
                <TimerIcon :size="12" />
                {{ formatDuration(trace.duration_ms) }}
              </span>
              <span class="meta-item">
                <StepsIcon :size="12" />
                {{ trace.step_count }} 步骤
              </span>
            </div>
          </div>
          <div class="trace-actions">
            <button
              class="action-btn"
              :class="{ active: expandedTrace === trace.id }"
              @click.stop="toggleExpand(trace.id)"
            >
              <ChevronDownIcon :size="16" :class="{ rotated: expandedTrace === trace.id }" />
            </button>
          </div>
        </div>

        <!-- 展开的详情 -->
        <div v-if="expandedTrace === trace.id" class="trace-detail">
          <TraceTimeline
            :trace-id="trace.id"
            @close="expandedTrace = null"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  ActivityIcon,
  RefreshCwIcon,
  ClockIcon,
  TimerIcon,
  ChevronDownIcon,
  CheckCircleIcon,
  XCircleIcon,
  LoaderIcon
} from 'lucide-vue-next'
import { listTraces, getTraceStats, type Trace } from '@/api/trace'
import { formatDuration, formatTimestamp, getStatusColor } from '@/api/trace'
import TraceTimeline from '@/components/trace/TraceTimeline.vue'

// ---- State ----
const traces = ref<Trace[]>([])
const stats = ref<{
  total_traces: number
  completed: number
  errors: number
  avg_duration_ms: number
} | null>(null)
const loading = ref(false)
const filterStatus = ref('')
const expandedTrace = ref<string | null>(null)

// ---- Computed ----
const filteredTraces = computed(() => {
  if (!filterStatus.value) return traces.value
  return traces.value.filter(t => t.status === filterStatus.value)
})

// ---- Methods ----
async function loadTraces() {
  loading.value = true
  try {
    const [tracesRes, statsRes] = await Promise.all([
      listTraces({ limit: 100 }),
      getTraceStats()
    ])
    traces.value = tracesRes?.traces || []
    stats.value = statsRes || { total_traces: 0, completed: 0, errors: 0, avg_duration_ms: 0 }
  } catch (err) {
    console.error('Failed to load traces:', err)
    traces.value = []
    stats.value = { total_traces: 0, completed: 0, errors: 0, avg_duration_ms: 0 }
  } finally {
    loading.value = false
  }
}

function toggleExpand(traceId: string) {
  expandedTrace.value = expandedTrace.value === traceId ? null : traceId
}

function getStatusIcon(status: string) {
  switch (status) {
    case 'completed': return CheckCircleIcon
    case 'error': return XCircleIcon
    case 'running': return LoaderIcon
    default: return ActivityIcon
  }
}

import { h, type PropType } from 'vue'

// StepsIcon component
const StepsIcon = {
  props: {
    size: {
      type: Number as PropType<number>,
      default: 12
    }
  },
  setup(props: { size: number }) {
    return () => h('svg', {
      width: props.size,
      height: props.size,
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': 2
    }, [
      h('path', { d: 'M4 19h16M4 12h16M4 5h16' })
    ])
  }
}

// ---- Lifecycle ----
onMounted(() => {
  loadTraces()
})
</script>

<style scoped>
.traces-root {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 20px;
  overflow: hidden;
}

.traces-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.header-left {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
}

.page-subtitle {
  font-size: 13px;
  color: var(--text-tertiary);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-select {
  padding: 6px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-input);
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
}

.refresh-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-input);
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}

.refresh-btn:hover:not(:disabled) {
  background: var(--bg-overlay);
}

.refresh-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* Stats Row */
.stats-row {
  display: flex;
  gap: 16px;
  margin-bottom: 20px;
}

.stat-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 8px;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
}

.stat-label {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-top: 4px;
}

/* Loading & Empty */
.traces-loading,
.traces-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-tertiary);
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.empty-icon {
  color: var(--text-tertiary);
  opacity: 0.5;
}

.empty-text {
  font-size: 15px;
  color: var(--text-secondary);
}

.empty-hint {
  font-size: 13px;
  color: var(--text-tertiary);
}

/* Traces List */
.traces-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.trace-item {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.15s;
}

.trace-item:hover {
  border-color: var(--border-hover);
}

.trace-item.expanded {
  border-color: var(--accent);
}

.trace-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
}

.trace-status {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  color: white;
}

.trace-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.trace-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.trace-id {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  font-family: monospace;
}

.trace-session {
  font-size: 12px;
  color: var(--text-tertiary);
}

.trace-meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--text-tertiary);
}

.trace-actions {
  display: flex;
  align-items: center;
}

.action-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: all 0.15s;
}

.action-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-secondary);
}

.action-btn.active {
  color: var(--accent);
}

.rotated {
  transform: rotate(180deg);
}

/* Trace Detail */
.trace-detail {
  border-top: 1px solid var(--border);
  padding: 16px;
  background: var(--bg-app);
}
</style>
