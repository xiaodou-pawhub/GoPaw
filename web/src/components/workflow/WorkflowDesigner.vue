<template>
  <div class="workflow-designer">
    <!-- 工具栏 -->
    <div class="designer-toolbar">
      <button class="toolbar-btn toolbar-btn-primary" @click="saveWorkflow">
        <SaveIcon :size="14" /> 保存
      </button>
      <button class="toolbar-btn" @click="validateWorkflow">
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
          v-model="(elements as any)"
          :node-types="(vueFlowNodeTypes as any)"
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
          <MousePointerIcon :size="24" />
          <p>选择一个步骤以编辑属性</p>
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
import type { Connection } from '@vue-flow/core'
import { TaskNode, NotifyNode, QueryNode } from './nodes'
import NodeProperties from './NodeProperties.vue'

import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'

interface WorkflowStep {
  id: string
  name: string
  action: 'task' | 'notify' | 'query'
  agent: string
  input?: Record<string, unknown>
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
  color: string
  abbr: string
}

const props = defineProps<{
  workflow?: WorkflowDefinition
  agents?: Agent[]
}>()

const emit = defineEmits<{
  save: [definition: WorkflowDefinition]
  validate: [definition: WorkflowDefinition]
}>()

const { addNodes, addEdges, removeNodes } = useVueFlow()

const nodeTypes: NodeType[] = [
  { id: 'task',   name: '任务', color: '#3b82f6', abbr: 'T' },
  { id: 'notify', name: '通知', color: '#f59e0b', abbr: 'N' },
  { id: 'query',  name: '查询', color: '#6366f1', abbr: 'Q' },
]

const vueFlowNodeTypes: Record<string, unknown> = { task: TaskNode, notify: NotifyNode, query: QueryNode }

const defaultEdgeOptions = {
  animated: true,
  style: { stroke: '#666', strokeWidth: 2 },
}

const elements = ref<{ nodes: unknown[]; edges: unknown[] }>({ nodes: [], edges: [] })
const selectedNode = ref<unknown>(null)

watch(() => props.workflow, (workflow) => {
  if (workflow) loadWorkflow(workflow)
}, { immediate: true })

function loadWorkflow(definition: WorkflowDefinition) {
  const nodes: unknown[] = []
  const edges: unknown[] = []
  const gridSize = 200
  const positions = new Map<string, { x: number; y: number }>()

  definition.steps.forEach((step) => {
    const level = calculateLevel(step, definition.steps)
    const siblings = definition.steps.filter(s => calculateLevel(s, definition.steps) === level)
    const siblingIndex = siblings.findIndex(s => s.id === step.id)
    positions.set(step.id, { x: 100 + level * gridSize, y: 100 + siblingIndex * 150 })
  })

  definition.steps.forEach((step) => {
    const pos = positions.get(step.id) || { x: 100, y: 100 }
    nodes.push({ id: step.id, type: step.action, position: pos, data: { ...step } })
    step.depends_on?.forEach((depId) => {
      edges.push({ id: `${depId}-${step.id}`, source: depId, target: step.id, animated: true })
    })
  })

  elements.value = { nodes, edges }
}

function calculateLevel(step: WorkflowStep, allSteps: WorkflowStep[], visited: Set<string> = new Set()): number {
  if (visited.has(step.id)) return 0
  if (!step.depends_on || step.depends_on.length === 0) return 0
  visited.add(step.id)
  let maxLevel = 0
  step.depends_on.forEach((depId) => {
    const dep = allSteps.find(s => s.id === depId)
    if (dep) maxLevel = Math.max(maxLevel, calculateLevel(dep, allSteps, visited) + 1)
  })
  visited.delete(step.id)
  return maxLevel
}

function exportWorkflow(): WorkflowDefinition {
  const nodes = (elements.value.nodes || []) as Array<{ id: string; type: string; data?: Record<string, unknown> }>
  const edges = (elements.value.edges || []) as Array<{ source: string; target: string }>

  const steps: WorkflowStep[] = nodes.map((node) => {
    const dependsOn = edges.filter(e => e.target === node.id).map(e => e.source)
    return {
      id: node.id,
      name: (node.data?.name as string) || node.type,
      action: node.type as 'task' | 'notify' | 'query',
      agent: (node.data?.agent as string) || '',
      input: (node.data?.input as Record<string, unknown>) || {},
      output: (node.data?.output as string[]) || [],
      depends_on: dependsOn,
      condition: node.data?.condition as string | undefined,
      timeout: node.data?.timeout as number | undefined,
      retry: node.data?.retry as number | undefined,
      retry_delay: node.data?.retry_delay as number | undefined,
      priority: node.data?.priority as string | undefined,
    }
  })

  return { steps }
}

function saveWorkflow() { emit('save', exportWorkflow()) }
function validateWorkflow() { emit('validate', exportWorkflow()) }

function onDragStart(event: DragEvent, type: NodeType) {
  event.dataTransfer?.setData('nodeType', type.id)
}

function onDrop(event: DragEvent) {
  const nodeType = event.dataTransfer?.getData('nodeType')
  if (!nodeType) return
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const newNode = {
    id: `step_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
    type: nodeType,
    position: { x: event.clientX - rect.left, y: event.clientY - rect.top },
    data: { name: nodeTypes.find(t => t.id === nodeType)?.name || nodeType, agent: '', input: {} },
  }
  addNodes([newNode])
}

function onNodeClick(_event: unknown, node: unknown) {
  selectedNode.value = node
}

function onConnect(connection: Connection) {
  if (connection.source && connection.target) {
    addEdges([{ id: `${connection.source}-${connection.target}`, source: connection.source, target: connection.target, animated: true }])
  }
}

function onNodeUpdate(updatedNode: unknown) {
  const nodes = (elements.value.nodes || []) as Array<{ id: string }>
  const index = nodes.findIndex(n => n.id === (updatedNode as { id: string }).id)
  if (index !== -1) {
    nodes[index] = updatedNode as { id: string }
    selectedNode.value = updatedNode
  }
}

function deleteSelectedNode() {
  if (selectedNode.value) {
    removeNodes([(selectedNode.value as { id: string }).id])
    selectedNode.value = null
  }
}
</script>

<style scoped>
.workflow-designer {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #f8fafc;
}

.designer-toolbar {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  background: #fff;
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

.toolbar-divider {
  width: 1px;
  height: 24px;
  background: #e2e8f0;
  margin: 0 4px;
}

.designer-content {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.component-library {
  width: 180px;
  background: #fff;
  border-right: 1px solid #e2e8f0;
  padding: 16px;
  overflow-y: auto;
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
  gap: 10px;
  padding: 10px 12px;
  margin-bottom: 6px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-left: 3px solid #e2e8f0;
  border-radius: 6px;
  cursor: grab;
  transition: all 0.15s;
  font-size: 13px;
  color: #475569;
}

.component-item:hover {
  background: #f1f5f9;
  transform: translateX(3px);
}

.component-item:active { cursor: grabbing; }

.comp-icon {
  width: 22px;
  height: 22px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
}

.canvas-container {
  flex: 1;
  position: relative;
}

.properties-panel {
  width: 280px;
  background: #fff;
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
}

.no-selection {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  gap: 8px;
  color: #94a3b8;
}

.no-selection p { font-size: 13px; margin: 0; }

:deep(.vue-flow__node) { border: none; padding: 0; background: transparent; }
</style>
