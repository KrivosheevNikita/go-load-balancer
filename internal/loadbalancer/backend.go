package loadbalancer

import (
	"net/url"
	"sync/atomic"
)

// backend реализует интерфейс Backend, представляя один реальный сервер
type backend struct {
	url         *url.URL     // Адрес
	alive       atomic.Bool  // Состояние: жив/мертв
	activeConns atomic.Int64 // Количество активных соединений
}

// Создание нового бэкенда
func NewBackend(raw string) (*backend, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	b := &backend{url: u}
	b.alive.Store(true) // По умолчанию считаем живым
	return b, nil
}

// Возвращает URL бэкенда
func (b *backend) URL() *url.URL { return b.url }

// Возвращает текущее состояние сервера
func (b *backend) Alive() bool { return b.alive.Load() }

// Обновляет состояние сервера
func (b *backend) SetAlive(v bool) { b.alive.Store(v) }

// Увеличивает счётчик активных соединений
func (b *backend) Inc() { b.activeConns.Add(1) }

// Уменьшает счётчик активных соединений
func (b *backend) Done() {
	if b.activeConns.Load() > 0 {
		b.activeConns.Add(-1)
	}
}

// Возвращает текущее число активных соединений
func (b *backend) Conns() int64 { return b.activeConns.Load() }
