<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h3>{{ isEditing ? '编辑 Agent' : '新建 Agent' }}</h3>
        <button class="close-btn" @click="$emit('close')">
          <XIcon :size="18" />
        </button>
      </div>

      <div class="modal-body">
        <!-- 基本信息 -->
        <div class="form-section">
          <h4>基本信息</h4>
          <div class="form-row">
            <div class="form-field">
              <label>ID</label>
              <input
                v-model="form.id"
                :disabled="isEditing"
                placeholder="唯一标识，如：coder"
              />
            </div>
            <div class="form-field">
              <label>名称</label>
              <input v-model="form.name" placeholder="显示名称" />
            </div>
          </div>
          <div class="form-row">
            <div class="form-field">
              <label>头像</label>
              <input v-model="form.avatar" placeholder="emoji，如：🤖" />
            </div>
            <div class="form-field">
              <label>描述</label>
              <input v-model="form.description" placeholder="简短描述" />
            </div>
          </div>
        </div>

        <!-- LLM 配置 -->
        <div class="form-section">
          <h4>LLM 配置</h4>
          <div class="form-row">
            <div class="form-field">
              <label>模型</label>
              <input v-model="form.config.llm.model" placeholder="gpt-4" />
            </div>
            <div class="form-field">
              <label>Temperature</label>
              <input
                v-model.number="form.config.llm.temperature"
                type="number"
                min="0"
                max="2"
                step="0.1"
              />
            </div>
          </div>
        </div>

        <!-- 系统提示词 -->
        <div class="form-section">
          <h4>系统提示词</h4>
          <textarea
            v-model="form.config.system_prompt"
            rows="4"
            placeholder="定义 Agent 的角色和行为..."
          />
        </div>

        <!-- 工具配置 -->
        <div class="form-section">
          <h4>工具配置</h4>
          <div class="tools-grid">
            <label
              v-for="tool in availableTools"
              :key="tool"
              class="tool-checkbox"
            >
              <input
                type="checkbox"
                :checked="isToolEnabled(tool)"
                @change="toggleTool(tool)"
              />
              <span>{{ tool }}</span>
            </label>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn secondary" @click="$emit('close')">取消</button>
        <button class="btn primary" :disabled="saving" @click="save">
          <LoaderIcon v-if="saving" :size="14" class="spinning" />
          <span>{{ saving ? '保存中...' : '保存' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { XIcon, LoaderIcon } from 'lucide-vue-next'
import { createAgent, updateAgent, getDefaultConfig, getAvailableTools, type Agent } from '@/api/agents'

// ---- Props ----
const props = defineProps<{
  agent: Agent | null
}>()

// ---- Emits ----
const emit = defineEmits<{
  close: []
  save: []
}>()

// ---- State ----
const isEditing = computed(() => props.agent !== null)
const saving = ref(false)
const availableTools = getAvailableTools()

const form = ref({
  id: props.agent?.id || '',
  name: props.agent?.name || '',
  description: props.agent?.description || '',
  avatar: props.agent?.avatar || '🤖',
  config: props.agent?.config ? { ...props.agent.config } : getDefaultConfig()
})

// ---- Methods ----
function isToolEnabled(tool: string): boolean {
  const enabled = form.value.config.tools.enabled
  const disabled = form.value.config.tools.disabled
  
  if (enabled.length > 0) {
    return enabled.includes(tool)
  }
  return !disabled.includes(tool)
}

function toggleTool(tool: string) {
  const enabled = form.value.config.tools.enabled
  const disabled = form.value.config.tools.disabled
  
  if (enabled.length > 0) {
    // Whitelist mode
    const idx = enabled.indexOf(tool)
    if (idx >= 0) {
      enabled.splice(idx, 1)
    } else {
      enabled.push(tool)
    }
  } else {
    // Blacklist mode
    const idx = disabled.indexOf(tool)
    if (idx >= 0) {
      disabled.splice(idx, 1)
    } else {
      disabled.push(tool)
    }
  }
}

async function save() {
  if (!form.value.id || !form.value.name) {
    alert('请填写 ID 和名称')
    return
  }
  
  saving.value = true
  try {
    if (isEditing.value) {
      await updateAgent(props.agent!.id, {
        name: form.value.name,
        description: form.value.description,
        avatar: form.value.avatar,
        config: form.value.config
      })
    } else {
      await createAgent({
        id: form.value.id,
        name: form.value.name,
        description: form.value.description,
        avatar: form.value.avatar,
        config: form.value.config
      })
    }
    emit('save')
  } catch (err) {
    console.error('Failed to save agent:', err)
    alert('保存失败：' + (err as Error).message)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  width: 90%;
  max-width: 600px;
  max-height: 90vh;
  background: var(--bg-card);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.modal-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.close-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-tertiary);
  cursor: pointer;
}

.close-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-secondary);
}

.modal-body {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
}

.form-section {
  margin-bottom: 20px;
}

.form-section h4 {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 12px;
  text-transform: uppercase;
}

.form-row {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}

.form-field {
  flex: 1;
}

.form-field label {
  display: block;
  font-size: 12px;
  color: var(--text-tertiary);
  margin-bottom: 4px;
}

.form-field input,
.form-field textarea {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-input);
  color: var(--text-primary);
  font-size: 13px;
}

.form-field input:focus,
.form-field textarea:focus {
  outline: none;
  border-color: var(--accent);
}

.form-field input:disabled {
  background: var(--bg-overlay);
  color: var(--text-tertiary);
}

.tools-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 8px;
}

.tool-checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}

.tool-checkbox:hover {
  background: var(--bg-overlay);
}

.tool-checkbox input {
  width: 16px;
  height: 16px;
}

.tool-checkbox span {
  font-size: 13px;
  color: var(--text-secondary);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid var(--border);
}

.btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn.secondary {
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-secondary);
}

.btn.secondary:hover {
  background: var(--bg-overlay);
}

.btn.primary {
  border: none;
  background: var(--accent);
  color: white;
}

.btn.primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
