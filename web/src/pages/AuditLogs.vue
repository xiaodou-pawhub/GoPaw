<template>
  <div class="audit-root">
    <!-- 头部 -->
    <div class="audit-header">
      <div class="header-left">
        <h2 class="page-title">审计日志</h2>
        <span class="page-subtitle">记录系统中所有关键操作</span>
      </div>
      <div class="header-right">
        <button class="refresh-btn" @click="loadLogs" :disabled="loading">
          <RefreshCwIcon :size="14" :class="{ spinning: loading }" />
          <span>刷新</span>
        </button>
        <button class="export-btn" @click="showExportModal = true">
          <DownloadIcon :size="14" />
          <span>导出</span>
        </button>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <select v-model="filter.category" class="filter-select" @change="loadLogs">
        <option value="">全部分类</option>
        <option value="auth">认证</option>
        <option value="agent">Agent</option>
        <option value="workflow">工作流</option>
        <option value="knowledge">知识库</option>
        <option value="system">系统</option>
        <option value="user">用户管理</option>
      </select>
      <select v-model="filter.status" class="filter-select" @change="loadLogs">
        <option value="">全部状态</option>
        <option value="success">成功</option>
        <option value="failure">失败</option>
      </select>
      <input
        v-model="filter.user_id"
        class="filter-input"
        placeholder="用户 ID"
        @keyup.enter="loadLogs"
      />
      <input
        v-model="filter.start_time"
        type="datetime-local"
        class="filter-input"
        @change="loadLogs"
      />
      <span class="filter-sep">—</span>
      <input
        v-model="filter.end_time"
        type="datetime-local"
        class="filter-input"
        @change="loadLogs"
      />
      <button class="clear-btn" @click="clearFilter">清除</button>
    </div>

    <!-- 加载 -->
    <div v-if="loading && logs.length === 0" class="audit-loading">
      <div class="loading-spinner" />
      <span>加载审计日志...</span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="logs.length === 0" class="audit-empty">
      <ShieldCheckIcon :size="48" class="empty-icon" />
      <p class="empty-text">暂无审计日志</p>
      <p class="empty-hint">系统操作发生后将自动记录</p>
    </div>

    <!-- 日志表格 -->
    <div v-else class="audit-table">
      <div class="table-header">
        <span class="col-time">时间</span>
        <span class="col-category">分类</span>
        <span class="col-action">操作</span>
        <span class="col-user">用户</span>
        <span class="col-resource">资源</span>
        <span class="col-status">状态</span>
        <span class="col-duration">耗时</span>
      </div>
      <div
        v-for="log in logs"
        :key="log.id"
        class="table-row"
        @click="selectedLog = log"
      >
        <span class="col-time">{{ formatTime(log.timestamp) }}</span>
        <span class="col-category">
          <span class="tag tag-category">{{ log.category }}</span>
        </span>
        <span class="col-action">{{ log.action }}</span>
        <span class="col-user">{{ log.user_id || '—' }}</span>
        <span class="col-resource">
          <span v-if="log.resource_type">{{ log.resource_type }}{{ log.resource_id ? ':' + log.resource_id.slice(0, 8) : '' }}</span>
          <span v-else>—</span>
        </span>
        <span class="col-status">
          <span class="status-badge" :class="log.status === 'success' ? 'status-ok' : 'status-err'">
            {{ log.status === 'success' ? '成功' : '失败' }}
          </span>
        </span>
        <span class="col-duration">{{ log.duration ? log.duration + 'ms' : '—' }}</span>
      </div>
    </div>

    <!-- 分页 -->
    <div v-if="logs.length > 0" class="pagination">
      <button class="page-btn" :disabled="offset === 0" @click="prevPage">上一页</button>
      <span class="page-info">第 {{ Math.floor(offset / limit) + 1 }} 页</span>
      <button class="page-btn" :disabled="logs.length < limit" @click="nextPage">下一页</button>
    </div>

    <!-- 详情弹窗 -->
    <div v-if="selectedLog" class="modal-overlay" @click.self="selectedLog = null">
      <div class="modal-card">
        <div class="modal-head">
          <h3>日志详情</h3>
          <button class="close-btn" @click="selectedLog = null">✕</button>
        </div>
        <div class="detail-grid">
          <div class="detail-row"><span class="detail-label">ID</span><span class="detail-value mono">{{ selectedLog.id }}</span></div>
          <div class="detail-row"><span class="detail-label">时间</span><span class="detail-value">{{ formatTime(selectedLog.timestamp) }}</span></div>
          <div class="detail-row"><span class="detail-label">分类</span><span class="detail-value">{{ selectedLog.category }}</span></div>
          <div class="detail-row"><span class="detail-label">操作</span><span class="detail-value">{{ selectedLog.action }}</span></div>
          <div class="detail-row"><span class="detail-label">用户</span><span class="detail-value">{{ selectedLog.user_id || '—' }}</span></div>
          <div class="detail-row"><span class="detail-label">IP</span><span class="detail-value">{{ selectedLog.user_ip || '—' }}</span></div>
          <div class="detail-row"><span class="detail-label">资源</span><span class="detail-value">{{ selectedLog.resource_type || '—' }} {{ selectedLog.resource_id || '' }}</span></div>
          <div class="detail-row"><span class="detail-label">状态</span><span class="detail-value">{{ selectedLog.status }}</span></div>
          <div class="detail-row"><span class="detail-label">耗时</span><span class="detail-value">{{ selectedLog.duration ? selectedLog.duration + 'ms' : '—' }}</span></div>
          <div v-if="selectedLog.error" class="detail-row"><span class="detail-label">错误</span><span class="detail-value err">{{ selectedLog.error }}</span></div>
        </div>
        <div v-if="selectedLog.details" class="detail-details">
          <div class="detail-label">详情</div>
          <pre class="detail-pre">{{ JSON.stringify(selectedLog.details, null, 2) }}</pre>
        </div>
      </div>
    </div>

    <!-- 导出弹窗 -->
    <div v-if="showExportModal" class="modal-overlay" @click.self="showExportModal = false">
      <div class="modal-card modal-sm">
        <div class="modal-head">
          <h3>导出日志</h3>
          <button class="close-btn" @click="showExportModal = false">✕</button>
        </div>
        <div class="form-group">
          <label>格式</label>
          <select v-model="exportForm.format" class="filter-select w-full">
            <option value="csv">CSV</option>
            <option value="json">JSON</option>
          </select>
        </div>
        <div class="form-group">
          <label>分类（可选）</label>
          <select v-model="exportForm.category" class="filter-select w-full">
            <option value="">全部</option>
            <option value="auth">认证</option>
            <option value="agent">Agent</option>
            <option value="workflow">工作流</option>
            <option value="system">系统</option>
          </select>
        </div>
        <div v-if="exportError" class="error-text">{{ exportError }}</div>
        <div class="modal-actions">
          <button class="btn-ghost" @click="showExportModal = false">取消</button>
          <button class="btn-primary" :disabled="exporting" @click="handleExport">
            {{ exporting ? '导出中...' : '导出' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { RefreshCwIcon, DownloadIcon, ShieldCheckIcon } from 'lucide-vue-next'

interface AuditLog {
  id: string
  timestamp: string
  category: string
  action: string
  user_id: string
  user_ip: string
  resource_type: string
  resource_id: string
  status: string
  details: any
  error: string
  duration: number
  request_id: string
}

const logs = ref<AuditLog[]>([])
const loading = ref(false)
const selectedLog = ref<AuditLog | null>(null)
const showExportModal = ref(false)
const exporting = ref(false)
const exportError = ref('')
const limit = 50
const offset = ref(0)

const filter = ref({
  category: '',
  status: '',
  user_id: '',
  start_time: '',
  end_time: '',
})

const exportForm = ref({ format: 'csv', category: '' })

async function loadLogs() {
  loading.value = true
  try {
    const params: Record<string, any> = {
      limit,
      offset: offset.value,
    }
    if (filter.value.category) params.category = filter.value.category
    if (filter.value.status) params.status = filter.value.status
    if (filter.value.user_id) params.user_id = filter.value.user_id
    if (filter.value.start_time) params.start_time = new Date(filter.value.start_time).toISOString()
    if (filter.value.end_time) params.end_time = new Date(filter.value.end_time).toISOString()

    const res = await axios.get('/api/audit/logs', { params })
    logs.value = res.data?.data ?? res.data ?? []
  } catch {
    logs.value = []
  } finally {
    loading.value = false
  }
}

function clearFilter() {
  filter.value = { category: '', status: '', user_id: '', start_time: '', end_time: '' }
  offset.value = 0
  loadLogs()
}

function prevPage() {
  offset.value = Math.max(0, offset.value - limit)
  loadLogs()
}

function nextPage() {
  offset.value += limit
  loadLogs()
}

function formatTime(ts: string) {
  if (!ts) return '—'
  return new Date(ts).toLocaleString('zh-CN', { hour12: false })
}

async function handleExport() {
  exporting.value = true
  exportError.value = ''
  try {
    const res = await axios.post('/api/audit/export', exportForm.value, { responseType: 'blob' })
    const url = URL.createObjectURL(res.data)
    const a = document.createElement('a')
    a.href = url
    a.download = `audit_logs.${exportForm.value.format}`
    a.click()
    URL.revokeObjectURL(url)
    showExportModal.value = false
  } catch {
    exportError.value = '导出失败，请重试'
  } finally {
    exporting.value = false
  }
}

onMounted(loadLogs)
</script>

<style scoped>
.audit-root {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  padding: 24px 28px;
  gap: 16px;
  box-sizing: border-box;
}

.audit-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.page-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: 12px;
  color: var(--text-tertiary);
}

.header-right {
  display: flex;
  gap: 8px;
}

.refresh-btn,
.export-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-elevated);
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.refresh-btn:hover:not(:disabled),
.export-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-primary);
}

