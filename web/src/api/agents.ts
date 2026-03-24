import api from './index'

// ---- Types ----

export interface Agent {
  id: string
  name: string
  description: string
  avatar: string
  config_path: string
  is_active: boolean
  is_default: boolean
  created_at: number
  updated_at: number
  last_active_at?: number  // millisecond timestamp of latest session activity
  config?: AgentConfig
}

export interface AgentChannelConfig {
  type: string        // feishu | dingtalk | webhook | email
  enabled: boolean
  config: Record<string, string>
}

export interface AgentConfig {
  emoji: string
  description: string
  provider_ids: string[]
  mcp_servers: string[]
  knowledge_bases: string[]
  channels: AgentChannelConfig[]
  llm: LLMConfig
  system_prompt: string
  tools: ToolsConfig
  skills: string[]
  autonomy: Record<string, string>
  workspace: WorkspaceConfig
  memory: MemoryConfig
  max_steps: number
}

export interface LLMConfig {
  model: string
  temperature: number
  max_tokens: number
  top_p: number
  presence_penalty: number
  frequency_penalty: number
}

export interface ToolsConfig {
  enabled: string[]
  disabled: string[]
}

export interface WorkspaceConfig {
  root: string
  allowed_paths: string[]
}

export interface MemoryConfig {
  enabled: boolean
  namespace: string
}

export interface CreateAgentRequest {
  id: string
  name: string
  description?: string
  avatar?: string
  config?: AgentConfig
  visibility?: 'global' | 'private' | 'shared'
}

export interface UpdateAgentRequest {
  name?: string
  description?: string
  avatar?: string
  is_active?: boolean
  config?: AgentConfig
}

// ---- API Functions ----

// 解析标准响应格式
function parseData<T>(res: any): T {
  if (res && res.data !== undefined) {
    return res.data as T
  }
  return res as T
}

export async function listAgents(): Promise<{ agents: Agent[] }> {
  const res = await api.get('/agents')
  const agents = parseData<Agent[]>(res)
  return { agents }
}

export async function getAgent(id: string): Promise<Agent> {
  const res = await api.get(`/agents/${encodeURIComponent(id)}`)
  return parseData<Agent>(res)
}

export async function getDefaultAgent(): Promise<Agent> {
  const res = await api.get('/agents/default')
  return parseData<Agent>(res)
}

export async function createAgent(data: CreateAgentRequest): Promise<Agent> {
  const res = await api.post('/agents', data)
  return parseData<Agent>(res)
}

export async function updateAgent(id: string, data: UpdateAgentRequest): Promise<Agent> {
  const res = await api.put(`/agents/${encodeURIComponent(id)}`, data)
  return parseData<Agent>(res)
}

export async function deleteAgent(id: string): Promise<{ deleted: string }> {
  const res = await api.delete(`/agents/${encodeURIComponent(id)}`)
  return parseData<{ deleted: string }>(res)
}

export async function setDefaultAgent(id: string): Promise<{ default: string }> {
  const res = await api.post(`/agents/${encodeURIComponent(id)}/default`)
  return parseData<{ default: string }>(res)
}

export async function getAgentConfig(id: string): Promise<AgentConfig> {
  const res = await api.get(`/agents/${encodeURIComponent(id)}/config`)
  return parseData<AgentConfig>(res)
}

export async function updateAgentConfig(id: string, config: AgentConfig): Promise<AgentConfig> {
  const res = await api.put(`/agents/${encodeURIComponent(id)}/config`, config)
  return parseData<AgentConfig>(res)
}

// ---- Helpers ----

export function getDefaultConfig(): AgentConfig {
  return {
    emoji: '🤖',
    description: '',
    provider_ids: [],
    mcp_servers: [],
    llm: {
      model: 'gpt-4',
      temperature: 0.7,
      max_tokens: 4000,
      top_p: 1.0,
      presence_penalty: 0,
      frequency_penalty: 0
    },
    knowledge_bases: [],
    channels: [],
    system_prompt: 'You are a helpful AI assistant.',
    tools: {
      enabled: [],
      disabled: []
    },
    skills: [],
    autonomy: {
      read_file: 'L1',
      write_file: 'L2',
      file_edit: 'L2',
      file_manage: 'L2',
      file_search: 'L1',
      shell: 'L3',
      web_search: 'L1',
      http_get: 'L1',
      http_post: 'L2',
      memory_store: 'L1',
      memory_recall: 'L1',
      memory_search: 'L1',
      send_to_user: 'L2'
    },
    workspace: {
      root: '',
      allowed_paths: []
    },
    memory: {
      enabled: true,
      namespace: 'default'
    },
    max_steps: 20
  }
}

export function getAvailableTools(): string[] {
  return [
    'read_file',
    'write_file',
    'file_edit',
    'file_manage',
    'file_search',
    'shell',
    'web_search',
    'http_get',
    'http_post',
    'memory_store',
    'memory_recall',
    'memory_search',
    'send_to_user',
    'send_email',
    'image_info',
    'image_process'
  ]
}

export function getAutonomyLevels(): string[] {
  return ['L1', 'L2', 'L3']
}

export function getAutonomyLabel(level: string): string {
  const labels: Record<string, string> = {
    'L1': 'L1 - 自动执行',
    'L2': 'L2 - 执行并通知',
    'L3': 'L3 - 需要审批'
  }
  return labels[level] || level
}
