<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">流程管理</h1>
        <p class="page-desc">统一管理对话流和任务流，支持可视化设计</p>
      </div>
      <div class="header-right">
        <div class="type-filter">
          <button
            class="filter-btn"
            :class="{ active: typeFilter === '' }"
            @click="typeFilter = ''"
          >全部</button>
          <button
            class="filter-btn"
            :class="{ active: typeFilter === 'conversation' }"
            @click="typeFilter = 'conversation'"
          >对话流</button>
          <button
            class="filter-btn"
            :class="{ active: typeFilter === 'task' }"
            @click="typeFilter = 'task'"
          >任务流</button>
        </div>
        <div class="dropdown">
          <button class="btn-secondary" @click="showTemplates = !showTemplates">
            <FileTextIcon :size="16" /> 从模板创建
            <ChevronDownIcon :size="14" />
          </button>
          <div v-if="showTemplates" class="dropdown-menu">
            <button
              v-for="tpl in templates"
              :key="tpl.id"
              class="dropdown-item"
              @click="createFromTemplate(tpl)"
            >
              <span class="tpl-name">{{ tpl.name }}</span>
              <span class="tpl-desc">{{ tpl.description }}</span>
            </button>
          </div>
        </div>
        <button class="btn-primary" @click="openCreateDialog">
          <PlusIcon :size="16" /> 新建流程
        </button>
      </div>
    </div>

    <!-- 流程列表 -->
    <div class="flow-list">
      <div v-if="loading" class="loading-state">
        <LoaderIcon :size="24" class="spin" />
        <span>加载中...</span>
      </div>
      <div v-else-if="!flows || flows.length === 0" class="empty-state">
        <GitBranchIcon :size="48" />
        <p>暂无流程</p>
        <button class="btn-primary" @click="openCreateDialog">创建第一个流程</button>
      </div>
      <div v-else class="flow-grid">
        <div
          v-for="flow in filteredFlows"
          :key="flow.id"
          class="flow-card"
          :class="{ active: flow.status === 'active' }"
        >
          <div class="card-header">
            <div class="card-type" :class="flow.type">
              {{ flow.type === 'conversation' ? '对话流' : '任务流' }}
            </div>
            <div class="card-status" :class="flow.status">
              {{ flow.status === 'active' ? '已启用' : flow.status === 'draft' ? '草稿' : '已停用' }}
            </div>
          </div>
          <h3 class="card-title">{{ flow.name }}</h3>
          <p class="card-desc">{{ flow.description || '暂无描述' }}</p>
          <div class="card-meta">
            <span>{{ flow.definition?.nodes?.length || 0 }} 个节点</span>
            <span>{{ formatDate(flow.updated_at) }}</span>
          </div>
          <div class="card-actions">
            <button class="action-btn" @click="openEditDialog(flow)" title="编辑">
              <EditIcon :size="14" />
            </button>
            <button
              v-if="flow.status !== 'active'"
              class="action-btn"
              @click="activateFlow(flow.id)"
              title="启用"
            >
              <PlayIcon :size="14" />
            </button>
            <button
              v-else
              class="action-btn"
              @click="deactivateFlow(flow.id)"
              title="停用"
            >
              <PauseIcon :size="14" />
            </button>
            <button class="action-btn" @click="executeFlow(flow)" title="执行">
              <RocketIcon :size="14" />
            </button>
            <button class="action-btn danger" @click="deleteFlow(flow.id)" title="删除">
              <Trash2Icon :size="14" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 创建/编辑对话框 -->
    <div v-if="dialog.show" class="modal-overlay" @click.self="dialog.show = false">
      <div class="modal-fullscreen">
        <div class="modal-fheader">
          <div class="header-info">
            <h2 class="modal-title">{{ dialog.isEdit ? '编辑流程' : '新建流程' }}</h2>
            <div class="header-form">
              <input
                v-model="dialog.data.name"
                type="text"
                class="name-input"
                placeholder="流程名称"
              />
              <input
                v-model="dialog.data.description"
                type="text"
                class="desc-input"
                placeholder="描述（可选）"
              />
              <select v-model="dialog.data.type" class="type-select">
                <option value="conversation">对话流</option>
                <option value="task">任务流</option>
              </select>
            </div>
          </div>
          <div class="header-actions">
            <div class="tab-switch">
              <button class="switch-btn" :class="{ active: dialog.activeTab === 'designer' }" @click="dialog.activeTab = 'designer'">设计器</button>
              <button class="switch-btn" :class="{ active: dialog.activeTab === 'json' }" @click="dialog.activeTab = 'json'">JSON</button>
            </div>
            <button class="btn-ghost-sm" @click="dialog.show = false">取消</button>
            <button class="btn-primary-sm" @click="saveFlow">保存</button>
            <button class="btn-icon-close" @click="dialog.show = false"><XIcon :size="18" /></button>
          </div>
        </div>
        <div class="modal-fbody">
          <div v-if="dialog.activeTab === 'designer'" class="designer-wrap">
            <FlowDesigner
              :definition="dialog.data.definition"
              :agents="agents"
              :flows="flows"
              @save="onDesignerSave"
              @validate="onDesignerValidate"
            />
          </div>
          <div v-if="dialog.activeTab === 'json'" class="json-wrap">
            <textarea v-model="dialog.definitionText" class="json-editor" spellcheck="false" />
            <span v-if="dialog.jsonError" class="field-error">{{ dialog.jsonError }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 执行监控面板 -->
    <div v-if="executionPanel.show" class="modal-overlay" @click.self="executionPanel.show = false">
      <div class="execution-panel-wrap">
        <FlowExecutionPanel
          :flow-id="executionPanel.flowId"
          :execution-id="executionPanel.executionId"
          :flow-name="executionPanel.flowName"
          @close="executionPanel.show = false"
          @completed="executionPanel.show = false"
        />
      </div>
    </div>

    <!-- 执行对话框 -->
    <div v-if="executeDialog.show" class="modal-overlay" @click.self="executeDialog.show = false">
      <div class="modal">
        <div class="modal-header">
          <h3>执行流程</h3>
          <button class="btn-icon" @click="executeDialog.show = false"><XIcon :size="16" /></button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>输入内容</label>
            <textarea v-model="executeDialog.input" rows="4" placeholder="输入流程执行的初始内容..." />
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-ghost" @click="executeDialog.show = false">取消</button>
          <button class="btn-primary" @click="confirmExecute">执行</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  PlusIcon, EditIcon, Trash2Icon, PlayIcon, PauseIcon, RocketIcon,
  LoaderIcon, GitBranchIcon, XIcon, FileTextIcon, ChevronDownIcon
} from 'lucide-vue-next'
import FlowDesigner from '@/components/flow/FlowDesigner.vue'
import FlowExecutionPanel from '@/components/flow/FlowExecutionPanel.vue'

