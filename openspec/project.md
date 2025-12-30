# Project Context

## Purpose
MMemory is a Telegram Bot-based intelligent reminder tool that enables conversational interaction for managing daily habits and task reminders through AI-powered natural language processing. The system provides smart conversation capabilities with 30-day context history and a robust four-layer fallback strategy for reliable service delivery.

## Tech Stack
- **Language**: Go 1.21+
- **Database**: SQLite with GORM
- **Bot Framework**: Telegram Bot API
- **AI Integration**: OpenAI GPT models
- **Scheduler**: Cron-based job scheduling
- **Monitoring**: Prometheus metrics + Grafana dashboards
- **Configuration**: Viper with hot-reload support
- **Logging**: Logrus with structured JSON output
- **Container**: Docker & Docker Compose

## Project Conventions

### Code Style
- **Package Naming**: Use lowercase, no underscores (e.g., `aiparser`, `reminder`)
- **Function Naming**: CamelCase for exported functions, camelCase for internal
- **Variable Naming**: Meaningful names, avoid single letters except in loops
- **Error Handling**: Always return errors, wrap with context using `fmt.Errorf`
- **Comments**: Use Go conventions - start with function name for exported functions
- **Imports**: Group standard library, external, and internal imports separately

### Architecture Patterns
- **Layered Architecture**: Clear separation between bot, service, repository, and models
- **Dependency Injection**: Services injected through constructors
- **Interface Segregation**: Repository interfaces in `internal/repository/interfaces`
- **Error Wrapping**: Consistent error handling with context
- **Context Propagation**: Use `context.Context` for cancellation and timeouts
- **Configuration Management**: Centralized config with environment variable support

### Testing Strategy
- **Unit Tests**: Every service method must have corresponding unit tests
- **Integration Tests**: Database operations tested with in-memory SQLite
- **Mock Dependencies**: External services (Telegram API) mocked in tests
- **Test Fixtures**: Test data in `testdata/` directories
- **Coverage**: Aim for >80% coverage on business logic
- **Test Naming**: `Test<FunctionName>_<Scenario>` format

### Git Workflow
- **Branch Naming**: `feature/description`, `fix/issue-description`, `docs/update`
- **Commit Messages**: Conventional commits - `type: description`
- **Types**: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`
- **Main Branch**: `master` (protected, requires PR reviews)
- **PR Process**: Link to OpenSpec change proposal when applicable
- **Version Tagging**: Semantic versioning with `v` prefix (e.g., `v1.2.3`)

## Domain Context

### Core Concepts
- **Reminder**: A scheduled task with natural language time expressions
- **Conversation**: 30-day message history for AI context
- **AI Parser**: OpenAI-powered natural language understanding with fallback chain
- **Scheduler**: Cron-based execution with persistence and recovery
- **Fallback Strategy**: Four-layer degradation (Primary AI → Backup AI → Regex → Fallback chat)

### Schedule Patterns
- `daily`: Every day at specified time
- `weekly:1,3,5`: Monday, Wednesday, Friday
- `monthly:1,15`: 1st and 15th of each month
- `once:2024-10-01`: One-time reminder on specific date

### AI Integration
- **Primary Model**: Configurable OpenAI model (default: gpt-4o-mini)
- **Backup Model**: Same as primary for compatibility
- **Temperature**: 0.1 for consistent parsing
- **Max Tokens**: 1000 for parse responses
- **Timeout**: 30 seconds per request

### Natural Language Processing
The system handles complex Chinese time expressions:
- "工作日早上醒来后提醒我看书" (Remind me to read after waking up on weekdays)
- "每周三下午3点提醒我开会" (Remind me of meetings every Wednesday at 3 PM)
- "下个月第一天提醒我交房租" (Remind me to pay rent on the first day of next month)

## Important Constraints

### Technical Constraints
- **SQLite Limitations**: Single-file database, no concurrent writes from multiple processes
- **Telegram API Limits**: 30 messages per second global limit, 20 messages per minute per chat
- **OpenAI API Limits**: Rate limiting and token costs must be considered
- **Memory Constraints**: Designed for personal/small team use (< 10,000 users)

### Business Constraints
- **Privacy First**: All data stored locally, no external analytics
- **Chinese Language Focus**: Primary support for Chinese natural language processing
- **Telegram Platform**: Tied to Telegram's ecosystem and limitations
- **Open Source**: MIT license, community contributions welcome

### Performance Requirements
- **Response Time**: < 3 seconds for AI parsing with fallback
- **Reminder Accuracy**: ± 1 minute execution accuracy
- **Uptime Target**: 99.5% availability for reminder execution
- **Memory Usage**: < 100MB RSS for bot process

## External Dependencies

### Critical Services
- **Telegram Bot API**: Core messaging platform (https://api.telegram.org)
- **OpenAI API**: AI-powered natural language processing (https://api.openai.com)

### Optional Services
- **Prometheus**: Metrics collection and monitoring (http://localhost:9090)
- **Grafana**: Metrics visualization and dashboards (http://localhost:3000)
- **Docker Registry**: Container image distribution (configurable)

### Configuration Dependencies
- **Environment Variables**: `MMEMORY_BOT_TOKEN` (required), `MMEMORY_AI_OPENAI_API_KEY` (if AI enabled)
- **Config Files**: YAML configuration with hot-reload support
- **File System**: SQLite database file, log files, configuration files
