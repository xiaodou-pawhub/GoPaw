<template>
  <BaseNode
    :id="id"
    :selected="selected"
    title="Agent"
    icon-color="#3b82f6"
    :executing="data.executing"
    :completed="data.completed"
  >
    <span class="node-chip agent-chip">{{ data.agent_id || '未指定' }}</span>
    <div v-if="data.role" class="node-role">{{ data.role }}</div>
  </BaseNode>
</template>

<script setup lang="ts">
import BaseNode from './BaseNode.vue'

interface NodeData {
  name?: string
  agent_id?: string
  role?: string
  prompt?: string
  executing?: boolean
  completed?: boolean
}

defineProps<{ id: string; selected?: boolean; data: NodeData }>()
</script>

<style scoped>
.node-chip {
  display: inline-block;
  padding: 2px 7px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 500;
}

.agent-chip { background: rgba(59,130,246,0.12); color: #2563eb; }

.node-role {
  font-size: 11px;
  color: #64748b;
  margin-top: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
