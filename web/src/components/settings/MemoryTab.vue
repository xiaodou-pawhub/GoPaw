<template>
  <div class="tab-root">
    <div class="tab-header">
      <h2 class="tab-title">记忆管理</h2>
      <p class="tab-desc">管理 Agent 的结构化记忆、每日笔记和记忆文件</p>
    </div>

    <div class="memory-layout">
      <!-- 左侧导航 -->
      <aside class="mem-sidebar">
        <div class="sidebar-section">
          <div class="sidebar-label">结构化记忆</div>
          <button
            v-for="cat in structuredCategories"
            :key="cat.key"
            class="sidebar-item"
            :class="{ active: selectedView === cat.key }"
            @click="selectView(cat.key)"
          >
            <span>{{ cat.label }}</span>
            <span v-if="stats" class="item-badge">{{ getCatCount(cat.key) }}</span>
          </button>
        </div>
        <div class="sidebar-divider" />
        <div class="sidebar-section">
          <div class="sidebar-label">记忆文件</div>
          <button
            class="sidebar-item"
            :class="{ active: selectedView === 'memory-md' }"
            @click="selectView('memory-md')"
          >📝 MEMORY.md</button>
          <button
            class="sidebar-item"
            :class="{ active: selectedView === 'daily-notes' }"
            @click="selectView('daily-notes')"
          >📅 每日笔记</button>
          <button
            class="sidebar-item"
            :class="{ active: selectedView === 'archives' }"
            @click="selectView('archives')"
          >📦 对话归档</button>
        </div>
      </aside>

      <!-- 结构化记忆视图 -->
      <div v-if="isStructuredView" class="mem-main">
        <div class="mem-header">
          <div class="search-box">
            <SearchIcon :size="12" class="search-icon" />
            <input v-model="searchQuery" placeholder="搜索 key / 内容..." class="search-input" @input="debouncedSearch" />
          </div>
          <div class="mem-actions">
            <button class="btn-secondary btn-sm" @click="showImport = true">导入 MD</button>
            <button class="btn-primary btn-sm" @click="openCreate">+ 新建记忆</button>
          </div>
        </div>

        <div v-if="memLoading" class="mem-loading">加载中...</div>
        <div v-else-if="memories.length === 0" class="mem-empty">
          <p>{{ searchQuery ? '未找到匹配的记忆' : 'Agent 尚未写入任何记忆' }}</p>
        </div>
        <div v-else class="memory-list">
          <div
            v-for="entry in memories"
            :key="entry.id"
            class="memory-card"
            @click="openEdit(entry)"
          >
            <div class="card-header">
              <span class="cat-tag" :class="entry.category">{{ entry.category }}</span>
              <span class="entry-key">{{ entry.key }}</span>
              <div class="card-actions" @click.stop>
                <button class="icon-btn-xs" @click.stop="openEdit(entry)"><PencilIcon :size="11" /></button>
                <button class="icon-btn-xs danger" @click.stop="handleDeleteMemory(entry.key)"><TrashIcon :size="11" /></button>
              </div>
            </div>
            <div class="card-content">{{ truncate(entry.content, 120) }}</div>
            <div class="card-meta">更新于 {{ formatTime(entry.updated_at) }}</div>
          </div>
        </div>
      </div>

      <!-- MEMORY.md 编辑器 -->
      <div v-else-if="selectedView === 'memory-md'" class="mem-main">
        <div class="mem-header">
          <span class="view-title">MEMORY.md</span>
          <button class="btn-primary btn-sm" :disabled="memSaving" @click="saveMemoryMD">{{ memSaving ? '保存中...' : '保存' }}</button>
        </div>
        <textarea v-model="memoryMDContent" class="mem-editor" placeholder="Agent 的跨会话记忆文件..." />
      </div>

      <!-- 每日笔记 -->
      <div v-else-if="selectedView === 'daily-notes'" class="mem-main">
        <div class="mem-header">
          <span class="view-title">每日笔记</span>
          <span class="view-subtitle">{{ todayDate }}</span>
        </div>
        <div v-if="todayNote !== null" class="note-content">{{ todayNote || '今日暂无笔记' }}</div>
        <div class="note-append">
          <textarea v-model="appendNoteText" class="mem-editor small" placeholder="追加笔记..." rows="4" />
          <button class="btn-primary btn-sm" :disabled="appendLoading" @click="handleAppendNote">
            {{ appendLoading ? '追加中...' : '追加' }}
          </button>
        </div>
      </div>

      <!-- 归档 -->
      <div v-else-if="selectedView === 'archives'" class="mem-main">
        <div class="mem-header">
          <span class="view-title">对话归档</span>
          <span class="view-badge">只读</span>
        </div>
        <div v-if="archives.length === 0" class="mem-empty"><p>暂无归档文件</p></div>
        <div v-else class="archive-list">
          <div v-for="arc in archives" :key="arc.name" class="archive-item" @click="loadArchive(arc.name)">
            <span class="arc-name">{{ arc.name }}</span>
            <span class="arc-meta">{{ formatTime(arc.mod_time) }} · {{ formatSize(arc.size) }}</span>
          </div>
        </div>
        <div v-if="archiveContent" class="archive-content">
          <pre>{{ archiveContent }}</pre>
        </div>
      </div>
    </div>

    <!-- 新建/编辑记忆弹窗 -->
    <div v-if="showDrawer" class="modal-overlay" @click.self="showDrawer = false">
      <div class="modal-card">
        <div class="modal-header">
          <h3 class="modal-title">{{ editingEntry ? '编辑记忆' : '新建记忆' }}</h3>
          <button class="icon-close" @click="showDrawer = false"><XIcon :size="16" /></button>
        </div>
        <div class="modal-body">
          <div class="form-field">
            <label>Key（唯一标识）</label>
            <input v-model="form.key" placeholder="例如：user.name、project.tech-stack" class="form-input" :disabled="!!editingEntry" />
          </div>
          <div class="form-field">
            <label>分类</label>
            <select v-model="form.category" class="form-select">
              <option value="core">Core</option>
              <option value="daily">Daily</option>
              <option value="conversation">Conversation</option>
            </select>
          </div>
          <div class="form-field">
            <label>内容</label>
            <textarea v-model="form.content" placeholder="记忆内容，支持 Markdown 格式..." class="form-textarea" rows="6" />
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-secondary" @click="showDrawer = false">取消</button>
          <button class="btn-primary" :disabled="drawerSaving" @click="handleSaveMemory">{{ drawerSaving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>

    <!-- 导入 MD 弹窗 -->
    <div v-if="showImport" class="modal-overlay" @click.self="showImport = false">
      <div class="modal-card">
        <div class="modal-header">
          <h3 class="modal-title">导入 Markdown</h3>
          <button class="icon-close" @click="showImport = false"><XIcon :size="16" /></button>
        </div>
        <div class="modal-body">
          <div class="form-field">
            <label>粘贴 Markdown 内容</label>
            <textarea v-model="importContent" class="form-textarea" rows="8" placeholder="粘贴包含 ## 二级标题的 Markdown，每个标题将作为一条记忆..." />
          </div>
          <div class="form-field">
            <label>导入为分类</label>
            <select v-model="importCategory" class="form-select">
              <option value="core">Core</option>
              <option value="daily">Daily</option>
              <option value="conversation">Conversation</option>
            </select>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-secondary" @click="showImport = false">取消</button>
          <button class="btn-primary" :disabled="importLoading" @click="handleImport">{{ importLoading ? '导入中...' : '确认导入' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { SearchIcon, PencilIcon, TrashIcon, XIcon } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import {
  listMemories, createMemory, updateMemory, deleteMemory, getMemoryStats,
  importMarkdown, getMemoryMD, putMemoryMD,
  appendNote, getNote,
  listArchives, getArchive
} from '@/api/memory'
import type { MemoryEntry, MemoryStats, ArchiveFileInfo } from '@/api/memory'

const selectedView = ref<string>('all')
const stats = ref<MemoryStats | null>(null)

const structuredCategories = [
  { key: 'all', label: '全部' },
  { key: 'core', label: 'Core' },
  { key: 'daily', label: 'Daily' },
  { key: 'conversation', label: 'Conversation' },
]

const isStructuredView = computed(() =>
  ['all', 'core', 'daily', 'conversation'].includes(selectedView.value)
)

function getCatCount(key: string): number {
  if (!stats.value) return 0
  if (key === 'all') return stats.value.total
  return (stats.value as any)[key] ?? 0
}

function selectView(key: string) {
  selectedView.value = key
  if (key === 'memory-md') loadMemoryMD()
  else if (key === 'daily-notes') loadTodayNote()
  else if (key === 'archives') loadArchives()
}

// ===== Structured memories =====
const memories = ref<MemoryEntry[]>([])
const memLoading = ref(false)
const searchQuery = ref('')
let searchTimer: ReturnType<typeof setTimeout> | null = null

function debouncedSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => loadMemories(), 300)
}

async function loadMemories() {
  memLoading.value = true
  try {
    const cat = isStructuredView.value && selectedView.value !== 'all' ? selectedView.value : undefined
    const res = await listMemories({ category: cat, q: searchQuery.value || undefined })
    memories.value = res.memories || []
  } catch { toast.error('加载失败') }
  finally { memLoading.value = false }
}

async function loadStats() {
  try {
    const res = await getMemoryStats()
    // @ts-ignore
    stats.value = res.stats
  } catch {}
}

watch(selectedView, (v) => {
  if (['all', 'core', 'daily', 'conversation'].includes(v)) loadMemories()
})

// ===== Create/Edit memory =====
const showDrawer = ref(false)
const drawerSaving = ref(false)
const editingEntry = ref<MemoryEntry | null>(null)
const form = ref({ key: '', content: '', category: 'core' as string })

function openCreate() {
  editingEntry.value = null
  form.value = { key: '', content: '', category: 'core' }
  showDrawer.value = true
}

function openEdit(entry: MemoryEntry) {
  editingEntry.value = entry
  form.value = { key: entry.key, content: entry.content, category: entry.category }
  showDrawer.value = true
}

async function handleSaveMemory() {
  if (!form.value.key || !form.value.content) {
    toast.error('请填写 Key 和内容')
    return
  }
  drawerSaving.value = true
  try {
    if (editingEntry.value) {
      await updateMemory(form.value.key, { content: form.value.content, category: form.value.category })
      toast.success('记忆已更新')
    } else {
      await createMemory({ key: form.value.key, content: form.value.content, category: form.value.category })
      toast.success('记忆已创建')
    }
    showDrawer.value = false
    loadMemories()
    loadStats()
  } catch {
    toast.error('保存失败')
  } finally {
    drawerSaving.value = false
  }
}

async function handleDeleteMemory(key: string) {
  if (!confirm(`删除记忆 "${key}" 后无法恢复，是否继续？`)) return
  try {
    await deleteMemory(key)
    toast.success('已删除')
    loadMemories()
    loadStats()
  } catch {
    toast.error('删除失败')
  }
}

// ===== MEMORY.md =====
const memoryMDContent = ref('')
const memSaving = ref(false)

async function loadMemoryMD() {
  try {
    const res = await getMemoryMD()
    memoryMDContent.value = res.content || ''
  } catch {}
}

async function saveMemoryMD() {
  memSaving.value = true
  try {
    await putMemoryMD(memoryMDContent.value)
    toast.success('已保存')
  } catch {
    toast.error('保存失败')
  } finally {
    memSaving.value = false
  }
}

// ===== Daily notes =====
const todayNote = ref<string | null>(null)
const appendNoteText = ref('')
const appendLoading = ref(false)
const todayDate = new Date().toISOString().slice(0, 10)
const todayDateKey = todayDate.replace(/-/g, '')

async function loadTodayNote() {
  try {
    const res = await getNote(todayDateKey)
    todayNote.value = res.content
  } catch {
    todayNote.value = ''
  }
}

async function handleAppendNote() {
  if (!appendNoteText.value.trim()) return
  appendLoading.value = true
  try {
    await appendNote(todayDateKey, appendNoteText.value.trim())
    appendNoteText.value = ''
    toast.success('已追加')
    loadTodayNote()
  } catch {
    toast.error('追加失败')
  } finally {
    appendLoading.value = false
  }
}

// ===== Archives =====
const archives = ref<ArchiveFileInfo[]>([])
const archiveContent = ref('')

async function loadArchives() {
  try {
    const res = await listArchives()
    archives.value = res.archives || []
  } catch {}
}

async function loadArchive(name: string) {
  try {
    const res = await getArchive(name)
    archiveContent.value = res.content
  } catch {
    toast.error('加载归档失败')
  }
}

// ===== Import =====
const showImport = ref(false)
const importContent = ref('')
const importCategory = ref('core')
const importLoading = ref(false)

async function handleImport() {
  if (!importContent.value.trim()) { toast.error('内容不能为空'); return }
  importLoading.value = true
  try {
    const res = await importMarkdown({ content: importContent.value, category: importCategory.value })
    toast.success(`成功导入 ${res.imported} 条记忆`)
    showImport.value = false
    importContent.value = ''
    loadMemories()
    loadStats()
  } catch {
    toast.error('导入失败')
  } finally {
    importLoading.value = false
  }
}

// ===== Helpers =====
function truncate(text: string, len: number): string {
  return text.length > len ? text.slice(0, len) + '…' : text
}

function formatTime(ts: number): string {
  return new Date(ts * 1000).toLocaleDateString()
}

function formatSize(bytes: number): string {
  return bytes > 1024 ? `${(bytes / 1024).toFixed(1)}KB` : `${bytes}B`
}

onMounted(() => {
  loadMemories()
  loadStats()
})
</script>

<style scoped>
.tab-root { display: flex; flex-direction: column; gap: 16px; }
.tab-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin: 0 0 4px; }
.tab-desc { font-size: 12px; color: var(--text-secondary); margin: 0; }

.memory-layout {
  display: flex;
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
  min-height: 450px;
  background: var(--bg-elevated);
}

.mem-sidebar {
  width: 180px;
  background: var(--bg-panel);
  border-right: 1px solid var(--border-subtle);
  padding: 10px 0;
  flex-shrink: 0;
}

.sidebar-section { padding: 0 8px; }
.sidebar-label { font-size: 10px; font-weight: 600; color: var(--text-tertiary); text-transform: uppercase; letter-spacing: 0.06em; padding: 8px 6px 4px; }
.sidebar-divider { height: 1px; background: var(--border); margin: 6px 8px; }

.sidebar-item {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 6px 8px;
  border: none;
  background: transparent;
  border-radius: 5px;
  cursor: pointer;
  color: var(--text-secondary);
  font-size: 12px;
  text-align: left;
  transition: background 0.12s;
}
.sidebar-item:hover { background: var(--bg-overlay); }
.sidebar-item.active { background: var(--accent-dim); color: var(--accent); }

.item-badge { margin-left: auto; font-size: 10px; color: var(--text-tertiary); background: var(--bg-elevated); border-radius: 8px; padding: 1px 5px; }

.mem-main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }

