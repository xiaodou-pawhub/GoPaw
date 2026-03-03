<template>
  <div class="page-container">
    <div class="page-header">
      <div class="header-main">
        <h1 class="page-title">{{ t('logs.title') }}</h1>
        <p class="page-description">实时监控系统运行状态与错误日志</p>
      </div>
      <n-space>
        <n-switch v-model:value="autoRefresh">
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

    <n-card bordered class="page-card" content-style="padding: 0;">
      <div class="logs-container">
        <div v-if="logs.length === 0" class="empty-logs">
          <n-empty :description="t('logs.noLogs')" />
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { NCard, NSwitch, NEmpty, NSpace, NButton, NIcon } from 'naive-ui'
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

const getLogLevelClass = (raw: string) => {
  const lower = raw.toLowerCase()
  if (lower.includes('"level":"error"') || lower.includes('error')) return 'level-error'
  if (lower.includes('"level":"warn"') || lower.includes('warn')) return 'level-warn'
  return 'level-info'
}

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
@use '@/styles/page-layout';

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
</style>