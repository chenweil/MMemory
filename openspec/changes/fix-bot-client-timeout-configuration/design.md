# 技术设计文档：Bot 客户端超时配置优化

## 设计目标

优化 Telegram Bot 客户端的超时配置和重试逻辑，提高 Bot 在网络不稳定环境下的稳定性，减少不必要的重试日志。

## 架构设计

### 当前架构问题

```
┌─────────────────────────────────────────┐
│  HTTP Client (client.go)                │
│  - Timeout: 120s                        │
│  - Transport: Custom                    │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│  GetUpdatesChan (main.go)               │
│  - Timeout: 30s ❌ (覆盖 HTTP Client)   │
│  - Retry: Fixed 3s delay                │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│  Error Handling                         │
│  - No error type distinction            │
│  - Log spam on retries                  │
└─────────────────────────────────────────┘
```

### 优化后架构

```
┌─────────────────────────────────────────┐
│  HTTP Client (client.go)                │
│  - Timeout: 120s                        │
│  - Transport: Custom                    │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│  GetUpdatesChan (main.go)               │
│  - Timeout: 60s ✅ (合理配置)           │
│  - Retry: Exponential backoff           │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│  Error Handler                          │
│  - Distinguish transient/fatal errors   │
│  - Controlled retry logging             │
│  - Connection health check              │
└─────────────────────────────────────────┘
```

## 详细设计

### 1. 超时配置优化

#### 1.1 GetUpdatesChan 超时时间调整

**当前配置**：
```go
u.Timeout = 30 // 30秒超时
```

**优化后配置**：
```go
u.Timeout = 60 // 60秒超时，更合理的网络等待时间
```

**设计理由**：
- Telegram Bot API 的 GetUpdates 方法使用长轮询，超时时间应该足够长
- 60 秒是 Telegram 推荐的合理超时时间范围（30-120 秒）
- 与 HTTP 客户端的 120 秒总超时保持 1:2 的比例，留出足够的时间余量

#### 1.2 HTTP 客户端配置验证

确认 `internal/bot/client.go` 中的配置：
```go
client := &http.Client{
    Timeout: 120 * time.Second, // 总超时时间
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        TLSHandshakeTimeout: 10 * time.Second,
        DisableKeepAlives:   false,
        ForceAttemptHTTP2:   true,
    },
}
```

**设计理由**：
- 120 秒总超时为 GetUpdatesChan 的 60 秒超时留出足够余量
- HTTP/2 支持提高连接复用效率
- 连接池配置优化并发性能

### 2. 指数退避重试策略

#### 2.1 退避算法

```go
type RetryConfig struct {
    MaxRetries      int           // 最大重试次数: 5
    InitialDelay    time.Duration // 初始延迟: 1s
    MaxDelay        time.Duration // 最大延迟: 60s
    BackoffFactor   float64       // 退避因子: 2.0
}

func calculateBackoff(attempt int, config RetryConfig) time.Duration {
    delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt))
    if delay > float64(config.MaxDelay) {
        delay = float64(config.MaxDelay)
    }
    return time.Duration(delay)
}
```

**重试延迟序列**：
- 第 1 次重试：1s
- 第 2 次重试：2s
- 第 3 次重试：4s
- 第 4 次重试：8s
- 第 5 次重试：16s
- 第 6 次重试：32s
- 第 7 次重试：60s（达到最大值）

**设计理由**：
- 指数退避避免短时间内大量重试
- 最大延迟 60 秒避免等待时间过长
- 初始延迟 1 秒快速响应临时网络波动

#### 2.2 重试逻辑实现

```go
func runUpdatesWithRetry(ctx context.Context, botAPI *tgbotapi.BotAPI, ...) error {
    config := RetryConfig{
        MaxRetries:    5,
        InitialDelay:  1 * time.Second,
        MaxDelay:      60 * time.Second,
        BackoffFactor: 2.0,
    }

    for attempt := 0; attempt <= config.MaxRetries; attempt++ {
        u := tgbotapi.NewUpdate(0)
        u.Timeout = 60

        updates := botAPI.GetUpdatesChan(u)
        err := processUpdates(ctx, updates, ...)

        if err == nil {
            return nil // 成功，无需重试
        }

        if isFatalError(err) {
            return fmt.Errorf("fatal error, giving up: %w", err)
        }

        if attempt < config.MaxRetries {
            delay := calculateBackoff(attempt, config)
            logger.Warnf("Attempt %d failed, retrying in %v: %v", attempt+1, delay, err)
            time.Sleep(delay)
        }
    }

    return fmt.Errorf("max retries exceeded")
}
```

### 3. 错误类型分类

#### 3.1 临时错误（Transient Errors）

这些错误通常由临时网络问题引起，应该重试：

```go
func isTransientError(err error) bool {
    if err == nil {
        return false
    }
    errStr := err.Error()
    return strings.Contains(errStr, "timeout") ||
           strings.Contains(errStr, "EOF") ||
           strings.Contains(errStr, "connection reset") ||
           strings.Contains(errStr, "broken pipe") ||
           strings.Contains(errStr, "temporary failure")
}
```

**处理策略**：使用指数退避重试

#### 3.2 致命错误（Fatal Errors）

这些错误通常由配置或认证问题引起，不应该重试：

