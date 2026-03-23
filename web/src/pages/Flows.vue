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
        <button class="btn-secondary" @click="triggerImport">
          <UploadIcon :size="16" /> 导入
        </button>
        <input
          ref="importInput"
          type="file"
          accept=".json"
          style="display: none"
          @change="handleImport"
        />
        <div class="dropdown">
          <button class="btn-secondary" @click="showTemplates = !showTemplates">
            <FileTextIcon :size="16" /> 从模板创建
            <ChevronDownIcon :size="14" />
          </button>
          <div v-if="showTemplates" class="template-dropdown">
            <div class="template-header">
              <span>选择模板</span>
              <select v-model="selectedCategory" class="category-select" @change="loadTemplates">
                <option value="">全部分类</option>
                <option v-for="cat in templateCategories" :key="cat.id" :value="cat.id">
                  {{ cat.name }}
                </option>
              </select>
            </div>
            <div v-if="templatesLoading" class="template-loading">
              <LoaderIcon :size="16" class="spin" /> 加载中...
            </div>
            <div v-else-if="templates.length === 0" class="template-empty">
              暂无模板
            </div>
            <div v-else class="template-list">
              <button
                v-for="tpl in templates"
                :key="tpl.id"
                class="template-item"
                @click="useTemplate(tpl); showTemplates = false"
              >
                <div class="tpl-icon">
                  <component :is="getIconComponent(tpl.icon)" :size="20" />
                </div>
                <div class="tpl-info">
                  <span class="tpl-name">{{ tpl.name }}</span>
                  <span class="tpl-desc">{{ tpl.description }}</span>
                  <div class="tpl-meta">
                    <span v-if="tpl.use_count" class="tpl-uses">{{ tpl.use_count }} 次使用</span>
                    <span v-for="tag in tpl.tags?.slice(0, 2)" :key="tag" class="tpl-tag">{{ tag }}</span>
                  </div>
                </div>
              </button>
            </div>
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
            <button class="action-btn" @click="openVersionDialog(flow)" title="版本管理">
              <HistoryIcon :size="14" />
            </button>
            <button class="action-btn" @click="exportFlow(flow)" title="导出">
              <DownloadIcon :size="14" />
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
              <button class="switch-btn" :class="{ active: dialog.activeTab === 'trigger' }" @click="dialog.activeTab = 'trigger'">触发器</button>
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
          <div v-if="dialog.activeTab === 'trigger'" class="trigger-wrap">
            <div class="trigger-section">
              <h3 class="section-title">触发方式</h3>
              <p class="section-desc">配置流程的触发方式，支持手动触发、定时触发、Webhook 触发等</p>

              <div class="form-group">
                <label>触发类型</label>
                <select v-model="triggerConfig.type" class="form-select">
                  <option value="manual">手动触发</option>
                  <option value="cron">定时触发 (Cron)</option>
                  <option value="webhook">Webhook 触发</option>
                  <option value="event">事件触发 (开发中)</option>
                </select>
              </div>

              <!-- Cron 配置 -->
              <template v-if="triggerConfig.type === 'cron'">
                <div class="form-group">
                  <label>Cron 表达式</label>
                  <input v-model="triggerConfig.config.schedule" type="text" class="form-input" placeholder="0 0 9 * * *" />
                  <span class="hint">格式: 秒 分 时 日 月 周，如 "0 0 9 * * *" 表示每天 9:00 执行</span>
                </div>
                <div class="form-group">
                  <label>任务描述</label>
                  <input v-model="triggerConfig.config.task" type="text" class="form-input" placeholder="执行每日报告流程" />
                </div>
                <div class="cron-preview" v-if="triggerConfig.config.schedule">
                  <span class="preview-label">下次执行时间:</span>
                  <span class="preview-value">{{ getNextCronTime(triggerConfig.config.schedule) }}</span>
                </div>
              </template>

              <!-- Webhook 配置 -->
              <template v-if="triggerConfig.type === 'webhook'">
                <div class="webhook-info">
                  <div class="info-row">
                    <span class="info-label">Webhook URL:</span>
                    <code class="info-value">{{ getWebhookUrl(dialog.data.id) }}</code>
                    <button class="btn-copy" @click="copyWebhookUrl">复制</button>
                  </div>
                  <div class="info-row">
                    <span class="info-label">请求方法:</span>
                    <code class="info-value">POST</code>
                  </div>
                  <div class="info-row">
                    <span class="info-label">请求体:</span>
                    <code class="info-value">{ "input": "your input data" }</code>
                  </div>
                </div>
                <p class="hint">激活流程后，外部系统可以通过 POST 请求触发此流程</p>
              </template>

              <!-- 事件触发配置 -->
              <template v-if="triggerConfig.type === 'event'">
                <div class="coming-soon">
                  <span>事件触发功能开发中，敬请期待...</span>
                </div>
              </template>
            </div>
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

  <!-- 版本管理对话框 -->
  <div v-if="versionDialog.show" class="modal-overlay" @click.self="versionDialog.show = false">
    <div class="modal-dialog">
      <div class="modal-header">
        <h3>版本管理 - {{ versionDialog.flowName }}</h3>
        <button class="btn-icon" @click="versionDialog.show = false"><XIcon :size="16" /></button>
      </div>

      <div class="modal-body">
        <!-- 创建新版本 -->
        <div class="version-create">
          <h4>保存新版本</h4>
          <div class="create-form">
            <input
              v-model="versionDialog.newVersionName"
              type="text"
              placeholder="版本名称（可选）"
              class="input-name"
            />
            <input
              v-model="versionDialog.newVersionDesc"
              type="text"
              placeholder="版本描述（可选）"
              class="input-desc"
            />
            <button
              class="btn-primary"
              :disabled="versionDialog.creating"
              @click="createVersion"
            >
              {{ versionDialog.creating ? '保存中...' : '保存版本' }}
            </button>
          </div>
        </div>

        <!-- 版本列表 -->
        <div class="version-list">
          <h4>历史版本</h4>
          <div v-if="versionDialog.loading" class="loading-state">
            <LoaderIcon :size="24" class="spin" />
            <span>加载中...</span>
          </div>
          <div v-else-if="versionDialog.versions.length === 0" class="empty-versions">
            暂无历史版本
          </div>
          <div v-else class="versions">
            <div
              v-for="v in versionDialog.versions"
              :key="v.id"
              class="version-item"
            >
              <div class="version-info">
                <div class="version-header">
                  <span class="version-badge">v{{ v.version }}</span>
                  <span class="version-name">{{ v.name || '未命名版本' }}</span>
                </div>
                <div class="version-meta">
                  <span>{{ formatDate(v.created_at) }}</span>
                  <span v-if="v.created_by"> · {{ v.created_by }}</span>
                </div>
                <div v-if="v.description" class="version-desc">{{ v.description }}</div>
              </div>
              <div class="version-actions">
                <button class="btn-small" @click="rollbackVersion(v.version)" title="回滚到此版本">
                  回滚
                </button>
                <button class="btn-small danger" @click="deleteVersion(v.version)" title="删除此版本">
                  删除
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  PlusIcon, EditIcon, Trash2Icon, PlayIcon, PauseIcon, RocketIcon,
  LoaderIcon, GitBranchIcon, XIcon, FileTextIcon, ChevronDownIcon,
  UploadIcon, DownloadIcon, HistoryIcon
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

