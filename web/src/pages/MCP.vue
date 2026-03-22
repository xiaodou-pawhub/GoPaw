<template>
  <div class="mcp-root">
    <!-- 头部 -->
    <div class="mcp-header">
      <div class="header-left">
        <h2 class="page-title">MCP 服务</h2>
        <span class="page-subtitle">Model Context Protocol 服务器管理</span>
      </div>
      <div class="view-toggle">
        <button class="toggle-btn" :class="{ active: viewMode === 'installed' }" @click="viewMode = 'installed'">已安装</button>
        <button class="toggle-btn" :class="{ active: viewMode === 'market' }" @click="viewMode = 'market'">市场</button>
      </div>
    </div>

    <!-- ========== 已安装视图 ========== -->
    <div v-if="viewMode === 'installed'" class="view-content">

      <!-- 内联添加表单 -->
      <div class="add-section">
        <button v-if="!showAddForm" class="add-toggle-btn" @click="showAddForm = true">
          <PlusIcon :size="14" />
          <span>添加自定义服务器</span>
        </button>

        <div v-else class="add-form">
          <div class="form-header">
            <span class="form-title">新增 MCP 服务器</span>
            <div class="form-mode-toggle">
              <button class="mode-btn" :class="{ active: addMode === 'form' }" @click="addMode = 'form'">表单</button>
              <button class="mode-btn" :class="{ active: addMode === 'json' }" @click="addMode = 'json'">JSON</button>
            </div>
            <button class="icon-close" @click="resetAddForm"><XIcon :size="14" /></button>
          </div>

          <!-- 表单模式 -->
          <template v-if="addMode === 'form'">
            <div class="form-row-2">
              <div class="form-field">
                <label>名称 <span class="required">*</span></label>
                <input v-model="addForm.name" placeholder="如：My Filesystem" class="form-input" />
              </div>
              <div class="form-field">
                <label>命令 <span class="required">*</span></label>
                <input v-model="addForm.command" placeholder="如：npx" class="form-input" />
              </div>
            </div>
            <div class="form-field">
              <label>参数（每行一个）</label>
              <textarea v-model="addForm.argsText" rows="3" placeholder="-y&#10;@modelcontextprotocol/server-filesystem&#10;/path/to/dir" class="form-textarea" />
            </div>
            <div class="form-field">
              <label>环境变量（每行 KEY=VALUE）</label>
              <textarea v-model="addForm.envText" rows="2" placeholder="API_KEY=xxx" class="form-textarea" />
            </div>
          </template>

          <!-- JSON 模式 -->
          <template v-else>
            <div class="form-field">
              <label>JSON 配置</label>
              <textarea
                v-model="addJsonText"
                rows="10"
                class="form-textarea"
                :class="{ 'json-error': addJsonError }"
                placeholder='{"id":"my-server","name":"My MCP","command":"npx","args":["-y","@some/mcp-server"],"env":["API_KEY=xxx"],"transport":"stdio"}'
                spellcheck="false"
                @input="validateJson"
              />
              <span v-if="addJsonError" class="json-error-msg">{{ addJsonError }}</span>
            </div>
          </template>

          <!-- 测试结果 -->
          <div v-if="testResult !== null" class="test-result" :class="{ error: testError }">
            <template v-if="testError">
              <XCircleIcon :size="14" class="test-icon" />
              <span>{{ testError }}</span>
            </template>
            <template v-else>
              <CheckCircleIcon :size="14" class="test-icon success" />
              <span>发现 {{ testResult.length }} 个工具：{{ testResult.join(', ') || '(无)' }}</span>
            </template>
          </div>

          <div class="form-actions">
            <button v-if="addMode === 'form'" class="btn-secondary" :disabled="testing" @click="handleTest">
              {{ testing ? '测试中...' : '测试连接' }}
            </button>
            <div class="form-spacer" />
            <button class="btn-secondary" @click="resetAddForm">取消</button>
            <button
              class="btn-primary"
              :disabled="addLoading || (addMode === 'form' ? (!addForm.name || !addForm.command) : !!addJsonError)"
              @click="handleAdd"
            >
              {{ addLoading ? '添加中...' : '确认添加' }}
            </button>
          </div>
        </div>
      </div>

      <!-- 已安装服务器列表 -->
      <div v-if="loading" class="loading-hint">加载中...</div>
      <div v-else-if="servers.length === 0" class="empty-hint">
        <ServerIcon :size="36" class="empty-icon" />
        <p>暂无 MCP 服务器</p>
        <p class="empty-sub">点击上方按钮添加，或从市场选择模板</p>
      </div>
      <div v-else class="servers-list">
        <div v-for="srv in servers" :key="srv.id" class="server-row">
          <div class="srv-left">
            <div class="srv-icon">
              <ServerIcon :size="16" />
            </div>
            <div class="srv-info">
              <span class="srv-name">{{ srv.name }}</span>
              <span class="srv-cmd">{{ srv.command }} {{ srv.args?.join(' ') }}</span>
            </div>
          </div>
          <div class="srv-right">
            <span class="srv-badge" :class="srv.is_active ? 'active' : 'inactive'">
              {{ srv.is_active ? '运行中' : '已停止' }}
            </span>
            <span v-if="srv.is_builtin" class="srv-badge builtin">内置</span>
            <span v-if="agentBindCount(srv.id) > 0" class="srv-badge agents">
              {{ agentBindCount(srv.id) }} Agent
            </span>
            <button class="icon-btn" :title="srv.is_active ? '停止' : '启动'" :disabled="srv.is_builtin" @click="toggleActive(srv)">
              <PowerIcon :size="13" />
            </button>
            <button class="icon-btn" title="删除" :disabled="srv.is_builtin" @click="confirmDelete(srv)">
              <TrashIcon :size="13" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- ========== 市场视图 ========== -->
    <div v-else class="view-content market-view">
      <div class="market-search">
        <SearchIcon :size="13" class="search-icon" />
        <input v-model="marketQuery" placeholder="搜索 MCP 服务..." class="search-input" />
      </div>

      <div class="catalog-grid">
        <div
          v-for="item in filteredCatalog"
          :key="item.name"
          class="catalog-card"
          @click="selectTemplate(item)"
        >
          <div class="cat-header">
            <span class="cat-name">{{ item.display }}</span>
            <span v-for="tag in item.tags" :key="tag" class="cat-tag">{{ tag }}</span>
          </div>
          <div class="cat-cmd">{{ item.command }} {{ item.args?.join(' ') }}</div>
          <div class="cat-hint">点击使用此模板</div>
        </div>
      </div>
    </div>

    <!-- 删除确认 -->
    <div v-if="deletingServer" class="overlay" @click.self="deletingServer = null">
      <div class="confirm-dialog">
        <h4 class="confirm-title">删除 MCP 服务器</h4>
        <p class="confirm-msg">确定要删除 <strong>{{ deletingServer.name }}</strong> 吗？此操作不可恢复。</p>
        <div class="confirm-btns">
          <button class="btn-secondary" @click="deletingServer = null">取消</button>
          <button class="btn-danger" @click="doDelete">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  PlusIcon, ServerIcon, PowerIcon, TrashIcon,
  XIcon, SearchIcon, XCircleIcon, CheckCircleIcon
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import {
  listMCPServers, createMCPServer, deleteMCPServer, setMCPServerActive, testMCPServer,
  type MCPServer
} from '@/api/mcp'
import { listAgents } from '@/api/agents'

