## Context

Сейчас в репозитории `Djarvur/ccr-models-usage` нет CI: PR-ы мержатся без автоматической проверки, покрытие и линтинг контролируются вручную, а обновления зависимостей и уязвимости не отслеживаются регулярно. Репозиторий — Go 1.26 CLI (`cmd/ccr-models-usage`); в [Makefile](Makefile) уже определены таргеты `build`, `test`, `lint`, `ci`, в [.golangci.yml](.golangci.yml) — конфигурация линтера.

Целевая картина: на каждый push/PR в основные ветки запускается `ci.yml` (build + test с покрытием + lint), Dependabot раз в неделю присылает PR на обновление Go-модулей и экшенов, и отдельный `security.yml` по расписанию прогоняет `govulncheck`.

## Goals / Non-Goals

**Goals:**

- Автоматически проверять PR: сборка, юнит-тесты, покрытие, golangci-lint.
- Не ломать зелёный main: падающий CI блокирует merge (через branch protection, настраивается отдельно).
- Экономить минуты CI: кэш Go-модулей, отмена повторных запусков для того же ref, разумные таймауты.
- Еженедельно обновлять зависимости (Go modules, github-actions) через Dependabot малыми группами.
- Еженедельно проверять уязвимости в Go-зависимостях (`govulncheck`).
- Минимальные `permissions:` и фиксированные версии экшенов по SHA (где критично).

**Non-Goals:**

- Релиз-пайплайн, публикация бинарников, подпись релизов.
- Матрица по нескольким Go-версиям / нескольким ОС (проект таргетит актуальный Go 1.26).
- Инфраструктура как код, self-hosted runners.
- Включение уведомлений в Slack/Jira/Mattermost.
- Настройка branch protection rules (это делается в GitHub UI вне репозитория).

## Decisions

### D1. Один `ci.yml` workflow с job-ами: `build`, `test`, `lint`

Один workflow проще поддерживать и расшарить триггеры/permissions, чем три отдельных файла. Job-ы параллельны, общий `setup-go` шаг можно вынести в reusable-этап, но ради простоты оставляем каждый job самодостаточным.

Альтернативы:

- Три файла (`build.yml`, `test.yml`, `lint.yml`) — больше шума, дублирование `setup-go` и `actions/checkout`.
- Один `make ci` job — медленнее (нет параллелизма), не локализует причину падения.

### D2. Runner `ubuntu-latest`, Go из `go.mod` через `actions/setup-go@v5`

Go кросс-платформенный, проект не использует CGO и специфичные Linux-фичи — `ubuntu-latest` самый дешёвый и быстрый. `actions/setup-go@v5` читает версию из `go.mod` (`go-version-file: go.mod`), что устраняет расхождения с локальной разработкой.

Альтернативы:

- macOS / Windows — дороже и медленнее без выигрыша.
- Зафиксировать Go pin явно — плодит дрейф версий между CI и `go.mod`.

### D3. Кэш Go-модулей + кэш `~/.cache/go-build`

`actions/setup-go@v5` с `cache: true` кэширует Go modules по `go.sum`; `golangci-lint-action` кэширует свой кэш по версии линтера. Этого достаточно; отдельный `actions/cache` для `GOMODCACHE` не нужен.

### D4. Покрытие: `go test -coverprofile=coverage.out -covermode=atomic ./...` + сравнение с порогом

Используем стандартные инструменты Go. Шаг `coverage` запускается после `test`, читает `coverage.out`, парсит процент через `go tool cover -func`, и сравнивает с порогом (например, 70% — фиксируем в specs). Артефакт `coverage.out` загружается для отладки.

Альтернативы:

- Codecov/Coveralls — внешняя зависимость, аккаунт, токен; для проекта такого размера избыточно.
- `go test -cover` без `-coverprofile` — не даёт машиночитаемый порог.

### D5. Lint: `golangci/golangci-lint-action@v7` с версией из `.golangci.yml`

Если в `.golangci.yml` указана конкретная версия линтера — используем её; иначе берём последнюю стабильную v2.x. Action шарит кэш по версии, что ускоряет повторные запуски.

Альтернативы:

- `go install golangci-lint` внутри job — без кэша, медленнее, дублирует работу.