// 版本管理
interface FlowVersion {
  id: string
  flow_id: string
  version: number
  name: string
  description: string
  definition: FlowDefinition
  created_at: string
  created_by: string
}
const versionDialog = ref({
  show: false,
  flowId: '',
  flowName: '',
  versions: [] as FlowVersion[],
  loading: false,
  creating: false,
  newVersionName: '',
  newVersionDesc: ''
})

// 预置模板
interface FlowTemplate {
  id: string
  name: string
  description: string
  category: string
  tags: string[]
  icon: string
  type: 'conversation' | 'task'
  definition: FlowDefinition
  use_count: number
}

// 模板分类
interface TemplateCategory {
  id: string
  name: string
  description: string
  icon: string
}

const templates = ref<FlowTemplate[]>([])
const templateCategories = ref<TemplateCategory[]>([])
const selectedCategory = ref('')
const templatesLoading = ref(false)

const filteredFlows = computed(() => {
  if (!flows.value) return []
  if (!typeFilter.value) return flows.value
  return flows.value.filter(f => f.type === typeFilter.value)
})

// 加载模板
async function loadTemplates() {
  templatesLoading.value = true
  try {
    // 加载分类
    const catRes = await fetch('/api/flows/templates/categories')
    if (catRes.ok) {
      templateCategories.value = await catRes.json()
    }

    // 加载模板
    const url = selectedCategory.value
      ? `/api/flows/templates?category=${selectedCategory.value}`
      : '/api/flows/templates'
    const tplRes = await fetch(url)
    if (tplRes.ok) {
      templates.value = await tplRes.json()
    }
  } catch (e) {
    console.error('Failed to load templates:', e)
  } finally {
    templatesLoading.value = false
  }
}

