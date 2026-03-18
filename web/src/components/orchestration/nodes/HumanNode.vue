<template>
  <BaseNode
    :id="id"
    :selected="selected"
    :data="data"
    :icon="'mdi-account'"
    :title="data.name || '人工'"
    :executing="data.waiting"
    :completed="false"
    :show-input-handle="true"
    :show-output-handle="true"
  >
    <template #body>
      <!-- 等待提示 -->
      <div v-if="data.waiting" class="waiting-indicator">
        <v-progress-circular
          indeterminate
          size="14"
          width="2"
          color="warning"
          class="mr-1"
        />
        <span class="text-caption text-warning">等待输入...</span>
      </div>

      <!-- 提示摘要 -->
      <div v-if="data.prompt" class="node-prompt text-caption">
        {{ truncatePrompt(data.prompt) }}
      </div>

      <!-- 快捷选项 -->
      <div v-if="data.options && data.options.length > 0" class="node-options">
        <v-chip
          v-for="(option, index) in data.options.slice(0, 3)"
          :key="index"
          size="x-small"
          variant="outlined"
          class="mr-1 mb-1"
        >
          {{ option }}
        </v-chip>
        <span v-if="data.options.length > 3" class="text-caption">+{{ data.options.length - 3 }}</span>
      </div>
    </template>
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

interface Props {
  id: string
  selected?: boolean
  data: NodeData
}

const props = defineProps<Props>()

function truncatePrompt(prompt: string): string {
  const maxLength = 40
  if (prompt.length <= maxLength) return prompt
  return prompt.substring(0, maxLength) + '...'
}
</script>

<style scoped>
.node-prompt {
  color: #666;
  margin-top: 4px;
}

.node-options {
  margin-top: 4px;
}

.waiting-indicator {
  display: flex;
  align-items: center;
  margin-bottom: 4px;
}
</style>