interface FlowNode { id: string; type: string; name: string; position: { x: number; y: number } }
interface FlowEdge { id: string; source: string; target: string }
interface FlowDefinition { nodes: FlowNode[]; edges: FlowEdge[]; start_node_id?: string }
interface Flow {
  id: string
  name: string
  description: string
  type: 'conversation' | 'task'
  definition: FlowDefinition
  status: string
  created_at: string
  updated_at: string
}
interface Agent { id: string; name: string }

const loading = ref(true)
const flows = ref<Flow[]>([])
const agents = ref<Agent[]>([])
const typeFilter = ref('')
const showTemplates = ref(false)

// 预置模板
interface FlowTemplate {
  id: string
  name: string
  description: string
  type: 'conversation' | 'task'
  definition: FlowDefinition
}

const templates: FlowTemplate[] = [
  {
    id: 'customer_service',
    name: '客服对话流程',
    description: '意图识别 → 分支处理 → 统一回复',
    type: 'conversation',
    definition: {
      nodes: [
        { id: 'start_1', type: 'start', name: '开始', position: { x: 250, y: 30 } },
        { id: 'agent_1', type: 'agent', name: '意图识别', position: { x: 250, y: 120 } },
        { id: 'condition_1', type: 'condition', name: '意图判断', position: { x: 250, y: 220 } },
        { id: 'agent_2', type: 'agent', name: '查询处理', position: { x: 100, y: 320 } },
        { id: 'agent_3', type: 'agent', name: '投诉处理', position: { x: 250, y: 320 } },
        { id: 'agent_4', type: 'agent', name: '闲聊回复', position: { x: 400, y: 320 } },
        { id: 'end_1', type: 'end', name: '结束', position: { x: 250, y: 420 } }
      ],
      edges: [
        { id: 'e1', source: 'start_1', target: 'agent_1' },
        { id: 'e2', source: 'agent_1', target: 'condition_1' },
        { id: 'e3', source: 'condition_1', target: 'agent_2' },
        { id: 'e4', source: 'condition_1', target: 'agent_3' },
        { id: 'e5', source: 'condition_1', target: 'agent_4' },
        { id: 'e6', source: 'agent_2', target: 'end_1' },
        { id: 'e7', source: 'agent_3', target: 'end_1' },
        { id: 'e8', source: 'agent_4', target: 'end_1' }
      ],
      start_node_id: 'start_1'
    }
  },
  {
    id: 'approval_flow',
    name: '审批流程',
    description: '初审 → 人工审批 → 结果通知',
    type: 'conversation',
    definition: {
      nodes: [
        { id: 'start_1', type: 'start', name: '开始', position: { x: 250, y: 30 } },
        { id: 'agent_1', type: 'agent', name: '初审', position: { x: 250, y: 120 } },
        { id: 'condition_1', type: 'condition', name: '是否需要人工', position: { x: 250, y: 220 } },
        { id: 'human_1', type: 'human', name: '人工审批', position: { x: 150, y: 320 } },
        { id: 'agent_2', type: 'agent', name: '自动通过', position: { x: 350, y: 320 } },
        { id: 'agent_3', type: 'agent', name: '结果通知', position: { x: 250, y: 420 } },
        { id: 'end_1', type: 'end', name: '结束', position: { x: 250, y: 520 } }
      ],
      edges: [
        { id: 'e1', source: 'start_1', target: 'agent_1' },
        { id: 'e2', source: 'agent_1', target: 'condition_1' },
        { id: 'e3', source: 'condition_1', target: 'human_1' },
        { id: 'e4', source: 'condition_1', target: 'agent_2' },
        { id: 'e5', source: 'human_1', target: 'agent_3' },
        { id: 'e6', source: 'agent_2', target: 'agent_3' },
        { id: 'e7', source: 'agent_3', target: 'end_1' }
      ],
      start_node_id: 'start_1'
    }
  },
  {
    id: 'data_pipeline',
    name: '数据处理流程',
    description: '数据获取 → 并行处理 → 汇总输出',
    type: 'task',
    definition: {
      nodes: [
        { id: 'start_1', type: 'start', name: '开始', position: { x: 250, y: 30 } },
        { id: 'agent_1', type: 'agent', name: '数据获取', position: { x: 250, y: 120 } },
        { id: 'parallel_1', type: 'parallel', name: '并行处理', position: { x: 250, y: 220 } },
        { id: 'agent_2', type: 'agent', name: '数据清洗', position: { x: 100, y: 320 } },
        { id: 'agent_3', type: 'agent', name: '数据分析', position: { x: 250, y: 320 } },
        { id: 'agent_4', type: 'agent', name: '数据可视化', position: { x: 400, y: 320 } },
        { id: 'agent_5', type: 'agent', name: '汇总报告', position: { x: 250, y: 420 } },
        { id: 'end_1', type: 'end', name: '结束', position: { x: 250, y: 520 } }
      ],
      edges: [
        { id: 'e1', source: 'start_1', target: 'agent_1' },
        { id: 'e2', source: 'agent_1', target: 'parallel_1' },
        { id: 'e3', source: 'parallel_1', target: 'agent_2' },
        { id: 'e4', source: 'parallel_1', target: 'agent_3' },
        { id: 'e5', source: 'parallel_1', target: 'agent_4' },
        { id: 'e6', source: 'agent_2', target: 'agent_5' },
        { id: 'e7', source: 'agent_3', target: 'agent_5' },
        { id: 'e8', source: 'agent_4', target: 'agent_5' },
        { id: 'e9', source: 'agent_5', target: 'end_1' }
      ],
      start_node_id: 'start_1'
    }
  },
  {
    id: 'retry_flow',
    name: '重试流程',
    description: '执行任务 → 失败重试 → 成功结束',
    type: 'task',
    definition: {
      nodes: [
        { id: 'start_1', type: 'start', name: '开始', position: { x: 250, y: 30 } },
        { id: 'loop_1', type: 'loop', name: '重试循环', position: { x: 250, y: 120 } },
        { id: 'agent_1', type: 'agent', name: '执行任务', position: { x: 250, y: 220 } },
        { id: 'condition_1', type: 'condition', name: '执行结果', position: { x: 250, y: 320 } },
        { id: 'end_1', type: 'end', name: '成功结束', position: { x: 250, y: 420 } }
      ],
      edges: [
        { id: 'e1', source: 'start_1', target: 'loop_1' },
        { id: 'e2', source: 'loop_1', target: 'agent_1' },
        { id: 'e3', source: 'agent_1', target: 'condition_1' },
        { id: 'e4', source: 'condition_1', target: 'end_1' },
        { id: 'e5', source: 'condition_1', target: 'loop_1' }
      ],
      start_node_id: 'start_1'
    }
  }
]

