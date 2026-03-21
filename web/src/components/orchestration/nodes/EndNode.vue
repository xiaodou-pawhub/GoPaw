<template>
  <BaseNode
    :id="id"
    :selected="selected"
    title="结束"
    icon-color="#16a34a"
    :executing="false"
    :completed="data.completed"
    :show-output-handle="false"
  >
    <div v-if="data.completed" class="done-status">✓ 已完成</div>
  </BaseNode>
</template>

<script setup lang="ts">
import BaseNode from './BaseNode.vue'

interface NodeData {
  name?: string
  completed?: boolean
  output_template?: string
}

defineProps<{ id: string; selected?: boolean; data: NodeData }>()
</script>

<style scoped>
.done-status {
  font-size: 11px;
  color: #16a34a;
  font-weight: 600;
  margin-top: 3px;
}
</style>
