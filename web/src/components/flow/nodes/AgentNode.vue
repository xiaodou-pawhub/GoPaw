<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { BotIcon, LoaderIcon, CheckCircleIcon, XCircleIcon } from 'lucide-vue-next'

defineProps<{
  data?: {
    name?: string
    agent_id?: string
    execStatus?: string
    isCurrent?: boolean
  }
}>()
</script>

<template>
  <div
    class="flow-node agent-node"
    :class="{
      'node-running': data?.execStatus === 'running' || data?.isCurrent,
      'node-completed': data?.execStatus === 'completed',
      'node-failed': data?.execStatus === 'failed',
      'node-waiting': data?.execStatus === 'waiting'
    }"
  >
    <Handle type="target" :position="Position.Top" />
    <div class="node-icon">
      <BotIcon :size="16" />
    </div>
    <div class="node-content">
      <span class="node-label">{{ data?.name || 'Agent' }}</span>
      <span v-if="data?.agent_id" class="node-meta">{{ data.agent_id }}</span>
    </div>
    <!-- 执行状态指示器 -->
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
  border: 2px solid #3b82f6;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 120px;
  transition: all 0.2s ease;
}

/* 执行中状态 */
.node-running {
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.3);
  animation: pulse 1.5s ease-in-out infinite;
}

/* 完成状态 */
.node-completed {
  border-color: #22c55e;
  background: #f0fdf4;
}

/* 失败状态 */
.node-failed {
  border-color: #ef4444;
  background: #fef2f2;
}

/* 等待状态 */
.node-waiting {
  border-color: #f59e0b;
  background: #fffbeb;
}

@keyframes pulse {
  0%, 100% { box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.3); }
  50% { box-shadow: 0 0 0 6px rgba(59, 130, 246, 0.2); }
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
  background: #3b82f6;
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

.node-meta {
  font-size: 10px;
  color: #666;
}

.exec-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  color: inherit;
}

.node-running .exec-indicator { color: #3b82f6; }
.node-completed .exec-indicator { color: #22c55e; }
.node-failed .exec-indicator { color: #ef4444; }
.node-waiting .exec-indicator { color: #f59e0b; }
</style>