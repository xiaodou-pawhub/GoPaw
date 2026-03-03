import api from './index'
import type { BackendProvider, ChannelStatus } from '@/types'

// 获取初始化引导状态
export async function getSetupStatus(): Promise<{ llm_configured: boolean, setup_required: boolean, hint: string }> {
  return await api.get('/settings/setup-status')
}

// 获取所有 LLM 提供商
export async function getProviders(): Promise<BackendProvider[]> {
  const res = await api.get<{ providers: BackendProvider[] }>('/settings/providers')
  // @ts-ignore - 响应拦截器返回response.data
  return res.providers || []
}

// 保存/更新提供商
export async function saveProvider(data: Partial<BackendProvider>): Promise<{ id: string }> {
  return await api.post('/settings/providers', data)
}

// 删除提供商
export async function deleteProvider(id: string): Promise<{ deleted: string }> {
  return await api.delete(`/settings/providers/${id}`)
}

// 设置活跃提供商
export async function setActiveProvider(id: string): Promise<{ active: string }> {
  return await api.put(`/settings/providers/${id}/active`)
}

// 获取 Agent 设定 (System Prompt)
export async function getAgentConfig(): Promise<{ content: string }> {
  return await api.get('/settings/agent')
}

// 保存 Agent 设定
export async function saveAgentConfig(content: string): Promise<{ saved: boolean }> {
  return await api.put('/settings/agent', { content })
}

// 获取指定频道的配置
export async function getChannelConfig<T = Record<string, string>>(name: string): Promise<T> {
  const res = await api.get<{ channel: string, config: string }>(`/settings/channels/${name}`)
  try {
    // @ts-ignore - 响应拦截器返回response.data
    return JSON.parse(res.config) as T
  } catch (e) {
    return {} as T
  }
}

// 保存频道配置
export async function saveChannelConfig(name: string, config: Record<string, string>): Promise<{ ok: boolean }> {
  return await api.put(`/settings/channels/${name}`, {
    config: JSON.stringify(config)
  })
}

// 获取所有频道健康状态
export async function getChannelsHealth(): Promise<ChannelStatus[]> {
  const res = await api.get<{ channels: ChannelStatus[] }>('/channels/health')
  // @ts-ignore - 响应拦截器返回response.data
  return res.channels || []
}

// --- Skill 相关接口 ---

export interface Skill {
  name: string
  version: string
  display_name: string
  description: string
  enabled: boolean
  level?: number
  author?: string
}

// 获取所有技能列表
// 修正 P1: 后端路径为 /skills
export async function getSkills(): Promise<Skill[]> {
  const res = await api.get<{ skills: Skill[] }>('/skills')
  // @ts-ignore - 响应拦截器返回response.data
  return res.skills || []
}

// 设置技能启用状态
// 修正 P1: 后端路径为 /skills/:name/enabled
export async function setSkillEnabled(name: string, enabled: boolean): Promise<{ ok: boolean }> {
  return await api.put(`/skills/${name}/enabled`, { enabled })
}
