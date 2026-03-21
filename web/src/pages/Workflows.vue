<template>
  <div class="workflows-page">
    <div class="page-header">
      <h1 class="page-title">工作流</h1>
      <button class="btn-primary" @click="openCreateDialog">
        <PlusIcon :size="16" /> 新建工作流
      </button>
    </div>

    <div class="wf-layout">
      <!-- 工作流列表 -->
      <div class="wf-sidebar">
        <div class="sidebar-title">工作流列表</div>
        <div
          v-for="wf in workflows"
          :key="wf.id"
          class="wf-item"
          :class="{ active: selectedWorkflow?.id === wf.id }"
          @click="selectWorkflow(wf)"
        >
          <div class="wf-item-top">
            <span class="wf-name">{{ wf.name }}</span>
            <span class="badge" :class="getStatusClass(wf.status)">{{ wf.status }}</span>
          </div>
          <div class="wf-meta">{{ wf.id }}</div>
        </div>
        <div v-if="workflows.length === 0" class="empty-state">暂无工作流</div>
      </div>

      <!-- 详情面板 -->
      <div v-if="selectedWorkflow" class="wf-detail">
        <div class="detail-header">
          <h2 class="detail-title">{{ selectedWorkflow.name }}</h2>
          <div class="detail-actions">
            <button class="btn-icon-label btn-success" :disabled="selectedWorkflow.status !== 'active'" @click="executeWorkflow">
              <PlayIcon :size="14" /> 执行
            </button>
            <button class="btn-ghost-sm" @click="openEditDialog">
              <PencilIcon :size="14" /> 编辑
            </button>
            <button class="btn-danger-sm" @click="confirmDelete">
              <Trash2Icon :size="14" />
            </button>
          </div>
        </div>

        <p class="detail-desc">{{ selectedWorkflow.description }}</p>

        <!-- 统计 -->
        <div v-if="stats" class="stat-bar">
          <div class="stat-item"><div class="stat-value">{{ stats.total_executions }}</div><div class="stat-label">总执行</div></div>
          <div class="stat-item"><div class="stat-value text-success">{{ stats.completed_count }}</div><div class="stat-label">成功</div></div>
          <div class="stat-item"><div class="stat-value text-error">{{ stats.failed_count }}</div><div class="stat-label">失败</div></div>
          <div class="stat-item"><div class="stat-value text-info">{{ stats.running_count }}</div><div class="stat-label">运行中</div></div>
        </div>

        <!-- 步骤列表 -->
        <div class="steps-section">
          <div class="section-title">步骤</div>
          <div class="steps-list">
            <div v-for="step in selectedWorkflow.definition.steps" :key="step.id" class="step-item">
              <div class="step-icon" :style="{ background: getStepColor(step.action) }">{{ step.action[0].toUpperCase() }}</div>
              <div class="step-info">
                <div class="step-name">{{ step.name || step.id }}</div>
                <div class="step-meta">Agent: {{ step.agent }} · Action: {{ step.action }}</div>
                <div v-if="step.depends_on?.length" class="step-deps">依赖: {{ step.depends_on.join(', ') }}</div>
              </div>
            </div>
          </div>
        </div>

        <!-- 执行历史 -->
        <div class="section-title" style="margin-top:20px">执行历史</div>
        <div class="data-table">
          <div class="data-thead">
            <span>ID</span><span>状态</span><span>开始时间</span><span>操作</span>
          </div>
          <div v-if="loadingExecutions" class="empty-state">加载中...</div>
          <div v-else-if="executions.length === 0" class="empty-state">暂无执行记录</div>
          <div v-for="exec in executions" :key="exec.id" class="data-row">
            <span class="mono">{{ exec.id.substring(0, 8) }}...</span>
            <span><span class="badge" :class="getExecStatusClass(exec.status)">{{ exec.status }}</span></span>
            <span class="text-sm">{{ exec.started_at ? formatDate(exec.started_at) : '-' }}</span>
            <span class="actions">
              <button class="action-btn" title="查看" @click="viewExecution(exec)"><EyeIcon :size="13" /></button>
              <button v-if="exec.status === 'running'" class="action-btn action-danger" title="取消" @click="cancelExecution(exec)"><XIcon :size="13" /></button>
            </span>
          </div>
        </div>
      </div>

      <div v-else class="wf-empty">
        <GitBranchIcon :size="48" />
        <p>请选择或创建一个工作流</p>
      </div>
    </div>

    <!-- 创建/编辑弹窗 -->
    <div v-if="dialog.show" class="modal-overlay">
      <div class="modal-fullscreen">
        <div class="modal-fheader">
          <h2 class="modal-title">{{ dialog.isEdit ? '编辑工作流' : '新建工作流' }}</h2>
          <button class="btn-icon-close" @click="dialog.show = false"><XIcon :size="20" /></button>
        </div>
        <div class="modal-fbody">
          <!-- 左：基本信息 -->
          <div class="modal-sidebar">
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
              <textarea v-model="dialog.data.description" rows="3" />
            </div>
            <div class="form-group">
              <label>状态</label>
              <select v-model="dialog.data.status">
                <option value="draft">草稿</option>
                <option value="active">启用</option>
                <option value="disabled">禁用</option>
              </select>
            </div>
          </div>
          <!-- 右：设计器/JSON -->
          <div class="modal-main">
            <div class="tab-bar">
              <button class="tab-btn" :class="{ active: dialog.activeTab === 'designer' }" @click="switchTab('designer')">设计器</button>
              <button class="tab-btn" :class="{ active: dialog.activeTab === 'json' }" @click="switchTab('json')">JSON</button>
            </div>
            <div v-if="dialog.activeTab === 'designer'" class="designer-wrap">
              <WorkflowDesigner
                :workflow="dialog.data.definition"
                :agents="[]"
                @save="onDesignerSave"
                @validate="onDesignerValidate"
              />
            </div>
            <div v-if="dialog.activeTab === 'json'" class="json-wrap">
              <textarea v-model="dialog.definitionText" class="json-editor" />
            </div>
          </div>
        </div>
        <div class="modal-ffooter">
          <button class="btn-ghost" @click="dialog.show = false">取消</button>
          <button class="btn-primary" @click="saveWorkflow">保存</button>
        </div>
      </div>
    </div>

    <!-- 执行弹窗 -->
    <div v-if="executeDialog.show" class="modal-overlay" @click.self="executeDialog.show = false">
      <div class="modal-card">
        <h2 class="modal-title">执行工作流</h2>
        <div class="form-group">
          <label>输入参数 (JSON)</label>
          <textarea v-model="executeDialog.inputText" rows="5" placeholder='{"key": "value"}' />
        </div>
        <div class="modal-actions">
          <button class="btn-ghost" @click="executeDialog.show = false">取消</button>
          <button class="btn-primary" @click="confirmExecute">执行</button>
        </div>
      </div>
    </div>

    <!-- 执行详情弹窗 -->
    <div v-if="executionDialog.show && executionDialog.execution" class="modal-overlay" @click.self="executionDialog.show = false">
      <div class="modal-card modal-wide">
        <h2 class="modal-title">执行详情</h2>
        <div class="detail-grid">
          <div class="detail-item"><div class="detail-label">ID</div><div class="detail-value mono">{{ executionDialog.execution.id }}</div></div>
          <div class="detail-item"><div class="detail-label">状态</div><div class="detail-value"><span class="badge" :class="getExecStatusClass(executionDialog.execution.status)">{{ executionDialog.execution.status }}</span></div></div>
          <div class="detail-item"><div class="detail-label">开始时间</div><div class="detail-value text-sm">{{ executionDialog.execution.started_at ? formatDate(executionDialog.execution.started_at) : '-' }}</div></div>
          <div class="detail-item"><div class="detail-label">完成时间</div><div class="detail-value text-sm">{{ executionDialog.execution.completed_at ? formatDate(executionDialog.execution.completed_at) : '-' }}</div></div>
          <div v-if="executionDialog.execution.error" class="detail-item full"><div class="detail-label">错误</div><div class="detail-value text-error">{{ executionDialog.execution.error }}</div></div>
        </div>
        <div class="section-title" style="margin-top:16px">步骤执行</div>
        <div class="steps-exec-list">
          <div v-for="step in executionDialog.execution.steps" :key="step.id" class="step-exec-item">
            <div class="step-exec-icon" :class="getStepStatusClass(step.status)">{{ step.status[0].toUpperCase() }}</div>
            <div><div class="step-exec-id">{{ step.step_id }}</div><div class="step-exec-meta">{{ step.agent_id }} · {{ step.status }}</div></div>
          </div>
        </div>
        <div class="modal-actions"><button class="btn-ghost" @click="executionDialog.show = false">关闭</button></div>
      </div>
    </div>

    <!-- 删除确认 -->
    <div v-if="deleteDialog.show" class="modal-overlay" @click.self="deleteDialog.show = false">
      <div class="modal-card modal-sm">
        <h2 class="modal-title">确认删除</h2>
        <p class="confirm-text">确定要删除工作流 "{{ deleteDialog.workflow?.name }}" 吗？</p>
        <div class="modal-actions">
          <button class="btn-ghost" @click="deleteDialog.show = false">取消</button>
          <button class="btn-danger" @click="deleteWorkflow">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import {
  PlusIcon, PlayIcon, PencilIcon, Trash2Icon, EyeIcon, XIcon, GitBranchIcon,
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { workflowsApi, type Workflow, type Execution, type ExecutionStats } from '@/api/workflows'
import WorkflowDesigner from '@/components/workflow/WorkflowDesigner.vue'

const workflows = ref<Workflow[]>([])
const selectedWorkflow = ref<Workflow | null>(null)
const executions = ref<Execution[]>([])
const stats = ref<ExecutionStats | null>(null)
const loadingExecutions = ref(false)

const dialog = reactive({
  show: false, isEdit: false,
  activeTab: 'designer' as 'designer' | 'json',
  data: { id: '', name: '', description: '', status: 'draft' as 'draft' | 'active' | 'disabled', definition: { steps: [] } as any },
  definitionText: '',
})

const executeDialog = reactive({ show: false, inputText: '{}' })

const executionDialog = reactive({ show: false, execution: null as Execution | null })

const deleteDialog = reactive({ show: false, workflow: null as Workflow | null })

function getStatusClass(status: string) {
  const map: Record<string, string> = { active: 'badge-success', draft: 'badge-warning', disabled: 'badge-neutral' }
  return map[status] || 'badge-neutral'
}

function getExecStatusClass(status: string) {
  const map: Record<string, string> = { completed: 'badge-success', running: 'badge-info', pending: 'badge-warning', failed: 'badge-error', cancelled: 'badge-neutral' }
  return map[status] || 'badge-neutral'
}

function getStepStatusClass(status: string) {
  const map: Record<string, string> = { completed: 'step-success', running: 'step-info', pending: 'step-warning', failed: 'step-error' }
  return map[status] || 'step-neutral'
}

function getStepColor(action: string) {
  const map: Record<string, string> = { task: '#3b82f6', notify: '#f59e0b', query: '#6366f1' }
  return map[action] || '#94a3b8'
}

function formatDate(date: string) { return new Date(date).toLocaleString('zh-CN') }

async function loadWorkflows() {
  try { workflows.value = await workflowsApi.list() }
  catch { toast.error('加载工作流失败') }
}

async function selectWorkflow(wf: Workflow) {
  selectedWorkflow.value = wf
  loadExecutions(wf.id)
  loadStats(wf.id)
}

async function loadExecutions(id: string) {
  loadingExecutions.value = true
  try { executions.value = await workflowsApi.listExecutions(id, 10) }
  catch { toast.error('加载执行历史失败') }
  finally { loadingExecutions.value = false }
}

async function loadStats(id: string) {
  try { stats.value = await workflowsApi.getStats(id) }
  catch { /* ignore */ }
}

function openCreateDialog() {
  dialog.isEdit = false
  dialog.activeTab = 'designer'
  dialog.data = { id: '', name: '', description: '', status: 'draft', definition: { steps: [] } }
  dialog.definitionText = JSON.stringify({ steps: [{ id: 'step1', name: '第一步', agent: '', action: 'task', input: {} }] }, null, 2)
  dialog.show = true
}

function openEditDialog() {
  if (!selectedWorkflow.value) return
  dialog.isEdit = true
  dialog.activeTab = 'designer'
  dialog.data = { ...selectedWorkflow.value }
  dialog.definitionText = JSON.stringify(selectedWorkflow.value.definition, null, 2)
  dialog.show = true
}

function switchTab(tab: 'designer' | 'json') {
  if (tab === 'designer') {
    try { dialog.data.definition = JSON.parse(dialog.definitionText) }
    catch { toast.error('JSON 格式无效，无法切换到设计器'); return }
  } else {
    dialog.definitionText = JSON.stringify(dialog.data.definition, null, 2)
  }
  dialog.activeTab = tab
}

function onDesignerSave(definition: unknown) {
  dialog.data.definition = definition
  dialog.definitionText = JSON.stringify(definition, null, 2)
  toast.success('工作流定义已更新')
}

function onDesignerValidate(definition: unknown) {
  dialog.data.definition = definition
  dialog.definitionText = JSON.stringify(definition, null, 2)
  toast.success('工作流定义已验证')
}

async function saveWorkflow() {
  try {
    const definition = JSON.parse(dialog.definitionText)
    if (dialog.isEdit) {
      await workflowsApi.update(dialog.data.id, { name: dialog.data.name, description: dialog.data.description, definition, status: dialog.data.status })
      toast.success('工作流更新成功')
    } else {
      await workflowsApi.create({ id: dialog.data.id, name: dialog.data.name, description: dialog.data.description, definition })
      toast.success('工作流创建成功')
    }
    dialog.show = false
    loadWorkflows()
  } catch (error: unknown) {
    const msg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
    toast.error(msg || '保存失败')
  }
}

function executeWorkflow() { executeDialog.inputText = '{}'; executeDialog.show = true }

async function confirmExecute() {
  if (!selectedWorkflow.value) return
  try {
    let input = {}
    if (executeDialog.inputText) { try { input = JSON.parse(executeDialog.inputText) } catch { toast.error('输入 JSON 格式错误'); return } }
    await workflowsApi.execute(selectedWorkflow.value.id, { input })
    toast.success('工作流执行已启动')
    executeDialog.show = false
    loadExecutions(selectedWorkflow.value.id)
  } catch (error: unknown) {
    const msg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
    toast.error(msg || '执行失败')
  }
}

async function viewExecution(exec: Execution) {
  try { executionDialog.execution = await workflowsApi.getExecution(exec.id); executionDialog.show = true }
  catch { toast.error('加载执行详情失败') }
}

async function cancelExecution(exec: Execution) {
  try { await workflowsApi.cancelExecution(exec.id); toast.success('执行已取消'); if (selectedWorkflow.value) loadExecutions(selectedWorkflow.value.id) }
  catch { toast.error('取消失败') }
}

function confirmDelete() { if (!selectedWorkflow.value) return; deleteDialog.workflow = selectedWorkflow.value; deleteDialog.show = true }

async function deleteWorkflow() {
  if (!deleteDialog.workflow) return
  try {
    await workflowsApi.delete(deleteDialog.workflow.id)
    toast.success('工作流删除成功')
    deleteDialog.show = false
    selectedWorkflow.value = null
    loadWorkflows()
  } catch { toast.error('删除失败') }
}

onMounted(() => { loadWorkflows() })
</script>

<style scoped>
.workflows-page { padding: 24px 32px; height: 100%; overflow-y: auto; }

.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 24px; }

