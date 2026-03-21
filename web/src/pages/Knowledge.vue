<template>
  <div class="knowledge-page">
    <div class="page-header">
      <h1 class="page-title">知识库</h1>
      <button class="btn-primary" @click="openCreateDialog">
        <PlusIcon :size="16" />
        新建知识库
      </button>
    </div>

    <div class="kb-layout">
      <!-- 左侧：知识库列表 -->
      <div class="kb-sidebar">
        <div class="sidebar-title">知识库列表</div>
        <div
          v-for="kb in knowledgeBases"
          :key="kb.id"
          class="kb-item"
          :class="{ active: selectedKB?.id === kb.id }"
          @click="selectKB(kb)"
        >
          <div class="kb-name">{{ kb.name }}</div>
          <div class="kb-meta">{{ kb.document_count }} 文档 · {{ kb.chunk_count }} 块</div>
        </div>
        <div v-if="knowledgeBases.length === 0" class="empty-state">暂无知识库</div>
      </div>

      <!-- 右侧：详情 -->
      <div class="kb-detail" v-if="selectedKB">
        <div class="detail-header">
          <h2 class="detail-title">{{ selectedKB.name }}</h2>
          <div class="detail-actions">
            <button class="btn-ghost" @click="openEditDialog">
              <PencilIcon :size="14" /> 编辑
            </button>
            <button class="btn-danger-outline" @click="confirmDelete">
              <Trash2Icon :size="14" /> 删除
            </button>
          </div>
        </div>

        <p class="kb-desc">{{ selectedKB.description }}</p>

        <!-- 统计卡片 -->
        <div class="stat-cards">
          <div class="stat-card">
            <div class="stat-value">{{ stats?.document_count || 0 }}</div>
            <div class="stat-label">文档</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">{{ stats?.chunk_count || 0 }}</div>
            <div class="stat-label">文本块</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">{{ formatNumber(stats?.total_tokens || 0) }}</div>
            <div class="stat-label">Token</div>
          </div>
          <div class="stat-card">
            <div class="stat-value" :class="getProcessedClass(stats)">
              {{ stats?.completed_count || 0 }}/{{ stats?.document_count || 0 }}
            </div>
            <div class="stat-label">已处理</div>
          </div>
        </div>

        <!-- Tab 切换 -->
        <div class="tab-bar">
          <button class="tab-btn" :class="{ active: activeTab === 'documents' }" @click="activeTab = 'documents'">文档</button>
          <button class="tab-btn" :class="{ active: activeTab === 'search' }" @click="activeTab = 'search'">搜索测试</button>
        </div>

        <!-- 文档列表 -->
        <div v-if="activeTab === 'documents'" class="tab-content">
          <div class="tab-toolbar">
            <button class="btn-ghost" @click="openUploadDialog">
              <UploadIcon :size="14" /> 上传文档
            </button>
          </div>
          <div class="data-table">
            <div class="data-thead">
              <span>文件名</span><span>类型</span><span>大小</span><span>状态</span><span>块数</span><span>上传时间</span><span>操作</span>
            </div>
            <div v-if="loadingDocs" class="empty-state">加载中...</div>
            <div v-else-if="documents.length === 0" class="empty-state">暂无文档，请上传</div>
            <div v-for="doc in documents" :key="doc.id" class="data-row">
              <span class="doc-name">{{ doc.filename }}</span>
              <span class="text-sm">{{ doc.file_type }}</span>
              <span class="text-sm">{{ formatFileSize(doc.file_size) }}</span>
              <span>
                <span class="badge" :class="getDocStatusClass(doc.status)">{{ doc.status }}</span>
              </span>
              <span class="text-sm">{{ doc.chunk_count }}</span>
              <span class="text-sm">{{ formatDate(doc.created_at) }}</span>
              <span class="actions">
                <button
                  v-if="doc.status === 'failed'"
                  class="action-btn action-warning"
                  title="重试"
                  @click="retryDocument(doc)"
                >
                  <RefreshCwIcon :size="13" />
                </button>
                <button
                  class="action-btn action-danger"
                  title="删除"
                  @click="deleteDocument(doc)"
                >
                  <Trash2Icon :size="13" />
                </button>
              </span>
            </div>
          </div>
        </div>

        <!-- 搜索测试 -->
        <div v-if="activeTab === 'search'" class="tab-content">
          <div class="search-bar">
            <input
              v-model="searchQuery"
              type="text"
              placeholder="输入查询..."
              class="search-input"
              @keyup.enter="performSearch"
            />
            <select v-model="searchType" class="search-type">
              <option value="vector">向量搜索</option>
              <option value="fulltext">全文搜索</option>
              <option value="hybrid">混合搜索</option>
            </select>
            <button class="btn-primary" @click="performSearch">
              <SearchIcon :size="14" /> 搜索
            </button>
          </div>

          <div v-if="searchResults.length > 0" class="search-results">
            <div v-for="(result, index) in searchResults" :key="result.chunk_id" class="result-item">
              <div class="result-rank">{{ index + 1 }}</div>
              <div class="result-content">
                <div class="result-text">{{ result.content.substring(0, 200) }}...</div>
                <div class="result-meta">
                  来自: {{ result.document_name }} ·
                  相似度: {{ ((1 - result.distance) * 100).toFixed(1) }}%
                </div>
              </div>
            </div>
          </div>
          <div v-else-if="hasSearched" class="empty-state">未找到相关结果</div>
        </div>
      </div>

      <div v-else class="kb-empty">
        <BookOpenIcon :size="48" />
        <p>请选择或创建一个知识库</p>
      </div>
    </div>

    <!-- 创建/编辑知识库弹窗 -->
    <div v-if="dialog.show" class="modal-overlay" @click.self="dialog.show = false">
      <div class="modal-card">
        <h2 class="modal-title">{{ dialog.isEdit ? '编辑知识库' : '新建知识库' }}</h2>
        <form @submit.prevent="saveKnowledgeBase">
          <div class="form-group">
            <label>ID</label>
            <input v-model="dialog.data.id" type="text" :disabled="dialog.isEdit" required />
          </div>
          <div class="form-group">
            <label>名称</label>
            <input v-model="dialog.data.name" type="text" required />
          </div>
          <div class="form-group">
            <label>描述</label>
            <textarea v-model="dialog.data.description" rows="2" />
          </div>
          <div class="form-group">
            <label>Embedding 模型</label>
            <select v-model="dialog.data.embedding_model">
              <option value="nomic-embed-text">nomic-embed-text (本地)</option>
              <option value="mxbai-embed-large">mxbai-embed-large (本地)</option>
              <option value="text-embedding-3-small">text-embedding-3-small (OpenAI)</option>
            </select>
          </div>
          <div class="form-row">
            <div class="form-group">
              <label>分块大小 ({{ dialog.data.chunk_size }})</label>
              <input v-model.number="dialog.data.chunk_size" type="range" min="100" max="2000" step="100" />
            </div>
            <div class="form-group">
              <label>分块重叠 ({{ dialog.data.chunk_overlap }})</label>
              <input v-model.number="dialog.data.chunk_overlap" type="range" min="0" max="200" step="10" />
            </div>
          </div>
          <div class="modal-actions">
            <button type="button" class="btn-ghost" @click="dialog.show = false">取消</button>
            <button type="submit" class="btn-primary">保存</button>
          </div>
        </form>
      </div>
    </div>

    <!-- 上传文档弹窗 -->
    <div v-if="uploadDialog.show" class="modal-overlay" @click.self="uploadDialog.show = false">
      <div class="modal-card modal-sm">
        <h2 class="modal-title">上传文档</h2>
        <div class="form-group">
          <label>选择文件</label>
          <input
            type="file"
            accept=".pdf,.md,.txt,.doc,.docx"
            class="file-input"
            @change="onFileChange"
          />
        </div>
        <div class="form-group">
          <label>文件类型（可选，自动检测）</label>
          <select v-model="uploadDialog.fileType">
            <option value="">自动检测</option>
            <option value="pdf">PDF</option>
            <option value="md">Markdown</option>
            <option value="txt">纯文本</option>
          </select>
        </div>
        <div class="modal-actions">
          <button class="btn-ghost" @click="uploadDialog.show = false">取消</button>
          <button class="btn-primary" :disabled="!uploadDialog.file" @click="uploadDocument">上传</button>
        </div>
      </div>
    </div>

    <!-- 删除确认弹窗 -->
    <div v-if="deleteDialog.show" class="modal-overlay" @click.self="deleteDialog.show = false">
      <div class="modal-card modal-sm">
        <h2 class="modal-title">确认删除</h2>
        <p class="confirm-text">
          确定要删除知识库 "{{ deleteDialog.kb?.name }}" 吗？<br />
          此操作将删除所有相关文档和文本块，不可恢复。
        </p>
        <div class="modal-actions">
          <button class="btn-ghost" @click="deleteDialog.show = false">取消</button>
          <button class="btn-danger" @click="deleteKnowledgeBase">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import {
  PlusIcon, PencilIcon, Trash2Icon, UploadIcon,
  RefreshCwIcon, SearchIcon, BookOpenIcon,
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { knowledgeApi, type KnowledgeBase, type Document, type SearchResult } from '@/api/knowledge'

const knowledgeBases = ref<KnowledgeBase[]>([])
const selectedKB = ref<KnowledgeBase | null>(null)
const documents = ref<Document[]>([])
const stats = ref<Record<string, number> | null>(null)
const loadingDocs = ref(false)
const activeTab = ref('documents')

const dialog = reactive({
  show: false,
  isEdit: false,
  data: {
    id: '', name: '', description: '',
    embedding_model: 'nomic-embed-text', chunk_size: 500, chunk_overlap: 50,
  },
})

const uploadDialog = reactive({ show: false, file: null as File | null, fileType: '' })

const deleteDialog = reactive({ show: false, kb: null as KnowledgeBase | null })

const searchQuery = ref('')
const searchType = ref('vector')
const searchResults = ref<SearchResult[]>([])
const hasSearched = ref(false)

onMounted(() => { loadKnowledgeBases() })

async function loadKnowledgeBases() {
  try {
    knowledgeBases.value = await knowledgeApi.listBases()
  } catch {
    toast.error('加载知识库列表失败')
  }
}

async function selectKB(kb: KnowledgeBase) {
  selectedKB.value = kb
  activeTab.value = 'documents'
  loadDocuments(kb.id)
  loadStats(kb.id)
}

async function loadDocuments(kbId: string) {
  loadingDocs.value = true
  try {
    documents.value = await knowledgeApi.listDocuments(kbId)
  } catch {
    toast.error('加载文档失败')
  } finally {
    loadingDocs.value = false
  }
}

async function loadStats(kbId: string) {
  try {
    stats.value = await knowledgeApi.getStats(kbId)
  } catch {
    // 忽略
  }
}

function openCreateDialog() {
  dialog.isEdit = false
  dialog.data = { id: '', name: '', description: '', embedding_model: 'nomic-embed-text', chunk_size: 500, chunk_overlap: 50 }
  dialog.show = true
}

function openEditDialog() {
  if (!selectedKB.value) return
  dialog.isEdit = true
  dialog.data = {
    id: selectedKB.value.id, name: selectedKB.value.name,
    description: selectedKB.value.description,
    embedding_model: selectedKB.value.embedding_model,
    chunk_size: selectedKB.value.chunk_size, chunk_overlap: selectedKB.value.chunk_overlap,
  }
  dialog.show = true
}

async function saveKnowledgeBase() {
  try {
    if (dialog.isEdit) {
      await knowledgeApi.updateBase(dialog.data.id, { name: dialog.data.name, description: dialog.data.description })
    } else {
      await knowledgeApi.createBase(dialog.data)
    }
    dialog.show = false
    loadKnowledgeBases()
  } catch {
    toast.error('保存失败')
  }
}

function confirmDelete() {
  if (!selectedKB.value) return
  deleteDialog.kb = selectedKB.value
  deleteDialog.show = true
}

async function deleteKnowledgeBase() {
  if (!deleteDialog.kb) return
  try {
    await knowledgeApi.deleteBase(deleteDialog.kb.id)
    deleteDialog.show = false
    selectedKB.value = null
    loadKnowledgeBases()
  } catch {
    toast.error('删除失败')
  }
}

function openUploadDialog() {
  uploadDialog.file = null
  uploadDialog.fileType = ''
  uploadDialog.show = true
}

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  uploadDialog.file = input.files?.[0] || null
}

