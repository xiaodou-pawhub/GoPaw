<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { SquareIcon, CheckCircleIcon } from 'lucide-vue-next'

defineProps<{
  data?: {
    name?: string
    execStatus?: string
    isCurrent?: boolean
  }
}>()
</script>

<template>
  <div
    class="flow-node end-node"
    :class="{
      'node-running': data?.execStatus === 'running' || data?.isCurrent,
      'node-completed': data?.execStatus === 'completed'
    }"
  >
    <Handle type="target" :position="Position.Top" />
    <div class="node-icon">
      <SquareIcon :size="14" />
    </div>
    <span class="node-label">{{ data?.name || '结束' }}</span>
    <div v-if="data?.execStatus === 'completed'" class="exec-indicator">
      <CheckCircleIcon :size="12" />
    </div>
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
  transition: all 0.2s ease;
}

.node-running {
  border-color: #ef4444;
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.3);
  animation: pulse 1.5s ease-in-out infinite;
}

.node-completed {
  border-color: #22c55e;
  background: #f0fdf4;
}

@keyframes pulse {
  0%, 100% { box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.3); }
  50% { box-shadow: 0 0 0 6px rgba(239, 68, 68, 0.2); }
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

.exec-indicator {
  color: #22c55e;
}
</style>