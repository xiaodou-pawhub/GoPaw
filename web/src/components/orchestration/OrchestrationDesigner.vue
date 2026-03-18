<template>
  <div class="orchestration-designer">
    <!-- 工具栏 -->
    <div class="designer-toolbar">
      <v-btn
        size="small"
        prepend-icon="mdi-content-save"
        color="primary"
        @click="saveOrchestration"
      >
        保存
      </v-btn>
      <v-btn
        size="small"
        prepend-icon="mdi-check-circle"
        @click="validateOrchestration"
      >
        验证
      </v-btn>
      <v-divider vertical class="mx-2" />
      <v-btn
        size="small"
        icon="mdi-delete"
        color="error"
        :disabled="!selectedNode"
        @click="deleteSelectedNode"
      />
    </div>

    <div class="designer-content">
      <!-- 组件库 -->
      <div class="component-library">
        <div class="library-title">组件库</div>
        <div
          v-for="type in nodeTypes"
          :key="type.id"
          class="component-item"
          draggable="true"
          @dragstart="onDragStart($event, type)"
        >
          <v-icon :icon="type.icon" :color="type.color" class="mr-2" />
          <span>{{ type.name }}</span>
        </div>
      </div>

      <!-- 画布 -->
      <div class="canvas-container" @drop="onDrop" @dragover.prevent>
        <VueFlow
          v-model="elements"
          :node-types="vueFlowNodeTypes"
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
          <v-alert type="info" variant="tonal">
            选择一个节点以编辑属性
          </v-alert>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
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
  config?: Record<string, any>
  position: { x: number; y: number }
}

