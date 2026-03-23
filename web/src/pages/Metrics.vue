<template>
  <div class="metrics-page">
    <div class="page-header">
      <h1 class="page-title">性能监控</h1>
      <button class="btn-icon" :class="{ spinning: loading }" title="刷新" @click="loadData">
        <RefreshCwIcon :size="16" />
      </button>
    </div>

    <div class="metrics-content">
    <div class="stat-cards">
      <div class="stat-card stat-primary">
        <div class="stat-label">Agent 调用</div>
        <div class="stat-value">{{ formatNumber(dashboard?.agent?.call_count) }}</div>
        <div class="stat-sub">成功率: {{ formatPercent(dashboard?.agent?.success_rate) }}</div>
      </div>
      <div class="stat-card stat-warning">
        <div class="stat-label">队列堆积</div>
        <div class="stat-value">{{ formatNumber(dashboard?.queue?.pending_count) }}</div>
        <div class="stat-sub">处理中: {{ formatNumber(dashboard?.queue?.processing_count) }}</div>
      </div>
      <div class="stat-card stat-success">
        <div class="stat-label">工作流执行</div>
        <div class="stat-value">{{ formatNumber(dashboard?.workflow?.execution_count) }}</div>
        <div class="stat-sub">成功率: {{ formatPercent(dashboard?.workflow?.success_rate) }}</div>
      </div>
      <div class="stat-card stat-info">
        <div class="stat-label">系统状态</div>
        <div class="stat-value">{{ formatNumber(dashboard?.system?.memory_mb) }} MB</div>
        <div class="stat-sub">Goroutines: {{ formatNumber(dashboard?.system?.goroutines) }}</div>
      </div>
    </div>

    <!-- 详细统计表格 -->
    <div class="tables-row">
      <!-- Agent 统计 -->
      <div class="data-card">
        <div class="card-title">Agent 统计</div>
        <div class="data-table">
          <div class="data-thead">
            <span>Agent ID</span><span>调用次数</span><span>成功率</span>
          </div>
          <div v-if="!dashboard?.agent?.top_agents?.length" class="empty-state">暂无数据</div>
          <div v-for="row in dashboard?.agent?.top_agents" :key="row.agent_id" class="data-row">
            <span class="mono">{{ row.agent_id }}</span>
            <span>{{ formatNumber(row.call_count) }}</span>
            <span>
              <span class="badge" :class="getSuccessRateClass(row.success_rate)">
                {{ formatPercent(row.success_rate) }}
              </span>
            </span>
          </div>
        </div>
      </div>

      <!-- 队列统计 -->
      <div class="data-card">
        <div class="card-title">队列统计</div>
        <div class="data-table">
          <div class="data-thead">
            <span>队列</span><span>待处理</span><span>处理中</span><span>失败</span>
          </div>
          <div v-if="!dashboard?.queue?.queue_stats?.length" class="empty-state">暂无数据</div>
          <div v-for="row in dashboard?.queue?.queue_stats" :key="row.queue_name" class="data-row">
            <span>{{ row.queue_name }}</span>
            <span class="badge badge-warning">{{ row.pending_count }}</span>
            <span class="badge badge-info">{{ row.processing_count }}</span>
            <span class="badge badge-error">{{ row.failed_count }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 工作流统计 -->
    <div class="data-card">
      <div class="card-title">工作流统计</div>
      <div class="data-table">
        <div class="data-thead">
          <span>工作流</span><span>执行次数</span><span>成功率</span>
        </div>
        <div v-if="!dashboard?.workflow?.top_workflows?.length" class="empty-state">暂无数据</div>
        <div v-for="row in dashboard?.workflow?.top_workflows" :key="row.workflow_name" class="data-row">
          <span>{{ row.workflow_name }}</span>
          <span>{{ formatNumber(row.execution_count) }}</span>
          <span>
            <span class="badge" :class="getSuccessRateClass(row.success_rate)">
              {{ formatPercent(row.success_rate) }}
            </span>
          </span>
        </div>
      </div>
    </div>

    <!-- 最近活动 -->
    <div class="data-card">
      <div class="card-title">最近活动</div>
      <div v-if="!recentActivity.length" class="empty-state">暂无活动</div>
      <div v-for="activity in recentActivity" :key="activity.id" class="activity-item">
        <div class="activity-dot" :class="getActivityClass(activity.status)" />
        <div class="activity-icon">
          <ActivityIcon :size="14" />
        </div>
        <div class="activity-desc">{{ activity.description }}</div>
        <div class="activity-time">{{ formatTime(activity.timestamp) }}</div>
      </div>
    </div>

    <!-- 更新时间 -->
    <div class="update-time">
      更新时间: {{ formatTime(dashboard?.updated_at) }}
    </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { RefreshCwIcon, ActivityIcon } from 'lucide-vue-next'
import { metricsApi, type DashboardData, type RecentActivity } from '@/api/metrics'

const dashboard = ref<DashboardData | null>(null)
const recentActivity = ref<RecentActivity[]>([])
const loading = ref(false)
let refreshInterval: number | null = null

function formatNumber(num: number | undefined): string {
  if (num === undefined || num === null) return '-'
  return num.toLocaleString('zh-CN')
}

function formatPercent(rate: number | undefined): string {
  if (rate === undefined || rate === null) return '-'
  return (rate * 100).toFixed(1) + '%'
}

function formatTime(time: string | undefined): string {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

function getSuccessRateClass(rate: number): string {
  if (rate >= 0.95) return 'badge-success'
  if (rate >= 0.8) return 'badge-warning'
  return 'badge-error'
}

function getActivityClass(status: string): string {
  const map: Record<string, string> = {
    success: 'dot-success', completed: 'dot-success',
    failed: 'dot-error',
    pending: 'dot-warning',
    processing: 'dot-info',
  }
  return map[status] || 'dot-neutral'
}

async function loadData() {
  loading.value = true
  try {
    const [dashboardRes, activityRes] = await Promise.all([
      metricsApi.getDashboard(),
      metricsApi.getRecentActivity(10),
    ])
    dashboard.value = dashboardRes
    recentActivity.value = activityRes
  } catch (error) {
    console.error('Failed to load metrics:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadData()
  refreshInterval = window.setInterval(loadData, 30000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})
</script>

<style scoped>
.metrics-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 24px;
  height: 100%;
  overflow: hidden;
}

.metrics-content {
  flex: 1;
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

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-secondary);
  transition: background 0.1s;
}

.btn-icon:hover { background: var(--bg-overlay); }

@keyframes spin { to { transform: rotate(360deg); } }
.spinning svg { animation: spin 1s linear infinite; }

/* 概览卡片 */
.stat-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  padding: 20px;
  border-radius: 8px;
  border: 1px solid var(--border);
}

.stat-primary { background: rgba(59,130,246,0.08); border-color: rgba(59,130,246,0.2); }
.stat-warning { background: rgba(234,179,8,0.08); border-color: rgba(234,179,8,0.2); }
.stat-success { background: rgba(34,197,94,0.08); border-color: rgba(34,197,94,0.2); }
.stat-info    { background: rgba(99,102,241,0.08); border-color: rgba(99,102,241,0.2); }

.stat-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.stat-sub {
  font-size: 12px;
  color: var(--text-secondary);
}

/* 表格行 */
.tables-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 16px;
}

