<template>
  <div class="cost-report-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">成本统计</h1>
        <p class="page-desc">流程执行成本分析与统计</p>
      </div>
      <div class="header-right">
        <select v-model="periodDays" class="period-select" @change="loadReport">
          <option :value="7">最近 7 天</option>
          <option :value="14">最近 14 天</option>
          <option :value="30">最近 30 天</option>
        </select>
        <button class="btn-secondary" @click="loadReport">
          <RefreshCwIcon :size="16" /> 刷新
        </button>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading-state">
      <LoaderIcon :size="24" class="spin" />
      <span>加载中...</span>
    </div>

    <template v-else-if="report">
      <!-- 概览卡片 -->
      <div class="overview-cards">
        <div class="overview-card highlight">
          <div class="card-icon cost">
            <DollarSignIcon :size="20" />
          </div>
          <div class="card-content">
            <div class="card-value">${{ report.total_cost.toFixed(4) }}</div>
            <div class="card-label">总成本</div>
          </div>
        </div>
        <div class="overview-card">
          <div class="card-icon tokens">
            <CpuIcon :size="20" />
          </div>
          <div class="card-content">
            <div class="card-value">{{ formatNumber(report.total_tokens) }}</div>
            <div class="card-label">总 Token</div>
          </div>
        </div>
        <div class="overview-card">
          <div class="card-icon executions">
            <ActivityIcon :size="20" />
          </div>
          <div class="card-content">
            <div class="card-value">{{ report.executions }}</div>
            <div class="card-label">执行次数</div>
          </div>
        </div>
        <div class="overview-card">
          <div class="card-icon avg">
            <TrendingUpIcon :size="20" />
          </div>
          <div class="card-content">
            <div class="card-value">${{ report.avg_cost_per_run.toFixed(4) }}</div>
            <div class="card-label">平均成本/次</div>
          </div>
        </div>
      </div>

      <!-- Token 分布 -->
      <div class="token-stats">
        <h3><CpuIcon :size="18" /> Token 分布</h3>
        <div class="token-bars">
          <div class="token-bar">
            <span class="token-label">输入 Token</span>
            <div class="bar-container">
              <div class="bar-fill input" :style="{ width: getTokenPercent(report.tokens_in) + '%' }"></div>
            </div>
            <span class="token-value">{{ formatNumber(report.tokens_in) }}</span>
          </div>
          <div class="token-bar">
            <span class="token-label">输出 Token</span>
            <div class="bar-container">
              <div class="bar-fill output" :style="{ width: getTokenPercent(report.tokens_out) + '%' }"></div>
            </div>
            <span class="token-value">{{ formatNumber(report.tokens_out) }}</span>
          </div>
        </div>
      </div>

      <div class="report-grid">
        <!-- 按流程统计 -->
        <div class="report-card">
          <h3><GitBranchIcon :size="18" /> 流程成本排行</h3>
          <div v-if="!report.by_flow?.length" class="empty-state">暂无数据</div>
          <div v-else class="cost-list">
            <div v-for="f in report.by_flow" :key="f.flow_id" class="cost-item">
              <div class="item-info">
                <span class="item-name">{{ f.flow_name || f.flow_id }}</span>
                <span class="item-meta">{{ f.executions }} 次执行</span>
              </div>
              <div class="item-cost">
                <span class="cost-value">${{ f.total_cost.toFixed(4) }}</span>
                <span class="cost-tokens">{{ formatNumber(f.total_tokens) }} tokens</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 按模型统计 -->
        <div class="report-card">
          <h3><BoxIcon :size="18" /> 模型成本排行</h3>
          <div v-if="!report.by_model?.length" class="empty-state">暂无数据</div>
          <div v-else class="cost-list">
            <div v-for="m in report.by_model" :key="m.model" class="cost-item">
              <div class="item-info">
                <span class="item-name">{{ m.model }}</span>
                <span class="item-meta">{{ m.calls }} 次调用</span>
              </div>
              <div class="item-cost">
                <span class="cost-value">${{ m.total_cost.toFixed(4) }}</span>
                <span class="cost-tokens">{{ formatNumber(m.tokens_in + m.tokens_out) }} tokens</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 按节点类型统计 -->
        <div class="report-card">
          <h3><LayersIcon :size="18" /> 节点类型成本</h3>
          <div v-if="!report.by_node?.length" class="empty-state">暂无数据</div>
          <div v-else class="cost-list">
            <div v-for="n in report.by_node" :key="n.node_type" class="cost-item">
              <div class="item-info">
                <span class="item-name">{{ getNodeTypeLabel(n.node_type) }}</span>
                <span class="item-meta">{{ n.executions }} 次执行</span>
              </div>
              <div class="item-cost">
                <span class="cost-value">${{ n.total_cost.toFixed(4) }}</span>
                <span class="cost-tokens">{{ formatNumber(n.total_tokens) }} tokens</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 每日趋势 -->
        <div class="report-card full-width">
          <h3><BarChart3Icon :size="18" /> 每日趋势</h3>
          <div v-if="!report.daily_trend?.length" class="empty-state">暂无数据</div>
          <div v-else class="trend-chart">
            <div class="chart-header">
              <span>日期</span>
              <span>执行</span>
              <span>成本</span>
              <span>Token</span>
            </div>
            <div class="chart-body">
              <div v-for="d in report.daily_trend" :key="d.date" class="chart-row">
                <span class="chart-date">{{ formatDate(d.date) }}</span>
                <span class="chart-exec">{{ d.total }}</span>
                <span class="chart-cost">${{ d.total_cost.toFixed(4) }}</span>
                <span class="chart-tokens">{{ formatNumber(d.total_tokens) }}</span>
                <div class="chart-bar">
                  <div class="bar" :style="{ width: getBarWidth(d.total_cost) + '%' }"></div>
                </div>
              </div>
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
  RefreshCwIcon, LoaderIcon, DollarSignIcon, CpuIcon, ActivityIcon,
  TrendingUpIcon, GitBranchIcon, BoxIcon, LayersIcon, BarChart3Icon
} from 'lucide-vue-next'

