<template>
  <div class="edit-overlay" @click.self="$emit('close')">
    <div class="edit-modal">
      <!-- 头部 -->
      <div class="edit-header">
        <h3 class="edit-title">编辑数字员工</h3>
        <button class="icon-close" @click="$emit('close')">
          <XIcon :size="16" />
        </button>
      </div>

      <!-- Tab 导航 -->
      <div class="edit-tabs">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="edit-tab"
          :class="{ active: activeTab === tab.key }"
          @click="activeTab = tab.key"
        >{{ tab.label }}</button>
      </div>

      <!-- Tab 内容 -->
      <div class="edit-body">
        <!-- 基础设置 -->
        <div v-if="activeTab === 'basic'" class="tab-content">
          <div class="form-row-2">
            <div class="form-field">
              <label>名称 <span class="required">*</span></label>
              <input v-model="form.name" maxlength="50" class="form-input" />
            </div>
            <div class="form-field">
              <label>图标</label>
              <div class="emoji-row">
                <button
                  v-for="e in emojiOptions"
                  :key="e"
                  class="emoji-btn"
                  :class="{ selected: form.emoji === e }"
                  @click="form.emoji = e"
                >{{ e }}</button>
              </div>
            </div>
          </div>

          <div class="form-field">
            <label>描述</label>
            <input v-model="form.description" maxlength="120" placeholder="简短描述此数字员工的用途" class="form-input" />
          </div>

          <div class="form-field">
            <label>系统提示词</label>
            <textarea v-model="form.system_prompt" rows="5" placeholder="留空继承全局配置" class="form-textarea" />
          </div>

          <div class="form-section">
            <div class="section-title">模型配置</div>
            <div v-if="providers.length === 0" class="empty-hint">暂无可用模型</div>
            <div v-else class="provider-list">
              <div
                v-for="p in providers"
                :key="p.id"
                class="provider-item"
                :class="{ selected: isProviderSelected(p.id) }"
                @click="toggleProvider(p.id)"
              >
                <div class="provider-check">
                  <span v-if="isProviderSelected(p.id)" class="priority-badge">{{ getProviderIndex(p.id) + 1 }}</span>
                  <div v-else class="check-empty" />
                </div>
                <div class="provider-info">
                  <span class="provider-name">{{ p.name }}</span>
                  <span class="provider-id">{{ p.id }}</span>
                </div>
              </div>
            </div>
          </div>

          <div class="form-section">
            <div class="section-title">技能</div>
            <div v-if="skills.length === 0" class="empty-hint">暂无可用技能</div>
            <div v-else class="skills-grid">
              <label
                v-for="skill in skills"
                :key="skill.name"
                class="skill-item"
                :class="{ selected: isSkillSelected(skill.name) }"
              >
                <input type="checkbox" class="skill-check" :checked="isSkillSelected(skill.name)" @change="toggleSkill(skill.name)" />
                <span class="skill-name">{{ skill.display_name || skill.name }}</span>
              </label>
            </div>
          </div>

          <div class="form-section">
            <div class="section-title">MCP 服务器</div>
            <div v-if="mcpServers.length === 0" class="empty-hint">暂无已安装的 MCP 服务器</div>
            <div v-else class="mcp-list">
              <label
                v-for="srv in mcpServers"
                :key="srv.id"
                class="mcp-item"
                :class="{ selected: isMCPSelected(srv.id) }"
              >
                <input type="checkbox" class="skill-check" :checked="isMCPSelected(srv.id)" @change="toggleMCP(srv.id)" />
                <div class="mcp-info">
                  <span class="mcp-name">{{ srv.name }}</span>
                  <span class="mcp-cmd">{{ srv.command }}</span>
                </div>
              </label>
            </div>
          </div>

          <div class="form-section">
            <div class="section-title">知识库</div>
            <div v-if="knowledgeBases.length === 0" class="empty-hint">暂无知识库</div>
            <div v-else class="mcp-list">
              <label
                v-for="kb in knowledgeBases"
                :key="kb.id"
                class="mcp-item"
                :class="{ selected: isKBSelected(kb.id) }"
              >
                <input type="checkbox" class="skill-check" :checked="isKBSelected(kb.id)" @change="toggleKB(kb.id)" />
                <div class="mcp-info">
                  <span class="mcp-name">{{ kb.name }}</span>
                  <span class="mcp-cmd">{{ kb.description || kb.mode }}</span>
                </div>
              </label>
            </div>
          </div>

          <div class="form-section">
            <div class="section-title">频道集成</div>
            <div class="channels-list">
              <div v-for="ch in channelConfigs" :key="ch.type" class="channel-item">
                <div class="channel-header clickable" @click="ch.enabled = !ch.enabled">
                  <span class="channel-toggle" :class="{ on: ch.enabled }">{{ ch.enabled ? '●' : '○' }}</span>
                  <span class="channel-label">{{ getChannelLabel(ch.type) }}</span>
                  <span class="channel-toggle-text">{{ ch.enabled ? '已启用' : '未启用' }}</span>
                </div>
                <div v-if="ch.enabled" class="channel-fields">
                  <div v-for="field in getChannelFields(ch.type)" :key="field.key" class="channel-field">
                    <label>{{ field.label }}</label>
                    <input
                      v-model="ch.config[field.key]"
                      :type="field.secret ? 'password' : 'text'"
                      :placeholder="field.placeholder || field.key"
                        class="form-input"
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 高级 -->
        <div v-if="activeTab === 'advanced'" class="tab-content">
          <div class="form-row-2">
            <div class="form-field">
              <label>最大步数</label>
              <input v-model.number="form.max_steps" type="number" min="1" max="100" class="form-input" />
              <span class="field-hint">工具调用最大轮数（1-100）</span>
            </div>
            <div class="form-field">
              <label>Temperature</label>
              <input v-model.number="form.llm_temperature" type="number" min="0" max="2" step="0.1" class="form-input" />
              <span class="field-hint">生成随机性（0-2）</span>
            </div>
          </div>
          <div class="form-field">
            <label>记忆命名空间</label>
            <input v-model="form.memory_namespace" placeholder="默认 default" class="form-input" />
            <span class="field-hint">隔离此数字员工的记忆空间</span>
          </div>
        </div>
      </div>

      <!-- 底部 -->
      <div class="edit-footer">
        <button
          class="btn-danger"
          @click="$emit('delete', agent!.id)"
        >删除</button>
        <button
          class="btn-secondary"
          :disabled="saving"
          @click="setDefault"
        >设为默认</button>
        <div class="footer-spacer" />
        <button class="btn-secondary" @click="$emit('close')">取消</button>
        <button class="btn-primary" :disabled="saving" @click="handleSave">
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { XIcon } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { updateAgent, updateAgentConfig, setDefaultAgent, type Agent, type AgentChannelConfig } from '@/api/agents'
import { knowledgeApi, type KnowledgeBase } from '@/api/knowledge'