// ---- State ----
const servers = ref<MCPServer[]>([])
const agents = ref<any[]>([])
const loading = ref(false)
const viewMode = ref<'installed' | 'market'>('installed')
const deletingServer = ref<MCPServer | null>(null)
const marketQuery = ref('')

// Add form
const showAddForm = ref(false)
const addLoading = ref(false)
const testing = ref(false)
const testResult = ref<string[] | null>(null)
const testError = ref('')
const addMode = ref<'form' | 'json'>('form')
const addJsonText = ref('')
const addJsonError = ref('')
const addForm = ref({
  name: '',
  command: '',
  argsText: '',
  envText: '',
})

// ---- Market Catalog ----
const mcpCatalog = [
  { name: 'filesystem', display: '文件系统', command: 'npx', args: ['-y', '@modelcontextprotocol/server-filesystem', '/path/to/dir'], tags: ['官方'] },
  { name: 'git',        display: 'Git',       command: 'uvx', args: ['mcp-server-git', '--repository', '/path/to/repo'], tags: ['官方'] },
  { name: 'github',     display: 'GitHub',    command: 'npx', args: ['-y', '@modelcontextprotocol/server-github'], env: ['GITHUB_PERSONAL_ACCESS_TOKEN='], tags: ['官方'] },
  { name: 'fetch',      display: 'Web Fetch', command: 'uvx', args: ['mcp-server-fetch'], tags: ['官方'] },
  { name: 'sqlite',     display: 'SQLite',    command: 'uvx', args: ['mcp-server-sqlite', '--db-path', '/path/to/db.sqlite'], tags: ['数据库'] },
  { name: 'postgres',   display: 'PostgreSQL', command: 'npx', args: ['-y', '@modelcontextprotocol/server-postgres', 'postgresql://user:pass@localhost/db'], tags: ['数据库'] },
  { name: 'memory',     display: '知识图谱',   command: 'npx', args: ['-y', '@modelcontextprotocol/server-memory'], tags: ['官方'] },
  { name: 'brave',      display: 'Brave 搜索', command: 'npx', args: ['-y', '@modelcontextprotocol/server-brave-search'], env: ['BRAVE_API_KEY='], tags: ['搜索'] },
  { name: 'puppeteer',  display: 'Puppeteer', command: 'npx', args: ['-y', '@modelcontextprotocol/server-puppeteer'], tags: ['浏览器'] },
]

