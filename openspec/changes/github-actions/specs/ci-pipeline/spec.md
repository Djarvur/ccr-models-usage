# ci-pipeline

## ADDED Requirements

### Requirement: CI workflow triggers on push and pull request

Репозиторий MUST содержать файл `.github/workflows/ci.yml`, который запускает CI на push в основные ветки и на каждый pull request.

#### Scenario: push в основную ветку запускает CI
- **WHEN** происходит `push` в ветку `main` или `master`
- **THEN** workflow `ci.yml` стартует автоматически

#### Scenario: pull request запускает CI
- **WHEN** открывается, синхронизируется или переоткрывается pull request в `main` или `master`
- **THEN** workflow `ci.yml` стартует автоматически

#### Scenario: повторный запуск для того же ref отменяется
- **WHEN** в рамках одной ветки/PR появляется новый коммит
- **THEN** предыдущий in-progress запуск CI отменяется через `concurrency.cancel-in-progress: true`, за исключением запусков по push в `main`/`master`, которые не отменяются

### Requirement: CI запускает job-ы build, test, lint параллельно

Workflow MUST содержать как минимум три job-а: `build`, `test` и `lint`, выполняющиеся параллельно.

#### Scenario: три job-а выполняются параллельно
- **WHEN** workflow `ci.yml` стартует
- **THEN** job-ы `build`, `test` и `lint` запускаются параллельно в рамках одной матрицы или без зависимостей

#### Scenario: падение любого job-а валит CI
- **WHEN** любой из job-ов (`build`, `test`, `lint`) завершается с ненулевым кодом
- **THEN** статус workflow — `failure`, и блокирующие branch protection check-и (если настроены) падают

### Requirement: Job `build` собирает бинарник

Job `build` MUST собирать CLI-бинарник `ccr-models-usage` командой `make build` (или эквивалентом: `go build -o bin/ccr-models-usage ./cmd/ccr-models-usage`).

#### Scenario: сборка успешна
- **WHEN** job `build` запускается на чистом репозитории
- **THEN** артефакт `bin/ccr-models-usage` создаётся и job завершается с кодом 0

#### Scenario: ошибка компиляции валит job
- **WHEN** исходный код не компилируется
- **THEN** job `build` завершается с ненулевым кодом

### Requirement: Job `test` запускает тесты с покрытием

Job `test` MUST выполнить `go test ./...` с генерацией coverage-профиля и загрузить отчёт как артефакт workflow.

#### Scenario: тесты проходят
- **WHEN** все unit-тесты проходят
- **THEN** job `test` завершается с кодом 0

#### Scenario: упавший тест валит job
- **WHEN** хотя бы один тест падает
- **THEN** job `test` завершается с ненулевым кодом

#### Scenario: coverage-профиль загружается как артефакт
- **WHEN** job `test` выполняет `go test -coverprofile=coverage.out -covermode=atomic ./...`
- **THEN** файл `coverage.out` загружается как artifact под именем `coverage-report`
- **AND** job печатает общий процент покрытия через `go tool cover -func=coverage.out | tail -1`

### Requirement: Job `lint` запускает golangci-lint

Job `lint` MUST использовать `golangci/golangci-lint-action` с конфигурацией из `.golangci.yml` репозитория.

#### Scenario: линтер успешен
- **WHEN** код проходит все правила `.golangci.yml`
- **THEN** job `lint` завершается с кодом 0

#### Scenario: нарушение правила валит job
- **WHEN** код нарушает хотя бы одно правило `.golangci.yml`
- **THEN** job `lint` завершается с ненулевым кодом и в логах указано нарушающее правило

### Requirement: CI использует Go-версию из go.mod

Workflow MUST определять версию Go через `actions/setup-go` с параметром `go-version-file: go.mod` — без хардкода.

#### Scenario: версия Go соответствует go.mod
- **WHEN** в `go.mod` указано `go 1.26.4`
- **THEN** runner использует Go 1.26.4 для всех job-ов

### Requirement: CI кэширует Go-модули и build-кэш

Job-ы `build`, `test`, `lint` MUST использовать встроенный кэш `actions/setup-go` (параметр `cache: true`) для модулей и `golangci-lint-action` — для своего кэша.

#### Scenario: повторный запуск использует кэш
- **WHEN** job повторно запускается в рамках workflow-run с теми же зависимостями
- **THEN** шаги кэширования сообщают `Cache hit` в логах

### Requirement: CI работает с минимальными permissions

Workflow MUST объявлять `permissions:` на уровне workflow со значением `contents: read`; повышения разрешений — только в job-ах, где это явно требуется (например, для загрузки SARIF — `security-events: write`).

#### Scenario: стандартные permissions ограничены
- **WHEN** workflow стартует
- **THEN** GITHUB_TOKEN имеет `contents: read` по умолчанию

### Requirement: Таймауты job-ов ограничены

Каждый job MUST иметь явный `timeout-minutes` (по умолчанию 10 минут) для защиты от зависших рантаймов.

#### Scenario: длительный job прерывается по таймауту
- **WHEN** job выполняется дольше `timeout-minutes`
- **THEN** GitHub Actions принудительно завершает job с ошибкой таймаута
