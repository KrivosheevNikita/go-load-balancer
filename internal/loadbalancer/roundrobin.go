package loadbalancer

import "sync"

// roundRobin реализует алгоритм по кругу
type roundRobin struct {
	backends []Backend
	mu       sync.Mutex
	idx      int
}

func NewRoundRobin(bs []Backend) Selector {
	return &roundRobin{backends: bs}
}

// Выбирает следующий живой backend по очереди
func (rr *roundRobin) Next() Backend {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	n := len(rr.backends)
	for i := 0; i != n; i++ {
		b := rr.backends[(rr.idx+i)%n]
		if b.Alive() {
			rr.idx = (rr.idx + i + 1) % n
			return b
		}
	}
	return nil
}
