# Repository Guidelines

## Project Structure & Module Organization
The runtime entry point lives in `cmd/bot/main.go`. Domain code resides under `internal/`, with `internal/bot/handlers` coordinating Telegram updates, `internal/service` holding scheduling logic, and `internal/repository` mediating persistence. Shared utilities sit in `pkg/config` and `pkg/logger`. Configuration samples live in `configs/`, persisted SQLite artifacts in `data/`, docs in `docs/`, and integration harnesses in `test/`. The `scripts/` directory provides release automation, while Docker assets live in `docker/` alongside `docker-compose*.yml`.

## Build, Test, and Development Commands
Use `make run` for a local bot instance (wired to `cmd/bot`). `make build` assembles a versioned binary at `bin/mmemory`. `make test` runs the full suite with CGO enabled; `make test-cover` augments that with `coverage.out` and `coverage.html`. `make fmt` and `make lint` wrap `go fmt ./...` and `golangci-lint run`. For quick iteration you can call `go run cmd/bot/main.go` or execute targeted testing via `go test ./internal/service`.

## Coding Style & Naming Conventions
Format Go sources with `make fmt` before review; Go's default tab indentation is expected. Keep package names lowercase and align filenames with the package focus (`reminder_service.go`, `logger.go`). Exported identifiers use PascalCase, while unexported helpers stay camelCase. Environment configuration follows the `MMEMORY_` prefix and YAML keys mirror the structure in `configs/config.example.yaml`. Prefer structured logging via `pkg/logger` and avoid mixing custom loggers.

## Testing Guidelines
Unit tests live beside implementation packages; integration workflows sit in `test/integration`. Name test files using `_test.go` and functions as `TestXxx_Scenario`. Run `make test` before opening a PR, and regenerate coverage when behavior changes touch handlers or repositories. Attach or refresh artifacts such as `coverage.html` only when they meaningfully change.

## Commit & Pull Request Guidelines
Follow Conventional Commits (`feat: add reminder snooze`, `fix: resolve cron regression`). Write present-tense summaries under 72 characters and group related changes per commit. PRs should include: 1) purpose and scope, 2) testing evidence (command output or coverage summary), 3) configuration or migration notes, and 4) screenshots or log excerpts if bot interactions changed. Reference issue IDs or roadmap stages when applicable.

## Security & Configuration Tips
Never commit real Telegram tokens or database files—use `.env` and `configs/config.example.yaml` as templates. Keep secrets out of version control by storing them locally in `configs/config.yaml`. When rotating credentials, update both the YAML and any deployment manifests (`deploy.sh`, `docker-compose.yml`), and document the change in the PR description.
