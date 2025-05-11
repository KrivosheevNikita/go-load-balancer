package proxy

import (
	"net/http"
	"net/http/httputil"

	"loadbalancer/internal/loadbalancer"
	"loadbalancer/internal/logging"
)

// Инкапсулирует выбор серверов
type Proxy struct {
	sel loadbalancer.Selector // Алгоритм выбора
}

// Создание нового Proxy с выбранным алгоритмом
func New(sel loadbalancer.Selector) *Proxy {
	return &Proxy{sel: sel}
}

// Основной обработчик HTTP-запросов
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const maxTries = 10 // Максимальное количество попыток на разные серверы

	for i := 0; i != maxTries; i++ {
		b := p.sel.Next()
		if b == nil {
			logging.L.Error("no backend alive")
			http.Error(w, "no backend available", http.StatusServiceUnavailable)
			return
		}

		logging.L.Info("selected backend", "url", b.URL().String())

		// Создание прокси на конкретный сервер
		rp := httputil.NewSingleHostReverseProxy(b.URL())

		// Обработка ошибок соединения
		rp.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			logging.L.Warn("error", "backend", b.URL().String(), "error", err)
			b.SetAlive(false) // Помечаем сервер как мертвый
			rw.WriteHeader(http.StatusBadGateway)
			_, _ = rw.Write([]byte("backend unreachable"))
		}

		// Увеличиваем счетчик активных соединений
		b.Inc()

		// Уменьшаем счетчик после получения ответа
		rp.ModifyResponse = func(resp *http.Response) error {
			b.Done()
			return nil
		}

		r.Host = b.URL().Host
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))

		rp.ServeHTTP(w, r)

		// Если backend ответил без ошибки
		if b.Alive() {
			return
		}
	}

	// Если ни один backend не сработал - ошибка
	logging.L.Error("all backends failed")
	http.Error(w, "all backends failed", http.StatusBadGateway)
}
