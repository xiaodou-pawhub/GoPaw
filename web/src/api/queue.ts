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
  // List all queues with stats
  listQueues: () => api.get<QueueInfo[]>('/queues'),

  // Get queue statistics
  getStats: (queueName: string) => api.get<QueueStats>(`/queues/${queueName}/stats`),

  // List messages in a queue
  listMessages: (queueName: string, status?: MessageStatus, limit?: number) =>
    api.get<Message[]>(`/queues/${queueName}/messages`, { params: { status, limit } }),

  // Publish a message
  publishMessage: (queueName: string, data: PublishMessageRequest) =>
    api.post<Message>(`/queues/${queueName}/messages`, data),

  // Get a message by ID
  getMessage: (id: string) => api.get<Message>(`/messages/${id}`),

  // Retry a failed message
  retryMessage: (id: string) => api.post(`/messages/${id}/retry`, {}),

  // Delete a message
  deleteMessage: (id: string) => api.delete(`/messages/${id}`),

  // Pause a queue
  pauseQueue: (queueName: string) => api.post(`/queues/${queueName}/pause`, {}),

  // Resume a queue
  resumeQueue: (queueName: string) => api.post(`/queues/${queueName}/resume`, {}),

  // Cleanup a queue
  cleanupQueue: (queueName: string, status: MessageStatus) =>
    api.post(`/queues/${queueName}/cleanup`, {}, { params: { status } }),
}
