
# Load Balancer
HTTP-балансировщик нагрузки на Go
## Используемые технологии

- Go 
- PostgreSQL
- Docker

## Установка и запуск

### 1. Клонируйте проект

```bash
git clone https://github.com/KrivosheevNikita/go-load-balancer.git
cd go-load-balancer
```


### 2. Настройте переменные окружения и конфигурацию
#### `env`

```bash
cp .env.example .env
```

#### `configs/config.yaml`

Пример:
```yaml
listen_addres: ":8080"
algorithm: "round_robin"
health_interval: "5s"

backends:
  - name: backend-1
    url:  "url1"
  - name: backend-2
    url:  "url2"

default_rate_limit:
  capacity: 10
  rate_per_sec: 12
```



### 3.1. Запуск через Docker (рекомендуется)

```bash
docker compose up --build
```
### 3.2. Запуск без Docker
```bash
go run ./cmd/loadbalancer -config configs/config.yaml
```
Балансировщик будет доступен на `http://localhost:8080`


## Тестирование

### Юнит-тесты

```bash
go test ./... -race
```

### Интеграционные тесты

```bash
go test ./tests -v -race
```

### Бенчмарк

```bash
go test -bench=. ./tests
```

### Apache Bench

```bash
ab -n 5000 -c 1000 http://localhost:8080/
```
[Результат нагрузочного тестирования](./assets/apache_bench_result.png)

## API

### Добавление клиента

```bash
curl -X POST http://localhost:8080/clients \
  -H "Content-Type: application/json" \
  -d '{"client_id":"user1", "capacity":100, "rate_per_sec":10}'
```

### Удаление клиента

```bash
curl -X DELETE http://localhost:8080/clients/user1
```

### Получение списка клиентов

```bash
curl http://localhost:8080/clients
```

## Реализовано

- Round-Robin, Least Connections, Random алгоритмы балансировки
- Health Check с автоматическим исключением недоступных бэкендов
- Rate Limiting (Token Bucket) с поддержкой настройки разных лимитов для разных клиентов, автоматического пополнения токенов
- Конфигурация через YAML
- Персистентность токенов (PostgreSQL)
- API для управления клиентами
- Логирование
- Обеспечена одновременная обработка нескольких запросов и потокобезопасность


## Доступные алгоритмы балансировки

- `round_robin` (по умолчанию)
- `least_conn`
- `random`
