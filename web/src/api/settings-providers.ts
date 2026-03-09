// 新增：模型优先级管理相关的 API 函数

import api from './index'
import type { BackendProvider } from '@/types'

/**
 * 快速启用/禁用模型提供商
 */
export async function toggleProvider(id: string, enabled: boolean): Promise<{ id: string, enabled: boolean }> {
  return await api.post(`/settings/providers/${id}/toggle`, { enabled })
}

/**
 * 批量调整模型优先级
 * @param providerIds 按新优先级排序的提供商 ID 列表
 */
export async function reorderProviders(providerIds: string[]): Promise<{ success: boolean }> {
  return await api.post('/settings/providers/reorder', { provider_ids: providerIds })
}

/**
 * 获取支持特定能力的模型提供商
 * @param capability 能力标签，如 "vision", "multimodal", "function_call"
 */
export async function getCapableProviders(capability: string): Promise<BackendProvider[]> {
  const res = await api.get<{ providers: BackendProvider[] }>(`/settings/providers/capable/${capability}`)
  // @ts-ignore - 响应拦截器返回 response.data
  return res.providers || []
}

/**
 * 获取第一个支持 Vision 的模型
 */
export async function getFirstVisionProvider(): Promise<BackendProvider | null> {
  const providers = await getCapableProviders('vision')
  if (providers.length > 0) {
    return providers[0]
  }
  // Fallback 到 multimodal
  const multimodal = await getCapableProviders('multimodal')
  if (multimodal.length > 0) {
    return multimodal[0]
  }
  return null
}
