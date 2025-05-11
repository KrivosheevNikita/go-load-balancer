package ratelimiter

import (
	"sync"
	"time"
)

// Реализация Token Bucket
type Bucket struct {
	capacity int64     // Максимальное количество токенов
	tokens   int64     // Текущее количество токенов
	refill   int64     // Сколько токенов пополняется в секунду
	last     time.Time // Момент последнего пополнения
	mu       sync.Mutex
}

// Создает новый bucket с заданной емкостью и скоростью пополнения
func NewBucket(capacity, refill int64) *Bucket {
	return &Bucket{
		capacity: capacity,
		tokens:   capacity,
		refill:   refill,
		last:     time.Now(),
	}
}

// Проверяет, можно ли выполнить запрос, если есть токен, то разрешает и списывает
func (b *Bucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	t := now.Sub(b.last).Seconds()
	newTokens := int64(t * float64(b.refill))

	if newTokens > 0 {
		// Пополняем токены, но не превышаем capacity
		b.tokens = min(b.capacity, b.tokens+newTokens)
		b.last = now
	}

	if b.tokens == 0 {
		return false
	}
	b.tokens--
	return true
}
