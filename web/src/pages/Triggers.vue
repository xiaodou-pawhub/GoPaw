<template>
  <div class="triggers-page">
    <div class="page-header">
      <h1 class="page-title">Triggers</h1>
      <button class="btn-primary" @click="openCreateDialog">
        <PlusIcon :size="16" />
        新建 Trigger
      </button>
    </div>

    <div class="data-card">
      <div class="data-table">
        <div class="data-thead">
          <span>ID</span>
          <span>名称</span>
          <span>Agent</span>
          <span>类型</span>
          <span>启用</span>
          <span>触发次数</span>
          <span>上次触发</span>
          <span>操作</span>
        </div>
        <div v-if="loading" class="empty-state">加载中...</div>
        <div v-else-if="triggers.length === 0" class="empty-state">暂无 Trigger</div>
        <div v-for="item in triggers" :key="item.id" class="data-row">
          <span class="mono">{{ item.id }}</span>
          <span>{{ item.name }}</span>
          <span class="mono text-sm">{{ item.agent_id }}</span>
          <span>
            <span class="badge" :class="getTypeClass(item.type)">{{ item.type }}</span>
          </span>
          <span>
            <button
              class="toggle-btn"
              :class="{ enabled: item.is_enabled }"
              @click="toggleEnabled(item)"
            >
              <span class="toggle-inner" />
            </button>
          </span>
          <span>{{ item.fire_count }}</span>
          <span class="text-sm">
            <span v-if="item.last_fired_at">{{ formatDate(item.last_fired_at) }}</span>
            <span v-else class="text-tertiary">从未触发</span>
          </span>
          <span class="actions">
            <button class="action-btn action-success" title="手动触发" @click="fireTrigger(item)">
              <PlayIcon :size="13" />
            </button>
            <button class="action-btn" title="历史" @click="showHistory(item)">
              <ClockIcon :size="13" />
            </button>
            <button class="action-btn" title="编辑" @click="openEditDialog(item)">
              <PencilIcon :size="13" />
            </button>
            <button class="action-btn action-danger" title="删除" @click="confirmDelete(item)">
              <Trash2Icon :size="13" />
            </button>
          </span>
        </div>
      </div>
    </div>

    <!-- 创建/编辑弹窗 -->
    <div v-if="dialog.show" class="modal-overlay" @click.self="dialog.show = false">
      <div class="modal-card">
        <h2 class="modal-title">{{ dialog.isEdit ? '编辑 Trigger' : '新建 Trigger' }}</h2>
        <form @submit.prevent="saveTrigger">
          <div class="form-group">
            <label>ID</label>
            <input v-model="dialog.data.id" type="text" :disabled="dialog.isEdit" required />
          </div>
          <div class="form-group">
            <label>名称</label>
            <input v-model="dialog.data.name" type="text" required />
          </div>
          <div class="form-group">
            <label>描述</label>
            <input v-model="dialog.data.description" type="text" />
          </div>
          <div class="form-group">
            <label>目标 Agent</label>
            <select v-model="dialog.data.agent_id" required>
              <option value="">请选择...</option>
              <option v-for="a in agents" :key="a.id" :value="a.id">{{ a.name }}</option>
            </select>
          </div>
          <div class="form-group">
            <label>类型</label>
            <select v-model="dialog.data.type" :disabled="dialog.isEdit" required>
              <option value="cron">Cron (定时)</option>
              <option value="webhook">Webhook (外部事件)</option>
              <option value="message">Message (Agent 消息)</option>
            </select>
          </div>

          <!-- Cron Config -->
          <template v-if="dialog.data.type === 'cron'">
            <div class="form-group">
              <label>Cron 表达式</label>
              <div class="input-row">
                <input v-model="dialog.data.config.expression" type="text" placeholder="0 9 * * 1-5" required />
                <button type="button" class="btn-ghost" :disabled="validating" @click="validateCron">
                  {{ validating ? '验证中...' : '验证' }}
                </button>
              </div>
              <div v-if="cronValidation.description" class="hint-text">
                {{ cronValidation.description }}
                <span v-if="cronValidation.next_run">· 下次: {{ formatDate(cronValidation.next_run) }}</span>
              </div>
            </div>
          </template>

          <!-- Webhook Config -->
          <template v-if="dialog.data.type === 'webhook'">
            <div class="form-group">
              <label>Webhook Secret (可选)</label>
              <input v-model="dialog.data.config.secret" type="password" />
            </div>
            <div v-if="dialog.data.id" class="hint-text">
              Webhook URL: {{ webhookUrl }}
            </div>
          </template>

          <!-- Message Config -->
          <template v-if="dialog.data.type === 'message'">
            <div class="form-group">
              <label>来源 Agent (可选)</label>
              <select v-model="dialog.data.config.from_agent">
                <option value="">任意</option>
                <option v-for="a in agents" :key="a.id" :value="a.id">{{ a.name }}</option>
              </select>
            </div>
          </template>

          <div class="form-group">
            <label>触发原因</label>
            <input v-model="dialog.data.reason" type="text" placeholder="触发时传递给 Agent 的上下文" />
          </div>
          <div class="form-row">
            <div class="form-group">
              <label>冷却时间 (秒)</label>
              <input v-model.number="dialog.data.cooldown_seconds" type="number" min="0" />
            </div>
            <div class="form-group">
              <label>最大触发次数</label>
              <input v-model.number="dialog.data.max_fires" type="number" min="1" placeholder="无限制" />
            </div>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn-ghost" @click="dialog.show = false">取消</button>
            <button type="submit" class="btn-primary">保存</button>
          </div>
        </form>
      </div>
    </div>

    <!-- 历史记录弹窗 -->
    <div v-if="historyDialog.show" class="modal-overlay" @click.self="historyDialog.show = false">
      <div class="modal-card modal-wide">
        <h2 class="modal-title">Trigger 历史: {{ historyDialog.trigger?.name }}</h2>
        <div class="data-table">
          <div class="data-thead history-grid">
            <span>时间</span><span>成功</span><span>数据</span><span>错误</span>
          </div>
          <div v-if="historyDialog.loading" class="empty-state">加载中...</div>
          <div v-else-if="historyDialog.history.length === 0" class="empty-state">暂无历史</div>
          <div v-for="h in historyDialog.history" :key="h.fired_at" class="data-row history-grid">
            <span class="text-sm">{{ formatDate(h.fired_at) }}</span>
            <span>
              <CheckCircleIcon v-if="h.success" :size="16" class="text-success" />
              <XCircleIcon v-else :size="16" class="text-error" />
            </span>
            <span class="mono text-sm">{{ h.payload ? JSON.stringify(JSON.parse(h.payload)).substring(0, 40) : '-' }}</span>
            <span class="text-sm text-error">{{ h.error_message || '-' }}</span>
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn-ghost" @click="historyDialog.show = false">关闭</button>
        </div>
      </div>
    </div>

    <!-- 删除确认弹窗 -->
    <div v-if="deleteDialog.show" class="modal-overlay" @click.self="deleteDialog.show = false">
      <div class="modal-card modal-sm">
        <h2 class="modal-title">确认删除</h2>
        <p class="confirm-text">确定要删除 Trigger "{{ deleteDialog.trigger?.name }}" 吗？</p>
        <div class="modal-actions">
          <button class="btn-ghost" @click="deleteDialog.show = false">取消</button>
          <button class="btn-danger" @click="deleteTrigger">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue'
