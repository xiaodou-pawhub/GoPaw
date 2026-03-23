<template>
  <div class="flow-designer">
    <!-- 工具栏 -->
    <div class="designer-toolbar">
      <button class="toolbar-btn toolbar-btn-primary" @click="saveFlow">
        <SaveIcon :size="14" /> 保存
      </button>
      <button class="toolbar-btn" @click="validateFlow">
        <CheckCircleIcon :size="14" /> 验证
      </button>
      <div class="toolbar-divider" />
      <button
        class="toolbar-btn toolbar-btn-danger"
        :disabled="!selectedNode"
        @click="deleteSelectedNode"
      >
        <Trash2Icon :size="14" />
      </button>
    </div>

    <div class="designer-content">
      <!-- 组件库 -->
      <div class="component-library">
        <div class="library-title">节点库</div>
        
        <!-- 基础节点 -->
        <div class="node-category">
          <div class="category-title">基础</div>
          <div
            v-for="type in basicNodes"
            :key="type.type"
            class="component-item"
            :style="{ borderLeftColor: type.color }"
            draggable="true"
            :title="type.usage"
            @dragstart="onDragStart($event, type)"
          >
            <span class="comp-icon" :style="{ background: type.color }">
              <component :is="type.icon" :size="14" />
            </span>
            <div class="comp-info">
              <span class="comp-name">{{ type.name }}</span>
              <span class="comp-desc">{{ type.description }}</span>
            </div>
          </div>
        </div>

        <!-- 控制节点 -->
        <div class="node-category">
          <div class="category-title">控制</div>
          <div
            v-for="type in controlNodes"
            :key="type.type"
            class="component-item"
            :style="{ borderLeftColor: type.color }"
            draggable="true"
            :title="type.usage"
            @dragstart="onDragStart($event, type)"
          >
            <span class="comp-icon" :style="{ background: type.color }">
              <component :is="type.icon" :size="14" />
            </span>
            <div class="comp-info">
              <span class="comp-name">{{ type.name }}</span>
              <span class="comp-desc">{{ type.description }}</span>
            </div>
          </div>
        </div>

        <!-- 高级节点 -->
        <div class="node-category">
          <div class="category-title">高级</div>
          <div
            v-for="type in advancedNodes"
            :key="type.type"
            class="component-item"
            :style="{ borderLeftColor: type.color }"
            draggable="true"
            :title="type.usage"
            @dragstart="onDragStart($event, type)"
          >
            <span class="comp-icon" :style="{ background: type.color }">
              <component :is="type.icon" :size="14" />
            </span>
            <div class="comp-info">
              <span class="comp-name">{{ type.name }}</span>
              <span class="comp-desc">{{ type.description }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 画布 -->
      <div class="canvas-container" @drop="onDrop" @dragover.prevent>
        <VueFlow
          v-model="elements"
          :node-types="vueFlowNodeTypes as any"
          :default-edge-options="defaultEdgeOptions"
          :connectable="true"
          :zoom-on-scroll="true"
          :pan-on-drag="true"
          @node-click="onNodeClick"
          @connect="onConnect"
        >
          <Background pattern-color="#aaa" :gap="20" />
          <Controls />
          <MiniMap />
        </VueFlow>
      </div>

      <!-- 属性面板 -->
      <div class="properties-panel">
        <div v-if="selectedNode">
          <div class="panel-title">属性</div>
          <NodeProperties
            :node-type="selectedNode.type || ''"
            :node-data="selectedNode.data"
            :node-id="selectedNode.id"
            :agents="agents"
            :flows="availableFlows"
            :edges="(elements.filter(el => !('position' in el)) as any)"
            :nodes="(elements.filter(el => 'position' in el) as any)"
            @update="onNodeUpdate"
            @delete="deleteSelectedNode"
          />
        </div>
        <div v-else>
          <!-- 无选中节点时显示变量面板 -->
          <div class="panel-title-row">
            <span class="panel-title">变量</span>
            <button class="panel-title-btn" @click="addVariable">+ 添加</button>
          </div>
          <div v-if="variables.length === 0" class="no-selection">
            <p>暂无变量，点击"添加"定义流程输入变量</p>
          </div>
          <div v-else class="var-list">
            <div v-for="(v, i) in variables" :key="i" class="var-item">
              <div class="var-header">
                <input v-model="v.name" class="var-name-input" placeholder="变量名" />
                <select v-model="v.type" class="var-type-select">
                  <option value="string">文本</option>
                  <option value="number">数字</option>
                  <option value="boolean">布尔</option>
                </select>
                <button class="var-del-btn" @click="removeVariable(i)">×</button>
              </div>
              <input v-model="v.default" class="var-input" placeholder="默认值（可选）" />
              <input v-model="v.description" class="var-input" placeholder="说明（可选）" />
              <label class="var-required">
                <input v-model="v.required" type="checkbox" /> 必填
              </label>
            </div>
          </div>
          <div class="var-hint" v-if="variables.length > 0">
            在 Prompt 中用 <code>&#123;&#123;变量名&#125;&#125;</code> 引用
          </div>
          <div class="panel-divider" />
          <div class="no-selection-hint">
            <MousePointerIcon :size="20" />
            <p>点击节点编辑属性</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, markRaw, computed } from 'vue'
