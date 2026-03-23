<template>
  <div class="logs-page">
    <div class="logs-header">
      <div>
        <h1 class="logs-title">系统日志</h1>
        <p class="logs-desc">实时监控系统运行状态与错误日志</p>
      </div>
      <div class="header-actions">
        <label class="auto-refresh-toggle">
          <input type="checkbox" v-model="autoRefresh" @change="handleAutoRefreshChange" />
          <span class="toggle-slider" />
          <span>自动刷新</span>
        </label>
        <div class="filter-group">
          <select v-model="levelFilter" class="filter-select">
            <option value="">全部级别</option>
            <option value="debug">DEBUG</option>
            <option value="info">INFO</option>
            <option value="warn">WARN</option>
            <option value="error">ERROR</option>
          </select>
        </div>
        <button class="btn-secondary" @click="loadLogs">
          <RefreshCwIcon :size="13" :class="{ spin: loading }" />
          刷新
        </button>
      </div>
    </div>

    <div class="log-container">
      <div v-if="loading && parsedLogs.length === 0" class="log-loading">
        <LoaderIcon :size="20" class="spin" />
        <span>加载中...</span>
      </div>
      <div v-else-if="parsedLogs.length === 0" class="log-empty">
        <FileTextIcon :size="32" />
        <span>暂无日志数据</span>
      </div>
      <div v-else class="log-list">
        <div v-for="(log, i) in filteredLogs" :key="i" class="log-item" :class="log.level">
          <span class="log-level" :class="log.level">{{ log.level?.toUpperCase() || 'INFO' }}</span>
          <span class="log-time">{{ log.time }}</span>
          <span class="log-msg" :title="log.message">{{ log.message }}</span>
        </div>
      </div>
    </div>

    <div v-if="filteredLogs.length > 0" class="log-footer">
      <span class="log-count">共 {{ filteredLogs.length }} 条日志</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { RefreshCwIcon, LoaderIcon, FileTextIcon } from 'lucide-vue-next'
import { getSystemLogs } from '@/api/system'

interface ParsedLog {
  level: string
  time: string
  message: string
  raw: string
}

const rawLogs = ref<{ raw: string }[]>([])
const loading = ref(false)
const autoRefresh = ref(false)
const levelFilter = ref('')
let refreshTimer: ReturnType<typeof setInterval> | null = null

// 解析日志行
function parseLogLine(raw: string): ParsedLog {
  const result: ParsedLog = {
    level: 'info',
    time: '',
    message: raw,
    raw
  }

  // 尝试解析 JSON 格式日志
  if (raw.startsWith('{')) {
    try {
      const json = JSON.parse(raw)
      result.level = json.level || 'info'
      result.time = json.ts || json.time || ''
      result.message = json.msg || json.message || raw
      return result
    } catch {
      // 解析失败，继续使用正则
    }
  }

  // 解析 console 格式日志
  // 格式: 2026-03-23T15:23:45.123+0800  INFO  main  message
  // 或: 2026/03/23 15:23:45 INFO message
  const consolePattern = /^(\d{4}[-/]\d{2}[-/]\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:[+-]\d{4})?)\s+(\w+)\s+(.*)$/i
  const match = raw.match(consolePattern)
  
  if (match) {
    result.time = match[1]
    result.level = match[2].toLowerCase()
    result.message = match[3]
  } else {
    // 尝试提取级别关键字
    const levelMatch = raw.match(/\b(DEBUG|INFO|WARN|WARNING|ERROR|FATAL)\b/i)
    if (levelMatch) {
      result.level = levelMatch[1].toLowerCase()
    }
  }

  return result
}

// 解析后的日志
const parsedLogs = computed<ParsedLog[]>(() => {
  return rawLogs.value.map(log => parseLogLine(log.raw))
})

// 过滤后的日志
const filteredLogs = computed(() => {
  if (!levelFilter.value) return parsedLogs.value
  return parsedLogs.value.filter(log => log.level === levelFilter.value)
})

async function loadLogs() {
  loading.value = true
  try {
    const res = await getSystemLogs()
    // 处理后端返回的格式: { logs: [{ raw: "..." }] } 或直接是数组
    if (res?.logs && Array.isArray(res.logs)) {
      rawLogs.value = res.logs
    } else if (Array.isArray(res)) {
      rawLogs.value = res
    } else {
      rawLogs.value = []
    }
  } catch {
    rawLogs.value = []
  } finally {
    loading.value = false
  }
}

function handleAutoRefreshChange() {
  if (refreshTimer) { clearInterval(refreshTimer); refreshTimer = null }
  if (autoRefresh.value) {
    refreshTimer = setInterval(loadLogs, 5000)
  }
}

onMounted(loadLogs)
onUnmounted(() => { if (refreshTimer) clearInterval(refreshTimer) })
</script>

<style scoped>
.logs-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 24px 28px;
  height: 100%;
  overflow: hidden;
}

.logs-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.logs-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px;
}

.logs-desc {
  font-size: 12px;
  color: var(--text-secondary);
  margin: 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.auto-refresh-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
}

.auto-refresh-toggle input {
  display: none;
}

.toggle-slider {
  position: relative;
  width: 32px;
  height: 18px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 9px;
  display: inline-block;
  flex-shrink: 0;
  transition: background 0.2s;
}

.toggle-slider::before {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 12px;
  height: 12px;
  background: var(--text-tertiary);
  border-radius: 50%;
  transition: transform 0.2s, background 0.2s;
}

.auto-refresh-toggle input:checked + .toggle-slider {
  background: var(--accent);
  border-color: var(--accent);
}

.auto-refresh-toggle input:checked + .toggle-slider::before {
  transform: translateX(14px);
  background: #fff;
}

.filter-select {
  padding: 6px 10px;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 12px;
  cursor: pointer;
}

.filter-select:focus {
  outline: none;
  border-color: var(--accent);
}

.btn-secondary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-secondary:hover {
  background: var(--bg-overlay);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.log-container {
  flex: 1;
  overflow-y: auto;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px;
  min-height: 0;
}

.log-loading,
.log-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 60px;
  color: var(--text-tertiary);
  font-size: 13px;
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.log-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 8px 10px;
  background: var(--bg-app);
  border-radius: 6px;
  font-family: 'SF Mono', 'JetBrains Mono', Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  border-left: 3px solid transparent;
}

.log-item:hover {
  background: var(--bg-overlay);
}

.log-item.error {
  border-left-color: var(--red);
  background: rgba(239, 68, 68, 0.05);
}

.log-item.warn {
  border-left-color: var(--yellow);
  background: rgba(245, 158, 11, 0.05);
}

.log-item.info {
  border-left-color: var(--accent);
}

.log-item.debug {
  border-left-color: var(--text-tertiary);
}

.log-level {
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  min-width: 48px;
  text-align: center;
  flex-shrink: 0;
}

.log-level.info {
  background: rgba(124, 106, 247, 0.15);
  color: var(--accent);
}

.log-level.warn,
.log-level.warning {
  background: rgba(245, 158, 11, 0.15);
  color: var(--yellow);
}

.log-level.error {
  background: rgba(239, 68, 68, 0.15);
  color: var(--red);
}

.log-level.debug {
  background: var(--bg-overlay);
  color: var(--text-tertiary);
}

.log-time {
  color: var(--text-tertiary);
  font-size: 11px;
  white-space: nowrap;
  flex-shrink: 0;
  min-width: 180px;
}

.log-msg {
  color: var(--text-primary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.log-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 8px;
  flex-shrink: 0;
}

.log-count {
  font-size: 11px;
  color: var(--text-tertiary);
}
</style>