import api from './index'

// ---- Types ----

export interface MCPServer {
  id: string
  name: string
  description: string
  command: string
  args: string[]
  env: string[]
  transport: 'stdio' | 'sse'
  url?: string
  is_active: boolean
  is_builtin: boolean
  created_at: number
  updated_at: number
}

export interface MCPTool {
  name: string
  description: string
  inputSchema: Record<string, any>
}

export interface CreateMCPServerRequest {
  id: string
  name: string
  description?: string
  command: string
  args?: string[]
  env?: string[]
  transport?: 'stdio' | 'sse'
  url?: string
}

export interface UpdateMCPServerRequest {
  name?: string
  description?: string
  command?: string
  args?: string[]
  env?: string[]
  transport?: 'stdio' | 'sse'
  url?: string
}

// ---- API Functions ----

export async function listMCPServers(): Promise<{ servers: MCPServer[] }> {
  return await api.get('/mcp/servers')
}

export async function getMCPServer(id: string): Promise<MCPServer> {
  return await api.get(`/mcp/servers/${encodeURIComponent(id)}`)
}

export async function createMCPServer(data: CreateMCPServerRequest): Promise<MCPServer> {
  return await api.post('/mcp/servers', data)
}

export async function updateMCPServer(id: string, data: UpdateMCPServerRequest): Promise<MCPServer> {
  return await api.put(`/mcp/servers/${encodeURIComponent(id)}`, data)
}

export async function deleteMCPServer(id: string): Promise<{ deleted: string }> {
  return await api.delete(`/mcp/servers/${encodeURIComponent(id)}`)
}

export async function setMCPServerActive(id: string, active: boolean): Promise<{ id: string; active: boolean }> {
  return await api.post(`/mcp/servers/${encodeURIComponent(id)}/active`, { active })
}

export async function getMCPServerTools(id: string): Promise<{ tools: MCPTool[] }> {
  return await api.get(`/mcp/servers/${encodeURIComponent(id)}/tools`)
}

export async function getAllMCPTools(): Promise<{ tools: MCPTool[] }> {
  return await api.get('/mcp/tools')
}

// ---- Helpers ----

export function getTransportLabel(transport: string): string {
  const labels: Record<string, string> = {
    'stdio': 'Stdio (标准输入输出)',
    'sse': 'SSE (服务器推送)'
  }
  return labels[transport] || transport
}

export function getBuiltinServers(): CreateMCPServerRequest[] {
  return [
    {
      id: 'filesystem',
      name: '文件系统',
      description: '访问本地文件系统',
      command: 'npx',
      args: ['-y', '@modelcontextprotocol/server-filesystem'],
      transport: 'stdio'
    },
    {
      id: 'github',
      name: 'GitHub',
      description: '访问 GitHub API',
      command: 'npx',
      args: ['-y', '@modelcontextprotocol/server-github'],
      transport: 'stdio'
    },
    {
      id: 'postgres',
      name: 'PostgreSQL',
      description: '访问 PostgreSQL 数据库',
      command: 'npx',
      args: ['-y', '@modelcontextprotocol/server-postgres'],
      transport: 'stdio'
    },
    {
      id: 'sqlite',
      name: 'SQLite',
      description: '访问 SQLite 数据库',
      command: 'npx',
      args: ['-y', '@modelcontextprotocol/server-sqlite'],
      transport: 'stdio'
    }
  ]
}
