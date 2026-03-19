import api from './index'

export interface AgentMetrics {
  call_count: number
  success_rate: number
  avg_duration_ms: number
  error_rate: number
  top_agents?: AgentStat[]
}

export interface AgentStat {
  agent_id: string
  agent_name: string
  call_count: number
  success_rate: number
  avg_duration_ms: number
}

export interface QueueMetrics {
  pending_count: number
  processing_count: number
  completed_count: number
  failed_count: number
  queue_stats?: QueueStat[]
}

export interface QueueStat {
  queue_name: string
  pending_count: number
  processing_count: number
  failed_count: number
}

export interface WorkflowMetrics {
  execution_count: number
  success_rate: number
  avg_duration_sec: number
  top_workflows?: WorkflowStat[]
}

export interface WorkflowStat {
  workflow_id: string
  workflow_name: string
  execution_count: number
  success_rate: number
}

export interface SystemMetrics {
  memory_mb: number
  goroutines: number
  db_size_mb: number
  uptime_sec: number
}

export interface DashboardData {
  agent: AgentMetrics
  queue: QueueMetrics
  workflow: WorkflowMetrics
  system: SystemMetrics
  updated_at: string
}

export interface RecentActivity {
  id: string
  type: string
  action: string
  description: string
  status: string
  timestamp: string
}

export const metricsApi = {
  // 解析标准响应格式
  parseData<T>(res: any): T {
    if (res && res.data !== undefined) {
      return res.data as T
    }
    return res as T
  },

  // Get dashboard data
  getDashboard: async () => {
    const res = await api.get('/metrics/dashboard')
    return metricsApi.parseData<DashboardData>(res)
  },

  // Get recent activity
  getRecentActivity: async (limit?: number) => {
    const res = await api.get('/metrics/activity', { params: { limit } })
    return metricsApi.parseData<RecentActivity[]>(res)
  },

  // Trigger metrics collection (admin only)
  collect: async () => {
    const res = await api.post('/metrics/collect', {})
    return metricsApi.parseData<any>(res)
  },
}
