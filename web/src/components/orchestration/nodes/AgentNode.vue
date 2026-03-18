<template>
  <div
    class="orchestration-node agent-node"
    :class="{ selected: selected, executing: data.executing }"
  >
    <!-- 输入连接点 -->
    <Handle
      type="target"
      :position="Position.Top"
      class="node-handle"
    />

    <div class="node-content">
      <!-- 节点头部：图标 + 名称 -->
      <div class="node-header">
        <div class="node-icon">
          <v-icon icon="mdi-robot" size="20" color="white" />
        </div>
        <span class="node-title">{{ data.name || 'Agent' }}</span>
      </div>

      <!-- 节点主体 -->
      <div class="node-body">
        <!-- Agent ID -->
        <v-chip
          v-if="data.agent_id"
          size="x-small"
          color="primary"
          variant="tonal"
          class="mb-1"
        >
          {{ data.agent_id }}
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

        <!-- 角色描述 -->
        <div v-if="data.role" class="node-role text-caption">
          {{ data.role }}
        </div>

        <!-- 执行状态 -->
        <div v-if="data.executing" class="node-status">
          <v-progress-circular
            indeterminate
            size="16"
            width="2"
            color="primary"
            class="mr-1"
          />
          <span class="text-caption">执行中...</span>
        </div>

        <!-- 完成标记 -->
        <div v-if="data.completed" class="node-status">
          <v-icon icon="mdi-check-circle" size="16" color="success" class="mr-1" />
          <span class="text-caption text-success">已完成</span>
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
  agent_id?: string
  role?: string
  prompt?: string
  config?: Record<string, any>
  executing?: boolean
  completed?: boolean
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
  border-radius: 12px;
  border: 2px solid #e0e0e0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
}

.orchestration-node:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.orchestration-node.selected {
  border-color: #1976d2;
  box-shadow: 0 0 0 3px rgba(25, 118, 210, 0.2);
}

.orchestration-node.executing {
  border-color: #4caf50;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(76, 175, 80, 0.4);
  }
  50% {
    box-shadow: 0 0 0 8px rgba(76, 175, 80, 0);
  }
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
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
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

.node-role {
  color: #666;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-status {
  display: flex;
  align-items: center;
  margin-top: 4px;
}

.node-handle {
  width: 10px;
  height: 10px;
  background: #1976d2;
  border: 2px solid white;
}

.node-handle:hover {
  background: #1565c0;
  transform: scale(1.2);
}
</style>
