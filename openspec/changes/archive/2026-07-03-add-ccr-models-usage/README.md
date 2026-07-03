# add-ccr-models-usage

Читает конфиг CCR, дедуплицирует провайдеров по паре (hostname, api_key) и выводит процент использования для тех провайдеров, для которых у программы есть адаптер (z.ai, OpenCode Go).

## Установка

```text
go install github.com/Djarvur/ccr-models-usage/cmd/ccr-models-usage@latest
```

Имя модуля — `github.com/Djarvur/ccr-models-usage` (case sensitive: `Djarvur` с большой буквы).

## Конфигурация OpenCode Go

API-ключ OpenCode Go авторизует только инференс; usage-эндпоинта у провайдера нет. Адаптер ожидает пару `username` + `password` — из env-переменных или из `~/.config/ccr-models-usage/opencode-go.json`:

```text
export OPENCODE_GO_USERNAME="..."
export OPENCODE_GO_PASSWORD="..."
```

```json
{
  "username": "...",
  "password": "..."
}
```

Файл должен иметь permissions `0600`.

## Пример вывода

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

## Разработка

```text
make ci    # = lint (golangci-lint с enable:all) + test + build
```

TDD: каждое изменение начинается с падающего теста, см. `specs/tdd-process/spec.md`.
