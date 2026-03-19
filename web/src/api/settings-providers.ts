// 新增：模型优先级管理相关的 API 函数

import api from './index'
import type { BackendProvider } from '@/types'

/**
 * 解析标准响应格式 { code, message, data }
 */
function parseData<T>(res: any): T {
  if (res && res.data !== undefined) {
    return res.data as T
  }
  return res as T
}

/**
 * 快速启用/禁用模型提供商
 */
export async function toggleProvider(id: string, enabled: boolean): Promise<{ id: string, enabled: boolean }> {
  const res = await api.post(`/settings/providers/${id}/toggle`, { enabled })
  return parseData(res)
}

/**
 * 批量调整模型优先级
 * @param providerIds 按新优先级排序的提供商 ID 列表
 */
export async function reorderProviders(providerIds: string[]): Promise<{ success: boolean }> {
  const res = await api.post('/settings/providers/reorder', { provider_ids: providerIds })
  return parseData(res)
}

/**
 * 获取支持特定能力的模型提供商
 * @param capability 能力标签，如 "vision", "multimodal", "function_call"
 */
export async function getCapableProviders(capability: string): Promise<BackendProvider[]> {
  const res = await api.get(`/settings/providers/capable/${capability}`)
  const data = parseData<{ providers: BackendProvider[] }>(res)
  return data.providers || []
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
