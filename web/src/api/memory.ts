import api from './index'

// ---- Types ----

export interface MemoryEntry {
  id: string
  key: string
  content: string
  category: 'core' | 'daily' | 'conversation' | string
  created_at: number
  updated_at: number
  score?: number
}

export interface MemoryStats {
  total: number
  core: number
  daily: number
  conversation: number
  custom: number
}

export interface NoteFileInfo {
  date: string    // YYYYMMDD
  month: string   // YYYYMM
  path: string
  mod_time: number
  size: number
}

export interface ArchiveFileInfo {
  name: string
  mod_time: number
  size: number
}

// ---- Structured Memories (memories.db) ----

// 解析标准响应格式
function parseData<T>(res: any): T {
  if (res && res.data !== undefined) {
    return res.data as T
  }
  return res as T
}

export async function listMemories(params?: {
  category?: string
  q?: string
  limit?: number
  namespace?: string
}): Promise<{ memories: MemoryEntry[]; total: number }> {
  const p: Record<string, string> = {}
  if (params?.category) p.category = params.category
  if (params?.q) p.q = params.q
  if (params?.limit) p.limit = String(params.limit)
  if (params?.namespace) p.namespace = params.namespace
  const res = await api.get('/memories', { params: p })
  return parseData<{ memories: MemoryEntry[]; total: number }>(res)
}

export async function createMemory(data: {
  key: string
  content: string
  category?: string
}): Promise<{ memory: MemoryEntry }> {
  const res = await api.post('/memories', data)
  return parseData<{ memory: MemoryEntry }>(res)
}

export async function updateMemory(
  key: string,
  data: { content: string; category?: string }
): Promise<{ memory: MemoryEntry }> {
  const res = await api.put(`/memories/${encodeURIComponent(key)}`, data)
  return parseData<{ memory: MemoryEntry }>(res)
}

export async function deleteMemory(key: string): Promise<{ deleted: string }> {
  const res = await api.delete(`/memories/${encodeURIComponent(key)}`)
  return parseData<{ deleted: string }>(res)
}

export async function getMemoryStats(): Promise<{ stats: MemoryStats }> {
  const res = await api.get('/memories/stats')
  return parseData<{ stats: MemoryStats }>(res)
}

export async function importMarkdown(data: {
  content: string
  category?: string
  strategy?: string
}): Promise<{ imported: number; failures: string[]; total: number }> {
  const res = await api.post('/memories/import-md', data)
  return parseData<{ imported: number; failures: string[]; total: number }>(res)
}

// ---- Memory Files (MD files) ----

export async function getMemoryMD(): Promise<{ content: string }> {
  const res = await api.get('/memory-files/memory')
  return parseData<{ content: string }>(res)
}

export async function putMemoryMD(content: string): Promise<{ ok: boolean }> {
  const res = await api.put('/memory-files/memory', { content })
  return parseData<{ ok: boolean }>(res)
}

export async function listNotes(): Promise<{ notes: NoteFileInfo[] }> {
  const res = await api.get('/memory-files/notes')
  return parseData<{ notes: NoteFileInfo[] }>(res)
}

export async function getNote(date: string): Promise<{ content: string; date: string }> {
  const res = await api.get(`/memory-files/notes/${date}`)
  return parseData<{ content: string; date: string }>(res)
}

export async function putNote(
  date: string,
  content: string
): Promise<{ ok: boolean; date: string }> {
  const res = await api.put(`/memory-files/notes/${date}`, { content })
  return parseData<{ ok: boolean; date: string }>(res)
}

export async function appendNote(
  date: string,
  content: string
): Promise<{ ok: boolean; date: string }> {
  const res = await api.post(`/memory-files/notes/${date}/append`, { content })
  return parseData<{ ok: boolean; date: string }>(res)
}

export async function deleteNote(date: string): Promise<{ deleted: string }> {
  const res = await api.delete(`/memory-files/notes/${date}`)
  return parseData<{ deleted: string }>(res)
}

export async function listArchives(): Promise<{ archives: ArchiveFileInfo[] }> {
  const res = await api.get('/memory-files/archives')
  return parseData<{ archives: ArchiveFileInfo[] }>(res)
}

export async function getArchive(name: string): Promise<{ content: string; name: string }> {
  const res = await api.get(`/memory-files/archives/${encodeURIComponent(name)}`)
  return parseData<{ content: string; name: string }>(res)
}
