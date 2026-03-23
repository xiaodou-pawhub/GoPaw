<template>
  <div class="agents-root">
    <!-- 头部 -->
    <div class="agents-header">
      <div class="header-left">
        <h2 class="page-title">数字员工</h2>
        <span class="page-subtitle">管理 Agent 智能助手</span>
      </div>
      <div class="header-right">
        <div class="search-box">
          <SearchIcon :size="13" class="search-icon" />
          <input v-model="searchQuery" placeholder="搜索 Agent..." class="search-input" />
        </div>
        <button class="create-btn" @click="showCreateWizard = true">
          <PlusIcon :size="15" />
          <span>新建 Agent</span>
        </button>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="agents-loading">
      <div class="loading-spinner" />
      <span>加载 Agents...</span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="filteredAgents.length === 0" class="agents-empty">
      <BotIcon :size="48" class="empty-icon" />
      <p class="empty-text">{{ searchQuery ? '未找到匹配的数字员工' : '还没有数字员工' }}</p>
      <p class="empty-hint">{{ searchQuery ? '换个关键词试试' : '数字员工可以拥有记忆、知识库和技能，让每次对话更智能' }}</p>
      <button v-if="!searchQuery" class="empty-create-btn" @click="showCreateWizard = true">
        <PlusIcon :size="15" />
        创建我的第一个数字员工
      </button>
    </div>

    <!-- Agent 卡片网格 -->
    <div v-else class="agents-grid">
      <div
        v-for="agent in filteredAgents"
        :key="agent.id"
        class="agent-card"
        :class="{ 'is-default': agent.is_default }"
      >
        <!-- 右上角菜单 -->
        <div class="card-actions">
          <button class="icon-btn" title="版本管理" @click="openVersionDialog(agent)">
            <HistoryIcon :size="14" />
          </button>
          <button class="icon-btn" title="编辑" @click="openEdit(agent)">
            <EditIcon :size="14" />
          </button>
          <button
            class="icon-btn danger"
            title="删除"
            @click="confirmDelete(agent)"
          >
            <TrashIcon :size="14" />
          </button>
        </div>

        <!-- Avatar + 名称 -->
        <div class="card-avatar">{{ agent.config?.emoji || agent.avatar || '🤖' }}</div>
        <h3 class="card-name">{{ agent.name }}</h3>
        <p v-if="agent.config?.description || agent.description" class="card-desc">
          {{ agent.config?.description || agent.description }}
        </p>

        <!-- 活跃状态 -->
        <div class="card-status">
          <span class="status-dot" :class="getStatusClass(agent)" />
          <span class="status-text">{{ getStatusText(agent) }}</span>
        </div>

        <!-- 元信息 -->
        <div class="card-meta">
          <span v-if="(agent.config?.provider_ids?.length || 0) > 0" class="meta-chip">
            {{ agent.config!.provider_ids!.length }} 模型
          </span>
          <span v-else class="meta-chip muted">全局模型</span>
          <span v-if="(agent.config?.skills?.length || 0) > 0" class="meta-chip">
            {{ agent.config!.skills!.length }} 技能
          </span>
          <span v-if="(agent.config?.mcp_servers?.length || 0) > 0" class="meta-chip">
            {{ agent.config!.mcp_servers!.length }} MCP
          </span>
          <span v-if="agent.is_default" class="meta-chip accent">默认</span>
        </div>

        <!-- 开始对话按钮 -->
        <button class="chat-btn" @click="startChat(agent)">
          <MessageSquareIcon :size="13" />
          <span>开始对话</span>
        </button>
      </div>
    </div>

    <!-- CreateWizard 弹窗 -->
    <CreateWizard
      v-if="showCreateWizard"
      :providers="providers"
      :skills="skills"
      @close="showCreateWizard = false"
      @created="onCreated"
    />

    <!-- EditDialog 弹窗 -->
    <EditDialog
      v-if="editingAgent"
      :agent="editingAgent"
      :providers="providers"
      :skills="skills"
      :mcp-servers="mcpServers"
      @close="editingAgent = null"
      @saved="onSaved"
      @delete="handleDelete"
    />

    <!-- 删除确认 -->
    <div v-if="deletingAgent" class="overlay" @click.self="deletingAgent = null">
      <div class="confirm-dialog">
        <h4 class="confirm-title">删除 Agent</h4>
        <p class="confirm-msg">确定要删除 Agent <strong>{{ deletingAgent.name }}</strong> 吗？此操作不可恢复。</p>
        <div class="confirm-btns">
          <button class="btn-secondary" @click="deletingAgent = null">取消</button>
          <button class="btn-danger" @click="doDelete">删除</button>
        </div>
      </div>
    </div>

    <!-- 版本管理对话框 -->
    <div v-if="versionAgent" class="overlay" @click.self="versionAgent = null">
      <div class="version-dialog">
        <div class="version-header">
          <h4 class="version-title">
            <HistoryIcon :size="18" />
            版本管理 - {{ versionAgent.name }}
          </h4>
          <button class="close-btn" @click="versionAgent = null">×</button>
        </div>

        <div class="version-toolbar">
          <button class="btn-primary" :disabled="creatingVersion" @click="createNewVersion">
            <PlusIcon :size="14" />
            保存当前版本
          </button>
        </div>

        <div v-if="loadingVersions" class="version-loading">
          <div class="loading-spinner" />
          <span>加载版本...</span>
        </div>

        <div v-else-if="versions.length === 0" class="version-empty">
          <HistoryIcon :size="32" class="empty-icon" />
          <p>暂无历史版本</p>
          <p class="hint">保存版本后可随时回滚</p>
        </div>

        <div v-else class="version-list">
          <div
            v-for="v in versions"
            :key="v.id"
            class="version-item"
          >
            <div class="version-info">
              <span class="version-badge">v{{ v.version }}</span>
              <span class="version-name">{{ v.name || '未命名版本' }}</span>
              <span class="version-date">{{ formatDate(v.created_at) }}</span>
            </div>
            <div class="version-actions">
              <button class="btn-sm" title="回滚到此版本" @click="rollbackVersion(v.version)">
                <RotateCcwIcon :size="12" />
                回滚
              </button>
              <button class="btn-sm danger" title="删除版本" @click="deleteVersion(v.version)">
                <TrashIcon :size="12" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  PlusIcon, BotIcon, SearchIcon,
  EditIcon, TrashIcon, MessageSquareIcon,
  HistoryIcon, RotateCcwIcon
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { listAgents, deleteAgent, type Agent } from '@/api/agents'
import { getProviders, getSkills, type Skill } from '@/api/settings'
import { listMCPServers, type MCPServer } from '@/api/mcp'
import type { BackendProvider } from '@/types'
import CreateWizard from '@/components/agents/CreateWizard.vue'
import EditDialog from '@/components/agents/EditDialog.vue'
import axios from 'axios'

