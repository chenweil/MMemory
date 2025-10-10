# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## about me
My name is chenwl. I am a software engineer based in Beijing  China. My English is not good, please communicate with me in Chinese.

## Project Overview

MMemory is a Telegram Bot-based intelligent reminder tool built with Go. The system enables conversational interaction for managing daily habits and task reminders through AI-powered natural language processing.

### Key Features
- **AI-Powered Parsing**: OpenAI integration for intelligent message understanding (C1 phase completed)
- **Smart Conversation**: 30-day conversation history for context-aware interactions
- **Fallback Strategy**: Four-layer degradation (Primary AI → Backup AI → Regex → Fallback chat)
- **Scheduler System**: Cron-based reminder execution with persistence
- **Monitoring**: Comprehensive Prometheus metrics and Grafana dashboards

## Development Commands

### Initial Setup
```bash
# Initialize Go module and dependencies
go mod tidy

# Copy and configure settings
cp configs/config.example.yaml configs/config.yaml
# Set TELEGRAM_BOT_TOKEN in config.yaml
```

### Quick Start with Makefile (推荐)
```bash
# 查看所有可用命令
make help

# 构建项目（输出到 bin/mmemory）
make build

# 运行应用
make run

# 运行测试
make test

# 运行测试并生成覆盖率报告
make test-cover

# 清理构建产物（包括根目录的bot文件）
make clean

# Docker操作
make docker-build      # 构建镜像
make docker-up         # 启动容器
make docker-down       # 停止容器
make docker-rebuild    # 重新构建并启动
make docker-logs       # 查看日志

# 代码质量
make fmt               # 格式化代码
make tidy              # 整理依赖
```

### Development (Manual Commands)
```bash
# Run the application
go run cmd/bot/main.go

# Run tests
go test ./...

# Run specific test suite
go test ./internal/service -run TestReminderService
go test ./pkg/config -run TestConfig

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Build for production
go build -o bin/mmemory cmd/bot/main.go

# Run with race detection
go run -race cmd/bot/main.go
```

### Database Operations
```bash
# Database migrations are handled automatically on startup
# SQLite database will be created at the path specified in config.yaml
```

### Docker Operations
```bash
# Build Docker image
docker build -t mmemory:latest .

# Run with docker-compose (basic)
docker-compose up -d

# Run with monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d

# Production deployment
docker run -d \
  -e MMEMORY_BOT_TOKEN=your_token \
  -v /path/to/data:/app/data \
  mmemory:latest
```

### Monitoring Operations
```bash
# Check application metrics
curl http://localhost:9090/metrics

# View monitoring stack
# Prometheus: http://localhost:9091
# Grafana: http://localhost:3000
# Alertmanager: http://localhost:9093
```

## Architecture Overview

The codebase follows a layered architecture pattern with clean separation of concerns:

### Core Components
- **Bot Layer** (`internal/bot/`): Telegram API integration and message routing
- **Service Layer** (`internal/service/`): Business logic including reminder management, scheduling, and AI parsing
- **AI Layer** (`internal/ai/`, `pkg/ai/`): AI service integration with OpenAI client and prompt management
- **Repository Layer** (`internal/repository/`): Data access abstraction with SQLite implementation
- **Models** (`internal/models/`): Domain entities including reminders, conversations, and AI parse results
- **Config Layer** (`pkg/config/`): Configuration management with hot-reload support
- **Server Layer** (`pkg/server/`): HTTP server for health checks and metrics

### Key Services
- **AIParserService**: OpenAI-powered natural language understanding with fallback strategy
- **ConversationService**: 30-day conversation history management for context-aware parsing
- **ReminderService**: Core business logic for creating, managing, and tracking reminders
- **SchedulerService**: Cron-based job scheduling with persistence and recovery
- **NotificationService**: Telegram message sending and user interaction handling
- **MonitoringService**: Prometheus metrics collection and system monitoring
- **ConfigManager**: Dynamic configuration loading with file watching

### Data Flow
1. User message → Bot handler → AIParserService (with fallback) → ReminderService → Repository → SQLite
2. Cron scheduler triggers → ReminderService → NotificationService → Telegram Bot API
3. User responses → Bot handlers → Service layer for status updates

