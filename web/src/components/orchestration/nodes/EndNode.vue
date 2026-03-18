<template>
  <div
    class="orchestration-node end-node"
    :class="{ selected: selected, completed: data.completed }"
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
          <v-icon icon="mdi-flag-checkered" size="20" color="white" />
        </div>
        <span class="node-title">{{ data.name || '结束' }}</span>
      </div>

      <!-- 完成状态 -->
      <div v-if="data.completed" class="completion-status">
        <v-icon icon="mdi-check-circle" size="16" color="success" />
        <span class="text-caption text-success ml-1">已完成</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'

interface NodeData {
  name?: string
  completed?: boolean
  output_template?: string
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
  min-width: 100px;
  max-width: 140px;
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
  border-color: #4caf50;
  box-shadow: 0 0 0 3px rgba(76, 175, 80, 0.2);
}

.orchestration-node.completed {
  border-color: #4caf50;
  background: #f1f8e9;
}

.node-content {
  padding: 12px;
}

.node-header {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
}

.node-icon {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: linear-gradient(135deg, #4caf50 0%, #8bc34a 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 8px;
}

.node-title {
  font-weight: 600;
  font-size: 14px;
  color: #333;
}

.completion-status {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 8px;
  padding: 4px 8px;
  background: #e8f5e9;
  border-radius: 4px;
}

.node-handle {
  width: 10px;
  height: 10px;
  background: #4caf50;
  border: 2px solid white;
}

.node-handle:hover {
  background: #388e3c;
  transform: scale(1.2);
}
</style>
