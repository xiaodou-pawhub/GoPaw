<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { UserIcon, LoaderIcon, CheckCircleIcon } from 'lucide-vue-next'

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
    class="flow-node human-node"
    :class="{
      'node-running': data?.execStatus === 'running' || data?.isCurrent,
      'node-completed': data?.execStatus === 'completed',
      'node-waiting': data?.execStatus === 'waiting'
    }"
  >
    <Handle type="target" :position="Position.Top" />
    <div class="node-icon">
      <UserIcon :size="16" />
    </div>
    <span class="node-label">{{ data?.name || '人工' }}</span>
    <div v-if="data?.execStatus" class="exec-indicator">
      <LoaderIcon v-if="data.execStatus === 'running'" :size="14" class="spin" />
      <CheckCircleIcon v-else-if="data.execStatus === 'completed'" :size="14" />
      <span v-else-if="data.execStatus === 'waiting'" class="waiting-badge">等待</span>
    </div>
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
  transition: all 0.2s ease;
}

.node-running {
  border-color: #f59e0b;
  box-shadow: 0 0 0 3px rgba(245, 158, 11, 0.3);
  animation: pulse 1.5s ease-in-out infinite;
}

.node-completed {
  border-color: #22c55e;
  background: #f0fdf4;
}

.node-waiting {
  border-color: #f59e0b;
  background: #fffbeb;
  box-shadow: 0 0 0 3px rgba(245, 158, 11, 0.3);
}

@keyframes pulse {
  0%, 100% { box-shadow: 0 0 0 3px rgba(245, 158, 11, 0.3); }
  50% { box-shadow: 0 0 0 6px rgba(245, 158, 11, 0.2); }
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
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

.exec-indicator {
  display: flex;
  align-items: center;
}

.exec-indicator .spin { color: #f59e0b; }
.exec-indicator .check-circle { color: #22c55e; }

.waiting-badge {
  font-size: 10px;
  padding: 2px 6px;
  background: #f59e0b;
  color: #fff;
  border-radius: 4px;
}
</style>