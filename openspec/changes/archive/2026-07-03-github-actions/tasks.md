# Tasks: github-actions

## 1. CI workflow

- [x] 1.1 Создать `.github/workflows/ci.yml` с триггерами `push` и `pull_request` на ветки `main` и `master`
- [x] 1.2 Прописать `permissions: contents: read` и `concurrency` с `cancel-in-progress: true` (с `if: github.event_name == 'pull_request'` или аналогом, чтобы не отменять main)
- [x] 1.3 Добавить job `build` на `ubuntu-latest`: `actions/checkout@v4`, `actions/setup-go@v5` с `go-version-file: go.mod` и `cache: true`, шаг `make build`
- [x] 1.4 Добавить job `test` на `ubuntu-latest`: `make test` (или `go test -coverprofile=coverage.out -covermode=atomic ./...`), печать покрытия через `go tool cover -func=coverage.out | tail -1`, загрузка `coverage.out` через `actions/upload-artifact@v4` с именем `coverage-report`
- [x] 1.5 Добавить job `lint` на `ubuntu-latest`: `golangci/golangci-lint-action@v7` (версия из `.golangci.yml` либо последняя стабильная v2.x)
- [x] 1.6 Выставить `timeout-minutes: 10` на каждый job
- [ ] 1.7 Проверить, что workflow валиден: открыть PR (или прогнать через `act` локально) — все job-ы зелёные на текущем `main`  *(ручной шаг: требует открытия PR на GitHub)*

## 2. Dependabot

- [x] 2.1 Создать `.github/dependabot.yml` с `version: 2` и блоком `updates:`
- [x] 2.2 Включить ecosystem `gomod` с `directory: "/"`, `schedule.interval: "weekly"`, `day: "monday"`, `time: "06:00"`, `open-pull-requests-limit: 5`
- [x] 2.3 Включить ecosystem `github-actions` с теми же `schedule` и `open-pull-requests-limit: 5`
- [x] 2.4 Для каждой экосистемы задать `groups.minor-and-patch` с `applies-to: version-updates` и pattern-фильтром `["minor", "patch"]`
- [x] 2.5 Назначить `labels: ["dependencies"]` для PR обеих экосистем
- [ ] 2.6 Убедиться, что в GitHub → Insights → Dependency graph видны Go-модули (иначе Dependabot для gmod не сработает)  *(ручной шаг: проверка в GitHub UI)*

## 3. Security workflow

- [x] 3.1 Создать `.github/workflows/security.yml` с триггерами `schedule: cron: "0 6 * * 1"` (понедельник 06:00 UTC) и `workflow_dispatch`
- [x] 3.2 Завести единственный job на `ubuntu-latest` с `permissions: contents: read`
- [x] 3.3 В job: `actions/checkout@v4` + `actions/setup-go@v5` (`go-version-file: go.mod`, `cache: true`)
- [x] 3.4 Шаг: `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
- [x] 3.5 Выставить `timeout-minutes: 15` на job
- [ ] 3.6 Прогнать вручную через `workflow_dispatch` и убедиться, что workflow завершается успешно на текущем `main`  *(ручной шаг: workflow_dispatch в GitHub UI)*

## 4. Проверка и ревью

- [x] 4.1 Сделать commit с тремя новыми файлами (`.github/workflows/ci.yml`, `.github/workflows/security.yml`, `.github/dependabot.yml`) одним изменением
- [ ] 4.2 Открыть PR — убедиться, что `ci.yml` отработал на PR: build, test (с coverage-артефактом), lint — все зелёные  *(ручной шаг: открыть PR на GitHub)*
- [ ] 4.3 Скачать coverage-артефакт из PR-run и проверить, что в нём есть `coverage.out` с валидным `mode: atomic`  *(ручной шаг: проверка артефакта в GitHub UI)*
- [ ] 4.4 После merge в `main` убедиться, что:
  - CI продолжает зеленеть на push в main
  - Dependabot зарегистрировал репозиторий (письмо или первый PR по расписанию)
  - Security workflow отработает по cron или через `workflow_dispatch`
  *(ручной шаг: проверка в GitHub UI после merge)*
- [x] 4.5 Документировать в `README.md` (опционально): badge статуса CI и ссылку на `govulncheck` workflow

## 5. Опциональные улучшения (не блокируют merge)

- [ ] 5.1 Включить в GitHub branch protection на `main`/`master` required check `CI` (настраивается вне репозитория)
- [ ] 5.2 Добавить SARIF-загрузку для `govulncheck` (permissions: `security-events: write`, шаг `github/codeql-action/upload-sarif@v3`) — только при реальной необходимости
- [ ] 5.3 Добавить отдельный job `tidy` с `go mod tidy -diff`, если появится дрейф зависимостей
- [ ] 5.4 Зафиксировать coverage-порог в `ci-pipeline` spec, когда базовый уровень будет измерен на первом прогоне
