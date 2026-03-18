<template>
  <div class="node-properties">
    <!-- 通用属性：节点名称 -->
    <v-text-field
      v-model="localData.name"
      label="节点名称"
      density="compact"
      variant="outlined"
      class="mb-4"
      @update:model-value="onUpdate"
    />

    <!-- Agent 节点属性 -->
    <template v-if="nodeType === 'agent'">
      <v-select
        v-model="localData.agent_id"
        :items="agentItems"
        label="Agent"
        density="compact"
        variant="outlined"
        class="mb-4"
        @update:model-value="onUpdate"
      />

      <v-text-field
        v-model="localData.role"
        label="角色描述"
        placeholder="如：负责需求分析"
        density="compact"
        variant="outlined"
        class="mb-4"
        @update:model-value="onUpdate"
      />

      <v-textarea
        v-model="localData.prompt"
        label="角色 Prompt"
        placeholder="输入角色 Prompt 前缀..."
        rows="4"
        density="compact"
        variant="outlined"
        class="mb-4"
        @update:model-value="onUpdate"
      />
    </template>

    <!-- 人工节点属性 -->
    <template v-if="nodeType === 'human'">
      <v-textarea
        v-model="localData.prompt"
        label="提示模板"
        placeholder="如：请确认以下方案：{{content}}"
        rows="3"
        density="compact"
        variant="outlined"
        class="mb-4"
        @update:model-value="onUpdate"
      />

      <div class="mb-4">
        <div class="text-subtitle-2 mb-2">快捷选项</div>
        <div v-for="(_option, index) in (localData.config?.options || [])" :key="index" class="d-flex mb-2">
          <v-text-field
            v-model="localData.config.options[index]"
            density="compact"
            variant="outlined"
            hide-details
            class="flex-grow-1 mr-2"
            @update:model-value="onUpdate"
          />
          <v-btn
            icon="mdi-delete"
            size="small"
            variant="text"
            color="error"
            @click="removeOption(index as number)"
          />
        </div>
        <v-btn
          size="small"
          variant="text"
          prepend-icon="mdi-plus"
          @click="addOption"
        >
          添加选项
        </v-btn>
      </div>

      <v-slider
        v-model="localData.config.timeout"
        label="超时时间（秒）"
        :min="0"
        :max="3600"
        :step="60"
        thumb-label
        class="mb-4"
        @update:model-value="onUpdate"
      />
    </template>

    <!-- 条件节点属性 -->
    <template v-if="nodeType === 'condition'">
      <v-select
        v-model="localData.config.condition_type"
        :items="conditionTypes"
        label="条件类型"
        density="compact"
        variant="outlined"
        class="mb-4"
        @update:model-value="onUpdate"
      />
    </template>

    <!-- 工作流节点属性 -->
    <template v-if="nodeType === 'workflow'">
      <v-select
        v-model="localData.config.workflow_id"
        :items="workflowItems"
        label="工作流"
        density="compact"
        variant="outlined"
        class="mb-4"
        @update:model-value="onUpdate"
      />
    </template>

    <!-- 结束节点属性 -->
    <template v-if="nodeType === 'end'">
      <v-textarea
        v-model="localData.config.output_template"
        label="输出模板"
        placeholder="如：最终答案：{{content}}"
        rows="3"
        density="compact"
        variant="outlined"
        class="mb-4"
        @update:model-value="onUpdate"
      />
    </template>

    <!-- 删除按钮 -->
    <v-btn
      block
      color="error"
      variant="outlined"
      prepend-icon="mdi-delete"
      @click="emit('delete')"
    >
      删除节点
    </v-btn>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'

interface Agent {
  id: string
  name: string
}

interface Workflow {
  id: string
  name: string
}

interface Props {
  nodeType: string
  nodeData: Record<string, any>
  agents?: Agent[]
  workflows?: Workflow[]
}

const props = withDefaults(defineProps<Props>(), {
  agents: () => [],
  workflows: () => [],
})

const emit = defineEmits<{
  update: [data: Record<string, any>]
  delete: []
}>()

// 本地数据副本
const localData = ref<Record<string, any>>({})

// 条件类型选项
const conditionTypes = [
  { title: '表达式', value: 'expression' },
  { title: '意图匹配', value: 'intent' },
  { title: 'LLM 判断', value: 'llm' },
]

// Agent 选项
const agentItems = computed(() => {
  return props.agents.map(a => ({ title: a.name, value: a.id }))
})

// 工作流选项
const workflowItems = computed(() => {
  return props.workflows.map(w => ({ title: w.name, value: w.id }))
})

// 监听节点数据变化
watch(() => props.nodeData, (newData) => {
  localData.value = { ...newData }
  // 确保 config 存在
  if (!localData.value.config) {
    localData.value.config = {}
  }
  // 确保 options 是数组
  if (props.nodeType === 'human' && !localData.value.config.options) {
    localData.value.config.options = []
  }
}, { immediate: true, deep: true })

// 更新节点数据
function onUpdate() {
  emit('update', { ...localData.value })
}

// 添加选项
function addOption() {
  if (!localData.value.config) {
    localData.value.config = {}
  }
  if (!localData.value.config.options) {
    localData.value.config.options = []
  }
  localData.value.config.options.push('')
  onUpdate()
}

// 删除选项
function removeOption(index: number) {
  if (localData.value.config?.options) {
    localData.value.config.options.splice(index, 1)
    onUpdate()
  }
}
</script>

<style scoped>
.node-properties {
  padding: 16px;
}
</style>
