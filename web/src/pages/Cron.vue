<template>
  <div class="page-root">
    <!-- 顶栏 -->
    <div class="page-header">
      <div>
        <h1 class="page-title">定时任务</h1>
        <p class="page-desc">自动化执行 Agent 任务，支持灵活调度</p>
      </div>
      <button class="btn-primary" @click="openModal('create')">
        <PlusIcon :size="13" /> 新建任务
      </button>
    </div>

    <!-- 任务列表 -->
    <div class="task-list">
      <div v-if="loading" class="list-empty">加载中...</div>
      <div v-else-if="jobs.length === 0" class="list-empty">
        <ClockIcon :size="32" class="empty-icon" />
        <p>暂无定时任务</p>
        <button class="btn-primary" @click="openModal('create')">创建第一个任务</button>
      </div>
      <div v-else class="task-cards">
        <div v-for="job in jobs" :key="job.id" class="task-card">
          <!-- 卡片头部 -->
          <div class="card-header">
            <div class="card-title-row">
              <span class="card-name">{{ job.name }}</span>
              <label class="toggle" :title="job.enabled ? '点击禁用' : '点击启用'">
                <input type="checkbox" :checked="job.enabled" @change="handleToggle(job)" />
                <span class="toggle-slider" />
              </label>
            </div>
            <div class="card-schedule">
              <CalendarClockIcon :size="13" />
              <span>{{ describeSchedule(job.schedule || job.cron_expr) }}</span>
              <code class="expr-hint">{{ job.schedule || job.cron_expr }}</code>
            </div>
          </div>

          <!-- Agent 关联 -->
          <div v-if="getAgentName(job.target_id)" class="card-agent">
            <BotIcon :size="12" />
            <span>{{ getAgentName(job.target_id) }}</span>
          </div>

          <!-- 任务内容预览 -->
          <div class="card-task">{{ job.task || job.prompt }}</div>

          <!-- 最近运行状态 -->
          <div v-if="job.last_run_at" class="card-last-run">
            <span class="run-dot" :class="job.last_result === 'success' ? 'ok' : 'err'" />
            上次运行：{{ formatTime(job.last_run_at) }}
            <span v-if="job.last_result" class="run-result" :class="job.last_result">
              {{ job.last_result === 'success' ? '成功' : '失败' }}
            </span>
          </div>

          <!-- 操作栏 -->
          <div class="card-actions">
            <button class="action-btn" title="立即执行" @click="handleTrigger(job)">
              <PlayIcon :size="12" /> 立即执行
            </button>
            <button class="action-btn" title="执行历史" @click="openHistory(job)">
              <HistoryIcon :size="12" /> 历史
            </button>
            <div class="action-spacer" />
            <button class="action-btn icon-only" title="编辑" @click="openModal('edit', job)">
              <PencilIcon :size="12" />
            </button>
            <button class="action-btn icon-only danger" title="删除" @click="handleDelete(job)">
              <TrashIcon :size="12" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 新增/编辑弹窗 -->
    <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
      <div class="modal-card">
        <div class="modal-header">
          <h3 class="modal-title">{{ modalType === 'create' ? '新建定时任务' : '编辑定时任务' }}</h3>
          <button class="icon-close" @click="showModal = false"><XIcon :size="16" /></button>
        </div>
        <div class="modal-body">
          <div class="form-field">
            <label>任务名称 <span class="required">*</span></label>
            <input v-model="formModel.name" placeholder="给任务取个名字" class="form-input" />
          </div>

          <div class="form-field">
            <label>执行计划 <span class="required">*</span></label>
            <input v-model="formModel.schedule" placeholder="例如: 0 9 * * * 1-5（工作日 9 点）" class="form-input" />
            <div v-if="schedulePreview" class="schedule-preview">
              <CalendarClockIcon :size="12" />
              <span>{{ schedulePreview }}</span>
            </div>
            <div v-if="scheduleError" class="schedule-error">{{ scheduleError }}</div>
            <div class="form-tip">
              使用 6 位 Cron 格式：<code>秒 分 时 日 月 周</code>
              &nbsp;·&nbsp;
              <a href="#" class="tip-link" @click.prevent="fillExample('0 0 9 * * 1-5')">工作日 9 点</a>
              &nbsp;·&nbsp;
              <a href="#" class="tip-link" @click.prevent="fillExample('0 0 8 * * *')">每天 8 点</a>
              &nbsp;·&nbsp;
              <a href="#" class="tip-link" @click.prevent="fillExample('0 0 10 * * 1')">每周一 10 点</a>
            </div>
          </div>

          <div class="form-field">
            <label>关联 Agent</label>
            <select v-model="formModel.target_id" class="form-select">
              <option value="">使用默认 Agent</option>
              <option v-for="agent in agents" :key="agent.id" :value="agent.id">{{ agent.name }}</option>
            </select>
          </div>

          <div class="form-field">
            <label>触发提示词 <span class="required">*</span></label>
            <textarea v-model="formModel.task" placeholder="定时触发时发给 Agent 的内容" rows="4" class="form-textarea" />
          </div>

          <div class="form-field inline">
            <label class="toggle">
              <input type="checkbox" v-model="formModel.enabled" />
              <span class="toggle-slider" />
            </label>
            <span class="toggle-label">立即启用</span>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-secondary" @click="showModal = false">取消</button>
          <button class="btn-primary" :disabled="submitting" @click="handleSubmit">
            {{ submitting ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 执行历史侧边栏 -->
    <div v-if="showHistory" class="history-overlay" @click.self="showHistory = false">
      <div class="history-drawer">
        <div class="drawer-header">
          <div>
            <h3 class="drawer-title">执行历史</h3>
            <p class="drawer-subtitle">{{ currentJob?.name }}</p>
          </div>
          <button class="icon-close" @click="showHistory = false"><XIcon :size="16" /></button>
        </div>
        <div class="drawer-body">
          <div v-if="historyLoading" class="history-empty">加载中...</div>
          <div v-else-if="runHistory.length === 0" class="history-empty">暂无执行记录</div>
          <div v-else class="history-list">
            <div v-for="run in runHistory" :key="run.id" class="history-item">
              <div class="history-header">
                <span class="run-badge" :class="run.status">{{ statusLabel(run.status) }}</span>
                <span class="run-time">{{ formatTime(run.triggered_at) }}</span>
              </div>
              <div v-if="run.output" class="run-output">{{ run.output }}</div>
              <div v-if="run.error" class="run-error">{{ run.error }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import {
  PlusIcon, PencilIcon, TrashIcon, PlayIcon, ClockIcon,
  XIcon, CalendarClockIcon, BotIcon, HistoryIcon
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import cronstrue from 'cronstrue/i18n'
import { getCronJobs, createCronJob, updateCronJob, deleteCronJob, triggerCronJob, getCronRunHistory } from '@/api/cron'
import { listAgents } from '@/api/agents'
import type { CronJob, CronRun } from '@/types'
import type { Agent } from '@/api/agents'

// ---- State ----

const jobs = ref<CronJob[]>([])
const agents = ref<Agent[]>([])
const loading = ref(false)
const submitting = ref(false)

const showModal = ref(false)
const modalType = ref<'create' | 'edit'>('create')
const editingId = ref<string | null>(null)

const showHistory = ref(false)
const currentJob = ref<CronJob | null>(null)
const runHistory = ref<CronRun[]>([])
const historyLoading = ref(false)

const formModel = reactive({
  name: '',
  schedule: '',
  task: '',
  target_id: '',
  enabled: true,
})

// ---- Schedule preview ----

const schedulePreview = ref('')
const scheduleError = ref('')

watch(() => formModel.schedule, (val) => {
  if (!val) { schedulePreview.value = ''; scheduleError.value = ''; return }
  try {
    schedulePreview.value = cronstrue.toString(val, { locale: 'zh_CN', use24HourTimeFormat: true })
    scheduleError.value = ''
  } catch {
    schedulePreview.value = ''
    scheduleError.value = '表达式无效，请检查格式'
  }
})

// ---- Helpers ----

const agentMap = computed(() => {
  const m: Record<string, string> = {}
  agents.value.forEach(a => { m[a.id] = a.name })
  return m
})

function getAgentName(id: string): string {
  return id ? agentMap.value[id] || '' : ''
}

function describeSchedule(expr: string): string {
  if (!expr) return '未设置'
  try {
    return cronstrue.toString(expr, { locale: 'zh_CN', use24HourTimeFormat: true })
  } catch {
    return expr
  }
}

function formatTime(ts: string | undefined | null): string {
  if (!ts) return '-'
  const d = new Date(ts)
  return isNaN(d.getTime()) ? ts : d.toLocaleString('zh-CN', { hour12: false })
}

function statusLabel(status: string): string {
  const map: Record<string, string> = { success: '成功', error: '失败', running: '运行中' }
  return map[status] || status
}

function fillExample(expr: string) {
  formModel.schedule = expr
}

// ---- Load ----

async function loadJobs() {
  loading.value = true
  try { jobs.value = await getCronJobs() } catch {}
  finally { loading.value = false }
}

async function loadAgents() {
  try {
    const res = await listAgents()
    agents.value = res.agents || []
  } catch {}
}

// ---- Modal ----

function openModal(type: 'create' | 'edit', job?: CronJob) {
  modalType.value = type
  schedulePreview.value = ''
  scheduleError.value = ''
  if (type === 'edit' && job) {
    editingId.value = job.id
    Object.assign(formModel, {
      name: job.name,
      schedule: job.schedule || job.cron_expr || '',
      task: job.task || job.prompt || '',
      target_id: job.target_id || '',
      enabled: job.enabled,
    })
  } else {
    editingId.value = null
    Object.assign(formModel, { name: '', schedule: '', task: '', target_id: '', enabled: true })
  }
  showModal.value = true
}

async function handleSubmit() {
  if (!formModel.name || !formModel.schedule || !formModel.task) {
    toast.error('请填写必填字段')
    return
  }
  if (scheduleError.value) {
    toast.error('Cron 表达式无效')
    return
  }
  submitting.value = true
  try {
    const payload = {
      name: formModel.name,
      schedule: formModel.schedule,
      task: formModel.task,
      target_id: formModel.target_id,
      enabled: formModel.enabled,
      channel: 'console',
    }
    if (modalType.value === 'create') {
      await createCronJob(payload)
    } else if (editingId.value) {
      await updateCronJob(editingId.value, payload)
    }
    toast.success('保存成功')
    showModal.value = false
    loadJobs()
  } catch (err: any) {
    toast.error(err.response?.data?.error || '保存失败')
  } finally {
    submitting.value = false
  }
}

// ---- Actions ----

async function handleToggle(job: CronJob) {
  try {
    await updateCronJob(job.id, {
      name: job.name,
      schedule: job.schedule || job.cron_expr,
      task: job.task || job.prompt,
      target_id: job.target_id,
      enabled: !job.enabled,
      channel: job.channel,
    })
    await loadJobs()
  } catch {
    toast.error('切换状态失败')
  }
}

async function handleTrigger(job: CronJob) {
  if (!confirm(`确认立即执行任务「${job.name}」？`)) return
  try {
    await triggerCronJob(job.id)
    toast.success('已触发执行')
  } catch {
    toast.error('触发失败')
  }
}

async function handleDelete(job: CronJob) {
  if (!confirm(`删除任务「${job.name}」后无法恢复，是否继续？`)) return
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

onMounted(() => {
  loadJobs()
  loadAgents()
})
</script>

<style scoped>
.page-root {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 24px;
  max-width: 1100px;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.page-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 4px;
}

.page-desc {
  font-size: 12px;
  color: var(--text-tertiary);
  margin: 0;
}

/* Task grid */
.task-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 14px;
}

.list-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 64px 24px;
  color: var(--text-tertiary);
  font-size: 13px;
}

.empty-icon { opacity: 0.3; }

/* Task Card */
.task-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: border-color 0.15s;
}

.task-card:hover { border-color: var(--accent); }

.card-header { display: flex; flex-direction: column; gap: 6px; }

.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.card-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-schedule {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
}

.expr-hint {
  font-family: monospace;
  font-size: 10px;
  color: var(--text-tertiary);
  background: var(--bg-overlay);
  padding: 1px 5px;
  border-radius: 3px;
}

.card-agent {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  color: var(--accent);
}

.card-task {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.card-last-run {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--text-tertiary);
}

.run-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.run-dot.ok { background: var(--green); }
.run-dot.err { background: var(--red); }

.run-result {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 3px;
  font-weight: 600;
}

.run-result.success { background: rgba(34,197,94,0.1); color: var(--green); }
.run-result.error { background: rgba(239,68,68,0.1); color: var(--red); }

.card-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  padding-top: 6px;
  border-top: 1px solid var(--border-subtle);
}

.action-spacer { flex: 1; }

.action-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 5px;
  color: var(--text-secondary);
  font-size: 11px;
  cursor: pointer;
  transition: all 0.12s;
}

