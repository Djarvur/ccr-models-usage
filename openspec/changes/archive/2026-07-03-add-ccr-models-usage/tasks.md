# Задачи

## 0. Дисциплина (TDD + lint)

Каждая нетривиальная единица работы идёт по циклу **red → green → refactor → lint**:

- [x] 0.1 **Красный.** Сначала пишется падающий тест в `internal/<pkg>/<pkg>_test.go`. Без него задача не считается «в работе».
- [x] 0.2 **Зелёный.** Имплементация добавляется ровно настолько, чтобы тест прошёл. Никакого preemptive-рефакторинга.
- [x] 0.3 **Рефактор.** После зелёного — убрать дубли, вытащить хелперы. Тесты не ломаются.
- [x] 0.4 **Линт.** `make lint` (т.е. `golangci-lint run ./...`) без ошибок и warning'ов. Это gate: PR с красным линтом не мёрджится.

Порядок подзадач внутри каждой секции ниже отражает этот цикл: сначала test, потом impl, потом refactor, потом lint. Секция считается закрытой, только когда все её подзадачи зелёные и `go test ./...` + `make lint` чистые.

## 1. Бутстрап

- [x] 1.1 Инициализировать `go.mod` для `github.com/Djarvur/ccr-models-usage` (Go 1.22+, case-sensitive — `Djarvur` с большой буквы), без сторонних рантайм-зависимостей
- [x] 1.2 Создать раскладку пакетов под Go-конвенцию: `cmd/ccr-models-usage/main.go` (package main) и приватные пакеты в `internal/`: `internal/config`, `internal/provider`, `internal/providers/zai`, `internal/providers/opencodego`, `internal/render`
- [x] 1.3 Добавить `Makefile` с целями `build`, `test`, `lint`, `ci` (`ci` = `lint test build`)
- [x] 1.4 Добавить `.golangci.yml` с `linters: enable: all` + разумные `issues.exclusions` для `*_test.go` и сгенерированных моков
- [x] 1.5 Закоммитить `go.sum` пустым; в README короткий блок «установить `golangci-lint` через официальный скрипт перед `make lint`»
- [x] 1.6 Смоук-тест: `make ci` на пустом репозитории зелёный (только `make lint` против пустого `internal/`)

## 2. Чтение конфига CCR

- [x] 2.1 **Красный.** Юнит-тесты на `config.Dedup`: схлопывание одинаковых (host, key), разделение по разным ключам, разделение по разным хостам, пропуск записей с битым `api_base_url` (без прерывания), пустой массив `Providers`
- [x] 2.2 **Зелёный.** Определить `config.RawConfig` / `config.RawProvider` под форму JSON (`encoding/json`); реализовать `config.Dedup(raw) []Provider` с ключом `(hostname, api_key)`; объединять списки `models` с дедупликацией, сохраняя порядок
- [x] 2.3 **Красный.** Юнит-тесты на флаг `-config`: путь по умолчанию (`~/.claude-code-router/config.json`, с раскрытием `~`), переопределение пути, отсутствие файла → однострочное сообщение в stderr + exit 2, невалидный JSON → то же
- [x] 2.4 **Зелёный.** Реализовать разбор флага `-config` + хелпер expandHome; понятные сообщения об ошибках
- [x] 2.5 **Рефактор.** Вытащить общий логический «read-or-error» слой, чтобы тесты на 2.3 не дублировали код 2.2
- [x] 2.6 Интеграционный тест на реальном `~/.claude-code-router/config.json` под build-тегом `integration` (вкл/выкл через `CCR_CONFIG_TEST`)
- [x] 2.7 `make lint` зелёный

## 3. Реестр провайдеров

- [x] 3.1 **Красный.** Юнит-тесты на `provider.Registry.Match`: совпадение по host, отсутствие адаптера → `nil`, два провайдера с одним host делят один экземпляр адаптера
- [x] 3.2 **Зелёный.** Определить интерфейс `provider.Adapter` (`Host()`, `Fetch(ctx, key) (Limits, error)`, `NeedsSessionCreds() bool`), типы `provider.Limit` (`Label`, `UsedPct`, `ResetAt`, `Detail`) и `provider.Limits`; реализовать `Registry` с `Register(Adapter)` и `Match(host) Adapter`
- [x] 3.3 **Красный.** Юнит-тесты на параллельный fetcher: медленный провайдер не блокирует быстрый, ошибка одного адаптера не валит прогон, упавший по таймауту показывает `skip (timeout)`, уважается лимит воркеров (≤4 одновременных)
- [x] 3.4 **Зелёный.** Параллельный fetcher на `golang.org/x/sync/errgroup` с `SetLimit(4)`, per-call context с таймаутом 10s; ошибка/таймаут изолируется, остальные продолжают
- [x] 3.5 **Рефактор.** Подумать, не вытащить ли «сборку результата» из main-цикла в отдельный helper (если main становится длинным)
- [x] 3.6 `make lint` зелёный

## 4. Адаптер z.ai

- [x] 4.1 **Красный.** Юнит-тесты на `providers/zai.Adapter` через `httptest.Server`: успех с одним типом лимита, успех с несколькими типами, HTTP 401 → ошибка содержит `auth failed`, HTTP 404 → повтор на CN-эндпоинт, оба эндпоинта 5xx → ошибка содержит последний статус, битое тело → ошибка содержит `decode`
- [x] 4.2 **Зелёный.** Реализовать `providers/zai.Adapter` (`Host() = "api.z.ai"`), запрос `GET https://api.z.ai/api/monitor/usage/quota/limit` с `Authorization: Bearer <key>`, `Accept: application/json`, таймаут 10s
- [x] 4.3 **Зелёный.** Парсинг ответа: для каждой записи в `data.limits[]` эмитить `Limit` (`type` → label через таблицу, `percentage` → `UsedPct`, `nextResetTime` → `ResetAt` через `time.UnixMilli`, `remaining` → `Detail` формата `remaining <N>`); тариф `data.level` прокидывается в рендерер как заголовок
- [x] 4.4 **Зелёный.** Фолбэк: 404 на международном эндпоинте → повтор на `https://open.bigmodel.cn/api/monitor/usage/quota/limit` (те же заголовки); любой другой статус — без фолбэка
- [x] 4.5 **Рефактор.** Вытащить human-table для типов лимитов в named-const блок; вынести декодер в отдельный pure-функцию, чтобы было легче тестировать без HTTP
- [x] 4.6 `make lint` зелёный

