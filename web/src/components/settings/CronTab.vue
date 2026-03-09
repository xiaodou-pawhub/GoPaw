<template>
  <div class="tab-root">
    <div class="tab-header">
      <div>
        <h2 class="tab-title">定时任务</h2>
        <p class="tab-desc">自动化执行 Agent 任务，支持定时推送至不同频道</p>
      </div>
      <button class="btn-primary" @click="openModal('create')">
        <PlusIcon :size="13" /> 新增任务
      </button>
    </div>

    <div class="cron-table">
      <div v-if="loading" class="table-loading">加载中...</div>
      <div v-else-if="jobs.length === 0" class="table-empty">暂无定时任务</div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>任务名称</th>
            <th>Cron 表达式</th>
            <th>频道</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="job in jobs" :key="job.id">
            <td>{{ job.name }}</td>
            <td><code class="code-badge">{{ job.cron_expr }}</code></td>
            <td>{{ job.channel }}</td>
            <td>
              <span class="status-badge" :class="job.enabled ? 'enabled' : 'disabled'">
                {{ job.enabled ? '启用' : '禁用' }}
              </span>
            </td>
            <td>
              <div class="table-actions">
                <button class="action-btn" title="立即执行" @click="handleTrigger(job)"><PlayIcon :size="12" /></button>
                <button class="action-btn" title="执行历史" @click="openHistory(job)"><ClockIcon :size="12" /></button>
                <button class="action-btn" title="编辑" @click="openModal('edit', job)"><PencilIcon :size="12" /></button>
                <button class="action-btn danger" title="删除" @click="handleDelete(job)"><TrashIcon :size="12" /></button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 新增/编辑弹窗 -->
    <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
      <div class="modal-card">
        <div class="modal-header">
          <h3 class="modal-title">{{ modalType === 'create' ? '新增任务' : '编辑任务' }}</h3>
          <button class="icon-close" @click="showModal = false"><XIcon :size="16" /></button>
        </div>
        <div class="modal-body">
          <div class="form-field">
            <label>任务名称 *</label>
            <input v-model="formModel.name" placeholder="请输入任务名称" class="form-input" />
          </div>
          <div class="form-field">
            <label>Cron 表达式 *</label>
            <input v-model="formModel.cron_expr" placeholder="例如: 0 9 * * 1-5" class="form-input" />
            <span class="form-tip">例如 "0 9 * * 1-5" 表示工作日 9 点</span>
          </div>
          <div class="form-field">
            <label>频道 *</label>
            <select v-model="formModel.channel" class="form-select">
              <option v-for="opt in channelOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
            </select>
          </div>
          <div class="form-field">
            <label>触发提示词 *</label>
            <textarea v-model="formModel.prompt" placeholder="触发时发给 Agent 的内容" rows="3" class="form-textarea" />
          </div>
          <div class="form-field inline">
            <label class="toggle">
              <input type="checkbox" v-model="formModel.enabled" />
              <span class="toggle-slider" />
            </label>
            <span class="toggle-label">启用任务</span>
          </div>
          <div class="form-row">
            <div class="form-field">
              <label>活跃开始时间</label>
              <input v-model="formModel.active_from" type="time" class="form-input" />
            </div>
            <div class="form-field">
              <label>活跃结束时间</label>
              <input v-model="formModel.active_until" type="time" class="form-input" />
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-secondary" @click="showModal = false">取消</button>
          <button class="btn-primary" :disabled="submitting" @click="handleSubmit">
            {{ submitting ? '保存中...' : '确认' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 执行历史侧边栏 -->
    <div v-if="showHistory" class="history-drawer" @click.self="showHistory = false">
      <div class="drawer-card">
        <div class="drawer-header">
          <h3 class="drawer-title">执行历史 - {{ currentJob?.name }}</h3>
          <button class="icon-close" @click="showHistory = false"><XIcon :size="16" /></button>
        </div>
        <div class="drawer-body">
          <div v-if="historyLoading" class="history-loading">加载中...</div>
          <div v-else-if="runHistory.length === 0" class="history-empty">暂无执行记录</div>
          <div v-else class="history-list">
            <div v-for="run in runHistory" :key="run.id" class="history-item">
              <div class="history-header">
                <span class="run-status" :class="run.status">{{ run.status.toUpperCase() }}</span>
                <span class="run-time">{{ formatTime(run.triggered_at) }}</span>
              </div>
              <div v-if="run.output" class="run-output">{{ run.output }}</div>
              <div v-if="run.error_msg" class="run-error">{{ run.error_msg }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { PlusIcon, PencilIcon, TrashIcon, PlayIcon, ClockIcon, XIcon } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { getCronJobs, createCronJob, updateCronJob, deleteCronJob, triggerCronJob, getCronRunHistory } from '@/api/cron'
import type { CronJob, CronRun } from '@/types'

const jobs = ref<CronJob[]>([])
const loading = ref(false)
const submitting = ref(false)
const showModal = ref(false)
const modalType = ref<'create' | 'edit'>('create')
const editingId = ref<string | null>(null)

const showHistory = ref(false)
const currentJob = ref<CronJob | null>(null)
const runHistory = ref<CronRun[]>([])
const historyLoading = ref(false)

const formModel = reactive<Partial<CronJob>>({
  name: '', cron_expr: '', channel: 'console', prompt: '',
  enabled: true, active_from: null, active_until: null
})

const channelOptions = [
  { label: '控制台 (Console)', value: 'console' },
  { label: '飞书 (Feishu)', value: 'feishu' },
  { label: '钉钉 (DingTalk)', value: 'dingtalk' },
  { label: 'Webhook', value: 'webhook' }
]

async function loadJobs() {
  loading.value = true
  try { jobs.value = await getCronJobs() } catch {}
  finally { loading.value = false }
}

function openModal(type: 'create' | 'edit', job?: CronJob) {
  modalType.value = type
  if (type === 'edit' && job) {
    editingId.value = job.id
    Object.assign(formModel, { ...job, active_from: job.active_from || null, active_until: job.active_until || null })
  } else {
    editingId.value = null
    Object.assign(formModel, { name: '', cron_expr: '', channel: 'console', prompt: '', enabled: true, active_from: null, active_until: null })
  }
  showModal.value = true
}

async function handleSubmit() {
  if (!formModel.name || !formModel.cron_expr || !formModel.channel || !formModel.prompt) {
    toast.error('请填写必填字段')
    return
  }
  submitting.value = true
  try {
    const payload = { ...formModel }
    if (!payload.active_from) payload.active_from = ''
    if (!payload.active_until) payload.active_until = ''
    if (modalType.value === 'create') await createCronJob(payload)
    else if (editingId.value) await updateCronJob(editingId.value, payload)
    toast.success('保存成功')
    showModal.value = false
    loadJobs()
  } catch (err: any) {
    toast.error(err.response?.data?.error || '保存失败')
  } finally {
    submitting.value = false
  }
}

async function handleTrigger(job: CronJob) {
  if (!confirm(`确认立即执行任务 "${job.name}" 吗？`)) return
  try {
    await triggerCronJob(job.id)
    toast.success('已触发执行请求')
  } catch {
    toast.error('触发失败')
  }
}

async function handleDelete(job: CronJob) {
  if (!confirm(`删除任务 "${job.name}" 后无法恢复，是否继续？`)) return
  try {
    await deleteCronJob(job.id)
    toast.success('已删除')
    loadJobs()
  } catch {
    toast.error('删除失败')
  }
}

async function openHistory(job: CronJob) {
  currentJob.value = job
  showHistory.value = true
  historyLoading.value = true
  try { runHistory.value = await getCronRunHistory(job.id) }
  catch { toast.error('加载历史失败') }
  finally { historyLoading.value = false }
}

function formatTime(ts: number) { return new Date(ts * 1000).toLocaleString() }

onMounted(loadJobs)
</script>

<style scoped>
.tab-root { display: flex; flex-direction: column; gap: 20px; }

.tab-header { display: flex; align-items: flex-start; justify-content: space-between; }
.tab-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin: 0 0 4px; }
.tab-desc { font-size: 12px; color: var(--text-secondary); margin: 0; }

.cron-table {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
}

.table-loading, .table-empty {
  padding: 32px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 13px;
}

.table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.table th {
  padding: 10px 14px;
  text-align: left;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  border-bottom: 1px solid var(--border);
  background: var(--bg-panel);
}

.table td {
  padding: 10px 14px;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-subtle);
}

.table tr:last-child td { border-bottom: none; }

.code-badge {
  font-family: monospace;
  font-size: 11px;
  background: var(--bg-overlay);
  padding: 2px 6px;
  border-radius: 4px;
  color: var(--accent);
}

.status-badge {
  font-size: 10px;
  padding: 2px 7px;
  border-radius: 4px;
  font-weight: 600;
}

.status-badge.enabled { background: rgba(34, 197, 94, 0.1); color: var(--green); }
.status-badge.disabled { background: var(--bg-overlay); color: var(--text-tertiary); }

.table-actions { display: flex; gap: 2px; }

.action-btn {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}

.action-btn:hover { background: var(--bg-overlay); color: var(--text-secondary); }
.action-btn.danger:hover { color: var(--red); }

/* Modal */
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.65); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal-card { width: 520px; max-height: 90vh; background: var(--bg-elevated); border: 1px solid var(--border); border-radius: 12px; display: flex; flex-direction: column; overflow: hidden; }
.modal-header { display: flex; align-items: center; justify-content: space-between; padding: 16px 20px; border-bottom: 1px solid var(--border-subtle); }
.modal-title { font-size: 14px; font-weight: 600; color: var(--text-primary); margin: 0; }
.icon-close { background: transparent; border: none; color: var(--text-tertiary); cursor: pointer; display: flex; align-items: center; padding: 4px; border-radius: 4px; }
.icon-close:hover { color: var(--text-primary); background: var(--bg-overlay); }
.modal-body { flex: 1; overflow-y: auto; padding: 16px 20px; display: flex; flex-direction: column; gap: 14px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 20px; border-top: 1px solid var(--border-subtle); }

