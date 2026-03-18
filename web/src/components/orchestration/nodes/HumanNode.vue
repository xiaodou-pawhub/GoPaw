<template>
  <div
    class="orchestration-node human-node"
    :class="{ selected: selected, waiting: data.waiting }"
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
          <v-icon icon="mdi-account" size="20" color="white" />
        </div>
        <span class="node-title">{{ data.name || '人工' }}</span>
      </div>

      <!-- 节点主体 -->
      <div class="node-body">
        <!-- 等待提示 -->
        <div v-if="data.waiting" class="waiting-indicator">
          <v-progress-circular
            indeterminate
            size="14"
            width="2"
            color="warning"
            class="mr-1"
          />
          <span class="text-caption text-warning">等待输入...</span>
        </div>

        <!-- 提示摘要 -->
        <div v-if="data.prompt" class="node-prompt text-caption">
          {{ truncatePrompt(data.prompt) }}
        </div>

        <!-- 快捷选项 -->
        <div v-if="data.options && data.options.length > 0" class="node-options">
          <v-chip
            v-for="(option, index) in data.options.slice(0, 3)"
            :key="index"
            size="x-small"
            variant="outlined"
            class="mr-1 mb-1"
          >
            {{ option }}
          </v-chip>
          <span v-if="data.options.length > 3" class="text-caption">+{{ data.options.length - 3 }}</span>
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
  prompt?: string
  options?: string[]
  waiting?: boolean
  timeout?: number
}

interface Props {
  id: string
  selected?: boolean
  data: NodeData
}

const props = defineProps<Props>()

function truncatePrompt(prompt: string): string {
  const maxLength = 40
  if (prompt.length <= maxLength) return prompt
  return prompt.substring(0, maxLength) + '...'
}
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
  border-color: #ff9800;
  box-shadow: 0 0 0 3px rgba(255, 152, 0, 0.2);
}

.orchestration-node.waiting {
  border-color: #ff9800;
  animation: pulse-warning 2s infinite;
}

@keyframes pulse-warning {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(255, 152, 0, 0.4);
  }
  50% {
    box-shadow: 0 0 0 8px rgba(255, 152, 0, 0);
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
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
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

.waiting-indicator {
  display: flex;
  align-items: center;
  background: #fff3e0;
  padding: 4px 8px;
  border-radius: 4px;
}

.node-prompt {
  color: #666;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-options {
  display: flex;
  flex-wrap: wrap;
  margin-top: 4px;
}

.node-handle {
  width: 10px;
  height: 10px;
  background: #ff9800;
  border: 2px solid white;
}

.node-handle:hover {
  background: #f57c00;
  transform: scale(1.2);
}
</style>
