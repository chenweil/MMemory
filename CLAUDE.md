# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## about me
My name is chenwl. I am a software engineer based in Beijing  China. My English is not good, please communicate with me in Chinese.

## Project Overview

MMemory is a Telegram Bot-based intelligent reminder tool built with Go. The system enables conversational interaction for managing daily habits and task reminders through AI-powered natural language processing.

### Key Features
- **Multi-AI Provider Support**: OpenAI, Claude with intelligent provider selection (C3 phase completed)
- **Smart Conversation**: 30-day conversation history with intelligent context management
- **Intelligent Degradation**: C3 four-layer degradation with cost control and monitoring
- **Context-Aware Suggestions**: AI-powered reminder optimization based on user behavior patterns
- **Weather Integration**: Location-aware weather-based reminder triggers
- **Scheduler System**: Cron-based reminder execution with persistence and recovery
- **Advanced Monitoring**: Prometheus metrics, Grafana dashboards, and intelligent alerting

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
make lint              # 代码质量检查

# 版本管理
make version           # 显示版本信息
make build VERSION=v1.0.0  # 指定版本号构建
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
- **ReminderService**: Core business logic for creating, managing, and tracking reminders
- **SchedulerService**: Cron-based job scheduling with persistence and recovery
- **NotificationService**: Telegram message sending and intelligent user interaction handling
- **AIParserService**: Multi-AI natural language understanding with C3 intelligent degradation
- **IntelligentAIManager**: Multi-provider AI management with intelligent selection and cost control
- **ContextManager**: Intelligent conversation context management and user behavior analysis
- **ConversationService**: 30-day conversation history with enhanced context awareness
- **SuggestionService**: AI-powered reminder optimization and behavioral suggestions
- **WeatherService**: Location-aware weather integration and conditional triggers
- **DailyActivityService**: User activity tracking (drinking water, reading, exercise, etc.)
- **ActivityAnalysisService**: Pattern detection and habit formation analysis
- **ActivityVisualizationService**: ASCII charts and activity statistics
- **PatternDetectorService**: User behavior pattern recognition
- **ConditionEvaluatorService**: Conditional trigger evaluation engine
- **CostMonitorService**: AI usage cost tracking and budget management
- **MonitoringService**: Advanced Prometheus metrics with intelligent alerting

### Data Flow
1. User message → Bot handler → ContextManager → IntelligentAIManager → AIParserService → ReminderService → Repository → SQLite
2. ContextManager updates user behavior patterns → SuggestionService → IntelligentAIManager for personalized responses
3. WeatherService integration → Location-based triggers → ReminderService enhancement
4. Cron scheduler triggers → ReminderService → NotificationService → Telegram Bot API
5. User responses → Bot handlers → Service layer → C3 monitoring and cost control updates

## AI Integration Architecture

### Multi-Provider AI Integration (C3 Phase - Completed)
The system features an advanced multi-provider AI architecture with intelligent degradation and cost control:

**Core Components**:
- `pkg/ai/manager.go`: Unified AI provider management with intelligent selection
- `pkg/ai/provider_interface.go`: Standardized provider interface for extensibility
- `pkg/ai/openai_provider.go`: OpenAI provider implementation with cost tracking
- `pkg/ai/claude_provider.go`: Claude AI provider implementation
- `pkg/ai/intelligent_manager.go`: Intelligent provider selection and degradation
- `pkg/ai/c3_integration.go`: C3 intelligent degradation mechanism
- `pkg/ai/fallback_strategy.go`: Smart fallback strategy with confidence scoring
- `pkg/ai/cost_controller.go`: AI usage cost optimization and budget management
- `pkg/ai/advanced_monitoring.go`: Real-time monitoring and alerting

**C3 Intelligent Degradation** (Enhanced Four layers):
1. **Primary AI**: OpenAI GPT-4o-mini with intelligent cost monitoring
2. **Secondary AI**: Claude 3.5 Sonnet with provider failover
3. **Enhanced Regex Parser**: Context-aware pattern matching
4. **Intelligent Fallback**: Context-preserving chat responses with learning

**Provider Selection Strategy**:
- **Cost-based**: Automatically select most cost-effective provider based on task complexity
- **Performance-based**: Route to providers with best response times for specific tasks
- **Availability-based**: Real-time health checks with automatic failover
- **Budget-aware**: Respect per-user and global AI usage budgets

**Configuration**:
```yaml
ai:
  enabled: true
  openai:
    api_key: "${MMEMORY_AI_OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    primary_model: "gpt-4o-mini"
    backup_model: "gpt-3.5-turbo"
    temperature: 0.1
    max_tokens: 1000
    timeout: "30s"
    max_retries: 3
```

**Advanced Features**:
- **Intelligent Cost Control**: Real-time budget monitoring and provider selection
- **Dynamic Provider Selection**: AI-powered provider choice based on task requirements
- **Confidence Scoring**: Automatic degradation when confidence drops below threshold
- **Learning System**: Improves provider selection based on historical performance
- **Context Preservation**: Maintains conversation context across provider switches
- **Real-time Monitoring**: Provider health, response times, and cost tracking

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
- `MMEMORY_AI_OPENAI_API_KEY`: OpenAI API key
- `MMEMORY_AI_OPENAI_BASE_URL`: OpenAI API endpoint (supports third-party)
- `MMEMORY_AI_OPENAI_PRIMARY_MODEL`: Primary model name (default: "gpt-4o-mini")
- `MMEMORY_AI_OPENAI_BACKUP_MODEL`: Backup model name (default: "gpt-3.5-turbo")