## 5. Адаптер OpenCode Go

- [x] 5.1 **Красный.** Юнит-тесты на резолв creds: обе env-переменные заданы → используются, env пуст + файл валиден → используются из файла, ничего не сконфигурировано → sentinel-ошибка `credentials missing — set OPENCODE_GO_USERNAME and OPENCODE_GO_PASSWORD`, задана только одна из двух → ошибка с именем пропущенной
- [x] 5.2 **Зелёный.** Резолв creds: `OPENCODE_GO_USERNAME` + `OPENCODE_GO_PASSWORD` (Trim) → JSON-файл `$XDG_CONFIG_HOME/ccr-models-usage/opencode-go.json` или `~/.config/ccr-models-usage/opencode-go.json` (поля `username` + `password`) → sentinel-ошибка
- [x] 5.3 **Красный.** Юнит-тесты на `authenticate(ctx, user, pass)`: успешный login через `httptest.Server`, неверные креды → ошибка `auth failed`, 5xx → ошибка содержит статус
- [x] 5.4 **Зелёный.** `authenticate(ctx, user, pass) (session, error)` — конкретный flow (POST на эндпоинт входа) выделен, чтобы тесты шли против мок-сервера
- [x] 5.5 **Красный.** Юнит-тесты на запрос дашборда: аутентифицированный GET 200 → переход к парсингу, 401/403/302 → ошибка `auth failed` / `cookie expired`, Cloudflare challenge (HTML без `__next_f.push`) → ошибка `dashboard markup may have changed`
- [x] 5.6 **Зелёный.** Запрос дашборда с desktop-Chrome `User-Agent`, сессией от `authenticate`, `Accept: text/html,application/xhtml+xml`, таймаут 15s
- [x] 5.7 **Красный.** Юнит-тесты на парсер: escape-форма `__next_f.push([1, "...{...}..."])` со всеми тремя окнами, форма с `$R[N]={...rollingUsage: $R[M]={...}...}`, частичные окна (только rolling), нет ни одного → пустой `Limits` + ошибка `dashboard markup may have changed`, Cloudflare-страница (нет ни одного блока) → то же
- [x] 5.8 **Зелёный.** Парсер: регулярка по обоим форматам, `usagePercent` → `UsedPct` как есть (это **использовано**), `resetInSec` → `ResetAt = time.Now().Add(time.Duration(s) * time.Second)`, label-ы `rolling (5h)` / `weekly` / `monthly`
- [x] 5.9 **Рефактор.** Вынести «сканер по обоим паттернам» и «декодер одного блока» в отдельные функции, чтобы тесты на 5.7 были декомпозированы
- [x] 5.10 `make lint` зелёный

## 6. Рендерер

- [x] 6.1 **Красный.** Golden-file тесты: z.ai с двумя типами лимитов, OpenCode Go со всеми тремя окнами, неизвестный хост (`skip (no adapter)`), ошибка адаптера (`skip (<msg>)`), порядок стабилен между запусками
- [x] 6.2 **Зелёный.** `render.Row` + `render.Write(w io.Writer, rows []Row)`: заголовок `<name-or-host> <host> [tariff]`, строки лимитов с двумя пробелами, label выровнен по 14 символов, `N%`, опц. `Detail` и `resets <время>`, пустая строка между блоками
- [x] 6.3 **Красный.** Юнит-тесты на форматтер относительного времени: `in 3h 42m`, `in 4d 12h`, `in 16d 5h`, абсолютная локальная дата после 30 дней, `now` для прошедшего
- [x] 6.4 **Зелёный.** Форматтер времени: relative в пределах 30d и в будущем, абсолютное (`YYYY-MM-DD HH:MM`) иначе, `now` для прошедшего
- [x] 6.5 **Рефактор.** Отделить «формат одной строки» от «Write», чтобы golden-тесты были компактными
- [x] 6.6 `make lint` зелёный

## 7. Сборка и CLI

- [x] 7.1 **Красный.** Интеграционный тест (build-тег `integration` или e2e): запустить собранный `cmd/ccr-models-usage` против временного конфига с двумя провайдерами — z.ai (успех через `httptest.Server`) и неизвестный (skip) — и сверить вывод с golden-файлом
- [x] 7.2 **Зелёный.** `main.go` в `cmd/ccr-models-usage/`: разобрать `-config`, прочитать и дедуплицировать, зарегистрировать адаптеры z.ai и OpenCode Go, разослать запросы через `provider.Fetcher`, отрендерить, выйти согласно спеке `cli-report`
- [x] 7.3 Прогнать на реальном `~/.claude-code-router/config.json`; вывод должен совпасть с примером из `proposal.md` (с точностью до зависящего от времени форматирования `resets`)
- [x] 7.4 README: блок про установку (`go install github.com/Djarvur/ccr-models-usage@latest`) и про то, как достать `OPENCODE_GO_USERNAME` / `OPENCODE_GO_PASSWORD`
- [x] 7.5 `make ci` (lint + test + build) зелёный
