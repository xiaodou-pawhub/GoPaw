<template>
  <div class="wizard-overlay" @click.self="$emit('close')">
    <div class="wizard-modal">
      <!-- 头部 -->
      <div class="wizard-header">
        <h3 class="wizard-title">新建数字员工</h3>
        <button class="icon-close" @click="$emit('close')">
          <XIcon :size="16" />
        </button>
      </div>

      <!-- 步骤条 -->
      <div class="steps-bar">
        <div
          v-for="(step, i) in steps"
          :key="i"
          class="step-item"
          :class="{ active: currentStep === i, done: currentStep > i }"
        >
          <div class="step-circle">
            <CheckIcon v-if="currentStep > i" :size="12" />
            <span v-else>{{ i + 1 }}</span>
          </div>
          <span class="step-label">{{ step }}</span>
        </div>
        <div class="steps-line" />
      </div>

      <!-- 步骤内容 -->
      <div class="wizard-body">

        <!-- Step 0: 基础信息 -->
        <div v-if="currentStep === 0" class="step-content">
          <div class="form-field">
            <label>名称 <span class="required">*</span></label>
            <input v-model="form.name" maxlength="50" placeholder="给你的数字员工起个名字" class="form-input" autofocus />
          </div>
          <div class="form-field">
            <label>图标</label>
            <div class="emoji-grid">
              <button
                v-for="e in emojiOptions"
                :key="e"
                class="emoji-btn"
                :class="{ selected: form.emoji === e }"
                @click="form.emoji = e"
              >{{ e }}</button>
            </div>
          </div>
          <div class="form-field">
            <label>描述</label>
            <input v-model="form.description" maxlength="120" placeholder="简短描述此数字员工的用途" class="form-input" />
          </div>
          <div class="form-field">
            <label>系统提示词</label>
            <textarea v-model="form.system_prompt" rows="6" placeholder="定义数字员工的行为、角色和能力范围（留空继承全局配置）" class="form-textarea" />
          </div>
        </div>

        <!-- Step 1: 大模型配置 -->
        <div v-if="currentStep === 1" class="step-content">
          <!-- 无模型警告 -->
          <div v-if="providers.length === 0" class="warn-box">
            <AlertTriangleIcon :size="16" class="warn-icon" />
            <div class="warn-content">
              <span class="warn-title">尚未配置大模型</span>
              <span class="warn-desc">数字员工将无法正常工作，请先前往
                <router-link to="/models" class="warn-link" @click="$emit('close')">模型</router-link>
                页面配置大模型。
              </span>
            </div>
          </div>

          <div class="form-section" v-else>
            <div class="section-title">选择大模型（可多选，按优先级排序）</div>
            <div class="provider-list">
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
            <p class="hint-text">未选择模型时，将使用系统全局默认模型</p>
          </div>
        </div>

        <!-- Step 2: 能力集成 -->
        <div v-if="currentStep === 2" class="step-content">
          <!-- 技能 -->
          <div class="cap-section">
            <button class="cap-header" @click="openSection.skills = !openSection.skills">
              <span class="cap-title">🔧 技能</span>
              <span class="cap-badge">{{ form.skills.length }} / {{ skills.length }}</span>
              <ChevronDownIcon :size="14" class="cap-chevron" :class="{ open: openSection.skills }" />
            </button>
            <div v-if="openSection.skills" class="cap-body">
              <div v-if="skills.length === 0" class="empty-hint">暂无可用技能</div>
              <div v-else class="skills-grid">
                <label
                  v-for="skill in skills"
                  :key="skill.name"
                  class="skill-item"
                  :class="{ selected: isSkillSelected(skill.name) }"
                >
                  <input type="checkbox" class="skill-check" :checked="isSkillSelected(skill.name)" @change="toggleSkill(skill.name)" />
                  <div class="skill-info">
                    <span class="skill-name">{{ skill.display_name || skill.name }}</span>
                    <span class="skill-desc">{{ skill.description }}</span>
                  </div>
                </label>
              </div>
            </div>
          </div>

          <!-- MCP 服务器 -->
          <div class="cap-section">
            <button class="cap-header" @click="openSection.mcp = !openSection.mcp">
              <span class="cap-title">🔌 MCP 服务器</span>
              <span class="cap-badge">{{ form.mcp_servers.length }} / {{ mcpServers.length }}</span>
              <ChevronDownIcon :size="14" class="cap-chevron" :class="{ open: openSection.mcp }" />
            </button>
            <div v-if="openSection.mcp" class="cap-body">
              <div v-if="mcpServers.length === 0" class="empty-hint">暂无已安装的 MCP 服务器</div>
              <div v-else class="check-list">
                <label v-for="srv in mcpServers" :key="srv.id" class="check-item" :class="{ selected: form.mcp_servers.includes(srv.id) }">
                  <input type="checkbox" :checked="form.mcp_servers.includes(srv.id)" @change="toggleArr(form.mcp_servers, srv.id)" />
                  <div class="check-item-info">
                    <span class="check-item-name">{{ srv.name }}</span>
                    <span class="check-item-sub">{{ srv.command }} {{ srv.args?.join(' ') }}</span>
                  </div>
                </label>
              </div>
            </div>
          </div>

          <!-- 知识库 -->
          <div class="cap-section">
            <button class="cap-header" @click="openSection.knowledge = !openSection.knowledge">
              <span class="cap-title">📚 知识库</span>
              <span class="cap-badge">{{ form.knowledge_bases.length }} / {{ knowledgeBases.length }}</span>
              <ChevronDownIcon :size="14" class="cap-chevron" :class="{ open: openSection.knowledge }" />
            </button>
            <div v-if="openSection.knowledge" class="cap-body">
              <div v-if="knowledgeBases.length === 0" class="empty-hint">暂无知识库</div>
              <div v-else class="check-list">
                <label v-for="kb in knowledgeBases" :key="kb.id" class="check-item" :class="{ selected: form.knowledge_bases.includes(kb.id) }">
                  <input type="checkbox" :checked="form.knowledge_bases.includes(kb.id)" @change="toggleArr(form.knowledge_bases, kb.id)" />
                  <div class="check-item-info">
                    <span class="check-item-name">{{ kb.name }}</span>
                    <span class="check-item-sub">{{ kb.description || kb.mode }}</span>
                  </div>
                </label>
              </div>
            </div>
          </div>

          <!-- 频道集成 -->
          <div class="cap-section">
            <button class="cap-header" @click="openSection.channels = !openSection.channels">
              <span class="cap-title">📡 频道集成</span>
              <span class="cap-badge">{{ enabledChannels }} 个已启用</span>
              <ChevronDownIcon :size="14" class="cap-chevron" :class="{ open: openSection.channels }" />
            </button>
            <div v-if="openSection.channels" class="cap-body channels-body">
              <div v-for="ch in channelConfigs" :key="ch.type" class="channel-item">
                <div class="channel-header" @click="ch.enabled = !ch.enabled">
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

        <!-- Step 3: 确认创建 -->
        <div v-if="currentStep === 3" class="step-content confirm-step">
          <div class="confirm-card">
            <div class="confirm-emoji">{{ form.emoji }}</div>
            <h3 class="confirm-name">{{ form.name }}</h3>
            <p class="confirm-desc">{{ form.description || '暂无描述' }}</p>
            <div class="confirm-meta">
              <div class="meta-row">
                <span class="meta-key">大模型</span>
                <span class="meta-val">{{ form.provider_ids.length ? form.provider_ids.join(', ') : '全局默认' }}</span>
              </div>
              <div class="meta-row">
                <span class="meta-key">技能</span>
                <span class="meta-val">{{ form.skills.length ? `${form.skills.length} 个` : '无' }}</span>
              </div>
              <div class="meta-row">
                <span class="meta-key">MCP</span>
                <span class="meta-val">{{ form.mcp_servers.length ? `${form.mcp_servers.length} 个` : '无' }}</span>
              </div>
              <div class="meta-row">
                <span class="meta-key">知识库</span>
                <span class="meta-val">{{ form.knowledge_bases.length ? `${form.knowledge_bases.length} 个` : '无' }}</span>
              </div>
              <div class="meta-row">
                <span class="meta-key">频道</span>
                <span class="meta-val">{{ enabledChannels ? `${enabledChannels} 个已配置` : '无' }}</span>
              </div>
              <div class="meta-row">
                <span class="meta-key">可见性</span>
                <select v-model="form.visibility" class="visibility-select">
                  <option value="private">私有（仅创建者可见）</option>
                  <option value="shared">共享（需授权）</option>
                  <option value="global">全局（所有用户可见）</option>
                </select>
              </div>
            </div>
          </div>
          <p class="confirm-hint">点击"创建数字员工"完成创建</p>
        </div>

        <!-- Step 4: 完成 -->
        <div v-if="currentStep === 4" class="step-content finish-step">
          <div class="finish-icon">✅</div>
          <h3 class="finish-title">数字员工创建成功！</h3>
          <p class="finish-desc">你可以在数字员工列表中找到 <strong>{{ form.name }}</strong>，点击"开始对话"与它互动。</p>
        </div>
      </div>

      <!-- 底部按钮 -->
      <div class="wizard-footer">
        <button v-if="currentStep > 0 && currentStep < 4" class="btn-secondary" @click="currentStep--">
          上一步
        </button>
        <div class="footer-spacer" />
        <button
          v-if="currentStep < 3"
          class="btn-primary"
          :disabled="!canProceed"
          @click="currentStep++"
        >
          下一步
        </button>
        <button
          v-else-if="currentStep === 3"
          class="btn-primary"
          :disabled="creating"
          @click="handleCreate"
        >
          {{ creating ? '创建中...' : '创建数字员工' }}
        </button>
        <button
          v-else-if="currentStep === 4"
          class="btn-primary"
          @click="$emit('created')"
        >
          完成
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { XIcon, CheckIcon, ChevronDownIcon, AlertTriangleIcon } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { createAgent, getDefaultConfig, type AgentChannelConfig } from '@/api/agents'
import { listMCPServers, type MCPServer } from '@/api/mcp'
import { knowledgeApi, type KnowledgeBase } from '@/api/knowledge'