const filteredCatalog = computed(() => {
  if (!marketQuery.value) return mcpCatalog
  const q = marketQuery.value.toLowerCase()
  return mcpCatalog.filter(c =>
    c.display.toLowerCase().includes(q) ||
    c.name.toLowerCase().includes(q) ||
    c.tags.some(t => t.includes(q))
  )
})

// ---- Agent bind count ----
function agentBindCount(serverId: string): number {
  return agents.value.filter(a => a.config?.mcp_servers?.includes(serverId)).length
}

// ---- Load ----
async function loadAll() {
  loading.value = true
  try {
    const [srvRes, agentRes] = await Promise.allSettled([listMCPServers(), listAgents()])
    if (srvRes.status === 'fulfilled') servers.value = srvRes.value.servers
    if (agentRes.status === 'fulfilled') agents.value = agentRes.value.agents
  } catch {}
  finally { loading.value = false }
}

// ---- Toggle active ----
async function toggleActive(srv: MCPServer) {
  try {
    await setMCPServerActive(srv.id, !srv.is_active)
    await loadAll()
  } catch {
    toast.error('操作失败')
  }
}

// ---- Delete ----
function confirmDelete(srv: MCPServer) { deletingServer.value = srv }

async function doDelete() {
  if (!deletingServer.value) return
  try {
    await deleteMCPServer(deletingServer.value.id)
    toast.success('已删除')
    deletingServer.value = null
    await loadAll()
  } catch {
    toast.error('删除失败')
  }
}

// ---- Add form ----
function resetAddForm() {
  showAddForm.value = false
  testResult.value = null
  testError.value = ''
  addMode.value = 'form'
  addJsonText.value = ''
  addJsonError.value = ''
  addForm.value = { name: '', command: '', argsText: '', envText: '' }
}

function validateJson() {
  if (!addJsonText.value.trim()) {
    addJsonError.value = ''
    return
  }
  try {
    JSON.parse(addJsonText.value)
    addJsonError.value = ''
  } catch (e: any) {
    addJsonError.value = e.message || 'JSON 格式错误'
  }
}

function parseLines(text: string): string[] {
  return text.split('\n').map(l => l.trim()).filter(Boolean)
}

async function handleTest() {
  if (!addForm.value.command) { toast.error('请填写命令'); return }
  testing.value = true
  testResult.value = null
  testError.value = ''
  try {
    const res = await testMCPServer({
      command: addForm.value.command,
      args: parseLines(addForm.value.argsText),
      env: parseLines(addForm.value.envText),
    })
    if (res.error) {
      testError.value = res.error
    } else {
      testResult.value = res.tools
      testError.value = ''
    }
  } catch (err: any) {
    testError.value = err?.message || '测试失败'
  } finally {
    testing.value = false
  }
}