.action-btn:hover { background: var(--bg-overlay); color: var(--text-primary); border-color: var(--border); }
.action-btn.icon-only { padding: 4px 6px; }
.action-btn.danger:hover { color: var(--red); border-color: rgba(239,68,68,0.3); }

/* Toggle */
.toggle { position: relative; width: 34px; height: 18px; cursor: pointer; display: block; flex-shrink: 0; }
.toggle input { display: none; }
.toggle-slider { position: absolute; inset: 0; background: var(--bg-overlay); border: 1px solid var(--border); border-radius: 9px; transition: background 0.2s; }
.toggle-slider::before { content: ''; position: absolute; width: 12px; height: 12px; background: var(--text-tertiary); border-radius: 50%; top: 2px; left: 2px; transition: transform 0.2s, background 0.2s; }
.toggle input:checked + .toggle-slider { background: rgba(124, 106, 247, 0.2); border-color: var(--accent); }
.toggle input:checked + .toggle-slider::before { transform: translateX(16px); background: var(--accent); }

/* Buttons */
.btn-primary { display: flex; align-items: center; gap: 6px; padding: 7px 14px; background: var(--accent); border: none; border-radius: 6px; color: #fff; font-size: 12px; font-weight: 500; cursor: pointer; transition: background 0.15s; }
.btn-primary:hover { background: var(--accent-hover); }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-secondary { padding: 7px 14px; background: var(--bg-overlay); border: 1px solid var(--border); border-radius: 6px; color: var(--text-secondary); font-size: 12px; cursor: pointer; transition: background 0.15s; }
.btn-secondary:hover { background: var(--bg-elevated); }

/* Modal */
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.65); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal-card { width: 540px; max-height: 90vh; background: var(--bg-elevated); border: 1px solid var(--border); border-radius: 12px; display: flex; flex-direction: column; overflow: hidden; }
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
.required { color: var(--red); }

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

