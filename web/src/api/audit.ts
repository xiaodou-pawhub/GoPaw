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
  // List audit logs with filtering
  list: (params?: QueryAuditLogsParams) =>
    api.get<AuditLog[]>('/audit-logs', { params }),

  // Get recent audit logs
  recent: (limit?: number) =>
    api.get<AuditLog[]>('/audit-logs/recent', { params: { limit } }),

  // Get a single audit log
  get: (id: string) => api.get<AuditLog>(`/audit-logs/${id}`),

  // Get audit statistics
  getStats: () => api.get<AuditStats>('/audit-logs/stats'),

  // Export audit logs
  export: (data: ExportAuditLogsRequest) =>
    api.post('/audit-logs/export', data, { responseType: 'blob' }),

  // Cleanup old audit logs
  cleanup: (data: CleanupAuditLogsRequest) =>
    api.post('/audit-logs/cleanup', data),
}
