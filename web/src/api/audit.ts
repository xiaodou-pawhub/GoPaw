import api from './index'

export type AuditCategory = 'auth' | 'agent' | 'workflow' | 'trigger' | 'mcp' | 'message' | 'system' | 'config' | 'http'
export type AuditAction = 
  | 'login' | 'logout' | 'token_refresh' | 'password_change'
  | 'agent_create' | 'agent_update' | 'agent_delete' | 'agent_switch' | 'agent_execute'
  | 'workflow_create' | 'workflow_update' | 'workflow_delete' | 'workflow_execute' | 'workflow_cancel'
  | 'trigger_create' | 'trigger_update' | 'trigger_delete' | 'trigger_fire'
  | 'mcp_create' | 'mcp_update' | 'mcp_delete' | 'mcp_connect'
  | 'message_send' | 'message_receive'
  | 'system_start' | 'system_stop' | 'system_error' | 'system_warning'
  | 'config_update' | 'http_request'
export type AuditStatus = 'success' | 'failed' | 'pending'

export interface AuditLog {
  id: string
  timestamp: string
  category: AuditCategory
  action: AuditAction
  user_id?: string
  user_ip?: string
  resource_type?: string
  resource_id?: string
  status: AuditStatus
  details?: Record<string, any>
  error?: string
  duration?: number
  request_id?: string
}

export interface AuditStats {
  total_count: number
  success_count: number
  failed_count: number
  by_category: Record<string, number>
  by_action: Record<string, number>
  by_user: Record<string, number>
  by_day: Record<string, number>
}

export interface QueryAuditLogsParams {
  category?: AuditCategory
  action?: AuditAction
  user_id?: string
  resource_type?: string
  resource_id?: string
  status?: AuditStatus
  start_time?: string
  end_time?: string
  limit?: number
  offset?: number
}

export interface ExportAuditLogsRequest {
  format: 'csv' | 'json'
  category?: AuditCategory
  user_id?: string
  start_time?: string
  end_time?: string
}

export interface CleanupAuditLogsRequest {
  older_than_days: number
}

export const auditApi = {
  // 解析标准响应格式
  parseData<T>(res: any): T {
    if (res && res.data !== undefined) {
      return res.data as T
    }
    return res as T
  },

  // List audit logs with filtering
  list: async (params?: QueryAuditLogsParams) => {
    const res = await api.get('/audit-logs', { params })
    return auditApi.parseData<AuditLog[]>(res)
  },

  // Get recent audit logs
  recent: async (limit?: number) => {
    const res = await api.get('/audit-logs/recent', { params: { limit } })
    return auditApi.parseData<AuditLog[]>(res)
  },

  // Get a single audit log
  get: async (id: string) => {
    const res = await api.get(`/audit-logs/${id}`)
    return auditApi.parseData<AuditLog>(res)
  },

  // Get audit statistics
  getStats: async () => {
    const res = await api.get('/audit-logs/stats')
    return auditApi.parseData<AuditStats>(res)
  },

  // Export audit logs
  export: async (data: ExportAuditLogsRequest) => {
    const res = await api.post('/audit-logs/export', data, { responseType: 'blob' })
    return res
  },

  // Cleanup old audit logs
  cleanup: async (data: CleanupAuditLogsRequest) => {
    const res = await api.post('/audit-logs/cleanup', data)
    return auditApi.parseData<any>(res)
  },
}
