// Package optimizer provides performance optimization utilities for channel plugins.
package optimizer

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	limiter *rate.Limiter
	mu      sync.RWMutex
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(requestsPerSecond int, burst int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
	}
}

// Wait 等待直到可以发送请求
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.limiter.Wait(ctx)
}

// Allow 检查是否允许发送请求
func (rl *RateLimiter) Allow() bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.limiter.Allow()
}

// BatchProcessor 批处理器
type BatchProcessor[T any] struct {
	items     []T
	mu        sync.Mutex
	maxSize   int
	timeout   time.Duration
	flushFunc func([]T) error
	timer     *time.Timer
}

// NewBatchProcessor 创建批处理器
func NewBatchProcessor[T any](maxSize int, timeout time.Duration, flushFunc func([]T) error) *BatchProcessor[T] {
	return &BatchProcessor[T]{
		items:     make([]T, 0, maxSize),
		maxSize:   maxSize,
		timeout:   timeout,
		flushFunc: flushFunc,
	}
}

// Add 添加项目到批处理
func (bp *BatchProcessor[T]) Add(item T) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.items = append(bp.items, item)

	// 达到批次大小，立即刷新
	if len(bp.items) >= bp.maxSize {
		return bp.flushLocked()
	}

	// 启动定时器
	if bp.timer == nil {
		bp.timer = time.AfterFunc(bp.timeout, func() {
			bp.mu.Lock()
			defer bp.mu.Unlock()
			if len(bp.items) > 0 {
				bp.flushLocked()
			}
		})
	}

	return nil
}

// Flush 手动刷新批处理
func (bp *BatchProcessor[T]) Flush() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.flushLocked()
}

func (bp *BatchProcessor[T]) flushLocked() error {
	if len(bp.items) == 0 {
		return nil
	}

	err := bp.flushFunc(bp.items)
	bp.items = bp.items[:0]
	if bp.timer != nil {
		bp.timer.Stop()
		bp.timer = nil
	}
	return err
}

// Cache 通用缓存
type Cache[T any] struct {
	data  map[string]cacheEntry[T]
	mu    sync.RWMutex
	ttl   time.Duration
	maxSize int
}

type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

// NewCache 创建缓存
func NewCache[T any](ttl time.Duration, maxSize int) *Cache[T] {
	return &Cache[T]{
		data:    make(map[string]cacheEntry[T]),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get 获取缓存值
func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok {
		var zero T
		return zero, false
	}

	if time.Now().After(entry.expiresAt) {
		var zero T
		return zero, false
	}

	return entry.value, true
}

// Set 设置缓存值
func (c *Cache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 清理过期条目
	if len(c.data) >= c.maxSize {
		c.cleanupLocked()
	}

	c.data[key] = cacheEntry[T]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete 删除缓存值
func (c *Cache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// Clear 清空缓存
func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]cacheEntry[T])
}

func (c *Cache[T]) cleanupLocked() {
	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.expiresAt) {
			delete(c.data, key)
		}
	}
}

// ConnectionPool 连接池
type ConnectionPool[T any] struct {
	connections []T
	mu          sync.Mutex
	factory     func() (T, error)
	closer      func(T) error
	maxSize     int
	current     int
}

// NewConnectionPool 创建连接池
func NewConnectionPool[T any](factory func() (T, error), closer func(T) error, maxSize int) *ConnectionPool[T] {
	return &ConnectionPool[T]{
		connections: make([]T, 0, maxSize),
		factory:     factory,
		closer:      closer,
		maxSize:     maxSize,
	}
}

// Get 获取连接
func (cp *ConnectionPool[T]) Get() (T, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	var zero T

	if len(cp.connections) > 0 {
		conn := cp.connections[len(cp.connections)-1]
		cp.connections = cp.connections[:len(cp.connections)-1]
		return conn, nil
	}

	if cp.current >= cp.maxSize {
		return zero, ErrPoolExhausted
	}

	cp.current++
	return cp.factory()
}

// Put 归还连接
func (cp *ConnectionPool[T]) Put(conn T) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if len(cp.connections) < cp.maxSize {
		cp.connections = append(cp.connections, conn)
	} else {
		cp.current--
		if cp.closer != nil {
			cp.closer(conn)
		}
	}
}

// Close 关闭连接池
func (cp *ConnectionPool[T]) Close() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	var lastErr error
	for _, conn := range cp.connections {
		if cp.closer != nil {
			if err := cp.closer(conn); err != nil {
				lastErr = err
			}
		}
	}
	cp.connections = nil
	cp.current = 0
	return lastErr
}

// ErrPoolExhausted 连接池耗尽错误
var ErrPoolExhausted = &poolExhaustedError{}

type poolExhaustedError struct{}

func (e *poolExhaustedError) Error() string {
	return "connection pool exhausted"
}
