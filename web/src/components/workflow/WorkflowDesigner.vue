<template>
  <div class="workflow-designer">
    <!-- 工具栏 -->
    <div class="designer-toolbar">
      <v-btn
        size="small"
        prepend-icon="mdi-content-save"
        color="primary"
        @click="saveWorkflow"
      >
        保存
      </v-btn>
      <v-btn
        size="small"
        prepend-icon="mdi-check-circle"
        @click="validateWorkflow"
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
          @node-click="onNodeClick as any"
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
          :node="selectedNode"
          :agents="agents"
          @update="onNodeUpdate"
          @delete="deleteSelectedNode"
        />
        <div v-else class="no-selection">
          <v-alert type="info" variant="tonal">
            选择一个步骤以编辑属性
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
import type { Connection } from '@vue-flow/core'
import { TaskNode, NotifyNode, QueryNode } from './nodes'
import NodeProperties from './NodeProperties.vue'

// 导入 Vue Flow 样式
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'

interface WorkflowStep {
  id: string
  name: string
  action: 'task' | 'notify' | 'query'
  agent: string
  input?: Record<string, any>
  output?: string[]
  depends_on?: string[]
  condition?: string
  timeout?: number
  retry?: number
  retry_delay?: number
  priority?: string
}

interface WorkflowDefinition {
  steps: WorkflowStep[]
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
  workflow?: WorkflowDefinition
  agents?: Agent[]
}>()

const emit = defineEmits<{
  save: [definition: WorkflowDefinition]
  validate: [definition: WorkflowDefinition]
}>()

// Vue Flow 实例
const { addNodes, addEdges, removeNodes } = useVueFlow()

// 节点类型配置
const nodeTypes: NodeType[] = [
  { id: 'task', name: '任务', icon: 'mdi-robot', color: 'primary' },
  { id: 'notify', name: '通知', icon: 'mdi-bell', color: 'warning' },
  { id: 'query', name: '查询', icon: 'mdi-magnify', color: 'info' },
]

// Vue Flow 节点类型映射
const vueFlowNodeTypes: any = {
  task: TaskNode,
  notify: NotifyNode,
  query: QueryNode,
}

// 默认边选项
const defaultEdgeOptions: any = {
  animated: true,
  style: { stroke: '#666', strokeWidth: 2 },
}

// 画布元素
const elements = ref<any>({ nodes: [], edges: [] })

// 选中的节点
const selectedNode = ref<any>(null)

// 监听 workflow 变化，加载到画布
watch(() => props.workflow, (workflow) => {
  if (workflow) {
    loadWorkflow(workflow)
  }
}, { immediate: true })

// 加载工作流到画布
function loadWorkflow(definition: WorkflowDefinition) {
  const nodes: any[] = []
  const edges: any[] = []
  
  // 计算节点位置（简单的网格布局）
  const gridSize = 200
  const positions = new Map<string, { x: number; y: number }>()
  
  definition.steps.forEach((step) => {
    // 根据依赖关系计算层级
    const level = calculateLevel(step, definition.steps)
    const siblings = definition.steps.filter(s => calculateLevel(s, definition.steps) === level)
    const siblingIndex = siblings.findIndex(s => s.id === step.id)
    
    positions.set(step.id, {
      x: 100 + level * gridSize,
      y: 100 + siblingIndex * 150,
    })
  })
  
  // 创建节点
  definition.steps.forEach((step) => {
    const pos = positions.get(step.id) || { x: 100, y: 100 }
    nodes.push({
      id: step.id,
      type: step.action,
      position: pos,
      data: {
        name: step.name,
        agent: step.agent,
        input: step.input,
        output: step.output,
        condition: step.condition,
        timeout: step.timeout,
        retry: step.retry,
        retry_delay: step.retry_delay,
        priority: step.priority,
      },
    })
    
    // 创建边
    step.depends_on?.forEach((depId) => {
      edges.push({
        id: `${depId}-${step.id}`,
        source: depId,
        target: step.id,
        animated: true,
      })
    })
  })
  
  elements.value = { nodes, edges }
}

// 计算节点层级（带循环依赖检测）
function calculateLevel(step: WorkflowStep, allSteps: WorkflowStep[], visited: Set<string> = new Set()): number {
  // 检测循环依赖
  if (visited.has(step.id)) {
    console.warn(`Circular dependency detected at step: ${step.id}`)
    return 0
  }
  
  if (!step.depends_on || step.depends_on.length === 0) {
    return 0
  }
  
  visited.add(step.id)
  let maxLevel = 0
  
  step.depends_on.forEach((depId) => {
    const dep = allSteps.find(s => s.id === depId)
    if (dep) {
      maxLevel = Math.max(maxLevel, calculateLevel(dep, allSteps, visited) + 1)
    }
  })
  
  visited.delete(step.id)
  return maxLevel
}

