# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## about me
My name is chenwl. I am a software engineer based in Beijing  China. My English is not good, please communicate with me in Chinese.

## Project Overview

MMemory is a Telegram Bot-based intelligent reminder tool built with Go. The system enables conversational interaction for managing daily habits and task reminders through natural language processing.

## Development Commands

### Initial Setup
```bash
# Initialize Go module and dependencies
go mod init mmemory
go mod tidy

# Copy and configure settings
cp configs/config.example.yaml configs/config.yaml
# Set TELEGRAM_BOT_TOKEN in config.yaml
```

### Development
```bash
# Run the application
go run cmd/bot/main.go

# Run tests
go test ./...

# Run specific test
go test ./internal/service -run TestReminderService

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
docker build -t mmemory:latest -f docker/Dockerfile .

# Run with docker-compose
docker-compose -f docker/docker-compose.yml up -d

# Production deployment
docker run -d \
  -e TELEGRAM_BOT_TOKEN=your_token \
  -v /path/to/data:/app/data \
  mmemory:latest
```

## Architecture Overview

The codebase follows a layered architecture pattern with clean separation of concerns:

### Core Components
- **Bot Layer** (`internal/bot/`): Telegram API integration and message routing
- **Service Layer** (`internal/service/`): Business logic including reminder management, scheduling, and natural language parsing
- **Repository Layer** (`internal/repository/`): Data access abstraction with SQLite implementation
- **Models** (`internal/models/`): Domain entities and data structures

### Key Services
- **ReminderService**: Core business logic for creating, managing, and tracking reminders
- **SchedulerService**: Cron-based job scheduling with persistence and recovery
- **ParserService**: Natural language processing for converting user messages to structured reminders
- **NotificationService**: Telegram message sending and user interaction handling

### Data Flow
1. Telegram messages → Bot handlers → Service layer → Repository layer → SQLite
2. Cron scheduler triggers → Service layer → Notification service → Telegram Bot API
3. User responses → Bot handlers → Service layer for status updates

## Database Schema

The system uses 4 core tables:
- **users**: User profiles and preferences
- **reminders**: Reminder configurations with schedule patterns
- **reminder_logs**: Execution history and status tracking  
- **conversations**: Context management for complex interactions

### Schedule Pattern Format
- `daily`: Every day
- `weekly:1,3,5`: Monday, Wednesday, Friday
- `monthly:1,15`: 1st and 15th of each month
- `once:2024-10-01`: One-time reminder on specific date

## Natural Language Processing

The parser supports Chinese language patterns:
- Daily: `每天X点提醒我Y`
- Weekly: `每周[星期]X点提醒我Y`  
- One-time: `YYYY年MM月DD日X点提醒我Y`

New patterns should be added to the parser service with corresponding regex patterns and type classification.

## Concurrency and State Management

- Each user message is processed in a separate goroutine
- Context-based timeout control for all operations
- Channel-based communication between scheduler and notification services
- GORM handles database connection pooling for SQLite

## Configuration Management

Uses Viper for configuration with support for:
- YAML configuration files (`configs/config.yaml`)
- Environment variable overrides
- Default values for development

Critical environment variables:
- `TELEGRAM_BOT_TOKEN`: Required for bot authentication
- `DATABASE_PATH`: SQLite database file location
- `PORT`: HTTP server port for health checks

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