# MMemory 配置管理文档

## 概述

MMemory 使用增强的配置管理系统，支持多种配置源、热更新、环境变量覆盖等功能。配置系统采用分层设计，优先级从高到低为：

1. 环境变量
2. 配置文件
3. 默认值

## 配置源

### 1. 配置文件

支持 YAML 格式的配置文件，默认查找路径：
- `./configs/config.yaml`
- `./config.yaml`

### 2. 环境变量

所有配置项都可通过环境变量覆盖，命名规则：
- 前缀：`MMEMORY_`
- 路径分隔符：`.` → `_`

例如：
```bash
MMEMORY_BOT_TOKEN=your_token_here
MMEMORY_DATABASE_DSN=./data/mmemory.db
MMEMORY_LOGGING_LEVEL=debug
```

### 3. 默认值

系统内置了合理的默认值，确保最小化配置即可运行。

## 配置结构

### 根配置结构

```yaml
bot:        # Telegram Bot 配置
database:   # 数据库配置
server:     # 服务器配置
scheduler:  # 调度器配置
logging:    # 日志配置
app:        # 应用配置
monitoring: # 监控配置
```

### 详细配置说明

#### Bot 配置 (`bot`)

| 字段 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `token` | string | 是 | - | Telegram Bot Token |
| `debug` | bool | 否 | false | 调试模式 |
| `webhook.enabled` | bool | 否 | false | 启用 Webhook |
| `webhook.url` | string | 否 | - | Webhook URL |
| `webhook.port` | int | 否 | 8443 | Webhook 端口 |

#### 数据库配置 (`database`)

| 字段 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `driver` | string | 否 | sqlite3 | 数据库驱动 (sqlite3, mysql, postgres) |
| `dsn` | string | 是 | - | 数据库连接字符串 |
| `max_open_conns` | int | 否 | 25 | 最大连接数 |
| `max_idle_conns` | int | 否 | 10 | 最大空闲连接数 |

#### 服务器配置 (`server`)

| 字段 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `port` | string | 否 | "8080" | 服务器端口 |
| `host` | string | 否 | "0.0.0.0" | 服务器主机地址 |

#### 调度器配置 (`scheduler`)

| 字段 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `timezone` | string | 否 | "Asia/Shanghai" | 时区 |
| `max_workers` | int | 否 | 10 | 最大工作线程数 |

#### 日志配置 (`logging`)

| 字段 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `level` | string | 否 | "info" | 日志级别 (debug, info, warn, error) |
| `format` | string | 否 | "json" | 日志格式 (json, text) |
| `output` | string | 否 | "stdout" | 输出方式 (stdout, file, both) |
| `file_path` | string | 否 | "./data/mmemory.log" | 日志文件路径 |

#### 应用配置 (`app`)

| 字段 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `name` | string | 否 | "MMemory" | 应用名称 |
| `version` | string | 否 | "v0.0.1" | 应用版本 |
| `environment` | string | 否 | "development" | 应用环境 (development, testing, staging, production) |

#### 监控配置 (`monitoring`)

| 字段 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `enabled` | bool | 否 | true | 启用监控 |
| `port` | int | 否 | 9090 | 监控端口 |
| `path` | string | 否 | "/metrics" | 监控路径 |

## 配置热更新

系统支持配置热更新，无需重启服务即可应用配置变更。

### 启用热更新

```go
import (
    "context"
    "mmemory/pkg/config"
    "mmemory/pkg/logger"
)

func main() {
    // 创建配置管理器
    configManager := config.NewConfigManager()
    
    // 加载配置
    cfg, err := configManager.Load()
    if err != nil {
        log.Fatal("加载配置失败:", err)
    }
    
    // 启用配置热更新
    ctx := context.Background()
    if err := configManager.WatchConfig(ctx); err != nil {
        log.Fatal("启用配置监听失败:", err)
    }
}
```

### 配置变更监听器

可以添加自定义的配置变更监听器：

```go
// 创建日志配置监听器
loggingListener := config.NewLoggingConfigListener(func(level, format, output, filePath string) {
    logger.Infof("日志配置已更新: level=%s, format=%s, output=%s", level, format, output)
    // 重新初始化日志系统
    logger.Init(level, format, output, filePath)
})

// 注册监听器
configManager.AddWatcher(loggingListener)
```

### 热更新管理器

对于复杂的配置变更，可以使用热更新管理器：