interface Provider { id: string; name: string }
interface Skill { name: string; display_name?: string; description?: string }

const props = defineProps<{ providers: Provider[]; skills: Skill[] }>()
const emit = defineEmits<{ close: []; created: [] }>()

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
      { key: 'url', label: 'Webhook URL', placeholder: 'https://...' },
    ]
  },
  {
    type: 'email', label: '邮件',
    fields: [
      { key: 'host', label: 'SMTP Host' },
      { key: 'port', label: '端口', placeholder: '587' },
      { key: 'username', label: '用户名' },
      { key: 'password', label: '密码', secret: true },
      { key: 'from', label: '发件人', placeholder: 'noreply@example.com' },
      { key: 'ssl', label: 'SSL', placeholder: 'true / false' },
    ]
  },
]

function getChannelLabel(type: string) {
  return CHANNEL_DEFS.find(d => d.type === type)?.label || type
}

function getChannelFields(type: string) {
  return CHANNEL_DEFS.find(d => d.type === type)?.fields || []
}

// ---- Steps ----
const steps = ['基础信息', '大模型', '能力集成', '确认创建', '完成']
const currentStep = ref(0)
const creating = ref(false)

// ---- Remote data ----
const mcpServers = ref<MCPServer[]>([])
const knowledgeBases = ref<KnowledgeBase[]>([])