const filteredFlows = computed(() => {
  if (!flows.value) return []
  if (!typeFilter.value) return flows.value
  return flows.value.filter(f => f.type === typeFilter.value)
})

// 对话框状态
const dialog = ref<{
  show: boolean
  isEdit: boolean
  data: {
    id: string
    name: string
    description: string
    type: 'conversation' | 'task'
    definition: FlowDefinition
  }
  activeTab: 'designer' | 'json'
  definitionText: string
  jsonError: string
}>({
  show: false,
  isEdit: false,
  data: { id: '', name: '', description: '', type: 'conversation', definition: { nodes: [], edges: [] } },
  activeTab: 'designer',
  definitionText: '',
  jsonError: ''
})

// 执行对话框
const executeDialog = ref<{
  show: boolean
  flowId: string
  flowName: string
  input: string
}>({
  show: false,
  flowId: '',
  flowName: '',
  input: ''
})

// 执行监控面板
const executionPanel = ref<{
  show: boolean
  flowId: string
  flowName: string
  executionId: string
}>({
  show: false,
  flowId: '',
  flowName: '',
  executionId: ''
})

onMounted(async () => {
  await Promise.all([loadFlows(), loadAgents()])
})

async function loadFlows() {
  try {
    const res = await fetch('/api/flows')
    if (res.ok) {
      const data = await res.json()
      flows.value = Array.isArray(data) ? data : []
    }
  } catch (e) {
    console.error('Failed to load flows:', e)
    flows.value = []
  } finally {
    loading.value = false
  }
}