.mem-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.search-box {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 5px 10px;
}

.search-icon { color: var(--text-tertiary); flex-shrink: 0; }

.search-input {
  flex: 1;
  background: none;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-size: 12px;
}

.search-input::placeholder { color: var(--text-tertiary); }

.mem-actions { display: flex; gap: 6px; flex-shrink: 0; }

.mem-loading, .mem-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
  font-size: 13px;
}

.memory-list {
  flex: 1;
  overflow-y: auto;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.memory-card {
  background: var(--bg-panel);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  padding: 10px 12px;
  cursor: pointer;
  transition: border-color 0.12s;
}

.memory-card:hover { border-color: rgba(124, 106, 247, 0.3); }

.card-header { display: flex; align-items: center; gap: 6px; margin-bottom: 5px; }
.cat-tag { font-size: 10px; padding: 1px 6px; border-radius: 3px; font-weight: 600; text-transform: uppercase; }
.cat-tag.core { background: rgba(124,106,247,0.15); color: var(--accent); }
.cat-tag.daily { background: rgba(34,197,94,0.1); color: var(--green); }
.cat-tag.conversation { background: rgba(245,158,11,0.1); color: var(--yellow); }

.entry-key { font-size: 12px; font-weight: 600; color: var(--text-primary); font-family: monospace; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.card-actions { display: none; gap: 2px; }
.memory-card:hover .card-actions { display: flex; }

.icon-btn-xs { width: 20px; height: 20px; display: flex; align-items: center; justify-content: center; background: transparent; border: none; border-radius: 4px; color: var(--text-tertiary); cursor: pointer; }
.icon-btn-xs:hover { background: var(--bg-overlay); color: var(--text-secondary); }
.icon-btn-xs.danger:hover { color: var(--red); }

.card-content { font-size: 12px; color: var(--text-secondary); line-height: 1.5; margin-bottom: 4px; }
.card-meta { font-size: 11px; color: var(--text-disabled); }

.view-title { font-size: 13px; font-weight: 600; color: var(--text-primary); }
.view-subtitle { font-size: 11px; color: var(--text-tertiary); }
.view-badge { font-size: 10px; padding: 2px 6px; background: var(--bg-overlay); border: 1px solid var(--border); border-radius: 4px; color: var(--text-tertiary); }

.mem-editor {
  flex: 1;
  padding: 14px 16px;
  background: transparent;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-family: "SF Mono", Menlo, monospace;
  font-size: 12px;
  line-height: 1.7;
  resize: none;
  min-height: 280px;
  width: 100%;
  box-sizing: border-box;
}

.mem-editor.small { min-height: 80px; }
.mem-editor::placeholder { color: var(--text-disabled); }

.note-content {
  flex: 1;
  padding: 14px 16px;
  font-family: "SF Mono", Menlo, monospace;
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.7;
  overflow-y: auto;
  border-bottom: 1px solid var(--border-subtle);
  white-space: pre-wrap;
}

.note-append { padding: 10px 14px; display: flex; flex-direction: column; gap: 8px; }

.archive-list { flex: 1; overflow-y: auto; padding: 8px 12px; display: flex; flex-direction: column; gap: 4px; }

.archive-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  background: var(--bg-panel);
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
  cursor: pointer;
  transition: border-color 0.12s;
}

.archive-item:hover { border-color: rgba(124, 106, 247, 0.3); }
.arc-name { font-size: 12px; color: var(--text-primary); font-family: monospace; }
.arc-meta { font-size: 11px; color: var(--text-tertiary); }

.archive-content {
  border-top: 1px solid var(--border-subtle);
  max-height: 200px;
  overflow-y: auto;
  padding: 10px 14px;
}

.archive-content pre { font-size: 11px; color: var(--text-secondary); white-space: pre-wrap; word-break: break-all; margin: 0; }

/* Modal */
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.65); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal-card { width: 520px; max-height: 90vh; background: var(--bg-elevated); border: 1px solid var(--border); border-radius: 12px; display: flex; flex-direction: column; overflow: hidden; }
.modal-header { display: flex; align-items: center; justify-content: space-between; padding: 16px 20px; border-bottom: 1px solid var(--border-subtle); }
.modal-title { font-size: 14px; font-weight: 600; color: var(--text-primary); margin: 0; }
.icon-close { background: transparent; border: none; color: var(--text-tertiary); cursor: pointer; display: flex; align-items: center; padding: 4px; border-radius: 4px; }
.icon-close:hover { color: var(--text-primary); background: var(--bg-overlay); }
.modal-body { flex: 1; overflow-y: auto; padding: 16px 20px; display: flex; flex-direction: column; gap: 14px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 20px; border-top: 1px solid var(--border-subtle); }

.form-field { display: flex; flex-direction: column; gap: 5px; }
.form-field label { font-size: 12px; font-weight: 500; color: var(--text-secondary); }

.form-input, .form-select, .form-textarea {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  box-sizing: border-box;
  transition: border-color 0.15s;
  font-family: inherit;
}

.form-textarea { resize: vertical; min-height: 100px; font-family: "SF Mono", Menlo, monospace; font-size: 12px; }
.form-input:focus, .form-select:focus, .form-textarea:focus { border-color: var(--accent); }
.form-input:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-primary { display: flex; align-items: center; gap: 5px; padding: 6px 12px; background: var(--accent); border: none; border-radius: 6px; color: #fff; font-size: 12px; font-weight: 500; cursor: pointer; transition: background 0.15s; white-space: nowrap; }
.btn-primary:hover { background: var(--accent-hover); }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-secondary { padding: 6px 12px; background: var(--bg-overlay); border: 1px solid var(--border); border-radius: 6px; color: var(--text-secondary); font-size: 12px; cursor: pointer; transition: background 0.15s; white-space: nowrap; }
.btn-secondary:hover { background: var(--bg-elevated); }

.btn-sm { padding: 5px 10px; font-size: 11px; }
</style>
