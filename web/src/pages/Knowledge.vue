<template>
  <div class="knowledge-page">
    <div class="page-header">
      <h1 class="text-h5">知识库</h1>
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreateDialog">
        新建知识库
      </v-btn>
    </div>

    <v-row class="mt-4">
      <!-- 知识库列表 -->
      <v-col cols="3">
        <v-card>
          <v-card-title>知识库列表</v-card-title>
          <v-list density="compact">
            <v-list-item
              v-for="kb in knowledgeBases"
              :key="kb.id"
              :active="selectedKB?.id === kb.id"
              @click="selectKB(kb)"
            >
              <v-list-item-title>{{ kb.name }}</v-list-item-title>
              <v-list-item-subtitle>
                {{ kb.document_count }} 文档 · {{ kb.chunk_count }} 块
              </v-list-item-subtitle>
            </v-list-item>
          </v-list>
        </v-card>
      </v-col>

      <!-- 知识库详情 -->
      <v-col cols="9" v-if="selectedKB">
        <v-card>
          <v-card-title class="d-flex align-center">
            <span>{{ selectedKB.name }}</span>
            <v-spacer />
            <v-btn
              icon="mdi-pencil"
              size="small"
              @click="openEditDialog"
            />
            <v-btn
              icon="mdi-delete"
              size="small"
              color="error"
              @click="confirmDelete"
            />
          </v-card-title>
          
          <v-card-text>
            <p class="text-body-2 text-grey">{{ selectedKB.description }}</p>
            
            <!-- 统计信息 -->
            <v-row class="mt-4">
              <v-col cols="3">
                <v-card variant="outlined">
                  <v-card-text class="text-center">
                    <div class="text-h4">{{ stats?.document_count || 0 }}</div>
                    <div class="text-caption">文档</div>
                  </v-card-text>
                </v-card>
              </v-col>
              <v-col cols="3">
                <v-card variant="outlined">
                  <v-card-text class="text-center">
                    <div class="text-h4">{{ stats?.chunk_count || 0 }}</div>
                    <div class="text-caption">文本块</div>
                  </v-card-text>
                </v-card>
              </v-col>
              <v-col cols="3">
                <v-card variant="outlined">
                  <v-card-text class="text-center">
                    <div class="text-h4">{{ formatNumber(stats?.total_tokens || 0) }}</div>
                    <div class="text-caption">Token</div>
                  </v-card-text>
                </v-card>
              </v-col>
              <v-col cols="3">
                <v-card variant="outlined">
                  <v-card-text class="text-center">
                    <div class="text-h4" :class="getStatusColor(stats)">
                      {{ stats?.completed_count || 0 }}/{{ stats?.document_count || 0 }}
                    </div>
                    <div class="text-caption">已处理</div>
                  </v-card-text>
                </v-card>
              </v-col>
            </v-row>

            <v-tabs v-model="activeTab" class="mt-4">
              <v-tab value="documents">文档</v-tab>
              <v-tab value="search">搜索测试</v-tab>
            </v-tabs>

            <v-window v-model="activeTab">
              <!-- 文档列表 -->
              <v-window-item value="documents">
                <div class="d-flex justify-end mb-4">
                  <v-btn prepend-icon="mdi-upload" @click="openUploadDialog">
                    上传文档
                  </v-btn>
                </div>
                
                <v-data-table
                  :headers="docHeaders"
                  :items="documents"
                  :loading="loadingDocs"
                  density="compact"
                >
                  <template #item.status="{ item }">
                    <v-chip :color="getDocStatusColor(item.status)" size="small">
                      {{ item.status }}
                    </v-chip>
                  </template>
                  <template #item.file_size="{ item }">
                    {{ formatFileSize(item.file_size) }}
                  </template>
                  <template #item.created_at="{ item }">
                    {{ formatDate(item.created_at) }}
                  </template>
                  <template #item.actions="{ item }">
                    <v-btn
                      v-if="item.status === 'failed'"
                      icon="mdi-refresh"
                      size="x-small"
                      @click="retryDocument(item)"
                    />
                    <v-btn
                      icon="mdi-delete"
                      size="x-small"
                      color="error"
                      @click="deleteDocument(item)"
                    />
                  </template>
                </v-data-table>
              </v-window-item>

              <!-- 搜索测试 -->
              <v-window-item value="search">
                <v-row class="mt-4">
                  <v-col cols="8">
                    <v-text-field
                      v-model="searchQuery"
                      label="输入查询"
                      append-inner-icon="mdi-magnify"
                      @click:append-inner="performSearch"
                      @keyup.enter="performSearch"
                    />
                  </v-col>
                  <v-col cols="4">
                    <v-select
                      v-model="searchType"
                      :items="searchTypes"
                      label="搜索类型"
                    />
                  </v-col>
                </v-row>

                <v-list v-if="searchResults.length > 0">
                  <v-list-item
                    v-for="(result, index) in searchResults"
                    :key="result.chunk_id"
                  >
                    <template #prepend>
                      <v-avatar color="primary" size="32">
                        {{ index + 1 }}
                      </v-avatar>
                    </template>
                    <v-list-item-title>
                      {{ result.content.substring(0, 150) }}...
                    </v-list-item-title>
                    <v-list-item-subtitle>
                      来自: {{ result.document_name }} | 
                      相似度: {{ ((1 - result.distance) * 100).toFixed(1) }}%
                    </v-list-item-subtitle>
                  </v-list-item>
                </v-list>

                <v-alert v-else-if="hasSearched" type="info" class="mt-4">
                  未找到相关结果
                </v-alert>
              </v-window-item>
            </v-window>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- 创建/编辑对话框 -->
    <v-dialog v-model="dialog.show" max-width="600">
      <v-card>
        <v-card-title>{{ dialog.isEdit ? '编辑知识库' : '新建知识库' }}</v-card-title>
        <v-card-text>
          <v-form ref="form" v-model="dialog.valid">
            <v-text-field
              v-model="dialog.data.id"
              label="ID"
              :disabled="dialog.isEdit"
              :rules="[(v: string) => !!v || 'ID 不能为空']"
              required
            />
            <v-text-field
              v-model="dialog.data.name"
              label="名称"
              :rules="[(v: string) => !!v || '名称不能为空']"
              required
            />
            <v-textarea
              v-model="dialog.data.description"
              label="描述"
              rows="2"
            />
            <v-select
              v-model="dialog.data.embedding_model"
              :items="embeddingModels"
              label="Embedding 模型"
            />
            <v-slider
              v-model="dialog.data.chunk_size"
              label="分块大小"
              min="100"
              max="2000"
              step="100"
              thumb-label
            />
            <v-slider
              v-model="dialog.data.chunk_overlap"
              label="分块重叠"
              min="0"
              max="200"
              step="10"
              thumb-label
            />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="dialog.show = false">取消</v-btn>
          <v-btn color="primary" :disabled="!dialog.valid" @click="saveKnowledgeBase">
            保存
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 上传对话框 -->
    <v-dialog v-model="uploadDialog.show" max-width="500">
      <v-card>
        <v-card-title>上传文档</v-card-title>
        <v-card-text>
          <v-file-input
            v-model="uploadDialog.file"
            label="选择文件"
            accept=".pdf,.md,.txt,.doc,.docx"
            :rules="[(v: File[]) => v && v.length > 0 || '请选择文件']"
          />
          <v-select
            v-model="uploadDialog.fileType"
            :items="fileTypes"
            label="文件类型（可选，自动检测）"
            clearable
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="uploadDialog.show = false">取消</v-btn>
          <v-btn color="primary" :disabled="!uploadDialog.file" @click="uploadDocument">
            上传
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 删除确认 -->
    <v-dialog v-model="deleteDialog.show" max-width="400">
      <v-card>
        <v-card-title>确认删除</v-card-title>
        <v-card-text>
          确定要删除知识库 "{{ deleteDialog.kb?.name }}" 吗？
          <br />此操作将删除所有相关文档和文本块，不可恢复。
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="deleteDialog.show = false">取消</v-btn>
          <v-btn color="error" @click="deleteKnowledgeBase">删除</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { knowledgeApi, type KnowledgeBase, type Document, type SearchResult } from '@/api/knowledge'

