import api from './index'

export interface KnowledgeBase {
  id: string
  name: string
  description: string
  status: string
  document_count: number
  chunk_count: number
  total_tokens: number
  created_at: string
  updated_at: string
}

export interface Document {
  id: string
  knowledge_base_id: string
  filename: string
  file_type: string
  file_size: number
  file_hash: string
  metadata: Record<string, any>
  status: string
  error_message: string
  chunk_count: number
  processed_at: string
  created_at: string
}

export interface SearchResult {
  chunk_id: string
  content: string
  document_id: string
  document_name: string
  distance: number
  metadata: Record<string, any>
}

export interface SearchRequest {
  query: string
  top_k?: number
  search_type?: 'vector' | 'fulltext' | 'hybrid'
  filters?: Record<string, string>
}

export interface SearchResponse {
  results: SearchResult[]
  total: number
}

export const knowledgeApi = {
  // 解析标准响应格式
  parseData<T>(res: any): T {
    if (res && res.data !== undefined) {
      return res.data as T
    }
    return res as T
  },

  // 知识库管理
  listBases: async (): Promise<KnowledgeBase[]> => {
    const res = await api.get('/knowledge/bases')
    const data = knowledgeApi.parseData<KnowledgeBase[]>(res)
    return data || []
  },

  getBase: async (id: string) => {
    const res = await api.get(`/knowledge/bases/${id}`)
    return knowledgeApi.parseData<KnowledgeBase>(res)
  },

  createBase: async (data: {
    id: string
    name: string
    description?: string
  }) => {
    const res = await api.post('/knowledge/bases', data)
    return knowledgeApi.parseData<KnowledgeBase>(res)
  },

  updateBase: async (id: string, data: {
    name?: string
    description?: string
    status?: string
  }) => {
    const res = await api.put(`/knowledge/bases/${id}`, data)
    return knowledgeApi.parseData<any>(res)
  },

  deleteBase: async (id: string) => {
    const res = await api.delete(`/knowledge/bases/${id}`)
    return knowledgeApi.parseData<any>(res)
  },

  getStats: async (id: string) => {
    const res = await api.get(`/knowledge/bases/${id}/stats`)
    return knowledgeApi.parseData<{
      document_count: number
      chunk_count: number
      total_tokens: number
      pending_count: number
      completed_count: number
      failed_count: number
    }>(res)
  },

  // 文档管理
  listDocuments: async (kbId: string): Promise<Document[]> => {
    const res = await api.get(`/knowledge/bases/${kbId}/documents`)
    const data = knowledgeApi.parseData<Document[]>(res)
    return data || []
  },

  uploadDocument: async (kbId: string, file: File, fileType?: string) => {
    const formData = new FormData()
    formData.append('file', file)
    if (fileType) {
      formData.append('file_type', fileType)
    }
    const res = await api.post(`/knowledge/bases/${kbId}/documents`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
    return knowledgeApi.parseData<Document>(res)
  },

  deleteDocument: async (kbId: string, docId: string) => {
    const res = await api.delete(`/knowledge/bases/${kbId}/documents/${docId}`)
    return knowledgeApi.parseData<any>(res)
  },

  retryDocument: async (kbId: string, docId: string) => {
    const res = await api.post(`/knowledge/bases/${kbId}/documents/${docId}/retry`)
    return knowledgeApi.parseData<any>(res)
  },

  // 搜索
  search: async (kbId: string, req: SearchRequest) => {
    const res = await api.post(`/knowledge/bases/${kbId}/search`, req)
    return knowledgeApi.parseData<SearchResponse>(res)
  },
}
