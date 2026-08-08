# simulation

Пакет генерирует и очищает большие объёмы детерминированных тестовых данных (users, items, wishes) для нагрузочного тестирования приложения и проверки цепочек обмена (см. `internal/chain_builder`).

## Архитектура

| Файл | Роль |
|---|---|
| `simulation.go` | Публичный API: `SeedData`, `ClearData`, `SeedOptions`, `SeedResult` |
| `repository.go` | Контракт `Repository` — bulk-операции, которые нужны сидеру |
| `postgres_repository.go` | Реализация `Repository` на pgx с `COPY` |
| `generator.go` | Генератор данных: `generateBatch`, пулы названий, условия предметов |

## Публичный API

### `SeedData(ctx, repo, opts) (SeedResult, error)`

Наполняет БД:

1. `opts.normalized()` — приводит опции к дефолтам.
2. `repo.LoadCategories` — каталог категорий; если пусто — ошибка (сначала нужны миграции).
3. `repo.LoadLastIDs` — текущие `MAX(id)` у users/items/wishes, чтобы не конфликтовать с существующими данными.
4. Хэширует общий пароль `simulatedPassword` (`bcrypt` через `security.PasswordHasher`).
5. Инициализирует детерминированный RNG: `rand.New(rand.NewPCG(opts.RandomSeed, 1))`.
6. Идёт по батчам размера `BatchSize`; на каждом батче `generateBatch` → `BulkInsertUsers` → `BulkInsertItems` → `BulkInsertWishes`.
7. `repo.SyncSequences` — подтягивает serial-последовательности до фактических ID.

Возвращает `SeedResult{Users, Items, Wishes, Elapsed}`.

### `ClearData(ctx, repo) error`

Удаляет все данные сида. В реализации — `TRUNCATE users, items, wishes RESTART IDENTITY CASCADE`, что заодно каскадом чистит зависимые `chains`, `chain_barters`, `chain_participants`, `item_photos`. Категории сохраняются.

## SeedOptions и нормализация

| Поле | Дефолт | Назначение |
|---|---|---|
| `UserCount` | 100_000 | всего пользователей |
| `MinGives` / `MaxGives` | 1 / 5 | диапазон предметов на пользователя |
| `MinWishes` / `MaxWishes` | 1 / 5 | диапазон желаний на пользователя |
| `RandomSeed` | 0 | воспроизводимость генерации |
| `BatchSize` | 5_000 | сколько пользователей пишется одним COPY |

Правила `normalized()`: дефолты применяются только когда обе границы нулевые; отрицательные границы обрезаются до 0; `Max* < Min*` → `Max* = Min*`.

## Генератор (`generator.go`)

### Детерминизм

`generateBatch(rng, categories, passwordHash, opts, userStart, itemStart, wishStart, count)`:

- один поток RNG на весь прогон;
- ID присваиваются последовательно от стартовых смещений: `userID = userStart + i`, предметы/желания — аккумуляторы от `itemStart+1` / `wishStart+1`;
- тот же сид → те же данные (проверяется тестом `TestDeterministicWithSameSeed`).

### Что генерируется

**Users**:
- `Login`: `user_%08d` по ID — уникален, валиден под regex `^[a-z0-9][a-z0-9_]*$`.
- `Name`: `Пользователь N`.
- `PublicRating`: `rng.Float64() * 5` (0..5).
- `ShadowRating`: `rng.IntN(15)` (0..14).
- Общий `PasswordHash` для всех.

**Items** (от `MinGives` до `MaxGives` на пользователя):
- `Title`: случайное имя из `itemNamePools` по категории.
- `Condition`: случайное из `new/like_new/good/fair/poor`.
- `CategoryID` и `WantedCategoryID`: случайные категории.
- `PriceMin`: 1..100_000; `PriceMax = PriceMin + 0..50_000` (валидный диапазон).
- `Status`: `searching`.

**Wishes** (от `MinWishes` до `MaxWishes` на пользователя):
- `CategoryID`: случайная категория.
- `TitleKeywords`: имя из того же пула категории — важно для нечёткого матчинга в chain_builder (`fuzzySimilarity`).
- `PriceMin`: 1..50_000; `PriceMax = PriceMin + 0..500_000` (широкий диапазон, чтобы покрывать цены предметов).

### Пулы названий

`itemNamePools` — map `имя категории → названия`. Для 17 категорий из миграций (Гитары, Пианино, Палатки, Барабаны, Клавишные, Автомобили, Гаражи, Телефоны, Ноутбуки, Телевизоры, Холодильники, Велосипеды, Книги, Мебель, Инструменты, Садовая техника, Одежда). Неизвестная категория → фолбэк `Предмет категории %s`.

Один пул на категорию и для предметов, и для желаний: названия внутри категории совпадают, поэтому fuzzy-скоринг находит связи, а это делает цепи более плотными.

