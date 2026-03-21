<template>
  <div class="workflow-node task-node" :class="{ selected }">
    <Handle type="target" :position="Position.Top" class="node-handle task-handle" />
    <div class="node-content">
      <div class="node-header">
        <span class="node-icon task-icon">T</span>
        <span class="node-title">{{ data.name || '任务' }}</span>
      </div>
      <div class="node-body">
        <span class="node-chip task-chip">{{ data.agent || '未指定' }}</span>
        <span v-if="data.condition" class="node-tag">有条件</span>
      </div>
    </div>
    <Handle type="source" :position="Position.Bottom" class="node-handle task-handle" />
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'

interface NodeData {
  name?: string
  agent?: string
  input?: Record<string, unknown>
  output?: string[]
  condition?: string
  timeout?: number
  retry?: number
  priority?: string
}

defineProps<{ id: string; selected?: boolean; data: NodeData }>()
</script>

<style scoped>
.workflow-node {
  border-radius: 8px;
  padding: 12px;
  min-width: 160px;
  background: #fff;
  border: 2px solid #e2e8f0;
  box-shadow: 0 2px 4px rgba(0,0,0,0.08);
  transition: all 0.2s ease;
}

.workflow-node:hover { box-shadow: 0 4px 8px rgba(0,0,0,0.12); }
.workflow-node.selected { border-color: #3b82f6; box-shadow: 0 0 0 3px rgba(59,130,246,0.2); }

.task-node { border-left: 4px solid #3b82f6; }

.node-content { display: flex; flex-direction: column; gap: 8px; }

.node-header { display: flex; align-items: center; gap: 6px; }

.node-icon {
  width: 20px; height: 20px; border-radius: 4px;
  display: flex; align-items: center; justify-content: center;
  font-size: 11px; font-weight: 700; color: #fff; flex-shrink: 0;
}

.task-icon { background: #3b82f6; }

.node-title { font-weight: 500; font-size: 13px; color: #1e293b; }

.node-body { display: flex; flex-wrap: wrap; gap: 4px; }

.node-chip {
  display: inline-block;
  padding: 2px 7px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 500;
}

.task-chip { background: rgba(59,130,246,0.12); color: #2563eb; }

.node-tag {
  display: inline-block;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 10px;
  background: rgba(100,116,139,0.1);
  color: #64748b;
}

.node-handle {
  width: 10px !important; height: 10px !important;
  border: 2px solid white !important;
}

.task-handle { background: #3b82f6 !important; }

.node-handle:hover { transform: scale(1.3); }
</style>
