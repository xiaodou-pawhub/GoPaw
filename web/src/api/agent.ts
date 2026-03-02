import api from './index'
import type { SessionInfo, ChatMessage, SessionStats } from '@/types'

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

// 中文：获取会话统计信息
// English: Get session statistics
export async function getSessionStats(sessionId: string): Promise<SessionStats> {
	return await api.get(`/agent/sessions/${sessionId}/stats`)
}

// 中文：删除会话
// English: Delete session
export async function deleteSession(sessionId: string) {
	return await api.delete(`/agent/sessions/${sessionId}`)
}

// 中文：获取流式对话的 SSE URL
// English: Get SSE URL for streaming chat
export function getChatStreamUrl(sessionId: string, content: string): string {
	return `/api/agent/chat/stream?session_id=${sessionId}&content=${encodeURIComponent(content)}`
}