interface OrchestrationEdge {
  id: string
  source: string
  target: string
  message_type: string
  condition?: any
  transform?: any
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

interface NodeType {
  id: string
  name: string
  icon: string
  color: string
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

// 节点类型配置
const nodeTypes: NodeType[] = [
  { id: 'agent', name: 'Agent', icon: 'mdi-robot', color: 'primary' },
  { id: 'human', name: '人工', icon: 'mdi-account', color: 'warning' },
  { id: 'condition', name: '条件', icon: 'mdi-source-branch', color: 'info' },
  { id: 'workflow', name: '工作流', icon: 'mdi-file-tree', color: 'secondary' },
  { id: 'end', name: '结束', icon: 'mdi-flag-checkered', color: 'success' },
]

// Vue Flow 节点类型映射
const vueFlowNodeTypes: any = {
  agent: AgentNode,
  human: HumanNode,
  condition: ConditionNode,
  workflow: WorkflowNode,
  end: EndNode,
}

// 默认边选项
const defaultEdgeOptions: any = {
  animated: true,
  style: { stroke: '#666', strokeWidth: 2 },
  labelStyle: { fill: '#666', fontSize: 12 },
}

// 画布元素
const elements = ref<(Node | Edge)[]>([])
const selectedNode = ref<Node | null>(null)
let nodeIdCounter = 1
let edgeIdCounter = 1

// 监听定义变化
watch(() => props.definition, (newDef) => {
  if (newDef) {
    loadDefinition(newDef)
  }
}, { immediate: true })

// 加载编排定义到画布
function loadDefinition(def: OrchestrationDefinition) {
  const nodes: Node[] = def.nodes.map(n => ({
    id: n.id,
    type: n.type,
    position: n.position,
    data: { ...n },
  }))

  const edges: Edge[] = def.edges.map(e => ({
    id: e.id,
    source: e.source,
    target: e.target,
    label: e.label || e.message_type,
    data: { ...e },
  }))

  elements.value = [...nodes, ...edges]
  
  // 更新计数器
  const maxNodeId = def.nodes.reduce((max: number, n: OrchestrationNode) => {
    const parts = n.id.split('_')
    const num = parts.length > 1 ? parseInt(parts[1]) : 0
    return Math.max(max, num)
  }, 0)
  const maxEdgeId = def.edges.reduce((max: number, e: OrchestrationEdge) => {
    const parts = e.id.split('_')
    const num = parts.length > 1 ? parseInt(parts[1]) : 0
    return Math.max(max, num)
  }, 0)
  nodeIdCounter = maxNodeId + 1
  edgeIdCounter = maxEdgeId + 1
}

// 拖拽开始
function onDragStart(event: DragEvent, type: NodeType) {
  event.dataTransfer?.setData('application/vueflow', type.id)
  event.dataTransfer?.setData('text/plain', type.id)
}

// 放置节点
function onDrop(event: DragEvent) {
  const typeId = event.dataTransfer?.getData('application/vueflow')
  if (!typeId) return

  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const x = event.clientX - rect.left
  const y = event.clientY - rect.top

  const newNode: Node = {
    id: `${typeId}_${nodeIdCounter++}`,
    type: typeId,
    position: { x, y },
    data: {
      name: nodeTypes.find(t => t.id === typeId)?.name || typeId,
    },
  }

  addNodes([newNode])
}

// 节点点击
function onNodeClick({ node }: { node: Node }) {
  selectedNode.value = node
}

// 连接节点
function onConnect(connection: Connection) {
  const newEdge: Edge = {
    id: `edge_${edgeIdCounter++}`,
    source: connection.source,
    target: connection.target,
    label: 'task',
    data: {
      message_type: 'task',
    },
  }
  addEdges([newEdge])
}

// 更新节点数据
function onNodeUpdate(data: Record<string, any>) {
  if (!selectedNode.value) return
  
  const nodeIndex = elements.value.findIndex(e => e.id === selectedNode.value!.id)
  if (nodeIndex >= 0) {
    const node = elements.value[nodeIndex]
    if (!(node as Edge).source) {
      node.data = { ...node.data, ...data }
    }
  }
}

// 删除选中节点
function deleteSelectedNode() {
  if (!selectedNode.value) return
  removeNodes([selectedNode.value.id])
  selectedNode.value = null
}

// 保存编排
function saveOrchestration() {
  const nodes: OrchestrationNode[] = []
  const edges: OrchestrationEdge[] = []
  let startNodeId = ''

  elements.value.forEach(el => {
    if (el.type === 'edge' || (el as Edge).source) {
      // 边
      const edge = el as Edge
      edges.push({
        id: edge.id,
        source: edge.source,
        target: edge.target,
        message_type: edge.data?.message_type || 'task',
        condition: edge.data?.condition,
        transform: edge.data?.transform,
        label: edge.label as string,
      })
    } else {
      // 节点
      const node = el as Node
      const nodeData: OrchestrationNode = {
        id: node.id,
        type: node.type as OrchestrationNode['type'],
        name: node.data.name,
        position: node.position,
      }
      
      if (node.type === 'agent') {
        nodeData.agent_id = node.data.agent_id
        nodeData.role = node.data.role
        nodeData.prompt = node.data.prompt
      } else if (node.type === 'human') {
        nodeData.prompt = node.data.prompt
        nodeData.config = {
          options: node.data.options,
          timeout: node.data.timeout,
        }
      } else if (node.type === 'condition') {
        nodeData.config = {
          condition_type: node.data.condition_type,
        }
      } else if (node.type === 'workflow') {
        nodeData.config = {
          workflow_id: node.data.workflow_id,
        }
      } else if (node.type === 'end') {
        nodeData.config = {
          output_template: node.data.output_template,
        }
      }
      
      nodes.push(nodeData)
      
      // 第一个节点作为起始节点
      if (!startNodeId) {
        startNodeId = node.id
      }
    }
  })

  const definition: OrchestrationDefinition = {
    nodes,
    edges,
    start_node_id: startNodeId,
  }

  emit('save', definition)
}

// 验证编排
function validateOrchestration() {
  saveOrchestration()
  const definition = elements.value
  emit('validate', definition as any)
}
</script>

<style scoped>
.orchestration-designer {
  display: flex;
  flex-direction: column;
  height: 100%;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  overflow: hidden;
}

.designer-toolbar {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  background: #f5f5f5;
  border-bottom: 1px solid #e0e0e0;
}

.designer-content {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.component-library {
  width: 160px;
  background: #fafafa;
  border-right: 1px solid #e0e0e0;
  padding: 16px;
}

.library-title {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 12px;
  color: #333;
}

.component-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  margin-bottom: 8px;
  background: white;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  cursor: grab;
  transition: all 0.2s;
}

.component-item:hover {
  border-color: #1976d2;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.canvas-container {
  flex: 1;
  position: relative;
}

.properties-panel {
  width: 280px;
  background: #fafafa;
  border-left: 1px solid #e0e0e0;
  overflow-y: auto;
}

.panel-title {
  font-weight: 600;
  font-size: 14px;
  padding: 16px;
  background: #f0f0f0;
  border-bottom: 1px solid #e0e0e0;
}

.no-selection {
  padding: 16px;
}
</style>