async function loadAgents() {
  try {
    const res = await fetch('/api/agents')
    if (res.ok) {
      const data = await res.json()
      // API 返回 { code, message, data: [...] } 包装格式
      agents.value = Array.isArray(data) ? data : (Array.isArray(data?.data) ? data.data : [])
    }
  } catch (e) {
    console.error('Failed to load agents:', e)
    agents.value = []
  }
}

function createFromTemplate(tpl: FlowTemplate) {
  showTemplates.value = false
  dialog.value = {
    show: true,
    isEdit: false,
    data: {
      id: generateId(),
      name: tpl.name,
      description: tpl.description,
      type: tpl.type,
      definition: JSON.parse(JSON.stringify(tpl.definition))
    },
    activeTab: 'designer',
    definitionText: '',
    jsonError: ''
  }
}

function openCreateDialog() {
  dialog.value = {
    show: true,
    isEdit: false,
    data: {
      id: generateId(),
      name: '',
      description: '',
      type: 'conversation',
      definition: {
        nodes: [
          { id: 'start_1', type: 'start', name: '开始', position: { x: 250, y: 50 } },
          { id: 'end_1', type: 'end', name: '结束', position: { x: 250, y: 400 } }
        ],
        edges: [],
        start_node_id: 'start_1'
      }
    },
    activeTab: 'designer',
    definitionText: '',
    jsonError: ''
  }
}

