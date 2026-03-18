import api from './index'

export interface KnowledgeBase {
  id: string
  name: string
  description: string
  embedding_model: string
  chunk_size: number
  chunk_overlap: number
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
  // 知识库管理
  listBases: () => api.get<KnowledgeBase[]>('/knowledge/bases'),
  
  getBase: (id: string) => api.get<KnowledgeBase>(`/knowledge/bases/${id}`),
  
  createBase: (data: {
    id: string
    name: string
    description?: string
    embedding_model?: string
    chunk_size?: number
    chunk_overlap?: number
  }) => api.post<KnowledgeBase>('/knowledge/bases', data),
  
  updateBase: (id: string, data: {
    name?: string
    description?: string
    status?: string
  }) => api.put(`/knowledge/bases/${id}`, data),
  
  deleteBase: (id: string) => api.delete(`/knowledge/bases/${id}`),
  
  getStats: (id: string) => api.get<{
    document_count: number
    chunk_count: number
    total_tokens: number
    pending_count: number
    completed_count: number
    failed_count: number
  }>(`/knowledge/bases/${id}/stats`),
  
  // 文档管理
  listDocuments: (kbId: string) => api.get<Document[]>(`/knowledge/bases/${kbId}/documents`),
  
  uploadDocument: (kbId: string, file: File, fileType?: string) => {
    const formData = new FormData()
    formData.append('file', file)
    if (fileType) {
      formData.append('file_type', fileType)
    }
    return api.post<Document>(`/knowledge/bases/${kbId}/documents`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
  },
  
  deleteDocument: (kbId: string, docId: string) => 
    api.delete(`/knowledge/bases/${kbId}/documents/${docId}`),
  
  retryDocument: (kbId: string, docId: string) => 
    api.post(`/knowledge/bases/${kbId}/documents/${docId}/retry`),
  
  // 搜索
  search: (kbId: string, req: SearchRequest) => 
    api.post<SearchResponse>(`/knowledge/bases/${kbId}/search`, req),
}