import {
  SaveIcon, CheckCircleIcon, Trash2Icon, MousePointerIcon,
  PlayIcon, BotIcon, UserIcon, GitBranchIcon, GitMergeIcon,
  RepeatIcon, FolderIcon, WebhookIcon, SquareIcon
} from 'lucide-vue-next'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import type { Connection, Node, Edge, NodeMouseEvent } from '@vue-flow/core'
import {
  StartNode, AgentNode, HumanNode, ConditionNode,
  ParallelNode, LoopNode, SubFlowNode, WebhookNode, EndNode
} from './nodes'
import NodeProperties from './properties/NodeProperties.vue'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'

interface FlowNode {
  id: string
  type: string
  name: string
  agent_id?: string
  role?: string
  prompt?: string
  config?: Record<string, unknown>
  position: { x: number; y: number }
}

interface FlowEdge {
  id: string
  source: string
  target: string
  label?: string
  condition?: { type: string; expression?: string; intent?: string; llm_query?: string }
  transform?: { template: string }
}

interface FlowDefinition {
  nodes: FlowNode[]
  edges: FlowEdge[]
  variables?: Record<string, unknown>
  start_node_id?: string
}

interface Agent { id: string; name: string }
interface Flow { id: string; name: string }

interface NodeTypeInfo {
  type: string
  name: string
  description: string
  usage: string      // 使用场景
  color: string
  icon: any          // lucide 图标组件
  category: string   // 分类：basic/control/advanced
}

const props = defineProps<{
  definition?: FlowDefinition
  agents?: Agent[]
  flows?: Flow[]
}>()

const emit = defineEmits<{
  save: [definition: FlowDefinition]
  validate: [definition: FlowDefinition]
}>()

const { addNodes, addEdges, removeNodes } = useVueFlow()

// 节点类型配置
const nodeTypes: NodeTypeInfo[] = [
  { type: 'start',     name: '开始', description: '流程的起点，每个流程必须有一个', usage: '流程开始时执行', color: '#22c55e', icon: PlayIcon, category: 'basic' },
  { type: 'agent',     name: 'Agent', description: '调用数字员工执行任务', usage: '需要 AI 处理、工具调用、决策时', color: '#3b82f6', icon: BotIcon, category: 'basic' },
  { type: 'human',     name: '人工',  description: '等待人工输入或确认', usage: '需要人工审核、选择、补充信息时', color: '#f59e0b', icon: UserIcon, category: 'basic' },
  { type: 'condition', name: '条件',  description: '根据条件分支执行不同路径', usage: '意图识别、结果判断、状态检查', color: '#4facfe', icon: GitBranchIcon, category: 'control' },
  { type: 'parallel',  name: '并行',  description: '同时执行多个分支', usage: '多个独立任务需要并行处理', color: '#8b5cf6', icon: GitMergeIcon, category: 'control' },
  { type: 'loop',      name: '循环',  description: '重复执行直到条件满足', usage: '需要迭代处理、重试机制', color: '#ec4899', icon: RepeatIcon, category: 'control' },
  { type: 'subflow',   name: '子流程', description: '嵌套执行另一个流程', usage: '复用已有流程、模块化设计', color: '#06b6d4', icon: FolderIcon, category: 'advanced' },
  { type: 'webhook',   name: 'Webhook', description: '等待外部事件触发', usage: '需要外部系统回调、异步等待', color: '#64748b', icon: WebhookIcon, category: 'advanced' },
  { type: 'end',       name: '结束',  description: '流程的终点，输出最终结果', usage: '流程结束时执行', color: '#ef4444', icon: SquareIcon, category: 'basic' },
]

const vueFlowNodeTypes: Record<string, unknown> = {
  start: markRaw(StartNode),
  agent: markRaw(AgentNode),
  human: markRaw(HumanNode),
  condition: markRaw(ConditionNode),
  parallel: markRaw(ParallelNode),
  loop: markRaw(LoopNode),
  subflow: markRaw(SubFlowNode),
  webhook: markRaw(WebhookNode),
  end: markRaw(EndNode),
}

