import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface Provider {
  id: string
  name: string
  baseURL: string
  apiKey: string
  model: string
  isActive: boolean
}

// 中文：应用全局状态管理
// English: App global state management
export const useAppStore = defineStore('app', () => {
  // 中文：主题状态
  // English: Theme state
  const isDark = ref(false)
  
  // 中文：语言状态
  // English: Language state
  const locale = ref('zh-CN')
  
  // 中文：LLM 提供商列表
  // English: LLM provider list
  const providers = ref<Provider[]>([])
  
  // 中文：是否已配置 LLM
  // English: Whether LLM is configured
  const isLLMConfigured = ref(false)

  // 中文：切换主题
  // English: Toggle theme
  function toggleTheme() {
    isDark.value = !isDark.value
  }

  // 中文：设置语言
  // English: Set language
  function setLocale(lang: string) {
    locale.value = lang
  }

  // 中文：更新提供商列表
  // English: Update provider list
  function setProviders(list: Provider[]) {
    providers.value = list
    isLLMConfigured.value = list.some(p => p.isActive)
  }

  // 中文：添加提供商
  // English: Add provider
  function addProvider(provider: Provider) {
    providers.value.push(provider)
    if (provider.isActive) {
      isLLMConfigured.value = true
    }
  }

  // 中文：删除提供商
  // English: Delete provider
  function removeProvider(id: string) {
    providers.value = providers.value.filter(p => p.id !== id)
    isLLMConfigured.value = providers.value.some(p => p.isActive)
  }

  return {
    isDark,
    locale,
    providers,
    isLLMConfigured,
    toggleTheme,
    setLocale,
    setProviders,
    addProvider,
    removeProvider
  }
}, {
  persist: {
    key: 'gopaw-app-store',
    storage: localStorage,
    paths: ['isDark', 'locale']
  }
})
