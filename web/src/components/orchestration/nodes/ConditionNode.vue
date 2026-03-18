<template>
  <div
    class="orchestration-node condition-node"
    :class="{ selected: selected }"
  >
    <!-- 输入连接点（顶部） -->
    <Handle
      type="target"
      :position="Position.Top"
      class="node-handle"
    />

    <!-- 菱形内容 -->
    <div class="diamond-shape">
      <div class="diamond-content">
        <v-icon icon="mdi-source-branch" size="18" color="white" class="mb-1" />
        <span class="node-title">{{ data.name || '条件' }}</span>
        
        <!-- 条件类型标签 -->
        <v-chip
          v-if="conditionTypeLabel"
          size="x-small"
          color="white"
          variant="flat"
          class="mt-1 condition-type"
        >
          {{ conditionTypeLabel }}
        </v-chip>
      </div>
    </div>

    <!-- 输出连接点（左、右、下） -->
    <Handle
      type="source"
      :position="Position.Left"
      class="node-handle handle-left"
      id="left"
    />
    <Handle
      type="source"
      :position="Position.Right"
      class="node-handle handle-right"
      id="right"
    />
    <Handle
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

interface NodeData {
  name?: string
  condition_type?: 'expression' | 'intent' | 'llm'
}

interface Props {
  id: string
  selected?: boolean
  data: NodeData
}

const props = defineProps<Props>()

const conditionTypeLabel = computed(() => {
  switch (props.data.condition_type) {
    case 'expression': return '表达式'
    case 'intent': return '意图'
    case 'llm': return 'LLM'
    default: return ''
  }
})
</script>

<style scoped>
.orchestration-node {
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
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  transform: rotate(45deg);
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.orchestration-node:hover .diamond-shape {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  transform: rotate(45deg) scale(1.05);
}

.orchestration-node.selected .diamond-shape {
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

.condition-type {
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