async function handleAdd() {
  addLoading.value = true
  try {
    if (addMode.value === 'json') {
      // JSON 模式
      let parsed: any
      try {
        parsed = JSON.parse(addJsonText.value)
      } catch (e: any) {
        toast.error('JSON 格式错误：' + e.message)
        return
      }
      if (!parsed.name || !parsed.command) {
        toast.error('JSON 中 name 和 command 为必填项')
        return
      }
      const id = parsed.id || (parsed.name.toLowerCase().replace(/[^a-z0-9]+/g, '-') + '-' + Date.now().toString(36))
      await createMCPServer({
        id,
        name: parsed.name,
        command: parsed.command,
        args: Array.isArray(parsed.args) ? parsed.args : [],
        env: Array.isArray(parsed.env) ? parsed.env : [],
        transport: parsed.transport || 'stdio',
      })
    } else {
      // 表单模式
      if (!addForm.value.name || !addForm.value.command) {
        toast.error('请填写名称和命令')
        return
      }
      const id = addForm.value.name.toLowerCase().replace(/[^a-z0-9]+/g, '-') + '-' + Date.now().toString(36)
      await createMCPServer({
        id,
        name: addForm.value.name,
        command: addForm.value.command,
        args: parseLines(addForm.value.argsText),
        env: parseLines(addForm.value.envText),
        transport: 'stdio',
      })
    }
    toast.success('MCP 服务器已添加')
    resetAddForm()
    await loadAll()
  } catch (err: any) {
    toast.error(err?.message || '添加失败')
  } finally {
    addLoading.value = false
  }
}

// ---- Market template selection ----
function selectTemplate(item: typeof mcpCatalog[0]) {
  viewMode.value = 'installed'
  showAddForm.value = true
  addMode.value = 'form'
  addForm.value.name = item.display
  addForm.value.command = item.command
  addForm.value.argsText = (item.args || []).join('\n')
  addForm.value.envText = ((item as any).env || []).join('\n')
}

// ---- Lifecycle ----
onMounted(loadAll)
</script>

<style scoped>
.mcp-root {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 20px 24px;
  overflow: hidden;
  height: 100%;
}

.mcp-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  flex-shrink: 0;
}

.header-left { display: flex; align-items: baseline; gap: 12px; }
.page-title { font-size: 18px; font-weight: 600; color: var(--text-primary); margin: 0; }
.page-subtitle { font-size: 12px; color: var(--text-tertiary); }

.view-toggle {
  display: flex;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 7px;
  padding: 2px;
  gap: 2px;
}

.toggle-btn {
  padding: 5px 14px;
  border-radius: 5px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}
.toggle-btn.active { background: var(--bg-elevated); color: var(--text-primary); }

/* Content */
.view-content {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* Add Section */
.add-section { flex-shrink: 0; }

.add-toggle-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  background: var(--bg-app);
  border: 1px dashed var(--border);
  border-radius: 7px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}
.add-toggle-btn:hover { border-color: var(--accent); color: var(--accent); }

/* Add Form */
.add-form {
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.form-title { font-size: 13px; font-weight: 600; color: var(--text-primary); }
.icon-close {
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  display: flex;
  align-items: center;
  padding: 2px;
  border-radius: 4px;
}
.icon-close:hover { color: var(--text-primary); }

.form-row-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }

.form-field { display: flex; flex-direction: column; gap: 5px; }
.form-field label { font-size: 11px; font-weight: 500; color: var(--text-secondary); }
.required { color: var(--red); }

.form-input {
  padding: 7px 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 12px;
  outline: none;
}
.form-input:focus { border-color: var(--accent); }

.form-textarea {
  padding: 7px 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 11px;
  outline: none;
  resize: vertical;
  font-family: "SF Mono", Menlo, monospace;
  line-height: 1.5;
}
.form-textarea:focus { border-color: var(--accent); }
.form-textarea::placeholder { color: var(--text-disabled); }

