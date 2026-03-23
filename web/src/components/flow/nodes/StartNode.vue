<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { PlayIcon, CheckCircleIcon } from 'lucide-vue-next'

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
    class="flow-node start-node"
    :class="{
      'node-running': data?.execStatus === 'running' || data?.isCurrent,
      'node-completed': data?.execStatus === 'completed'
    }"
  >
    <div class="node-icon">
      <PlayIcon :size="16" />
    </div>
    <span class="node-label">{{ data?.name || '开始' }}</span>
    <div v-if="data?.execStatus === 'completed'" class="exec-indicator">
      <CheckCircleIcon :size="12" />
    </div>
    <Handle type="source" :position="Position.Bottom" />
  </div>
</template>

<style scoped>
.flow-node {
  padding: 8px 12px;
  border-radius: 8px;
  background: #fff;
  border: 2px solid #22c55e;
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 80px;
  transition: all 0.2s ease;
}

.node-running {
  border-color: #22c55e;
  box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.3);
  animation: pulse 1.5s ease-in-out infinite;
}

.node-completed {
  border-color: #22c55e;
  background: #f0fdf4;
}

@keyframes pulse {
  0%, 100% { box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.3); }
  50% { box-shadow: 0 0 0 6px rgba(34, 197, 94, 0.2); }
}

.node-icon {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #22c55e;
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