const knowledgeBases = ref<KnowledgeBase[]>([])
const selectedKB = ref<KnowledgeBase | null>(null)
const documents = ref<Document[]>([])
const stats = ref<any>(null)
const loadingDocs = ref(false)
const activeTab = ref('documents')

const docHeaders = [
  { title: '文件名', key: 'filename' },
  { title: '类型', key: 'file_type' },
  { title: '大小', key: 'file_size' },
  { title: '状态', key: 'status' },
  { title: '块数', key: 'chunk_count' },
  { title: '上传时间', key: 'created_at' },
  { title: '操作', key: 'actions', sortable: false },
]

const embeddingModels = [
  { title: 'nomic-embed-text (本地)', value: 'nomic-embed-text' },
  { title: 'mxbai-embed-large (本地)', value: 'mxbai-embed-large' },
  { title: 'text-embedding-3-small (OpenAI)', value: 'text-embedding-3-small' },
]

const fileTypes = [
  { title: 'PDF', value: 'pdf' },
  { title: 'Markdown', value: 'md' },
  { title: '纯文本', value: 'txt' },
]

const searchTypes = [
  { title: '向量搜索', value: 'vector' },
  { title: '全文搜索', value: 'fulltext' },
  { title: '混合搜索', value: 'hybrid' },
]

