import api from './index'

export type MessageType = 'task' | 'response' | 'notify' | 'query' | 'result'
export type MessageStatus = 'pending' | 'processing' | 'completed' | 'failed'

export interface AgentMessage {
  id: string
  type: MessageType
  from_agent: string
  to_agent: string
  content: string
  payload: Record<string, any>
  parent_id: string
  status: MessageStatus
  error?: string
  created_at: string
  updated_at: string
  processed_at?: string
}

export interface Conversation {
  id: string
  agent_ids: string[]
  title: string
  message_count: number
  last_message_at?: string
  created_at: string
  updated_at: string
}

export interface MessageStats {
  total_sent: number
  total_received: number
  pending_count: number
  failed_count: number
}

export interface SendMessageRequest {
  type: MessageType
  from_agent: string
  to_agent: string
  content: string
  payload?: Record<string, any>
  parent_id?: string
}

export interface SendTaskRequest {
  from_agent: string
  to_agent: string
  description: string
  task_id?: string
  priority?: 'low' | 'normal' | 'high' | 'urgent'
  data?: Record<string, any>
}

export interface SendResponseRequest {
  from_agent: string
  to_agent: string
  in_reply_to: string
  success: boolean
  message: string
  data?: Record<string, any>
}

export interface SendNotifyRequest {
  from_agent: string
  to_agent: string
  event: string
  details?: Record<string, any>
}

export interface SendQueryRequest {
  from_agent: string
  to_agent: string
  question: string
  context?: Record<string, any>
}

export const agentMessagesApi = {
  // Send a generic message
  send: (data: SendMessageRequest) => api.post<AgentMessage>('/agent-messages', data),

  // Send a task message
  sendTask: (data: SendTaskRequest) => api.post<AgentMessage>('/agent-messages/task', data),

  // Send a response message
  sendResponse: (data: SendResponseRequest) => api.post<AgentMessage>('/agent-messages/response', data),

  // Send a notification message
  sendNotify: (data: SendNotifyRequest) => api.post<AgentMessage>('/agent-messages/notify', data),

  // Send a query message
  sendQuery: (data: SendQueryRequest) => api.post<AgentMessage>('/agent-messages/query', data),

  // Get a specific message
  get: (id: string) => api.get<AgentMessage>(`/agent-messages/${id}`),

  // List messages for an agent (received)
  list: (agentId: string, status?: MessageStatus) =>
    api.get<AgentMessage[]>(`/agent-messages/agent/${agentId}`, { params: { status } }),

  // List messages sent by an agent
  listSent: (agentId: string) =>
    api.get<AgentMessage[]>(`/agent-messages/agent/${agentId}/sent`),

  // List messages in a conversation
  listConversation: (parentId: string) =>
    api.get<AgentMessage[]>(`/agent-messages/conversation/${parentId}`),

  // Update message status
  updateStatus: (id: string, status: MessageStatus, error?: string) =>
    api.put(`/agent-messages/${id}/status`, { status, error }),

  // Get pending messages for an agent
  getPending: (agentId: string) =>
    api.get<AgentMessage[]>(`/agent-messages/agent/${agentId}/pending`),

  // Get message statistics for an agent
  getStats: (agentId: string) =>
    api.get<MessageStats>(`/agent-messages/agent/${agentId}/stats`),

  // List conversations for an agent
  listConversations: (agentId: string) =>
    api.get<Conversation[]>(`/agent-messages/agent/${agentId}/conversations`),
}
