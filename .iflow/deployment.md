# MMemory - 部署文档

## 开发环境设置

### 基础要求
- Go 1.23+
- SQLite3
- Telegram Bot Token

### 快速启动
```bash
# 1. 安装依赖
go mod tidy

# 2. 复制配置文件
cp configs/config.example.yaml configs/config.yaml

# 3. 编辑配置文件，设置 Bot Token
# 编辑 configs/config.yaml 中的 bot.token

# 4. 运行应用
go run cmd/bot/main.go

# 或构建二进制文件
go build -o bin/mmemory cmd/bot/main.go
```

### 测试
```bash
# 运行所有测试
go test ./...

# 运行特定模块测试
go test ./internal/service
```

## 配置管理

### 配置文件位置
- 主配置: `configs/config.yaml`
- 环境变量: `.env` (通过 deploy.sh 使用)

### 关键配置项
```yaml
bot:
  token: "your_telegram_bot_token"  # 必须设置
  debug: false                      # 调试模式

database:
  dsn: "./data/mmemory.db"         # 数据库路径
  max_open_conns: 25               # 连接池设置
  max_idle_conns: 10

monitoring:
  enabled: true                    # 监控开关
  port: 9090                       # 指标端口

logging:
  level: "info"                    # 日志级别
  format: "json"                   # 日志格式
```

### 配置热更新
项目支持配置热更新功能：
- 日志配置实时生效
- 数据库连接池动态调整
- Bot 调试模式切换

## 部署和运维

### Docker 部署
```bash
# 使用部署脚本
./deploy.sh start

# 手动部署
docker-compose up -d
```

### 部署脚本功能
- `./deploy.sh start` - 启动服务
- `./deploy.sh stop` - 停止服务
- `./deploy.sh restart` - 重启服务
- `./deploy.sh status` - 查看状态
- `./deploy.sh logs` - 查看日志
- `./deploy.sh backup` - 数据备份

### 监控和告警
- Prometheus 指标端点: `:9090/metrics`
- Grafana 仪表板配置在 `configs/grafana/`
- 告警规则在 `configs/alerts/mmemory.yml`