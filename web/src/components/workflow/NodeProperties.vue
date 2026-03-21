<template>
  <div class="node-properties">
    <div class="form-group">
      <label>步骤名称</label>
      <input v-model="localNode.data.name" type="text" @input="onUpdate" />
    </div>

    <div class="form-group">
      <label>Agent</label>
      <select v-model="localNode.data.agent" @change="onUpdate">
        <option value="">请选择...</option>
        <option v-for="agent in agents" :key="agent.id" :value="agent.id">{{ agent.name }}</option>
      </select>
    </div>

    <div class="form-group">
      <label>输入 (JSON)</label>
      <textarea v-model="inputJson" rows="5" :class="{ error: !!inputError }" @input="onInputChange(inputJson)" />
      <div v-if="inputError" class="error-text">{{ inputError }}</div>
    </div>

    <div class="form-group">
      <label>执行条件</label>
      <input v-model="localNode.data.condition" type="text" placeholder="{{variables.count}} > 0" @input="onUpdate" />
      <div class="hint">可选：满足条件时才执行</div>
    </div>

    <div class="form-group">
      <label>超时 (秒): {{ localNode.data.timeout || 0 }}</label>
      <input v-model.number="localNode.data.timeout" type="range" min="0" max="600" step="10" @input="onUpdate" />
    </div>

    <div class="form-group">
      <label>重试次数: {{ localNode.data.retry || 0 }}</label>
      <input v-model.number="localNode.data.retry" type="range" min="0" max="5" step="1" @input="onUpdate" />
    </div>

    <div v-if="(localNode.data.retry as number) > 0" class="form-group">
      <label>重试间隔 (秒): {{ localNode.data.retry_delay || 1 }}</label>
      <input v-model.number="localNode.data.retry_delay" type="range" min="1" max="60" step="1" @input="onUpdate" />
    </div>

    <div class="form-group">
      <label>优先级</label>
      <select v-model="localNode.data.priority" @change="onUpdate">
        <option value="high">高</option>
        <option value="normal">普通</option>
        <option value="low">低</option>
      </select>
    </div>

    <div class="divider" />

    <button class="btn-danger" @click="emit('delete')">
      <Trash2Icon :size="14" /> 删除步骤
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
  node: unknown
  agents?: Agent[]
}>()

const emit = defineEmits<{
  update: [node: unknown]
  delete: []
}>()

const localNode = ref<{ data: Record<string, unknown> }>({ data: {} })
const inputJson = ref('')
const inputError = ref('')

watch(() => props.node, (newNode) => {
  localNode.value = JSON.parse(JSON.stringify(newNode))
  inputJson.value = JSON.stringify((newNode as { data?: { input?: unknown } })?.data?.input || {}, null, 2)
}, { immediate: true })

function onInputChange(value: string) {
  try {
    localNode.value.data.input = JSON.parse(value)
    inputError.value = ''
    emit('update', localNode.value)
  } catch {
    inputError.value = '无效的 JSON 格式'
  }
}

function onUpdate() {
  emit('update', localNode.value)
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
.form-group textarea:focus {
  outline: none;
  border-color: #3b82f6;
}

.form-group textarea.error { border-color: #ef4444; }

.form-group input[type="range"] {
  width: 100%;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
}

.form-group textarea { resize: vertical; font-family: monospace; }

.hint { font-size: 11px; color: #94a3b8; }

.error-text { font-size: 12px; color: #ef4444; }

.divider { height: 1px; background: #e2e8f0; margin: 4px 0; }

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
  transition: all 0.1s;
  width: 100%;
  justify-content: center;
}

.btn-danger:hover { background: rgba(239,68,68,0.08); }
</style>
