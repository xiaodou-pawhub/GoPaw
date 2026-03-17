<template>
  <div class="mcp-root">
    <!-- 头部 -->
    <div class="mcp-header">
      <div class="header-left">
        <h2 class="page-title">MCP 服务器</h2>
        <span class="page-subtitle">管理 Model Context Protocol 服务器</span>
      </div>
      <button class="create-btn" @click="showCreateModal = true">
        <PlusIcon :size="16" />
        <span>添加服务器</span>
      </button>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="mcp-loading">
      <div class="loading-spinner" />
      <span>加载 MCP 服务器...</span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="servers.length === 0" class="mcp-empty">
      <ServerIcon :size="48" class="empty-icon" />
      <p class="empty-text">暂无 MCP 服务器</p>
      <p class="empty-hint">点击右上角按钮添加第一个服务器</p>
    </div>

    <!-- 服务器列表 -->
    <div v-else class="servers-grid">
      <div
        v-for="server in servers"
        :key="server.id"
        class="server-card"
        :class="{ active: server.is_active, builtin: server.is_builtin }"
      >
        <div class="server-header">
          <div class="server-icon">
            <ServerIcon :size="24" />
          </div>
          <div class="server-badges">
            <span v-if="server.is_builtin" class="badge builtin">内置</span>
            <span v-if="server.is_active" class="badge active">运行中</span>
            <span v-else class="badge inactive">已停止</span>
          </div>
        </div>

        <div class="server-info">
          <h3 class="server-name">{{ server.name }}</h3>
          <p class="server-id">{{ server.id }}</p>
          <p class="server-desc">{{ server.description || '暂无描述' }}</p>
        </div>

        <div class="server-meta">
          <span class="meta-item">
            <TerminalIcon :size="12" />
            {{ server.transport }}
          </span>
          <span class="meta-item">
            <CommandIcon :size="12" />
            {{ server.command }}
          </span>
        </div>

        <div class="server-actions">
          <button
            class="action-btn"
            :class="{ active: server.is_active }"
            :disabled="server.is_builtin"
            @click="toggleActive(server)"
          >
            <PowerIcon :size="14" />
            <span>{{ server.is_active ? '停止' : '启动' }}</span>
          </button>
          <button class="action-btn" @click="viewTools(server)">
            <WrenchIcon :size="14" />
            <span>工具</span>
          </button>
          <button
            class="action-btn"
            :disabled="server.is_builtin"
            @click="editServer(server)"
          >
            <EditIcon :size="14" />
            <span>编辑</span>
          </button>
          <button
            class="action-btn danger"
            :disabled="server.is_builtin"
            @click="confirmDelete(server)"
          >
            <TrashIcon :size="14" />
            <span>删除</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 创建/编辑模态框 -->
    <MCPServerModal
      v-if="showCreateModal || editingServer"
      :server="editingServer"
      @close="closeModal"
      @save="handleSave"
    />

    <!-- 工具列表模态框 -->
    <MCPToolsModal
      v-if="viewingServer"
      :server="viewingServer"
      :tools="serverTools"
      @close="viewingServer = null"
    />

    <!-- 删除确认 -->
    <ConfirmDialog
      v-if="deletingServer"
      title="删除 MCP 服务器"
      :message="deleteConfirmMessage"
      @confirm="handleDelete"
      @cancel="deletingServer = null"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  PlusIcon,
  ServerIcon,
  TerminalIcon,
  PowerIcon,
  WrenchIcon,
  EditIcon,
  TrashIcon
} from 'lucide-vue-next'
import {
  listMCPServers,
  deleteMCPServer,
  setMCPServerActive,
  getMCPServerTools,
  type MCPServer,
  type MCPTool
} from '@/api/mcp'
import MCPServerModal from '@/components/mcp/MCPServerModal.vue'
import MCPToolsModal from '@/components/mcp/MCPToolsModal.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

// ---- State ----
const servers = ref<MCPServer[]>([])
const loading = ref(false)
const showCreateModal = ref(false)
const editingServer = ref<MCPServer | null>(null)
const deletingServer = ref<MCPServer | null>(null)
const viewingServer = ref<MCPServer | null>(null)
const serverTools = ref<MCPTool[]>([])

// ---- Computed ----
const deleteConfirmMessage = computed(() => {
  const name = deletingServer.value?.name || ''
  return `确定要删除 MCP 服务器 "${name}" 吗？此操作不可恢复。`
})

// ---- Methods ----
async function loadServers() {
  loading.value = true
  try {
    const res = await listMCPServers()
    servers.value = res.servers
  } catch (err) {
    console.error('Failed to load MCP servers:', err)
  } finally {
    loading.value = false
  }
}

async function toggleActive(server: MCPServer) {
  try {
    await setMCPServerActive(server.id, !server.is_active)
    await loadServers()
  } catch (err) {
    console.error('Failed to toggle server status:', err)
  }
}

async function viewTools(server: MCPServer) {
  viewingServer.value = server
  try {
    const res = await getMCPServerTools(server.id)
    serverTools.value = res.tools
  } catch (err) {
    console.error('Failed to load server tools:', err)
    serverTools.value = []
  }
}

function editServer(server: MCPServer) {
  editingServer.value = server
}

function closeModal() {
  showCreateModal.value = false
  editingServer.value = null
}

async function handleSave() {
  await loadServers()
  closeModal()
}

function confirmDelete(server: MCPServer) {
  deletingServer.value = server
}

async function handleDelete() {
  if (!deletingServer.value) return
  try {
    await deleteMCPServer(deletingServer.value.id)
    await loadServers()
  } catch (err) {
    console.error('Failed to delete server:', err)
  } finally {
    deletingServer.value = null
  }
}

// ---- Lifecycle ----
onMounted(() => {
  loadServers()
})
</script>

<style scoped>
.mcp-root {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 20px;
  overflow: hidden;
}

.mcp-header {
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
.mcp-loading,
.mcp-empty {
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

/* Servers Grid */
.servers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
  overflow-y: auto;
}

.server-card {
  display: flex;
  flex-direction: column;
  padding: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 8px;
  transition: all 0.15s;
}

.server-card:hover {
  border-color: var(--border-hover);
}

.server-card.active {
  border-color: var(--green);
  background: var(--green-dim);
}

.server-card.builtin {
  border-left: 3px solid var(--accent);
}

.server-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.server-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-overlay);
  border-radius: 8px;
  color: var(--accent);
}

.server-badges {
  display: flex;
  gap: 4px;
}

.badge {
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 500;
  border-radius: 4px;
}

.badge.active {
  background: var(--green);
  color: white;
}

.badge.inactive {
  background: var(--bg-overlay);
  color: var(--text-tertiary);
}

.badge.builtin {
  background: var(--accent-dim);
  color: var(--accent);
}

.server-info {
  flex: 1;
  margin-bottom: 12px;
}

.server-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.server-id {
  font-size: 12px;
  color: var(--text-tertiary);
  font-family: monospace;
  margin-bottom: 8px;
}

.server-desc {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.server-meta {
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

.server-actions {
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
  background: var(--green-dim);
  color: var(--green);
  border-color: var(--green);
}

.action-btn.danger {
  color: var(--red);
}

.action-btn.danger:hover:not(:disabled) {
  background: var(--red-dim);
  color: var(--red);
}
</style>
