# opencode-go-usage

## Purpose

Узнать usage OpenCode Go через аутентификацию парой (username, password) и парсинг дашборда workspace, потому что провайдер не предоставляет публичного usage-API, а API-ключ предназначен только для инференса (см. [[opencode-go-no-api-for-usage]]).

## ADDED Requirements

### Requirement: Резолв учётных данных

Адаптер MUST резолвить пару `(username, password)` в следующем порядке:

1. Переменные окружения `OPENCODE_GO_USERNAME` и `OPENCODE_GO_PASSWORD`. Если обе заданы и непустые после `Trim`, используются.
2. JSON-файл по пути `$XDG_CONFIG_HOME/ccr-models-usage/opencode-go.json`, или `~/.config/ccr-models-usage/opencode-go.json` на macOS. Файл MUST быть JSON-объектом со строковыми полями `username` и `password`. Если оба поля есть и непустые после `Trim`, используются.
3. Если ни один источник не сконфигурирован, адаптер MUST вернуть sentinel-ошибку с сообщением `credentials missing — set OPENCODE_GO_USERNAME and OPENCODE_GO_PASSWORD` (рендерер показывает это дословно).

Если в любом источнике задано только одно из двух, адаптер MUST вернуть ошибку с именем пропущенной переменной. Сообщение НЕ должно содержать значение `password` — только имя.

#### Scenario: Обе переменные окружения заданы

- **WHEN** обе переменные `OPENCODE_GO_USERNAME` и `OPENCODE_GO_PASSWORD` заданы непустыми строками
- **THEN** адаптер использует их и не смотрит в файл конфига

#### Scenario: Задана только одна переменная

- **WHEN** задана только `OPENCODE_GO_USERNAME`
- **THEN** адаптер возвращает ошибку с именем `OPENCODE_GO_PASSWORD` как пропущенной переменной

#### Scenario: Env пуст, файл валиден

- **WHEN** переменные окружения не заданы, а в файле конфига есть и `username`, и `password`
- **THEN** адаптер использует значения из файла

#### Scenario: Ничего не сконфигурировано

- **WHEN** ни переменные окружения, ни файл конфига не дают учётных данных
- **THEN** адаптер возвращает sentinel-ошибку `credentials missing`

#### Scenario: Сообщение об ошибке не утекает пароль

- **WHEN** адаптер возвращает ошибку о пропущенном или неверном `password`
- **THEN** строка ошибки НЕ содержит значение `password` (ни из env, ни из файла) — только имя переменной

### Requirement: Аутентификация

Адаптер MUST обменять пару `(username, password)` на сессию, пригодную для подписи запроса к usage-эндпоинту OpenCode Go. Конкретный flow аутентификации (POST на эндпоинт входа, обмен на session cookie, и т.п.) — деталь реализации; контракт MUST быть выражен отдельной функцией `authenticate(ctx, username, password) (session, error)`, чтобы тесты шли против `httptest.Server` и не зависели от формы реального auth-эндпоинта.

#### Scenario: Успешный login

- **WHEN** `authenticate` против сервера, принимающего корректные креды, возвращает сессию
- **THEN** адаптер использует эту сессию для запроса дашборда

#### Scenario: Неверные креды

- **WHEN** сервер отвечает отказом аутентификации (например, HTTP 401)
- **THEN** адаптер возвращает ошибку, чьё сообщение содержит `auth failed`; рендерер показывает `skip (auth failed)`

### Requirement: Запрос дашборда

Адаптер MUST сделать `GET` на usage-эндпоинт OpenCode Go (URL определяется реализацией, но должен быть стабильным и документированным) с:

- Заголовком `Cookie` (или иным auth-header), соответствующим полученной сессии. Конкретное имя cookie/заголовка — деталь реализации.
- Заголовком `User-Agent`, соответствующим реальному desktop-Chrome на macOS (например, `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36`). Дефолтный Go `User-Agent` MUST NOT использоваться; Cloudflare вместо дашборда вернёт страницу challenge.
- `Accept: text/html,application/xhtml+xml` (или `application/json`, если реализация переключится на JSON-эндпоинт; в любом случае — заголовок должен явно отражать желаемый формат).

HTTP-таймаут MUST быть 15 секунд.

#### Scenario: Аутентифицированный ответ

- **WHEN** сервер возвращает HTTP 200 с телом, содержащим данные usage
- **THEN** адаптер переходит к парсингу

#### Scenario: Сессия истекла

- **WHEN** сервер возвращает HTTP 401, 403 или 302-редирект на страницу авторизации
- **THEN** адаптер возвращает ошибку, содержащую подстроку `auth failed` или `cookie expired`

#### Scenario: Cloudflare challenge

- **WHEN** сервер возвращает HTML, не содержащий блок `__next_f.push(...)` с данными usage
- **THEN** адаптер возвращает ошибку с сообщением `dashboard markup may have changed`

### Requirement: Парсинг HTML

Адаптер MUST извлечь три окна usage (`rollingUsage`, `weeklyUsage`, `monthlyUsage`) из тела ответа. Две известные формы:

- `self.__next_f.push([1, "...{...}..."])` — где JSON-объект лежит строковым литералом и содержит поля окон.
- `$R[N]($R[M], $R[P] = { mine: true, ..., rollingUsage: $R[Q] = { status: "ok", resetInSec: N, usagePercent: N }, ... });` — где окна вложены как object-references.

Для каждого найденного окна адаптер MUST извлечь `usagePercent` (число) и `resetInSec` (целое). `usagePercent` — это **использовано**, не остаток; он передаётся как есть.

Окно, у которого `usagePercent` не удаётся извлечь, молча отбрасывается. Адаптер возвращает срез `Limits`, содержащий только найденные окна; пустой результат допустим и не считается ошибкой.

#### Scenario: Все три окна на месте (escape-форма JSON)

- **WHEN** тело ответа содержит блок `__next_f.push([1, "{\"rollingUsage\":{\"usagePercent\":12.5,\"resetInSec\":3600},\"weeklyUsage\":{\"usagePercent\":25,\"resetInSec\":7200},\"monthlyUsage\":{\"usagePercent\":50,\"resetInSec\":10800}}"])`
- **THEN** адаптер возвращает три записи `Limit` с label-ами `rolling (5h)`, `weekly`, `monthly`, `UsedPct` 12.5/25/50 и `ResetAt`, равным now+3600s / now+7200s / now+10800s

#### Scenario: Присутствует только rolling-окно

- **WHEN** тело ответа содержит только `rollingUsage`, а другие два отсутствуют
- **THEN** адаптер возвращает один `Limit` (rolling); рендерер показывает только эту строку

#### Scenario: Все окна отсутствуют

- **WHEN** тело ответа не содержит парсируемых данных по окнам
- **THEN** адаптер возвращает пустой срез `Limits` И ненулевую ошибку с сообщением `dashboard markup may have changed`

### Requirement: Форматирование времени сброса

Адаптер MUST устанавливать каждому `Limit.ResetAt` значение `time.Time`, вычисленное как `time.Now().Add(time.Duration(resetInSec) * time.Second)`. За отображение времени в локальной таймзоне пользователя отвечает рендерер.

#### Scenario: resetInSec 3600

- **WHEN** дашборд сообщает `rollingUsage.resetInSec: 3600`
- **THEN** результирующий `Limit.ResetAt` равен `time.Now().Add(1 * time.Hour)` с точностью до одной секунды