```go
func isFatalError(err error) bool {
    if err == nil {
        return false
    }
    errStr := err.Error()
    return strings.Contains(errStr, "Unauthorized") ||
           strings.Contains(errStr, "401") ||
           strings.Contains(errStr, "403") ||
           strings.Contains(errStr, "404") ||
           strings.Contains(errStr, "conflict") ||
           strings.Contains(errStr, "invalid token")
}
```

**处理策略**：快速失败，记录详细错误日志

### 4. 日志优化

#### 4.1 减少重试日志频率

```go
// 只在特定重试次数时记录日志
if attempt == 0 || attempt == 2 || attempt == 4 || attempt == config.MaxRetries {
    logger.Warnf("Retry attempt %d/%d: %v", attempt+1, config.MaxRetries, err)
}
```

#### 4.2 添加重试统计

```go
type RetryStats struct {
    TotalRetries    int
    SuccessCount    int
    FailureCount    int
    LastRetryTime   time.Time
    AverageDelay    time.Duration
}

func (s *RetryStats) RecordRetry(delay time.Duration, success bool) {
    s.TotalRetries++
    if success {
        s.SuccessCount++
    } else {
        s.FailureCount++
    }
    s.LastRetryTime = time.Now()
    // 计算平均延迟
}
```

#### 4.3 定期健康状态日志

```go
func logHealthStatus(stats *RetryStats) {
    logger.Infof("Bot Health Status - Total: %d, Success: %d, Failure: %d, Success Rate: %.2f%%",
        stats.TotalRetries, stats.SuccessCount, stats.FailureCount,
        float64(stats.SuccessCount)/float64(stats.TotalRetries)*100)
}
```

### 5. 连接健康检查

#### 5.1 健康检查实现

```go
func checkBotHealth(botAPI *tgbotapi.BotAPI) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // 尝试获取 Bot 信息
    _, err := botAPI.GetMe()
    if err != nil {
        return fmt.Errorf("health check failed: %w", err)
    }

    return nil
}

func monitorBotHealth(ctx context.Context, botAPI *tgbotapi.BotAPI, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := checkBotHealth(botAPI); err != nil {
                logger.Errorf("Bot health check failed: %v", err)
                // 可以在这里触发连接重建逻辑
            } else {
                logger.Debug("Bot health check passed")
            }
        }
    }
}
```

#### 5.2 连接重建策略

```go
func reconnectBot(token string, debug bool) (*tgbotapi.BotAPI, error) {
    logger.Info("Attempting to reconnect to Telegram API...")
    newBot, err := bot.NewBotWithCustomClient(token, debug)
    if err != nil {
        return nil, fmt.Errorf("reconnection failed: %w", err)
    }
    logger.Info("Successfully reconnected to Telegram API")
    return newBot, nil
}
```

## 配置参数

### 推荐配置

| 参数 | 当前值 | 推荐值 | 说明 |
|------|--------|--------|------|
| GetUpdatesChan Timeout | 30s | 60s | 长轮询超时时间 |
| HTTP Client Timeout | 120s | 120s | 保持不变 |
| Max Retries | 3 | 5 | 增加重试次数 |
| Initial Retry Delay | 3s | 1s | 减少初始延迟 |
| Max Retry Delay | N/A | 60s | 限制最大延迟 |
| Backoff Factor | N/A | 2.0 | 指数退避因子 |
| Health Check Interval | N/A | 5m | 健康检查间隔 |

## 测试策略

### 单元测试

1. **超时配置测试**：验证 GetUpdatesChan 超时时间为 60 秒
2. **退避算法测试**：验证指数退避计算正确性
3. **错误分类测试**：验证临时错误和致命错误的正确识别
4. **重试逻辑测试**：验证重试次数和延迟的正确性

### 集成测试

1. **网络波动模拟**：使用工具模拟网络延迟和中断
2. **超时场景测试**：模拟 API 响应超时
3. **连接中断测试**：模拟连接意外断开
4. **长时间运行测试**：验证 Bot 稳定性

### 性能测试

1. **日志输出频率**：验证重试日志显著减少
2. **重试成功率**：验证重试策略的有效性
3. **资源使用**：验证内存和 CPU 使用合理

## 风险评估

### 低风险

- 仅修改超时配置和重试逻辑
- 不涉及核心业务逻辑
- 向后兼容

### 潜在问题

1. **超时时间增加**：可能导致消息接收延迟增加
   - **缓解措施**：60 秒是合理范围，影响可控

2. **重试次数增加**：可能导致错误恢复时间延长
   - **缓解措施**：使用指数退避，快速失败致命错误

3. **健康检查开销**：可能增加 API 调用次数
   - **缓解措施**：5 分钟间隔，影响可忽略

## 监控指标

建议添加以下监控指标：

1. `bot_retry_total` - 总重试次数
2. `bot_retry_success` - 重试成功次数
3. `bot_retry_failure` - 重试失败次数
4. `bot_retry_delay_seconds` - 重试延迟分布
5. `bot_health_check_status` - 健康检查状态
6. `bot_connection_errors` - 连接错误次数

## 回滚计划

如果优化后出现问题，可以快速回滚：

1. 将 GetUpdatesChan 超时改回 30 秒
2. 移除指数退避逻辑，恢复固定延迟
3. 简化错误处理逻辑

## 参考资料

- [Telegram Bot API - getUpdates](https://core.telegram.org/bots/api#getupdates)
- [Exponential Backoff and Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)
- [HTTP Client Timeout Best Practices](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/)