interface Provider { id: string; name: string }
interface Skill { name: string; display_name?: string; description?: string }
interface MCPServer { id: string; name: string; command: string }

const props = defineProps<{
  agent: Agent
  providers: Provider[]
  skills: Skill[]
  mcpServers: MCPServer[]
}>()

const emit = defineEmits<{
  close: []
  saved: []
  delete: [id: string]
}>()

// ---- Channel definitions ----
const CHANNEL_DEFS = [
  {
    type: 'feishu', label: '飞书',
    fields: [
      { key: 'app_id', label: 'App ID' },
      { key: 'app_secret', label: 'App Secret', secret: true },
    ]
  },
  {
    type: 'dingtalk', label: '钉钉',
    fields: [
      { key: 'client_id', label: 'Client ID' },
      { key: 'client_secret', label: 'Client Secret', secret: true },
    ]
  },
  {
    type: 'webhook', label: 'Webhook',
    fields: [
      { key: 'url', label: 'Webhook URL', placeholder: 'https://' },
    ]
  },
  {
    type: 'email', label: '邮件',
    fields: [
      { key: 'host', label: 'SMTP Host' },
      { key: 'port', label: '端口', placeholder: '587' },
      { key: 'username', label: '用户名' },
      { key: 'password', label: '密码', secret: true },
      { key: 'from', label: '发件人' },
      { key: 'ssl', label: 'SSL', placeholder: 'true / false' },
    ]
  },
]

function getChannelLabel(type: string) { return CHANNEL_DEFS.find(d => d.type === type)?.label || type }
function getChannelFields(type: string) { return CHANNEL_DEFS.find(d => d.type === type)?.fields || [] }

// ---- Tabs ----
const tabs = [
  { key: 'basic', label: '基础设置' },
  { key: 'advanced', label: '高级' },
]
const activeTab = ref('basic')
const saving = ref(false)

