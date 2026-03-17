import api from './index'

export interface Trigger {
  id: string
  agent_id: string
  name: string
  description: string
  type: 'cron' | 'webhook' | 'message'
  config: CronConfig | WebhookConfig | MessageConfig
  reason: string
  is_enabled: boolean
  last_fired_at: string | null
  fire_count: number
  max_fires: number | null
  cooldown_seconds: number
  created_at: string
  updated_at: string
}

export interface CronConfig {
  expression: string
}

export interface WebhookConfig {
  secret: string
}

export interface MessageConfig {
  from_agent: string
}

export interface TriggerHistory {
  id: number
  trigger_id: string
  agent_id: string
  fired_at: string
  payload: string
  success: boolean
  error_message: string
}

export interface CreateTriggerRequest {
  id: string
  agent_id: string
  name: string
  description?: string
  type: 'cron' | 'webhook' | 'message'
  config: CronConfig | WebhookConfig | MessageConfig
  reason?: string
  is_enabled?: boolean
  max_fires?: number
  cooldown_seconds?: number
}

export interface UpdateTriggerRequest {
  name?: string
  description?: string
  type?: 'cron' | 'webhook' | 'message'
  config?: CronConfig | WebhookConfig | MessageConfig
  reason?: string
  is_enabled?: boolean
  max_fires?: number
  cooldown_seconds?: number
}

export interface ValidateCronResponse {
  valid: boolean
  description?: string
  next_run?: string
  error?: string
}

export const triggersApi = {
  // List all triggers
  list: () => api.get<Trigger[]>('/triggers'),

  // Get a specific trigger
  get: (id: string) => api.get<Trigger>(`/triggers/${id}`),

  // Create a new trigger
  create: (data: CreateTriggerRequest) => api.post<Trigger>('/triggers', data),

  // Update a trigger
  update: (id: string, data: UpdateTriggerRequest) => api.put<Trigger>(`/triggers/${id}`, data),

  // Delete a trigger
  delete: (id: string) => api.delete(`/triggers/${id}`),

  // Enable a trigger
  enable: (id: string) => api.post(`/triggers/${id}/enable`, {}),

  // Disable a trigger
  disable: (id: string) => api.post(`/triggers/${id}/disable`, {}),

  // Manually fire a trigger
  fire: (id: string, payload?: Record<string, any>) => api.post(`/triggers/${id}/fire`, payload || {}),

  // Get trigger history
  getHistory: (id: string) => api.get<TriggerHistory[]>(`/triggers/${id}/history`),

  // List triggers by agent
  listByAgent: (agentId: string) => api.get<Trigger[]>(`/triggers/by-agent/${agentId}`),

  // Validate cron expression
  validateCron: (expression: string) => api.post<ValidateCronResponse>('/triggers/validate-cron', { expression }),

  // Send message to trigger another agent
  sendMessage: (from: string, to: string, payload?: Record<string, any>) =>
    api.post('/messages', { from, to, payload }),
}
