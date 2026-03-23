<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { computed } from 'vue'
import { GitBranchIcon, CheckCircleIcon } from 'lucide-vue-next'

const props = defineProps<{
  data?: {
    name?: string
    config?: {
      condition_type?: string
      branches?: Array<{ label: string; condition: string }>
    }
    execStatus?: string
    isCurrent?: boolean
  }
}>()

// 计算分支列表
const branches = computed(() => {
  if (props.data?.config?.branches && props.data.config.branches.length > 0) {
    return props.data.config.branches
  }
  // 默认两个分支：是/否
  return [
    { label: '是', condition: 'true' },
    { label: '否', condition: 'false' }
  ]
})

// 动态计算 Handle 位置
const getHandlePosition = (index: number, total: number) => {
  if (total === 1) {
    return Position.Bottom
  }
  if (total === 2) {
    return index === 0 ? Position.Bottom : Position.Right
  }
  // 多分支：从左到右分布
  return Position.Bottom
}

const getHandleStyle = (index: number, total: number) => {
  if (total <= 2) {
    return {}
  }
  // 多分支时，计算水平位置
  const nodeWidth = 120
  const spacing = nodeWidth / (total + 1)
  return {
    left: `${spacing * (index + 1)}px`,
    transform: 'translateX(-50%)'
  }
}
</script>

<template>
  <div
    class="flow-node condition-node"
    :class="{
      'node-running': data?.execStatus === 'running' || data?.isCurrent,
      'node-completed': data?.execStatus === 'completed'
    }"
  >
    <Handle type="target" :position="Position.Top" />
    <div class="node-icon">
      <GitBranchIcon :size="16" />
    </div>
    <div class="node-content">
      <span class="node-label">{{ data?.name || '条件分支' }}</span>
      <span v-if="data?.config?.condition_type" class="node-meta">
        {{ data.config.condition_type === 'expression' ? '表达式' :
           data.config.condition_type === 'intent' ? '意图匹配' :
           data.config.condition_type === 'llm' ? 'LLM判断' :
           data.config.condition_type === 'switch' ? '多分支' : '' }}
      </span>
      <span v-else class="node-meta">{{ branches.length }} 个分支</span>
    </div>
    <!-- 执行状态指示器 -->
    <div v-if="data?.execStatus === 'completed'" class="exec-indicator">
      <CheckCircleIcon :size="14" />
    </div>
    <!-- 动态渲染多个出口 Handle -->
    <Handle
      v-for="(branch, index) in branches"
      :key="index"
      type="source"
      :position="getHandlePosition(index, branches.length)"
      :id="branch.label || `case_${index}`"
      :style="getHandleStyle(index, branches.length)"
      class="branch-handle"
    />
  </div>
</template>

<style scoped>
.flow-node {
  padding: 10px 14px;
  border-radius: 8px;
  background: #fff;
  border: 2px solid #4facfe;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 100px;
  position: relative;
  transition: all 0.2s ease;
}

.node-running {
  border-color: #4facfe;
  box-shadow: 0 0 0 3px rgba(79, 172, 254, 0.3);
  animation: pulse 1.5s ease-in-out infinite;
}

.node-completed {
  border-color: #22c55e;
  background: #f0fdf4;
}

@keyframes pulse {
  0%, 100% { box-shadow: 0 0 0 3px rgba(79, 172, 254, 0.3); }
  50% { box-shadow: 0 0 0 6px rgba(79, 172, 254, 0.2); }
}

.node-icon {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: #4facfe;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}
.node-content {
  display: flex;
  flex-direction: column;
}
.node-label {
  font-size: 13px;
  font-weight: 500;
  color: #333;
}
.node-meta {
  font-size: 10px;
  color: #666;
}
.exec-indicator {
  color: #22c55e;
}
.branch-handle {
  position: absolute;
  bottom: -6px;
}
</style>