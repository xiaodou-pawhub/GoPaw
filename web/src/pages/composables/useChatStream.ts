import { ref } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { sendChatStream } from '@/api/agent'

export function useChatStream(onDone: () => void) {
  const { t } = useI18n()
  const message = useMessage()

  const isThinking = ref(false)
  const isStreaming = ref(false)
  let abortController: AbortController | null = null

  // 停止/中断当前对话
  function stopChatStream() {
    if (abortController) {
      abortController.abort()
      abortController = null
    }
    isStreaming.value = false
    isThinking.value = false
  }

  // 启动流式对话 (POST 模式)
  async function startChat(sessionId: string, content: string, onDelta: (delta: string) => void) {
    stopChatStream()
    
    isThinking.value = true
    isStreaming.value = true
    abortController = new AbortController()

    try {
      await sendChatStream(
        sessionId,
        content,
        {
          onDelta: (delta) => {
            isThinking.value = false
            onDelta(delta)
          },
          onDone: () => {
            isStreaming.value = false
            onDone()
          },
          onError: (err) => {
            if (err !== 'Request cancelled') {
              message.error(err)
            }
            isStreaming.value = false
            isThinking.value = false
          }
        },
        { signal: abortController.signal }
      )
    } catch (err) {
      console.error('Chat stream failed:', err)
      isStreaming.value = false
      isThinking.value = false
    }
  }

  return {
    isThinking,
    isStreaming,
    startChat,
    stopChatStream
  }
}