const router = useRouter()

// 版本类型
interface AgentVersion {
  id: string
  agent_id: string
  version: number
  name: string
  description: string
  config: any
  created_at: string
  created_by: string
}

// ---- State ----
const agents = ref<Agent[]>([])
const providers = ref<BackendProvider[]>([])
const skills = ref<Skill[]>([])
const mcpServers = ref<MCPServer[]>([])
const loading = ref(false)
const searchQuery = ref('')
const showCreateWizard = ref(false)
const editingAgent = ref<Agent | null>(null)
const deletingAgent = ref<Agent | null>(null)

// 版本管理
const versionAgent = ref<Agent | null>(null)
const versions = ref<AgentVersion[]>([])
const loadingVersions = ref(false)
const creatingVersion = ref(false)

// ---- Computed ----
const filteredAgents = computed(() => {
  if (!searchQuery.value) return agents.value
  const q = searchQuery.value.toLowerCase()
  return agents.value.filter(a =>
    a.name.toLowerCase().includes(q) ||
    (a.description || '').toLowerCase().includes(q) ||
    (a.config?.description || '').toLowerCase().includes(q)
  )
})

// ---- Status ----
function getStatusClass(agent: Agent): string {
  if (!agent.last_active_at) return 'offline'
  const diff = Date.now() - agent.last_active_at
  return diff < 3600000 ? 'online' : 'offline'
}

function getStatusText(agent: Agent): string {
  if (!agent.last_active_at) return '未激活'
  const diff = Date.now() - agent.last_active_at
  if (diff < 60000) return '刚刚活跃'
  if (diff < 3600000) return '活跃'
  return '离线'
}

// ---- Methods ----
async function loadAll() {
  loading.value = true
  try {
    const [agentsRes, providersRes, skillsRes, mcpRes] = await Promise.allSettled([
      listAgents(),
      getProviders(),
      getSkills(),
      listMCPServers(),
    ])
    if (agentsRes.status === 'fulfilled') agents.value = agentsRes.value.agents
    if (providersRes.status === 'fulfilled') providers.value = providersRes.value
    if (skillsRes.status === 'fulfilled') skills.value = skillsRes.value
    if (mcpRes.status === 'fulfilled') mcpServers.value = mcpRes.value.servers
  } catch (err) {
    console.error('Failed to load agents page data:', err)
  } finally {
    loading.value = false
  }
}

