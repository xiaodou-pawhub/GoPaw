import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Provider } from '@/types'

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

  return {
    isDark,
    locale,
    providers,
    isLLMConfigured,
    toggleTheme,
    setLocale,
    setProviders
  }
}, {
  persist: {
    key: 'gopaw-app-store',
    storage: localStorage,
    paths: ['isDark', 'locale']
  }
})
