<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { SquareIcon } from 'lucide-vue-next'

defineProps<{ data?: { name?: string } }>()
</script>

<template>
  <div class="flow-node end-node">
    <Handle type="target" :position="Position.Top" />
    <div class="node-icon">
      <SquareIcon :size="14" />
    </div>
    <span class="node-label">{{ data?.name || '结束' }}</span>
  </div>
</template>

<style scoped>
.flow-node {
  padding: 8px 12px;
  border-radius: 8px;
  background: #fff;
  border: 2px solid #ef4444;
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 80px;
}
.node-icon {
  width: 20px;
  height: 20px;
  border-radius: 4px;
  background: #ef4444;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}
.node-label {
  font-size: 12px;
  font-weight: 500;
  color: #333;
}
</style>