async function uploadDocument() {
  if (!uploadDialog.file || !selectedKB.value) return
  try {
    await knowledgeApi.uploadDocument(selectedKB.value.id, uploadDialog.file, uploadDialog.fileType)
    toast.success('文档上传成功')
    uploadDialog.show = false
    loadDocuments(selectedKB.value.id)
    loadStats(selectedKB.value.id)
  } catch {
    toast.error('上传失败')
  }
}

async function deleteDocument(doc: Document) {
  if (!selectedKB.value) return
  try {
    await knowledgeApi.deleteDocument(selectedKB.value.id, doc.id)
    loadDocuments(selectedKB.value.id)
    loadStats(selectedKB.value.id)
  } catch {
    toast.error('删除文档失败')
  }
}

async function retryDocument(doc: Document) {
  if (!selectedKB.value) return
  try {
    await knowledgeApi.retryDocument(selectedKB.value.id, doc.id)
    setTimeout(() => loadDocuments(selectedKB.value!.id), 1000)
  } catch {
    toast.error('重试失败')
  }
}

async function performSearch() {
  if (!selectedKB.value || !searchQuery.value) return
  try {
    const response = await knowledgeApi.search(selectedKB.value.id, {
      query: searchQuery.value, top_k: 5, search_type: searchType.value as 'vector' | 'fulltext' | 'hybrid',
    })
    searchResults.value = response.results
    hasSearched.value = true
  } catch {
    toast.error('搜索失败')
  }
}