const defaultEdgeOptions = {
  animated: true,
  style: { stroke: '#666', strokeWidth: 2 },
  labelStyle: { fill: '#666', fontSize: 12 },
}

const elements = ref<(Node | Edge)[]>([])
const selectedNode = ref<Node | null>(null)
const availableFlows = ref<Flow[]>([])

// 变量面板
interface FlowVariable {
  name: string
  type: string
  required: boolean
  default: string
  description: string
}
const variables = ref<FlowVariable[]>([])

let nodeIdCounter = 1
let edgeIdCounter = 1

// 按分类过滤节点
const basicNodes = computed(() => nodeTypes.filter(n => n.category === 'basic'))
const controlNodes = computed(() => nodeTypes.filter(n => n.category === 'control'))
const advancedNodes = computed(() => nodeTypes.filter(n => n.category === 'advanced'))

watch(() => props.definition, (newDef) => {
  if (newDef) loadDefinition(newDef)
}, { immediate: true })

watch(() => props.flows, (newFlows) => {
  if (newFlows) availableFlows.value = newFlows
}, { immediate: true })

function loadDefinition(def: FlowDefinition) {
  const nodes: Node[] = def.nodes.map(n => ({
    id: n.id,
    type: n.type,
    position: n.position,
    data: { ...n }
  }))
  const edges: Edge[] = def.edges.map(e => ({
    id: e.id,
    source: e.source,
    target: e.target,
    label: e.label,
    sourceHandle: (e as any).source_handle,
    data: { ...e }
  }))
  elements.value = [...nodes, ...edges]

  // 加载变量
  if (def.variables) {
    variables.value = Object.entries(def.variables).map(([name, v]: [string, any]) => ({
      name,
      type: v.type || 'string',
      required: v.required || false,
      default: v.default != null ? String(v.default) : '',
      description: v.description || ''
    }))
  }

  nodeIdCounter = def.nodes.reduce((max, n) => {
    const num = parseInt(n.id.split('_')[1] || '0')
    return Math.max(max, num)
  }, 0) + 1

  edgeIdCounter = def.edges.reduce((max, e) => {
    const num = parseInt(e.id.split('_')[1] || '0')
    return Math.max(max, num)
  }, 0) + 1
}

function onDragStart(event: DragEvent, type: NodeTypeInfo) {
  event.dataTransfer!.setData('application/vueflow', JSON.stringify(type))
  event.dataTransfer!.effectAllowed = 'move'
}

function onDrop(event: DragEvent) {
  const typeInfo = JSON.parse(event.dataTransfer!.getData('application/vueflow'))
  const { left, top } = (event.target as HTMLElement).getBoundingClientRect()
  const position = {
    x: event.clientX - left,
    y: event.clientY - top
  }

  const newNode: Node = {
    id: `node_${nodeIdCounter++}`,
    type: typeInfo.type,
    position,
    data: {
      id: `node_${nodeIdCounter - 1}`,
      type: typeInfo.type,
      name: typeInfo.name,
      config: {}
    }
  }

  addNodes([newNode])
}

function onNodeClick(event: NodeMouseEvent) {
  selectedNode.value = event.node
}

function onConnect(connection: Connection) {
  // 条件节点：自动为连线设置 true/false 标签
  const allElements = elements.value as any[]
  const sourceNode = allElements.find((el: any) => 'position' in el && el.id === connection.source)
  const handleLabel = connection.sourceHandle === 'true' ? 'true'
    : connection.sourceHandle === 'false' ? 'false'
    : undefined
  const autoLabel = sourceNode?.type === 'condition' && handleLabel ? handleLabel : undefined

  const newEdge: Edge = {
    id: `edge_${edgeIdCounter++}`,
    source: connection.source!,
    target: connection.target!,
    sourceHandle: connection.sourceHandle,
    targetHandle: connection.targetHandle,
    label: autoLabel,
    data: autoLabel ? { condition_branch: autoLabel } : {}
  }
  addEdges([newEdge])
}

function onNodeUpdate(data: Record<string, any>) {
  if (!selectedNode.value) return
  selectedNode.value.data = data
}

function deleteSelectedNode() {
  if (!selectedNode.value) return
  const nodeToRemove = selectedNode.value
  selectedNode.value = null
  removeNodes([nodeToRemove as any])
}

