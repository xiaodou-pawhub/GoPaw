<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { GitMergeIcon } from 'lucide-vue-next'

defineProps<{ data?: { name?: string } }>()
</script>

<template>
  <div class="flow-node parallel-node">
    <Handle type="target" :position="Position.Top" />
    <div class="node-icon">
      <GitMergeIcon :size="16" />
    </div>
    <span class="node-label">{{ data?.name || '并行' }}</span>
    <Handle type="source" :position="Position.Bottom" id="branch1" />
    <Handle type="source" :position="Position.Left" id="branch2" />
    <Handle type="source" :position="Position.Right" id="branch3" />
  </div>
</template>

<style scoped>
.flow-node {
  padding: 10px 14px;
  border-radius: 8px;
  background: #fff;
  border: 2px solid #8b5cf6;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 100px;
}
.node-icon {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: #8b5cf6;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}
.node-label {
  font-size: 13px;
  font-weight: 500;
  color: #333;
}
</style>