## AI Integration Architecture

### OpenAI Integration (C1 Phase - Completed)
The system integrates OpenAI for intelligent natural language understanding with a robust fallback strategy:

**Core Components**:
- `pkg/ai/config.go`: AI configuration with default Prompt templates
- `internal/ai/openai_client.go`: OpenAI API client wrapper
- `internal/service/ai_parser.go`: AI parsing service with fallback chain
- `internal/service/conversation.go`: Conversation history management

**Fallback Strategy** (Four layers):
1. **Primary AI**: OpenAI primary model (configurable, e.g., LongCat-Flash-Chat)
2. **Backup AI**: OpenAI backup model (same as primary to ensure compatibility)
3. **Regex Parser**: Traditional pattern matching for simple commands
4. **Fallback Chat**: Generic response when all else fails

**Prompt Management**:
- Default Prompt templates built into `pkg/ai/config.go`
- Override via `configs/config.yaml` or environment variables
- Templates include ReminderParse and ChatResponse

**Configuration**:
```yaml
ai:
  enabled: true
  openai:
    api_key: "${MMEMORY_AI_OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"  # or custom endpoint
    primary_model: "gpt-4o-mini"
    backup_model: "gpt-4o-mini"  # should match primary for compatibility
    temperature: 0.1
    max_tokens: 1000
    timeout: "30s"
```

**Key Features**:
- Smart context building from conversation history
- Automatic fallback when AI fails or returns low confidence
- Empty prompt configuration defaults to built-in templates
- Support for third-party OpenAI-compatible APIs

## Database Schema

The system uses 5 core tables:
- **users**: User profiles and preferences
- **reminders**: Reminder configurations with schedule patterns
- **reminder_logs**: Execution history and status tracking
- **conversations**: Context management for AI parsing (30-day retention)
- **messages**: Individual message records within conversations

### Schedule Pattern Format
- `daily`: Every day
- `weekly:1,3,5`: Monday, Wednesday, Friday
- `monthly:1,15`: 1st and 15th of each month
- `once:2024-10-01`: One-time reminder on specific date

## Natural Language Processing

### AI-Powered Parsing (Primary)
The system uses OpenAI for intelligent natural language understanding:
- Complex time expressions: "工作日早上醒来后提醒我看书"
- Context-aware parsing using 30-day conversation history
- Intent recognition: reminder creation, chat, query, summary
- Confidence scoring with automatic fallback on low confidence

### Traditional Parser (Fallback)
Regex-based pattern matching for simple Chinese commands:
- Daily: `每天X点提醒我Y`
- Weekly: `每周[星期]X点提醒我Y`
- One-time: `YYYY年MM月DD日X点提醒我Y`

New patterns should be added to the traditional parser as a last resort. The AI parser should handle most natural language variations.

## Concurrency and State Management

- Each user message is processed in a separate goroutine
- Context-based timeout control for all operations
- Channel-based communication between scheduler and notification services
- GORM handles database connection pooling for SQLite

## Configuration Management

Uses Viper for configuration with hot-reload capabilities:
- YAML configuration files (`configs/config.yaml`, `configs/config.full.yaml`)
- Environment variable overrides with `MMEMORY_` prefix
- File watching for runtime configuration updates
- Validation with default fallbacks

Critical environment variables:
- `MMEMORY_BOT_TOKEN`: Required for bot authentication
- `MMEMORY_DATABASE_DSN`: SQLite database file location
- `MMEMORY_SERVER_PORT`: HTTP server port for health checks
- `MMEMORY_MONITORING_ENABLED`: Enable Prometheus metrics

