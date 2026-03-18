<template>
  <BaseNode
    :id="id"
    :selected="selected"
    :data="data"
    :icon="'mdi-file-tree'"
    :title="data.name || '工作流'"
    :executing="data.executing"
    :completed="false"
    :show-input-handle="true"
    :show-output-handle="true"
  >
    <template #body>
      <!-- 工作流 ID -->
      <v-chip
        v-if="data.workflow_id"
        size="x-small"
        color="secondary"
        variant="tonal"
        class="mb-1"
      >
        {{ data.workflow_id }}
      </v-chip>
      <v-chip
        v-else
        size="x-small"
        color="error"
        variant="tonal"
        class="mb-1"
      >
        未指定
      </v-chip>
    </template>
  </BaseNode>
</template>

<script setup lang="ts">
import BaseNode from './BaseNode.vue'

interface NodeData {
  name?: string
  workflow_id?: string
  config?: Record<string, any>
  executing?: boolean
}

interface Props {
  id: string
  selected?: boolean
  data: NodeData
}

defineProps<Props>()
</script>

<style scoped>
/* 工作流节点特定样式 */
</style>
