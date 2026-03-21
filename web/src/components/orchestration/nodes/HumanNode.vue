<template>
  <BaseNode
    :id="id"
    :selected="selected"
    title="人工"
    icon-color="#f59e0b"
    :executing="data.waiting"
    :completed="false"
  >
    <div v-if="data.waiting" class="waiting-status">
      <span class="dot-pulse" />
      <span class="wait-text">等待输入...</span>
    </div>
    <div v-if="data.prompt" class="node-prompt">{{ truncate(data.prompt, 40) }}</div>
    <div v-if="data.options?.length" class="node-options">
      <span v-for="(opt, i) in data.options.slice(0, 2)" :key="i" class="option-chip">{{ opt }}</span>
      <span v-if="data.options.length > 2" class="more-chip">+{{ data.options.length - 2 }}</span>
    </div>
  </BaseNode>
</template>

<script setup lang="ts">
import BaseNode from './BaseNode.vue'

interface NodeData {
  name?: string
  prompt?: string
  options?: string[]
  waiting?: boolean
  timeout?: number
}

defineProps<{ id: string; selected?: boolean; data: NodeData }>()

function truncate(s: string, len: number) {
  return s.length <= len ? s : s.substring(0, len) + '...'
}
</script>

<style scoped>
.waiting-status { display: flex; align-items: center; gap: 5px; margin-bottom: 3px; }

@keyframes pulse-dot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.dot-pulse {
  width: 8px; height: 8px; border-radius: 50%;
  background: #f59e0b;
  animation: pulse-dot 1.2s ease-in-out infinite;
  display: inline-block;
}

.wait-text { font-size: 11px; color: #d97706; }

.node-prompt { font-size: 11px; color: #64748b; margin-top: 2px; }

.node-options { display: flex; flex-wrap: wrap; gap: 3px; margin-top: 4px; }

.option-chip, .more-chip {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 10px;
}

.option-chip { background: rgba(245,158,11,0.1); color: #d97706; border: 1px solid rgba(245,158,11,0.3); }
.more-chip { color: #94a3b8; }
</style>
