import axios, { AxiosError } from 'axios'

// 中文：创建 Axios 实例，配置基础 URL 和超时
// English: Create Axios instance with base URL and timeout
const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 中文：统一的错误消息映射
// English: Unified error message mapping
const errorMessages: Record<string, string> = {
  // 认证错误
  'PROVIDER_NOT_CONFIGURED': '请先配置 LLM 模型',
  'API_KEY_INVALID': 'API Key 无效，请检查配置',
  'API_KEY_MISSING': 'API Key 未配置',
  'MODEL_NOT_FOUND': '模型不存在',
  
  // 网络错误
  'NETWORK_ERROR': '网络连接失败，请检查网络',
  'TIMEOUT': '请求超时，请重试',
  'SERVER_ERROR': '服务器错误，请稍后重试',
  
  // 权限错误
  'UNAUTHORIZED': '未授权，请重新登录',
  'FORBIDDEN': '无权访问此资源',
  
  // 资源错误
  'NOT_FOUND': '资源不存在',
  'CONFLICT': '资源冲突',
  
  // 默认错误
  'DEFAULT': '操作失败，请稍后重试'
}

/**
 * 中文：获取友好的错误消息
 * English: Get user-friendly error message
 */
function getUserFriendlyMessage(error: AxiosError): string {
  // 网络错误
  if (!error.response) {
    if (error.message.includes('timeout')) {
      return errorMessages['TIMEOUT']
    }
    return errorMessages['NETWORK_ERROR']
  }
  
  // HTTP 错误
  const { status, data } = error.response
  
  // 尝试从响应中获取错误代码
  const errorCode = (data as any)?.code || (data as any)?.error
  
  if (errorCode && errorMessages[errorCode]) {
    return errorMessages[errorCode]
  }
  
  // 根据状态码返回错误消息
  switch (status) {
    case 401:
      return errorMessages['UNAUTHORIZED']
    case 403:
      return errorMessages['FORBIDDEN']
    case 404:
      return errorMessages['NOT_FOUND']
    case 409:
      return errorMessages['CONFLICT']
    case 500:
      return errorMessages['SERVER_ERROR']
    default:
      return (data as any)?.message || errorMessages['DEFAULT']
  }
}

// 中文：请求拦截器
// English: Request interceptor
api.interceptors.request.use(
  (config) => {
    // 中文：可以在这里添加认证 token
    // English: Add auth token here if needed
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 中文：响应拦截器
// English: Response interceptor
api.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error) => {
    // 401 = cookie 过期或未登录，强制刷新页面触发 App.vue 的认证检查
    if (error.response?.status === 401) {
      const url = error.config?.url ?? ''
      // 排除登录接口本身，避免循环
      if (!url.includes('/auth/login') && !url.includes('/auth/status')) {
        window.location.reload()
      }
    }
    
    // 添加友好的错误消息到错误对象
    error.userMessage = getUserFriendlyMessage(error)
    
    // 开发环境下打印详细错误
    const isDev = typeof process !== 'undefined' && process.env?.NODE_ENV === 'development'
    if (isDev) {
      console.error('API Error:', {
        url: error.config?.url,
        method: error.config?.method,
        status: error.response?.status,
        message: error.message,
        userMessage: error.userMessage
      })
    }
    
    return Promise.reject(error)
  }
)

// 扩展 AxiosError 类型，添加用户友好消息
declare module 'axios' {
  interface AxiosError {
    userMessage?: string
  }
}

export default api