function openEdit(agent: Agent) {
  editingAgent.value = agent
}

function confirmDelete(agent: Agent) {
  deletingAgent.value = agent
}

async function doDelete() {
  if (!deletingAgent.value) return
  try {
    await deleteAgent(deletingAgent.value.id)
    toast.success('已删除')
    deletingAgent.value = null
    await loadAll()
  } catch {
    toast.error('删除失败')
  }
}

async function handleDelete(id: string) {
  const agent = agents.value.find(a => a.id === id)
  if (agent) {
    editingAgent.value = null
    deletingAgent.value = agent
  }
}

async function onCreated() {
  showCreateWizard.value = false
  await loadAll()
}

async function onSaved() {
  editingAgent.value = null
  await loadAll()
}

function startChat(agent: Agent) {
  router.push({ path: '/chat', query: { agent_id: agent.id } })
}

// ---- 版本管理 ----
function openVersionDialog(agent: Agent) {
  versionAgent.value = agent
  loadVersions()
}

async function loadVersions() {
  if (!versionAgent.value) return
  loadingVersions.value = true
  try {
    const res = await axios.get(`/api/agents/${versionAgent.value.id}/versions`)
    versions.value = res.data.data || []
  } catch (err) {
    console.error('Failed to load versions:', err)
    toast.error('加载版本失败')
  } finally {
    loadingVersions.value = false
  }
}

async function createNewVersion() {
  if (!versionAgent.value) return
  creatingVersion.value = true
  try {
    const name = `版本 ${new Date().toLocaleString('zh-CN')}`
    await axios.post(`/api/agents/${versionAgent.value.id}/versions`, { name })
    toast.success('版本已保存')
    await loadVersions()
  } catch (err) {
    console.error('Failed to create version:', err)
    toast.error('保存版本失败')
  } finally {
    creatingVersion.value = false
  }
}

async function rollbackVersion(version: number) {
  if (!versionAgent.value) return
  if (!confirm(`确定要回滚到版本 v${version} 吗？当前配置将被覆盖。`)) return
  try {
    await axios.post(`/api/agents/${versionAgent.value.id}/versions/${version}/rollback`)
    toast.success('已回滚到版本 v' + version)
    await loadAll()
    await loadVersions()
  } catch (err) {
    console.error('Failed to rollback:', err)
    toast.error('回滚失败')
  }
}

async function deleteVersion(version: number) {
  if (!versionAgent.value) return
  if (!confirm(`确定要删除版本 v${version} 吗？`)) return
  try {
    await axios.delete(`/api/agents/${versionAgent.value.id}/versions/${version}`)
    toast.success('版本已删除')
    await loadVersions()
  } catch (err) {
    console.error('Failed to delete version:', err)
    toast.error('删除版本失败')
  }
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

// ---- Lifecycle ----
onMounted(loadAll)
</script>

<style scoped>
.agents-root {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 20px 24px;
  overflow: hidden;
  height: 100%;
}

.agents-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: 12px;
  color: var(--text-tertiary);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 7px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 7px;
  width: 180px;
}
.search-icon { color: var(--text-tertiary); flex-shrink: 0; }
.search-input {
  flex: 1;
  background: none;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-size: 12px;
  min-width: 0;
}
.search-input::placeholder { color: var(--text-tertiary); }

.create-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  border: none;
  border-radius: 7px;
  background: var(--accent);
  color: white;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
  white-space: nowrap;
}
.create-btn:hover { opacity: 0.9; }

/* Loading & Empty */
.agents-loading,
.agents-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--text-tertiary);
}

.loading-spinner {
  width: 28px;
  height: 28px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

.empty-icon { opacity: 0.4; }
.empty-text { font-size: 15px; font-weight: 500; color: var(--text-secondary); }
.empty-hint { font-size: 13px; color: var(--text-tertiary); text-align: center; max-width: 320px; line-height: 1.5; }

.empty-create-btn {
  display: flex; align-items: center; gap: 7px;
  margin-top: 8px;
  padding: 9px 20px;
  background: var(--accent); border: none; border-radius: 8px;
  color: #fff; font-size: 13px; font-weight: 500;
  cursor: pointer; transition: opacity 0.15s;
}
.empty-create-btn:hover { opacity: 0.9; }

/* Grid */
.agents-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 14px;
  overflow-y: auto;
  align-content: start;
}

