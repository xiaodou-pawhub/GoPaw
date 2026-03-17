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
  // List all workflows
  list: (status?: WorkflowStatus) =>
    api.get<Workflow[]>('/workflows', { params: { status } }),

  // Get a specific workflow
  get: (id: string) => api.get<Workflow>(`/workflows/${id}`),

  // Create a new workflow
  create: (data: CreateWorkflowRequest) => api.post<Workflow>('/workflows', data),

  // Update a workflow
  update: (id: string, data: UpdateWorkflowRequest) => api.put<Workflow>(`/workflows/${id}`, data),

  // Delete a workflow
  delete: (id: string) => api.delete(`/workflows/${id}`),

  // Execute a workflow
  execute: (id: string, data?: ExecuteWorkflowRequest) =>
    api.post<Execution>(`/workflows/${id}/execute`, data || {}),

  // Get an execution
  getExecution: (id: string) => api.get<Execution>(`/workflows/executions/${id}`),

  // List executions for a workflow
  listExecutions: (workflowId: string, limit?: number) =>
    api.get<Execution[]>(`/workflows/${workflowId}/executions`, { params: { limit } }),

  // Cancel an execution
  cancelExecution: (id: string) => api.post(`/workflows/executions/${id}/cancel`, {}),

  // Get execution statistics
  getStats: (workflowId: string) => api.get<ExecutionStats>(`/workflows/${workflowId}/stats`),

  // Validate workflow definition
  validate: (definition: WorkflowDef) =>
    api.post<ValidateWorkflowResponse>('/workflows/validate', { definition }),
}
