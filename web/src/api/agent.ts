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

// 中文：更新会话名称
// English: Update session name
export async function updateSessionName(sessionId: string, name: string) {
	return await api.put(`/agent/sessions/${sessionId}/name`, { name })
}

// 中文：获取流式对话的 SSE URL（已废弃，仅用于短消息）
// English: Get SSE URL for streaming chat (deprecated, use only for short messages)
// @deprecated Use sendChatStream for large content support
export function getChatStreamUrl(sessionId: string, content: string): string {
	return `/api/agent/chat/stream?session_id=${sessionId}&content=${encodeURIComponent(content)}`
}

// 中文：流式对话回调接口
// English: Streaming chat callback interface
export interface StreamCallbacks {
	onDelta: (delta: string) => void
	onDone: () => void
	onError: (error: string) => void
}

// 中文：流式请求控制选项
// English: Streaming request control options
export interface StreamOptions {
	signal?: AbortSignal
}

// 中文：发送流式对话请求（POST，支持大内容如文件附件）
// English: Send streaming chat request (POST, supports large content like file attachments)
export async function sendChatStream(sessionId: string, content: string, callbacks: StreamCallbacks, options?: StreamOptions): Promise<void> {
	const init: RequestInit = {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		body: JSON.stringify({
			session_id: sessionId,
			content: content,
		}),
	}
	
	if (options?.signal) {
		init.signal = options.signal
	}
	
	const response = await fetch('/api/agent/chat/stream', init)

	if (!response.ok) {
		const errorData = await response.json().catch(() => ({ error: 'Request failed' }))
		callbacks.onError(errorData.error || `HTTP ${response.status}`)
		return
	}

	const reader = response.body?.getReader()
	if (!reader) {
		callbacks.onError('No response body')
		return
	}

	const decoder = new TextDecoder()
	let buffer = ''

	try {
		while (true) {
			// 检查是否被取消
			if (options?.signal?.aborted) {
				callbacks.onError('Request cancelled')
				break
			}
			
			const { done, value } = await reader.read()
			if (done) break

			buffer += decoder.decode(value, { stream: true })
			const lines = buffer.split('\n')
			buffer = lines.pop() || ''

			for (const line of lines) {
				if (line.startsWith('data: ')) {
					try {
						const data = JSON.parse(line.slice(6))
						if (data.delta) {
							callbacks.onDelta(data.delta)
						}
						if (data.done) {
							callbacks.onDone()
						}
						if (data.error) {
							callbacks.onError(data.error)
						}
					} catch {
						// 中文：忽略解析错误 / English: Ignore parse errors
					}
				}
			}
		}
	} finally {
		reader.releaseLock()
	}
}