function getDefinition(): FlowDefinition {
  const nodes: FlowNode[] = []
  const edges: FlowEdge[] = []
  let startNodeId = ''

  for (const el of elements.value) {
    if ('position' in el) {
      // Node
      const data = el.data as any
      nodes.push({
        id: el.id as string,
        type: el.type as string,
        name: data?.name || el.type,
        agent_id: data?.agent_id,
        role: data?.role,
        prompt: data?.prompt,
        config: { ...(data?.config || {}), output_var: data?.output_var },
        position: el.position
      })
      if (el.type === 'start') startNodeId = el.id as string
    } else {
      // Edge
      const edgeEl = el as any
      edges.push({
        id: edgeEl.id as string,
        source: edgeEl.source,
        target: edgeEl.target,
        label: edgeEl.label as string,
        condition: edgeEl.data?.condition,
        transform: edgeEl.data?.transform,
        source_handle: edgeEl.sourceHandle,
      } as any)
    }
  }

  // 构建变量 map
  const variablesMap: Record<string, unknown> = {}
  for (const v of variables.value) {
    if (v.name) {
      variablesMap[v.name] = {
        type: v.type,
        required: v.required,
        default: v.default || undefined,
        description: v.description || undefined
      }
    }
  }

  return {
    nodes,
    edges,
    variables: Object.keys(variablesMap).length > 0 ? variablesMap : undefined,
    start_node_id: startNodeId || (nodes[0]?.id)
  }
}

function addVariable() {
  variables.value.push({ name: '', type: 'string', required: false, default: '', description: '' })
}

function removeVariable(index: number) {
  variables.value.splice(index, 1)
}

function saveFlow() {
  emit('save', getDefinition())
}

function validateFlow() {
  emit('validate', getDefinition())
}
</script>

<style scoped>
.flow-designer {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-app);
}

.designer-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-elevated);
}

.toolbar-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 12px;
  cursor: pointer;
}
.toolbar-btn:hover { background: var(--bg-overlay); }
.toolbar-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.toolbar-btn-primary { background: var(--accent); border-color: var(--accent); color: #fff; }
.toolbar-btn-primary:hover { opacity: 0.9; }
.toolbar-btn-danger { color: #ef4444; border-color: #ef4444; }
.toolbar-btn-danger:hover { background: #fee2e2; }
.toolbar-divider { width: 1px; height: 20px; background: var(--border); }

.designer-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.component-library {
  width: 200px;
  padding: 12px;
  border-right: 1px solid var(--border);
  overflow-y: auto;
  flex-shrink: 0;
}

.library-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.node-category {
  margin-bottom: 12px;
}

.category-title {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-muted);
  margin-bottom: 6px;
  padding-left: 4px;
}

.component-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px;
  margin-bottom: 4px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-left-width: 3px;
  border-radius: 4px;
  cursor: grab;
  transition: all 0.15s;
}
.component-item:hover { background: var(--bg-overlay); }
.component-item:active { cursor: grabbing; }

.comp-icon {
  width: 24px;
  height: 24px;
  border-radius: 4px;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}

.comp-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.comp-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-primary);
}

.comp-desc {
  font-size: 10px;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.canvas-container {
  flex: 1;
  overflow: hidden;
}

.properties-panel {
  width: 280px;
  border-left: 1px solid var(--border);
  overflow-y: auto;
  flex-shrink: 0;
  background: var(--bg-elevated);
}

.panel-title {
  padding: 12px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.no-selection {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: var(--text-muted);
  text-align: center;
}
.no-selection p { margin-top: 8px; font-size: 12px; }
.panel-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  border-bottom: 1px solid var(--border);
}
.panel-title-btn {
  font-size: 11px;
  padding: 3px 8px;
  background: var(--accent-dim);
  color: var(--accent);
  border: 1px solid var(--accent);
  border-radius: 4px;
  cursor: pointer;
}
.var-list { padding: 8px 12px; }
.var-item {
  margin-bottom: 12px;
  padding: 8px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
}
.var-header { display: flex; gap: 4px; margin-bottom: 4px; }
.var-name-input {
  flex: 1;
  padding: 4px 6px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 500;
}
.var-type-select {
  padding: 4px 6px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: 11px;
  width: 52px;
}
.var-del-btn {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 14px;
}
.var-del-btn:hover { background: #fee2e2; color: #ef4444; border-color: #ef4444; }
.var-input {
  width: 100%;
  padding: 4px 6px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: 11px;
  margin-bottom: 4px;
}
.var-required { display: flex; align-items: center; gap: 4px; font-size: 11px; color: var(--text-muted); }
.var-hint { padding: 6px 12px 12px; font-size: 11px; color: var(--text-muted); }
.var-hint code { background: var(--bg-app); padding: 1px 4px; border-radius: 3px; font-size: 11px; }
.panel-divider { height: 1px; background: var(--border); margin: 8px 0; }
.no-selection-hint {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px;
  color: var(--text-muted);
  text-align: center;
}
.no-selection-hint p { margin-top: 6px; font-size: 11px; }
</style>