const emojiOptions = ['🤖', '🧑', '📊', '💡', '🔧', '🎯', '📝', '🔍', '⚡', '🛠️', '🌐', '📦', '🎨', '📱', '🔐']

// ---- Remote data ----
const knowledgeBases = ref<KnowledgeBase[]>([])

// ---- Channel state ----
const channelConfigs = ref<(AgentChannelConfig & { config: Record<string, string> })[]>(
  CHANNEL_DEFS.map(d => ({
    type: d.type,
    enabled: false,
    config: Object.fromEntries(d.fields.map(f => [f.key, ''])),
  }))
)

// ---- Form ----
const form = ref({
  name: '',
  emoji: '🤖',
  description: '',
  system_prompt: '',
  provider_ids: [] as string[],
  skills: [] as string[],
  mcp_servers: [] as string[],
  knowledge_bases: [] as string[],
  max_steps: 20,
  llm_temperature: 0.7,
  memory_namespace: 'default',
})

// Initialize form from agent
watch(() => props.agent, (agent) => {
  if (!agent) return
  form.value.name = agent.name
  form.value.emoji = agent.config?.emoji || agent.avatar || '🤖'
  form.value.description = agent.config?.description || agent.description || ''
  form.value.system_prompt = agent.config?.system_prompt || ''
  form.value.provider_ids = [...(agent.config?.provider_ids || [])]
  form.value.skills = [...(agent.config?.skills || [])]
  form.value.mcp_servers = [...(agent.config?.mcp_servers || [])]
  form.value.knowledge_bases = [...(agent.config?.knowledge_bases || [])]
  form.value.max_steps = agent.config?.max_steps || 20
  form.value.llm_temperature = agent.config?.llm?.temperature || 0.7
  form.value.memory_namespace = agent.config?.memory?.namespace || 'default'

  // Initialize channel configs from saved data
  const savedChannels = agent.config?.channels || []
  channelConfigs.value = CHANNEL_DEFS.map(d => {
    const saved = savedChannels.find((c: AgentChannelConfig) => c.type === d.type)
    return {
      type: d.type,
      enabled: saved?.enabled || false,
      config: {
        ...Object.fromEntries(d.fields.map(f => [f.key, ''])),
        ...(saved?.config || {}),
      }
    }
  })
}, { immediate: true })

// ---- Helpers ----
function isProviderSelected(id: string) { return form.value.provider_ids.includes(id) }
function getProviderIndex(id: string) { return form.value.provider_ids.indexOf(id) }
function toggleProvider(id: string) {
  const idx = form.value.provider_ids.indexOf(id)
  if (idx >= 0) form.value.provider_ids.splice(idx, 1)
  else form.value.provider_ids.push(id)
}

function isSkillSelected(name: string) { return form.value.skills.includes(name) }
function toggleSkill(name: string) {
  const idx = form.value.skills.indexOf(name)
  if (idx >= 0) form.value.skills.splice(idx, 1)
  else form.value.skills.push(name)
}

function isMCPSelected(id: string) { return form.value.mcp_servers.includes(id) }
function toggleMCP(id: string) {
  const idx = form.value.mcp_servers.indexOf(id)
  if (idx >= 0) form.value.mcp_servers.splice(idx, 1)
  else form.value.mcp_servers.push(id)
}

function isKBSelected(id: string) { return form.value.knowledge_bases.includes(id) }
function toggleKB(id: string) {
  const idx = form.value.knowledge_bases.indexOf(id)
  if (idx >= 0) form.value.knowledge_bases.splice(idx, 1)
  else form.value.knowledge_bases.push(id)
}

