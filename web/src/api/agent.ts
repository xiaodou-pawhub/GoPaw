import api from './index'
import type { SessionInfo, ChatMessage } from '@/types'

// 中文：获取所有会话列表
// English: Get all sessions
export async function getSessions(): Promise<SessionInfo[]> {
  const res: any = await api.get('/agent/sessions')
  return res.sessions || []
}

// 中文：获取指定会话的历史消息
// English: Get history messages for a specific session
export async function getSessionMessages(sessionId: string, limit: number = 100): Promise<ChatMessage[]> {
  const res: any = await api.get(`/agent/sessions/${sessionId}/messages`, {
    params: { limit }
  })
  const backendMsgs = res.messages || []
  // 中文：将后端结构转换为前端 ChatMessage 结构
  // English: Map backend structure to frontend ChatMessage structure
  return backendMsgs.map((m: any, index: number) => ({
    id: `hist-${sessionId}-${index}`,
    role: m.role,
    content: m.content,
    time: new Date(m.created_at).toLocaleTimeString()
  }))
}

// 中文：发送对话消息（同步）
// English: Send chat message (sync)
export async function sendChat(sessionId: string, content: string) {
  return await api.post('/agent/chat', {
    session_id: sessionId,
    content: content,
    msg_type: 'text'
  })
}

// 中文：获取流式对话的 SSE URL
// English: Get SSE URL for streaming chat
export function getChatStreamUrl(sessionId: string, content: string): string {
  return `/api/agent/chat/stream?session_id=${sessionId}&content=${encodeURIComponent(content)}`
}
