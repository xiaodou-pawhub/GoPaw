<template>
  <div class="orchestration-designer">
    <!-- 工具栏 -->
    <div class="designer-toolbar">
      <button class="toolbar-btn toolbar-btn-primary" @click="saveOrchestration">
        <SaveIcon :size="14" /> 保存
      </button>
      <button class="toolbar-btn" @click="validateOrchestration">
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
        <div class="library-title">组件库</div>
        <div
          v-for="type in nodeTypes"
          :key="type.id"
          class="component-item"
          :style="{ borderLeftColor: type.color }"
          draggable="true"
          @dragstart="onDragStart($event, type)"
        >
          <span class="comp-icon" :style="{ background: type.color }">{{ type.abbr }}</span>
          <span>{{ type.name }}</span>
        </div>
      </div>

      <!-- 画布 -->
      <div class="canvas-container" @drop="onDrop" @dragover.prevent>
        <VueFlow
          v-model="elements"
          :node-types="(vueFlowNodeTypes as any)"
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
        <div class="panel-title">属性</div>
        <NodeProperties
          v-if="selectedNode"
          :node-type="selectedNode.type || ''"
          :node-data="selectedNode.data"
          :agents="agents"
          @update="onNodeUpdate"
          @delete="deleteSelectedNode"
        />
        <div v-else class="no-selection">
          <MousePointerIcon :size="24" />
          <p>选择一个节点以编辑属性</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { SaveIcon, CheckCircleIcon, Trash2Icon, MousePointerIcon } from 'lucide-vue-next'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import type { Connection, Node, Edge } from '@vue-flow/core'
import { AgentNode, HumanNode, ConditionNode, WorkflowNode, EndNode } from './nodes'
import NodeProperties from './properties/NodeProperties.vue'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'

interface OrchestrationNode {
  id: string
  type: 'agent' | 'human' | 'condition' | 'workflow' | 'end'
  agent_id?: string
  name: string
  role?: string
  prompt?: string
  config?: Record<string, unknown>
  position: { x: number; y: number }
}

interface OrchestrationEdge {
  id: string
  source: string
  target: string
  message_type: string
  condition?: unknown
  transform?: unknown
  label?: string
}

interface OrchestrationDefinition {
  nodes: OrchestrationNode[]
  edges: OrchestrationEdge[]
  start_node_id: string
}

interface Agent {
  id: string
  name: string
}

interface NodeTypeConfig {
  id: string
  name: string
  color: string
  abbr: string
}

const props = defineProps<{
  definition?: OrchestrationDefinition
  agents?: Agent[]
}>()

const emit = defineEmits<{
  save: [definition: OrchestrationDefinition]
  validate: [definition: OrchestrationDefinition]
}>()

const { addNodes, addEdges, removeNodes } = useVueFlow()

const nodeTypes: NodeTypeConfig[] = [
  { id: 'agent',     name: 'Agent',  color: '#3b82f6', abbr: 'A' },
  { id: 'human',     name: '人工',   color: '#f59e0b', abbr: 'H' },
  { id: 'condition', name: '条件',   color: '#4facfe', abbr: 'C' },
  { id: 'workflow',  name: '工作流', color: '#8b5cf6', abbr: 'W' },
  { id: 'end',       name: '结束',   color: '#16a34a', abbr: 'E' },
]

const vueFlowNodeTypes: Record<string, unknown> = {
  agent: AgentNode, human: HumanNode, condition: ConditionNode,
  workflow: WorkflowNode, end: EndNode,
}

const defaultEdgeOptions = {
  animated: true,
  style: { stroke: '#666', strokeWidth: 2 },
  labelStyle: { fill: '#666', fontSize: 12 },
}

const elements = ref<(Node | Edge)[]>([])
const selectedNode = ref<Node | null>(null)
let nodeIdCounter = 1
let edgeIdCounter = 1

watch(() => props.definition, (newDef) => {
  if (newDef) loadDefinition(newDef)
}, { immediate: true })

function loadDefinition(def: OrchestrationDefinition) {
  const nodes: Node[] = def.nodes.map(n => ({ id: n.id, type: n.type, position: n.position, data: { ...n } }))
  const edges: Edge[] = def.edges.map(e => ({ id: e.id, source: e.source, target: e.target, label: e.label || e.message_type, data: { ...e } }))
  elements.value = [...nodes, ...edges]

  nodeIdCounter = def.nodes.reduce((max, n) => {
    const num = parseInt(n.id.split('_')[1] || '0')
    return Math.max(max, num)
  }, 0) + 1

  edgeIdCounter = def.edges.reduce((max, e) => {
    const num = parseInt(e.id.split('_')[1] || '0')
    return Math.max(max, num)
  }, 0) + 1
}

function onDragStart(event: DragEvent, type: NodeTypeConfig) {
  event.dataTransfer?.setData('application/vueflow', type.id)
}

