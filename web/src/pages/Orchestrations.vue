<template>
  <div class="orchestrations-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div>
        <h1 class="page-title">Agent 编排器</h1>
        <p class="page-subtitle">管理多 Agent 协作编排</p>
      </div>
      <button class="btn-primary" @click="openCreateDialog">
        <PlusIcon :size="15" /> 新建编排
      </button>
    </div>

    <!-- 主内容区 -->
    <div class="main-layout">
      <!-- 左侧列表 -->
      <div class="sidebar">
        <div class="sidebar-header">
          <span class="sidebar-title">编排列表</span>
          <span class="count-badge">{{ orchestrations.length }}</span>
        </div>
        <div class="sidebar-list">
          <div
            v-if="orchestrations.length === 0"
            class="empty-list"
          >
            暂无编排
          </div>
          <div
            v-for="orch in orchestrations"
            :key="orch.id"
            class="list-item"
            :class="{ active: selectedOrch?.id === orch.id }"
            @click="selectOrch(orch)"
          >
            <div class="list-item-name">{{ orch.name }}</div>
            <div class="list-item-meta">
              <span class="status-dot" :class="orch.status" />
              <span class="list-item-id">{{ orch.id }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧详情 -->
      <div class="detail-panel" v-if="selectedOrch">
        <!-- 详情头部 -->
        <div class="detail-header">
          <div class="detail-title-group">
            <h2 class="detail-title">{{ selectedOrch.name }}</h2>
            <p v-if="selectedOrch.description" class="detail-desc">{{ selectedOrch.description }}</p>
          </div>
          <div class="detail-actions">
            <button class="btn-success-sm" @click="openExecuteDialog">
              <PlayIcon :size="13" /> 执行
            </button>
            <button class="btn-icon" @click="openEditDialog" title="编辑">
              <PencilIcon :size="14" />
            </button>
            <button class="btn-icon btn-icon-danger" @click="confirmDelete" title="删除">
              <Trash2Icon :size="14" />
            </button>
          </div>
        </div>

        <!-- Tabs -->
        <div class="tabs-bar">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            class="tab-btn"
            :class="{ active: activeTab === tab.id }"
            @click="activeTab = tab.id"
          >
            {{ tab.label }}
          </button>
        </div>

        <!-- 设计器 tab -->
        <div v-show="activeTab === 'designer'" class="tab-content designer-tab">
          <div class="designer-container">
            <OrchestrationDesigner
              :definition="selectedOrch.definition"
              :agents="availableAgents"
              @save="onDesignerSave"
              @validate="onDesignerValidate"
            />
          </div>
        </div>

        <!-- JSON tab -->
        <div v-show="activeTab === 'json'" class="tab-content">
          <div class="json-panel">
            <div class="json-header">
              <span class="json-label">编排定义 (JSON)</span>
            </div>
            <textarea class="json-textarea" :value="definitionText" readonly rows="20" />
          </div>
        </div>

        <!-- 执行记录 tab -->
        <div v-show="activeTab === 'executions'" class="tab-content">
          <div v-if="executions.length === 0" class="empty-executions">
            <GitBranchIcon :size="32" />
            <p>暂无执行记录</p>
          </div>
          <table v-else class="exec-table">
            <thead>
              <tr>
                <th>执行 ID</th>
                <th>状态</th>
                <th>开始时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="exec in executions" :key="exec.id">
                <td class="exec-id">{{ exec.id }}</td>
                <td>
                  <span class="exec-status-badge" :class="exec.status">{{ exec.status }}</span>
                </td>
                <td class="exec-time">{{ formatDate(exec.start_time) }}</td>
                <td>
                  <button class="btn-icon-sm" @click="viewExecution(exec)" title="查看">
                    <EyeIcon :size="13" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 未选择状态 -->
      <div v-else class="detail-panel detail-empty">
        <GitBranchIcon :size="36" />
        <p>从左侧选择一个编排</p>
      </div>
    </div>

    <!-- 创建/编辑对话框 -->
    <div v-if="dialog.show" class="modal-overlay" @click.self="dialog.show = false">
      <div class="modal modal-lg">
        <div class="modal-header">
          <h3>{{ dialog.isEdit ? '编辑编排' : '新建编排' }}</h3>
          <button class="btn-icon" @click="dialog.show = false"><XIcon :size="16" /></button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>ID <span class="required">*</span></label>
            <input v-model="dialog.data.id" type="text" :disabled="dialog.isEdit" placeholder="唯一标识符" />
          </div>
          <div class="form-group">
            <label>名称 <span class="required">*</span></label>
            <input v-model="dialog.data.name" type="text" placeholder="编排名称" />
          </div>
          <div class="form-group">
            <label>描述</label>
            <textarea v-model="dialog.data.description" rows="2" placeholder="可选描述" />
          </div>
          <div class="form-group">
            <label>编排定义 (JSON) <span class="required">*</span></label>
            <textarea v-model="dialog.definitionText" rows="15" class="code-textarea" spellcheck="false" />
            <span v-if="dialog.jsonError" class="field-error">{{ dialog.jsonError }}</span>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-ghost" @click="dialog.show = false">取消</button>
          <button class="btn-primary" @click="saveOrchestration">保存</button>
        </div>
      </div>
    </div>

    <!-- 执行对话框 -->
    <div v-if="executeDialog.show" class="modal-overlay" @click.self="executeDialog.show = false">
      <div class="modal">
        <div class="modal-header">
          <h3>执行编排</h3>
          <button class="btn-icon" @click="executeDialog.show = false"><XIcon :size="16" /></button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>输入</label>
            <textarea v-model="executeDialog.input" rows="5" placeholder="请输入初始输入..." />
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-ghost" @click="executeDialog.show = false">取消</button>
          <button class="btn-primary" @click="executeOrchestration">执行</button>
        </div>
      </div>
    </div>

    <!-- 删除确认 -->
    <div v-if="deleteDialog.show" class="modal-overlay" @click.self="deleteDialog.show = false">
      <div class="modal modal-sm">
        <div class="modal-header">
          <h3>确认删除</h3>
          <button class="btn-icon" @click="deleteDialog.show = false"><XIcon :size="16" /></button>
        </div>
        <div class="modal-body">
          <p>确定要删除编排 <strong>"{{ deleteDialog.orch?.name }}"</strong> 吗？此操作不可撤销。</p>
        </div>
        <div class="modal-footer">
          <button class="btn-ghost" @click="deleteDialog.show = false">取消</button>
          <button class="btn-danger" @click="deleteOrchestration">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue'
import { toast } from 'vue-sonner'
import { PlusIcon, PlayIcon, PencilIcon, Trash2Icon, EyeIcon, XIcon, GitBranchIcon } from 'lucide-vue-next'
import { orchestrationApi, type Orchestration, type ExecutionContext } from '@/api/orchestration'
import OrchestrationDesigner from '@/components/orchestration/OrchestrationDesigner.vue'

const orchestrations = ref<Orchestration[]>([])
const selectedOrch = ref<Orchestration | null>(null)
const executions = ref<ExecutionContext[]>([])
const activeTab = ref('designer')

const tabs = [
  { id: 'designer', label: '设计器' },
  { id: 'json', label: 'JSON' },
  { id: 'executions', label: '执行记录' },
]

const dialog = reactive({
  show: false,
  isEdit: false,
  jsonError: '',
  data: {
    id: '',
    name: '',
    description: '',
  },
  definitionText: '',
})

const executeDialog = reactive({
  show: false,
  input: '',
})

const deleteDialog = reactive({
  show: false,
  orch: null as Orchestration | null,
})

const availableAgents = ref([
  { id: 'default', name: '默认 Agent' },
  { id: 'product_manager', name: '产品经理' },
  { id: 'developer', name: '开发工程师' },
  { id: 'tester', name: '测试工程师' },
])

const definitionText = computed(() => {
  if (!selectedOrch.value) return ''
  return JSON.stringify(selectedOrch.value.definition, null, 2)
})

onMounted(() => {
  loadOrchestrations()
})

async function loadOrchestrations() {
  try {
    orchestrations.value = await orchestrationApi.list()
  } catch (error: any) {
    toast.error('加载编排列表失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

async function selectOrch(orch: Orchestration) {
  selectedOrch.value = orch
  loadExecutions(orch.id)
}

async function loadExecutions(orchId: string) {
  try {
    executions.value = await orchestrationApi.listExecutions(orchId)
  } catch (error: any) {
    toast.error('加载执行记录失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

function openCreateDialog() {
  dialog.isEdit = false
  dialog.jsonError = ''
  dialog.data = { id: '', name: '', description: '' }
  dialog.definitionText = JSON.stringify({
    nodes: [{ id: 'agent_1', type: 'agent', agent_id: 'default', name: 'Agent', position: { x: 100, y: 100 } }],
    edges: [],
    start_node_id: 'agent_1',
  }, null, 2)
  dialog.show = true
}

function openEditDialog() {
  if (!selectedOrch.value) return
  dialog.isEdit = true
  dialog.jsonError = ''
  dialog.data = {
    id: selectedOrch.value.id,
    name: selectedOrch.value.name,
    description: selectedOrch.value.description,
  }
  dialog.definitionText = JSON.stringify(selectedOrch.value.definition, null, 2)
  dialog.show = true
}

async function saveOrchestration() {
  dialog.jsonError = ''
  if (!dialog.data.id || !dialog.data.name) {
    toast.error('ID 和名称不能为空')
    return
  }
  let definition: any
  try {
    definition = JSON.parse(dialog.definitionText)
  } catch (e: any) {
    dialog.jsonError = 'JSON 格式错误: ' + e.message
    return
  }
  if (!Array.isArray(definition.nodes) || !Array.isArray(definition.edges)) {
    dialog.jsonError = '编排定义必须包含 nodes 和 edges 数组'
    return
  }
  try {
    if (dialog.isEdit) {
      await orchestrationApi.update(dialog.data.id, { name: dialog.data.name, description: dialog.data.description, definition })
    } else {
      await orchestrationApi.create({ id: dialog.data.id, name: dialog.data.name, description: dialog.data.description, definition })
    }
    dialog.show = false
    toast.success(dialog.isEdit ? '编排已更新' : '编排已创建')
    loadOrchestrations()
  } catch (error: any) {
    toast.error('保存失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

function confirmDelete() {
  if (!selectedOrch.value) return
  deleteDialog.orch = selectedOrch.value
  deleteDialog.show = true
}

async function deleteOrchestration() {
  if (!deleteDialog.orch) return
  try {
    await orchestrationApi.delete(deleteDialog.orch.id)
    deleteDialog.show = false
    selectedOrch.value = null
    toast.success('编排已删除')
    loadOrchestrations()
  } catch (error: any) {
    toast.error('删除失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

function openExecuteDialog() {
  executeDialog.input = ''
  executeDialog.show = true
}

async function executeOrchestration() {
  if (!selectedOrch.value) return
  try {
    const response = await orchestrationApi.execute(selectedOrch.value.id, { input: executeDialog.input })
    executeDialog.show = false
    toast.success(`执行已开始，执行 ID: ${response.execution_id}`)
    loadExecutions(selectedOrch.value.id)
  } catch (error: any) {
    toast.error('执行失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

function viewExecution(exec: ExecutionContext) {
  console.log('View execution:', exec)
}

async function onDesignerSave(definition: any) {
  if (!selectedOrch.value) return
  try {
    await orchestrationApi.update(selectedOrch.value.id, { definition })
    selectedOrch.value = await orchestrationApi.get(selectedOrch.value.id)
    toast.success('编排已保存')
  } catch (error: any) {
    toast.error('保存失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

async function onDesignerValidate(definition: any) {
  try {
    await orchestrationApi.validate(definition)
    toast.success('编排定义有效')
  } catch (error: any) {
    toast.error('验证失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

function formatDate(date: string) {
  return new Date(date).toLocaleString('zh-CN')
}
</script>

<style scoped>
.orchestrations-page {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  height: 100%;
  box-sizing: border-box;
  overflow: hidden;
}

/* 页头 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-shrink: 0;
}
.page-title { font-size: 18px; font-weight: 600; color: var(--text-primary); margin: 0; }
.page-subtitle { font-size: 13px; color: var(--text-secondary); margin: 2px 0 0; }

/* 主布局 */
.main-layout {
  display: flex;
  gap: 16px;
  flex: 1;
  overflow: hidden;
}

/* 左侧 */
.sidebar {
  width: 220px;
  flex-shrink: 0;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.sidebar-header {
  padding: 12px 14px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.sidebar-title { font-size: 12px; font-weight: 600; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.04em; }
.count-badge { font-size: 11px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 10px; padding: 1px 7px; color: var(--text-secondary); }
.sidebar-list { flex: 1; overflow-y: auto; padding: 6px; }
.empty-list { padding: 20px 12px; text-align: center; color: var(--text-tertiary); font-size: 13px; }

.list-item {
  padding: 9px 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.1s;
  margin-bottom: 2px;
}
.list-item:hover { background: var(--bg-app); }
.list-item.active { background: var(--accent-dim); }
.list-item-name { font-size: 13px; font-weight: 500; color: var(--text-primary); }
.list-item-meta { display: flex; align-items: center; gap: 5px; margin-top: 3px; }
.list-item-id { font-size: 11px; color: var(--text-tertiary); font-family: monospace; }

.status-dot {
  width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0;
}
.status-dot.active { background: var(--green, #16a34a); }
.status-dot.draft { background: #f59e0b; }
.status-dot.disabled { background: var(--text-tertiary); }

/* 右侧详情 */
.detail-panel {
  flex: 1;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
}

.detail-empty {
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--text-tertiary);
  font-size: 13px;
}

.detail-header {
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  flex-shrink: 0;
}
.detail-title-group { flex: 1; min-width: 0; }
.detail-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin: 0; }
.detail-desc { font-size: 12px; color: var(--text-secondary); margin: 3px 0 0; }
.detail-actions { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }

/* Tabs */
.tabs-bar {
  display: flex;
  padding: 0 16px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}
.tab-btn {
  padding: 10px 14px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.1s;
  margin-bottom: -1px;
}
.tab-btn:hover { color: var(--text-primary); }
.tab-btn.active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 500; }

.tab-content {
  flex: 1;
  overflow: auto;
  padding: 16px;
}
.designer-tab { padding: 0; }

.designer-container {
  height: 100%;
}

/* JSON panel */
.json-panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
  height: 100%;
}
.json-header { display: flex; align-items: center; justify-content: space-between; }
.json-label { font-size: 12px; font-weight: 600; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.04em; }
.json-textarea {
  flex: 1;
  width: 100%;
  box-sizing: border-box;
  padding: 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 12px;
  font-family: monospace;
  resize: none;
  min-height: 400px;
}

/* 执行记录 */
.empty-executions {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 48px;
  color: var(--text-tertiary);
  font-size: 13px;
}
.exec-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.exec-table th {
  text-align: left;
  padding: 8px 12px;
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  border-bottom: 1px solid var(--border);
}
.exec-table td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border-subtle, var(--border));
  color: var(--text-primary);
}
.exec-id { font-family: monospace; font-size: 12px; color: var(--text-secondary); }
.exec-time { font-size: 12px; color: var(--text-secondary); }
.exec-status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 500;
}
.exec-status-badge.completed { background: rgba(22,163,74,0.12); color: #16a34a; }
.exec-status-badge.running { background: rgba(59,130,246,0.12); color: #3b82f6; }
.exec-status-badge.paused { background: rgba(245,158,11,0.12); color: #b45309; }
.exec-status-badge.failed { background: rgba(239,68,68,0.12); color: #dc2626; }

/* 按钮 */
.btn-primary {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 14px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 13px;
  cursor: pointer;
  white-space: nowrap;
}
.btn-primary:hover { background: var(--accent-hover, var(--accent)); opacity: 0.9; }

.btn-success-sm {
  display: flex; align-items: center; gap: 5px;
  padding: 5px 10px;
  background: rgba(22,163,74,0.1);
  border: 1px solid rgba(22,163,74,0.3);
  border-radius: 6px;
  color: #16a34a;
  font-size: 12px;
  cursor: pointer;
}
.btn-success-sm:hover { background: rgba(22,163,74,0.18); }

.btn-ghost {
  padding: 7px 14px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 13px;
  cursor: pointer;
}
.btn-ghost:hover { background: var(--bg-app); }

.btn-danger {
  padding: 7px 14px;
  background: transparent;
  border: 1px solid rgba(239,68,68,0.4);
  border-radius: 6px;
  color: #ef4444;
  font-size: 13px;
  cursor: pointer;
}
.btn-danger:hover { background: rgba(239,68,68,0.08); }

.btn-icon {
  width: 30px; height: 30px;
  display: flex; align-items: center; justify-content: center;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  cursor: pointer;
}
.btn-icon:hover { background: var(--bg-app); }
.btn-icon-danger:hover { color: #ef4444; border-color: rgba(239,68,68,0.3); background: rgba(239,68,68,0.08); }

.btn-icon-sm {
  width: 26px; height: 26px;
  display: flex; align-items: center; justify-content: center;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  cursor: pointer;
}
.btn-icon-sm:hover { background: var(--bg-app); }

/* 模态框 */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  width: 480px;
  max-width: calc(100vw - 40px);
  max-height: calc(100vh - 80px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.modal-sm { width: 380px; }
.modal-lg { width: 680px; }

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}
.modal-header h3 { font-size: 15px; font-weight: 600; color: var(--text-primary); margin: 0; }

.modal-body {
  padding: 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.modal-body p { font-size: 14px; color: var(--text-primary); margin: 0; }

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 14px 20px;
  border-top: 1px solid var(--border);
  flex-shrink: 0;
}

/* 表单 */
.form-group { display: flex; flex-direction: column; gap: 5px; }
.form-group label { font-size: 12px; font-weight: 600; color: var(--text-secondary); }
.form-group input,
.form-group textarea,
.form-group select {
  padding: 8px 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  width: 100%;
  box-sizing: border-box;
}
.form-group input:focus,
.form-group textarea:focus { outline: none; border-color: var(--accent); }
.form-group textarea { resize: vertical; }
.form-group input:disabled { opacity: 0.5; cursor: not-allowed; }
.code-textarea { font-family: monospace; font-size: 12px; }

.required { color: #ef4444; }
.field-error { font-size: 12px; color: #ef4444; }
</style>
