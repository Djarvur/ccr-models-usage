# add-ccr-models-usage — дизайн

## Context

Пользователь уже запускает [Claude Code Router](https://github.com/musistudio/claude-code-router) с пятью настроенными провайдерами (`zai`, `opencode`, `opencode-a`, `yadro`, `yadev`). У самого CCR нет встроенного отчёта об использовании. У каждого провайдера свой дашборд и своя схема лимитов; пользователь хочет одну команду, которая показывает «где я ближе всего к удушению».

Результаты проб (см. `[[opencode-go-no-api-for-usage]]` и `[[zai-quota-endpoint]]` для сырых доказательств):

- **z.ai** отдаёт недокументированный monitoring-эндпоинт, который принимает API-ключ как Bearer. Ответ — JSON, парсится тривиально.
- **OpenCode Go** НЕ отдаёт usage-API. API-ключ авторизует инференс; данные usage лежат в браузерном HTML-дашборде за Iron-зашифрованной cookie `auth`. Единственное публичное — эндпоинт chat-completions и список `/v1/models`.
- Остальные настроенные провайдеры (`yadro`, `yadev`) — корпоративный LiteLLM-прокси, публичного per-key usage-API нет, поэтому утилита будет писать для них `skip (no adapter)`.

У конфига CCR есть два неочевидных свойства (см. `[[ccr-config-shape]]`):

- `api_base_url` — это полный путь, а не только origin. Один и тот же провайдер может встречаться как две записи с разными путями, но одним ключом (реальный случай: `opencode` + `opencode-a`). Квота у них общая.
- Две записи на одном хосте с разными ключами (`yadro` + `yadev`) — квоты независимые.

Дизайн-решения, выбранные пользователем (из Q&A в explore-фазе):

- **Ключ дедупликации**: `(hostname, api_key)`.
- **OpenCode Go auth**: пара (username, password) — из переменных окружения или JSON-файла. API-ключ НЕ подходит (см. [[opencode-go-no-api-for-usage]]), cookie браузера НЕ в скоупе v1.
- **Дисциплина TDD**: каждая нетривиальная единица работы начинается с падающего теста. Имплементация пишется ровно настолько, чтобы тест прошёл; после зелёного допускается рефакторинг.
- **Линтинг CI**: `golangci-lint` с `linters: enable: all` запускается и в локальном `make lint`, и в CI; без зелёного линта PR не мёрджится.
- **Формат вывода**: простой текст, без таблиц, одна строка на лимит.
- **Неизвестные сайты**: выводятся со статусом `skip (no adapter)`.

## Goals / Non-Goals

### Goals

- Один статический Go-бинарник, без сторонних зависимостей (только stdlib).
- Раскладка строго под Go-конвенции: точка входа в `cmd/ccr-models-usage/main.go`, всё остальное — приватные пакеты в `internal/`. Никакого верхнеуровневого `main` пакета в корне.
- Читать `~/.claude-code-router/config.json` (переопределяется через `-config`).
- Дедуплицировать по `(hostname, api_key)`, группировать по хосту.
- Для каждой уникальной записи запустить подходящий адаптер, если он известен. Иначе — `skip`.
- Для каждого известного адаптера вывести все лимиты, которые вернул провайдер, одной строкой `label  N%  (опц.: remaining / reset)`.
- Один адаптер на хост провайдера. Добавить нового = один новый файл в `internal/providers/<name>/`.
- Параллельные запросы (по горутине на запись) с небольшим пулом воркеров.
- **TDD-дисциплина**: каждое публичное поведение сначала покрывается падающим тестом; имплементация пишется только чтобы тест позеленел.
- **Линтинг**: `golangci-lint` с `linters: enable: all` — must-pass для PR.

### Non-goals

- Web-UI, TUI, режим статус-бара.
- Запись на диск (никакого кеша, никакой истории).
- Адаптер для корпоративного `yadro` / `yadev` LiteLLM-прокси (нет публичного API).
- OpenCode Go: auto-извлечение cookie из браузера. v1 поддерживает только env + JSON-файл; discovery из браузера — потенциальный v2, если пользователь захочет.
- OAuth, refresh-токены или что угодно, требующее состояния между запусками.
- Настраиваемые ретраи HTTP, кастомные таймауты или прокси помимо дефолтов stdlib.

## Decisions

### 1. Только Go stdlib, без сторонних пакетов

**Почему:** у пользователя готовый `go.mod`-репо без зависимостей, поверхность маленькая (HTTP + JSON + regexp), а отсутствие зависимостей означает, что `go install` работает без танцев с прокси. Прецедент — [[opgginc/opencode-bar]] тоже без экзотики.

**Альтернативы, которые рассматривали:**

- `resty` / `gentleman` для HTTP — пакет ради ~30 строк `net/http`, которые иначе написали бы сами.
- `tablewriter` / `text/tabwriter` для вывода — пользователь явно выбрал «без таблиц».

### 2. Интерфейс адаптера провайдера

```go
// provider/adapter.go
type Adapter interface {
    Host() string                                    // "api.z.ai", "opencode.ai"
    Fetch(ctx context.Context, key string) (Limits, error)  // API-ключ из CCR
    NeedsSessionCreds() bool                          // true → адаптер сам читает env/файл
}
type Limit struct {
    Label    string  // "TIME_LIMIT", "rolling (5h)" и т.п.
    UsedPct  float64 // 0–100
    ResetAt  *time.Time
    Detail   string  // опц.: "remaining 82", разбивка по моделям и т.п.
}
type Limits []Limit
```

`Provider` несёт `(host, key, name)` из дедуплицированной записи конфига. Реестр сопоставляет `host → Adapter`. Неизвестный хост → `nil` адаптер, рендер `skip (no adapter)`.

**Зачем так:** один файл на провайдера, реестр отвязывает поиск от реализации, `NeedsSessionCreds()` позволяет адаптеру OpenCode Go опционально подгрузить env/файл, не засоряя main-цикл.

### 3. Дедуп как `(hostname, api_key)`

```go
// config/dedup.go
seen := map[string]*Provider{} // ключ = host + "|" + key
for _, p := range raw.Providers {
    u, err := url.Parse(p.APIBaseURL)
    h := strings.ToLower(u.Hostname())
    k := p.APIKey
    id := h + "|" + k
    if existing, ok := seen[id]; ok {
        existing.Models = union(existing.Models, p.Models)
        continue
    }
    seen[id] = &Provider{Host: h, Key: k, Name: p.Name, Models: p.Models}
}
```

Парсинг URL через `url.Parse` (stdlib) для робастности; нас интересует только `Hostname()`, который отбрасывает порт, userinfo, путь, query и фрагмент.

### 4. Адаптер z.ai

```text
GET https://api.z.ai/api/monitor/usage/quota/limit
Authorization: Bearer <key>
Accept: application/json
Timeout: 10s

если 404 → фолбэк на GET https://open.bigmodel.cn/api/monitor/usage/quota/limit
если 401 → возврат ошибки "auth failed"
```

Ответ парсится в плоский `Limits`. Имя тарифа из `data.level` рендерится как заголовок строки.

### 5. Адаптер OpenCode Go

Резолв учётных данных (по порядку):

1. Переменные окружения `OPENCODE_GO_USERNAME` и `OPENCODE_GO_PASSWORD`. Если обе заданы и непустые после `Trim`, используются.
2. JSON-файл `$XDG_CONFIG_HOME/ccr-models-usage/opencode-go.json`, или `~/.config/ccr-models-usage/opencode-go.json` на macOS. Файл — JSON-объект со строковыми полями `username` и `password`. Если оба поля есть и непустые после `Trim`, используются.
3. Если ни один источник не сконфигурирован, адаптер возвращает sentinel-ошибку с сообщением `credentials missing — set OPENCODE_GO_USERNAME and OPENCODE_GO_PASSWORD` (рендерер показывает это дословно).

Если в любом источнике задано только одно из двух, адаптер возвращает ошибку с именем пропущенной переменной.

Аутентификация: конкретный flow (POST на эндпоинт входа, обмен пары на session cookie, и т.п.) — деталь реализации; единственный MUST — итоговая сессия должна позволять читать usage. Тесты пишутся против `httptest.Server`, чтобы не зависеть от реальной формы auth-эндпоинта; контракт — функция `authenticate(ctx, username, password) (session, error)` где `session` — что-то пригодное для подписи запроса за usage.

Запрос usage:

```http
GET <dashboard-URL для авторизованной сессии>
Cookie: <session-cookie>
User-Agent: <UA реального desktop-Chrome на macOS — обязателен, иначе Cloudflare вернёт challenge>
Timeout: 15s
```

Парсинг: регулярка по телу ответа ищет блоки `__next_f.push([1, "...{...}..."])`. Каждый блок содержит escape-нутую JSON-строку; парсер делает unescape, затем `json.Unmarshal` в:

```go
type usageWindow struct {
    UsagePercent float64 `json:"usagePercent"`
    ResetInSec   int     `json:"resetInSec"`
}
type usagePayload struct {
    RollingUsage *usageWindow `json:"rollingUsage"`
    WeeklyUsage  *usageWindow `json:"weeklyUsage"`
    MonthlyUsage *usageWindow `json:"monthlyUsage"`
}
```

Второй паттерн ловит форму `$R[N]={...rollingUsage: $R[M]={...}...}` (те же поля). `usagePercent` — это **использовано**, не остаток; выводим как есть.

`User-Agent` обязателен — Cloudflare блокирует дефолтный Go UA HTML-страницей challenge. Используем `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 ...` (тот же UA, что и в opencode-bar, проверено).

### 6. Рендерер

```go
// render/text.go
type Row struct {
    Header  string   // "" или "zai api.z.ai"
    Limits  []Limit
    Skip    string   // "no adapter" или "cookie missing — ..."
}
func Write(w io.Writer, rows []Row) { ... }
```

Вывод: строки в стабильном порядке (порядок конфига, затем сортировка по хосту в пределах одного ключа). Пустая строка между блоками. Без цветов.

### 7. Параллелизм

Запросы запускаются параллельно через `errgroup` с лимитом 4. У каждого адаптера свой context с таймаутом 10–15 секунд. Медленный провайдер не блокирует остальных.

### 8. CLI-поверхность

```text
ccr-models-usage [-config PATH]

  -config string  путь к конфигу CCR (по умолчанию "~/.claude-code-router/config.json")
```

Всё. Никаких `-json`, `-provider`, `-verbose`. Легко расширить позже.

### 9. Раскладка пакетов

```text
ccr-models-usage/                 (репозиторий, module github.com/Djarvur/ccr-models-usage)
├── cmd/
│   └── ccr-models-usage/
│       └── main.go               package main — только разбор флагов и вызов internal-пакетов
├── internal/
│   ├── config/                   разбор config.json, дедуп по (host, key)
│   ├── provider/                 интерфейс Adapter, типы Limit/Limits, реестр, параллельный fetcher
│   ├── providers/
│   │   ├── zai/                  адаптер z.ai (Bearer)
│   │   └── opencodego/           адаптер OpenCode Go (username+password)
│   └── render/                   текстовый рендерер
├── .golangci.yml                 конфиг golangci-lint
├── go.mod
├── go.sum
└── Makefile                      цели build, test, lint, ci
```

**Почему `cmd/` + `internal/`:** это Go-конвенция (Effective Go, golang-standards/project-layout). `internal/` гарантирует, что ни один внешний модуль не сможет импортировать наши пакеты — `provider` и адаптеры могут рефакториться свободно. `cmd/ccr-models-usage/main.go` — единственное место, где живёт `package main`.

### 10. Методология TDD и линтинг

Цикл на каждую нетривиальную единицу работы:

1. **Красный.** Написать падающий тест в `internal/<pkg>/<pkg>_test.go` (или `*_integration_test.go` под build-тегом). Тест описывает наблюдаемое поведение.
2. **Зелёный.** Написать минимум кода, чтобы тест прошёл. Никакого «заодно поправлю» — рефакторинг отдельно.
3. **Рефактор.** После зелёного: убрать дублирование, вытащить хелперы, переименовать. Тесты не меняются (или только переименовываются вместе с кодом).
4. **Линт.** `golangci-lint run ./...` — без warning'ов и ошибок. Конфиг — `.golangci.yml` с `linters: enable: all` плюс разумные `exclusions` для тестовых файлов и сгенерированных моков.

Makefile:

```makefile
.PHONY: build test lint ci
build:
    go build -o bin/ccr-models-usage ./cmd/ccr-models-usage
test:
    go test ./...
lint:
    golangci-lint run ./...
ci: lint test build
```

`make ci` — must-pass для каждого PR.

## Risks / Trade-offs

- **HTML-скрап OpenCode Go хрупкий** — формат `__next_f.push(...)` это внутренняя деталь сериализации Next.js. Митигация: парсим защитно (регулярка ищет все три окна независимо, отсутствие любого — ок), и при отсутствии распарсенных окон выдаём понятную ошибку «dashboard markup may have changed». План — переключиться на настоящий API, когда OpenCode его опубликует (issue #18648).
- **Пароль пользователя в env/файле** — чувствительные данные, лежат на диске. Митигация: env-переменные не логируются (правило прописано в [[ccr-config-shape]] для API-ключей, то же применяется к password); файл должен иметь permissions `0600`; программа читает, но никогда не пишет. Сообщения об ошибках НЕ содержат значения `password` — только имя переменной.
- **Конкретный flow аутентификации OpenCode Go — деталь реализации** (POST на эндпоинт входа и т.п.). Митигация: выделен в `authenticate(ctx, user, pass) (session, error)`, чтобы тесты шли против `httptest.Server` и не ломались при смене формы auth-эндпоинта.
- **Без сторонних зависимостей нет ретраев, нет экспоненциального backoff** — если запрос упал по транзиентной причине, пользователь запускает заново. Допустимо для одноразовой CLI; позже можно добавить `cenkalti/backoff` при необходимости.
- **Спуфинг User-Agent** для обхода Cloudflare — серая зона. Митигация: используется только для запроса дашборда OpenCode, не для инференса; не для обхода какой-либо авторизации.
- **`golangci-lint` с `enable: all`** может ругаться на стилистические вещи, которые команда считает допустимыми. Митигация: `issues: exclusions` для тестовых файлов и сгенерированных моков; зафиксировать в `.golangci.yml` версию линтера.
- **`/etc/ssl/certs` не учитывается на урезанных системах** — stdlib `crypto/tls` использует системное хранилище по умолчанию на macOS, так что для целевой среды это не проблема.

## Migration Plan

Нет. Это новый бинарник; мигрировать нечего.

Откат: `go install` предыдущей версии или `rm` бинарника.

## Open Questions

- (Решено) Можно ли обменять API-ключ OpenCode Go на сессию? **Нет** — см. [[opencode-go-no-api-for-usage]].
- (Решено) Есть ли JSON-API для usage OpenCode Go? **Нет** — публичного `/v1/usage` не существует.
- (Открыт) Делать ли адаптер для корпоративного `yadro` / `yadev` LiteLLM-прокси, если у прокси есть `/v1/usage`? Пользователь может решить это в follow-up, если захочет. v1 выходит без него.
- (Открыт) Нужен ли `--json` сейчас или потом? v1 без него; слой рендеринга изолирован, добавить — один файл.
- (Открыт) Нужен ли режим periodic-refresh / watch? Вне scope; пользователь может обернуть в `watch -n 60`.
