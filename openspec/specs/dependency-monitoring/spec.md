# dependency-monitoring

## Purpose

Автоматически отслеживать обновления Go-зависимостей и используемых GitHub Actions (Dependabot, раз в неделю) и еженедельно проверять Go-зависимости на известные уязвимости (govulncheck) — без записи в репозиторий из самих security-проверок.

## Requirements

### Requirement: Dependabot сконфигурирован для Go-модулей и GitHub Actions

Репозиторий MUST содержать файл `.github/dependabot.yml` с включёнными экосистемами `gomod` и `github-actions` и расписанием `weekly`.

#### Scenario: dependabot.yml покрывает Go-модули
- **WHEN** репозиторий содержит `.github/dependabot.yml`
- **THEN** файл включает `version: 2` и `updates:` с элементом `- package-ecosystem: "gomod"` и `directory: "/"`, `schedule.interval: "weekly"`

#### Scenario: dependabot.yml покрывает GitHub Actions
- **WHEN** репозиторий содержит `.github/dependabot.yml`
- **THEN** в `updates:` присутствует `- package-ecosystem: "github-actions"` с `directory: "/"` и `schedule.interval: "weekly"`

#### Scenario: dependabot назначает labels и лимитирует PR-ы
- **WHEN** dependabot создаёт PR
- **THEN** PR получает label `dependencies`
- **AND** одновременно открытых PR от Dependabot — не больше 5

### Requirement: Dependabot группирует мелкие обновления

Dependabot MUST группировать minor и patch-обновления каждой экосистемы в один PR через секцию `groups:`.

#### Scenario: несколько minor-обновлений объединяются
- **WHEN** доступны minor- или patch-обновления нескольких Go-зависимостей
- **THEN** Dependabot открывает один PR с группой `minor-and-patch`, а не несколько отдельных PR

### Requirement: Security workflow запускается по расписанию

Репозиторий MUST содержать файл `.github/workflows/security.yml` с триггером `schedule` (cron) и workflow-командой для запуска `govulncheck` (или эквивалентного сканера уязвимостей Go).

#### Scenario: cron срабатывает
- **WHEN** наступает запланированное время cron (по умолчанию — понедельник, 06:00 UTC)
- **THEN** workflow `security.yml` стартует и выполняет govulncheck

#### Scenario: ручной запуск доступен
- **WHEN** пользователь запускает workflow через `workflow_dispatch` из UI GitHub
- **THEN** workflow `security.yml` стартует с теми же шагами, что и при cron

### Requirement: Security workflow запускает govulncheck

Job в `security.yml` MUST выполнить `govulncheck ./...` (через `go run golang.org/x/vuln/cmd/govulncheck@latest` либо предустановленный бинарь) и завершиться с ненулевым кодом, если уязвимости найдены.

#### Scenario: уязвимости не найдены
- **WHEN** govulncheck не сообщает о Known-уязвимостях в Go-зависимостях
- **THEN** job завершается с кодом 0

#### Scenario: уязвимости найдены
- **WHEN** govulncheck сообщает о Known-уязвимости
- **THEN** job завершается с ненулевым кодом
- **AND** в логах перечислены модуль, уязвимость и путь импорта

### Requirement: Security workflow не изменяет код

Job в `security.yml` MUST работать в режиме только-чтение: без checkout с `ref`, без push, без выдачи write-permissions.

#### Scenario: workflow не пишет в репозиторий
- **WHEN** `security.yml` запускается
- **THEN** workflow не выполняет `git push`, не создаёт коммиты и не открывает PR
- **AND** permissions ограничены `contents: read`

### Requirement: Конфигурация Dependabot и security-валидация изолированы

`.github/dependabot.yml` MUST создавать PR без автоматического merge, чтобы любое обновление зависимости или уязвимости проходило ревью и CI.

#### Scenario: PR от Dependabot проходит через CI
- **WHEN** Dependabot создаёт PR
- **THEN** PR запускает `ci.yml` (build/test/lint) и блокируется, если какой-либо job упал
