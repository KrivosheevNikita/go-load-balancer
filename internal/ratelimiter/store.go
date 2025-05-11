package ratelimiter

import (
	"context"
	"sync"
	"time"

	"loadbalancer/internal/logging"
	"loadbalancer/internal/storage"
)

// Управляет множеством Buckets по клиентам
type Store struct {
	defaultCap  int64              // Емкость по умолчанию
	defaultRate int64              // Скорость пополнения по умолчанию
	buckets     map[string]*Bucket // Мапа client_id - bucket
	mu          sync.RWMutex

	persistInterval time.Duration // Интервал сохранения в БД
	stopPersist     chan struct{} // Завершение сохранения
	refillInterval  time.Duration // Интервал пополнения токенов
	stopRefill      chan struct{} // Завершение пополнения

	repo storage.ClientRepository // Интерфейс доступа к БД
}

// Создает Store, загружает клиентов и запускает фоновые циклы
func NewStore(defaultCap, defaultRate int64, repo storage.ClientRepository) *Store {
	s := &Store{
		defaultCap:      defaultCap,
		defaultRate:     defaultRate,
		buckets:         make(map[string]*Bucket),
		repo:            repo,
		persistInterval: 5 * time.Second,
		stopPersist:     make(chan struct{}),
		refillInterval:  1 * time.Second,
		stopRefill:      make(chan struct{}),
	}

	// Загрузка клиентов и лимитов из БД
	if repo != nil {
		if list, err := repo.List(context.Background()); err == nil {
			for _, c := range list {
				s.buckets[c.ClientID] = NewBucket(c.Capacity, c.RatePerSec)
			}
			logging.L.Info("loaded client configs", "count", len(s.buckets))
		}

		// Восстановление состояния токенов
		states, err := repo.LoadBucketState(context.Background())
		if err == nil {
			s.mu.Lock()
			for _, st := range states {
				if b, exists := s.buckets[st.ClientID]; exists {
					b.mu.Lock()
					if st.Tokens < b.capacity {
						b.tokens = st.Tokens
					}
					b.mu.Unlock()
				}
			}
			s.mu.Unlock()
			logging.L.Info("restored token states", "count", len(states))
		}
	}

	// Запуск фоновых циклов в горутинах
	go s.persistLoop()
	go s.refillLoop()
	return s
}

// Добавление нового клиента, сохранение в БД и создание bucket
func (s *Store) AddClient(clientID string, cfg storage.ClientConfig) error {
	if s.repo != nil {
		if err := s.repo.Upsert(context.Background(), cfg); err != nil {
			logging.L.Error("db upsert client failed", "client", clientID, "error", err)
			return err
		}
		logging.L.Info("db upsert client", "client", clientID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.buckets[clientID] = NewBucket(cfg.Capacity, cfg.RatePerSec)
	logging.L.Info("bucket created", "client", clientID)
	return nil
}

// Удаление клиента из БД
func (s *Store) DeleteClient(clientID string) error {
	if s.repo != nil {
		if err := s.repo.Delete(context.Background(), clientID); err != nil {
			logging.L.Error("db delete client failed", "client", clientID, "error", err)
			return err
		}
		logging.L.Info("db delete client", "client", clientID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.buckets, clientID)
	logging.L.Info("bucket removed", "client", clientID)
	return nil
}

// Возвращает конфигурации клиентов из БД
func (s *Store) ListClients() map[string]storage.ClientConfig {
	if s.repo == nil {
		return map[string]storage.ClientConfig{}
	}
	list, err := s.repo.List(context.Background())
	if err != nil {
		logging.L.Error("list clients failed", "error", err)
		return map[string]storage.ClientConfig{}
	}
	out := make(map[string]storage.ClientConfig, len(list))
	for _, c := range list {
		out[c.ClientID] = c
	}
	logging.L.Info("listed clients", "count", len(out))
	return out
}

// Возвращает bucket клиента, создавая его с дефолтными значениями при отсутствии
func (s *Store) getBucket(id string) *Bucket {
	s.mu.RLock()
	b := s.buckets[id]
	s.mu.RUnlock()
	if b != nil {
		return b
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if b = s.buckets[id]; b != nil {
		return b
	}
	// Попробовать найти клиента в БД
	if s.repo != nil {
		list, _ := s.repo.List(context.Background())
		for _, c := range list {
			if c.ClientID == id {
				b = NewBucket(c.Capacity, c.RatePerSec)
				s.buckets[id] = b
				return b
			}
		}
	}
	// Используем дефолтные параметры
	b = NewBucket(s.defaultCap, s.defaultRate)
	s.buckets[id] = b
	return b
}

// Сохраняет текущее число токенов клиентов в БД раз в N секунд
func (s *Store) persistLoop() {
	if s.repo == nil {
		return
	}
	ticker := time.NewTicker(s.persistInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.RLock()
			for id, b := range s.buckets {
				if exist, err := s.repo.ExistsClient(context.Background(), id); err != nil {
					logging.L.Error("exists client failed", "client", id, "error", err)
					continue
				} else if !exist {
					continue
				}

				b.mu.Lock()
				tokens := b.tokens
				b.mu.Unlock()

				if err := s.repo.SaveBucketState(context.Background(),
					storage.BucketState{ClientID: id, Tokens: tokens},
				); err != nil {
					logging.L.Error("save bucket_state failed", "client", id, "error", err)
				}
			}
			s.mu.RUnlock()

		case <-s.stopPersist:
			return
		}
	}
}

// Пополняет токены в каждом bucket каждую секунду
func (s *Store) refillLoop() {
	ticker := time.NewTicker(s.refillInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.RLock()
			for _, b := range s.buckets {
				b.mu.Lock()
				now := time.Now()
				t := now.Sub(b.last).Seconds()
				add := int64(t * float64(b.refill))
				if add > 0 {
					if b.tokens+add > b.capacity {
						b.tokens = b.capacity
					} else {
						b.tokens += add
					}
					b.last = now
				}
				b.mu.Unlock()
			}
			s.mu.RUnlock()
		case <-s.stopRefill:
			return
		}
	}
}

// Останавливает фоновые циклы
func (s *Store) Close() {
	close(s.stopPersist)
	close(s.stopRefill)
}