// 使用模板创建流程
async function useTemplate(template: FlowTemplate) {
  try {
    const res = await fetch(`/api/flows/templates/${template.id}/use`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: template.name + ' (副本)',
        description: template.description
      })
    })

    if (res.ok) {
      const flow = await res.json()
      await loadFlows()
      openEditDialog(flow)
    } else {
      const err = await res.json()
      alert(err.error || '创建失败')
    }
  } catch (e) {
    console.error('Failed to use template:', e)
    alert('创建失败')
  }
}

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
    trigger?: { type: string; config: Record<string, any> }
  }
  activeTab: 'designer' | 'trigger' | 'json'
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

// 触发器配置
const triggerConfig = ref<{
  type: string
  config: { schedule?: string; task?: string }
}>({
  type: 'manual',
  config: {}
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
  await Promise.all([loadFlows(), loadAgents(), loadTemplates()])
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

// 获取图标组件
function getIconComponent(iconName: string) {
  const iconMap: Record<string, any> = {
    'HeadphonesIcon': FileTextIcon,
    'CheckCircleIcon': FileTextIcon,
    'BarChartIcon': FileTextIcon,
    'ZapIcon': FileTextIcon,
    'RefreshCwIcon': FileTextIcon,
    'BellIcon': FileTextIcon,
    'LayoutGridIcon': FileTextIcon
  }
  return iconMap[iconName] || FileTextIcon
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
  // 重置触发器配置
  triggerConfig.value = { type: 'manual', config: {} }
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
      definition: flow.definition || { nodes: [], edges: [] },
      trigger: (flow as any).trigger
    },
    activeTab: 'designer',
    definitionText: JSON.stringify(flow.definition, null, 2),
    jsonError: ''
  }
  // 加载触发器配置
  if ((flow as any).trigger) {
    triggerConfig.value = {
      type: (flow as any).trigger.type || 'manual',
      config: (flow as any).trigger.config || {}
    }
  } else {
    triggerConfig.value = { type: 'manual', config: {} }
  }
}

