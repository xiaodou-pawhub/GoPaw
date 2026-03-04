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

export async function listMemories(params?: {
  category?: string
  q?: string
  limit?: number
}): Promise<{ memories: MemoryEntry[]; total: number }> {
  const p: Record<string, string> = {}
  if (params?.category) p.category = params.category
  if (params?.q) p.q = params.q
  if (params?.limit) p.limit = String(params.limit)
  return await api.get('/memories', { params: p })
}

export async function createMemory(data: {
  key: string
  content: string
  category?: string
}): Promise<{ memory: MemoryEntry }> {
  return await api.post('/memories', data)
}

export async function updateMemory(
  key: string,
  data: { content: string; category?: string }
): Promise<{ memory: MemoryEntry }> {
  return await api.put(`/memories/${encodeURIComponent(key)}`, data)
}

export async function deleteMemory(key: string): Promise<{ deleted: string }> {
  return await api.delete(`/memories/${encodeURIComponent(key)}`)
}

export async function getMemoryStats(): Promise<{ stats: MemoryStats }> {
  return await api.get('/memories/stats')
}

export async function importMarkdown(data: {
  content: string
  category?: string
  strategy?: string
}): Promise<{ imported: number; failures: string[]; total: number }> {
  return await api.post('/memories/import-md', data)
}

// ---- Memory Files (MD files) ----

export async function getMemoryMD(): Promise<{ content: string }> {
  return await api.get('/memory-files/memory')
}

export async function putMemoryMD(content: string): Promise<{ ok: boolean }> {
  return await api.put('/memory-files/memory', { content })
}

export async function listNotes(): Promise<{ notes: NoteFileInfo[] }> {
  return await api.get('/memory-files/notes')
}

export async function getNote(date: string): Promise<{ content: string; date: string }> {
  return await api.get(`/memory-files/notes/${date}`)
}

export async function putNote(
  date: string,
  content: string
): Promise<{ ok: boolean; date: string }> {
  return await api.put(`/memory-files/notes/${date}`, { content })
}

export async function appendNote(
  date: string,
  content: string
): Promise<{ ok: boolean; date: string }> {
  return await api.post(`/memory-files/notes/${date}/append`, { content })
}

export async function deleteNote(date: string): Promise<{ deleted: string }> {
  return await api.delete(`/memory-files/notes/${date}`)
}

export async function listArchives(): Promise<{ archives: ArchiveFileInfo[] }> {
  return await api.get('/memory-files/archives')
}

export async function getArchive(name: string): Promise<{ content: string; name: string }> {
  return await api.get(`/memory-files/archives/${encodeURIComponent(name)}`)
}
