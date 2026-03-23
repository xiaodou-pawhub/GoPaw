<template>
  <div class="node-properties">
    <!-- 节点名称 -->
    <div class="form-group">
      <label>节点名称</label>
      <input v-model="localData.name" type="text" @input="onUpdate" />
    </div>

    <!-- Agent 节点 -->
    <template v-if="nodeType === 'agent'">
      <div class="form-group">
        <label>Agent</label>
        <Combobox
          v-model="localData.agent_id"
          :options="agentOptions"
          placeholder="请选择 Agent..."
          @change="onAgentChange"
        />
      </div>
      <div class="form-group">
        <label>角色描述</label>
        <input v-model="localData.role" type="text" placeholder="如：负责需求分析" @input="onUpdate" />
      </div>
      <div class="form-group">
        <label>Prompt 模板</label>
        <textarea v-model="localData.prompt" rows="4" placeholder="输入 Prompt 模板，可用 {{变量}}" @input="onUpdate" />
      </div>
    </template>

    <!-- 人工节点 -->
    <template v-if="nodeType === 'human'">
      <div class="form-group">
        <label>提示模板</label>
        <textarea v-model="localData.prompt" rows="3" placeholder="如：请确认以下方案：{{content}}" @input="onUpdate" />
      </div>
      <div class="form-group">
        <label>快捷选项</label>
        <div v-for="(_opt, index) in getOptions()" :key="index" class="option-row">
          <input v-model="(getOptions() as string[])[index]" type="text" @input="onUpdate" />
          <button class="icon-btn icon-danger" @click="removeOption(index)">×</button>
        </div>
        <button class="btn-add" @click="addOption">+ 添加选项</button>
      </div>
      <div class="form-group">
        <label>超时 (秒): {{ getConfig().timeout || 0 }}</label>
        <input v-model.number="getConfig().timeout" type="range" min="0" max="3600" step="60" @input="onUpdate" />
      </div>
    </template>

    <!-- 条件节点 -->
    <template v-if="nodeType === 'condition'">
      <div class="form-group">
        <label>条件类型</label>
        <Combobox
          v-model="getConfig().condition_type"
          :options="conditionTypeOptions"
          placeholder="请选择条件类型..."
          @change="onUpdate"
        />
      </div>
      <div v-if="getConfig().condition_type === 'expression'" class="form-group">
        <label>表达式</label>
        <input v-model="getConfig().expression" type="text" placeholder="{{score}} > 80" @input="onUpdate" />
        <span class="hint">支持变量替换和简单比较</span>
      </div>
      <div v-if="getConfig().condition_type === 'intent'" class="form-group">
        <label>意图关键词</label>
        <input v-model="getConfig().intent" type="text" placeholder="查询,订单" @input="onUpdate" />
        <span class="hint">多个关键词用逗号分隔</span>
      </div>
      <div v-if="getConfig().condition_type === 'llm'" class="form-group">
        <label>LLM 判断提示</label>
        <textarea v-model="getConfig().llm_query" rows="3" placeholder="根据用户情绪判断..." @input="onUpdate" />
      </div>
    </template>

    <!-- 并行节点 -->
    <template v-if="nodeType === 'parallel'">
      <div class="form-group">
        <label>最大并发数</label>
        <input v-model.number="getConfig().max_concurrent" type="number" min="1" max="10" @input="onUpdate" />
        <span class="hint">0 表示不限制</span>
      </div>
    </template>

    <!-- 循环节点 -->
    <template v-if="nodeType === 'loop'">
      <div class="form-group">
        <label>循环条件</label>
        <input v-model="getConfig().condition" type="text" placeholder="{{continue}} == true" @input="onUpdate" />
        <span class="hint">条件为 true 时继续循环</span>
      </div>
      <div class="form-group">
        <label>最大循环次数</label>
        <input v-model.number="getConfig().max_loop" type="number" min="1" max="100" @input="onUpdate" />
      </div>
    </template>

    <!-- 子流程节点 -->
    <template v-if="nodeType === 'subflow'">
      <div class="form-group">
        <label>子流程</label>
        <Combobox
          v-model="getConfig().flow_id"
          :options="flowOptions"
          placeholder="请选择子流程..."
          @change="onUpdate"
        />
      </div>
    </template>

    <!-- Webhook 节点 -->
    <template v-if="nodeType === 'webhook'">
      <div class="form-group">
        <label>Webhook 路径</label>
        <input v-model="getConfig().path" type="text" placeholder="/webhook/xxx" @input="onUpdate" />
        <span class="hint">自定义路径标识</span>
      </div>
      <div class="form-group">
        <label>超时 (秒)</label>
        <input v-model.number="getConfig().timeout" type="number" min="60" max="86400" @input="onUpdate" />
        <span class="hint">等待外部回调的最长时间</span>
      </div>
    </template>

    <!-- 结束节点 -->
    <template v-if="nodeType === 'end'">
      <div class="form-group">
        <label>输出模板</label>
        <textarea v-model="getConfig().output_template" rows="3" placeholder="最终答案：{{content}}" @input="onUpdate" />
      </div>
    </template>

    <div class="divider" />

    <button class="btn-danger" @click="emit('delete')">
      <Trash2Icon :size="14" /> 删除节点
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { Trash2Icon } from 'lucide-vue-next'
import Combobox from '@/components/common/Combobox.vue'

