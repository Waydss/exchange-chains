# chain_builder

Пакет строит цепочки обмена (chain) между пользователями: A отдаёт предмет, хочет получить предмет B; B отдаёт предмет, хочет C... — до тех пор, пока А не получит желанный предмет. Результат — замкнутый цикл обмена.

При создании нового предмета сервис `internal/item` автоматически ставит запрос на пересчёт цепочек для желания нового предмета (см. `EnqueueChainRequest`). Пересчёт также можно запустить вручную через `POST /api/v1/chain`.

## Работа с пакетом

Точка входа — `Service` (см. `service.go`), создаётся через `NewService(repository, userService, rejectionStore)` в `main.go`/`internal/app`:

- `EnqueueChainRequest(userID, wishedItemID int64)` — добавляет запрос на построение цепочек в асинхронную очередь (буфер 100). Это основной способ постановки запросов.
- `StartChainRequestWorkers()` — запускает `chain_worker_count` воркеров, которые разбирают очередь.
- Каждый запрос обрабатывается в `handleChainRequest()`: через `UserRequester.GetUserCandidateBase` загружаются рейтинги пользователя, затем `FillRootCandidate` заполняет его wishes и searching-gives, и данные валидируются. Если запрос валиден — запускается `buildClosedChains()`.
- По мере работы BFS-алгоритма `buildChainFromState()` записывает найденные замкнутые цепи в БД. Прочитать их можно через `GetChainsByUser`, `GetChainsByItem`, `GetChainByID` (см. HTTP API) или `ListChains` (только для внутреннего тестирования).
- `AcceptChain` фиксирует согласие участника и блокирует цепочку, когда согласились все. `DeclineChain` отменяет цепочку в PostgreSQL и записывает отклонённый предмет в Redis через `rejectionStore` (`RejectionWriter`).

## Архитектура

Пакет разделён на следующие модули:

| Файл | Роль |
|---|---|
| `service.go` | Очередь запросов, запуск воркеров, `UserRequester`, `AcceptChain`/`DeclineChain`, read-only доступ к цепочкам |
| `models.go` | Внутренние модели: `userCandidate`, `searchState`, `candidate`, `barterEdge` |
| `repository.go` | Контракт `Repository` — операции с БД, которые нужны построителю |
| `postgres_repository.go` | Реализация `Repository` на pgx |
| `chains.go` | Ядро: BFS-алгоритм, скоринг, кэш кандидатов |
| `fuzzy.go` | Нечёткое сравнение названий (Levenshtein) |
| `utils.go` | Вспомогательные функции: сигнатура цепи, глубина, скор кандидата |
| `handler.go` | HTTP-обработчики: запуск построения и работа с цепочками |
| `store.go` | Контракты `RejectionWriter`/`RejectionStore` — временные отказы от предметов |
| `redis_store.go` | Реализация отказов на Redis |
| `scheduler.go` | Ежедневная очистка просроченных отказов |

## Внешние зависимости

- `internal/exchange` — целевые доменные модели `Chain`, `Barter`, `GivenItem` (= `item.Item`), `WishedItem` (= `wish.WishedItem`).
- `internal/item` — статусы/состояния предметов.
- `internal/user` — `UserRequester.GetUserCandidateBase` для получения рейтингов участников (через интерфейс, без импорта структур пользователя).
- `github.com/jackc/pgx/v5` — доступ к PostgreSQL.
- `github.com/redis/go-redis/v9` — временные отказы от предметов.

## Модели

- `userCandidate` — аккаунт с его `Wishes` и `Gives` (только `status='searching'`). `CurrentGive()` — самый свежий предлагаемый предмет.
- `searchState` — вершина BFS: текущий пользователь, текущий даваемый предмет, цепочка родителей, глубина, лимит глубины, накопленный скор.
- `candidate` — кандидат для следующего слоя BFS: аккаунт + какой его wish совпал + скоры.
- `barterEdge` — ребро графа (одна передача): `From → To`, `Give`, `Wish`.

## Тюнинг-константы

В `chains.go`:

| Константа | Значение | Что делает |
|---|---|---|
| `max_barters_per_chain` | 5 | макс. длина цепи |
| `candidate_limit` | 100 | лимит SQL-выборки кандидатов |
| `min_fuzzy_score` | 0.45 | порог нечёткого совпадения названий |
| `min_final_score` | 10 | порог итогового скора кандидата |
| `chain_worker_count` | 1 | число воркеров очереди запросов |

Константы скоринга помечены как `NEEDS FINETUNING` — они подбираются эмпирически.

## BFS-алгоритм (`chains.go`)

`buildClosedChains(ctx, root, rootWishIndex)`:

1. Берётся root-пользователь и одно его желание (`rootWish`).
2. Для каждого найденного кандидата создаётся `searchState` (глубина 1).
3. Цикл по слоям: `expandLayerParallel` обрабатывает весь текущий слой параллельно, воркеров — по `GOMAXPROCS`.
4. `expandState` на каждом состоянии:
   - **Замыкание**: `bestClosingGive` отбирает из ВСЕХ предметов последнего пользователя (`status='searching'`) тот, что лучше всего соответствует `rootWish` (`wishMatchesGive` + `fuzzySimilarity >= min_fuzzy_score`), берёт максимальный fuzzy — цепь замкнута → `buildChainFromState` пишет её в БД. Это обратный отбор к `buildCandidatesForGive`: вместо проверки только свежайшего `Give` учитываются все предметы участника.
   - **Расширение**: ищет кандидатов, чьи желания совпадают с текущим `Give`, и добавляет новые состояния.
5. Ограничения:
   - `max_barters_per_chain = 5` — максимум передач в цепи.
   - `allowedDepth(shadowRating)` — лимит глубины по теневому рейтингу: рейтинг ≥ 10 → глубина 1, ≥ 6 → 2, ≥ 3 → 3, иначе 5.
   - Запрет повторного входа: `state.HasInPath` не даёт вернуться к уже посещённому пользователю.
6. Дубликаты отсекаются по сигнатуре цепи (`chainSignature`).

### Поиск кандидатов

`rankNextCandidates` → `Repository.FindCandidateAccounts`:

SQL-фильтр: пользователь с желанием в той же категории, цена желания покрывает цену предмета (`w.price_min <= give.PriceMin AND w.price_max >= give.PriceMax`), и у пользователя есть предмет со статусом `searching`. Лимит — `candidate_limit = 100`.

Пост-фильтры в Go:
- `fuzzySimilarity(give.Title, wish.TitleKeywords) >= min_fuzzy_score (0.45)`;
- `candidateScore(...) >= min_final_score (10)`;
- исключение `sourceID` (текущего) и уже пройденных.

Сортировка: по `MatchScore`, затем `FuzzyScore`, затем `PublicRating`.

### Кэш кандидатов

`giveCandidatesCache` (`sync.Map`) кэширует результат `buildCandidatesForGive` по `give.ID`. В глубоких слоях один и тот же предмет запрашивается многократно, кэш убирает повторные SQL-запросы.

`accountGivesCache` (`sync.Map`) кэширует все searching-gives аккаунта по `account.ID` для обратного отбора `bestClosingGive`.

Хиты/промахи считаются в атомарных счётчиках `cacheHits`/`cacheMisses` и логируются.

**Известная проблема**: кэш не инвалидируется при изменении данных о предметах пользователей.

## Скоринг (`fuzzy.go`, `utils.go`)

- `fuzzySimilarity(a, b)` — `1 - Levenshtein(a,b) / maxLen(a,b)` после нормализации (lowercase, trim, дефисы/подчёркивания → пробел). Идентичные строки дают 1.
- `candidateScore(fuzzy, publicRating, shadowRating) = fuzzy*100 + publicRating*3 - shadowRating*7`. Может быть отрицательным при высоком теневом рейтинге.
- `buildChainFromState` суммирует `MatchScore` по всем бартерам в `chain.Score`. **Скор может быть отрицательным** — репозиторий клампит его в 0 при записи, т.к. в БД `CHECK (score >= 0)`.

## Репозиторий (`postgres_repository.go`)