### D6. Concurrency: `concurrency: { group: ${{ github.workflow }}-${{ github.ref }}, cancel-in-progress: true }` для push/PR

Для PR это отменит устаревшие запуски при force-push. Для push в ветку, отличную от main, тоже полезно. Для main — `cancel-in-progress` отключаем через условный `if`, чтобы не отменить деплой/merge.

### D7. Dependabot: `gomod` + `github-actions`, еженедельно, с группировкой

`/.github/dependabot.yml` с интервалом `weekly`, `day: monday`, `time: "06:00"`, группой `minor-and-patch` (объединяет патчи в один PR, чтобы не плодить шум), `open-pull-requests-limit: 5`. Labels: `dependencies`, `ci`.

Альтернативы:

- Ежедневно — слишком шумно для проекта без CI.
- Renovate — мощнее, но добавляет ещё один конфиг и решение; не нужно.

### D8. Security: отдельный `security.yml` на `schedule` + `workflow_dispatch`

Cron `0 6 * * 1` (понедельник 06:00 UTC) — не совпадает с Dependabot, чтобы не нагружать GitHub в один момент. `govulncheck` через `golang.org/x/vuln/cmd/govulncheck@latest` (после появления официального action — перейти на него). `permissions: contents: read`, `security-events: write` (для SARIF-загрузки, если добавим позже).

Альтернативы:

- `osv-scanner` — шире по экосистемам, но проект pure-Go, `govulncheck` точнее для Go.
- CodeQL — избыточно, статический анализ уязвимостей в нашем коде не нужен (минимальный CLI).

### D9. Минимальные `permissions:` на уровне workflow

`contents: read` для всех job-ов. Конкретные повышения — только где нужны (`security-events: write` для SARIF, если включим). По умолчанию GitHub теперь поддерживает `permissions: read-all` — указываем явно для прозрачности.

## Risks / Trade-offs

- **Ложноотрицательные срабатывания Dependabot на major-апдейты Go** → mitigation: Dependabot создаёт PR, но мержат вручную после просмотра CI; в `dependabot.yml` включаем `ignore` для major-версий golangci-lint до ручной проверки.
- **`govulncheck` без фикса версии может дрейфовать** → mitigation: пин через Go toolchain, обновление — отдельный PR.
- **Секреты не нужны, но легко случайно добавить** → mitigation: workflow-файлы не объявляют `secrets:`; PR-ревью должно ловить добавление.
- **Покрытие как gating-фактор может фризить развитие** → mitigation: порог ставим консервативный (стартово 0% и поднимаем постепенно), не валим PR при падении покрытия, пока порог не зафиксирован в specs; спека фиксирует только то, что отчёт покрытия загружается.
- **Расписание security и Dependabot может совпасть с пиковой нагрузкой** → mitigation: разнесённые cron-времена (`0 6 * * 1` для security, `06:00` понедельник для Dependabot — допустимо, оба лёгкие).
- **Закрытый `govulncheck@latest`** при недоступности сети CI — низкий риск: GitHub-hosted runner имеет выход в сеть; при регулярных падениях — пин версии.

## Migration Plan

- Шаг 1: добавить `.github/workflows/ci.yml`, прогнать на текущем main — должен пройти зелёным.
- Шаг 2: добавить `.github/dependabot.yml` — Dependabot сам подтянет расписание, первый PR придёт по расписанию.
- Шаг 3: добавить `.github/workflows/security.yml` — первый прогон в ближайший понедельник 06:00 UTC либо вручную через `workflow_dispatch`.
- Шаг 4: рекомендовать включить branch protection на `main`/`master` с required check `CI / test` (настраивается вне репозитория).

Откат: удалить соответствующие файлы под `.github/` — workflow-ы перестают существовать немедленно, состояния PR/branch protection не затрагиваются. Скрытых миграций данных нет.

## Open Questions

- Какой начальный порог покрытия выставлять (или стартовать без gating и поднимать позже)? — фиксируется в `ci-pipeline` spec.
- Нужен ли SARIF-upload для `govulncheck` (включает `security-events: write` и UI в Security tab)? — на старте не включаем; добавим, если попросят.
- Делать ли отдельный job для `go mod tidy`? — на старте нет (CI поймает drift через тесты и lint); добавим, если появится реальный drift.
