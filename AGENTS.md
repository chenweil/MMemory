<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:
- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

# Repository Guidelines

## Project Structure & Module Organization
The bot entry point is `cmd/bot/main.go`. Core domain logic lives under `internal/`: `internal/bot/handlers` reacts to Telegram updates, `internal/service` implements scheduling, context, and suggestion flows, and `internal/repository` wraps persistence. Shared utilities stay in `pkg/` (configuration, AI helpers, logging). Reference configurations live in `configs/`, local data such as SQLite files in `data/`, documentation in `docs/`, and integration harnesses in `test/`. Build scripts are in `scripts/`, with container assets in `docker/` and `docker-compose*.yml`.

## Build, Test, and Development Commands
Run `make run` to launch the bot with the current `cmd/bot` target. Use `make build` to emit a versioned binary into `bin/mmemory`. `make test` executes the full Go test suite with CGO enabled; `make test-cover` also generates `coverage.out` and `coverage.html`. Format and lint with `make fmt` (`go fmt ./...`) and `make lint` (`golangci-lint run`). For quick iteration use `go run cmd/bot/main.go` or target a package with `go test ./internal/service`.

## Coding Style & Naming Conventions
Follow standard Go formatting (tabs, gofmt). Run `make fmt` before pushing. Keep package directories lowercase and descriptive of their role (`reminder_service.go`, `notification_service.go`). Exported identifiers use PascalCase, unexported helpers use camelCase. Configuration keys and environment variables follow the `MMEMORY_` prefix and align with `configs/config.example.yaml`. Prefer the structured logger in `pkg/logger` for all logging.

## Testing Guidelines
Co-locate unit tests with implementation packages and suffix files with `_test.go`. Name test functions `TestFeature_Scenario` for clarity. Expect to exercise new logic with table-driven tests where practical. Run `make test` (or `go test ./...`) before submitting a change; refresh coverage artifacts when handler, repository, or service behavior shifts. Integration scenarios belong in `test/integration`.

## Commit & Pull Request Guidelines
Use Conventional Commits, e.g., `feat: add reminder context manager` or `fix: handle empty reminder logs`. Group related changes and keep subject lines ≤72 characters. Pull requests should describe purpose, key changes, and any migrations or config updates. Include test evidence (command output, coverage diff) and add screenshots or logs when bot interactions change. Reference roadmap IDs or issues whenever applicable.

## Security & Configuration Tips
Never commit secrets, tokens, or local SQLite databases. Store private keys in `.env` or `configs/config.yaml` (git-ignored). Use `configs/config.example.yaml` to illustrate required fields. When rotating credentials, update any deployment scripts (`scripts/deploy.sh`, `docker-compose.yml`) and call out the change in the PR description.
