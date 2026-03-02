import api from './index'
import type { Provider, BackendProvider } from '@/types'

// 中文：前端转后端映射 (LLM Provider)
// English: Map frontend to backend (LLM Provider)
function mapProviderToApi(provider: Partial<Provider>): Partial<BackendProvider> {
  const mapped: any = {}
  if (provider.id !== undefined) mapped.id = provider.id
  if (provider.name !== undefined) mapped.name = provider.name
  if (provider.baseURL !== undefined) mapped.base_url = provider.baseURL
  if (provider.apiKey !== undefined) mapped.api_key = provider.apiKey
  if (provider.model !== undefined) mapped.model = provider.model
  if (provider.maxTokens !== undefined) mapped.max_tokens = provider.maxTokens
  if (provider.timeoutSec !== undefined) mapped.timeout_sec = provider.timeoutSec
  if (provider.isActive !== undefined) mapped.is_active = provider.isActive
  return mapped
}

// 中文：后端转前端映射 (LLM Provider)
// English: Map backend to frontend (LLM Provider)
function mapProviderFromApi(data: BackendProvider): Provider {
  return {
    id: data.id,
    name: data.name,
    baseURL: data.base_url,
    apiKey: data.api_key,
    model: data.model,
    maxTokens: data.max_tokens,
    timeoutSec: data.timeout_sec,
    isActive: data.is_active,
    createdAt: data.created_at,
    updatedAt: data.updated_at
  }
}

// ── LLM 提供商 / LLM Providers ──────────────────────────────────────────────

export async function getProviders(): Promise<Provider[]> {
  const res: any = await api.get('/settings/providers')
  const backendList = res.providers || []
  return backendList.map((p: BackendProvider) => mapProviderFromApi(p))
}

export async function saveProvider(provider: Partial<Provider>) {
  const payload = mapProviderToApi(provider)
  return await api.post('/settings/providers', payload)
}

export async function setActiveProvider(id: string) {
  return await api.put(`/settings/providers/${id}/active`)
}

export async function deleteProvider(id: string) {
  return await api.delete(`/settings/providers/${id}`)
}

// ── 频道配置 / Channel Configs ───────────────────────────────────────────────

// 中文：获取指定频道的配置
// English: Get configuration for a specific channel
export async function getChannelConfig(name: string): Promise<any> {
  const res: any = await api.get(`/settings/channels/${name}`)
  try {
    return typeof res.config === 'string' ? JSON.parse(res.config) : res.config
  } catch (e) {
    return {}
  }
}

// 中文：保存频道配置
// English: Save channel configuration
export async function saveChannelConfig(name: string, config: any) {
  return await api.put(`/settings/channels/${name}`, {
    config: JSON.stringify(config)
  })
}

// 中文：获取所有频道的健康状态
// English: Get health status of all channels
export async function getChannelsHealth(): Promise<any[]> {
  const res: any = await api.get('/channels/health')
  return res.channels || []
}

// 中文：测试频道连接
// English: Test channel connection
export async function testChannel(name: string): Promise<{ success: boolean; message: string; details?: string }> {
  return await api.post(`/channels/${name}/test`)
}

// ── Agent 设定 / Agent Persona ───────────────────────────────────────────────

export async function getAgentMD(): Promise<{ content: string }> {
  return await api.get('/settings/agent')
}

export async function saveAgentMD(content: string) {
  return await api.put('/settings/agent', { content })
}

// ── 初始化状态 / Setup Status ───────────────────────────────────────────────

export async function getSetupStatus(): Promise<{ llm_configured: boolean, setup_required: boolean, hint: string }> {
  return await api.get('/settings/setup-status')
}

// ── 技能管理 / Skills ───────────────────────────────────────────────────────

// 中文：技能信息接口
// English: Skill information interface
export interface Skill {
  name: string
  display_name: string
  description: string
  author: string
  version: string
  level: number
  enabled: boolean
}

// 中文：获取所有技能
// English: Get all skills
export async function getSkills(): Promise<Skill[]> {
  const res: any = await api.get('/skills')
  return res.skills || []
}

// 中文：设置技能启用状态
// English: Set skill enabled state
export async function setSkillEnabled(name: string, enabled: boolean): Promise<{ ok: boolean }> {
  return await api.put(`/skills/${name}/enabled`, { enabled })
}