// ---- Collapsible sections ----
const openSection = ref({ skills: true, mcp: false, knowledge: false, channels: false })

// ---- Emoji ----
const emojiOptions = ['🤖', '🧑', '📊', '💡', '🔧', '🎯', '📝', '🔍', '⚡', '🛠️', '🌐', '📦', '🎨', '📱', '🔐']

// ---- Channel config state ----
const channelConfigs = ref<(AgentChannelConfig & { config: Record<string, string> })[]>(
  CHANNEL_DEFS.map(d => ({
    type: d.type,
    enabled: false,
    config: Object.fromEntries(d.fields.map(f => [f.key, ''])),
  }))
)

const enabledChannels = computed(() => channelConfigs.value.filter(c => c.enabled).length)

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
  visibility: 'private' as 'global' | 'private' | 'shared',
})

// ---- canProceed ----
const canProceed = computed(() => {
  if (currentStep.value === 0) return form.value.name.trim().length > 0
  if (currentStep.value === 1) return props.providers.length > 0  // block if no providers
  return true
})

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

function toggleArr(arr: string[], val: string) {
  const idx = arr.indexOf(val)
  if (idx >= 0) arr.splice(idx, 1)
  else arr.push(val)
}

// ---- Create ----
async function handleCreate() {
  if (!form.value.name.trim()) { toast.error('请填写名称'); return }
  creating.value = true
  try {
    const config = getDefaultConfig()
    config.emoji = form.value.emoji
    config.description = form.value.description
    config.system_prompt = form.value.system_prompt || config.system_prompt
    config.provider_ids = form.value.provider_ids
    config.skills = form.value.skills
    config.mcp_servers = form.value.mcp_servers
    config.knowledge_bases = form.value.knowledge_bases
    config.channels = channelConfigs.value
      .filter(c => c.enabled)
      .map(c => ({ type: c.type, enabled: true, config: { ...c.config } }))

    const id = form.value.name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') + '-' + Date.now().toString(36)
    await createAgent({
      id,
      name: form.value.name,
      description: form.value.description,
      avatar: form.value.emoji,
      config,
      visibility: form.value.visibility,
    })
    toast.success('数字员工创建成功')
    currentStep.value = 4
  } catch (err: any) {
    toast.error(err?.message || '创建失败')
  } finally {
    creating.value = false
  }
}

