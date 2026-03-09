// API 响应缓存工具
// 用于缓存 GET 请求响应，减少重复请求

interface CacheEntry<T> {
  data: T
  timestamp: number
  ttl: number
}

class ApiCache {
  private cache = new Map<string, CacheEntry<any>>()
  
  /**
   * 获取缓存数据
   */
  get<T>(key: string): T | null {
    const entry = this.cache.get(key)
    if (!entry) return null
    
    // 检查是否过期
    if (Date.now() - entry.timestamp > entry.ttl) {
      this.cache.delete(key)
      return null
    }
    
    return entry.data
  }
  
  /**
   * 设置缓存数据
   */
  set<T>(key: string, data: T, ttl: number = 60000): void {
    this.cache.set(key, {
      data,
      timestamp: Date.now(),
      ttl
    })
  }
  
  /**
   * 删除缓存
   */
  delete(key: string): void {
    this.cache.delete(key)
  }
  
  /**
   * 清空缓存
   */
  clear(): void {
    this.cache.clear()
  }
  
  /**
   * 清理过期缓存
   */
  cleanup(): void {
    const now = Date.now()
    for (const [key, entry] of this.cache.entries()) {
      if (now - entry.timestamp > entry.ttl) {
        this.cache.delete(key)
      }
    }
  }
}

// 单例实例
export const apiCache = new ApiCache()

// 定期清理过期缓存（每 5 分钟）
setInterval(() => {
  apiCache.cleanup()
}, 5 * 60 * 1000)
