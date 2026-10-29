# Project_cache

In-memory **LRU-кэш с TTL** на Go. Один и тот же кэш доступен по **HTTP (REST)** и **gRPC**.

## Возможности

- LRU-вытеснение: хеш-таблица + двусвязный список, `O(1)` для get / set / delete
- TTL для каждой записи: просроченные записи удаляются при чтении и фоновым воркером
- Потокобезопасность (мьютекс), проверено через `go test -race`
- HTTP API (chi) и gRPC API (protobuf) работают с одним общим кэшем
- Статистика: попадания, промахи, вытеснения, текущий размер
- Конфигурация через переменные окружения с проверкой при старте
- Graceful shutdown по SIGINT / SIGTERM
- Многоэтапная сборка Docker, запуск через `docker compose`
- Unit-тесты для кэша, конфига и HTTP-хендлеров

## Как это работает

Каждый ключ указывает на узел двусвязного списка. Голова списка — самая недавно использованная запись, хвост — самая давняя. При `Get` узел переносится в голову. Когда кэш заполнен, `Set` вытесняет хвост. В каждом узле хранится время истечения: чтение просроченной записи удаляет её и считается промахом, а фоновая горутина раз в `TTL / 2` очищает просроченные записи.

## Структура проекта

```
cmd/              точка входа приложения
internal/cache/   LRU-кэш с TTL
internal/config/  конфигурация из переменных окружения
internal/handler/ HTTP- и gRPC-обработчики
proto/            описание protobuf и сгенерированный код
```

## Быстрый старт

Через Docker:

```bash
cp .env.example .env
docker compose up --build
```

Без Docker (Go 1.26+):

```bash
go run ./cmd
```

## Конфигурация

| Переменная       | По умолчанию | Описание                          |
|------------------|--------------|-----------------------------------|
| `HTTP_PORT`      | `8080`       | Порт HTTP-сервера                 |
| `GRPC_PORT`      | `50051`      | Порт gRPC-сервера                 |
| `CACHE_CAPACITY` | `100`        | Максимальное число записей        |
| `CACHE_TTL`      | `5m`         | Время жизни записи (Go duration)  |

Неверные значения (например, TTL меньше или равный нулю) останавливают сервис при запуске с понятной ошибкой.

## HTTP API

| Метод    | Путь                | Описание             |
|----------|---------------------|----------------------|
| `POST`   | `/set`              | Сохранить значение   |
| `GET`    | `/get?key=<key>`    | Получить значение    |
| `DELETE` | `/delete?key=<key>` | Удалить значение     |
| `GET`    | `/stats`            | Статистика кэша      |
| `GET`    | `/healthz`          | Проверка состояния   |

```bash
curl -X POST localhost:8080/set -d '{"key":"a","value":"hello"}'
curl "localhost:8080/get?key=a"        # {"data":"hello"}
curl "localhost:8080/stats"            # {"data":{"currentSize":1,"evictions":0,"hits":1,"misses":0}}
curl -X DELETE "localhost:8080/delete?key=a"
```

Коды ответов: `200` — успех, `400` — некорректный запрос, `404` — ключ не найден.

## gRPC API

Сервис `cache.CacheService` с методами `Get`, `Set`, `Delete`, `Stats` (см. [`proto/cache.proto`](proto/cache.proto)). Пример с [grpcurl](https://github.com/fullstorydev/grpcurl); поля типа `bytes` в JSON передаются в base64 (`aGVsbG8=` — это `hello`):

```bash
grpcurl -plaintext -proto proto/cache.proto \
  -d '{"key":"a","value":"aGVsbG8="}' localhost:50051 cache.CacheService/Set

grpcurl -plaintext -proto proto/cache.proto \
  -d '{"key":"a"}' localhost:50051 cache.CacheService/Get

grpcurl -plaintext -proto proto/cache.proto \
  localhost:50051 cache.CacheService/Stats
```

## Тесты

```bash
go test -race -cover ./...
```

Перегенерация protobuf-кода:

```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/cache.proto
```

## Идеи для развития

- Один общий мьютекс: просто и корректно, но ограничивает пропускную способность при высокой конкуренции. Следующий шаг - шардирование кэша.
- Данные хранятся только в памяти: нет персистентности и кластеризации.
