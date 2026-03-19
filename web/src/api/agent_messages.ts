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
  // 解析标准响应格式
  parseData<T>(res: any): T {
    if (res && res.data !== undefined) {
      return res.data as T
    }
    return res as T
  },

  // Send a generic message
  send: async (data: SendMessageRequest) => {
    const res = await api.post('/agent-messages', data)
    return agentMessagesApi.parseData<AgentMessage>(res)
  },

  // Send a task message
  sendTask: async (data: SendTaskRequest) => {
    const res = await api.post('/agent-messages/task', data)
    return agentMessagesApi.parseData<AgentMessage>(res)
  },

  // Send a response message
  sendResponse: async (data: SendResponseRequest) => {
    const res = await api.post('/agent-messages/response', data)
    return agentMessagesApi.parseData<AgentMessage>(res)
  },

  // Send a notification message
  sendNotify: async (data: SendNotifyRequest) => {
    const res = await api.post('/agent-messages/notify', data)
    return agentMessagesApi.parseData<AgentMessage>(res)
  },

  // Send a query message
  sendQuery: async (data: SendQueryRequest) => {
    const res = await api.post('/agent-messages/query', data)
    return agentMessagesApi.parseData<AgentMessage>(res)
  },

  // Get a specific message
  get: async (id: string) => {
    const res = await api.get(`/agent-messages/${id}`)
    return agentMessagesApi.parseData<AgentMessage>(res)
  },

  // List messages for an agent (received)
  list: async (agentId: string, status?: MessageStatus) => {
    const res = await api.get(`/agent-messages/agent/${agentId}`, { params: { status } })
    return agentMessagesApi.parseData<AgentMessage[]>(res)
  },

  // List messages sent by an agent
  listSent: async (agentId: string) => {
    const res = await api.get(`/agent-messages/agent/${agentId}/sent`)
    return agentMessagesApi.parseData<AgentMessage[]>(res)
  },

  // List messages in a conversation
  listConversation: async (parentId: string) => {
    const res = await api.get(`/agent-messages/conversation/${parentId}`)
    return agentMessagesApi.parseData<AgentMessage[]>(res)
  },

  // Update message status
  updateStatus: async (id: string, status: MessageStatus, error?: string) => {
    const res = await api.put(`/agent-messages/${id}/status`, { status, error })
    return agentMessagesApi.parseData<any>(res)
  },

  // Get pending messages for an agent
  getPending: async (agentId: string) => {
    const res = await api.get(`/agent-messages/agent/${agentId}/pending`)
    return agentMessagesApi.parseData<AgentMessage[]>(res)
  },

  // Get message statistics for an agent
  getStats: async (agentId: string) => {
    const res = await api.get(`/agent-messages/agent/${agentId}/stats`)
    return agentMessagesApi.parseData<MessageStats>(res)
  },

  // List conversations for an agent
  listConversations: async (agentId: string) => {
    const res = await api.get(`/agent-messages/agent/${agentId}/conversations`)
    return agentMessagesApi.parseData<Conversation[]>(res)
  },
}
