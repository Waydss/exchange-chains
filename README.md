# Цепочка обмена — backend

Backend MVP для многостороннего обмена вещами. Доступны регистрация с именем, JWT-авторизация, публичный профиль и создание предметов с фотографиями.

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
- `POST /api/v1/items` — создание предмета по JWT;
- `GET /swagger/index.html` — интерактивная Swagger UI.

Загруженные фотографии доступны по относительным URL вида `/uploads/items/{item_id}/{filename}`.

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
  -d '{"login":"vadim","password":"password123","name":"Вадим"}'
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

Создание предмета с несколькими фотографиями:

```bash
curl -X POST http://localhost:8080/api/v1/items \
  -H 'Authorization: Bearer <access_token>' \
  -F 'title=Гитара Yamaha F310, 6 струн' \
  -F 'description=Исправна, есть небольшие царапины' \
  -F 'condition=good' \
  -F 'category_name=Гитары' \
  -F 'price_min=40000' \
  -F 'price_max=50000' \
  -F 'wanted_category_name=Палатки' \
  -F 'photos=@/path/to/photo-1.jpg' \
  -F 'photos=@/path/to/photo-2.webp'
```

Фотографии необязательны. Допускается не более 5 файлов по 10 МиБ каждый; поддерживаются JPEG, PNG и WEBP.

## Swagger

После запуска приложения интерактивная документация доступна по адресу [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html).

Для авторизованного запроса нажмите **Authorize** и вставьте значение целиком в формате `Bearer <access_token>`. Swagger использует API-key схему и не добавляет префикс `Bearer` автоматически.

Общая информация API находится в `cmd/api/main.go`.

После изменения Swagger-аннотаций перегенерируйте документацию из корня проекта:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/api/main.go -o docs --parseInternal
```

Каталог `docs/` генерируется автоматически — не редактируйте его вручную и добавляйте в коммит вместе с изменёнными аннотациями. Документация подключена к бинарнику через blank import, поэтому после генерации приложение нужно пересобрать и перезапустить.

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
- `UPLOAD_DIR` — директория для загруженных файлов;
- `MAX_UPLOAD_SIZE` — максимальный размер одной фотографии в байтах;
- `MAX_ITEM_PHOTOS` — максимальное количество фотографий предмета.

При запуске через Docker файлы хранятся в `/app/uploads` внутри backend-контейнера и сохраняются в именованном volume `item_uploads` после пересоздания контейнера.

## Пока не реализовано

- просмотр, изменение и удаление предметов;
- matching и построение цепочек обмена;
- облачное хранилище, сжатие и модерация изображений.