```go
// 创建热更新管理器
hotReloadManager := config.NewHotReloadManager(configManager)

// 注册重载处理器
hotReloadManager.RegisterReloadHandler("database", func(newConfig *config.Config) error {
    logger.Info("数据库配置发生变更")
    // 更新数据库连接池配置
    return updateDatabaseConfig(newConfig.Database)
})

// 注册安全重载函数
hotReloadManager.RegisterSafeReloadFunc("logging", func(newConfig *config.Config) error {
    logger.Info("安全重载日志配置")
    return logger.Init(newConfig.Logging.Level, newConfig.Logging.Format, newConfig.Logging.Output, newConfig.Logging.FilePath)
})

// 启动热更新管理
if err := hotReloadManager.Start(ctx); err != nil {
    log.Fatal("启动热更新管理失败:", err)
}
```

## 配置验证

系统提供强大的配置验证功能，确保配置的正确性。

### 默认验证

```go
// 使用默认验证器
validator := config.GetDefaultValidator()
result := validator.Validate(cfg)

if !result.IsValid {
    for _, err := range result.Errors {
        fmt.Printf("配置错误 [%s]: %s\n", err.Field, err.Message)
    }
}
```

### 自定义验证规则

```go
// 创建自定义验证器
validator := config.NewConfigValidator()

// 添加自定义验证规则
validator.AddRule(config.ValidationRule{
    Field:       "bot.token",
    Required:    true,
    Description: "Telegram Bot Token",
    Validator: func(value interface{}) error {
        token, ok := value.(string)
        if !ok {
            return fmt.Errorf("Token必须是字符串类型")
        }
        if len(token) < 40 {
            return fmt.Errorf("Token格式不正确")
        }
        return nil
    },
})

// 执行验证
result := validator.Validate(cfg)
```

## 配置优先级示例

### 场景1：数据库配置

```yaml
# config.yaml
database:
  driver: sqlite3
  dsn: ./data/mmemory.db
```

```bash
# 环境变量覆盖
export MMEMORY_DATABASE_DRIVER=mysql
export MMEMORY_DATABASE_DSN="user:pass@tcp(localhost:3306)/mmemory"
```

结果：使用 MySQL 配置，因为环境变量优先级更高。

### 场景2：日志级别

```yaml
# config.yaml
logging:
  level: info
```

```bash
# 环境变量覆盖
export MMEMORY_LOGGING_LEVEL=debug
```

结果：日志级别为 debug。

## 最佳实践

### 1. 生产环境配置

```yaml
# config.production.yaml
app:
  environment: production
  
logging:
  level: info
  format: json
  output: file
  file_path: /var/log/mmemory/app.log

database:
  driver: postgres
  dsn: "host=postgres user=mmemory password=secret dbname=mmemory port=5432 sslmode=require"
  max_open_conns: 50
  max_idle_conns: 20

monitoring:
  enabled: true
  port: 9090
  path: /metrics
```

### 2. 开发环境配置

```yaml
# config.development.yaml
app:
  environment: development
  
bot:
  debug: true
  
logging:
  level: debug
  format: text
  output: stdout

database:
  driver: sqlite3
  dsn: ./data/mmemory_dev.db
```

### 3. Docker 环境配置

```yaml
# config.docker.yaml
server:
  host: 0.0.0.0
  port: "8080"

database:
  driver: postgres
  dsn: "host=db user=mmemory password=password dbname=mmemory port=5432 sslmode=disable"
```

## 故障排除

### 配置文件未找到

如果配置文件不存在，系统会使用默认值和环境变量。确保配置文件路径正确：

```bash
# 检查配置文件是否存在
ls -la ./configs/config.yaml
ls -la ./config.yaml
```

### 配置验证失败

仔细检查错误信息，确保所有必需字段都已正确配置：

```bash
# 使用完整配置示例作为起点
cp configs/config.full.yaml configs/config.yaml
# 然后修改必要的配置项
```

### 热更新不工作

确保：
1. 配置文件有写权限
2. 文件系统支持 inotify (Linux)
3. 配置文件路径正确

### 环境变量不生效

检查环境变量命名：
```bash
# 正确的命名
env | grep MMEMORY_

# 确保没有拼写错误
echo $MMEMORY_BOT_TOKEN
echo $MMEMORY_DATABASE_DSN
```

## 相关文件

- `configs/config.full.yaml` - 完整配置示例
- `configs/config.example.yaml` - 基础配置示例
- `configs/.env.example` - 环境变量示例
- `pkg/config/config.go` - 核心配置管理器
- `pkg/config/hot_reload.go` - 热更新管理器
- `pkg/config/validator.go` - 配置验证器
- `pkg/config/watcher.go` - 配置监听器