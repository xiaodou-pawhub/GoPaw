<template>
  <div class="logs-page">
    <n-space vertical :size="24">
      <div class="page-header">
        <div class="header-left">
          <n-h2>{{ t('logs.title') }}</n-h2>
          <n-text depth="3">实时监控系统运行状态与错误日志 / Real-time system monitoring and error logs</n-text>
        </div>
        <n-space>
          <n-switch v-model:value="autoRefresh" class="auto-refresh-switch">
            <template #checked>
              {{ t('logs.autoRefresh') }}
            </template>
            <template #unchecked>
              {{ t('logs.autoRefresh') }}
            </template>
          </n-switch>
          <n-button :loading="loading" @click="fetchLogs">
            <template #icon>
              <n-icon :component="RefreshOutline" />
            </template>
            {{ t('logs.refresh') }}
          </n-button>
        </n-space>
      </div>

      <n-card bordered class="logs-card" content-style="padding: 0;">
        <div class="logs-container">
          <div v-if="logs.length === 0" class="empty-logs">
            <n-empty description="暂无日志数据 / No logs available" />
          </div>
          <div v-else class="log-list">
            <div
              v-for="(log, index) in logs"
              :key="index"
              class="log-item"
              :class="getLogLevelClass(log.raw)"
            >
              <span class="log-raw">{{ log.raw }}</span>
            </div>
          </div>
        </div>
      </n-card>
    </n-space>
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, onMounted, onUnmounted, watch } from 'vue'
import {
  NSpace, NH2, NText, NButton, NIcon, NCard, NSwitch, NEmpty, useMessage
} from 'naive-ui'
import { RefreshOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { getSystemLogs } from '@/api/system'

const { t } = useI18n()
const message = useMessage()

const logs = ref<any[]>([])
const loading = ref(false)
const autoRefresh = ref(true)
let refreshTimer: ReturnType<typeof setInterval> | null = null

// 中文：根据内容判断日志级别并返回样式类
// English: Determine log level from content and return style class
function getLogLevelClass(raw: string) {
  const lower = raw.toLowerCase()
  if (lower.includes('"level":"error"') || lower.includes('error')) return 'level-error'
  if (lower.includes('"level":"warn"') || lower.includes('warn')) return 'level-warn'
  return 'level-info'
}

// 中文：获取系统日志
// English: Fetch system logs
async function fetchLogs() {
  loading.value = true
  try {
    const res = await getSystemLogs()
    logs.value = res.logs || []
  } catch (error) {
    console.error('Failed to fetch logs:', error)
  } finally {
    loading.value = false
  }
}

// 中文：设置/清除自动刷新定时器
// English: Set/Clear auto refresh timer
function toggleAutoRefresh(enabled: boolean) {
  if (enabled) {
    refreshTimer = setInterval(fetchLogs, 5000)
  } else if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

watch(autoRefresh, (newVal) => {
  toggleAutoRefresh(newVal)
})

onMounted(() => {
  fetchLogs()
  toggleAutoRefresh(autoRefresh.value)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<style scoped lang="scss">
.logs-page {
  padding: 12px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 8px;
}

.logs-card {
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.03);
  overflow: hidden;
}

.logs-container {
  height: calc(100vh - 240px);
  background: #1e1e1e; // 中文：代码风格背景 / English: Code-style background
  color: #d4d4d4;
  padding: 16px;
  overflow-y: auto;
  font-family: 'Fira Code', 'Courier New', Courier, monospace;
  font-size: 13px;
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.log-item {
  padding: 4px 8px;
  border-radius: 4px;
  word-break: break-all;
  white-space: pre-wrap;
  border-left: 3px solid transparent;

  &.level-error {
    background: rgba(240, 68, 68, 0.1);
    border-left-color: #f04444;
    color: #ff8888;
  }

  &.level-warn {
    background: rgba(245, 158, 11, 0.1);
    border-left-color: #f59e0b;
    color: #fbbf24;
  }

  &.level-info {
    border-left-color: #18a058;
  }
}

.empty-logs {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.auto-refresh-switch {
  margin-right: 12px;
}
</style>
