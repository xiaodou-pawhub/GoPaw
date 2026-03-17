<template>
  <div
    class="workflow-node task-node"
    :class="{ selected: selected }"
  >
    <Handle
      type="target"
      :position="Position.Top"
      class="node-handle"
    />
    
    <div class="node-content">
      <div class="node-header">
        <v-icon icon="mdi-robot" size="small" color="primary" class="mr-2" />
        <span class="node-title">{{ data.name || '任务' }}</span>
      </div>
      
      <div class="node-body">
        <v-chip
          size="x-small"
          color="primary"
          variant="tonal"
          class="mb-1"
        >
          {{ data.agent || '未指定' }}
        </v-chip>
        
        <div v-if="data.condition" class="node-condition">
          <v-icon icon="mdi-filter" size="x-small" class="mr-1" />
          <span class="text-caption">条件</span>
        </div>
      </div>
    </div>
    
    <Handle
      type="source"
      :position="Position.Bottom"
      class="node-handle"
    />
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'

interface NodeData {
  name?: string
  agent?: string
  input?: Record<string, any>
  output?: string[]
  condition?: string
  timeout?: number
  retry?: number
  priority?: string
}

interface Props {
  id: string
  selected?: boolean
  data: NodeData
}

defineProps<Props>()
</script>

<style scoped>
.workflow-node {
  border-radius: 8px;
  padding: 12px;
  min-width: 160px;
  background: white;
  border: 2px solid #e0e0e0;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
}

.workflow-node:hover {
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
}

.workflow-node.selected {
  border-color: rgb(var(--v-theme-primary));
  box-shadow: 0 4px 12px rgba(var(--v-theme-primary), 0.3);
}

.task-node {
  border-left: 4px solid rgb(var(--v-theme-primary));
}

.node-content {
  display: flex;
  flex-direction: column;
}

.node-header {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

.node-title {
  font-weight: 500;
  font-size: 14px;
  color: rgba(0, 0, 0, 0.87);
}

.node-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.node-condition {
  display: flex;
  align-items: center;
  color: rgba(0, 0, 0, 0.6);
}

.node-handle {
  width: 12px !important;
  height: 12px !important;
  background: rgb(var(--v-theme-primary)) !important;
  border: 2px solid white !important;
}

.node-handle:hover {
  transform: scale(1.2);
}
</style>
