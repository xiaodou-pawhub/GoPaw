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
  // 解析标准响应格式
  parseData<T>(res: any): T {
    if (res && res.data !== undefined) {
      return res.data as T
    }
    return res as T
  },

  // List all triggers
  list: async () => {
    const res = await api.get('/triggers')
    return triggersApi.parseData<Trigger[]>(res)
  },

  // Get a specific trigger
  get: async (id: string) => {
    const res = await api.get(`/triggers/${id}`)
    return triggersApi.parseData<Trigger>(res)
  },

  // Create a new trigger
  create: async (data: CreateTriggerRequest) => {
    const res = await api.post('/triggers', data)
    return triggersApi.parseData<Trigger>(res)
  },

  // Update a trigger
  update: async (id: string, data: UpdateTriggerRequest) => {
    const res = await api.put(`/triggers/${id}`, data)
    return triggersApi.parseData<Trigger>(res)
  },

  // Delete a trigger
  delete: async (id: string) => {
    const res = await api.delete(`/triggers/${id}`)
    return triggersApi.parseData<any>(res)
  },

  // Enable a trigger
  enable: async (id: string) => {
    const res = await api.post(`/triggers/${id}/enable`, {})
    return triggersApi.parseData<any>(res)
  },

  // Disable a trigger
  disable: async (id: string) => {
    const res = await api.post(`/triggers/${id}/disable`, {})
    return triggersApi.parseData<any>(res)
  },

  // Manually fire a trigger
  fire: async (id: string, payload?: Record<string, any>) => {
    const res = await api.post(`/triggers/${id}/fire`, payload || {})
    return triggersApi.parseData<any>(res)
  },

  // Get trigger history
  getHistory: async (id: string) => {
    const res = await api.get(`/triggers/${id}/history`)
    return triggersApi.parseData<TriggerHistory[]>(res)
  },

  // List triggers by agent
  listByAgent: async (agentId: string) => {
    const res = await api.get(`/triggers/by-agent/${agentId}`)
    return triggersApi.parseData<Trigger[]>(res)
  },

  // Validate cron expression
  validateCron: async (expression: string) => {
    const res = await api.post('/triggers/validate-cron', { expression })
    return triggersApi.parseData<ValidateCronResponse>(res)
  },

  // Send message to trigger another agent
  sendMessage: async (from: string, to: string, payload?: Record<string, any>) => {
    const res = await api.post('/messages', { from, to, payload })
    return triggersApi.parseData<any>(res)
  },
}