**C3 Degradation Variables**:
- `MMEMORY_AI_C3_ENABLED`: Enable C3 intelligent degradation (default: true)
- `MMEMORY_AI_C3_BUDGET_LIMIT_PER_USER`: Daily AI budget per user in USD (default: 10.0)
- `MMEMORY_AI_C3_GLOBAL_BUDGET_LIMIT`: Global daily AI budget in USD (default: 100.0)
- `MMEMORY_AI_C3_DEGRADATION_THRESHOLD`: Confidence threshold for fallback (default: 0.7)
- `MMEMORY_AI_FALLBACK_CONFIDENCE_THRESHOLD`: Minimum confidence for AI responses (default: 0.6)

**Weather Service Variables**:
- `MMEMORY_WEATHER_API_KEY`: Weather API key (e.g., QWeather)
- `MMEMORY_WEATHER_BASE_URL`: Weather API endpoint
- `MMEMORY_WEATHER_TIMEOUT`: Weather API request timeout (default: 10s)

**General AI Configuration**:
- `MMEMORY_AI_TEMPERATURE`: Model temperature (default: 0.1)
- `MMEMORY_AI_MAX_TOKENS`: Max tokens per request (default: 1000)
- `MMEMORY_AI_TIMEOUT`: AI request timeout (default: 30s)
- `MMEMORY_AI_MAX_RETRIES`: Max retry attempts (default: 3)

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
  - `internal/models/`: Domain models including reminders, activities, weather
  - `internal/repository/`: Data access layer with SQLite implementation
  - `internal/service/`: Business logic including AI parsing, activities, monitoring
- `pkg/`: Public packages that can be imported by external projects
  - `pkg/ai/`: AI configuration, types, provider interfaces, cost control
  - `pkg/config/`: Configuration management with hot-reload
  - `pkg/logger/`: Logging utilities
  - `pkg/metrics/`: Prometheus metrics
  - `pkg/server/`: HTTP server for metrics
  - `pkg/version/`: Version information management
- `configs/`: Configuration files and examples
- `docs/`: Technical documentation and implementation reports
- `scripts/`: Build and deployment automation scripts
- `test/`: Integration and end-to-end tests

## Additional Services

### Weather Integration
The system integrates with weather APIs to provide location-aware reminder triggers:
- **Core Components**:
  - `internal/service/weather.go`: Weather service implementation
  - `internal/models/weather.go`: Weather data models
  - Support for multiple weather providers (QWeather, OpenWeatherMap)
- **Features**:
  - Location-based weather monitoring
  - Conditional reminders based on weather conditions
  - Weather forecasts for intelligent reminder scheduling

### Context-Aware Suggestions
AI-powered suggestion system that learns from user behavior:
- **Core Components**:
  - `internal/service/suggestion.go`: Suggestion engine
  - `internal/service/context_manager.go`: Context management and analysis
  - Behavioral pattern recognition and optimization recommendations
- **Features**:
  - Analyzes user reminder patterns
  - Suggests optimal reminder timing
  - Identifies potential reminder conflicts
  - Provides personalized productivity insights

## Recent Changes and Project Status

### Phase 4 - C6 User Life Recording System (Completed C1-C3, C4, C6)
- ✅ **C1 Completed** (2025-10-10): AI Parser Interface Design
  - OpenAI client integration, fallback chain, conversation history
- ✅ **C2 Completed** (2025-10-15): Multi-AI Provider Support
  - Claude AI integration, dynamic provider selection, cost tracking
- ✅ **C3 Completed** (2025-10-20): Intelligent Degradation Mechanism
  - Cost control, confidence-based degradation, real-time monitoring
- ✅ **C4 Completed** (2025-11-09): Enhanced Features
  - Weather service, context-aware suggestions, Bot API improvements
- ✅ **C5 Completed** (2025-12-30): Performance Optimization
  - Database connection pool optimization, dynamic worker pools, AI cost reduction
- ✅ **C6 Completed** (2025-12-31): User Life Recording System
  - Activity tracking (drink water, read book, exercise, etc.)
  - AI-powered activity extraction from conversations
  - Pattern detection and habit formation analysis
  - ASCII visualization and activity statistics
  - Test coverage improved to ~60%

### Important Notes for Development
- **Build Process**: Use Makefile (`make build`) for consistent binary output to `bin/` with `CGO_ENABLED=1`
- **Version Management**: Current version is `v0.4.0-dev`, use `/version` command in bot for details
- **AI Testing**: Set `MMEMORY_AI_OPENAI_API_KEY` for AI integration tests
- **Activity Recording**: C6 introduces new daily activity tracking features
- **Monitoring**: Access metrics at `http://localhost:9090/metrics`
- **Docker**: All environment variables are auto-loaded from `.env` file