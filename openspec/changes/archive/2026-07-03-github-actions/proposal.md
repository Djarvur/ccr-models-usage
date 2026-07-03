## Why

В репозитории нет CI: PR-ы не проверяются автоматически, покрытие и линтинг контролируются вручную, а уязвимости и обновления зависимостей отслеживаются по инерции. Нужны GitHub Actions, которые на каждом push/PR гоняют сборку, тесты с покрытием и golangci-lint, плюс периодические задачи для Dependabot и security-проверки — чтобы регрессии и drift по зависимостям ловились автоматически.

## What Changes

- Добавить `.github/workflows/ci.yml` — workflow на `push` и `pull_request` в ветки `main` и `master`: `make build`, `make test` с отчётом покрытия, `make lint` (golangci-lint).
- Добавить `.github/dependabot.yml` — конфигурация Dependabot для Go-модулей и GitHub Actions с еженедельным расписанием.
- Добавить `.github/workflows/security.yml` — расписание (cron, еженедельно) с запуском `govulncheck` (и/или `osv-scanner`).
- Добавить инструкции/таймауты и кэш Go-модулей для ускорения пайплайна.
- Исходный код и публичные API не меняются.

## Capabilities

### New Capabilities

- `ci-pipeline`: непрерывная интеграция — сборка, тесты с покрытием, линтинг на push/PR.
- `dependency-monitoring`: автоматическое обновление зависимостей (Dependabot) и периодическая security-проверка уязвимостей.

### Modified Capabilities

Нет. Существующие capability (`ccr-config-reading`, `cli-report`, `opencode-go-usage`, `provider-registry`, `tdd-process`, `zai-usage`) описывают поведение продукта и не затрагиваются.

## Impact

- Новые файлы под `.github/`:
  - `.github/workflows/ci.yml`
  - `.github/workflows/security.yml`
  - `.github/dependabot.yml`
- Возможные сопутствующие правки: упоминание CI-бейджа в `README.md` (опционально).
- К Go-коду, модулям, бинарнику и существующим скриптам — без изменений.
- Внешние зависимости CI: `actions/checkout`, `actions/setup-go`, `golangci/golangci-lint-action`, `osv-scanner` (либо `govulncheck`).
