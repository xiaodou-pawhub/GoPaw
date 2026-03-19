import api from './index'

export type MessageStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'delayed'

export interface Message {
  id: string
  queue: string
  type: string
  payload: Record<string, any>
  priority: number
  status: MessageStatus
  attempts: number
  max_retries: number
  delay_until?: string
  processed_by?: string
  created_at: string
  updated_at: string
  processed_at?: string
  completed_at?: string
  error?: string
}

export interface QueueStats {
  queue: string
  pending_count: number
  processing_count: number
  completed_count: number
  failed_count: number
  delayed_count: number
  total_count: number
  updated_at: string
}

export interface QueueInfo {
  name: string
  pending_count: number
  processing_count: number
  completed_count: number
  failed_count: number
  delayed_count: number
  total_count: number
}

export interface PublishMessageRequest {
  type: string
  payload: Record<string, any>
  priority?: number
  max_retries?: number
  delay_seconds?: number
}

export const queueApi = {
  // 解析标准响应格式
  parseData<T>(res: any): T {
    if (res && res.data !== undefined) {
      return res.data as T
    }
    return res as T
  },

  // List all queues with stats
  listQueues: async () => {
    const res = await api.get('/queues')
    return queueApi.parseData<QueueInfo[]>(res)
  },

  // Get queue statistics
  getStats: async (queueName: string) => {
    const res = await api.get(`/queues/${queueName}/stats`)
    return queueApi.parseData<QueueStats>(res)
  },

  // List messages in a queue
  listMessages: async (queueName: string, status?: MessageStatus, limit?: number) => {
    const res = await api.get(`/queues/${queueName}/messages`, { params: { status, limit } })
    return queueApi.parseData<Message[]>(res)
  },

  // Publish a message
  publishMessage: async (queueName: string, data: PublishMessageRequest) => {
    const res = await api.post(`/queues/${queueName}/messages`, data)
    return queueApi.parseData<Message>(res)
  },

  // Get a message by ID
  getMessage: async (id: string) => {
    const res = await api.get(`/messages/${id}`)
    return queueApi.parseData<Message>(res)
  },

  // Retry a failed message
  retryMessage: async (id: string) => {
    const res = await api.post(`/messages/${id}/retry`, {})
    return queueApi.parseData<any>(res)
  },

  // Delete a message
  deleteMessage: async (id: string) => {
    const res = await api.delete(`/messages/${id}`)
    return queueApi.parseData<any>(res)
  },

  // Pause a queue
  pauseQueue: async (queueName: string) => {
    const res = await api.post(`/queues/${queueName}/pause`, {})
    return queueApi.parseData<any>(res)
  },

  // Resume a queue
  resumeQueue: async (queueName: string) => {
    const res = await api.post(`/queues/${queueName}/resume`, {})
    return queueApi.parseData<any>(res)
  },

  // Cleanup a queue
  cleanupQueue: async (queueName: string, status: MessageStatus) => {
    const res = await api.post(`/queues/${queueName}/cleanup`, {}, { params: { status } })
    return queueApi.parseData<any>(res)
  },
}
