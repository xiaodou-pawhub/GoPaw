<template>
  <DiamondBaseNode
    :id="id"
    :selected="selected"
    :icon="'mdi-source-branch'"
    :title="data.name || '条件'"
    :type-label="conditionTypeLabel"
    :icon-color="'#4facfe'"
    :show-input-handle="true"
    :show-left-handle="true"
    :show-right-handle="true"
    :show-bottom-handle="true"
  />
</template>

<script setup lang="ts">
import DiamondBaseNode from './DiamondBaseNode.vue'
import { computed } from 'vue'

interface NodeData {
  name?: string
  condition_type?: 'expression' | 'intent' | 'llm'
}

interface Props {
  id: string
  selected?: boolean
  data: NodeData
}

const props = defineProps<Props>()

const conditionTypeLabel = computed(() => {
  switch (props.data.condition_type) {
    case 'expression': return '表达式'
    case 'intent': return '意图'
    case 'llm': return 'LLM'
    default: return ''
  }
})
</script>