// ---- Load ----
onMounted(async () => {
  const [srvRes, kbRes] = await Promise.allSettled([listMCPServers(), knowledgeApi.listBases()])
  if (srvRes.status === 'fulfilled') mcpServers.value = srvRes.value.servers
  if (kbRes.status === 'fulfilled') knowledgeBases.value = kbRes.value
})
</script>

<style scoped>
.wizard-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.wizard-modal {
  width: 600px;
  max-height: 88vh;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.wizard-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.wizard-title { font-size: 15px; font-weight: 600; color: var(--text-primary); margin: 0; }

.icon-close {
  background: transparent; border: none; color: var(--text-tertiary);
  cursor: pointer; display: flex; align-items: center; padding: 4px; border-radius: 4px;
}
.icon-close:hover { color: var(--text-primary); background: var(--bg-overlay); }

/* Steps Bar */
.steps-bar {
  display: flex;
  align-items: center;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border-subtle);
  position: relative;
  flex-shrink: 0;
}

.steps-line {
  position: absolute;
  top: 50%;
  left: 44px;
  right: 44px;
  height: 1px;
  background: var(--border);
  z-index: 0;
  margin-top: 7px;
}

.step-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  flex: 1;
  position: relative;
  z-index: 1;
}

.step-circle {
  width: 26px; height: 26px; border-radius: 50%;
  border: 2px solid var(--border); background: var(--bg-app);
  display: flex; align-items: center; justify-content: center;
  font-size: 11px; font-weight: 600; color: var(--text-tertiary);
  transition: all 0.2s;
}
.step-item.active .step-circle { border-color: var(--accent); color: var(--accent); background: var(--accent-dim); }
.step-item.done .step-circle { border-color: var(--green); background: rgba(34,197,94,0.15); color: var(--green); }

.step-label { font-size: 10px; color: var(--text-tertiary); white-space: nowrap; }
.step-item.active .step-label { color: var(--accent); }
.step-item.done .step-label { color: var(--green); }

/* Body */
.wizard-body { flex: 1; overflow-y: auto; padding: 20px; }
.step-content { display: flex; flex-direction: column; gap: 16px; }

/* Form fields */
.form-field { display: flex; flex-direction: column; gap: 6px; }
.form-field label { font-size: 12px; font-weight: 500; color: var(--text-secondary); }
.required { color: var(--red); }

.form-input {
  padding: 8px 12px;
  background: var(--bg-app); border: 1px solid var(--border);
  border-radius: 6px; color: var(--text-primary);
  font-size: 13px; outline: none;
}
.form-input:focus { border-color: var(--accent); }

