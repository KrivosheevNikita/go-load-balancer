package server

import (
	"net/http"
	"time"

	"loadbalancer/internal/logging"

	"github.com/google/uuid"
)

// Оборачивает ResponseWriter и запоминает код ответа
type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

// Добавляет req_id и логирует вход и выход запроса
func requestCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.New().String() // Идентификатор запроса
		start := time.Now()          // Фиксируем старт времени

		logging.L.Info("request start",
			"req_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
		)

		sw := &statusWriter{ResponseWriter: w, code: 200}
		next.ServeHTTP(sw, r) // Передаем обработку дальше

		logging.L.Info("request done",
			"req_id", reqID,
			"status", sw.code,
			"latency_ms", time.Since(start).Milliseconds(),
		)
	})
}
