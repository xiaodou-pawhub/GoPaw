import axios from 'axios'

// 中文：创建 Axios 实例，配置基础 URL 和超时
// English: Create Axios instance with base URL and timeout
const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

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
    } else {
      console.error('API Error:', {
        url: error.config?.url,
        method: error.config?.method,
        status: error.response?.status,
        message: error.message
      })
    }
    return Promise.reject(error)
  }
)

export default api
