<template>
  <div
    class="orchestration-node workflow-node"
    :class="{ selected: selected }"
  >
    <!-- 输入连接点 -->
    <Handle
      type="target"
      :position="Position.Top"
      class="node-handle"
    />

    <div class="node-content">
      <!-- 节点头部 -->
      <div class="node-header">
        <div class="node-icon">
          <v-icon icon="mdi-file-tree" size="20" color="white" />
        </div>
        <span class="node-title">{{ data.name || '工作流' }}</span>
      </div>

      <!-- 节点主体 -->
      <div class="node-body">
        <!-- 工作流 ID -->
        <v-chip
          v-if="data.workflow_id"
          size="x-small"
          color="secondary"
          variant="tonal"
          class="mb-1"
        >
          {{ data.workflow_id }}
        </v-chip>
        <v-chip
          v-else
          size="x-small"
          color="error"
          variant="tonal"
          class="mb-1"
        >
          未指定
        </v-chip>

        <!-- 执行状态 -->
        <div v-if="data.executing" class="node-status">
          <v-progress-circular
            indeterminate
            size="14"
            width="2"
            color="secondary"
            class="mr-1"
          />
          <span class="text-caption">执行中...</span>
        </div>
      </div>
    </div>

    <!-- 输出连接点 -->
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
  workflow_id?: string
  config?: Record<string, any>
  executing?: boolean
}

interface Props {
  id: string
  selected?: boolean
  data: NodeData
}

defineProps<Props>()
</script>

<style scoped>
.orchestration-node {
  min-width: 140px;
  max-width: 180px;
  background: white;
  border-radius: 8px;
  border: 2px solid #e0e0e0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
}

.orchestration-node:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.orchestration-node.selected {
  border-color: #9c27b0;
  box-shadow: 0 0 0 3px rgba(156, 39, 176, 0.2);
}

.node-content {
  padding: 12px;
}

.node-header {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

.node-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: linear-gradient(135deg, #9c27b0 0%, #e91e63 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 8px;
}

.node-title {
  font-weight: 600;
  font-size: 14px;
  color: #333;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.node-status {
  display: flex;
  align-items: center;
  margin-top: 4px;
}

.node-handle {
  width: 10px;
  height: 10px;
  background: #9c27b0;
  border: 2px solid white;
}

.node-handle:hover {
  background: #7b1fa2;
  transform: scale(1.2);
}
</style>
