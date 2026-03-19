import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { BackendProvider } from '@/types'

// 中文：应用全局状态管理
// English: App global state management
export const useAppStore = defineStore('app', () => {
  // 中文：语言状态
  // English: Language state
  const locale = ref('zh-CN')

  // 中文：LLM 提供商列表
  // English: LLM provider list
  const providers = ref<BackendProvider[]>([])

  // 中文：是否已配置 LLM
  // English: Whether LLM is configured
  const isLLMConfigured = ref(false)

  // 中文：设置语言
  // English: Set language
  function setLocale(lang: string) {
    locale.value = lang
  }

  // 中文：更新提供商列表
  // English: Update provider list
  function setProviders(list: BackendProvider[]) {
    providers.value = list
    // 使用 enabled 字段判断（与后端 IsSetupRequired 一致）
    // 同时兼容 is_active 字段（旧版本）
    isLLMConfigured.value = list.some(p => p.enabled || p.is_active)
  }

  return {
    locale,
    providers,
    isLLMConfigured,
    setLocale,
    setProviders
  }
}, {
  persist: true
})
