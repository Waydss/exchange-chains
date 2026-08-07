# chain_builder

Пакет строит цепочки обмена (chain) между пользователями: A отдаёт предмет, хочет получить предмет B; B отдаёт предмет, хочет C... — до тех пор, пока А не получит желанный предмет. Результат — замкнутый цикл обмена.

При любых изменениях в предметах (изменением считается создание новой записи, и замена флага `Status` отличный от `searching`) -- цепочки будут перерасчитываться для данного предмета

## Работа с пакетом

`StartChainRequestWorkers()` — точка входа пакета, вызываемая в `main.go`. Функция начинает `chain_worket_count` количество `ChainRequest` обработчиков. Предварительно, каждый `ChainRequest` запрос обрабатывается в `handleChainRequest()`, чтобы отсечь невалидные данные и их формат

Если данные верны - запускается `buildClosedChains()`

Предназначенный способ добавления запросов на создание цепей - это вызов `EnqueChainRequst(req)`, который добавит запрос в очередь. 

По мере работы BFS-алгоритма, вызываается `buildChainFromState()`, который в конце своей работы записывает просчитанные и закрытые цепи в БД. Получить цепи можно оттуда.

## Архитектура

Пакет разеделён на следующие модули:

| Файл | Роль |
|---|---|
| `models.go` | Внутренние модели: `userCandidate`, `searchState`, `candidate`, `barterEdge` |
| `repository.go` | Контракт `Repository` — операции с БД, которые нужны построителю |
| `postgres_repository.go` | Реализация `Repository` на pgx |
| `chains.go` | Ядро: очередь запросов, BFS-алгоритм, скоринг, кэш кандидатов |
| `fuzzy.go` | Нечёткое сравнение названий (Levenshtein) |
| `utils.go` | Вспомогательные функции: сигнатура цепи, глубина, скор кандидата |
| `handler.go` | HTTP-обработчики: запуск построения и read-only просмотр результатов (не предназначены для использования в продакшне) |

## Внешние зависимости

- `internal/exchange` — целевые доменные модели `Chain`, `Barter`, `GivenItem` (= `item.Item`), `WishedItem` (= `wish.WishedItem`).
- `internal/item` — статусы/состояния предметов.
- `github.com/jackc/pgx/v5` — доступ к PostgreSQL.

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

- `GetAccount` — аккаунт + его wishes + searching-gives.
- `GetAccountGives` — все searching-gives аккаунта (для обратного отбора замыкания `bestClosingGive`).
- `FindCandidateAccounts` — SQL-фильтр кандидатов + подгрузка их wishes/gives.
- `CreateChain` — транзакция: `INSERT chains` с `ON CONFLICT (signature) DO NOTHING`; затем `chain_barters` (с позицией), затем `chain_participants` (уникальные участники: root + from/to всех бартеров). Если сигнатура уже есть — `pgx.ErrNoRows`, цепь не перезаписывается.
- `ListChains` — все цепи + вложенные бартеры (give/wish со статусами) + полный `root_wish`. Используется read-only endpoint'ом.

## HTTP API

> [!warning]
> Данный пакет не предназначен для внешнего доступа через ендпоинты. А перечисленные ниже - только для внутреннего тестирования

- `POST /api/v1/chain` (auth) — тело `{user_id, wished_item_id}` → `202 Accepted`, запуск построения.
- `GET /api/v1/chains` (публичный) — все цепи с вложенными бартерами в JSON.

Все пороги скора помечены `NEEDS FINETUNING` — подбираются эмпирически.

## Тестирование

`chain_integration_test.go` — интеграционный тест на живом Postgres: сидит 100k пользователей через `internal/simulation`, запускает реальный BFS и проверяет, что цепи пишутся и читаются (`ListChains`). Пропускается без env `SIMULATION_TEST_DATABASE_URL`. Запуск:

```bash
$env:SIMULATION_TEST_DATABASE_URL="postgres://exchange:exchange@localhost:5432/exchange?sslmode=disable"
go test -p 1 ./internal/chain_builder/ -timeout 25m
```

## Известные проблемы / TODO

- `giveCandidatesCache` и `accountGivesCache` не инвалидируются.
- Нестабильность fuzzy-оценивания при получении кандидатов
- `handler.go` для `CreateChainsForItem` принимает `user_id` из тела запроса, а не из JWT.
- `use_cache` помечен deprecated, но используется.
