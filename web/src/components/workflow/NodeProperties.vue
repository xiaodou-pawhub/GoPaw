<template>
  <div class="node-properties">
    <v-text-field
      v-model="localNode.data.name"
      label="步骤名称"
      density="compact"
      variant="outlined"
      class="mb-4"
      @update:model-value="onUpdate"
    />
    
    <v-select
      v-model="localNode.data.agent"
      :items="agentItems"
      label="Agent"
      density="compact"
      variant="outlined"
      class="mb-4"
      @update:model-value="onUpdate"
    />
    
    <v-textarea
      v-model="inputJson"
      label="输入 (JSON)"
      rows="5"
      density="compact"
      variant="outlined"
      class="mb-4"
      :error-messages="inputError"
      @update:model-value="onInputChange"
    />
    
    <v-text-field
      v-model="localNode.data.condition"
      label="执行条件"
      placeholder="{{variables.count}} > 0"
      density="compact"
      variant="outlined"
      class="mb-4"
      hint="可选：满足条件时才执行"
      @update:model-value="onUpdate"
    />
    
    <v-slider
      v-model="localNode.data.timeout"
      label="超时 (秒)"
      min="0"
      max="600"
      step="10"
      thumb-label
      class="mb-4"
      @update:model-value="onUpdate"
    />
    
    <v-slider
      v-model="localNode.data.retry"
      label="重试次数"
      min="0"
      max="5"
      step="1"
      thumb-label
      class="mb-4"
      @update:model-value="onUpdate"
    />
    
    <v-slider
      v-if="localNode.data.retry > 0"
      v-model="localNode.data.retry_delay"
      label="重试间隔 (秒)"
      min="1"
      max="60"
      step="1"
      thumb-label
      class="mb-4"
      @update:model-value="onUpdate"
    />
    
    <v-select
      v-model="localNode.data.priority"
      :items="priorityItems"
      label="优先级"
      density="compact"
      variant="outlined"
      class="mb-4"
      @update:model-value="onUpdate"
    />
    
    <v-divider class="my-4" />
    
    <v-btn
      color="error"
      variant="outlined"
      block
      prepend-icon="mdi-delete"
      @click="onDelete"
    >
      删除步骤
    </v-btn>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'

interface Agent {
  id: string
  name: string
}

const props = defineProps<{
  node: any
  agents?: Agent[]
}>()

const emit = defineEmits<{
  update: [node: any]
  delete: []
}>()

// 本地节点数据
const localNode = ref<any>({ ...props.node })

// 输入 JSON
const inputJson = ref('')
const inputError = ref('')

// Agent 选项
const agentItems = computed(() => {
  return props.agents?.map((agent) => ({
    title: agent.name,
    value: agent.id,
  })) || []
})

// 优先级选项
const priorityItems = [
  { title: '高', value: 'high' },
  { title: '普通', value: 'normal' },
  { title: '低', value: 'low' },
]

// 监听节点变化
watch(() => props.node, (newNode) => {
  localNode.value = { ...newNode }
  inputJson.value = JSON.stringify(newNode.data?.input || {}, null, 2)
}, { immediate: true })

// 输入变化
function onInputChange(value: string) {
  try {
    const parsed = JSON.parse(value)
    localNode.value.data.input = parsed
    inputError.value = ''
    onUpdate()
  } catch (e) {
    inputError.value = '无效的 JSON 格式'
  }
}

// 更新节点
function onUpdate() {
  emit('update', localNode.value)
}

// 删除节点
function onDelete() {
  emit('delete')
}
</script>

<style scoped>
.node-properties {
  padding: 8px 0;
}
</style>