import {
  PlusIcon, PlayIcon, ClockIcon, PencilIcon, Trash2Icon,
  CheckCircleIcon, XCircleIcon,
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { triggersApi, type Trigger, type TriggerHistory } from '@/api/triggers'
import { listAgents, type Agent } from '@/api/agents'

const loading = ref(false)
const triggers = ref<Trigger[]>([])
const agents = ref<Agent[]>([])
const validating = ref(false)

const dialog = reactive({
  show: false,
  isEdit: false,
  data: {
    id: '',
    agent_id: '',
    name: '',
    description: '',
    type: 'cron' as 'cron' | 'webhook' | 'message',
    config: {} as any,
    reason: '',
    is_enabled: true,
    cooldown_seconds: 0,
    max_fires: null as number | null,
  },
})

const cronValidation = reactive({ description: '', next_run: '', error: '' })

const historyDialog = reactive({
  show: false,
  trigger: null as Trigger | null,
  history: [] as TriggerHistory[],
  loading: false,
})

const deleteDialog = reactive({
  show: false,
  trigger: null as Trigger | null,
})

const webhookUrl = computed(() => {
  if (!dialog.data.id) return ''
  return `${window.location.origin}/webhook/${dialog.data.id}`
})

function getTypeClass(type: string) {
  const map: Record<string, string> = { cron: 'badge-info', webhook: 'badge-success', message: 'badge-warning' }
  return map[type] || 'badge-neutral'
}

function formatDate(date: string) {
  return new Date(date).toLocaleString('zh-CN')
}

async function loadTriggers() {
  loading.value = true
  try {
    triggers.value = await triggersApi.list()
  } catch {
    toast.error('加载 Triggers 失败')
  } finally {
    loading.value = false
  }
}

async function loadAgents() {
  try {
    const response = await listAgents()
    agents.value = response.agents
  } catch {
    // 忽略
  }
}

function openCreateDialog() {
  dialog.isEdit = false
  dialog.data = {
    id: '', agent_id: '', name: '', description: '', type: 'cron',
    config: { expression: '' }, reason: '', is_enabled: true, cooldown_seconds: 0, max_fires: null,
  }
  cronValidation.description = ''
  dialog.show = true
}

function openEditDialog(trigger: Trigger) {
  dialog.isEdit = true
  dialog.data = {
    id: trigger.id, agent_id: trigger.agent_id, name: trigger.name,
    description: trigger.description, type: trigger.type,
    config: JSON.parse(JSON.stringify(trigger.config)), reason: trigger.reason,
    is_enabled: trigger.is_enabled, cooldown_seconds: trigger.cooldown_seconds,
    max_fires: trigger.max_fires,
  }
  cronValidation.description = ''
  dialog.show = true
}

async function validateCron() {
  if (!dialog.data.config.expression) return
  validating.value = true
  try {
    const response = await triggersApi.validateCron(dialog.data.config.expression as string)
    cronValidation.description = response.description || ''
    cronValidation.next_run = response.next_run || ''
    if (!response.valid) toast.error(response.error || '表达式无效')
  } catch {
    toast.error('验证失败')
  } finally {
    validating.value = false
  }
}

async function saveTrigger() {
  try {
    const maxFires = dialog.data.max_fires === null ? undefined : dialog.data.max_fires
    if (dialog.isEdit) {
      await triggersApi.update(dialog.data.id, {
        name: dialog.data.name, description: dialog.data.description,
        config: dialog.data.config, reason: dialog.data.reason,
        is_enabled: dialog.data.is_enabled, cooldown_seconds: dialog.data.cooldown_seconds,
        max_fires: maxFires,
      })
      toast.success('Trigger 更新成功')
    } else {
      await triggersApi.create({
        id: dialog.data.id, agent_id: dialog.data.agent_id,
        name: dialog.data.name, description: dialog.data.description,
        type: dialog.data.type, config: dialog.data.config, reason: dialog.data.reason,
        is_enabled: dialog.data.is_enabled, cooldown_seconds: dialog.data.cooldown_seconds,
        max_fires: maxFires,
      })
      toast.success('Trigger 创建成功')
    }
    dialog.show = false
    loadTriggers()
  } catch (error: unknown) {
    const msg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
    toast.error(msg || '保存失败')
  }
}

async function toggleEnabled(trigger: Trigger) {
  try {
    if (trigger.is_enabled) {
      await triggersApi.disable(trigger.id)
    } else {
      await triggersApi.enable(trigger.id)
    }
    trigger.is_enabled = !trigger.is_enabled
    toast.success(`Trigger ${trigger.is_enabled ? '启用' : '禁用'}成功`)
  } catch {
    toast.error('操作失败')
  }
}

async function fireTrigger(trigger: Trigger) {
  try {
    await triggersApi.fire(trigger.id)
    toast.success('Trigger 手动触发成功')
    loadTriggers()
  } catch (error: unknown) {
    const msg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
    toast.error(msg || '触发失败')
  }
}

async function showHistory(trigger: Trigger) {
  historyDialog.trigger = trigger
  historyDialog.show = true
  historyDialog.loading = true
  try {
    historyDialog.history = await triggersApi.getHistory(trigger.id)
  } catch {
    toast.error('加载历史失败')
  } finally {
    historyDialog.loading = false
  }
}

function confirmDelete(trigger: Trigger) {
  deleteDialog.trigger = trigger
  deleteDialog.show = true
}

async function deleteTrigger() {
  if (!deleteDialog.trigger) return
  try {
    await triggersApi.delete(deleteDialog.trigger.id)
    toast.success('Trigger 删除成功')
    loadTriggers()
  } catch {
    toast.error('删除失败')
  } finally {
    deleteDialog.show = false
    deleteDialog.trigger = null
  }
}

onMounted(() => {
  loadTriggers()
  loadAgents()
})
</script>

<style scoped>
.triggers-page {
  padding: 24px 32px;
  height: 100%;
  overflow-y: auto;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.page-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 6px;
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

.btn-primary:hover { background: var(--accent-hover); }

.btn-ghost {
  padding: 7px 14px;
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
}

.btn-ghost:hover { background: var(--bg-overlay); }
.btn-ghost:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-danger {
  padding: 8px 16px;
  background: #ef4444;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-danger:hover { background: #dc2626; }

.data-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}

.data-table { overflow-x: auto; }

.data-thead,
.data-row {
  display: grid;
  grid-template-columns: 120px 1fr 120px 80px 60px 80px 160px 120px;
  padding: 10px 16px;
  align-items: center;
  gap: 8px;
}

.data-thead {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--bg-overlay);
  border-bottom: 1px solid var(--border);
}

.data-row {
  font-size: 13px;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-subtle);
}

