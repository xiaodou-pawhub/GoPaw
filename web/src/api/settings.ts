import api from './index'
import type { BackendProvider, BuiltinProvider, ChannelStatus } from '@/types'

/**
 * 解析标准响应格式
 * 后端返回：{ code, message, data }
 * 拦截器已返回 response.data，所以直接访问 data 字段
 */
function parseData<T>(res: any): T {
  // 标准格式：{ code, message, data }
  if (res && res.data !== undefined) {
    return res.data as T
  }
  // 向后兼容：直接返回数据
  return res as T
}

// 获取初始化引导状态
export async function getSetupStatus(): Promise<{ llm_configured: boolean, setup_required: boolean, hint: string }> {
  const res = await api.get('/settings/setup-status')
  return parseData(res)
}

// 获取所有 LLM 提供商
export async function getProviders(): Promise<BackendProvider[]> {
  const res = await api.get('/settings/providers')
  const data = parseData<{ providers: BackendProvider[] }>(res)
  return data.providers || []
}

// 获取内置预置厂商库
export async function getBuiltinProviders(): Promise<BuiltinProvider[]> {
  const res = await api.get('/settings/builtin-providers')
  const data = parseData<{ providers: BuiltinProvider[] }>(res)
  return data.providers || []
}

// 获取所有提供商的实时健康状态
export async function getProvidersHealth(): Promise<any[]> {
  const res = await api.get('/settings/providers/health')
  const data = parseData<{ health: any[] }>(res)
  return data.health || []
}

// 保存/更新提供商
export async function saveProvider(data: Partial<BackendProvider>): Promise<{ id: string }> {
  const res = await api.post('/settings/providers', data)
  return parseData(res)
}

// 删除提供商
export async function deleteProvider(id: string): Promise<{ deleted: string }> {
  const res = await api.delete(`/settings/providers/${id}`)
  return parseData(res)
}

// 设置活跃提供商
export async function setActiveProvider(id: string): Promise<{ active: string }> {
  const res = await api.put(`/settings/providers/${id}/active`)
  return parseData(res)
}

// 获取 Agent 设定 (System Prompt)
export async function getAgentConfig(): Promise<{ content: string }> {
  const res = await api.get('/settings/agent')
  return parseData(res)
}

// 保存 Agent 设定
export async function saveAgentConfig(content: string): Promise<{ saved: boolean }> {
  const res = await api.put('/settings/agent', { content })
  return parseData(res)
}

// 获取指定频道的配置
export async function getChannelConfig<T = Record<string, string>>(name: string): Promise<T> {
  const res = await api.get(`/settings/channels/${name}`)
  return parseData<T>(res)
}

// 保存频道配置
export async function saveChannelConfig(name: string, config: Record<string, string>): Promise<{ ok: boolean }> {
  const res = await api.put(`/settings/channels/${name}`, { config: JSON.stringify(config) })
  return parseData(res)
}

// 获取工作区背景描述（CONTEXT.md）
export async function getWorkspaceContext(): Promise<{ content: string }> {
  const res = await api.get('/workspace/context')
  return parseData(res)
}

// 保存工作区背景描述
export async function saveWorkspaceContext(content: string): Promise<{ saved: boolean }> {
  const res = await api.put('/workspace/context', { content })
  return parseData(res)
}

// 获取 Agent 记忆文件（MEMORY.md）
export async function getAgentMemory(): Promise<{ content: string }> {
  const res = await api.get('/workspace/memory')
  return parseData(res)
}

// 覆盖 Agent 记忆文件（用于用户手动校正）
export async function saveAgentMemory(content: string): Promise<{ saved: boolean }> {
  const res = await api.put('/workspace/memory', { content })
  return parseData(res)
}

// 获取所有频道健康状态
export async function getChannelsHealth(): Promise<ChannelStatus[]> {
  const res = await api.get('/channels/health')
  const data = parseData<{ channels: ChannelStatus[] }>(res)
  return data.channels || []
}

// 测试频道连通性
export async function testChannel(name: string): Promise<{ success: boolean, message: string }> {
  const res = await api.post(`/channels/${name}/test`)
  return parseData(res)
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
export async function getSkills(): Promise<Skill[]> {
  const res = await api.get('/skills')
  const data = parseData<{ skills: Skill[] }>(res)
  return data.skills || []
}

// 设置技能启用状态
export async function setSkillEnabled(name: string, enabled: boolean): Promise<{ ok: boolean }> {
  const res = await api.put(`/skills/${name}/enabled`, { enabled })
  return parseData(res)
}

// 重新扫描技能目录，加载新增技能（无需重启）
export async function reloadSkills(): Promise<{ ok: boolean; count: number }> {
  const res = await api.post('/skills/reload')
  return parseData(res)
}
