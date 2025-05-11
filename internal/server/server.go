package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"loadbalancer/internal/config"
	"loadbalancer/internal/logging"
)

type HTTPServer struct {
	srv *http.Server
}

// Создает HTTP-сервер с тайм-аутами из конфига и переданным handler.
func New(cfg *config.Config, handler http.Handler) *HTTPServer {
	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	return &HTTPServer{srv: srv}
}

// Запускает сервер в горутине
func (s *HTTPServer) Start() error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logging.L.Info("server start", "addr", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.L.Error("listen fail", "error", err)
			os.Exit(1)
		}
	}()

	<-stop // Блокировка до сигнала завершения
	logging.L.Info("server shutdown")

	// Контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		logging.L.Error("shutdown error", "error", err)
		return err
	}
	logging.L.Info("server stopped")
	return nil
}

// Применяет цепочку middleware в обратном порядке
func chain(h http.Handler, m ...func(http.Handler) http.Handler) http.Handler {
	for i := len(m) - 1; i != -1; i-- {
		h = m[i](h)
	}
	return h
}

// Объединяет proxy, ratelimiter и логирование в один обработчик
func BuildHandler(proxy http.Handler, rlMw func(http.Handler) http.Handler) http.Handler {
	return chain(proxy,
		rlMw,
		requestCtx, // Логирует начало и конец запроса
	)
}
