# ccr-models-usage

[![CI](https://github.com/Djarvur/ccr-models-usage/actions/workflows/ci.yml/badge.svg)](https://github.com/Djarvur/ccr-models-usage/actions/workflows/ci.yml)
[![Security](https://github.com/Djarvur/ccr-models-usage/actions/workflows/security.yml/badge.svg)](https://github.com/Djarvur/ccr-models-usage/actions/workflows/security.yml)

Check and report limits and usage for the providers defined in
[Claude Code Router](https://github.com/musistudio/claude-code-router) config.

## Install

```sh
go install github.com/Djarvur/ccr-models-usage@latest
```

The binary is installed as `ccr-models-usage` into `$GOBIN` (or `$HOME/go/bin`
by default).

## Usage

```sh
ccr-models-usage
# or with a custom config path:
ccr-models-usage -config /path/to/config.json
```

By default the program reads `~/.claude-code-router/config.json`, deduplicates
the configured providers by `(hostname, api_key)`, and prints a short
per-provider usage report. Providers the program does not know how to query
are still listed with `skip (no adapter)`.

## OpenCode Go credentials

OpenCode Go does not expose a usage API and its API key only authorizes
inference, so the program authenticates to the dashboard with a
`(username, password)` pair. Configure it via either of:

- Environment variables:

  ```sh
  export OPENCODE_GO_USERNAME=...
  export OPENCODE_GO_PASSWORD=...
  ```

- A JSON file at
  `$XDG_CONFIG_HOME/ccr-models-usage/opencode-go.json` (or
  `~/.config/ccr-models-usage/opencode-go.json`):

  ```json
  { "username": "...", "password": "..." }
  ```

The file should be `chmod 600`. The program never writes to it and never
logs the values.

## Development

```sh
make ci
```

runs lint, test, and build.

### Prerequisite: `golangci-lint`

`make lint` requires
[golangci-lint](https://github.com/golangci/golangci-lint). Install it via
the [official installer](https://golangci-lint.run/welcome/install/):

```sh
# Linux/macOS, one-liner
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

Other `make` targets: `make build`, `make test`, `make lint`.
