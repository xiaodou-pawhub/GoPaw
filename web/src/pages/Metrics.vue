<template>
  <div class="metrics-page">
    <div class="page-header">
      <h1 class="text-h5">性能监控</h1>
      <v-btn
        icon="mdi-refresh"
        variant="text"
        :loading="loading"
        @click="loadData"
      />
    </div>

    <!-- 概览卡片 -->
    <v-row class="mt-4">
      <v-col cols="12" sm="6" md="3">
        <v-card color="primary" variant="tonal">
          <v-card-text>
            <div class="text-overline">Agent 调用</div>
            <div class="text-h4">{{ formatNumber(dashboard?.agent?.call_count) }}</div>
            <div class="text-caption">
              成功率: {{ formatPercent(dashboard?.agent?.success_rate) }}
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" sm="6" md="3">
        <v-card color="warning" variant="tonal">
          <v-card-text>
            <div class="text-overline">队列堆积</div>
            <div class="text-h4">{{ formatNumber(dashboard?.queue?.pending_count) }}</div>
            <div class="text-caption">
              处理中: {{ formatNumber(dashboard?.queue?.processing_count) }}
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" sm="6" md="3">
        <v-card color="success" variant="tonal">
          <v-card-text>
            <div class="text-overline">工作流执行</div>
            <div class="text-h4">{{ formatNumber(dashboard?.workflow?.execution_count) }}</div>
            <div class="text-caption">
              成功率: {{ formatPercent(dashboard?.workflow?.success_rate) }}
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" sm="6" md="3">
        <v-card color="info" variant="tonal">
          <v-card-text>
            <div class="text-overline">系统状态</div>
            <div class="text-h4">{{ formatNumber(dashboard?.system?.memory_mb) }} MB</div>
            <div class="text-caption">
              Goroutines: {{ formatNumber(dashboard?.system?.goroutines) }}
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- 详细统计 -->
    <v-row class="mt-4">
      <!-- Agent 统计 -->
      <v-col cols="12" md="6">
        <v-card>
          <v-card-title>Agent 统计</v-card-title>
          <v-card-text>
            <v-data-table
              :headers="agentHeaders"
              :items="dashboard?.agent?.top_agents || []"
              :items-per-page="5"
              density="compact"
              hide-default-footer
            >
              <template #item.success_rate="{ item }">
                <v-chip
                  :color="getSuccessRateColor(item.success_rate)"
                  size="small"
                >
                  {{ formatPercent(item.success_rate) }}
                </v-chip>
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- 队列统计 -->
      <v-col cols="12" md="6">
        <v-card>
          <v-card-title>队列统计</v-card-title>
          <v-card-text>
            <v-data-table
              :headers="queueHeaders"
              :items="dashboard?.queue?.queue_stats || []"
              :items-per-page="5"
              density="compact"
              hide-default-footer
            >
              <template #item.pending_count="{ item }">
                <v-chip color="warning" size="small">
                  {{ item.pending_count }}
                </v-chip>
              </template>
              <template #item.processing_count="{ item }">
                <v-chip color="info" size="small">
                  {{ item.processing_count }}
                </v-chip>
              </template>
              <template #item.failed_count="{ item }">
                <v-chip color="error" size="small">
                  {{ item.failed_count }}
                </v-chip>
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- 工作流统计 -->
    <v-row class="mt-4">
      <v-col cols="12">
        <v-card>
          <v-card-title>工作流统计</v-card-title>
          <v-card-text>
            <v-data-table
              :headers="workflowHeaders"
              :items="dashboard?.workflow?.top_workflows || []"
              :items-per-page="5"
              density="compact"
              hide-default-footer
            >
              <template #item.success_rate="{ item }">
                <v-chip
                  :color="getSuccessRateColor(item.success_rate)"
                  size="small"
                >
                  {{ formatPercent(item.success_rate) }}
                </v-chip>
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- 最近活动 -->
    <v-row class="mt-4">
      <v-col cols="12">
        <v-card>
          <v-card-title>最近活动</v-card-title>
          <v-card-text>
            <v-timeline density="compact" align="start">
              <v-timeline-item
                v-for="activity in recentActivity"
                :key="activity.id"
                :dot-color="getActivityColor(activity.status)"
                size="small"
              >
                <div class="d-flex align-center">
                  <v-icon :icon="getActivityIcon(activity.type)" class="mr-2" size="small" />
                  <span class="text-body-2">{{ activity.description }}</span>
                  <v-spacer />
                  <span class="text-caption text-grey">{{ formatTime(activity.timestamp) }}</span>
                </div>
              </v-timeline-item>
            </v-timeline>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- 更新时间 -->
    <v-row class="mt-4">
      <v-col cols="12" class="text-center">
        <span class="text-caption text-grey">
          更新时间: {{ formatTime(dashboard?.updated_at) }}
        </span>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { metricsApi, type DashboardData, type RecentActivity } from '@/api/metrics'

const dashboard = ref<DashboardData | null>(null)
const recentActivity = ref<RecentActivity[]>([])
const loading = ref(false)
let refreshInterval: number | null = null

const agentHeaders = [
  { title: 'Agent ID', key: 'agent_id' },
  { title: '调用次数', key: 'call_count' },
  { title: '成功率', key: 'success_rate' },
]

const queueHeaders = [
  { title: '队列', key: 'queue_name' },
  { title: '待处理', key: 'pending_count' },
  { title: '处理中', key: 'processing_count' },
  { title: '失败', key: 'failed_count' },
]

const workflowHeaders = [
  { title: '工作流', key: 'workflow_name' },
  { title: '执行次数', key: 'execution_count' },
  { title: '成功率', key: 'success_rate' },
]

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

function getSuccessRateColor(rate: number): string {
  if (rate >= 0.95) return 'success'
  if (rate >= 0.8) return 'warning'
  return 'error'
}

function getActivityColor(status: string): string {
  const colors: Record<string, string> = {
    success: 'success',
    completed: 'success',
    failed: 'error',
    pending: 'warning',
    processing: 'info',
  }
  return colors[status] || 'grey'
}

function getActivityIcon(type: string): string {
  const icons: Record<string, string> = {
    agent: 'mdi-robot',
    workflow: 'mdi-sitemap',
    queue: 'mdi-queue',
    system: 'mdi-cog',
  }
  return icons[type] || 'mdi-circle'
}

async function loadData() {
  loading.value = true
  try {
    const [dashboardRes, activityRes] = await Promise.all([
      metricsApi.getDashboard(),
      metricsApi.getRecentActivity(10),
    ])
    dashboard.value = dashboardRes.data
    recentActivity.value = activityRes.data
  } catch (error) {
    console.error('Failed to load metrics:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadData()
  // Auto refresh every 30 seconds
  refreshInterval = window.setInterval(loadData, 30000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<style scoped>
.metrics-page {
  padding: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