.form-textarea {
  padding: 8px 12px;
  background: var(--bg-app); border: 1px solid var(--border);
  border-radius: 6px; color: var(--text-primary);
  font-size: 12px; outline: none; resize: vertical;
  font-family: "SF Mono", Menlo, monospace; line-height: 1.6;
}
.form-textarea:focus { border-color: var(--accent); }
.form-textarea::placeholder { color: var(--text-disabled); }

/* Emoji */
.emoji-grid { display: flex; flex-wrap: wrap; gap: 6px; }
.emoji-btn {
  width: 36px; height: 36px; font-size: 18px;
  background: var(--bg-app); border: 1px solid var(--border);
  border-radius: 6px; cursor: pointer; transition: all 0.12s;
  display: flex; align-items: center; justify-content: center;
}
.emoji-btn:hover { background: var(--bg-overlay); }
.emoji-btn.selected { border-color: var(--accent); background: var(--accent-dim); }

/* Warning box */
.warn-box {
  display: flex; align-items: flex-start; gap: 10px;
  padding: 12px 14px;
  background: rgba(245,158,11,0.08); border: 1px solid rgba(245,158,11,0.3);
  border-radius: 8px;
}
.warn-icon { color: var(--yellow); flex-shrink: 0; margin-top: 1px; }
.warn-content { display: flex; flex-direction: column; gap: 4px; }
.warn-title { font-size: 13px; font-weight: 600; color: var(--yellow); }
.warn-desc { font-size: 12px; color: var(--text-secondary); line-height: 1.5; }
.warn-link { color: var(--accent); text-decoration: underline; cursor: pointer; }

/* Provider */
.form-section { display: flex; flex-direction: column; gap: 8px; }
.section-title { font-size: 12px; font-weight: 600; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.04em; }
.provider-list { display: flex; flex-direction: column; gap: 4px; }
.provider-item {
  display: flex; align-items: center; gap: 10px;
  padding: 8px 10px; background: var(--bg-app);
  border: 1px solid var(--border); border-radius: 6px; cursor: pointer; transition: all 0.12s;
}
.provider-item:hover { border-color: var(--border-hover); }
.provider-item.selected { border-color: var(--accent); background: var(--accent-dim); }
.provider-check { width: 22px; height: 22px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; }
.priority-badge {
  width: 20px; height: 20px; border-radius: 50%;
  background: var(--accent); color: white;
  font-size: 11px; font-weight: 700;
  display: flex; align-items: center; justify-content: center;
}
.check-empty { width: 16px; height: 16px; border-radius: 50%; border: 2px solid var(--border); }
.provider-info { flex: 1; display: flex; flex-direction: column; gap: 1px; }
.provider-name { font-size: 13px; color: var(--text-primary); font-weight: 500; }
.provider-id { font-size: 11px; color: var(--text-tertiary); font-family: monospace; }
.hint-text { font-size: 11px; color: var(--text-tertiary); }

/* Capability sections */
.cap-section {
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}
.cap-header {
  display: flex; align-items: center; gap: 8px;
  width: 100%; padding: 10px 14px;
  background: var(--bg-panel); border: none;
  cursor: pointer; text-align: left; transition: background 0.12s;
}
.cap-header:hover { background: var(--bg-overlay); }
.cap-title { font-size: 13px; font-weight: 500; color: var(--text-primary); flex: 1; }
.cap-badge {
  font-size: 11px; padding: 1px 7px;
  background: var(--bg-overlay); border-radius: 8px; color: var(--text-tertiary);
}
.cap-chevron { color: var(--text-tertiary); transition: transform 0.2s; }
.cap-chevron.open { transform: rotate(180deg); }
.cap-body { padding: 12px 14px; border-top: 1px solid var(--border-subtle); background: var(--bg-app); }

