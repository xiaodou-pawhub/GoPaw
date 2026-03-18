<template>
  <div
    class="gopaw-node"
    :class="[type, { selected, executing, completed }]"
  >
    <!-- 输入连接点 -->
    <Handle
      v-if="showInputHandle"
      type="target"
      :position="Position.Top"
      class="node-handle"
    />

    <div class="node-content">
      <!-- 节点头部：图标 + 名称 -->
      <div class="node-header">
        <div class="node-icon" :style="iconStyle">
          <v-icon :icon="icon" size="20" color="white" />
        </div>
        <span class="node-title">{{ title }}</span>
      </div>

      <!-- 节点主体（插槽） -->
      <div class="node-body">
        <slot />
      </div>

      <!-- 执行状态 -->
      <div v-if="executing" class="node-status">
        <v-progress-circular
          indeterminate
          size="14"
          width="2"
          :color="statusColor"
          class="mr-1"
        />
        <span class="text-caption">执行中...</span>
      </div>

      <div v-if="completed" class="node-status">
        <v-icon icon="mdi-check-circle" size="14" :color="statusColor" class="mr-1" />
        <span class="text-caption text-success">已完成</span>
      </div>
    </div>

    <!-- 输出连接点 -->
    <Handle
      v-if="showOutputHandle"
      type="source"
      :position="Position.Bottom"
      class="node-handle"
    />
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { computed } from 'vue'

interface Props {
  id: string
  type?: string
  selected?: boolean
  executing?: boolean
  completed?: boolean
  icon: string
  title: string
  iconColor?: string
  showInputHandle?: boolean
  showOutputHandle?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  type: 'default',
  selected: false,
  executing: false,
  completed: false,
  iconColor: '#1976d2',
  showInputHandle: true,
  showOutputHandle: true,
})

const iconStyle = computed(() => ({
  background: props.iconColor,
}))

const statusColor = computed(() => {
  return props.iconColor
})
</script>

<style scoped>
.gopaw-node {
  min-width: 140px;
  max-width: 180px;
  background: white;
  border-radius: 12px;
  border: 2px solid #e0e0e0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
}

.gopaw-node:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.gopaw-node.selected {
  border-color: #1976d2;
  box-shadow: 0 0 0 3px rgba(25, 118, 210, 0.2);
}

.gopaw-node.executing {
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(25, 118, 210, 0.4);
  }
  50% {
    box-shadow: 0 0 0 8px rgba(25, 118, 210, 0);
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
  background: #1976d2;
  border: 2px solid white;
}

.node-handle:hover {
  transform: scale(1.2);
}
</style>
