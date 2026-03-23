<template>
  <div class="performance-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">性能分析</h1>
        <p class="page-desc">流程执行性能分析与优化建议</p>
      </div>
      <div class="header-right">
        <select v-model="selectedFlowId" class="flow-select" @change="loadAnalysis">
          <option value="">全部流程</option>
          <option v-for="f in flows" :key="f.id" :value="f.id">{{ f.name }}</option>
        </select>
        <button class="btn-secondary" @click="loadAnalysis">
          <RefreshCwIcon :size="16" /> 刷新
        </button>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading-state">
      <LoaderIcon :size="24" class="spin" />
      <span>分析中...</span>
    </div>

    <template v-else-if="analysis">
      <!-- 概览卡片 -->
      <div class="overview-cards">
        <div class="overview-card">
          <div class="card-icon executions">
            <ActivityIcon :size="20" />
          </div>
          <div class="card-content">
            <div class="card-value">{{ analysis.total_traces }}</div>
            <div class="card-label">总执行次数</div>
          </div>
        </div>
        <div class="overview-card">
          <div class="card-icon duration">
            <ClockIcon :size="20" />
          </div>
          <div class="card-content">
            <div class="card-value">{{ formatDuration(analysis.avg_duration) }}</div>
            <div class="card-label">平均耗时</div>
          </div>
        </div>
        <div class="overview-card">
          <div class="card-icon nodes">
            <GitBranchIcon :size="20" />
          </div>
          <div class="card-content">
            <div class="card-value">{{ analysis.node_stats?.length || 0 }}</div>
            <div class="card-label">节点数量</div>
          </div>
        </div>
        <div class="overview-card">
          <div class="card-icon bottlenecks">
            <AlertTriangleIcon :size="20" />
          </div>
          <div class="card-content">
            <div class="card-value">{{ analysis.bottlenecks?.length || 0 }}</div>
            <div class="card-label">性能瓶颈</div>
          </div>
        </div>
      </div>

      <!-- 优化建议 -->
      <div class="recommendations-section" v-if="analysis.recommendations?.length">
        <h3><LightbulbIcon :size="18" /> 优化建议</h3>
        <div class="recommendations-list">
          <div
            v-for="(rec, i) in analysis.recommendations"
            :key="i"
            class="recommendation-item"
          >
            <span class="rec-icon"><ZapIcon :size="14" /></span>
            <span class="rec-text">{{ rec }}</span>
          </div>
        </div>
      </div>

      <div class="analysis-grid">
        <!-- 瓶颈节点 -->
        <div class="analysis-card">
          <h3><AlertTriangleIcon :size="18" /> 性能瓶颈</h3>
          <p class="card-desc">耗时占比超过 20% 的节点</p>
          <div v-if="!analysis.bottlenecks?.length" class="empty-state">
            <CheckCircleIcon :size="24" />
            <span>暂无明显瓶颈</span>
          </div>
          <div v-else class="node-list">
            <div
              v-for="node in analysis.bottlenecks"
              :key="node.node_id"
              class="node-item bottleneck"
            >
              <div class="node-header">
                <span class="node-name">{{ node.node_name || node.node_id }}</span>
                <span class="node-type badge">{{ node.node_type }}</span>
              </div>
              <div class="node-stats">
                <div class="stat">
                  <span class="stat-label">平均耗时</span>
                  <span class="stat-value warning">{{ formatDuration(node.avg_duration) }}</span>
                </div>
                <div class="stat">
                  <span class="stat-label">执行次数</span>
                  <span class="stat-value">{{ node.execution_count }}</span>
                </div>
                <div class="stat">
                  <span class="stat-label">成功率</span>
                  <span class="stat-value" :class="node.success_rate >= 90 ? 'success' : 'error'">
                    {{ node.success_rate.toFixed(1) }}%
                  </span>
                </div>
              </div>
              <div class="duration-bar">
                <div class="bar-fill" :style="{ width: getBottleneckPercent(node) + '%' }"></div>
              </div>
            </div>
          </div>
        </div>

        <!-- 最慢节点 -->
        <div class="analysis-card">
          <h3><ClockIcon :size="18" /> 最慢节点</h3>
          <p class="card-desc">按平均耗时排序</p>
          <div v-if="!analysis.slowest_nodes?.length" class="empty-state">
            <span>暂无数据</span>
          </div>
          <div v-else class="node-list">
            <div
              v-for="(node, i) in analysis.slowest_nodes"
              :key="node.node_id"
              class="node-item"
            >
              <div class="node-rank">{{ i + 1 }}</div>
              <div class="node-info">
                <div class="node-header">
                  <span class="node-name">{{ node.node_name || node.node_id }}</span>
                  <span class="node-type badge">{{ node.node_type }}</span>
                </div>
                <div class="node-metrics">
                  <span class="metric">
                    <ClockIcon :size="12" />
                    {{ formatDuration(node.avg_duration) }}
                  </span>
                  <span class="metric">
                    <RefreshCwIcon :size="12" />
                    {{ node.execution_count }}次
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 错误节点 -->
        <div class="analysis-card">
          <h3><XCircleIcon :size="18" /> 高错误率节点</h3>
          <p class="card-desc">成功率低于 80% 的节点</p>
          <div v-if="!analysis.error_prone_nodes?.length" class="empty-state">
            <CheckCircleIcon :size="24" />
            <span>所有节点运行正常</span>
          </div>
          <div v-else class="node-list">
            <div
              v-for="node in analysis.error_prone_nodes"
              :key="node.node_id"
              class="node-item error"
            >
              <div class="node-header">
                <span class="node-name">{{ node.node_name || node.node_id }}</span>
                <span class="node-type badge">{{ node.node_type }}</span>
              </div>
              <div class="error-stats">
                <div class="error-stat">
                  <span class="error-label">成功</span>
                  <span class="error-value success">{{ node.success_count }}</span>
                </div>
                <div class="error-stat">
                  <span class="error-label">失败</span>
                  <span class="error-value error">{{ node.failed_count }}</span>
                </div>
                <div class="error-stat">
                  <span class="error-label">成功率</span>
                  <span class="error-value error">{{ node.success_rate.toFixed(1) }}%</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 节点统计表 -->
        <div class="analysis-card full-width">
          <h3><BarChart3Icon :size="18" /> 节点性能统计</h3>
          <div class="stats-table">
            <div class="table-header">
              <span>节点</span>
              <span>类型</span>
              <span>执行次数</span>
              <span>平均耗时</span>
              <span>最大耗时</span>
              <span>成功率</span>
              <span>Token</span>
              <span>成本</span>
            </div>
            <div
              v-for="node in analysis.node_stats"
              :key="node.node_id"
              class="table-row"
              :class="{ bottleneck: node.is_bottleneck }"
            >
              <span class="node-name-cell">
                {{ node.node_name || node.node_id }}
                <span v-if="node.is_bottleneck" class="bottleneck-badge">瓶颈</span>
              </span>
              <span><span class="badge">{{ node.node_type }}</span></span>
              <span>{{ node.execution_count }}</span>
              <span :class="{ 'text-warning': node.avg_duration > 5000 }">
                {{ formatDuration(node.avg_duration) }}
              </span>
              <span>{{ formatDuration(node.max_duration) }}</span>
              <span>
                <span class="success-rate" :class="getSuccessRateClass(node.success_rate)">
                  {{ node.success_rate.toFixed(1) }}%
                </span>
              </span>
              <span>{{ formatNumber(node.total_tokens_in + node.total_tokens_out) }}</span>
              <span>${{ node.total_cost.toFixed(4) }}</span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  RefreshCwIcon, LoaderIcon, ActivityIcon, ClockIcon, GitBranchIcon,
  AlertTriangleIcon, LightbulbIcon, ZapIcon, CheckCircleIcon,
  XCircleIcon, BarChart3Icon
} from 'lucide-vue-next'

