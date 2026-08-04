# Цепочка обмена — backend

Backend MVP для многостороннего обмена вещами. Сейчас доступны регистрация по логину и паролю, JWT-авторизация и получение собственного публичного профиля.

## Требования

- Docker с Docker Compose v2;
- для локального запуска без Docker: Go 1.25+ и доступный PostgreSQL;
- для локального линтинга: golangci-lint v2.

## Запуск

```bash
cp .env.example .env
docker compose up --build
```

API доступен на `http://localhost:8080`:

- `GET /health` — проверка процесса;
- `GET /ready` — проверка подключения к PostgreSQL.
- `POST /api/v1/auth/register` — регистрация;
- `POST /api/v1/auth/login` — вход;
- `GET /api/v1/me` — профиль по JWT.

Базовый CORS разрешает локальный frontend на портах `3000` и `5173` для `localhost` и `127.0.0.1`.

## Миграции

Применить миграции после запуска PostgreSQL:

```bash
docker compose --profile tools run --rm migrate
```

Откатить одну миграцию:

```bash
docker compose --profile tools run --rm migrate -path=/migrations -database="postgres://exchange:exchange@postgres:5432/exchange?sslmode=disable" down 1
```

Учетные данные в командах и `.env.example` предназначены только для локальной разработки.

Если старая начальная миграция уже применялась к тестовому volume, перед новым запуском удалите этот локальный volume командой `docker compose down -v`.

## Примеры запросов

Регистрация:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"login":"vadim","password":"password123"}'
```

Вход:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"login":"vadim","password":"password123"}'
```

Публичный профиль текущего пользователя:

```bash
curl http://localhost:8080/api/v1/me \
  -H 'Authorization: Bearer <access_token>'
```

## Проверки

```bash
go test ./...
golangci-lint run
```

Без локальной установки линтера:

```bash
docker run --rm -v "$(pwd):/app" -w /app golangci/golangci-lint:v2.5.0 golangci-lint run
```

## Конфигурация

Обязательные переменные приложения:

- `APP_PORT` — HTTP-порт;
- `APP_ENV` — имя окружения;
- `DATABASE_URL` — PostgreSQL connection URL;
- `SHUTDOWN_TIMEOUT` — таймаут graceful shutdown в формате Go duration, например `10s`.
- `JWT_SECRET` — секрет подписи access token;
- `JWT_TTL` — время жизни access token, например `168h`.

## Пока не реализовано

