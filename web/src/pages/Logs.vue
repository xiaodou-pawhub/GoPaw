<template>
  <div class="logs-page">
    <n-space vertical :size="24">
      <div class="page-header">
        <div class="header-left">
          <n-h2>{{ t('logs.title') }}</n-h2>
          <n-text depth="3">实时监控系统运行状态与错误日志</n-text>
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
            <n-empty description="暂无日志数据" />
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
import { ref, onMounted, onUnmounted, watch } from 'vue'
import {
  NSpace, NH2, NText, NButton, NIcon, NCard, NSwitch, NEmpty
} from 'naive-ui'
import { RefreshOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { getSystemLogs } from '@/api/system'

interface LogEntry {
  raw: string
}

const { t } = useI18n()

const logs = ref<LogEntry[]>([])
const loading = ref(false)
const autoRefresh = ref(true)
let refreshTimer: ReturnType<typeof setInterval> | null = null

// 根据内容判断日志级别并返回样式类
function getLogLevelClass(raw: string) {
  const lower = raw.toLowerCase()
  if (lower.includes('"level":"error"') || lower.includes('error')) return 'level-error'
  if (lower.includes('"level":"warn"') || lower.includes('warn')) return 'level-warn'
  return 'level-info'
}

// 获取系统日志
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

// 设置/清除自动刷新定时器
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
@use '@/styles/variables.scss' as *;

.logs-page {
  padding: $spacing-3;
  animation: fadeIn 0.4s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: $spacing-2;

  :deep(.n-h2) {
    margin: 0 0 $spacing-1;
    font-weight: $font-weight-bold;
    color: $color-text-primary;
  }
}

.logs-card {
  border-radius: $radius-xl;
  box-shadow: $shadow-md;
  overflow: hidden;
  animation: slideUp 0.5s ease-out;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.logs-container {
  height: calc(100vh - 240px);
  background: #1e1e1e;
  color: #d4d4d4;
  padding: $spacing-4;
  overflow-y: auto;
  font-family: $font-family-mono;
  font-size: $font-size-sm;
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-1;
}

.log-item {
  padding: $spacing-1 $spacing-2;
  border-radius: $radius-sm;
  word-break: break-all;
  white-space: pre-wrap;
  border-left: 3px solid transparent;
  transition: all 0.2s ease;

  &:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  &.level-error {
    background: rgba(240, 68, 68, 0.1);
    border-left-color: $color-error;
    color: #ff8888;
  }

  &.level-warn {
    background: rgba(245, 158, 11, 0.1);
    border-left-color: $color-warning;
    color: #fbbf24;
  }

  &.level-info {
    border-left-color: $color-success;
  }
}

.empty-logs {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.auto-refresh-switch {
  margin-right: $spacing-3;
}
</style>
