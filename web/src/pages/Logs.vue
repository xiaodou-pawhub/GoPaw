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
        <button class="btn-secondary" @click="loadLogs">
          <RefreshCwIcon :size="13" />
          刷新
        </button>
      </div>
    </div>

    <div class="log-container">
      <div v-if="loading" class="log-loading">加载中...</div>
      <div v-else-if="logs.length === 0" class="log-empty">暂无日志数据</div>
      <div v-else class="log-list">
        <div v-for="(log, i) in logs" :key="i" class="log-item">
          <span class="log-level" :class="log.level">{{ log.level?.toUpperCase() || 'INFO' }}</span>
          <span class="log-time">{{ log.time }}</span>
          <span class="log-msg">{{ log.message }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { RefreshCwIcon } from 'lucide-vue-next'
import { getSystemLogs } from '@/api/system'

const logs = ref<any[]>([])
const loading = ref(false)
const autoRefresh = ref(false)
let refreshTimer: ReturnType<typeof setInterval> | null = null

async function loadLogs() {
  loading.value = true
  try {
    logs.value = await getSystemLogs()
  } catch {
    // ignore
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
  gap: 20px;
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
  gap: 10px;
}

.auto-refresh-toggle {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
}

.auto-refresh-toggle input {
  display: none;
}

.toggle-slider {
  position: relative;
  width: 30px;
  height: 16px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 8px;
  display: inline-block;
  flex-shrink: 0;
  transition: background 0.2s;
}

.toggle-slider::before {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 10px;
  height: 10px;
  background: var(--text-tertiary);
  border-radius: 50%;
  transition: transform 0.2s, background 0.2s;
}

.auto-refresh-toggle input:checked + .toggle-slider {
  background: var(--accent);
}

.auto-refresh-toggle input:checked + .toggle-slider::before {
  transform: translateX(14px);
  background: #fff;
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

.log-container {
  flex: 1;
  overflow-y: auto;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px;
  min-width: 0;
}

.log-loading,
.log-empty {
  text-align: center;
  padding: 40px;
  color: var(--text-tertiary);
  font-size: 13px;
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.log-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  background: var(--bg-app);
  border-radius: 6px;
  font-family: 'SF Mono', 'JetBrains Mono', Menlo, monospace;
  font-size: 12px;
  line-height: 1.5;
  min-width: 0;
}

.log-level {
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  min-width: 42px;
  text-align: center;
  flex-shrink: 0;
}

.log-level.info {
  background: var(--accent-dim);
  color: var(--accent);
}

.log-level.warn,
.log-level.warning {
  background: var(--yellow-dim);
  color: var(--yellow);
}

.log-level.error {
  background: var(--red-dim);
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
}

.log-msg {
  color: var(--text-primary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
</style>
