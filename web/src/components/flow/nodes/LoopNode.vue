<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { RepeatIcon } from 'lucide-vue-next'

defineProps<{ data?: { name?: string; config?: { max_loop?: number } } }>()
</script>

<template>
  <div class="flow-node loop-node">
    <Handle type="target" :position="Position.Top" />
    <div class="node-icon">
      <RepeatIcon :size="16" />
    </div>
    <div class="node-content">
      <span class="node-label">{{ data?.name || '循环' }}</span>
      <span v-if="data?.config?.max_loop" class="node-meta">最多 {{ data.config.max_loop }} 次</span>
    </div>
    <Handle type="source" :position="Position.Bottom" id="loop" />
    <Handle type="source" :position="Position.Right" id="exit" />
  </div>
</template>

<style scoped>
.flow-node {
  padding: 10px 14px;
  border-radius: 8px;
  background: #fff;
  border: 2px solid #ec4899;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 100px;
}
.node-icon {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: #ec4899;
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