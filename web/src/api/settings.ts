import api from './index'

export interface Provider {
  id: string
  name: string
  baseURL: string
  apiKey: string
  model: string
  isActive: boolean
}

// 中文：获取 LLM 提供商列表
// English: Get LLM provider list
export async function getProviders(): Promise<Provider[]> {
  const res = await api.get('/settings/providers')
  return res.providers || []
}

// 中文：保存 LLM 提供商（创建或更新）
// English: Save LLM provider (create or update)
export async function saveProvider(provider: Partial<Provider>) {
  return await api.post('/settings/providers', provider)
}

// 中文：设置活跃提供商
// English: Set active provider
export async function setActiveProvider(id: string) {
  return await api.put(`/settings/providers/${id}/active`)
}

// 中文：删除提供商
// English: Delete provider
export async function deleteProvider(id: string) {
  return await api.delete(`/settings/providers/${id}`)
}

// 中文：获取设置状态
// English: Get setup status
export async function getSetupStatus() {
  return await api.get('/settings/setup-status')
}

// 中文：获取 Agent 配置
// English: Get Agent config
export async function getAgent() {
  return await api.get('/settings/agent')
}

// 中文：保存 Agent 配置
// English: Save Agent config
export async function saveAgent(content: string) {
  return await api.put('/settings/agent', { content })
}
