<template>
  <div class="workflow-node query-node" :class="{ selected }">
    <Handle type="target" :position="Position.Top" class="node-handle query-handle" />
    <div class="node-content">
      <div class="node-header">
        <span class="node-icon query-icon">Q</span>
        <span class="node-title">{{ data.name || '查询' }}</span>
      </div>
      <div class="node-body">
        <span class="node-chip query-chip">{{ data.agent || '未指定' }}</span>
      </div>
    </div>
    <Handle type="source" :position="Position.Bottom" class="node-handle query-handle" />
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'

interface NodeData {
  name?: string
  agent?: string
  input?: Record<string, unknown>
}

defineProps<{ id: string; selected?: boolean; data: NodeData }>()
</script>

<style scoped>
.workflow-node {
  border-radius: 8px; padding: 12px; min-width: 160px;
  background: #fff; border: 2px solid #e2e8f0;
  box-shadow: 0 2px 4px rgba(0,0,0,0.08); transition: all 0.2s ease;
}
.workflow-node:hover { box-shadow: 0 4px 8px rgba(0,0,0,0.12); }
.workflow-node.selected { border-color: #6366f1; box-shadow: 0 0 0 3px rgba(99,102,241,0.2); }
.query-node { border-left: 4px solid #6366f1; }
.node-content { display: flex; flex-direction: column; gap: 8px; }
.node-header { display: flex; align-items: center; gap: 6px; }
.node-icon { width: 20px; height: 20px; border-radius: 4px; display: flex; align-items: center; justify-content: center; font-size: 11px; font-weight: 700; color: #fff; flex-shrink: 0; }
.query-icon { background: #6366f1; }
.node-title { font-weight: 500; font-size: 13px; color: #1e293b; }
.node-body { display: flex; flex-wrap: wrap; gap: 4px; }
.node-chip { display: inline-block; padding: 2px 7px; border-radius: 10px; font-size: 11px; font-weight: 500; }
.query-chip { background: rgba(99,102,241,0.12); color: #4f46e5; }
.node-handle { width: 10px !important; height: 10px !important; border: 2px solid white !important; }
.query-handle { background: #6366f1 !important; }
.node-handle:hover { transform: scale(1.3); }
</style>
