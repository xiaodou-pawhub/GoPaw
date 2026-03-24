import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { BackendProvider } from '@/types'
import type { ModeInfo } from '@/api/auth'

export const useAppStore = defineStore('app', () => {
  const locale = ref('zh-CN')
  const providers = ref<BackendProvider[]>([])
  const isLLMConfigured = ref(false)
  const modeInfo = ref<ModeInfo | null>(null)

  const isMultiUser = computed(() => modeInfo.value?.is_multi_user ?? false)
  const isSoloMode = computed(() => modeInfo.value?.mode === 'solo')

  function setLocale(lang: string) {
    locale.value = lang
  }

  function setProviders(list: BackendProvider[]) {
    providers.value = list
    isLLMConfigured.value = list.some(p => p.enabled || p.is_active)
  }

  function setModeInfo(info: ModeInfo) {
    modeInfo.value = info
  }

  return {
    locale,
    providers,
    isLLMConfigured,
    modeInfo,
    isMultiUser,
    isSoloMode,
    setLocale,
    setProviders,
    setModeInfo,
  }
}, {
  persist: {
    // 只持久化 locale 和 providers，modeInfo 每次启动重新获取
    pick: ['locale', 'providers']
  }
})
