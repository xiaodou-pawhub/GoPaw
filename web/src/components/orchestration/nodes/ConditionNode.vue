<template>
  <DiamondBaseNode
    :id="id"
    :selected="selected"
    title="条件"
    :type-label="conditionTypeLabel"
    icon-color="#4facfe"
  />
</template>

<script setup lang="ts">
import DiamondBaseNode from './DiamondBaseNode.vue'
import { computed } from 'vue'

interface NodeData {
  name?: string
  condition_type?: 'expression' | 'intent' | 'llm'
}

const props = defineProps<{ id: string; selected?: boolean; data: NodeData }>()

const conditionTypeLabel = computed(() => {
  const map: Record<string, string> = { expression: '表达式', intent: '意图', llm: 'LLM' }
  return map[props.data.condition_type || ''] || ''
})
</script>
