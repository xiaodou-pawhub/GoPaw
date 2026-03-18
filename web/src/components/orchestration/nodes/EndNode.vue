<template>
  <BaseNode
    :id="id"
    :selected="selected"
    :data="data"
    :icon="'mdi-flag-checkered'"
    :title="data.name || '结束'"
    :executing="false"
    :completed="data.completed"
    :show-input-handle="true"
    :show-output-handle="false"
  >
    <template #body>
      <!-- 完成状态 -->
      <div v-if="data.completed" class="completion-status">
        <v-icon icon="mdi-check-circle" size="16" color="success" />
        <span class="text-caption text-success ml-1">已完成</span>
      </div>
    </template>
  </BaseNode>
</template>

<script setup lang="ts">
import BaseNode from './BaseNode.vue'

interface NodeData {
  name?: string
  completed?: boolean
  output_template?: string
}

interface Props {
  id: string
  selected?: boolean
  data: NodeData
}

defineProps<Props>()
</script>

<style scoped>
.completion-status {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 4px;
}
</style>