.form-field { display: flex; flex-direction: column; gap: 5px; }
.form-field label { font-size: 12px; font-weight: 500; color: var(--text-secondary); }
.form-field.inline { flex-direction: row; align-items: center; gap: 8px; }
.toggle-label { font-size: 12px; color: var(--text-secondary); }
.form-row { display: flex; gap: 10px; }

.form-input, .form-select, .form-textarea {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  box-sizing: border-box;
  transition: border-color 0.15s;
  font-family: inherit;
}

.form-textarea { resize: vertical; min-height: 72px; }
.form-input:focus, .form-select:focus, .form-textarea:focus { border-color: var(--accent); }
.form-tip { font-size: 11px; color: var(--text-tertiary); }

/* Toggle */
.toggle { position: relative; width: 34px; height: 18px; cursor: pointer; display: block; flex-shrink: 0; }
.toggle input { display: none; }
.toggle-slider { position: absolute; inset: 0; background: var(--bg-overlay); border: 1px solid var(--border); border-radius: 9px; transition: background 0.2s; }
.toggle-slider::before { content: ''; position: absolute; width: 12px; height: 12px; background: var(--text-tertiary); border-radius: 50%; top: 2px; left: 2px; transition: transform 0.2s, background 0.2s; }
.toggle input:checked + .toggle-slider { background: rgba(124, 106, 247, 0.2); border-color: var(--accent); }
.toggle input:checked + .toggle-slider::before { transform: translateX(16px); background: var(--accent); }

