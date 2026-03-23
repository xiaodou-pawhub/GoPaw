import api from './index'

// 告警类型
export type AlertType = 'metric' | 'error' | 'custom'

// 渠道类型
export type ChannelType = 'email' | 'dingtalk' | 'wecom' | 'webhook'

// 告警条件
export interface AlertCondition {
  metric: string      // 指标名称: latency, error_rate, token_usage
  operator: string    // 操作符: >, <, >=, <=, ==, !=
  threshold: number   // 阈值
  duration: number    // 持续时间（秒）
  aggregation: string // 聚合方式: avg, max, min, sum
}

// 告警规则
export interface AlertRule {
  id: string
  name: string
  description: string
  type: AlertType
  condition: AlertCondition
  channels: string[]
  enabled: boolean
  last_triggered?: string
  created_at: string
  updated_at: string
}

// 渠道配置
export interface ChannelConfig {
  // Email
  smtp_host?: string
  smtp_port?: number
  smtp_user?: string
  smtp_password?: string
  from?: string
  to?: string[]

  // DingTalk
  dingtalk_webhook?: string
  dingtalk_secret?: string

  // WeCom
  wecom_webhook?: string

  // Webhook
  webhook_url?: string
  webhook_method?: string
  webhook_headers?: Record<string, string>
}

// 通知渠道
export interface NotificationChannel {
  id: string
  name: string
  type: ChannelType
  config: ChannelConfig
  enabled: boolean
  last_used?: string
  created_at: string
  updated_at: string
}

// 告警历史
export interface AlertHistory {
  id: string
  rule_id: string
  rule_name: string
  type: AlertType
  message: string
  value: number
  threshold: number
  status: 'triggered' | 'resolved'
  channels: string[]
  created_at: string
  resolved_at?: string
}

// 创建/更新告警规则请求
export interface CreateAlertRuleRequest {
  name: string
  description?: string
  type: AlertType
  condition: AlertCondition
  channels: string[]
  enabled?: boolean
}

// 创建/更新通知渠道请求
export interface CreateChannelRequest {
  name: string
  type: ChannelType
  config: ChannelConfig
  enabled?: boolean
}

function parseData<T>(res: any): T {
  if (res && res.data !== undefined) {
    return res.data as T
  }
  return res as T
}

export const alertApi = {
  // ==================== 告警规则 ====================

  // 列出所有告警规则
  listRules: async (): Promise<AlertRule[]> => {
    const res = await api.get('/alert/rules')
    return parseData<AlertRule[]>(res)
  },

  // 获取单个告警规则
  getRule: async (id: string): Promise<AlertRule> => {
    const res = await api.get(`/alert/rules/${id}`)
    return parseData<AlertRule>(res)
  },

  // 创建告警规则
  createRule: async (data: CreateAlertRuleRequest): Promise<AlertRule> => {
    const res = await api.post('/alert/rules', data)
    return parseData<AlertRule>(res)
  },

  // 更新告警规则
  updateRule: async (id: string, data: Partial<CreateAlertRuleRequest>): Promise<AlertRule> => {
    const res = await api.put(`/alert/rules/${id}`, data)
    return parseData<AlertRule>(res)
  },

  // 删除告警规则
  deleteRule: async (id: string): Promise<void> => {
    await api.delete(`/alert/rules/${id}`)
  },

  // 启用/禁用告警规则
  toggleRule: async (id: string, enabled: boolean): Promise<AlertRule> => {
    const res = await api.put(`/alert/rules/${id}`, { enabled })
    return parseData<AlertRule>(res)
  },

  // ==================== 通知渠道 ====================

  // 列出所有通知渠道
  listChannels: async (): Promise<NotificationChannel[]> => {
    const res = await api.get('/alert/channels')
    return parseData<NotificationChannel[]>(res)
  },

  // 获取单个通知渠道
  getChannel: async (id: string): Promise<NotificationChannel> => {
    const res = await api.get(`/alert/channels/${id}`)
    return parseData<NotificationChannel>(res)
  },

  // 创建通知渠道
  createChannel: async (data: CreateChannelRequest): Promise<NotificationChannel> => {
    const res = await api.post('/alert/channels', data)
    return parseData<NotificationChannel>(res)
  },

  // 更新通知渠道
  updateChannel: async (id: string, data: Partial<CreateChannelRequest>): Promise<NotificationChannel> => {
    const res = await api.put(`/alert/channels/${id}`, data)
    return parseData<NotificationChannel>(res)
  },

  // 删除通知渠道
  deleteChannel: async (id: string): Promise<void> => {
    await api.delete(`/alert/channels/${id}`)
  },

  // 测试通知渠道
  testChannel: async (id: string): Promise<{ success: boolean; message: string }> => {
    const res = await api.post(`/alert/channels/${id}/test`)
    return parseData<{ success: boolean; message: string }>(res)
  },

  // ==================== 告警历史 ====================

  // 列出告警历史
  listHistory: async (params?: {
    rule_id?: string
    status?: 'triggered' | 'resolved'
    limit?: number
    offset?: number
  }): Promise<AlertHistory[]> => {
    const res = await api.get('/alert/history', { params })
    return parseData<AlertHistory[]>(res)
  }
}

export default alertApi