.data-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  margin-bottom: 16px;
}

.card-title {
  padding: 12px 16px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border);
  background: var(--bg-overlay);
}

.data-table { overflow-x: auto; }

.data-thead,
.data-row {
  display: grid;
  padding: 10px 16px;
  align-items: center;
  gap: 8px;
}

/* Agent/Workflow 3 cols */
.data-card:nth-child(1) .data-thead,
.data-card:nth-child(1) .data-row {
  grid-template-columns: 1fr 100px 100px;
}

/* Queue 4 cols */
.data-card:nth-child(2) .data-thead,
.data-card:nth-child(2) .data-row {
  grid-template-columns: 1fr 80px 80px 60px;
}

/* Workflow full width 3 cols */
.data-card.wf-card .data-thead,
.data-card.wf-card .data-row {
  grid-template-columns: 1fr 120px 100px;
}

.data-thead {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--bg-overlay);
  border-bottom: 1px solid var(--border);
}

.data-row {
  font-size: 13px;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-subtle);
}

.data-row:last-child { border-bottom: none; }

.mono {
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

.badge-success { background: rgba(34,197,94,0.15);  color: #16a34a; }
.badge-warning { background: rgba(234,179,8,0.15);  color: #ca8a04; }
.badge-error   { background: rgba(239,68,68,0.15);  color: #ef4444; }
.badge-info    { background: rgba(59,130,246,0.15); color: #3b82f6; }

.empty-state {
  padding: 20px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 13px;
}

/* 最近活动 */
.activity-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border-subtle);
}

.activity-item:last-child { border-bottom: none; }

.activity-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.dot-success { background: #16a34a; }
.dot-error   { background: #ef4444; }
.dot-warning { background: #ca8a04; }
.dot-info    { background: #3b82f6; }
.dot-neutral { background: var(--text-tertiary); }

.activity-icon { color: var(--text-secondary); flex-shrink: 0; }

.activity-desc {
  flex: 1;
  font-size: 13px;
  color: var(--text-primary);
}

.activity-time {
  font-size: 12px;
  color: var(--text-tertiary);
  white-space: nowrap;
}

.update-time {
  text-align: center;
  font-size: 12px;
  color: var(--text-tertiary);
  padding: 8px 0 16px;
}
</style>