/* History drawer */
.history-drawer { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: stretch; justify-content: flex-end; z-index: 1000; }
.drawer-card { width: 440px; height: 100%; background: var(--bg-elevated); border-left: 1px solid var(--border); display: flex; flex-direction: column; }
.drawer-header { display: flex; align-items: center; justify-content: space-between; padding: 16px 20px; border-bottom: 1px solid var(--border-subtle); }
.drawer-title { font-size: 14px; font-weight: 600; color: var(--text-primary); margin: 0; }
.drawer-body { flex: 1; overflow-y: auto; padding: 16px; }

.history-loading, .history-empty { text-align: center; color: var(--text-tertiary); font-size: 13px; padding: 24px; }

.history-list { display: flex; flex-direction: column; gap: 10px; }

.history-item {
  background: var(--bg-panel);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  padding: 10px 12px;
}

.history-header { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }

.run-status { font-size: 10px; font-weight: 700; padding: 2px 6px; border-radius: 3px; }
.run-status.success { background: rgba(34,197,94,0.1); color: var(--green); }
.run-status.error { background: rgba(239,68,68,0.1); color: var(--red); }
.run-status.running { background: rgba(124,106,247,0.1); color: var(--accent); }

.run-time { font-size: 11px; color: var(--text-tertiary); margin-left: auto; }

.run-output, .run-error {
  font-family: monospace;
  font-size: 11px;
  background: var(--bg-app);
  border-radius: 4px;
  padding: 6px 8px;
  white-space: pre-wrap;
  word-break: break-all;
}

.run-error { color: var(--red); }

.btn-primary { display: flex; align-items: center; gap: 6px; padding: 7px 14px; background: var(--accent); border: none; border-radius: 6px; color: #fff; font-size: 12px; font-weight: 500; cursor: pointer; transition: background 0.15s; }
.btn-primary:hover { background: var(--accent-hover); }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-secondary { padding: 7px 14px; background: var(--bg-overlay); border: 1px solid var(--border); border-radius: 6px; color: var(--text-secondary); font-size: 12px; cursor: pointer; transition: background 0.15s; }
.btn-secondary:hover { background: var(--bg-elevated); }
</style>
