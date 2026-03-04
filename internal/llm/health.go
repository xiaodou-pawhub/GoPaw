package llm

import (
	"sync"
	"time"
)

// ProviderStatus 定义提供商的实时运行状态
type ProviderStatus string

const (
	StatusHealthy   ProviderStatus = "healthy"   // 健康
	StatusCooldown  ProviderStatus = "cooldown"  // 冷却中 (网络波动/限流)
	StatusDegraded  ProviderStatus = "degraded"  // 已降级 (鉴权失败/欠费)
)

// HealthTracker 追踪所有 Provider 的健康状况
type HealthTracker struct {
	mu sync.RWMutex
	// providerID -> failureCount
	failures map[string]int
	// providerID -> retryAt (time when cooldown ends)
	cooldownUntil map[string]time.Time
	// providerID -> lastError
	lastErrors map[string]string
}

// GlobalHealthTracker 全局单例
var GlobalHealthTracker = &HealthTracker{
	failures:      make(map[string]int),
	cooldownUntil: make(map[string]time.Time),
	lastErrors:    make(map[string]string),
}

// RecordFailure 记录一次失败并计算冷却
func (t *HealthTracker) RecordFailure(id string, err error, isPersistent bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.lastErrors[id] = err.Error()
	if isPersistent {
		// 永久错误 (如 401, 欠费)
		t.failures[id] = 100 
		return
	}

	t.failures[id]++
	count := t.failures[id]
	
	// 指数退避冷却: 1min, 2min, 4min, 8min... (max 1h)
	waitSec := 60 * (1 << (uint(count) - 1))
	if waitSec > 3600 {
		waitSec = 3600
	}
	t.cooldownUntil[id] = time.Now().Add(time.Duration(waitSec) * time.Second)
}

// RecordSuccess 成功后重置计数
func (t *HealthTracker) RecordSuccess(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.failures, id)
	delete(t.cooldownUntil, id)
	delete(t.lastErrors, id)
}

// GetStatus 获取当前状态及详细信息
func (t *HealthTracker) GetStatus(id string) (ProviderStatus, string, time.Time) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if err, ok := t.lastErrors[id]; ok && t.failures[id] >= 100 {
		return StatusDegraded, err, time.Time{}
	}

	until, ok := t.cooldownUntil[id]
	if ok && time.Now().Before(until) {
		return StatusCooldown, t.lastErrors[id], until
	}

	return StatusHealthy, "", time.Time{}
}

// IsAvailable 判断当前是否可用
func (t *HealthTracker) IsAvailable(id string) bool {
	status, _, _ := t.GetStatus(id)
	return status == StatusHealthy
}
