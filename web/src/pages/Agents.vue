<template>
  <div class="agents-root">
    <!-- 头部 -->
    <div class="agents-header">
      <div class="header-left">
        <h2 class="page-title">Agents</h2>
        <span class="page-subtitle">管理多智能体配置</span>
      </div>
      <button class="create-btn" @click="showCreateModal = true">
        <PlusIcon :size="16" />
        <span>新建 Agent</span>
      </button>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="agents-loading">
      <div class="loading-spinner" />
      <span>加载 Agents...</span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="agents.length === 0" class="agents-empty">
      <BotIcon :size="48" class="empty-icon" />
      <p class="empty-text">暂无 Agents</p>
      <p class="empty-hint">点击右上角按钮创建第一个 Agent</p>
    </div>

    <!-- Agent 列表 -->
    <div v-else class="agents-grid">
      <div
        v-for="agent in agents"
        :key="agent.id"
        class="agent-card"
        :class="{ default: agent.is_default, inactive: !agent.is_active }"
      >
        <div class="agent-header">
          <div class="agent-avatar">{{ agent.avatar || '🤖' }}</div>
          <div class="agent-badges">
            <span v-if="agent.is_default" class="badge default">默认</span>
            <span v-if="!agent.is_active" class="badge inactive">停用</span>
          </div>
        </div>

        <div class="agent-info">
          <h3 class="agent-name">{{ agent.name }}</h3>
          <p class="agent-id">{{ agent.id }}</p>
          <p class="agent-desc">{{ agent.description || '暂无描述' }}</p>
        </div>

        <div class="agent-meta">
          <span class="meta-item">
            <SettingsIcon :size="12" />
            {{ agent.config?.llm?.model || '默认' }}
          </span>
          <span class="meta-item">
            <ToolIcon :size="12" />
            {{ getEnabledToolsCount(agent) }} 工具
          </span>
        </div>

        <div class="agent-actions">
          <button
            class="action-btn"
            :class="{ active: agent.is_default }"
            :disabled="agent.is_default"
            @click="setDefault(agent.id)"
          >
            <StarIcon :size="14" />
            <span>{{ agent.is_default ? '默认' : '设为默认' }}</span>
          </button>
          <button class="action-btn" @click="editAgent(agent)">
            <EditIcon :size="14" />
            <span>编辑</span>
          </button>
          <button
            class="action-btn danger"
            :disabled="agent.is_default"
            @click="confirmDelete(agent)"
          >
            <TrashIcon :size="14" />
            <span>删除</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 创建/编辑模态框 -->
    <AgentEditModal
      v-if="showCreateModal || editingAgent"
      :agent="editingAgent"
      @close="closeModal"
      @save="handleSave"
    />

    <!-- 删除确认 -->
    <ConfirmDialog
      v-if="deletingAgent"
      title="删除 Agent"
      :message="deleteConfirmMessage"
      @confirm="handleDelete"
      @cancel="deletingAgent = null"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  PlusIcon,
  BotIcon,
  SettingsIcon,
  StarIcon,
  EditIcon,
  TrashIcon,
  WrenchIcon as ToolIcon
} from 'lucide-vue-next'
import { listAgents, deleteAgent, setDefaultAgent, type Agent } from '@/api/agents'
import AgentEditModal from '@/components/agents/AgentEditModal.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

// ---- State ----
const agents = ref<Agent[]>([])
const loading = ref(false)
const showCreateModal = ref(false)
const editingAgent = ref<Agent | null>(null)
const deletingAgent = ref<Agent | null>(null)

// ---- Computed ----
const deleteConfirmMessage = computed(() => {
  const name = deletingAgent.value?.name || ''
  return `确定要删除 Agent "${name}" 吗？此操作不可恢复。`
})

// ---- Methods ----
async function loadAgents() {
  loading.value = true
  try {
    const res = await listAgents()
    agents.value = res.agents
  } catch (err) {
    console.error('Failed to load agents:', err)
  } finally {
    loading.value = false
  }
}

function getEnabledToolsCount(agent: Agent): number {
  if (!agent.config?.tools?.enabled) return 0
  return agent.config.tools.enabled.length
}

function editAgent(agent: Agent) {
  editingAgent.value = agent
}

function closeModal() {
  showCreateModal.value = false
  editingAgent.value = null
}

async function handleSave() {
  await loadAgents()
  closeModal()
}

function confirmDelete(agent: Agent) {
  deletingAgent.value = agent
}

async function handleDelete() {
  if (!deletingAgent.value) return
  try {
    await deleteAgent(deletingAgent.value.id)
    await loadAgents()
  } catch (err) {
    console.error('Failed to delete agent:', err)
  } finally {
    deletingAgent.value = null
  }
}

async function setDefault(id: string) {
  try {
    await setDefaultAgent(id)
    await loadAgents()
  } catch (err) {
    console.error('Failed to set default:', err)
  }
}

// ---- Lifecycle ----
onMounted(() => {
  loadAgents()
})
</script>

<style scoped>
.agents-root {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 20px;
  overflow: hidden;
}

.agents-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.header-left {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
}

.page-subtitle {
  font-size: 13px;
  color: var(--text-tertiary);
}

.create-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: none;
  border-radius: 6px;
  background: var(--accent);
  color: white;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
}

.create-btn:hover {
  opacity: 0.9;
}

/* Loading & Empty */
.agents-loading,
.agents-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-tertiary);
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.empty-icon {
  color: var(--text-tertiary);
  opacity: 0.5;
}

.empty-text {
  font-size: 15px;
  color: var(--text-secondary);
}

.empty-hint {
  font-size: 13px;
  color: var(--text-tertiary);
}

/* Agents Grid */
.agents-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
  overflow-y: auto;
}

.agent-card {
  display: flex;
  flex-direction: column;
  padding: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 8px;
  transition: all 0.15s;
}

.agent-card:hover {
  border-color: var(--border-hover);
}

.agent-card.default {
  border-color: var(--accent);
  background: var(--accent-dim);
}

.agent-card.inactive {
  opacity: 0.7;
}

.agent-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.agent-avatar {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  background: var(--bg-overlay);
  border-radius: 8px;
}

.agent-badges {
  display: flex;
  gap: 4px;
}

.badge {
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 500;
  border-radius: 4px;
}

.badge.default {
  background: var(--accent);
  color: white;
}

.badge.inactive {
  background: var(--bg-overlay);
  color: var(--text-tertiary);
}

.agent-info {
  flex: 1;
  margin-bottom: 12px;
}

.agent-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.agent-id {
  font-size: 12px;
  color: var(--text-tertiary);
  font-family: monospace;
  margin-bottom: 8px;
}

.agent-desc {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.agent-meta {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--text-tertiary);
}

.agent-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 6px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-input);
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.action-btn:hover:not(:disabled) {
  background: var(--bg-overlay);
  color: var(--text-primary);
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.action-btn.active {
  background: var(--accent-dim);
  color: var(--accent);
  border-color: var(--accent);
}

.action-btn.danger {
  color: var(--red);
}

.action-btn.danger:hover:not(:disabled) {
  background: var(--red-dim);
  color: var(--red);
}
</style>