function getDocStatusClass(status: string) {
  const map: Record<string, string> = { completed: 'badge-success', processing: 'badge-warning', failed: 'badge-error' }
  return map[status] || 'badge-neutral'
}

function getProcessedClass(s: Record<string, number> | null) {
  if (!s) return ''
  if (s.failed_count > 0) return 'text-error'
  if (s.pending_count > 0) return 'text-warning'
  return 'text-success'
}

function formatFileSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function formatNumber(num: number) {
  if (num >= 10000) return (num / 10000).toFixed(1) + 'w'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'k'
  return num.toString()
}

function formatDate(date: string) {
  return new Date(date).toLocaleString('zh-CN')
}
</script>

<style scoped>
.knowledge-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg-app);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-panel);
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-primary:hover { background: var(--accent-hover); }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-ghost {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 12px;
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
}

.btn-ghost:hover { background: var(--bg-overlay); }

.btn-danger-outline {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 12px;
  background: transparent;
  color: #ef4444;
  border: 1px solid rgba(239,68,68,0.4);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
}

.btn-danger-outline:hover { background: rgba(239,68,68,0.08); }

.btn-danger {
  padding: 8px 16px;
  background: #ef4444;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-danger:hover { background: #dc2626; }

.kb-layout {
  flex: 1;
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 16px;
  padding: 24px;
  overflow: hidden;
}

.kb-sidebar {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.sidebar-title {
  padding: 12px 16px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--border);
  background: var(--bg-overlay);
}

.kb-item {
  padding: 12px 16px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-subtle);
  transition: background 0.1s;
}

