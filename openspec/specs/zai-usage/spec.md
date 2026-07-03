# zai-usage

## Purpose

Опросить monitoring-API z.ai ради квоты и данных использования, используя только API-ключ, и преобразовать ответ в список лимитов.

## Requirements

### Requirement: Эндпоинт и авторизация

Адаптер MUST делать `GET` на `https://api.z.ai/api/monitor/usage/quota/limit` с заголовком `Authorization: Bearer <api_key>` и `Accept: application/json`. HTTP-таймаут MUST быть 10 секунд.

#### Scenario: Успешный запрос

- **WHEN** API отвечает HTTP 200 и тело JSON, соответствующее документированной схеме
- **THEN** адаптер возвращает непустой срез `Limits` и `nil`-ошибку

#### Scenario: Невалидный ключ

- **WHEN** API отвечает HTTP 401
- **THEN** адаптер возвращает ошибку, чьё сообщение содержит `auth failed`; рендерер показывает это как `skip (auth failed)`

### Requirement: Фолбэк-эндпоинт

Если международный эндпоинт возвращает HTTP 404, адаптер MUST повторить запрос на `https://open.bigmodel.cn/api/monitor/usage/quota/limit` (те же заголовки, тот же метод). Любой другой статус MUST возвращаться как есть, без фолбэка.

#### Scenario: Международный эндпоинт недоступен

- **WHEN** `https://api.z.ai/api/monitor/usage/quota/limit` возвращает 404
- **THEN** адаптер повторяет запрос на CN-эндпоинт в рамках того же бюджета таймаута; если тот успешен, возвращаются лимиты

#### Scenario: Оба эндпоинта не сработали

- **WHEN** оба эндпоинта возвращают не-2xx
- **THEN** адаптер возвращает ошибку, содержащую последний увиденный HTTP-статус

### Requirement: Парсинг ответа

Адаптер MUST парсить ответ в срез `Limits` следующим образом:

- Для каждой записи в `data.limits[]` эмитится один `Limit`:
  - `Label` = поле `type`, проведённое через человеческую таблицу:
    `TIME_LIMIT → "TIME_LIMIT"`, `TOKENS_LIMIT → "TOKENS_LIMIT"`, `RATE_LIMIT → "RATE_LIMIT"`, `TIMES_LIMIT → "TIMES_LIMIT"`, `SESSION_LIMIT → "SESSION_LIMIT"`. Неизвестные типы передаются как есть.
  - `UsedPct` = поле `percentage`.
  - `ResetAt` = `time.UnixMilli(nextResetTime)`, если `nextResetTime` есть и парсится, иначе `nil`.
  - `Detail` = `remaining <N>`, если поле `remaining` есть и > 0.
- Имя тарифа из `data.level` ДОЛЖНО быть доступно рендереру как заголовок (например, `Pro`).

#### Scenario: Несколько типов лимитов

- **WHEN** ответ содержит и `TIME_LIMIT`, и `TOKENS_LIMIT`
- **THEN** адаптер возвращает две записи `Limit` с соответствующими label-ами и процентами

#### Scenario: Отсутствует nextResetTime

- **WHEN** у лимита нет поля `nextResetTime`
- **THEN** соответствующий `Limit.ResetAt` равен `nil`; рендерер опускает время сброса

#### Scenario: Неожиданная форма JSON

- **WHEN** ответ не парсится в ожидаемую схему
- **THEN** адаптер возвращает ошибку, содержащую подстроку `decode`; рендерер показывает `skip (decode error: <причина>)`
