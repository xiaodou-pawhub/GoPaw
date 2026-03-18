<template>
  <BaseNode
    :id="id"
    :selected="selected"
    :data="data"
    :icon="'mdi-robot'"
    :title="data.name || 'Agent'"
    :executing="data.executing"
    :completed="data.completed"
    :show-input-handle="true"
    :show-output-handle="true"
  >
    <template #body>
      <!-- Agent ID -->
      <v-chip
        v-if="data.agent_id"
        size="x-small"
        color="primary"
        variant="tonal"
        class="mb-1"
      >
        {{ data.agent_id }}
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

      <!-- 角色描述 -->
      <div v-if="data.role" class="node-role text-caption">
        {{ data.role }}
      </div>
    </template>
  </BaseNode>
</template>

<script setup lang="ts">
import BaseNode from './BaseNode.vue'

interface NodeData {
  name?: string
  agent_id?: string
  role?: string
  prompt?: string
  config?: Record<string, any>
  executing?: boolean
  completed?: boolean
}

interface Props {
  id: string
  selected?: boolean
  data: NodeData
}

defineProps<Props>()
</script>

<style scoped>
.node-role {
  color: #666;
  margin-top: 4px;
}
</style>