// ---- Save ----
async function handleSave() {
  if (!form.value.name.trim()) { toast.error('请填写名称'); return }
  saving.value = true
  try {
    await updateAgent(props.agent.id, {
      name: form.value.name,
      description: form.value.description,
      avatar: form.value.emoji,
    })
    const config = { ...props.agent.config! }
    config.emoji = form.value.emoji
    config.description = form.value.description
    config.system_prompt = form.value.system_prompt
    config.provider_ids = form.value.provider_ids
    config.skills = form.value.skills
    config.mcp_servers = form.value.mcp_servers
    config.knowledge_bases = form.value.knowledge_bases
    config.channels = channelConfigs.value
      .filter(c => c.enabled)
      .map(c => ({ type: c.type, enabled: true, config: { ...c.config } }))
    config.max_steps = form.value.max_steps
    if (config.llm) config.llm.temperature = form.value.llm_temperature
    if (config.memory) config.memory.namespace = form.value.memory_namespace
    await updateAgentConfig(props.agent.id, config)
    toast.success('已保存')
    emit('saved')
  } catch (err: any) {
    toast.error(err?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function setDefault() {
  try {
    await setDefaultAgent(props.agent.id)
    toast.success('已设为默认数字员工')
    emit('saved')
  } catch {
    toast.error('操作失败')
  }
}

onMounted(async () => {
  try {
    knowledgeBases.value = await knowledgeApi.listBases()
  } catch {}
})
</script>

<style scoped>
.edit-overlay {
  position: fixed; inset: 0;
  background: rgba(0, 0, 0, 0.65);
  display: flex; align-items: center; justify-content: center;
  z-index: 1000;
}

.edit-modal {
  width: 620px; max-height: 90vh;
  background: var(--bg-elevated); border: 1px solid var(--border);
  border-radius: 12px; display: flex; flex-direction: column; overflow: hidden;
}

.edit-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 16px 20px; border-bottom: 1px solid var(--border-subtle); flex-shrink: 0;
}
.edit-title { font-size: 15px; font-weight: 600; color: var(--text-primary); margin: 0; }

.icon-close {
  background: transparent; border: none; color: var(--text-tertiary);
  cursor: pointer; display: flex; align-items: center; padding: 4px; border-radius: 4px;
}
.icon-close:hover { color: var(--text-primary); background: var(--bg-overlay); }


.edit-tabs {
  display: flex; border-bottom: 1px solid var(--border-subtle);
  padding: 0 20px; flex-shrink: 0;
}
.edit-tab {
  padding: 10px 14px; font-size: 13px; font-weight: 500;
  color: var(--text-secondary); background: transparent; border: none;
  border-bottom: 2px solid transparent; cursor: pointer; transition: all 0.15s; margin-bottom: -1px;
}
.edit-tab:hover { color: var(--text-primary); }
.edit-tab.active { color: var(--accent); border-bottom-color: var(--accent); }

.edit-body { flex: 1; overflow-y: auto; padding: 20px; }
.tab-content { display: flex; flex-direction: column; gap: 16px; }

.form-row-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.form-field { display: flex; flex-direction: column; gap: 6px; }
.form-field label { font-size: 12px; font-weight: 500; color: var(--text-secondary); }
.required { color: var(--red); }

.form-input {
  padding: 8px 12px; background: var(--bg-app);
  border: 1px solid var(--border); border-radius: 6px;
  color: var(--text-primary); font-size: 13px; outline: none;
}
.form-input:focus { border-color: var(--accent); }
.form-input:disabled { opacity: 0.5; cursor: not-allowed; }

.form-textarea {
  padding: 8px 12px; background: var(--bg-app);
  border: 1px solid var(--border); border-radius: 6px;
  color: var(--text-primary); font-size: 12px; outline: none; resize: vertical;
  font-family: "SF Mono", Menlo, monospace; line-height: 1.6;
}
.form-textarea:focus { border-color: var(--accent); }
.field-hint { font-size: 11px; color: var(--text-tertiary); }

.emoji-row { display: flex; flex-wrap: wrap; gap: 4px; }
.emoji-btn {
  width: 30px; height: 30px; font-size: 16px;
  background: var(--bg-app); border: 1px solid var(--border);
  border-radius: 5px; cursor: pointer; transition: all 0.12s;
  display: flex; align-items: center; justify-content: center;
}
.emoji-btn:hover { background: var(--bg-overlay); }
.emoji-btn.selected { border-color: var(--accent); background: var(--accent-dim); }

.form-section { display: flex; flex-direction: column; gap: 8px; }
.section-title {
  font-size: 11px; font-weight: 600; color: var(--text-tertiary);
  text-transform: uppercase; letter-spacing: 0.05em;
}

.provider-list { display: flex; flex-direction: column; gap: 4px; }
.provider-item {
  display: flex; align-items: center; gap: 10px;
  padding: 7px 10px; background: var(--bg-app);
  border: 1px solid var(--border); border-radius: 6px; cursor: pointer; transition: all 0.12s;
}
.provider-item:hover { border-color: var(--border-hover); }
.provider-item.selected { border-color: var(--accent); background: var(--accent-dim); }
.provider-check { width: 20px; height: 20px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; }
.priority-badge {
  width: 18px; height: 18px; border-radius: 50%;
  background: var(--accent); color: white;
  font-size: 10px; font-weight: 700;
  display: flex; align-items: center; justify-content: center;
}
.check-empty { width: 14px; height: 14px; border-radius: 50%; border: 2px solid var(--border); }
.provider-info { flex: 1; display: flex; flex-direction: column; gap: 1px; }
.provider-name { font-size: 12px; color: var(--text-primary); font-weight: 500; }
.provider-id { font-size: 10px; color: var(--text-tertiary); font-family: monospace; }

.skills-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 5px; }
.skill-item {
  display: flex; align-items: center; gap: 7px;
  padding: 7px 9px; background: var(--bg-app);
  border: 1px solid var(--border); border-radius: 5px; cursor: pointer; transition: all 0.12s;
}
.skill-item:hover { border-color: var(--border-hover); }
.skill-item.selected { border-color: var(--accent); background: var(--accent-dim); }
.skill-check { cursor: pointer; accent-color: var(--accent); flex-shrink: 0; }
.skill-name { font-size: 12px; color: var(--text-primary); }

.mcp-list { display: flex; flex-direction: column; gap: 4px; }
.mcp-item {
  display: flex; align-items: center; gap: 8px;
  padding: 7px 10px; background: var(--bg-app);
  border: 1px solid var(--border); border-radius: 6px; cursor: pointer; transition: all 0.12s;
}
.mcp-item:hover { border-color: var(--border-hover); }
.mcp-item.selected { border-color: var(--accent); background: var(--accent-dim); }
.mcp-info { flex: 1; display: flex; flex-direction: column; gap: 1px; }
.mcp-name { font-size: 12px; color: var(--text-primary); font-weight: 500; }
.mcp-cmd { font-size: 10px; color: var(--text-tertiary); font-family: monospace; }

.empty-hint { font-size: 12px; color: var(--text-tertiary); padding: 8px 0; }

/* Channels */
.channels-list { display: flex; flex-direction: column; gap: 6px; }
.channel-item { border: 1px solid var(--border); border-radius: 7px; overflow: hidden; }
.channel-header {
  display: flex; align-items: center; gap: 8px;
  padding: 9px 12px; background: var(--bg-panel);
}
.channel-header.clickable { cursor: pointer; transition: background 0.12s; }
.channel-header.clickable:hover { background: var(--bg-overlay); }
.channel-toggle { font-size: 14px; color: var(--text-tertiary); }
.channel-toggle.on { color: var(--accent); }
.channel-label { flex: 1; font-size: 13px; font-weight: 500; color: var(--text-primary); }
.channel-toggle-text { font-size: 11px; color: var(--text-tertiary); }
.channel-fields {
  padding: 10px 12px; background: var(--bg-app);
  border-top: 1px solid var(--border-subtle);
  display: grid; grid-template-columns: 1fr 1fr; gap: 8px;
}
.channel-field { display: flex; flex-direction: column; gap: 4px; }
.channel-field label { font-size: 11px; color: var(--text-secondary); font-weight: 500; }

/* Footer */
.edit-footer {
  display: flex; align-items: center; gap: 8px;
  padding: 14px 20px; border-top: 1px solid var(--border-subtle); flex-shrink: 0;
}
.footer-spacer { flex: 1; }

.btn-primary {
  padding: 7px 16px; background: var(--accent); border: none;
  border-radius: 6px; color: #fff; font-size: 13px; font-weight: 500;
  cursor: pointer; transition: opacity 0.15s;
}
.btn-primary:hover:not(:disabled) { opacity: 0.9; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-secondary {
  padding: 7px 14px; background: var(--bg-overlay);
  border: 1px solid var(--border); border-radius: 6px;
  color: var(--text-secondary); font-size: 13px; cursor: pointer; transition: background 0.15s;
}
.btn-secondary:hover:not(:disabled) { background: var(--bg-elevated); }
.btn-secondary:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-danger {
  padding: 7px 14px; background: transparent;
  border: 1px solid var(--border); border-radius: 6px;
  color: var(--red); font-size: 13px; cursor: pointer; transition: all 0.15s;
}
.btn-danger:hover:not(:disabled) { background: var(--red-dim); border-color: var(--red); }
.btn-danger:disabled { opacity: 0.4; cursor: not-allowed; }
</style>