## Репозиторий (`postgres_repository.go`)

Интерфейс `Repository` (repository.go):

- `LoadCategories(ctx) ([]category.Category, error)`
- `LoadLastIDs(ctx) (userID, itemID, wishID int64, err error)`
- `BulkInsertUsers(ctx, []user.User) error`
- `BulkInsertItems(ctx, []item.Item) error`
- `BulkInsertWishes(ctx, []wish.WishedItem) error`
- `SyncSequences(ctx) error`
- `ClearData(ctx) error`

Реализация на `*pgxpool.Pool`:

- Bulk-вставки через `pgx.CopyFrom` — один round-trip на батч, колонки перечислены явно (включая `id`).
- `LoadLastIDs` — один запрос с `COALESCE(MAX(id))` по трём таблицам.
- `SyncSequences` — `setval(pg_get_serial_sequence(table,'id'), MAX(id))` по каждой таблице, чтобы последующие INSERT-ы (например, через API) не конфликтовали.
- `ClearData` — `TRUNCATE ... RESTART IDENTITY CASCADE`.

## Производительность

Измерено на живом Postgres 17 (Docker): 100k пользователей + ~251k предметов + ~251k желаний (~600k строк) за **~29 секунд**. Узкое место — COPY; пакетная запись по 5_000 пользователей.

## Тестирование

`generator_test.go` — unit-тесты:
- ID последовательны и связаны (user_id предметов/желаний внутри батча).
- Ценовые диапазоны валидны.
- Нормализация опций.
- Детерминизм при одинаковом сиде.

`postgres_repository_test.go` — интеграционный тест на живом Postgres: clear → seed → count → sync sequence → insert после сида → clear. Пропускается без env `SIMULATION_TEST_DATABASE_URL`.

Запуск:

```bash
$env:SIMULATION_TEST_DATABASE_URL="postgres://exchange:exchange@localhost:5432/exchange?sslmode=disable"
go test -p 1 ./internal/simulation/ -timeout 25m
```

## Запуск сида БД

Для запуска нужен живой Postgres с применёнными миграциями (иначе `LoadCategories` вернёт пусто и `SeedData` упадёт с ошибкой) и env `SIMULATION_TEST_DATABASE_URL`:

```bash
$env:SIMULATION_TEST_DATABASE_URL="postgres://exchange:exchange@localhost:5432/exchange?sslmode=disable"
```

### Только сид (без просчёта цепочек)

Прямой вызов из Go:

```go
repo := simulation.NewPostgresRepository(pool)
result, err := simulation.SeedData(ctx, repo, simulation.SeedOptions{
    UserCount:  100_000,
    MinGives:   1,
    MaxGives:   5,
    MinWishes:  1,
    MaxWishes:  5,
    RandomSeed: 7,
    BatchSize:  5_000,
})
```

Либо готовый интеграционный тест — сид + проверка содержимого БД (clear → seed → count → insert → clear):

```bash
go test -p 1 ./internal/simulation/ -timeout 25m
```

### Сид + просчёт цепочек для аккаунта

После сида подключить chain_builder к той же БД и построить цепи для конкретного аккаунта. `buildClosedChains` — внутренняя функция пакета chain_builder, поэтому снаружи запуск идёт через HTTP `POST /api/v1/chain` (тело `{user_id, wished_item_id}`), который ставит запрос в очередь сервиса; воркеры (`StartChainRequestWorkers`) уже запущены в `cmd/api/main.go`:

```bash
curl -X POST http://localhost:8080/api/v1/chain \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"user_id": 42, "wished_item_id": 123}'
```

Результат — `202 Accepted`; цепи появляются в БД асинхронно и читаются через `GET /api/v1/chains`. Внутри пакета chain_builder (тесты) построение вызывается напрямую: `svc.buildClosedChains(ctx, &account, wishIndex)`.

Готовый end-to-end пример — интеграционный тест `chain_builder/chain_integration_test.go` (`TestBuildChainOnSeededData`): сидит 100k пользователей (сид выше, `RandomSeed: 7`) и гоняет реальный BFS по первым 500 аккаунтам, проверяя, что цепи строятся и читаются:

```bash
go test -p 1 ./internal/chain_builder/ -timeout 25m
```

**Важно**: интеграционные тесты simulation и chain_builder разделяют живую БД и пересоздают данные, поэтому их нельзя запускать параллельно — используйте `-p 1`. Также обратите внимание на off-by-one: сид продолжает нумерацию от `MAX(id)+1`, а не перезаписывает существующие записи.

## Как используется

Основной потребитель — интеграционный тест `chain_builder/chain_integration_test.go`: он сидит 100k пользователей и гоняет реальный BFS, чтобы убедиться, что цепи строятся и читаются.