/* Card */
.agent-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 16px 14px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 10px;
  transition: all 0.15s;
  text-align: center;
}

.agent-card:hover { border-color: var(--border-hover); box-shadow: 0 2px 8px rgba(0,0,0,0.12); }
.agent-card.is-default { border-color: var(--accent); }

/* Card Actions (top-right) */
.card-actions {
  position: absolute;
  top: 10px;
  right: 10px;
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s;
}
.agent-card:hover .card-actions { opacity: 1; }

.icon-btn {
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-overlay);
  border: none;
  border-radius: 5px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.12s;
}
.icon-btn:hover { background: var(--bg-elevated); color: var(--text-primary); }
.icon-btn.danger:hover { color: var(--red); }
.icon-btn:disabled { opacity: 0.35; cursor: not-allowed; }

.card-avatar {
  font-size: 40px;
  width: 60px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-overlay);
  border-radius: 14px;
  margin-bottom: 10px;
}

.card-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-desc {
  font-size: 11px;
  color: var(--text-secondary);
  margin: 0 0 10px;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* Status */
.card-status {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-bottom: 10px;
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}
.status-dot.online { background: var(--green); box-shadow: 0 0 5px rgba(34,197,94,0.5); }
.status-dot.offline { background: var(--text-disabled); }

.status-text { font-size: 11px; color: var(--text-tertiary); }

/* Meta chips */
.card-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  justify-content: center;
  margin-bottom: 12px;
}

.meta-chip {
  font-size: 10px;
  padding: 2px 7px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-subtle);
  border-radius: 10px;
  color: var(--text-secondary);
}
.meta-chip.muted { color: var(--text-tertiary); }
.meta-chip.accent { background: var(--accent-dim); border-color: var(--accent); color: var(--accent); }

/* Chat button */
.chat-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 6px 14px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
  width: 100%;
  justify-content: center;
}
.chat-btn:hover { background: var(--accent-dim); border-color: var(--accent); color: var(--accent); }

/* Delete Confirm Dialog */
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.confirm-dialog {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 20px 24px;
  width: 360px;
}

.confirm-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 8px;
}

.confirm-msg {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0 0 16px;
  line-height: 1.5;
}

.confirm-btns {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.btn-secondary {
  padding: 7px 14px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 13px;
  cursor: pointer;
}
.btn-secondary:hover { background: var(--bg-elevated); }

.btn-danger {
  padding: 7px 14px;
  background: var(--red-dim);
  border: 1px solid var(--red);
  border-radius: 6px;
  color: var(--red);
  font-size: 13px;
  cursor: pointer;
}
.btn-danger:hover { opacity: 0.85; }

/* 版本管理对话框 */
.version-dialog {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  width: 480px;
  max-height: 70vh;
  display: flex;
  flex-direction: column;
}

.version-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.version-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.close-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 18px;
  cursor: pointer;
  border-radius: 6px;
}
.close-btn:hover { background: var(--bg-overlay); }

.version-toolbar {
  padding: 12px 20px;
  border-bottom: 1px solid var(--border);
}

.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 13px;
  cursor: pointer;
}
.btn-primary:hover { opacity: 0.9; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.version-loading,
.version-empty {
  padding: 40px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
  font-size: 13px;
}
.version-empty .empty-icon { opacity: 0.4; }
.version-empty .hint { font-size: 12px; opacity: 0.6; }

.version-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.version-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  border-bottom: 1px solid var(--border);
}
.version-item:last-child { border-bottom: none; }
.version-item:hover { background: var(--bg-overlay); }

.version-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.version-badge {
  padding: 2px 8px;
  background: var(--accent-dim);
  color: var(--accent);
  font-size: 11px;
  font-weight: 600;
  border-radius: 4px;
}

.version-name {
  font-size: 13px;
  color: var(--text-primary);
}

.version-date {
  font-size: 12px;
  color: var(--text-secondary);
}

.version-actions {
  display: flex;
  gap: 6px;
}

.btn-sm {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 10px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
}
.btn-sm:hover { background: var(--bg-elevated); color: var(--text-primary); }
.btn-sm.danger { color: var(--red); border-color: var(--red-dim); }
.btn-sm.danger:hover { background: var(--red-dim); }
</style>
