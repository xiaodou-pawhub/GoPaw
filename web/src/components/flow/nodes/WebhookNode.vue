<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { WebhookIcon } from 'lucide-vue-next'

defineProps<{ data?: { name?: string } }>()
</script>

<template>
  <div class="flow-node webhook-node">
    <Handle type="target" :position="Position.Top" />
    <div class="node-icon">
      <WebhookIcon :size="16" />
    </div>
    <span class="node-label">{{ data?.name || 'Webhook' }}</span>
    <Handle type="source" :position="Position.Bottom" />
  </div>
</template>

<style scoped>
.flow-node {
  padding: 10px 14px;
  border-radius: 8px;
  background: #fff;
  border: 2px solid #64748b;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 120px;
}
.node-icon {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: #64748b;
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