interface NodeStats {
  node_id: string
  node_name: string
  node_type: string
  execution_count: number
  success_count: number
  failed_count: number
  avg_duration: number
  max_duration: number
  min_duration: number
  total_duration: number
  total_tokens_in: number
  total_tokens_out: number
  total_cost: number
  success_rate: number
  is_bottleneck: boolean
}

interface PerformanceAnalysis {
  flow_id: string
  flow_name: string
  total_traces: number
  avg_duration: number
  total_duration: number
  node_stats: NodeStats[]
  bottlenecks: NodeStats[]
  slowest_nodes: NodeStats[]
  error_prone_nodes: NodeStats[]
  recommendations: string[]
}

interface Flow {
  id: string
  name: string
}

const loading = ref(false)
const selectedFlowId = ref('')
const flows = ref<Flow[]>([])
const analysis = ref<PerformanceAnalysis | null>(null)

async function loadFlows() {
  try {
    const res = await fetch('/api/flows')
    if (res.ok) {
      const data = await res.json()
      flows.value = data || []
    }
  } catch (e) {
    console.error('Failed to load flows:', e)
  }
}

async function loadAnalysis() {
  loading.value = true
  try {
    const url = selectedFlowId.value
      ? `/api/flows/traces/analysis?flow_id=${selectedFlowId.value}`
      : '/api/flows/traces/analysis'
    const res = await fetch(url)
    if (res.ok) {
      analysis.value = await res.json()
    }
  } catch (e) {
    console.error('Failed to load analysis:', e)
  } finally {
    loading.value = false
  }
}

function formatDuration(ms: number): string {
  if (!ms) return '0ms'
  if (ms < 1000) return `${Math.round(ms)}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`
  return `${(ms / 60000).toFixed(2)}min`
}

function formatNumber(n: number): string {
  if (!n) return '0'
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
  return n.toString()
}

function getBottleneckPercent(node: NodeStats): number {
  if (!analysis.value?.total_duration) return 0
  return (node.total_duration / analysis.value.total_duration) * 100
}