.data-row:last-child { border-bottom: none; }

.mono { font-family: monospace; font-size: 12px; color: var(--text-secondary); }
.text-sm { font-size: 12px; color: var(--text-secondary); }
.text-tertiary { color: var(--text-tertiary); }
.text-success { color: #16a34a; }
.text-error { color: #ef4444; }

.badge {
  display: inline-block;
  padding: 2px 7px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.badge-info    { background: rgba(59,130,246,0.15); color: #3b82f6; }
.badge-success { background: rgba(34,197,94,0.15);  color: #16a34a; }
.badge-warning { background: rgba(234,179,8,0.15);  color: #ca8a04; }
.badge-neutral { background: var(--bg-overlay); color: var(--text-secondary); border: 1px solid var(--border); }

/* Toggle button */
.toggle-btn {
  width: 36px;
  height: 20px;
  border-radius: 10px;
  border: none;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  cursor: pointer;
  position: relative;
  transition: background 0.2s;
}

.toggle-btn.enabled {
  background: var(--accent);
  border-color: var(--accent);
}

.toggle-inner {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: white;
  transition: transform 0.2s;
  display: block;
}

.toggle-btn.enabled .toggle-inner {
  transform: translateX(16px);
}

.actions {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all 0.1s;
}

.action-btn:hover { background: var(--bg-overlay); }
.action-success:hover { background: rgba(34,197,94,0.1); color: #16a34a; border-color: rgba(34,197,94,0.3); }
.action-danger:hover  { background: rgba(239,68,68,0.1); color: #ef4444; border-color: rgba(239,68,68,0.3); }

.empty-state {
  padding: 24px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 13px;
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 24px;
  width: 500px;
  max-width: 95vw;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-wide { width: 700px; }
.modal-sm { width: 380px; }

.modal-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 20px;
}

.confirm-text {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 20px;
}

.form-group {
  margin-bottom: 14px;
}

.form-group label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.form-group input,
.form-group select {
  width: 100%;
  padding: 8px 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  box-sizing: border-box;
}

.form-group input:disabled { opacity: 0.6; }

.input-row {
  display: flex;
  gap: 8px;
}

.input-row input { flex: 1; }

.hint-text {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-top: 4px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 20px;
}

/* History table */
.history-grid {
  grid-template-columns: 160px 60px 1fr 1fr !important;
}
</style>