.page-title { font-size: 20px; font-weight: 700; color: var(--text-primary); margin: 0; }

.btn-primary {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 16px; background: var(--accent); color: #fff;
  border: none; border-radius: 6px; font-size: 13px; font-weight: 600; cursor: pointer;
}
.btn-primary:hover { background: var(--accent-hover); }

.btn-ghost { padding: 8px 16px; background: transparent; color: var(--text-secondary); border: 1px solid var(--border); border-radius: 6px; font-size: 13px; cursor: pointer; }
.btn-ghost:hover { background: var(--bg-overlay); }

.btn-icon-label {
  display: flex; align-items: center; gap: 5px;
  padding: 6px 12px; border: 1px solid var(--border); border-radius: 6px;
  background: transparent; color: var(--text-secondary); font-size: 13px; cursor: pointer;
}
.btn-success { color: #16a34a; border-color: rgba(34,197,94,0.4); }
.btn-success:hover { background: rgba(34,197,94,0.08); }
.btn-success:disabled { opacity: 0.4; cursor: not-allowed; }

.btn-ghost-sm { display: flex; align-items: center; gap: 5px; padding: 5px 10px; background: transparent; color: var(--text-secondary); border: 1px solid var(--border); border-radius: 6px; font-size: 12px; cursor: pointer; }
.btn-ghost-sm:hover { background: var(--bg-overlay); }

.btn-danger-sm { display: flex; align-items: center; width: 30px; height: 30px; background: transparent; color: #ef4444; border: 1px solid rgba(239,68,68,0.3); border-radius: 6px; cursor: pointer; justify-content: center; }
.btn-danger-sm:hover { background: rgba(239,68,68,0.08); }

.btn-danger { padding: 8px 16px; background: #ef4444; color: #fff; border: none; border-radius: 6px; font-size: 13px; font-weight: 600; cursor: pointer; }
.btn-danger:hover { background: #dc2626; }

.btn-icon-close { display: flex; align-items: center; justify-content: center; width: 32px; height: 32px; background: transparent; border: none; cursor: pointer; color: var(--text-secondary); border-radius: 6px; }
.btn-icon-close:hover { background: var(--bg-overlay); }

.wf-layout { display: grid; grid-template-columns: 240px 1fr; gap: 16px; }

.wf-sidebar { background: var(--bg-elevated); border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }

.sidebar-title { padding: 12px 16px; font-size: 11px; font-weight: 600; color: var(--text-tertiary); text-transform: uppercase; letter-spacing: 0.05em; border-bottom: 1px solid var(--border); background: var(--bg-overlay); }

.wf-item { padding: 12px 16px; cursor: pointer; border-bottom: 1px solid var(--border-subtle); transition: background 0.1s; }
.wf-item:hover { background: var(--bg-overlay); }
.wf-item.active { background: var(--accent-dim); }

.wf-item-top { display: flex; align-items: center; justify-content: space-between; margin-bottom: 3px; }

.wf-name { font-size: 13px; font-weight: 500; color: var(--text-primary); }
.wf-item.active .wf-name { color: var(--accent); }

.wf-meta { font-size: 11px; color: var(--text-tertiary); font-family: monospace; }

.badge { display: inline-block; padding: 2px 6px; border-radius: 4px; font-size: 10px; font-weight: 600; }
.badge-success { background: rgba(34,197,94,0.15); color: #16a34a; }
.badge-warning { background: rgba(234,179,8,0.15); color: #ca8a04; }
.badge-info    { background: rgba(59,130,246,0.15); color: #3b82f6; }
.badge-error   { background: rgba(239,68,68,0.15); color: #ef4444; }
.badge-neutral { background: var(--bg-overlay); color: var(--text-secondary); border: 1px solid var(--border); }

.wf-detail { background: var(--bg-elevated); border: 1px solid var(--border); border-radius: 8px; padding: 20px; }

.wf-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; color: var(--text-tertiary); background: var(--bg-elevated); border: 1px solid var(--border); border-radius: 8px; min-height: 300px; }
.wf-empty p { font-size: 14px; margin: 0; }

.detail-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.detail-title { font-size: 18px; font-weight: 700; color: var(--text-primary); margin: 0; }
.detail-actions { display: flex; gap: 8px; align-items: center; }
.detail-desc { font-size: 13px; color: var(--text-secondary); margin: 0 0 16px; }

.stat-bar { display: flex; gap: 24px; padding: 12px 0; border-bottom: 1px solid var(--border); margin-bottom: 16px; }
.stat-item { text-align: center; }
.stat-value { font-size: 22px; font-weight: 700; color: var(--text-primary); }
.text-success { color: #16a34a; }
.text-error { color: #ef4444; }
.text-info { color: #3b82f6; }
.stat-label { font-size: 12px; color: var(--text-tertiary); }

.section-title { font-size: 13px; font-weight: 600; color: var(--text-secondary); margin-bottom: 10px; text-transform: uppercase; letter-spacing: 0.04em; }

.steps-list { display: flex; flex-direction: column; gap: 6px; }
.step-item { display: flex; align-items: flex-start; gap: 10px; padding: 8px 12px; background: var(--bg-overlay); border-radius: 6px; }
.step-icon { width: 28px; height: 28px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 12px; font-weight: 700; color: #fff; flex-shrink: 0; }
.step-name { font-size: 13px; font-weight: 500; color: var(--text-primary); }
.step-meta { font-size: 12px; color: var(--text-tertiary); }
.step-deps { font-size: 11px; color: var(--text-tertiary); margin-top: 2px; }

.data-table { }
.data-thead, .data-row { display: grid; grid-template-columns: 120px 100px 150px 80px; padding: 8px 0; align-items: center; gap: 8px; border-bottom: 1px solid var(--border-subtle); }
.data-thead { font-size: 11px; font-weight: 600; color: var(--text-tertiary); text-transform: uppercase; letter-spacing: 0.05em; border-bottom: 1px solid var(--border); }
.data-row { font-size: 13px; color: var(--text-primary); }
.data-row:last-child { border-bottom: none; }
.mono { font-family: monospace; font-size: 12px; color: var(--text-secondary); }
.text-sm { font-size: 12px; color: var(--text-secondary); }

.actions { display: flex; gap: 4px; }
.action-btn { display: flex; align-items: center; justify-content: center; width: 26px; height: 26px; border: 1px solid var(--border); border-radius: 4px; background: transparent; cursor: pointer; color: var(--text-secondary); }
.action-btn:hover { background: var(--bg-overlay); }
.action-danger:hover { background: rgba(239,68,68,0.1); color: #ef4444; border-color: rgba(239,68,68,0.3); }

.empty-state { padding: 20px; text-align: center; color: var(--text-tertiary); font-size: 13px; }

/* Modal */
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
.modal-card { background: var(--bg-elevated); border: 1px solid var(--border); border-radius: 10px; padding: 24px; width: 500px; max-width: 95vw; max-height: 90vh; overflow-y: auto; }
.modal-wide { width: 640px; }
.modal-sm { width: 380px; }
.modal-title { font-size: 16px; font-weight: 700; color: var(--text-primary); margin: 0 0 16px; }
.confirm-text { font-size: 14px; color: var(--text-secondary); margin: 0 0 20px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 20px; }

.form-group { margin-bottom: 14px; }
.form-group label { display: block; font-size: 12px; font-weight: 600; color: var(--text-secondary); margin-bottom: 6px; }
.form-group input, .form-group select, .form-group textarea { width: 100%; padding: 8px 10px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 6px; color: var(--text-primary); font-size: 13px; box-sizing: border-box; }
.form-group input:disabled { opacity: 0.6; }
.form-group textarea { resize: vertical; }

/* Fullscreen modal */
.modal-fullscreen { background: var(--bg-elevated); border: 1px solid var(--border); border-radius: 10px; width: 95vw; max-width: 1200px; height: 90vh; display: flex; flex-direction: column; overflow: hidden; }
.modal-fheader { display: flex; align-items: center; justify-content: space-between; padding: 16px 20px; border-bottom: 1px solid var(--border); }
.modal-fheader .modal-title { margin: 0; }
.modal-fbody { flex: 1; display: flex; overflow: hidden; }
.modal-sidebar { width: 260px; padding: 16px; border-right: 1px solid var(--border); overflow-y: auto; flex-shrink: 0; }
.modal-main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.modal-ffooter { display: flex; justify-content: flex-end; gap: 8px; padding: 12px 20px; border-top: 1px solid var(--border); }

.tab-bar { display: flex; gap: 4px; padding: 8px 16px; border-bottom: 1px solid var(--border); }
.tab-btn { padding: 6px 14px; border: none; background: transparent; color: var(--text-secondary); font-size: 13px; cursor: pointer; border-bottom: 2px solid transparent; }
.tab-btn.active { color: var(--accent); border-bottom-color: var(--accent); }

.designer-wrap { flex: 1; overflow: hidden; height: 100%; }
.json-wrap { flex: 1; padding: 16px; }
.json-editor { width: 100%; height: 100%; font-family: monospace; font-size: 13px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 6px; color: var(--text-primary); padding: 12px; resize: none; box-sizing: border-box; }

/* Execution detail */
.detail-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.detail-item { }
.detail-item.full { grid-column: 1/-1; }
.detail-label { font-size: 11px; font-weight: 600; color: var(--text-tertiary); text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 3px; }
.detail-value { font-size: 13px; color: var(--text-primary); }
.detail-value.mono { font-family: monospace; font-size: 12px; }
.detail-value.text-error { color: #ef4444; }

.steps-exec-list { display: flex; flex-direction: column; gap: 6px; margin-bottom: 16px; }
.step-exec-item { display: flex; align-items: center; gap: 10px; }
.step-exec-icon { width: 24px; height: 24px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 11px; font-weight: 700; color: #fff; }
.step-success { background: #16a34a; }
.step-info { background: #3b82f6; }
.step-warning { background: #ca8a04; }
.step-error { background: #ef4444; }
.step-neutral { background: #94a3b8; }
.step-exec-id { font-size: 13px; font-weight: 500; color: var(--text-primary); }
.step-exec-meta { font-size: 12px; color: var(--text-tertiary); }
</style>
