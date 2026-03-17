<template>
  <div
    class="workflow-node query-node"
    :class="{ selected: selected }"
  >
    <Handle
      type="target"
      :position="Position.Top"
      class="node-handle"
    />
    
    <div class="node-content">
      <div class="node-header">
        <v-icon icon="mdi-magnify" size="small" color="info" class="mr-2" />
        <span class="node-title">{{ data.name || '查询' }}</span>
      </div>
      
      <div class="node-body">
        <v-chip
          size="x-small"
          color="info"
          variant="tonal"
        >
          {{ data.agent || '未指定' }}
        </v-chip>
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
  border-color: rgb(var(--v-theme-info));
  box-shadow: 0 4px 12px rgba(var(--v-theme-info), 0.3);
}

.query-node {
  border-left: 4px solid rgb(var(--v-theme-info));
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

.node-handle {
  width: 12px !important;
  height: 12px !important;
  background: rgb(var(--v-theme-info)) !important;
  border: 2px solid white !important;
}

.node-handle:hover {
  transform: scale(1.2);
}
</style>
