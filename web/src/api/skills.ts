import api from './index'

function parseData<T>(res: any): T {
  if (res && res.data !== undefined) return res.data as T
  return res as T
}

// ---- 本地技能 ----

export interface Skill {
  name: string
  version: string
  display_name: string
  description: string
  enabled: boolean
  /** 1 = 提示词技能, 2 = 代码技能 */
  level?: number
  author?: string
}

export async function getSkills(): Promise<Skill[]> {
  const res = await api.get('/skills')
  const data = parseData<{ skills: Skill[] }>(res)
  return data.skills || []
}

export async function setSkillEnabled(name: string, enabled: boolean): Promise<{ ok: boolean }> {
  const res = await api.put(`/skills/${name}/enabled`, { enabled })
  return parseData(res)
}

export async function reloadSkills(): Promise<{ ok: boolean; count: number }> {
  const res = await api.post('/skills/reload')
  return parseData(res)
}

/** 安装来自市场的技能 */
export async function installSkill(name: string, version = 'latest'): Promise<{ ok: boolean; name: string }> {
  const res = await api.post('/skills/install', { name, version, source: 'market' })
  return parseData(res)
}

/** 导入本地 zip 压缩包 */
export async function importSkillZip(file: File): Promise<{ ok: boolean; name: string }> {
  const form = new FormData()
  form.append('file', file)
  const res = await api.post('/skills/import', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return parseData(res)
}

// ---- 技能市场 ----

export interface MarketSkill {
  id: number
  name: string
  display_name: string
  description: string
  readme?: string
  author: string
  author_url?: string
  tags: string[]
  level: string      // "L1" | "L2"
  featured: boolean
  install_count: number
  latest_version: string
}

export interface MarketListResult {
  items: MarketSkill[]
  total: number
  page: number
  page_size: number
}

export async function getMarketSkills(params?: {
  q?: string
  featured?: boolean
  page?: number
  page_size?: number
}): Promise<MarketListResult> {
  const query: Record<string, string> = {}
  if (params?.q) query.q = params.q
  if (params?.featured) query.featured = 'true'
  if (params?.page) query.page = String(params.page)
  if (params?.page_size) query.page_size = String(params.page_size)

  const res = await api.get('/skills/market', { params: query })
  const data = parseData<MarketListResult>(res)
  return data
}
