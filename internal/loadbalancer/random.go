package loadbalancer

import "math/rand"

// random реализует случайный выбор сервера
type random struct {
	backends []Backend
}

func NewRandom(bs []Backend) Selector {
	return &random{backends: bs}
}

// Возвращает случайный живой backend
func (r *random) Next() Backend {
	alive := make([]Backend, 0, len(r.backends))
	for _, b := range r.backends {
		if b.Alive() {
			alive = append(alive, b)
		}
	}
	if len(alive) == 0 {
		return nil
	}
	return alive[rand.Intn(len(alive))]
}