/* Form mode toggle */
.form-mode-toggle {
  display: flex;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 2px;
  gap: 2px;
}
.mode-btn {
  padding: 3px 10px;
  border-radius: 4px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.12s;
}
.mode-btn.active { background: var(--bg-elevated); color: var(--text-primary); }

.form-textarea.json-error { border-color: var(--red); }
.json-error-msg { font-size: 11px; color: var(--red); }

/* Test result */
.test-result {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  padding: 8px 10px;
  background: rgba(34, 197, 94, 0.08);
  border: 1px solid rgba(34, 197, 94, 0.25);
  border-radius: 6px;
  font-size: 12px;
  color: var(--green);
}
.test-result.error {
  background: rgba(239, 68, 68, 0.08);
  border-color: rgba(239, 68, 68, 0.25);
  color: var(--red);
}
.test-icon { flex-shrink: 0; margin-top: 1px; }
.test-icon.success { color: var(--green); }

.form-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.form-spacer { flex: 1; }

/* Loading & Empty */
.loading-hint { font-size: 13px; color: var(--text-tertiary); padding: 20px 0; text-align: center; }

.empty-hint {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-tertiary);
  font-size: 13px;
  padding: 40px 0;
}
.empty-icon { opacity: 0.35; }
.empty-sub { font-size: 12px; }

/* Servers List */
.servers-list {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.server-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  transition: border-color 0.15s;
}
.server-row:hover { border-color: var(--border-hover); }

.srv-left { display: flex; align-items: center; gap: 10px; flex: 1; min-width: 0; }
.srv-icon { color: var(--text-tertiary); flex-shrink: 0; }
.srv-info { display: flex; flex-direction: column; gap: 1px; min-width: 0; }
.srv-name { font-size: 13px; font-weight: 500; color: var(--text-primary); }
.srv-cmd { font-size: 11px; color: var(--text-tertiary); font-family: monospace; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.srv-right {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.srv-badge {
  font-size: 10px;
  padding: 2px 7px;
  border-radius: 10px;
}
.srv-badge.active { background: rgba(34,197,94,0.12); color: var(--green); }
.srv-badge.inactive { background: var(--bg-overlay); color: var(--text-tertiary); }
.srv-badge.builtin { background: var(--accent-dim); color: var(--accent); }
.srv-badge.agents { background: rgba(245,158,11,0.1); color: var(--yellow); }

.icon-btn {
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 5px;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: all 0.12s;
}
.icon-btn:hover:not(:disabled) { background: var(--bg-overlay); color: var(--text-secondary); }
.icon-btn:disabled { opacity: 0.35; cursor: not-allowed; }

/* Market view */
.market-view { gap: 16px; }

.market-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 8px;
  flex-shrink: 0;
}
.search-icon { color: var(--text-tertiary); flex-shrink: 0; }
.search-input {
  flex: 1;
  background: none;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-size: 13px;
}
.search-input::placeholder { color: var(--text-tertiary); }

.catalog-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 10px;
  overflow-y: auto;
}

.catalog-card {
  padding: 14px 16px;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
}
.catalog-card:hover { border-color: var(--accent); background: var(--accent-dim); }

.cat-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 6px;
}
.cat-name { font-size: 13px; font-weight: 600; color: var(--text-primary); flex: 1; }
.cat-tag { font-size: 10px; padding: 1px 6px; background: var(--bg-overlay); border-radius: 8px; color: var(--text-tertiary); }
.cat-cmd { font-size: 11px; color: var(--text-tertiary); font-family: monospace; margin-bottom: 8px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cat-hint { font-size: 10px; color: var(--accent); opacity: 0; transition: opacity 0.15s; }
.catalog-card:hover .cat-hint { opacity: 1; }

/* Confirm Dialog */
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

.btn-primary {
  padding: 7px 16px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
}
.btn-primary:hover:not(:disabled) { opacity: 0.9; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-secondary {
  padding: 7px 14px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s;
}
.btn-secondary:hover:not(:disabled) { background: var(--bg-elevated); }
.btn-secondary:disabled { opacity: 0.5; cursor: not-allowed; }

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
</style>