function onDrop(event: DragEvent) {
  const typeId = event.dataTransfer?.getData('application/vueflow')
  if (!typeId) return
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  addNodes([{
    id: `${typeId}_${nodeIdCounter++}`,
    type: typeId,
    position: { x: event.clientX - rect.left, y: event.clientY - rect.top },
    data: { name: nodeTypes.find(t => t.id === typeId)?.name || typeId },
  }])
}

function onNodeClick({ node }: { node: Node }) {
  selectedNode.value = node
}

function onConnect(connection: Connection) {
  addEdges([{
    id: `edge_${edgeIdCounter++}`,
    source: connection.source,
    target: connection.target,
    label: 'task',
    data: { message_type: 'task' },
  }])
}

function onNodeUpdate(data: Record<string, unknown>) {
  if (!selectedNode.value) return
  const nodeIndex = elements.value.findIndex(e => e.id === selectedNode.value!.id)
  if (nodeIndex >= 0) {
    const el = elements.value[nodeIndex]
    if (!(el as Edge).source) {
      el.data = { ...el.data, ...data }
    }
  }
}

function deleteSelectedNode() {
  if (!selectedNode.value) return
  removeNodes([selectedNode.value.id])
  selectedNode.value = null
}

function saveOrchestration() {
  const nodes: OrchestrationNode[] = []
  const edges: OrchestrationEdge[] = []
  let startNodeId = ''

  elements.value.forEach(el => {
    if ((el as Edge).source) {
      const edge = el as Edge
      edges.push({
        id: edge.id, source: edge.source, target: edge.target,
        message_type: (edge.data?.message_type as string) || 'task',
        condition: edge.data?.condition, transform: edge.data?.transform,
        label: edge.label as string,
      })
    } else {
      const node = el as Node
      const nd: OrchestrationNode = {
        id: node.id, type: node.type as OrchestrationNode['type'],
        name: node.data.name, position: node.position,
      }
      if (node.type === 'agent') { nd.agent_id = node.data.agent_id; nd.role = node.data.role; nd.prompt = node.data.prompt }
      else if (node.type === 'human') { nd.prompt = node.data.prompt; nd.config = { options: node.data.options, timeout: node.data.timeout } }
      else if (node.type === 'condition') { nd.config = { condition_type: node.data.condition_type } }
      else if (node.type === 'workflow') { nd.config = { workflow_id: node.data.workflow_id } }
      else if (node.type === 'end') { nd.config = { output_template: node.data.output_template } }
      nodes.push(nd)
      if (!startNodeId) startNodeId = node.id
    }
  })

  emit('save', { nodes, edges, start_node_id: startNodeId })
}

function validateOrchestration() {
  saveOrchestration()
}
</script>

<style scoped>
.orchestration-designer {
  display: flex;
  flex-direction: column;
  height: 100%;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  overflow: hidden;
}

.designer-toolbar {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  background: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
  gap: 6px;
}

.toolbar-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 6px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  background: transparent;
  color: #475569;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.1s;
}

.toolbar-btn:hover { background: #f1f5f9; }
.toolbar-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.toolbar-btn-primary { background: #3b82f6; color: #fff; border-color: #3b82f6; }
.toolbar-btn-primary:hover { background: #2563eb; }
.toolbar-btn-danger:hover { background: rgba(239,68,68,0.1); color: #ef4444; border-color: rgba(239,68,68,0.3); }

.toolbar-divider { width: 1px; height: 24px; background: #e2e8f0; margin: 0 4px; }

.designer-content { display: flex; flex: 1; overflow: hidden; }

.component-library {
  width: 160px;
  background: #fafafa;
  border-right: 1px solid #e2e8f0;
  padding: 16px;
  flex-shrink: 0;
}

.library-title {
  font-weight: 600;
  font-size: 12px;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 12px;
}

.component-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  margin-bottom: 6px;
  background: white;
  border: 1px solid #e2e8f0;
  border-left: 3px solid #e2e8f0;
  border-radius: 6px;
  cursor: grab;
  transition: all 0.15s;
  font-size: 13px;
  color: #475569;
}

.component-item:hover { border-color: inherit; box-shadow: 0 2px 4px rgba(0,0,0,0.08); }
.component-item:active { cursor: grabbing; }

.comp-icon {
  width: 22px; height: 22px; border-radius: 4px;
  display: flex; align-items: center; justify-content: center;
  font-size: 11px; font-weight: 700; color: #fff; flex-shrink: 0;
}

.canvas-container { flex: 1; position: relative; }

.properties-panel {
  width: 260px;
  background: #fafafa;
  border-left: 1px solid #e2e8f0;
  overflow-y: auto;
  flex-shrink: 0;
}

.panel-title {
  font-weight: 600;
  font-size: 12px;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 14px 16px;
  border-bottom: 1px solid #e2e8f0;
  background: #f0f0f0;
}

.no-selection {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  gap: 8px;
  color: #94a3b8;
  padding: 16px;
}

.no-selection p { font-size: 13px; margin: 0; text-align: center; }
</style>