interface Agent { id: string; name: string }
interface Flow { id: string; name: string }

const props = defineProps<{
  nodeType: string
  nodeData: Record<string, any>
  agents?: Agent[]
  flows?: Flow[]
}>()

const emit = defineEmits<{
  update: [data: Record<string, any>]
  delete: []
}>()

const localData = ref<Record<string, any>>({ name: '', config: {} })

// Agent 选项
const agentOptions = computed(() => {
  return (props.agents || []).map(a => ({ value: a.id, label: a.name }))
})

// 流程选项
const flowOptions = computed(() => {
  return (props.flows || []).map(f => ({ value: f.id, label: f.name }))
})

// 条件类型选项
const conditionTypeOptions = [
  { value: 'expression', label: '表达式' },
  { value: 'intent', label: '意图匹配' },
  { value: 'llm', label: 'LLM 判断' }
]

function getConfig(): Record<string, any> {
  if (!localData.value.config) localData.value.config = {}
  return localData.value.config
}

function getOptions(): string[] {
  const cfg = getConfig()
  if (!cfg.options) cfg.options = []
  return cfg.options
}

watch(() => props.nodeData, (newData) => {
  localData.value = JSON.parse(JSON.stringify(newData || { name: '', config: {} }))
  if (!localData.value.config) localData.value.config = {}
}, { immediate: true, deep: true })

function onUpdate() {
  emit('update', JSON.parse(JSON.stringify(localData.value)))
}

function onAgentChange(value: string | number | null) {
  // 自动填充 Agent 名称作为节点名称
  const agent = (props.agents || []).find(a => a.id === value)
  if (agent && !localData.value.name) {
    localData.value.name = agent.name
  }
  onUpdate()
}

function addOption() {
  getOptions().push('')
  onUpdate()
}

function removeOption(index: number) {
  getOptions().splice(index, 1)
  onUpdate()
}
</script>

<style scoped>
.node-properties { padding: 12px; }
.form-group { margin-bottom: 12px; }
.form-group label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 4px;
}
.form-group input,
.form-group textarea {
  width: 100%;
  padding: 6px 8px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 12px;
}
.form-group textarea { resize: vertical; }
.hint { font-size: 10px; color: var(--text-muted); margin-top: 2px; display: block; }
.option-row { display: flex; gap: 4px; margin-bottom: 4px; }
.option-row input { flex: 1; }
.icon-btn {
  width: 24px; height: 24px;
  border: none; border-radius: 4px;
  cursor: pointer;
  background: var(--bg-overlay);
}
.icon-danger:hover { background: #fee2e2; color: #ef4444; }
.btn-add {
  width: 100%;
  padding: 6px;
  background: var(--bg-app);
  border: 1px dashed var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
}
.btn-add:hover { border-color: var(--accent); color: var(--accent); }
.divider { height: 1px; background: var(--border); margin: 12px 0; }
.btn-danger {
  width: 100%;
  padding: 8px;
  background: transparent;
  border: 1px solid #ef4444;
  border-radius: 4px;
  color: #ef4444;
  font-size: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}
.btn-danger:hover { background: #fee2e2; }
</style>