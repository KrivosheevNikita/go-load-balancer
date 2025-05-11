package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Описывает один backend сервер
type Backend struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// Описывает лимиты токенов по умолчанию для клиентов
type RateLimit struct {
	Capacity   int64 `yaml:"capacity"`     // Максимальное количество токенов
	RatePerSec int64 `yaml:"rate_per_sec"` // Скорость пополнения токенов в секунду
}

type Config struct {
	ListenAddr     string        `yaml:"listen_addres"`      // Адрес, на котором слушает HTTP-сервер
	Algorithm      string        `yaml:"algorithm"`          // Способ балансировки
	Backends       []Backend     `yaml:"backends"`           // Список серверов
	DefaultLimit   RateLimit     `yaml:"default_rate_limit"` // Лимиты по умолчанию
	HealthInterval string        `yaml:"health_interval"`    // Интервал проверки серверов
	DbDSN          string        // Строка подключения к PostgreSQL
	healthDur      time.Duration // Интервал для healthcheck
}

// Загрузка конфига из yaml
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	if cfg.Algorithm == "" {
		cfg.Algorithm = "round_robin"
	}
	if cfg.HealthInterval == "" {
		cfg.HealthInterval = "3s"
	}
	d, err := time.ParseDuration(cfg.HealthInterval)
	if err != nil {
		return nil, err
	}
	cfg.healthDur = d

	if env := os.Getenv("DB_DSN"); env != "" {
		cfg.DbDSN = env
	}
	if cfg.DbDSN == "" {
		return nil, fmt.Errorf("DB_DSN isn't defined in .env")
	}

	return &cfg, nil
}

// GetDSN возвращает строку подключения к бд
func (c *Config) GetDSN() string { return c.DbDSN }

// HealthDuration возвращает интервал между healthcheck
func (c *Config) HealthDuration() time.Duration {
	return c.healthDur
}
