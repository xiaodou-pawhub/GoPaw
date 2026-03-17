<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h3>{{ isEditing ? '编辑 MCP 服务器' : '添加 MCP 服务器' }}</h3>
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
                placeholder="唯一标识，如：filesystem"
              />
            </div>
            <div class="form-field">
              <label>名称</label>
              <input v-model="form.name" placeholder="显示名称" />
            </div>
          </div>
          <div class="form-field">
            <label>描述</label>
            <input v-model="form.description" placeholder="简短描述" />
          </div>
        </div>

        <!-- 传输方式 -->
        <div class="form-section">
          <h4>传输方式</h4>
          <div class="transport-options">
            <label
              v-for="t in transports"
              :key="t.value"
              class="transport-option"
              :class="{ active: form.transport === t.value }"
            >
              <input
                type="radio"
                v-model="form.transport"
                :value="t.value"
              />
              <span class="option-name">{{ t.label }}</span>
              <span class="option-desc">{{ t.desc }}</span>
            </label>
          </div>
        </div>

        <!-- Stdio 配置 -->
        <div v-if="form.transport === 'stdio'" class="form-section">
          <h4>命令配置</h4>
          <div class="form-field">
            <label>命令</label>
            <input v-model="form.command" placeholder="如：npx、python、node" />
          </div>
          <div class="form-field">
            <label>参数（每行一个）</label>
            <textarea
              v-model="argsText"
              rows="3"
              placeholder="-y&#10;@modelcontextprotocol/server-filesystem&#10;/path/to/workspace"
            />
          </div>
          <div class="form-field">
            <label>环境变量（每行一个，格式：KEY=value）</label>
            <textarea
              v-model="envText"
              rows="2"
              placeholder="GITHUB_TOKEN=xxx&#10;API_KEY=yyy"
            />
          </div>
        </div>

        <!-- SSE 配置 -->
        <div v-else class="form-section">
          <h4>SSE 配置</h4>
          <div class="form-field">
            <label>URL</label>
            <input v-model="form.url" placeholder="如：http://localhost:3000/sse" />
          </div>
        </div>

        <!-- 预设模板 -->
        <div v-if="!isEditing" class="form-section">
          <h4>快速添加预设</h4>
          <div class="preset-list">
            <button
              v-for="preset in presets"
              :key="preset.id"
              class="preset-btn"
              @click="applyPreset(preset)"
            >
              <span class="preset-name">{{ preset.name }}</span>
              <span class="preset-desc">{{ preset.description }}</span>
            </button>
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
import {
  createMCPServer,
  updateMCPServer,
  getBuiltinServers,
  type MCPServer,
  type CreateMCPServerRequest
} from '@/api/mcp'

// ---- Props ----
const props = defineProps<{
  server: MCPServer | null
}>()

// ---- Emits ----
const emit = defineEmits<{
  close: []
  save: []
}>()

// ---- State ----
const isEditing = computed(() => props.server !== null)
const saving = ref(false)

const transports = [
  { value: 'stdio', label: 'Stdio', desc: '通过标准输入输出通信（推荐）' },
  { value: 'sse', label: 'SSE', desc: '通过服务器推送事件通信' }
]

const presets = getBuiltinServers()

const form = ref<CreateMCPServerRequest>({
  id: props.server?.id || '',
  name: props.server?.name || '',
  description: props.server?.description || '',
  command: props.server?.command || '',
  args: props.server?.args || [],
  env: props.server?.env || [],
  transport: props.server?.transport || 'stdio',
  url: props.server?.url || ''
})

const argsText = ref(form.value.args?.join('\n') || '')
const envText = ref(form.value.env?.join('\n') || '')

// ---- Methods ----
function applyPreset(preset: CreateMCPServerRequest) {
  form.value = { ...preset }
  argsText.value = preset.args?.join('\n') || ''
  envText.value = ''
}

async function save() {
  if (!form.value.id || !form.value.name) {
    alert('请填写 ID 和名称')
    return
  }

  if (form.value.transport === 'stdio' && !form.value.command) {
    alert('请填写命令')
    return
  }

  if (form.value.transport === 'sse' && !form.value.url) {
    alert('请填写 URL')
    return
  }

  // Parse args and env from text
  form.value.args = argsText.value.split('\n').filter(s => s.trim())
  form.value.env = envText.value.split('\n').filter(s => s.trim())

  saving.value = true
  try {
    if (isEditing.value) {
      await updateMCPServer(props.server!.id, {
        name: form.value.name,
        description: form.value.description,
        command: form.value.command,
        args: form.value.args,
        env: form.value.env,
        transport: form.value.transport,
        url: form.value.url
      })
    } else {
      await createMCPServer(form.value)
    }
    emit('save')
  } catch (err) {
    console.error('Failed to save MCP server:', err)
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

.transport-options {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.transport-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.transport-option:hover {
  background: var(--bg-overlay);
}

.transport-option.active {
  border-color: var(--accent);
  background: var(--accent-dim);
}

.transport-option input {
  width: 16px;
  height: 16px;
}

.option-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  min-width: 60px;
}

.option-desc {
  font-size: 12px;
  color: var(--text-tertiary);
}

.preset-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}

.preset-btn {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-input);
  cursor: pointer;
  transition: all 0.15s;
  text-align: left;
}

.preset-btn:hover {
  border-color: var(--accent);
  background: var(--accent-dim);
}

.preset-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.preset-desc {
  font-size: 11px;
  color: var(--text-tertiary);
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
