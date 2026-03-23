<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { FolderIcon } from 'lucide-vue-next'

defineProps<{ data?: { name?: string; config?: { flow_id?: string } } }>()
</script>

<template>
  <div class="flow-node subflow-node">
    <Handle type="target" :position="Position.Top" />
    <div class="node-icon">
      <FolderIcon :size="16" />
    </div>
    <div class="node-content">
      <span class="node-label">{{ data?.name || '子流程' }}</span>
      <span v-if="data?.config?.flow_id" class="node-meta">{{ data.config.flow_id }}</span>
    </div>
    <Handle type="source" :position="Position.Bottom" />
  </div>
</template>

<style scoped>
.flow-node {
  padding: 10px 14px;
  border-radius: 8px;
  background: #fff;
  border: 2px solid #06b6d4;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 120px;
}
.node-icon {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: #06b6d4;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}
.node-content {
  display: flex;
  flex-direction: column;
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
</style>