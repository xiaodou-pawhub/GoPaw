<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { GitMergeIcon, LoaderIcon, CheckCircleIcon, XCircleIcon } from 'lucide-vue-next'

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
    class="flow-node merge-node"
    :class="{
      'node-running': data?.execStatus === 'running' || data?.isCurrent,
      'node-completed': data?.execStatus === 'completed',
      'node-failed': data?.execStatus === 'failed',
      'node-waiting': data?.execStatus === 'waiting'
    }"
  >
    <Handle type="target" :position="Position.Top" />
    <div class="node-icon">
      <GitMergeIcon :size="16" />
    </div>
    <div class="node-content">
      <span class="node-label">{{ data?.name || '合并' }}</span>
      <span class="node-type">分支合并</span>
    </div>
    <div v-if="data?.execStatus" class="exec-indicator">
      <LoaderIcon v-if="data.execStatus === 'running'" :size="14" class="spin" />
      <CheckCircleIcon v-else-if="data.execStatus === 'completed'" :size="14" />
      <XCircleIcon v-else-if="data.execStatus === 'failed'" :size="14" />
    </div>
    <Handle type="source" :position="Position.Bottom" />
  </div>
</template>

<style scoped>
.flow-node {
  padding: 10px 14px;
  border-radius: 8px;
  background: #fff;
  border: 2px solid #14b8a6;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 120px;
  transition: all 0.2s ease;
}

.node-running {
  border-color: #14b8a6;
  box-shadow: 0 0 0 3px rgba(20, 184, 166, 0.3);
  animation: pulse 1.5s ease-in-out infinite;
}

.node-completed {
  border-color: #22c55e;
  background: #f0fdf4;
}

.node-failed {
  border-color: #ef4444;
  background: #fef2f2;
}

.node-waiting {
  border-color: #f59e0b;
  background: #fffbeb;
}

@keyframes pulse {
  0%, 100% { box-shadow: 0 0 0 3px rgba(20, 184, 166, 0.3); }
  50% { box-shadow: 0 0 0 6px rgba(20, 184, 166, 0.2); }
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
  background: #14b8a6;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}

.node-content {
  display: flex;
  flex-direction: column;
  flex: 1;
}

.node-label {
  font-size: 13px;
  font-weight: 500;
  color: #333;
}

.node-type {
  font-size: 10px;
  color: #666;
}

.exec-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  color: inherit;
}

.node-running .exec-indicator { color: #14b8a6; }
.node-completed .exec-indicator { color: #22c55e; }
.node-failed .exec-indicator { color: #ef4444; }
.node-waiting .exec-indicator { color: #f59e0b; }
</style>