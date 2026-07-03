# add-ccr-models-usage

## Why

CCR настраивает несколько LLM-провайдеров, у каждого свои квоты и лимиты, но нет единого места, где видно «сколько у меня осталось прямо сейчас». Приходится открывать веб-дашборды и проверять каждого провайдера по отдельности. Это изменение добавляет маленькую Go-утилиту, которая читает `~/.claude-code-router/config.json`, дедуплицирует провайдеров по паре (hostname, api_key) и выводит процент использования для тех, кого умеет опрашивать. Первая версия поддерживает z.ai и OpenCode Go.

## What Changes

- **Новая CLI `ccr-models-usage`**: читает конфиг CCR из `~/.claude-code-router/config.json` (или `-config <путь>`), извлекает `Providers[]`, дедуплицирует по паре `(hostname, api_key)` и для каждой уникальной записи пытается получить процент использования.
- **Реестр провайдеров**: подключаемый адаптер на хост. Неизвестные хосты всё равно выводятся, со статусом `skip (no adapter)`.
- **Адаптер z.ai**: делает `GET https://api.z.ai/api/monitor/usage/quota/limit` (с фолбэком на `open.bigmodel.cn`) с `Authorization: Bearer <api_key>`. Выводит все типы лимитов, которые вернул API (TIME_LIMIT, TOKENS_LIMIT, RATE_LIMIT, TIMES_LIMIT, SESSION_LIMIT), плюс название тарифа.
- **Адаптер OpenCode Go**: требует **пару (username, password)**, а НЕ API-ключа — ключ предназначен только для инференса, и публичного API для usage у провайдера нет (см. [[opencode-go-no-api-for-usage]]). Читает их из переменных окружения `OPENCODE_GO_USERNAME` + `OPENCODE_GO_PASSWORD`, затем из файла `~/.config/ccr-models-usage/opencode-go.json`. Если ничего не настроено, печатает подсказку одной строкой. Конкретный flow аутентификации (POST на эндпоинт входа и т.п.) — деталь реализации; после аутентификации адаптер получает usage-данные и парсит из блоков `__next_f.push([1, "{...}"])` схему `{ rollingUsage, weeklyUsage, monthlyUsage }`, у каждого `{ usagePercent, resetInSec }`.
- **Вывод**: простой текст, одна строка на лимит. Без таблиц, без цветов. Пример:

  ```text
  zai api.z.ai Pro
    TIME_LIMIT      0%  remaining 100  resets 2026-04-16 10:31 SGT
    TOKENS_LIMIT   18%                 resets in 3h
  opencode-go opencode.ai Go
    rolling (5h)   40%  resets in 3h 42m
    weekly         31%  resets in 4d 12h
    monthly        21%  resets in 16d 5h
  yadro litellm-proxy.ai.yadro.com skip (no adapter)
  ```

- **Режим ошибок**: отсутствие учётных данных или сетевая/HTTP-ошибка для известного провайдера выводит `host skip (причина)` — никогда не прерывает весь прогон. Неизвестные хосты тоже выводятся, со статусом `skip (no adapter)`.

## Capabilities

### New Capabilities

- `ccr-config-reading`: разобрать конфиг CCR, извлечь провайдеров, дедуплицировать по (hostname, api_key).
- `provider-registry`: подключаемая система адаптеров, индексированная по хосту; неизвестные хосты всё равно попадают в вывод.
- `zai-usage`: опросить monitoring-API z.ai и преобразовать ответ в список строк `(label, percent_used, reset_human)`.
- `opencode-go-usage`: резолвить пару (username, password), аутентифицироваться, получить usage-данные, извлечь `rollingUsage`/`weeklyUsage`/`monthlyUsage`, преобразовать в строки.
- `cli-report`: текстовый рендерер с режимом «skip с причиной» при ошибках.
- `tdd-process`: дисциплина разработки через тесты — каждое нетривиальное изменение начинается с падающего теста, имплементация появляется ровно настолько, чтобы тест прошёл, рефакторинг идёт после зелёного.

### Modified Capabilities

- Нет. (Существующих specs нет.)

## Impact

- **Новый Go-модуль** `github.com/Djarvur/ccr-models-usage` (case sensitive, как задан пользователем), раскладка стандартная: точка входа `cmd/ccr-models-usage/main.go`, всё нетривиальное — в `internal/`. Ожидаемые пакеты: `internal/config`, `internal/provider`, `internal/providers/zai`, `internal/providers/opencodego`, `internal/render`.
- **Никаких рантайм-зависимостей** помимо стандартной библиотеки Go (`net/http`, `encoding/json`, `regexp`, `time`, `flag`).
- **Dev-зависимости только для разработки**: `github.com/golangci/golangci-lint` (через официальный установщик) + testify в тестах через `go.mod require` (транзитивно).
- **Не требует доступа в сеть на этапе установки**; программа ставится обычным `go install` из репозитория.
- **Файловая система**: только читает `~/.claude-code-router/config.json` и опционально `~/.config/ccr-models-usage/opencode-go.json`. Ничего не пишет.
- **Cookie браузера**: опциональное чтение sqlite-хранилища cookie и истории Chrome/Brave/Arc/Edge только на macOS. Срабатывает только при отсутствии env-переменных и файла конфига. На macOS используется Keychain — должен быть доступен пользователю.
- **Сеть**: 1 HTTP-запрос на каждого известного провайдера за прогон. Таймаут 10 секунд. Без ретраев.
- **Безопасность**: API-ключи и cookie читаются с диска пользователя; никогда не логируются, не записываются, не отправляются никуда кроме собственного эндпоинта провайдера.
