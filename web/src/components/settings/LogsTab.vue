<template>
  <div class="tab-root">
    <div class="tab-header">
      <div>
        <h2 class="tab-title">系统日志</h2>
        <p class="tab-desc">实时监控系统运行状态与错误日志</p>
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
.tab-root { display: flex; flex-direction: column; gap: 20px; height: 100%; }

.tab-header { display: flex; align-items: flex-start; justify-content: space-between; }
.tab-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin: 0 0 4px; }
.tab-desc { font-size: 12px; color: var(--text-secondary); margin: 0; }

.header-actions { display: flex; align-items: center; gap: 10px; }

.auto-refresh-toggle {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
}

.auto-refresh-toggle input { display: none; }

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
  width: 10px;
  height: 10px;
  background: var(--text-tertiary);
  border-radius: 50%;
  top: 2px;
  left: 2px;
  transition: transform 0.2s, background 0.2s;
}

.auto-refresh-toggle input:checked ~ .toggle-slider {
  background: rgba(124, 106, 247, 0.2);
  border-color: var(--accent);
}

.auto-refresh-toggle input:checked ~ .toggle-slider::before {
  transform: translateX(14px);
  background: var(--accent);
}

.log-container {
  flex: 1;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 300px;
}

.log-loading, .log-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
  font-size: 13px;
}

.log-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
  font-family: "SF Mono", "JetBrains Mono", Menlo, monospace;
  font-size: 11px;
}

.log-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 4px 14px;
  transition: background 0.1s;
}

.log-item:hover { background: var(--bg-overlay); }

.log-level {
  flex-shrink: 0;
  width: 40px;
  text-align: right;
  font-weight: 700;
  font-size: 10px;
}

.log-level.error, .log-level.ERROR { color: var(--red); }
.log-level.warn, .log-level.WARN { color: var(--yellow); }
.log-level.info, .log-level.INFO { color: var(--accent); }
.log-level.debug, .log-level.DEBUG { color: var(--text-tertiary); }

.log-time { color: var(--text-tertiary); flex-shrink: 0; }
.log-msg { color: var(--text-secondary); word-break: break-all; }

.btn-secondary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-secondary:hover { background: var(--bg-elevated); color: var(--text-primary); }
</style>
