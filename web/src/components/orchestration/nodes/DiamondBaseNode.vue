<template>
  <div
    class="diamond-node"
    :class="{ selected }"
  >
    <!-- 输入连接点（顶部） -->
    <Handle
      v-if="showInputHandle"
      type="target"
      :position="Position.Top"
      class="node-handle"
    />

    <!-- 菱形内容 -->
    <div class="diamond-shape" :style="shapeStyle">
      <div class="diamond-content">
        <v-icon :icon="icon" size="18" color="white" class="mb-1" />
        <span class="node-title">{{ title }}</span>

        <!-- 类型标签 -->
        <v-chip
          v-if="typeLabel"
          size="x-small"
          color="white"
          variant="flat"
          class="mt-1 type-label"
        >
          {{ typeLabel }}
        </v-chip>
      </div>
    </div>

    <!-- 输出连接点（左、右、下） -->
    <Handle
      v-if="showLeftHandle"
      type="source"
      :position="Position.Left"
      class="node-handle handle-left"
      id="left"
    />
    <Handle
      v-if="showRightHandle"
      type="source"
      :position="Position.Right"
      class="node-handle handle-right"
      id="right"
    />
    <Handle
      v-if="showBottomHandle"
      type="source"
      :position="Position.Bottom"
      class="node-handle handle-bottom"
      id="bottom"
    />
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { computed } from 'vue'

interface Props {
  id: string
  selected?: boolean
  icon: string
  title: string
  typeLabel?: string
  iconColor?: string
  showInputHandle?: boolean
  showLeftHandle?: boolean
  showRightHandle?: boolean
  showBottomHandle?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
  iconColor: '#4facfe',
  showInputHandle: true,
  showLeftHandle: true,
  showRightHandle: true,
  showBottomHandle: true,
})

const shapeStyle = computed(() => ({
  background: `linear-gradient(135deg, ${props.iconColor} 0%, ${props.iconColor}dd 100%)`,
}))
</script>

<style scoped>
.diamond-node {
  width: 100px;
  height: 100px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.diamond-shape {
  width: 80px;
  height: 80px;
  transform: rotate(45deg);
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.diamond-node:hover .diamond-shape {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  transform: rotate(45deg) scale(1.05);
}

.diamond-node.selected .diamond-shape {
  box-shadow: 0 0 0 4px rgba(79, 172, 254, 0.3);
}

.diamond-content {
  transform: rotate(-45deg);
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100px;
}

.node-title {
  font-weight: 600;
  font-size: 12px;
  color: white;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 70px;
}

.type-label {
  font-size: 10px !important;
  height: 18px !important;
}

.node-handle {
  width: 10px;
  height: 10px;
  background: #00f2fe;
  border: 2px solid white;
}

.node-handle:hover {
  background: #00c6fb;
  transform: scale(1.2);
}

.handle-left {
  left: -5px;
}

.handle-right {
  right: -5px;
}

.handle-bottom {
  bottom: -5px;
}
</style>