/* Skills grid */
.skills-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }
.skill-item {
  display: flex; align-items: flex-start; gap: 8px;
  padding: 8px 10px; background: var(--bg-panel);
  border: 1px solid var(--border); border-radius: 6px;
  cursor: pointer; transition: all 0.12s;
}
.skill-item:hover { border-color: var(--border-hover); }
.skill-item.selected { border-color: var(--accent); background: var(--accent-dim); }
.skill-check { margin-top: 2px; cursor: pointer; accent-color: var(--accent); }
.skill-info { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.skill-name { font-size: 12px; font-weight: 500; color: var(--text-primary); }
.skill-desc { font-size: 11px; color: var(--text-tertiary); }

/* Check list (MCP & Knowledge) */
.check-list { display: flex; flex-direction: column; gap: 4px; }
.check-item {
  display: flex; align-items: flex-start; gap: 8px;
  padding: 8px 10px; background: var(--bg-panel);
  border: 1px solid var(--border); border-radius: 6px;
  cursor: pointer; transition: all 0.12s;
}
.check-item:hover { border-color: var(--border-hover); }
.check-item.selected { border-color: var(--accent); background: var(--accent-dim); }
.check-item input { margin-top: 2px; cursor: pointer; accent-color: var(--accent); }
.check-item-info { flex: 1; display: flex; flex-direction: column; gap: 1px; }
.check-item-name { font-size: 12px; font-weight: 500; color: var(--text-primary); }
.check-item-sub { font-size: 11px; color: var(--text-tertiary); font-family: monospace; }

.empty-hint { font-size: 12px; color: var(--text-tertiary); padding: 4px 0; }

/* Channels */
.channels-body { display: flex; flex-direction: column; gap: 8px; }
.channel-item {
  border: 1px solid var(--border);
  border-radius: 7px; overflow: hidden;
}
.channel-header {
  display: flex; align-items: center; gap: 8px;
  padding: 9px 12px;
  background: var(--bg-panel); cursor: pointer;
  transition: background 0.12s;
}
.channel-header:hover { background: var(--bg-overlay); }
.channel-toggle { font-size: 14px; color: var(--text-tertiary); }
.channel-toggle.on { color: var(--accent); }
.channel-label { flex: 1; font-size: 13px; font-weight: 500; color: var(--text-primary); }
.channel-toggle-text { font-size: 11px; color: var(--text-tertiary); }
.channel-fields {
  padding: 10px 12px;
  background: var(--bg-app);
  border-top: 1px solid var(--border-subtle);
  display: grid; grid-template-columns: 1fr 1fr; gap: 8px;
}
.channel-field { display: flex; flex-direction: column; gap: 4px; }
.channel-field label { font-size: 11px; color: var(--text-secondary); font-weight: 500; }

/* Confirm */
.confirm-step { align-items: center; }
.confirm-card {
  text-align: center; padding: 24px;
  background: var(--bg-panel); border: 1px solid var(--border);
  border-radius: 10px; width: 100%;
}
.confirm-emoji { font-size: 48px; margin-bottom: 12px; }
.confirm-name { font-size: 18px; font-weight: 600; color: var(--text-primary); margin: 0 0 8px; }
.confirm-desc { font-size: 13px; color: var(--text-secondary); margin: 0 0 16px; }
.confirm-meta { display: flex; flex-direction: column; gap: 6px; text-align: left; }
.meta-row { display: flex; gap: 8px; }
.meta-key { font-size: 12px; color: var(--text-tertiary); min-width: 50px; }
.meta-val { font-size: 12px; color: var(--text-primary); }
.visibility-select {
  flex: 1;
  padding: 4px 8px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 12px;
  cursor: pointer;
}
.confirm-hint { font-size: 12px; color: var(--text-tertiary); margin-top: 4px; }

/* Finish */
.finish-step { align-items: center; padding: 32px 0; }
.finish-icon { font-size: 48px; margin-bottom: 12px; }
.finish-title { font-size: 18px; font-weight: 600; color: var(--text-primary); margin: 0 0 8px; }
.finish-desc { font-size: 13px; color: var(--text-secondary); text-align: center; line-height: 1.6; }

/* Footer */
.wizard-footer {
  display: flex; align-items: center;
  padding: 14px 20px; border-top: 1px solid var(--border-subtle); gap: 8px;
  flex-shrink: 0;
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
.btn-secondary:hover { background: var(--bg-elevated); }
</style>
