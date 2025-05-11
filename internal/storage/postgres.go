package storage

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// ClientConfig соответствует строке таблицы clients
type ClientConfig struct {
	ClientID   string
	Capacity   int64
	RatePerSec int64
}

// Состояние токенов клиента
type BucketState struct {
	ClientID string
	Tokens   int64
}

// Интерфейс репозитория клиентов и состояния токенов
type ClientRepository interface {
	List(ctx context.Context) ([]ClientConfig, error)
	Upsert(ctx context.Context, cfg ClientConfig) error
	Delete(ctx context.Context, clientID string) error

	InitBucketsTable() error
	LoadBucketState(ctx context.Context) ([]BucketState, error)
	SaveBucketState(ctx context.Context, st BucketState) error

	ExistsClient(ctx context.Context, clientID string) (bool, error)
}

type pgRepo struct {
	db *sql.DB
}

// Подключается к БД, создает таблицы и возвращает репозиторий
func NewPostgres(dsn string) (ClientRepository, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}

	// Создает таблицу clients, если не существует
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS clients(
        client_id   TEXT PRIMARY KEY,
        capacity    BIGINT NOT NULL,
        rate_per_sec BIGINT NOT NULL
    );`)
	if err != nil {
		return nil, err
	}

	repo := &pgRepo{db: db}

	// Создает таблицу bucket_state
	if err := repo.InitBucketsTable(); err != nil {
		return nil, err
	}

	return repo, nil
}

// Возвращает всех клиентов из таблицы clients
func (p *pgRepo) List(ctx context.Context) ([]ClientConfig, error) {
	rows, err := p.db.QueryContext(ctx, "SELECT client_id, capacity, rate_per_sec FROM clients")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ClientConfig
	for rows.Next() {
		var c ClientConfig
		if err := rows.Scan(&c.ClientID, &c.Capacity, &c.RatePerSec); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Вставляет или обновляет клиента
func (p *pgRepo) Upsert(ctx context.Context, cfg ClientConfig) error {
	_, err := p.db.ExecContext(ctx, `INSERT INTO clients(client_id, capacity, rate_per_sec)
        VALUES($1,$2,$3)
        ON CONFLICT(client_id) DO UPDATE
          SET capacity = EXCLUDED.capacity,
              rate_per_sec = EXCLUDED.rate_per_sec`,
		cfg.ClientID, cfg.Capacity, cfg.RatePerSec)
	return err
}

// Удаляет клиента по id
func (p *pgRepo) Delete(ctx context.Context, clientID string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM clients WHERE client_id=$1", clientID)
	return err
}

// Создает таблицу bucket_state, если еще не создана
func (p *pgRepo) InitBucketsTable() error {
	_, err := p.db.Exec(`CREATE TABLE IF NOT EXISTS bucket_state(
        client_id   TEXT PRIMARY KEY REFERENCES clients(client_id) ON DELETE CASCADE,
        tokens      BIGINT NOT NULL,
        updated_at  TIMESTAMPTZ DEFAULT NOW()
    );`)
	return err
}

// Возвращает сохраненные значения токенов для всех клиентов
func (p *pgRepo) LoadBucketState(ctx context.Context) ([]BucketState, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT client_id, tokens FROM bucket_state`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BucketState
	for rows.Next() {
		var st BucketState
		if err := rows.Scan(&st.ClientID, &st.Tokens); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// Сохраняет или обновляет количество токенов клиента
func (p *pgRepo) SaveBucketState(ctx context.Context, st BucketState) error {
	_, err := p.db.ExecContext(ctx, `INSERT INTO bucket_state(client_id, tokens)
        VALUES($1,$2)
        ON CONFLICT(client_id) DO UPDATE
          SET tokens = EXCLUDED.tokens,
              updated_at = NOW()`,
		st.ClientID, st.Tokens)
	return err
}

// Возвращает true, если client_id существует в таблице clients
func (p *pgRepo) ExistsClient(ctx context.Context, clientID string) (bool, error) {
	var exist bool
	query := `SELECT EXISTS(SELECT 1 FROM clients WHERE client_id = $1)`
	if err := p.db.QueryRowContext(ctx, query, clientID).Scan(&exist); err != nil {
		return false, err
	}
	return exist, nil
}