function openEditDialog(flow: Flow) {
  dialog.value = {
    show: true,
    isEdit: true,
    data: {
      id: flow.id,
      name: flow.name,
      description: flow.description,
      type: flow.type,
      definition: flow.definition || { nodes: [], edges: [] }
    },
    activeTab: 'designer',
    definitionText: JSON.stringify(flow.definition, null, 2),
    jsonError: ''
  }
}

async function saveFlow() {
  const url = dialog.value.isEdit ? `/api/flows/${dialog.value.data.id}` : '/api/flows'
  const method = dialog.value.isEdit ? 'PUT' : 'POST'

  try {
    const res = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(dialog.value.data)
    })
    if (res.ok) {
      dialog.value.show = false
      await loadFlows()
    } else {
      const err = await res.json()
      alert(err.error || '保存失败')
    }
  } catch (e) {
    console.error('Failed to save flow:', e)
    alert('保存失败')
  }
}

async function deleteFlow(id: string) {
  if (!confirm('确定要删除这个流程吗？')) return
  try {
    const res = await fetch(`/api/flows/${id}`, { method: 'DELETE' })
    if (res.ok) {
      flows.value = flows.value.filter(f => f.id !== id)
    }
  } catch (e) {
    console.error('Failed to delete flow:', e)
  }
}

async function activateFlow(id: string) {
  try {
    const res = await fetch(`/api/flows/${id}/activate`, { method: 'POST' })
    if (res.ok) {
      await loadFlows()
    }
  } catch (e) {
    console.error('Failed to activate flow:', e)
  }
}

async function deactivateFlow(id: string) {
  try {
    const res = await fetch(`/api/flows/${id}/deactivate`, { method: 'POST' })
    if (res.ok) {
      await loadFlows()
    }
  } catch (e) {
    console.error('Failed to deactivate flow:', e)
  }
}

function executeFlow(flow: Flow) {
  executeDialog.value = { show: true, flowId: flow.id, flowName: flow.name, input: '' }
}