// 导出工作流定义
function exportWorkflow(): WorkflowDefinition {
  const nodes = elements.value.nodes || []
  const edges = elements.value.edges || []
  
  const steps: WorkflowStep[] = nodes.map((node: any) => {
    const dependsOn = edges
      .filter((e: any) => e.target === node.id)
      .map((e: any) => e.source)
    
    return {
      id: node.id,
      name: node.data?.name || node.type,
      action: node.type as 'task' | 'notify' | 'query',
      agent: node.data?.agent || '',
      input: node.data?.input || {},
      output: node.data?.output || [],
      depends_on: dependsOn,
      condition: node.data?.condition,
      timeout: node.data?.timeout,
      retry: node.data?.retry,
      retry_delay: node.data?.retry_delay,
      priority: node.data?.priority,
    }
  })
  
  return { steps }
}

// 保存工作流
function saveWorkflow() {
  const definition = exportWorkflow()
  emit('save', definition)
}

// 验证工作流
function validateWorkflow() {
  const definition = exportWorkflow()
  emit('validate', definition)
}

// 拖拽开始
function onDragStart(event: DragEvent, type: NodeType) {
  event.dataTransfer?.setData('nodeType', type.id)
}

// 拖拽放置
function onDrop(event: DragEvent) {
  const nodeType = event.dataTransfer?.getData('nodeType')
  if (!nodeType) return
  
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const x = event.clientX - rect.left
  const y = event.clientY - rect.top
  
  const newNode: any = {
    id: `step_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
    type: nodeType,
    position: { x, y },
    data: {
      name: nodeTypes.find(t => t.id === nodeType)?.name || nodeType,
      agent: '',
      input: {},
    },
  }
  
  addNodes([newNode])
}

// 节点点击
function onNodeClick(_event: any, node: any) {
  selectedNode.value = node
}

// 连接节点
function onConnect(connection: Connection) {
  if (connection.source && connection.target) {
    const newEdge: any = {
      id: `${connection.source}-${connection.target}`,
      source: connection.source,
      target: connection.target,
      animated: true,
    }
    addEdges([newEdge])
  }
}

// 节点更新
function onNodeUpdate(updatedNode: any) {
  const nodes = elements.value.nodes || []
  const index = nodes.findIndex((n: any) => n.id === updatedNode.id)
  if (index !== -1) {
    nodes[index] = updatedNode
    selectedNode.value = updatedNode
  }
}

// 删除选中的节点
function deleteSelectedNode() {
  if (selectedNode.value) {
    removeNodes([selectedNode.value.id])
    selectedNode.value = null
  }
}
</script>

<style scoped>
.workflow-designer {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #f5f5f5;
}

.designer-toolbar {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  background: white;
  border-bottom: 1px solid #e0e0e0;
  gap: 8px;
}

.designer-content {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.component-library {
  width: 200px;
  background: white;
  border-right: 1px solid #e0e0e0;
  padding: 16px;
  overflow-y: auto;
}

.library-title {
  font-weight: 500;
  font-size: 14px;
  margin-bottom: 12px;
  color: rgba(0, 0, 0, 0.87);
}

.component-item {
  display: flex;
  align-items: center;
  padding: 12px;
  margin-bottom: 8px;
  background: #f5f5f5;
  border-radius: 8px;
  cursor: grab;
  transition: all 0.2s ease;
}

.component-item:hover {
  background: #e0e0e0;
  transform: translateX(4px);
}

.component-item:active {
  cursor: grabbing;
}

.canvas-container {
  flex: 1;
  position: relative;
}

.properties-panel {
  width: 300px;
  background: white;
  border-left: 1px solid #e0e0e0;
  padding: 16px;
  overflow-y: auto;
}

.panel-title {
  font-weight: 500;
  font-size: 14px;
  margin-bottom: 16px;
  color: rgba(0, 0, 0, 0.87);
}

.no-selection {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
}

/* Vue Flow 自定义样式 */
:deep(.vue-flow__node) {
  border: none;
  padding: 0;
  background: transparent;
}

:deep(.vue-flow__edge-path) {
  stroke: #666;
  stroke-width: 2;
}

:deep(.vue-flow__edge.animated .vue-flow__edge-path) {
  stroke-dasharray: 5;
  animation: dashdraw 0.5s linear infinite;
}

@keyframes dashdraw {
  from {
    stroke-dashoffset: 10;
  }
  to {
    stroke-dashoffset: 0;
  }
}
</style>
