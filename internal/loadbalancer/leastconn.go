package loadbalancer

// leastConn реализует "наименьшее количество соединений"
type leastConn struct {
	backends []Backend
}

// Возвращает селектор, выбирающий сервер с наименьшей нагрузкой
func NewLeastConnections(bs []Backend) Selector {
	return &leastConn{backends: bs}
}

// Next выбирает backend с минимальным количеством активных соединений
func (lc *leastConn) Next() Backend {
	var best Backend
	for _, b := range lc.backends {
		if !b.Alive() {
			continue // Игнорирует мертвые сервера
		}
		if best == nil || b.Conns() < best.Conns() {
			best = b
		}
	}
	return best
}