interface FlowCost {
  flow_id: string
  flow_name: string
  executions: number
  total_cost: number
  total_tokens: number
  avg_cost_per_run: number
}

interface ModelCost {
  model: string
  calls: number
  total_cost: number
  tokens_in: number
  tokens_out: number
  avg_cost_per_call: number
}

interface NodeCost {
  node_type: string
  executions: number
  total_cost: number
  total_tokens: number
}

interface DailyStats {
  date: string
  total: number
  completed: number
  failed: number
  avg_duration: number
  total_tokens: number
  total_cost: number
}

interface CostReport {
  period: string
  total_cost: number
  total_tokens: number
  tokens_in: number
  tokens_out: number
  executions: number
  avg_cost_per_run: number
  by_flow: FlowCost[]
  by_model: ModelCost[]
  by_node: NodeCost[]
  daily_trend: DailyStats[]
}

const loading = ref(false)
const periodDays = ref(7)
const report = ref<CostReport | null>(null)

async function loadReport() {
  loading.value = true
  try {
    const res = await fetch(`/api/flows/traces/cost?days=${periodDays.value}`)
    if (res.ok) {
      report.value = await res.json()
    }
  } catch (e) {
    console.error('Failed to load cost report:', e)
  } finally {
    loading.value = false
  }
}