.refresh-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  flex-wrap: wrap;
}

.filter-select,
.filter-input {
  height: 32px;
  padding: 0 10px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 12px;
  outline: none;
}

.filter-input {
  width: 140px;
}

.filter-select:focus,
.filter-input:focus {
  border-color: var(--accent);
}

.filter-sep {
  color: var(--text-tertiary);
  font-size: 12px;
}

.clear-btn {
  padding: 6px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.clear-btn:hover {
  background: var(--bg-overlay);
}

/* table */
.audit-table {
  flex: 1;
  overflow-y: auto;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
}

.table-header,
.table-row {
  display: grid;
  grid-template-columns: 160px 90px 160px 120px 160px 70px 70px;
  align-items: center;
  padding: 10px 14px;
  gap: 8px;
}

.table-header {
  background: var(--bg-overlay);
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
}

.table-row {
  border-bottom: 1px solid var(--border-subtle);
  font-size: 12px;
  color: var(--text-primary);
  cursor: pointer;
  transition: background 0.1s;
}

.table-row:last-child {
  border-bottom: none;
}

.table-row:hover {
  background: var(--bg-overlay);
}

.tag-category {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  color: var(--text-secondary);
}

.status-badge {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 600;
}

.status-ok {
  background: rgba(34, 197, 94, 0.12);
  color: var(--green, #22c55e);
}

.status-err {
  background: rgba(239, 68, 68, 0.12);
  color: var(--red, #ef4444);
}

/* loading / empty */
.audit-loading,
.audit-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-tertiary);
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.empty-icon {
  opacity: 0.3;
}

.empty-text {
  font-size: 14px;
  font-weight: 600;
  margin: 0;
}

.empty-hint {
  font-size: 12px;
  margin: 0;
}

/* pagination */
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  flex-shrink: 0;
}

.page-btn {
  padding: 6px 16px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-elevated);
  color: var(--text-primary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.page-btn:hover:not(:disabled) {
  background: var(--bg-overlay);
}

.page-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.page-info {
  font-size: 12px;
  color: var(--text-secondary);
}

/* modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9000;
}

.modal-card {
  width: 600px;
  max-height: 80vh;
  overflow-y: auto;
  padding: 24px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
}

.modal-sm {
  width: 360px;
}

.modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.modal-head h3 {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  font-size: 14px;
  padding: 2px 6px;
  border-radius: 4px;
}

.close-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-primary);
}

.detail-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.detail-row {
  display: grid;
  grid-template-columns: 80px 1fr;
  gap: 8px;
  font-size: 13px;
}

.detail-label {
  font-size: 11px;
  color: var(--text-tertiary);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding-top: 2px;
}

.detail-value {
  color: var(--text-primary);
  word-break: break-all;
}

.detail-value.mono {
  font-family: monospace;
  font-size: 11px;
  color: var(--text-secondary);
}

.detail-value.err {
  color: var(--red, #ef4444);
}

.detail-details {
  margin-top: 14px;
}

.detail-pre {
  margin: 6px 0 0;
  padding: 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 11px;
  color: var(--text-secondary);
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 200px;
  overflow-y: auto;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 14px;
}

.form-group label {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}

.w-full {
  width: 100%;
  box-sizing: border-box;
}

.error-text {
  font-size: 12px;
  color: var(--red, #ef4444);
  margin-bottom: 10px;
}

.modal-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.btn-primary {
  padding: 8px 16px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-hover);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-ghost {
  padding: 8px 16px;
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-ghost:hover {
  background: var(--bg-overlay);
}
</style>
