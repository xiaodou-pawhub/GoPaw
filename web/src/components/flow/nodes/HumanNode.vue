<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { UserIcon } from 'lucide-vue-next'

defineProps<{ data?: { name?: string } }>()
</script>

<template>
  <div class="flow-node human-node">
    <Handle type="target" :position="Position.Top" />
    <div class="node-icon">
      <UserIcon :size="16" />
    </div>
    <span class="node-label">{{ data?.name || '人工' }}</span>
    <Handle type="source" :position="Position.Bottom" />
  </div>
</template>

<style scoped>
.flow-node {
  padding: 10px 14px;
  border-radius: 8px;
  background: #fff;
  border: 2px solid #f59e0b;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 100px;
}
.node-icon {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: #f59e0b;
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