- `FillRootCandidate` — по базе `userCandidate` (ID и рейтинги) подгружает wishes и searching-gives root-пользователя.
- `GetAccountGives` — все searching-gives аккаунта (для обратного отбора замыкания `bestClosingGive`).
- `FindCandidateAccounts` — SQL-фильтр кандидатов + подгрузка их wishes/gives.
- `CreateChain` — транзакция: `INSERT chains` с `ON CONFLICT (signature) DO NOTHING`; затем `chain_barters` (с позицией), затем `chain_participants` (уникальные участники: root + from/to всех бартеров). Если сигнатура уже есть — `pgx.ErrNoRows`, цепь не перезаписывается.
- `GetChainsByUser` / `GetChainsByItem` — цепочки пользователя / конкретного желания с вложенными бартерами (через общий хелпер `getChains`).
- `GetChainByID` — одна цепочка; проверяет, что цепочка существует и пользователь — её участник.
- `AcceptChain` — транзакция: помечает участника как принявшего, при согласии всех блокирует цепочку (`status='locked'`) и резервирует предметы (`reserveChainItems`).
- `DeclineChain` — транзакция: помечает участника как отказавшегося и отменяет цепочку, возвращая ID предмета, который участник должен был получить.
- `ListChains` — все цепи + вложенные бартеры (give/wish со статусами) + полный `root_wish`. Используется read-only endpoint'ом (в роутере не зарегистрирован).
- `UpdateChainStatus` — заглушка `not implemented`.

## HTTP API

> [!warning]
> `POST /api/v1/chain` не предназначен для продакшена: принимает `user_id` из тела запроса, а не из JWT. Остальные endpoint'ы требуют JWT и сверяют пользователя с токена.

- `POST /api/v1/chain` (auth) — тело `{user_id, wished_item_id}` → `202 Accepted`, запуск построения.
- `GET /api/v1/chains/by-user` (auth) — query `user_id`, все цепочки пользователя с вложенными бартерами.
- `GET /api/v1/chains/by-item` (auth) — query `user_id` + `wished_item_id`, все цепочки конкретного желания.
- `GET /api/v1/chains/{chainId}` (auth) — одна цепочка; доступ только участнику.
- `POST /api/v1/chains/{chainId}/accept` (auth) — согласие участника; при согласии всех — блокировка цепочки.
- `POST /api/v1/chains/{chainId}/decline` (auth) — отмена цепочки в PostgreSQL + запись отклонённого предмета в Redis.

## Отказы от предметов (`store.go`, `redis_store.go`, `scheduler.go`)

При отказе от цепочки получаемый предмет блокируется для пользователя на срок `REJECTED_ITEM_TTL` (по умолчанию 336h). Для каждого пользователя используется Sorted Set `rejected_items:{user_id}`: member содержит ID товара, score — Unix timestamp окончания блокировки. `RejectionStore` дополнительно фильтрует кандидатов, исключая заблокированные предметы. `Scheduler` раз в сутки около 03:00 по `Europe/Moscow` находит ключи `rejected_items:*` через `SCAN` и удаляет просроченные элементы. Рассчитан на один экземпляр backend.

## Тестирование

`chain_integration_test.go` — интеграционный тест на живом Postgres: сидит 100k пользователей через `internal/simulation`, запускает реальный BFS и проверяет, что цепи пишутся и читаются (`ListChains`). Пропускается без env `SIMULATION_TEST_DATABASE_URL`. Запуск:

```bash
$env:SIMULATION_TEST_DATABASE_URL="postgres://exchange:exchange@localhost:5432/exchange?sslmode=disable"
go test -p 1 ./internal/chain_builder/ -timeout 25m
```

## Известные проблемы / TODO

- `giveCandidatesCache` и `accountGivesCache` не инвалидируются.
- Нестабильность fuzzy-оценивания при получении кандидатов.
- `CreateChainsForItem` принимает `user_id` из тела запроса, а не из JWT.
- `UpdateChainStatus` в репозитории — заглушка `not implemented`.
- `DiscardChain` в handler'е — заглушка `not implemented`.
- `use_cache` помечен deprecated, но используется.
