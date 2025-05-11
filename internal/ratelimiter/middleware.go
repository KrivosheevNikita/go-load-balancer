package ratelimiter

import (
	"encoding/json"
	"net/http"

	"loadbalancer/internal/logging"
)

// Проверяет лимит по заголовку x-api-key или ip
func (s *Store) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Определяем идентификатор клиента
		id := r.Header.Get("x-api-key")
		if id == "" {
			id = r.RemoteAddr
		}

		// Если нет токенов, то возвращает 429
		if !s.getBucket(id).Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    429,
				"message": "rate limit exceeded",
			})
			logging.L.Warn("rate limit exceeded", "client", id)
			return
		}

		// Если разрешен, то передаем дальше
		logging.L.Info("rate limit allow", "client", id)
		next.ServeHTTP(w, r)
	})
}