.form-textarea { resize: vertical; min-height: 80px; }
.form-input:focus, .form-select:focus, .form-textarea:focus { border-color: var(--accent); }

.form-tip {
  font-size: 11px;
  color: var(--text-tertiary);
  line-height: 1.5;
}

.tip-link {
  color: var(--accent);
  text-decoration: none;
  cursor: pointer;
}

.tip-link:hover { text-decoration: underline; }

.schedule-preview {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--green);
  background: rgba(34,197,94,0.08);
  border: 1px solid rgba(34,197,94,0.2);
  border-radius: 5px;
  padding: 5px 10px;
}

.schedule-error {
  font-size: 11px;
  color: var(--red);
}

/* History Drawer */
.history-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; justify-content: flex-end; z-index: 1000; }
.history-drawer { width: 460px; height: 100%; background: var(--bg-elevated); border-left: 1px solid var(--border); display: flex; flex-direction: column; }
.drawer-header { display: flex; align-items: flex-start; justify-content: space-between; padding: 16px 20px; border-bottom: 1px solid var(--border-subtle); }
.drawer-title { font-size: 14px; font-weight: 600; color: var(--text-primary); margin: 0 0 2px; }
.drawer-subtitle { font-size: 12px; color: var(--text-tertiary); margin: 0; }
.drawer-body { flex: 1; overflow-y: auto; padding: 16px; }

.history-empty { text-align: center; color: var(--text-tertiary); font-size: 13px; padding: 32px; }
.history-list { display: flex; flex-direction: column; gap: 10px; }

.history-item {
  background: var(--bg-panel);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  padding: 10px 12px;
}

.history-header { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }

.run-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 3px;
}

.run-badge.success { background: rgba(34,197,94,0.1); color: var(--green); }
.run-badge.error { background: rgba(239,68,68,0.1); color: var(--red); }
.run-badge.running { background: rgba(124,106,247,0.1); color: var(--accent); }

.run-time { font-size: 11px; color: var(--text-tertiary); margin-left: auto; }

.run-output, .run-error {
  font-family: monospace;
  font-size: 11px;
  background: var(--bg-app);
  border-radius: 4px;
  padding: 6px 8px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 120px;
  overflow-y: auto;
}

.run-error { color: var(--red); }
</style>