async function confirmExecute() {
  try {
    const res = await fetch(`/api/flows/${executeDialog.value.flowId}/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ input: executeDialog.value.input })
    })
    if (res.ok) {
      const result = await res.json()
      executeDialog.value.show = false
      executionPanel.value = {
        show: true,
        flowId: executeDialog.value.flowId,
        flowName: executeDialog.value.flowName,
        executionId: result.execution_id
      }
    } else {
      const err = await res.json()
      alert(err.error || '执行失败')
    }
  } catch (e) {
    console.error('Failed to execute flow:', e)
    alert('执行失败')
  }
}

function onDesignerSave(def: FlowDefinition) {
  dialog.value.data.definition = def
}

function onDesignerValidate(def: FlowDefinition) {
  console.log('Validate:', def)
}

function generateId(): string {
  return 'flow_' + Math.random().toString(36).substring(2, 10)
}

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
}
</script>

<style scoped>
.page-container { flex: 1; height: 100%; overflow: hidden; display: flex; flex-direction: column; background: var(--bg-app); }
.page-header {
  display: flex; justify-content: space-between; align-items: flex-start;
  padding: 20px 24px; border-bottom: 1px solid var(--border);
}
.header-left { flex: 1; }
.page-title { font-size: 20px; font-weight: 600; color: var(--text-primary); margin: 0 0 4px 0; }
.page-desc { font-size: 13px; color: var(--text-secondary); margin: 0; }
.header-right { display: flex; align-items: center; gap: 12px; }
.type-filter { display: flex; gap: 4px; }
.filter-btn {
  padding: 6px 12px; background: var(--bg-elevated); border: 1px solid var(--border);
  border-radius: 6px; color: var(--text-secondary); font-size: 12px; cursor: pointer;
}
.filter-btn:hover { background: var(--bg-overlay); }
.filter-btn.active { background: var(--accent-dim); color: var(--accent); border-color: var(--accent); }
.btn-primary {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 16px; background: var(--accent); border: none;
  border-radius: 6px; color: #fff; font-size: 13px; font-weight: 500; cursor: pointer;
}
.btn-primary:hover { opacity: 0.9; }
.btn-secondary {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 12px; background: var(--bg-elevated); border: 1px solid var(--border);
  border-radius: 6px; color: var(--text-secondary); font-size: 13px; cursor: pointer;
}
.btn-secondary:hover { background: var(--bg-overlay); }
.dropdown { position: relative; }
.dropdown-menu {
  position: absolute; top: 100%; left: 0; margin-top: 4px;
  min-width: 240px; background: var(--bg-elevated); border: 1px solid var(--border);
  border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.15); z-index: 100;
  overflow: hidden;
}
.dropdown-item {
  display: flex; flex-direction: column; width: 100%;
  padding: 10px 14px; background: transparent; border: none;
  text-align: left; cursor: pointer;
}
.dropdown-item:hover { background: var(--bg-overlay); }
.tpl-name { font-size: 13px; font-weight: 500; color: var(--text-primary); }
.tpl-desc { font-size: 11px; color: var(--text-muted); margin-top: 2px; }

.flow-list { flex: 1; overflow-y: auto; padding: 20px 24px; }
.loading-state, .empty-state {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  height: 300px; color: var(--text-muted); gap: 12px;
}
.spin { animation: spin 1s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

.flow-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 16px; }
.flow-card {
  background: var(--bg-elevated); border: 1px solid var(--border);
  border-radius: 8px; padding: 16px; transition: all 0.15s;
}
.flow-card:hover { border-color: var(--accent); }
.flow-card.active { border-left: 3px solid var(--accent); }
.card-header { display: flex; justify-content: space-between; margin-bottom: 8px; }
.card-type {
  font-size: 10px; padding: 2px 6px; border-radius: 4px;
  background: var(--bg-app); color: var(--text-secondary);
}
.card-type.conversation { background: #dbeafe; color: #3b82f6; }
.card-type.task { background: #fef3c7; color: #f59e0b; }
.card-status {
  font-size: 10px; padding: 2px 6px; border-radius: 4px;
}
.card-status.active { background: #dcfce7; color: #22c55e; }
.card-status.draft { background: var(--bg-app); color: var(--text-muted); }
.card-status.disabled { background: #fee2e2; color: #ef4444; }
.card-title { font-size: 15px; font-weight: 600; color: var(--text-primary); margin: 0 0 4px 0; }
.card-desc { font-size: 12px; color: var(--text-secondary); margin: 0 0 12px 0; line-height: 1.4; }
.card-meta { display: flex; gap: 12px; font-size: 11px; color: var(--text-muted); margin-bottom: 12px; }
.card-actions { display: flex; gap: 4px; }
.action-btn {
  width: 28px; height: 28px; display: flex; align-items: center; justify-content: center;
  background: var(--bg-app); border: 1px solid var(--border); border-radius: 4px;
  color: var(--text-secondary); cursor: pointer;
}
.action-btn:hover { background: var(--bg-overlay); color: var(--text-primary); }
.action-btn.danger:hover { background: #fee2e2; color: #ef4444; }

/* Modal styles */
.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center; z-index: 1000;
}
.modal {
  background: var(--bg-elevated); border: 1px solid var(--border);
  border-radius: 8px; width: 480px; max-width: 90vw;
}
.modal-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 16px; border-bottom: 1px solid var(--border);
}
.modal-header h3 { margin: 0; font-size: 16px; color: var(--text-primary); }
.modal-body { padding: 16px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px; border-top: 1px solid var(--border); }
.btn-icon { width: 32px; height: 32px; display: flex; align-items: center; justify-content: center; background: transparent; border: none; cursor: pointer; color: var(--text-secondary); border-radius: 4px; }
.btn-icon:hover { background: var(--bg-overlay); }
.btn-ghost { padding: 8px 16px; background: transparent; border: 1px solid var(--border); border-radius: 6px; color: var(--text-secondary); font-size: 13px; cursor: pointer; }
.btn-ghost:hover { background: var(--bg-overlay); }

/* Fullscreen modal */
.modal-fullscreen {
  background: var(--bg-elevated); border: 1px solid var(--border);
  border-radius: 10px; width: 98vw; max-width: 1600px; height: 92vh;
  display: flex; flex-direction: column; overflow: hidden;
}
.modal-fheader {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 20px; border-bottom: 1px solid var(--border); gap: 16px;
}
.header-info { display: flex; align-items: center; gap: 16px; flex: 1; min-width: 0; }
.modal-title { font-size: 15px; font-weight: 600; color: var(--text-primary); margin: 0; white-space: nowrap; }
.header-form { display: flex; align-items: center; gap: 10px; flex: 1; min-width: 0; }
.name-input { width: 200px; padding: 6px 10px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 6px; color: var(--text-primary); font-size: 13px; }
.name-input:focus { outline: none; border-color: var(--accent); }
.desc-input { flex: 1; min-width: 150px; padding: 6px 10px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 6px; color: var(--text-primary); font-size: 13px; }
.desc-input:focus { outline: none; border-color: var(--accent); }
.type-select { padding: 6px 10px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 6px; color: var(--text-primary); font-size: 13px; }
.header-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.tab-switch { display: flex; background: var(--bg-app); border: 1px solid var(--border); border-radius: 6px; padding: 2px; gap: 2px; }
.switch-btn { padding: 5px 12px; background: transparent; border: none; border-radius: 4px; color: var(--text-secondary); font-size: 12px; font-weight: 500; cursor: pointer; }
.switch-btn:hover { color: var(--text-primary); }
.switch-btn.active { background: var(--accent-dim); color: var(--accent); }
.btn-ghost-sm { padding: 6px 12px; background: transparent; border: 1px solid var(--border); border-radius: 6px; color: var(--text-secondary); font-size: 12px; cursor: pointer; }
.btn-ghost-sm:hover { background: var(--bg-overlay); }
.btn-primary-sm { padding: 6px 14px; background: var(--accent); border: none; border-radius: 6px; color: #fff; font-size: 12px; font-weight: 500; cursor: pointer; }
.btn-primary-sm:hover { opacity: 0.9; }
.btn-icon-close { width: 28px; height: 28px; display: flex; align-items: center; justify-content: center; background: transparent; border: none; cursor: pointer; color: var(--text-secondary); border-radius: 6px; }
.btn-icon-close:hover { background: var(--bg-overlay); }
.modal-fbody { flex: 1; display: flex; overflow: hidden; }
.designer-wrap { flex: 1; overflow: hidden; }
.json-wrap { flex: 1; padding: 16px; display: flex; flex-direction: column; gap: 8px; }
.json-editor { flex: 1; width: 100%; padding: 12px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 6px; color: var(--text-primary); font-size: 12px; font-family: monospace; resize: none; }
.field-error { font-size: 12px; color: #ef4444; }
.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 12px; font-weight: 500; color: var(--text-secondary); margin-bottom: 4px; }
.form-group textarea { width: 100%; padding: 8px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 4px; color: var(--text-primary); font-size: 13px; resize: vertical; }
.execution-panel-wrap { width: 560px; max-width: 95vw; max-height: 90vh; overflow: hidden; border-radius: 10px; }
</style>