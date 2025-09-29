# MMemory监控系统

MMemory监控系统基于Prometheus + Grafana + AlertManager构建，提供全面的系统监控和告警功能。

## 🎯 监控目标

- **系统可用性**: 确保MMemory Bot服务稳定运行
- **性能指标**: 监控响应时间、吞吐量等关键性能指标
- **业务指标**: 跟踪用户活跃度、提醒处理效率等业务数据
- **资源使用**: 监控系统资源使用情况（CPU、内存、磁盘）
- **外部依赖**: 监控Telegram API等外部服务的可用性

## 🏗️ 架构组件

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   MMemory   │───▶│ Prometheus  │───▶│   Grafana   │
│     Bot     │    │  (指标收集)  │    │  (可视化)    │
└─────────────┘    └─────────────┘    └─────────────┘
       │                    │                    │
       ▼                    ▼                    ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ Node        │    │ AlertManager│    │  Dashboard  │
│ Exporter    │    │  (告警管理)  │    │  (监控面板)  │
│ (系统监控)   │    └─────────────┘    └─────────────┘
└─────────────┘
       │
       ▼
┌─────────────┐
│ Blackbox    │
│ Exporter    │
│ (外部监控)   │
└─────────────┘
```

## 📊 监控指标

### 业务指标
- **用户指标**: 注册用户总数、活跃用户趋势
- **提醒指标**: 创建/完成/跳过数量、活跃提醒数
- **消息指标**: Bot消息处理速率、成功率
- **解析指标**: 提醒解析耗时、解析成功率

### 系统指标
- **性能指标**: 响应时间、数据库查询性能
- **错误指标**: 各类错误发生率和分布
- **资源指标**: CPU、内存、磁盘使用率
- **运行指标**: 系统运行时间、调度任务数

### 外部指标
- **Telegram API**: 可用性和响应时间
- **网络连通性**: 外部服务可达性

## 🚀 快速开始

### 1. 部署监控系统

```bash
# 使用部署脚本
./scripts/deploy-monitoring.sh

# 或者手动启动
docker-compose -f docker-compose.monitoring.yml up -d
```

### 2. 访问监控界面

- **Grafana**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9091
- **AlertManager**: http://localhost:9093

### 3. 测试监控系统

```bash
# 运行测试脚本
./scripts/test-monitoring.sh
```

## ⚙️ 配置说明

### Prometheus配置 (`configs/prometheus.yml`)

```yaml
# 主要配置项
- job_name: 'mmemory'
  static_configs:
    - targets: ['localhost:9090']  # MMemory指标端口
  scrape_interval: 15s
  metrics_path: '/metrics'
```

### 告警规则 (`configs/alerts/mmemory.yml`)

```yaml
# 关键告警规则
- alert: MMemoryDown
  expr: up{job="mmemory"} == 0
  for: 1m
  labels:
    severity: critical
```

### Grafana面板 (`configs/grafana/mmemory-dashboard.json`)

预配置的监控面板包含：
- 系统概览
- 消息处理监控
- 提醒处理监控
- 性能监控
- 资源监控
- 错误监控

## 📈 关键指标说明

| 指标名称 | 类型 | 说明 | 正常范围 |
|---------|------|------|----------|
| mmemory_system_uptime_seconds | Gauge | 系统运行时间 | > 3600 |
| mmemory_bot_users_total | Gauge | 注册用户总数 | 持续增长 |
| mmemory_reminders_total | Gauge | 提醒数量（按状态） | 根据业务变化 |
| mmemory_bot_messages_total | Counter | Bot消息处理总数 | 稳定增长 |
| mmemory_response_duration_seconds | Histogram | 响应时间 | < 2s (P95) |
| mmemory_errors_total | Counter | 错误总数 | 越低越好 |
| up{job="mmemory"} | Gauge | 服务可用性 | = 1 |

## 🚨 告警说明

### 关键告警（Critical）
- **MMemoryDown**: 服务不可用
- **MMemoryDatabaseErrors**: 数据库错误率过高
- **TelegramAPIUnavailable**: Telegram API不可用
- **MMemoryLowDiskSpace**: 磁盘空间不足

### 警告告警（Warning）
- **MMemoryHighErrorRate**: 错误率过高
- **MMemoryHighResponseTime**: 响应时间过长
- **MMemoryReminderBacklog**: 提醒积压过多
- **MMemoryHighCPUUsage**: CPU使用率过高

## 🔧 运维操作

### 查看服务状态
```bash
docker-compose -f docker-compose.monitoring.yml ps
```

### 查看日志
```bash
# 查看所有服务日志
docker-compose -f docker-compose.monitoring.yml logs -f

# 查看特定服务日志
docker-compose -f docker-compose.monitoring.yml logs -f prometheus
```

### 重启服务
```bash
docker-compose -f docker-compose.monitoring.yml restart
```

### 停止服务
```bash
docker-compose -f docker-compose.monitoring.yml down
```

### 数据备份
```bash
# 备份Prometheus数据
docker run --rm -v mmemory_prometheus_data:/data -v $(pwd):/backup alpine tar czf /backup/prometheus-backup.tar.gz /data

# 备份Grafana数据
docker run --rm -v mmemory_grafana_data:/data -v $(pwd):/backup alpine tar czf /backup/grafana-backup.tar.gz /data
```

## 🛠️ 故障排除

### 服务无法启动
1. 检查端口冲突
2. 验证配置文件语法
3. 查看Docker日志

### 指标收集异常
1. 检查MMemory应用是否正常
2. 验证Prometheus配置
3. 测试指标端点: `curl http://localhost:9090/metrics`

### 告警不触发
1. 检查告警规则语法
2. 验证AlertManager配置
3. 手动触发测试告警

### Grafana无数据显示
1. 检查数据源配置
2. 验证Prometheus查询
3. 检查时间范围设置

## 📚 进阶配置

### 自定义告警规则
编辑 `configs/alerts/mmemory.yml` 文件，添加新的告警规则。

### 配置邮件通知
修改 `configs/alertmanager.yml` 中的SMTP配置：

```yaml
global:
  smtp_smarthost: 'your-smtp-server:587'
  smtp_from: 'alerts@your-domain.com'
  smtp_auth_username: 'your-username'
  smtp_auth_password: 'your-password'
```

### 添加Slack通知
在 `configs/alertmanager.yml` 中添加Slack配置：

```yaml
receivers:
  - name: 'slack-alerts'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#alerts'
        title: 'MMemory Alert'
```

### 扩展监控目标
在 `configs/prometheus.yml` 中添加新的监控目标：

```yaml
scrape_configs:
  - job_name: 'new-service'
    static_configs:
      - targets: ['new-service:port']
```

## 📖 相关文档

- [Prometheus官方文档](https://prometheus.io/docs/)
- [Grafana官方文档](https://grafana.com/docs/)
- [AlertManager官方文档](https://prometheus.io/docs/alerting/latest/alertmanager/)
- [Docker Compose文档](https://docs.docker.com/compose/)

## 🤝 贡献指南

欢迎提交Issue和Pull Request来改进监控系统。

## 📄 许可证

本监控系统与MMemory项目使用相同的许可证。