const dialog = reactive({
  show: false,
  isEdit: false,
  valid: false,
  data: {
    id: '',
    name: '',
    description: '',
    embedding_model: 'nomic-embed-text',
    chunk_size: 500,
    chunk_overlap: 50,
  },
})

const uploadDialog = reactive({
  show: false,
  file: null as File | null,
  fileType: '',
})

const deleteDialog = reactive({
  show: false,
  kb: null as KnowledgeBase | null,
})

const searchQuery = ref('')
const searchType = ref('vector')
const searchResults = ref<SearchResult[]>([])
const hasSearched = ref(false)

onMounted(() => {
  loadKnowledgeBases()
})

async function loadKnowledgeBases() {
  try {
    const response = await knowledgeApi.listBases()
    knowledgeBases.value = response
  } catch (error) {
    console.error('Failed to load knowledge bases:', error)
    alert('加载知识库列表失败')
  }
}

async function selectKB(kb: KnowledgeBase) {
  selectedKB.value = kb
  loadDocuments(kb.id)
  loadStats(kb.id)
}

async function loadDocuments(kbId: string) {
  loadingDocs.value = true
  try {
    const response = await knowledgeApi.listDocuments(kbId)
    documents.value = response
  } catch (error) {
    console.error('Failed to load documents:', error)
  } finally {
    loadingDocs.value = false
  }
}

async function loadStats(kbId: string) {
  try {
    const response = await knowledgeApi.getStats(kbId)
    stats.value = response
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

function openCreateDialog() {
  dialog.isEdit = false
  dialog.data = {
    id: '',
    name: '',
    description: '',
    embedding_model: 'nomic-embed-text',
    chunk_size: 500,
    chunk_overlap: 50,
  }
  dialog.show = true
}

function openEditDialog() {
  if (!selectedKB.value) return
  dialog.isEdit = true
  dialog.data = {
    id: selectedKB.value.id,
    name: selectedKB.value.name,
    description: selectedKB.value.description,
    embedding_model: selectedKB.value.embedding_model,
    chunk_size: selectedKB.value.chunk_size,
    chunk_overlap: selectedKB.value.chunk_overlap,
  }
  dialog.show = true
}

async function saveKnowledgeBase() {
  try {
    if (dialog.isEdit) {
      await knowledgeApi.updateBase(dialog.data.id, {
        name: dialog.data.name,
        description: dialog.data.description,
      })
    } else {
      await knowledgeApi.createBase(dialog.data)
    }
    dialog.show = false
    loadKnowledgeBases()
  } catch (error) {
    console.error('Failed to save knowledge base:', error)
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
  } catch (error) {
    console.error('Failed to delete knowledge base:', error)
  }
}

function openUploadDialog() {
  uploadDialog.file = null
  uploadDialog.fileType = ''
  uploadDialog.show = true
}

async function uploadDocument() {
  if (!uploadDialog.file || !selectedKB.value) return
  try {
    await knowledgeApi.uploadDocument(
      selectedKB.value.id,
      uploadDialog.file,
      uploadDialog.fileType
    )
    uploadDialog.show = false
    loadDocuments(selectedKB.value.id)
    loadStats(selectedKB.value.id)
  } catch (error) {
    console.error('Failed to upload document:', error)
  }
}

async function deleteDocument(doc: Document) {
  if (!selectedKB.value) return
  try {
    await knowledgeApi.deleteDocument(selectedKB.value.id, doc.id)
    loadDocuments(selectedKB.value.id)
    loadStats(selectedKB.value.id)
  } catch (error) {
    console.error('Failed to delete document:', error)
  }
}

async function retryDocument(doc: Document) {
  if (!selectedKB.value) return
  try {
    await knowledgeApi.retryDocument(selectedKB.value.id, doc.id)
    setTimeout(() => loadDocuments(selectedKB.value!.id), 1000)
  } catch (error) {
    console.error('Failed to retry document:', error)
  }
}

async function performSearch() {
  if (!selectedKB.value || !searchQuery.value) return
  try {
    const response = await knowledgeApi.search(selectedKB.value.id, {
      query: searchQuery.value,
      top_k: 5,
      search_type: searchType.value as any,
    })
    searchResults.value = response.results
    hasSearched.value = true
  } catch (error) {
    console.error('Failed to search:', error)
  }
}

function getDocStatusColor(status: string) {
  switch (status) {
    case 'completed': return 'success'
    case 'processing': return 'warning'
    case 'failed': return 'error'
    default: return 'grey'
  }
}

function getStatusColor(stats: any) {
  if (!stats) return ''
  if (stats.failed_count > 0) return 'text-error'
  if (stats.pending_count > 0) return 'text-warning'
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
  padding: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
