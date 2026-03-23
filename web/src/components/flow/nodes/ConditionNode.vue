<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { GitBranchIcon } from 'lucide-vue-next'

defineProps<{ data?: { name?: string; config?: { condition_type?: string } } }>()
</script>

<template>
  <div class="flow-node condition-node">
    <Handle type="target" :position="Position.Top" />
    <div class="node-icon">
      <GitBranchIcon :size="16" />
    </div>
    <div class="node-content">
      <span class="node-label">{{ data?.name || '条件' }}</span>
      <span v-if="data?.config?.condition_type" class="node-meta">
        {{ data.config.condition_type === 'expression' ? '表达式' :
           data.config.condition_type === 'intent' ? '意图匹配' :
           data.config.condition_type === 'llm' ? 'LLM判断' : '' }}
      </span>
    </div>
    <Handle type="source" :position="Position.Bottom" id="true" />
    <Handle type="source" :position="Position.Right" id="false" />
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
</style>