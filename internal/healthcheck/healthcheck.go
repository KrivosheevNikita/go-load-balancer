package healthcheck

import (
	"net/http"
	"sync"
	"time"

	"loadbalancer/internal/loadbalancer"
	"loadbalancer/internal/logging"
)

// Класс, отвечающий за проверку состояния бэкендов
type Checker struct {
	backends []loadbalancer.Backend // Список проверяемых серверов
	interval time.Duration          // Частота проверок
	stop     chan struct{}          // Сигнал остановки
	wg       sync.WaitGroup         // Ожидание завершения горутин
}

// Создание нового healthchecker
func New(bs []loadbalancer.Backend, d time.Duration) *Checker {
	return &Checker{backends: bs, interval: d, stop: make(chan struct{})}
}

// Запуск цикл проверок в отдельной горутине
func (c *Checker) Start() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(c.interval)
		for {
			select {
			case <-ticker.C:
				// Проверяем каждый backend параллельно
				for _, b := range c.backends {
					go c.check(b)
				}
			case <-c.stop:
				ticker.Stop()
				return
			}
		}
	}()
	logging.L.Info("healthchecker started", "interval", c.interval)
}

// Останавливает checker и дожидается завершения
func (c *Checker) Stop() {
	close(c.stop)
	c.wg.Wait()
	logging.L.Info("healthchecker stopped")
}

// Проверяет доступность одного сервера через HEAD-запрос
func (c *Checker) check(b loadbalancer.Backend) {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Head(b.URL().String())

	// Считаем alive, если нет ошибки
	alive := err == nil && resp.StatusCode < 500

	// Если статус изменился, то обновляем
	if alive != b.Alive() {
		b.SetAlive(alive)
		if alive {
			logging.L.Info("backend recovered", "backend", b.URL().String())
		} else {
			logging.L.Warn("backend down", "backend", b.URL().String())
		}
	}
}
