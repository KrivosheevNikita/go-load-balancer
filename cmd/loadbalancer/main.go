package main

import (
	"flag"
	"net/http"

	"loadbalancer/internal/api"
	"loadbalancer/internal/config"
	"loadbalancer/internal/healthcheck"
	"loadbalancer/internal/loadbalancer"
	"loadbalancer/internal/logging"
	"loadbalancer/internal/proxy"
	"loadbalancer/internal/ratelimiter"
	"loadbalancer/internal/server"
	"loadbalancer/internal/storage"

	"github.com/joho/godotenv"
)

// Путь к конфиг файлу передается как флаг командной строки
var cfgPath = flag.String("config", "configs/config.yaml", "path to config file")

func main() {
	_ = godotenv.Load()

	flag.Parse()

	// Загрузка конфигурации из yaml
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		logging.L.Error("config load failed", "error", err)
		return
	}

	// Инициализация backend-серверов
	var bs []loadbalancer.Backend
	for _, backend := range cfg.Backends {
		b, err := loadbalancer.NewBackend(backend.URL)
		if err != nil {
			logging.L.Error("invalid backend URL", "url", backend.URL, "error", err)
			return
		}
		bs = append(bs, b)
	}

	// Выбор алгоритма балансировки
	var sel loadbalancer.Selector
	switch cfg.Algorithm {
	case "least_conn":
		sel = loadbalancer.NewLeastConnections(bs)
	case "random":
		sel = loadbalancer.NewRandom(bs)
	default:
		sel = loadbalancer.NewRoundRobin(bs)
	}

	px := proxy.New(sel)

	// healthcheck для проверки состояния бэкендов
	checker := healthcheck.New(bs, cfg.HealthDuration())
	checker.Start()
	defer checker.Stop()

	// Подключение к базе данных
	repo, err := storage.NewPostgres(cfg.GetDSN())
	if err != nil {
		logging.L.Error("db connect failed", "error", err)
		return
	}

	// Инициализация ratelimiter
	rl := ratelimiter.NewStore(cfg.DefaultLimit.Capacity, cfg.DefaultLimit.RatePerSec, repo)
	defer rl.Close() // Сохраняем состояние токенов перед завершением

	// Регистрация API-хендлеров для управления клиентами
	mux := http.NewServeMux()
	apiHandler := api.NewHandler(rl)
	apiHandler.Register(mux)

	// Регистрация основного хендлера
	mux.Handle("/", server.BuildHandler(px, rl.Middleware))

	// Запуск HTTP-сервера
	srv := server.New(cfg, mux)
	if err := srv.Start(); err != nil {
		logging.L.Error("server run failed", "error", err)
	}
}
