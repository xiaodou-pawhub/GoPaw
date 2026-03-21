<template>
  <div class="node-properties">
    <div class="form-group">
      <label>节点名称</label>
      <input v-model="localData.name" type="text" @input="onUpdate" />
    </div>

    <!-- Agent 节点 -->
    <template v-if="nodeType === 'agent'">
      <div class="form-group">
        <label>Agent</label>
        <select v-model="localData.agent_id" @change="onUpdate">
          <option value="">请选择...</option>
          <option v-for="a in agents" :key="a.id" :value="a.id">{{ a.name }}</option>
        </select>
      </div>
      <div class="form-group">
        <label>角色描述</label>
        <input v-model="localData.role" type="text" placeholder="如：负责需求分析" @input="onUpdate" />
      </div>
      <div class="form-group">
        <label>角色 Prompt</label>
        <textarea v-model="localData.prompt" rows="4" placeholder="输入角色 Prompt 前缀..." @input="onUpdate" />
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
        <div v-for="(_opt, index) in (localData.config?.options || [])" :key="index" class="option-row">
          <input v-model="localData.config.options[index]" type="text" @input="onUpdate" />
          <button class="icon-btn icon-danger" @click="removeOption(index as number)">×</button>
        </div>
        <button class="btn-add" @click="addOption">+ 添加选项</button>
      </div>
      <div class="form-group">
        <label>超时 (秒): {{ localData.config?.timeout || 0 }}</label>
        <input v-model.number="localData.config.timeout" type="range" min="0" max="3600" step="60" @input="onUpdate" />
      </div>
    </template>

    <!-- 条件节点 -->
    <template v-if="nodeType === 'condition'">
      <div class="form-group">
        <label>条件类型</label>
        <select v-model="localData.config.condition_type" @change="onUpdate">
          <option value="expression">表达式</option>
          <option value="intent">意图匹配</option>
          <option value="llm">LLM 判断</option>
        </select>
      </div>
    </template>

    <!-- 工作流节点 -->
    <template v-if="nodeType === 'workflow'">
      <div class="form-group">
        <label>工作流 ID</label>
        <input v-model="localData.config.workflow_id" type="text" @input="onUpdate" />
      </div>
    </template>

    <!-- 结束节点 -->
    <template v-if="nodeType === 'end'">
      <div class="form-group">
        <label>输出模板</label>
        <textarea v-model="localData.config.output_template" rows="3" placeholder="如：最终答案：{{content}}" @input="onUpdate" />
      </div>
    </template>

    <div class="divider" />

    <button class="btn-danger" @click="emit('delete')">
      <Trash2Icon :size="14" /> 删除节点
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Trash2Icon } from 'lucide-vue-next'

interface Agent {
  id: string
  name: string
}

const props = defineProps<{
  nodeType: string
  nodeData: Record<string, unknown>
  agents?: Agent[]
}>()

const emit = defineEmits<{
  update: [data: Record<string, unknown>]
  delete: []
}>()

const localData = ref<Record<string, any>>({})

watch(() => props.nodeData, (newData) => {
  localData.value = JSON.parse(JSON.stringify(newData))
  if (!localData.value.config) localData.value.config = {}
  if (props.nodeType === 'human' && !(localData.value.config as Record<string, unknown>).options) {
    (localData.value.config as Record<string, unknown>).options = []
  }
}, { immediate: true, deep: true })

function onUpdate() {
  emit('update', { ...localData.value })
}

function addOption() {
  const cfg = localData.value.config as Record<string, unknown>
  if (!cfg) localData.value.config = {}
  if (!Array.isArray((localData.value.config as Record<string, unknown>).options)) {
    (localData.value.config as Record<string, unknown>).options = []
  }
  ;(cfg.options as string[]).push('')
  onUpdate()
}

function removeOption(index: number) {
  const cfg = localData.value.config as Record<string, unknown>
  const opts = cfg?.options as string[]
  if (opts) {
    opts.splice(index, 1)
    onUpdate()
  }
}
</script>

<style scoped>
.node-properties {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form-group label {
  font-size: 11px;
  font-weight: 600;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.form-group input[type="text"],
.form-group select,
.form-group textarea {
  padding: 7px 10px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  color: #1e293b;
  font-size: 13px;
  width: 100%;
  box-sizing: border-box;
}

.form-group input[type="text"]:focus,
.form-group select:focus,
.form-group textarea:focus { outline: none; border-color: #3b82f6; }

.form-group input[type="range"] { width: 100%; padding: 0; border: none; background: transparent; }

.form-group textarea { resize: vertical; }

.option-row {
  display: flex;
  gap: 6px;
  margin-bottom: 6px;
  align-items: center;
}

.option-row input { flex: 1; }

.icon-btn {
  width: 26px; height: 26px;
  border-radius: 4px;
  border: 1px solid #e2e8f0;
  background: transparent;
  cursor: pointer;
  font-size: 16px;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}

.icon-danger { color: #ef4444; }
.icon-danger:hover { background: rgba(239,68,68,0.1); border-color: rgba(239,68,68,0.3); }

.btn-add {
  padding: 5px 10px;
  background: transparent;
  border: 1px dashed #cbd5e1;
  border-radius: 6px;
  color: #64748b;
  font-size: 12px;
  cursor: pointer;
  width: 100%;
}

.btn-add:hover { background: #f1f5f9; }

.divider { height: 1px; background: #e2e8f0; }

.btn-danger {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: transparent;
  border: 1px solid rgba(239,68,68,0.4);
  border-radius: 6px;
  color: #ef4444;
  font-size: 13px;
  cursor: pointer;
  width: 100%;
  justify-content: center;
}

.btn-danger:hover { background: rgba(239,68,68,0.08); }
</style>