async function saveFlow() {
  const url = dialog.value.isEdit ? `/api/flows/${dialog.value.data.id}` : '/api/flows'
  const method = dialog.value.isEdit ? 'PUT' : 'POST'

  // 构建请求数据，包含触发器配置
  const saveData = {
    ...dialog.value.data,
    trigger: triggerConfig.value.type !== 'manual' ? triggerConfig.value : null
  }

  try {
    const res = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(saveData)
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

// 导入导出功能
const importInput = ref<HTMLInputElement | null>(null)

function triggerImport() {
  importInput.value?.click()
}

async function handleImport(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  try {
    const text = await file.text()
    const data = JSON.parse(text)

    // 验证导入的数据结构
    if (!data.name || !data.definition || !data.definition.nodes) {
      alert('无效的流程文件格式')
      return
    }

    // 创建新流程
    const newFlow = {
      id: generateId(),
      name: data.name + ' (导入)',
      description: data.description || '',
      type: data.type || 'conversation',
      definition: data.definition,
      trigger: data.trigger,
      status: 'draft'
    }

    const res = await fetch('/api/flows', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(newFlow)
    })

    if (res.ok) {
      await loadFlows()
      alert('流程导入成功')
    } else {
      const err = await res.json()
      alert(err.error || '导入失败')
    }
  } catch (e) {
    console.error('Import failed:', e)
    alert('导入失败：文件格式错误')
  }

  // 清空 input 以便再次选择同一文件
  input.value = ''
}

function exportFlow(flow: Flow) {
  // 构建导出数据
  const exportData = {
    name: flow.name,
    description: flow.description,
    type: flow.type,
    definition: flow.definition,
    trigger: (flow as any).trigger,
    exported_at: new Date().toISOString(),
    version: '1.0'
  }

  // 创建下载
  const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${flow.name.replace(/[^a-zA-Z0-9\u4e00-\u9fa5]/g, '_')}_flow.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

// ========== 版本管理 ==========

async function openVersionDialog(flow: Flow) {
  versionDialog.value = {
    show: true,
    flowId: flow.id,
    flowName: flow.name,
    versions: [],
    loading: true,
    creating: false,
    newVersionName: '',
    newVersionDesc: ''
  }
  await loadVersions()
}

async function loadVersions() {
  try {
    const res = await fetch(`/api/flows/${versionDialog.value.flowId}/versions`)
    if (res.ok) {
      versionDialog.value.versions = await res.json()
    }
  } catch (e) {
    console.error('Failed to load versions:', e)
  } finally {
    versionDialog.value.loading = false
  }
}

async function createVersion() {
  if (versionDialog.value.creating) return
  versionDialog.value.creating = true

  try {
    const res = await fetch(`/api/flows/${versionDialog.value.flowId}/versions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: versionDialog.value.newVersionName || undefined,
        description: versionDialog.value.newVersionDesc || undefined
      })
    })

    if (res.ok) {
      versionDialog.value.newVersionName = ''
      versionDialog.value.newVersionDesc = ''
      await loadVersions()
    } else {
      const err = await res.json()
      alert(err.error || '创建版本失败')
    }
  } catch (e) {
    console.error('Failed to create version:', e)
    alert('创建版本失败')
  } finally {
    versionDialog.value.creating = false
  }
}

async function rollbackVersion(version: number) {
  if (!confirm(`确定要回滚到版本 ${version} 吗？当前流程定义将被替换。`)) return

  try {
    const res = await fetch(`/api/flows/${versionDialog.value.flowId}/versions/${version}/rollback`, {
      method: 'POST'
    })

    if (res.ok) {
      await loadFlows()
      await loadVersions()
      alert('回滚成功')
    } else {
      const err = await res.json()
      alert(err.error || '回滚失败')
    }
  } catch (e) {
    console.error('Failed to rollback:', e)
    alert('回滚失败')
  }
}

async function deleteVersion(version: number) {
  if (!confirm(`确定要删除版本 ${version} 吗？此操作不可恢复。`)) return

  try {
    const res = await fetch(`/api/flows/${versionDialog.value.flowId}/versions/${version}`, {
      method: 'DELETE'
    })

    if (res.ok) {
      await loadVersions()
    } else {
      const err = await res.json()
      alert(err.error || '删除失败')
    }
  } catch (e) {
    console.error('Failed to delete version:', e)
    alert('删除失败')
  }
}

// 触发器相关函数
function getWebhookUrl(flowId: string): string {
  const baseUrl = window.location.origin
  return `${baseUrl}/api/webhooks/flow/${flowId}`
}

function copyWebhookUrl() {
  const url = getWebhookUrl(dialog.value.data.id)
  navigator.clipboard.writeText(url).then(() => {
    alert('Webhook URL 已复制到剪贴板')
  }).catch(() => {
    alert('复制失败，请手动复制')
  })
}

function getNextCronTime(schedule: string): string {
  // 简单的 cron 解析预览（仅支持基本格式）
  // 实际应该使用 cron 库来计算
  try {
    const parts = schedule.split(' ')
    if (parts.length !== 6) return '格式错误'

    const [_sec, min, hour, _day, _month, _weekday] = parts

    // 简单示例：每天固定时间
    if (min !== '*' && hour !== '*') {
      const h = parseInt(hour)
      const m = parseInt(min)
      if (!isNaN(h) && !isNaN(m)) {
        const now = new Date()
        const next = new Date()
        next.setHours(h, m, 0, 0)
        if (next <= now) {
          next.setDate(next.getDate() + 1)
        }
        return `${next.getMonth() + 1}/${next.getDate()} ${h}:${String(m).padStart(2, '0')}`
      }
    }

    return '请使用标准 cron 表达式'
  } catch {
    return '解析失败'
  }
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

/* Trigger config styles */
.trigger-wrap { flex: 1; padding: 24px; overflow-y: auto; }
.trigger-section { max-width: 600px; }
.section-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin: 0 0 8px 0; }
.section-desc { font-size: 13px; color: var(--text-secondary); margin: 0 0 20px 0; }
.form-select { width: 100%; padding: 10px 12px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 6px; color: var(--text-primary); font-size: 14px; }
.form-select:focus { outline: none; border-color: var(--accent); }
.form-input { width: 100%; padding: 10px 12px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 6px; color: var(--text-primary); font-size: 14px; }
.form-input:focus { outline: none; border-color: var(--accent); }
.hint { font-size: 12px; color: var(--text-muted); margin-top: 4px; display: block; }
.cron-preview { margin-top: 16px; padding: 12px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 6px; }
.preview-label { font-size: 12px; color: var(--text-secondary); margin-right: 8px; }
.preview-value { font-size: 14px; font-weight: 500; color: var(--accent); }
.webhook-info { margin-bottom: 12px; padding: 16px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 8px; }
.info-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.info-row:last-child { margin-bottom: 0; }
.info-label { font-size: 12px; color: var(--text-secondary); min-width: 100px; }
.info-value { font-size: 13px; color: var(--text-primary); font-family: monospace; background: var(--bg-overlay); padding: 4px 8px; border-radius: 4px; flex: 1; }
.btn-copy { padding: 4px 10px; background: var(--accent); border: none; border-radius: 4px; color: #fff; font-size: 12px; cursor: pointer; }
.btn-copy:hover { opacity: 0.9; }
.coming-soon { padding: 24px; text-align: center; color: var(--text-muted); font-size: 14px; background: var(--bg-app); border: 1px dashed var(--border); border-radius: 8px; }

.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 12px; font-weight: 500; color: var(--text-secondary); margin-bottom: 4px; }
.form-group textarea { width: 100%; padding: 8px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 4px; color: var(--text-primary); font-size: 13px; resize: vertical; }

/* Template dropdown styles */
.template-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  min-width: 360px;
  max-width: 480px;
  max-height: 400px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
  z-index: 100;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.template-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
}
.category-select {
  padding: 4px 8px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 12px;
}
.template-loading, .template-empty {
  padding: 24px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}
.template-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}
.template-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  width: 100%;
  padding: 12px;
  background: transparent;
  border: none;
  border-radius: 6px;
  text-align: left;
  cursor: pointer;
}
.template-item:hover {
  background: var(--bg-overlay);
}
.tpl-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-dim);
  border-radius: 8px;
  color: var(--accent);
  flex-shrink: 0;
}
.tpl-info {
  flex: 1;
  min-width: 0;
}
.tpl-name {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 2px;
}
.tpl-desc {
  display: block;
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 6px;
}
.tpl-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.tpl-uses {
  font-size: 11px;
  color: var(--text-secondary);
}
.tpl-tag {
  font-size: 10px;
  padding: 1px 6px;
  background: var(--bg-app);
  border-radius: 3px;
  color: var(--text-secondary);
}

/* Version management styles */
.modal-dialog {
  width: 560px;
  max-height: 80vh;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}
.modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}
.modal-body {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
}
.version-create {
  padding: 16px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 8px;
  margin-bottom: 20px;
}
.version-create h4, .version-list h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}
.create-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.create-form .input-name,
.create-form .input-desc {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
}
.create-form .input-name:focus,
.create-form .input-desc:focus {
  outline: none;
  border-color: var(--accent);
}
.create-form .btn-primary {
  align-self: flex-end;
  padding: 8px 16px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
.create-form .btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.version-list {
  margin-top: 8px;
}
.empty-versions {
  text-align: center;
  padding: 32px;
  color: var(--text-muted);
  font-size: 14px;
}
.versions {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.version-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 8px;
}
.version-info {
  flex: 1;
  min-width: 0;
}
.version-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}
.version-badge {
  display: inline-block;
  padding: 2px 8px;
  background: var(--accent-dim);
  color: var(--accent);
  font-size: 11px;
  font-weight: 600;
  border-radius: 4px;
}
.version-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}
.version-meta {
  font-size: 12px;
  color: var(--text-muted);
}
.version-desc {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 4px;
}
.version-actions {
  display: flex;
  gap: 6px;
  margin-left: 12px;
}
.btn-small {
  padding: 4px 10px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
}
.btn-small:hover {
  background: var(--bg-elevated);
  color: var(--text-primary);
}
.btn-small.danger {
  border-color: #fecaca;
  color: #dc2626;
}
.btn-small.danger:hover {
  background: #fee2e2;
}

.execution-panel-wrap { width: 560px; max-width: 95vw; max-height: 90vh; overflow: hidden; border-radius: 10px; }
</style>