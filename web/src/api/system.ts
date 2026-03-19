import api from './index'

// 解析标准响应格式
function parseData<T>(res: any): T {
  if (res && res.data !== undefined) {
    return res.data as T
  }
  return res as T
}

// 获取系统日志
export async function getSystemLogs(): Promise<any> {
  const res = await api.get('/system/logs')
  return parseData<any>(res)
}

// 获取系统版本信息
export async function getSystemVersion(): Promise<any> {
  const res = await api.get('/system/version')
  return parseData<any>(res)
}
