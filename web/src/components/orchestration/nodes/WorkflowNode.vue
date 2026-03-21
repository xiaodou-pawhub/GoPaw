<template>
  <BaseNode
    :id="id"
    :selected="selected"
    title="工作流"
    icon-color="#8b5cf6"
    :executing="data.executing"
    :completed="false"
  >
    <span class="wf-chip">{{ data.workflow_id || '未指定' }}</span>
  </BaseNode>
</template>

<script setup lang="ts">
import BaseNode from './BaseNode.vue'

interface NodeData {
  name?: string
  workflow_id?: string
  executing?: boolean
}

defineProps<{ id: string; selected?: boolean; data: NodeData }>()
</script>

<style scoped>
.wf-chip {
  display: inline-block;
  padding: 2px 7px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 500;
  background: rgba(139,92,246,0.12);
  color: #7c3aed;
}
</style>