function formatNumber(n: number): string {
  if (!n) return '0'
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
  return n.toString()
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

function getTokenPercent(value: number): number {
  if (!report.value?.total_tokens) return 0
  return (value / report.value.total_tokens) * 100
}

function getBarWidth(cost: number): number {
  if (!report.value?.daily_trend?.length) return 0
  const maxCost = Math.max(...report.value.daily_trend.map(d => d.total_cost))
  if (!maxCost) return 0
  return (cost / maxCost) * 100
}

function getNodeTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    start: '开始节点',
    agent: 'Agent 节点',
    human: '人工节点',
    condition: '条件分支',
    parallel: '并行执行',
    loop: '循环执行',
    subflow: '子流程',
    webhook: 'Webhook',
    end: '结束节点',
  }
  return labels[type] || type
}

onMounted(() => {
  loadReport()
})
</script>

<style scoped>
.cost-report-page {
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

.period-select {
  padding: 8px 12px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
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

.overview-card.highlight {
  background: linear-gradient(135deg, rgba(34, 197, 94, 0.1), rgba(34, 197, 94, 0.05));
  border-color: rgba(34, 197, 94, 0.3);
}

.card-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-icon.cost { background: rgba(34, 197, 94, 0.15); color: #16a34a; }
.card-icon.tokens { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.card-icon.executions { background: rgba(139, 92, 246, 0.15); color: #8b5cf6; }
.card-icon.avg { background: rgba(234, 179, 8, 0.15); color: #ca8a04; }

.card-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
}

.card-label {
  font-size: 12px;
  color: var(--text-secondary);
}

/* Token 统计 */
.token-stats {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 20px;
  margin-bottom: 24px;
}

.token-stats h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 16px;
}

.token-bars {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.token-bar {
  display: flex;
  align-items: center;
  gap: 12px;
}

.token-label {
  width: 80px;
  font-size: 13px;
  color: var(--text-secondary);
}

.bar-container {
  flex: 1;
  height: 8px;
  background: var(--bg-overlay);
  border-radius: 4px;
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s;
}

.bar-fill.input { background: #3b82f6; }
.bar-fill.output { background: #8b5cf6; }

.token-value {
  width: 80px;
  text-align: right;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

/* 报表网格 */
.report-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.report-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 20px;
}

.report-card.full-width {
  grid-column: 1 / -1;
}

.report-card h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 16px;
}

.empty-state {
  padding: 30px;
  text-align: center;
  color: var(--text-tertiary);
}

.cost-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.cost-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px;
  background: var(--bg-app);
  border-radius: 6px;
}

.item-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.item-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.item-meta {
  font-size: 11px;
  color: var(--text-tertiary);
}

.item-cost {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

.cost-value {
  font-size: 14px;
  font-weight: 600;
  color: #16a34a;
}

.cost-tokens {
  font-size: 11px;
  color: var(--text-tertiary);
}

/* 趋势图 */
.trend-chart {
  overflow-x: auto;
}

.chart-header {
  display: grid;
  grid-template-columns: 80px 60px 80px 80px 1fr;
  gap: 8px;
  padding: 8px 0;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  border-bottom: 1px solid var(--border);
}

.chart-body {
  max-height: 300px;
  overflow-y: auto;
}

.chart-row {
  display: grid;
  grid-template-columns: 80px 60px 80px 80px 1fr;
  gap: 8px;
  padding: 10px 0;
  align-items: center;
  border-bottom: 1px solid var(--border-subtle);
}

.chart-date {
  font-size: 13px;
  color: var(--text-primary);
}

.chart-exec {
  font-size: 13px;
  color: var(--text-secondary);
}

.chart-cost {
  font-size: 13px;
  font-weight: 600;
  color: #16a34a;
}

.chart-tokens {
  font-size: 12px;
  color: var(--text-secondary);
}

.chart-bar {
  height: 6px;
  background: var(--bg-overlay);
  border-radius: 3px;
  overflow: hidden;
}

.chart-bar .bar {
  height: 100%;
  background: linear-gradient(90deg, #16a34a, #22c55e);
  border-radius: 3px;
}

/* 响应式 */
@media (max-width: 768px) {
  .overview-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .report-grid {
    grid-template-columns: 1fr;
  }
}
</style>