**AI-Specific Variables**:
- `MMEMORY_AI_ENABLED`: Enable/disable AI functionality (default: false)
- `MMEMORY_AI_OPENAI_API_KEY`: OpenAI API key (required if AI enabled)
- `MMEMORY_AI_OPENAI_BASE_URL`: API endpoint (default: OpenAI, supports third-party)
- `MMEMORY_AI_OPENAI_PRIMARY_MODEL`: Primary model name (e.g., "gpt-4o-mini")
- `MMEMORY_AI_OPENAI_BACKUP_MODEL`: Backup model name (should match primary)
- `MMEMORY_AI_OPENAI_TEMPERATURE`: Model temperature (default: 0.1)
- `MMEMORY_AI_OPENAI_MAX_TOKENS`: Max tokens per request (default: 1000)
- `MMEMORY_AI_OPENAI_TIMEOUT`: Request timeout (default: 30s)
- `MMEMORY_AI_OPENAI_MAX_RETRIES`: Max retry attempts (default: 3)

## Error Handling

- Repository layer returns domain-specific errors
- Service layer handles business logic validation
- Bot layer provides user-friendly error messages in Chinese
- All errors are logged with structured logging using Logrus

## Development Workflow

### 阶段性开发流程
每个开发阶段必须按以下顺序完成：

1. **编写单元测试** - 为当前阶段的核心功能编写测试
2. **运行测试验证** - 确保所有测试通过，功能正常
3. **更新技术文档** - 更新项目方案文档，记录完成情况
4. **代码提交** - 提交当前阶段的完整代码
5. **更新计划文档** - 更新下一阶段的开发计划

### 测试要求
- 每个service层方法必须有对应的单元测试
- 数据库操作需要集成测试
- 关键业务逻辑需要边界值测试
- 错误处理路径需要测试覆盖

## Testing Strategy

- Unit tests for service layer business logic
- Repository integration tests with in-memory SQLite
- Mock implementations for external dependencies (Telegram API)
- Test data fixtures in `testdata/` directories
- Configuration validation tests with edge cases
- Architecture compliance tests for layer dependencies

## Monitoring and Observability

The system includes comprehensive monitoring capabilities:
- **Prometheus metrics**: Application performance, reminder execution, error rates
- **Grafana dashboards**: Pre-configured visualizations for system health
- **Alertmanager**: Automated alerts for critical system states
- **Health checks**: HTTP endpoints for service status validation
- **Structured logging**: JSON-formatted logs with correlation IDs

## Project Structure Conventions

- `cmd/`: Application entry points
- `internal/`: Private application code (not importable by other projects)
  - `internal/ai/`: OpenAI client and prompt management
  - `internal/bot/`: Telegram bot handlers
  - `internal/models/`: Domain models including AI parse results
  - `internal/repository/`: Data access layer
  - `internal/service/`: Business logic including AI parsing
- `pkg/`: Public packages that can be imported by external projects
  - `pkg/ai/`: AI configuration and types
  - `pkg/config/`: Configuration management
  - `pkg/logger/`: Logging utilities
  - `pkg/metrics/`: Prometheus metrics
  - `pkg/server/`: HTTP server for metrics
- `configs/`: Configuration files and examples
- `docs/`: Technical documentation and implementation reports
  - `docs/C1-AI-Parser-Implementation-20250929.md`: AI parser implementation (completed)
  - `docs/C2-AI-Provider-Implementation-20250930.md`: Multi-provider support (planned)
- `scripts/`: Build and deployment automation scripts
- `test/`: Integration and end-to-end tests

## Recent Changes and Project Status

### Phase 3 - AI Integration (In Progress)
- ✅ **C1 Completed** (2025-10-10): AI Parser Interface Design
  - OpenAI client integration
  - Fallback chain implementation
  - Conversation history management
  - Default Prompt templates
  - Bug fixes: Empty prompt config and backup model compatibility
- 📋 **C2 Planned**: Multi-AI Provider Support (OpenAI + Claude)
- 📋 **C3 Planned**: Intelligent Degradation Mechanism
- 📋 **C4 Planned**: Dual Parser Architecture Deployment

### Important Notes for Development
- **AI Configuration**: Always ensure backup model matches primary for third-party APIs
- **Prompt Templates**: System auto-fills empty prompts with defaults from `pkg/ai/config.go`
- **Build Process**: Use Makefile (`make build`) for consistent binary output to `bin/`
- **Testing AI**: Set `OPENAI_API_KEY` or equivalent for integration tests
- **Docker**: Environment variables from `.env` are auto-loaded by docker-compose