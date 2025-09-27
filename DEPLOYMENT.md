# MMemory 部署指南

## 快速开始

### 1. 环境要求
- Docker 20.0+
- Docker Compose 2.0+
- 有效的 Telegram Bot Token

### 2. 获取 Telegram Bot Token
1. 在 Telegram 中搜索 `@BotFather`
2. 发送 `/newbot` 创建新的机器人
3. 按提示设置机器人名称和用户名
4. 保存获得的 Bot Token

### 3. 部署步骤

```bash
# 1. 克隆项目
git clone <repository-url>
cd MMemory

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env 文件，设置你的 TELEGRAM_BOT_TOKEN

# 3. 启动服务
./deploy.sh start

# 4. 查看状态
./deploy.sh status

# 5. 查看日志
./deploy.sh logs
```

## 部署脚本使用

`./deploy.sh` 脚本提供了完整的服务管理功能：

### 基本命令
```bash
./deploy.sh start     # 启动服务
./deploy.sh stop      # 停止服务
./deploy.sh restart   # 重启服务
./deploy.sh status    # 查看状态
./deploy.sh logs      # 查看日志
```

### 构建和维护
```bash
./deploy.sh build     # 重新构建镜像
./deploy.sh clean     # 清理所有数据（危险操作）
```

### 数据备份和恢复
```bash
./deploy.sh backup    # 备份数据
./deploy.sh restore backup_file.tar.gz  # 恢复备份
```

## 配置选项

### 环境变量
在 `.env` 文件中可以配置以下选项：

```bash
# 必需配置
TELEGRAM_BOT_TOKEN=your_bot_token_here

# 可选配置
ENVIRONMENT=production
LOG_LEVEL=info
LOG_FORMAT=json
TZ=Asia/Shanghai
```

### 数据持久化
- 数据库文件存储在 `./data/mmemory.db`
- 通过 Docker volume 自动持久化
- 定期备份建议使用 `./deploy.sh backup`

### 健康检查
服务自带健康检查机制：
- 每30秒检查一次进程状态
- 连续3次失败后重启容器
- 可通过 `./deploy.sh status` 查看健康状态

## 生产环境部署

### 服务器要求
- 最小配置：1 CPU, 512MB RAM, 10GB 存储
- 推荐配置：2 CPU, 1GB RAM, 20GB 存储
- 操作系统：Linux (Ubuntu 20.04+ 推荐)

### 安全建议
1. **防火墙配置**：只开放必要端口
2. **定期备份**：设置自动备份脚本
3. **监控告警**：监控服务状态和资源使用
4. **日志管理**：定期清理日志文件

### 自动备份脚本
```bash
#!/bin/bash
# 添加到 crontab: 0 2 * * * /path/to/backup.sh

cd /path/to/MMemory
./deploy.sh backup

# 清理7天前的备份
find . -name "mmemory_backup_*.tar.gz" -mtime +7 -delete
```

## 云平台部署

### Railway 部署
1. 连接 GitHub 仓库
2. 设置环境变量 `TELEGRAM_BOT_TOKEN`
3. 自动部署

### Render 部署
1. 选择 Docker 部署
2. 设置环境变量
3. 配置持久化磁盘

### 阿里云/腾讯云部署
1. 使用云服务器 ECS
2. 安装 Docker 和 Docker Compose
3. 按本指南步骤部署

## 故障排除

### 常见问题

#### 1. 容器启动失败
```bash
# 查看详细日志
./deploy.sh logs

# 检查配置
cat .env

# 重新构建
./deploy.sh build
./deploy.sh restart
```

#### 2. Bot 无响应
```bash
# 检查 Token 是否正确
# 确认网络连接正常
# 查看错误日志

./deploy.sh logs
```

#### 3. 数据库错误
```bash
# 检查数据目录权限
ls -la ./data/

# 重启服务
./deploy.sh restart
```

### 性能优化

#### 1. 资源限制
在 `docker-compose.yml` 中添加：
```yaml
services:
  mmemory:
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
```

#### 2. 日志优化
```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

## 监控和维护

### 系统监控
```bash
# 查看容器资源使用
docker stats

# 查看磁盘使用
df -h ./data/

# 查看数据库大小
ls -lh ./data/mmemory.db
```

### 定期维护
1. **每天**：检查服务状态
2. **每周**：查看日志和资源使用
3. **每月**：数据备份和清理
4. **每季度**：更新依赖和安全补丁

## 升级指南

### 应用升级
```bash
# 1. 备份数据
./deploy.sh backup

# 2. 拉取最新代码
git pull

# 3. 重新构建和启动
./deploy.sh build
./deploy.sh restart

# 4. 验证服务
./deploy.sh status
```

### 回滚操作
```bash
# 如果升级出现问题，可以回滚
git checkout previous_version
./deploy.sh build
./deploy.sh restart
```