function getSuccessRateClass(rate: number): string {
  if (rate >= 95) return 'excellent'
  if (rate >= 80) return 'good'
  if (rate >= 50) return 'warning'
  return 'error'
}

onMounted(() => {
  loadFlows()
  loadAnalysis()
})
</script>

<style scoped>
.performance-page {
  padding: 24px;
  max-width: 1400px;
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

.flow-select {
  padding: 8px 12px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  min-width: 150px;
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

.loading-state {
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

/* 概览卡片 */
.overview-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.overview-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 14px;
}

.card-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-icon.executions { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.card-icon.duration { background: rgba(234, 179, 8, 0.15); color: #ca8a04; }
.card-icon.nodes { background: rgba(139, 92, 246, 0.15); color: #8b5cf6; }
.card-icon.bottlenecks { background: rgba(239, 68, 68, 0.15); color: #ef4444; }

.card-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
}

.card-label {
  font-size: 12px;
  color: var(--text-secondary);
}

/* 优化建议 */
.recommendations-section {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 20px;
  margin-bottom: 24px;
}

.recommendations-section h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 16px;
}

.recommendations-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.recommendation-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px;
  background: rgba(234, 179, 8, 0.1);
  border-radius: 8px;
}

.rec-icon {
  color: #ca8a04;
  flex-shrink: 0;
  margin-top: 2px;
}

.rec-text {
  font-size: 13px;
  color: var(--text-primary);
  line-height: 1.5;
}

/* 分析网格 */
.analysis-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.analysis-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 20px;
}

.analysis-card.full-width {
  grid-column: 1 / -1;
}

.analysis-card h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px;
}

.card-desc {
  font-size: 12px;
  color: var(--text-tertiary);
  margin: 0 0 16px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 30px;
  color: var(--text-tertiary);
}

/* 节点列表 */
.node-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.node-item {
  padding: 12px;
  background: var(--bg-app);
  border-radius: 8px;
  border: 1px solid var(--border-subtle);
}

.node-item.bottleneck {
  border-color: rgba(234, 179, 8, 0.3);
  background: rgba(234, 179, 8, 0.05);
}

.node-item.error {
  border-color: rgba(239, 68, 68, 0.3);
  background: rgba(239, 68, 68, 0.05);
}

.node-rank {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--bg-overlay);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
}

.node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.node-name {
  font-weight: 500;
  color: var(--text-primary);
}

.node-type {
  font-size: 11px;
  background: var(--bg-overlay);
  padding: 2px 6px;
  border-radius: 4px;
}

.node-stats {
  display: flex;
  gap: 16px;
}

.stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stat-label {
  font-size: 11px;
  color: var(--text-tertiary);
}

.stat-value {
  font-size: 13px;
  font-weight: 600;
}

.stat-value.warning { color: #ca8a04; }
.stat-value.success { color: #16a34a; }
.stat-value.error { color: #ef4444; }

.duration-bar {
  height: 4px;
  background: var(--bg-overlay);
  border-radius: 2px;
  margin-top: 10px;
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  background: #ca8a04;
  border-radius: 2px;
}

.node-metrics {
  display: flex;
  gap: 12px;
  margin-top: 4px;
}

.metric {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--text-secondary);
}

.error-stats {
  display: flex;
  gap: 16px;
}

.error-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.error-label {
  font-size: 11px;
  color: var(--text-tertiary);
}

.error-value {
  font-size: 13px;
  font-weight: 600;
}

.error-value.success { color: #16a34a; }
.error-value.error { color: #ef4444; }

/* 统计表 */
.stats-table {
  overflow-x: auto;
}

.table-header, .table-row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1fr 1fr 1fr 1fr;
  padding: 10px 12px;
  align-items: center;
  gap: 8px;
}

.table-header {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  background: var(--bg-overlay);
  border-radius: 6px;
}

.table-row {
  font-size: 13px;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-subtle);
}

.table-row:last-child {
  border-bottom: none;
}

.table-row.bottleneck {
  background: rgba(234, 179, 8, 0.05);
}

.node-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.bottleneck-badge {
  font-size: 10px;
  padding: 1px 6px;
  background: rgba(234, 179, 8, 0.2);
  color: #ca8a04;
  border-radius: 4px;
}

.text-warning {
  color: #ca8a04;
}

.success-rate {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
}

.success-rate.excellent { background: rgba(34, 197, 94, 0.15); color: #16a34a; }
.success-rate.good { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.success-rate.warning { background: rgba(234, 179, 8, 0.15); color: #ca8a04; }
.success-rate.error { background: rgba(239, 68, 68, 0.15); color: #ef4444; }

.badge {
  display: inline-block;
  padding: 2px 6px;
  background: var(--bg-overlay);
  border-radius: 4px;
  font-size: 11px;
}

/* 响应式 */
@media (max-width: 768px) {
  .overview-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .analysis-grid {
    grid-template-columns: 1fr;
  }

  .table-header, .table-row {
    grid-template-columns: 2fr 1fr 1fr 1fr;
  }
}
</style>