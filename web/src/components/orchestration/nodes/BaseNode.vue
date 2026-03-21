<template>
  <div
    class="gopaw-node"
    :class="[type, { selected, executing, completed }]"
  >
    <Handle
      v-if="showInputHandle"
      type="target"
      :position="Position.Top"
      class="node-handle"
    />

    <div class="node-content">
      <div class="node-header">
        <div class="node-icon" :style="{ background: iconColor }">
          <span class="node-abbr">{{ abbr }}</span>
        </div>
        <span class="node-title">{{ title }}</span>
      </div>

      <div class="node-body">
        <slot />
      </div>

      <div v-if="executing" class="node-status">
        <span class="spinner-small" />
        <span class="status-text">执行中...</span>
      </div>

      <div v-if="completed" class="node-status completed-status">
        <span class="check-icon">✓</span>
        <span class="status-text text-success">已完成</span>
      </div>
    </div>

    <Handle
      v-if="showOutputHandle"
      type="source"
      :position="Position.Bottom"
      class="node-handle"
    />
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { computed } from 'vue'

interface Props {
  id: string
  type?: string
  selected?: boolean
  executing?: boolean
  completed?: boolean
  icon?: string
  title: string
  iconColor?: string
  showInputHandle?: boolean
  showOutputHandle?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  type: 'default',
  selected: false,
  executing: false,
  completed: false,
  iconColor: '#3b82f6',
  showInputHandle: true,
  showOutputHandle: true,
})

// Create abbreviation from title
const abbr = computed(() => {
  return props.title?.charAt(0)?.toUpperCase() || '?'
})
</script>

<style scoped>
.gopaw-node {
  min-width: 140px;
  max-width: 180px;
  background: white;
  border-radius: 12px;
  border: 2px solid #e2e8f0;
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
  transition: all 0.2s ease;
}

.gopaw-node:hover { box-shadow: 0 4px 12px rgba(0,0,0,0.12); }

.gopaw-node.selected { border-color: #3b82f6; box-shadow: 0 0 0 3px rgba(59,130,246,0.2); }

@keyframes pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(59,130,246,0.4); }
  50% { box-shadow: 0 0 0 8px rgba(59,130,246,0); }
}

.gopaw-node.executing { animation: pulse 2s infinite; }

.node-content { padding: 12px; }

.node-header { display: flex; align-items: center; margin-bottom: 8px; gap: 8px; }

.node-icon {
  width: 32px; height: 32px;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}

.node-abbr { font-size: 13px; font-weight: 700; color: #fff; }

.node-title {
  font-weight: 600;
  font-size: 13px;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-body { display: flex; flex-direction: column; gap: 4px; }

.node-status {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 6px;
}

@keyframes spin { to { transform: rotate(360deg); } }

.spinner-small {
  width: 12px; height: 12px;
  border: 2px solid #e2e8f0;
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  display: inline-block;
}

.check-icon { color: #16a34a; font-size: 13px; font-weight: 700; }

.status-text { font-size: 11px; color: #64748b; }
.text-success { color: #16a34a; }

.node-handle {
  width: 10px;
  height: 10px;
  background: #3b82f6;
  border: 2px solid white;
}

.node-handle:hover { transform: scale(1.2); }
</style>
