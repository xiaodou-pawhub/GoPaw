import api from './index'

export type WorkflowStatus = 'draft' | 'active' | 'disabled'
export type ExecutionStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
export type StepStatus = 'pending' | 'running' | 'completed' | 'failed' | 'skipped' | 'cancelled'

export interface Workflow {
  id: string
  name: string
  description: string
  definition: WorkflowDef
  version: string
  status: WorkflowStatus
  created_at: string
  updated_at: string
}

export interface WorkflowDef {
  variables?: VariableDef[]
  steps: StepDef[]
  trigger?: TriggerConfig
  error_handlers?: ErrorHandlerDef[]
  parallel?: ParallelConfig
}

export interface VariableDef {
  name: string
  type: 'string' | 'number' | 'boolean' | 'object' | 'array'
  required?: boolean
  default?: any
}

export interface StepDef {
  id: string
  name: string
  agent: string
  action: 'task' | 'notify' | 'query'
  input?: Record<string, any>
  output?: string[]
  depends_on?: string[]
  condition?: string
  timeout?: number
  retry?: number
  retry_delay?: number
}

export interface TriggerConfig {
  type: string
  config?: Record<string, any>
}

export interface ErrorHandlerDef {
  condition: string
  action: string
  agent?: string
  message?: string
  input?: Record<string, any>
}

export interface ParallelConfig {
  max_concurrent?: number
}

export interface Execution {
  id: string
  workflow_id: string
  status: ExecutionStatus
  input?: Record<string, any>
  output?: Record<string, any>
  variables?: Record<string, any>
  started_at?: string
  completed_at?: string
  error?: string
  triggered_by?: string
  steps?: StepExecution[]
}

export interface StepExecution {
  id: string
  execution_id: string
  step_id: string
  agent_id: string
  status: StepStatus
  input?: Record<string, any>
  output?: Record<string, any>
  started_at?: string
  completed_at?: string
  error?: string
  retry_count?: number
}

export interface ExecutionStats {
  total_executions: number
  completed_count: number
  failed_count: number
  running_count: number
  pending_count: number
  cancelled_count: number
  avg_execution_time: number
}

export interface CreateWorkflowRequest {
  id: string
  name: string
  description?: string
  definition: WorkflowDef
  version?: string
}

export interface UpdateWorkflowRequest {
  name?: string
  description?: string
  definition?: WorkflowDef
  version?: string
  status?: WorkflowStatus
}

export interface ExecuteWorkflowRequest {
  input?: Record<string, any>
}

export interface ValidateWorkflowResponse {
  valid: boolean
  errors?: string[]
}

export const workflowsApi = {
  // 解析标准响应格式
  parseData<T>(res: any): T {
    if (res && res.data !== undefined) {
      return res.data as T
    }
    return res as T
  },

  // List all workflows
  list: async (status?: WorkflowStatus) => {
    const res = await api.get('/workflows', { params: { status } })
    return workflowsApi.parseData<Workflow[]>(res)
  },

  // Get a specific workflow
  get: async (id: string) => {
    const res = await api.get(`/workflows/${id}`)
    return workflowsApi.parseData<Workflow>(res)
  },

  // Create a new workflow
  create: async (data: CreateWorkflowRequest) => {
    const res = await api.post('/workflows', data)
    return workflowsApi.parseData<Workflow>(res)
  },

  // Update a workflow
  update: async (id: string, data: UpdateWorkflowRequest) => {
    const res = await api.put(`/workflows/${id}`, data)
    return workflowsApi.parseData<Workflow>(res)
  },

  // Delete a workflow
  delete: async (id: string) => {
    const res = await api.delete(`/workflows/${id}`)
    return workflowsApi.parseData<any>(res)
  },

  // Execute a workflow
  execute: async (id: string, data?: ExecuteWorkflowRequest) => {
    const res = await api.post(`/workflows/${id}/execute`, data || {})
    return workflowsApi.parseData<Execution>(res)
  },

  // Get an execution
  getExecution: async (id: string) => {
    const res = await api.get(`/workflows/executions/${id}`)
    return workflowsApi.parseData<Execution>(res)
  },

  // List executions for a workflow
  listExecutions: async (workflowId: string, limit?: number) => {
    const res = await api.get(`/workflows/${workflowId}/executions`, { params: { limit } })
    return workflowsApi.parseData<Execution[]>(res)
  },

  // Cancel an execution
  cancelExecution: async (id: string) => {
    const res = await api.post(`/workflows/executions/${id}/cancel`, {})
    return workflowsApi.parseData<any>(res)
  },

  // Get execution statistics
  getStats: async (workflowId: string) => {
    const res = await api.get(`/workflows/${workflowId}/stats`)
    return workflowsApi.parseData<ExecutionStats>(res)
  },

  // Validate workflow definition
  validate: async (definition: WorkflowDef) => {
    const res = await api.post('/workflows/validate', { definition })
    return workflowsApi.parseData<ValidateWorkflowResponse>(res)
  },
}
