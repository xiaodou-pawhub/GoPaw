import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage, useDialog } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { getSessions, deleteSession as apiDeleteSession } from '@/api/agent'
import type { SessionInfo } from '@/types'

export function useSessions() {
  const { t } = useI18n()
  const message = useMessage()
  const dialog = useDialog()
  const router = useRouter()
  const route = useRoute()

  const sessions = ref<SessionInfo[]>([])
  const loading = ref(false)

  // 加载会话列表
  async function loadSessions() {
    loading.value = true
    try {
      const list = await getSessions()
      sessions.value = list
      return list
    } catch (error) {
      console.error('Failed to load sessions:', error)
      return []
    } finally {
      loading.value = false
    }
  }

  // 路由跳转切换会话
  function selectSession(id: string) {
    if (route.params.id === id) return
    router.push({ name: 'Chat', params: { id } })
  }

  // 降级恢复逻辑
  function fallbackToValidSession() {
    if (sessions.value.length > 0) {
      router.replace({ name: 'Chat', params: { id: sessions.value[0].id } })
    } else {
      // 如果完全没会话，由外部逻辑处理新建
      router.push({ name: 'Chat' })
    }
  }

  // 删除会话逻辑
  async function handleDeleteSession(id: string, onCurrentDeleted: () => void) {
    dialog.warning({
      title: t('common.confirm'),
      content: t('chat.deleteConfirm'),
      positiveText: t('common.delete'),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        try {
          const isCurrent = route.params.id === id
          if (isCurrent) {
            onCurrentDeleted()
          }
          
          await apiDeleteSession(id)
          await loadSessions()
          message.success(`${t('common.success')} (ID: ${id.substring(0, 8)})`)
          
          if (isCurrent) {
            router.push({ name: 'Chat' })
          }
        } catch (error) {
          message.error(t('common.error'))
        }
      }
    })
  }

  return {
    sessions,
    loading,
    loadSessions,
    selectSession,
    fallbackToValidSession,
    handleDeleteSession
  }
}