.kb-item:hover { background: var(--bg-overlay); }
.kb-item.active { background: var(--accent-dim); }

.kb-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.kb-item.active .kb-name { color: var(--accent); }

.kb-meta {
  font-size: 11px;
  color: var(--text-tertiary);
  margin-top: 2px;
}

.kb-detail {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 20px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.kb-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-tertiary);
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 48px;
}

.kb-empty p { font-size: 14px; margin: 0; }

.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.detail-title {
  font-size: 17px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.detail-actions { display: flex; gap: 8px; }

.kb-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0 0 16px;
}

.stat-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 20px;
}

.stat-card {
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 14px;
  text-align: center;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
}

.text-success { color: #16a34a; }
.text-warning { color: #ca8a04; }
.text-error   { color: #ef4444; }

.stat-label {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-top: 4px;
}

/* Tab */
.tab-bar {
  display: flex;
  gap: 4px;
  margin-bottom: 16px;
  border-bottom: 1px solid var(--border);
  padding-bottom: 0;
}

.tab-btn {
  padding: 8px 16px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 14px;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  transition: all 0.15s;
}

.tab-btn:hover { color: var(--text-primary); }
.tab-btn.active { color: var(--accent); border-bottom-color: var(--accent); }

.tab-content {
  flex: 1;
  overflow-y: auto;
}

.tab-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 12px;
}

/* Data table */
.data-table {
  overflow-x: auto;
  flex: 1;
}

.data-thead,
.data-row {
  display: grid;
  grid-template-columns: 1fr 60px 80px 90px 50px 140px 80px;
  padding: 10px 0;
  align-items: center;
  gap: 8px;
  border-bottom: 1px solid var(--border-subtle);
  min-width: 600px;
}

.data-thead {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--border);
  background: var(--bg-overlay);
  padding: 10px 12px;
  margin: 0 -12px;
}

.data-row {
  font-size: 13px;
  color: var(--text-primary);
}

.data-row:last-child { border-bottom: none; }

.doc-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.text-sm { font-size: 12px; color: var(--text-secondary); }

.badge {
  display: inline-block;
  padding: 2px 7px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.badge-success { background: rgba(34,197,94,0.15);  color: #16a34a; }
.badge-warning { background: rgba(234,179,8,0.15);  color: #ca8a04; }
.badge-error   { background: rgba(239,68,68,0.15);  color: #ef4444; }
.badge-neutral { background: var(--bg-overlay); color: var(--text-secondary); border: 1px solid var(--border); }

.actions { display: flex; gap: 4px; }

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all 0.1s;
}

.action-btn:hover { background: var(--bg-overlay); }
.action-warning:hover { background: rgba(234,179,8,0.1); color: #ca8a04; border-color: rgba(234,179,8,0.3); }
.action-danger:hover  { background: rgba(239,68,68,0.1); color: #ef4444; border-color: rgba(239,68,68,0.3); }

/* Search */
.search-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.search-input {
  flex: 1;
  padding: 8px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
}

.search-type {
  padding: 8px 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
}

.result-item {
  display: flex;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border-subtle);
}

.result-item:last-child { border-bottom: none; }

.result-rank {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--accent-dim);
  color: var(--accent);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  flex-shrink: 0;
}

.result-text {
  font-size: 13px;
  color: var(--text-primary);
  margin-bottom: 4px;
  line-height: 1.5;
}

.result-meta {
  font-size: 12px;
  color: var(--text-tertiary);
}

.empty-state {
  padding: 24px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 13px;
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.65);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 24px;
  width: 500px;
  max-width: 95vw;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.modal-sm { width: 380px; }

.modal-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 20px;
}

.confirm-text {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.6;
  margin: 0 0 20px;
}

.form-group {
  margin-bottom: 14px;
}

.form-group label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  box-sizing: border-box;
  outline: none;
  transition: border-color 0.15s;
}

.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
  border-color: var(--accent);
}

.form-group input[type="range"] {
  padding: 4px 0;
  border: none;
  background: transparent;
}

.form-group input:disabled { opacity: 0.6; cursor: not-allowed; }

.file-input {
  padding: 4px 0 !important;
  border: none !important;
  background: transparent !important;
}

.form-group textarea { resize: vertical; }

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--border-subtle);
}
</style>
