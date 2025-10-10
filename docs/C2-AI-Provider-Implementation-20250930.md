# C2: 多AI服务提供商集成 - 详细实现方案

**文档版本**: v1.0
**创建日期**: 2025年9月30日
**阶段**: 第三阶段 (Week 5-7)
**预计工时**: 3天
**状态**: 📋 计划中

---

## 📋 总体目标

实现统一的AI服务提供商抽象层，支持OpenAI GPT和Claude API的无缝切换，确保服务稳定性和成本可控性。

### 核心价值
- ✅ 多Provider支持，避免单点依赖
- ✅ 智能降级机制，确保服务可用性
- ✅ 成本可控，支持按需切换
- ✅ 统一接口，便于扩展新Provider

---

## 🏗️ 核心组件设计

### 1. AI Provider 接口层
**文件**: `pkg/ai/provider.go`

```go
package ai

import (
    "context"
    "time"
)

// AIProvider 定义AI服务提供商的统一接口
type AIProvider interface {
    // ParseReminder 解析自然语言为提醒信息
    ParseReminder(ctx context.Context, text string) (*ParseResult, error)

    // Name 获取提供商名称
    Name() string

    // HealthCheck 健康检查
    HealthCheck(ctx context.Context) error

    // GetConfig 获取配置信息
    GetConfig() ProviderConfig
}

// ParseResult AI解析结果
type ParseResult struct {
    Content     string    `json:"content"`      // 提醒内容
    Time        time.Time `json:"time"`         // 提醒时间
    Pattern     string    `json:"pattern"`      // 重复模式 (daily/weekly/once)
    Confidence  float64   `json:"confidence"`   // 置信度 0-1
    RawResponse string    `json:"raw_response"` // 原始响应
    TokensUsed  int       `json:"tokens_used"`  // Token使用量
}

// ProviderConfig Provider配置
type ProviderConfig struct {
    Name         string        `yaml:"name"`
    Endpoint     string        `yaml:"endpoint"`
    APIKey       string        `yaml:"api_key"`
    Model        string        `yaml:"model"`
    MaxTokens    int           `yaml:"max_tokens"`
    Temperature  float64       `yaml:"temperature"`
    Timeout      time.Duration `yaml:"timeout"`
    RateLimit    int           `yaml:"rate_limit"` // 每分钟请求数
}

// ProviderError 统一错误类型
type ProviderError struct {
    Provider string
    Err      error
    Type     ErrorType
}

type ErrorType int

const (
    ErrorTypeTimeout ErrorType = iota
    ErrorTypeRateLimit
    ErrorTypeInvalidResponse
    ErrorTypeAPIError
    ErrorTypeNetworkError
)

func (e *ProviderError) Error() string {
    return fmt.Sprintf("[%s] %s: %v", e.Provider, e.Type, e.Err)
}
```

**设计要点**:
- 统一的接口定义，支持多种AI服务
- 标准化的解析结果格式
- 详细的错误分类，便于降级决策
- Token使用追踪，支持成本监控

---

### 2. OpenAI Provider 实现
**文件**: `pkg/ai/openai_provider.go`

```go
package ai

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/sashabaranov/go-openai"
)

// OpenAIProvider OpenAI服务提供商实现
type OpenAIProvider struct {
    client *openai.Client
    config ProviderConfig
    limiter *RateLimiter
}

// NewOpenAIProvider 创建OpenAI Provider
func NewOpenAIProvider(config ProviderConfig) (*OpenAIProvider, error) {
    if config.APIKey == "" {
        return nil, fmt.Errorf("OpenAI API key is required")
    }

    client := openai.NewClient(config.APIKey)

    // 设置默认值
    if config.Model == "" {
        config.Model = "gpt-3.5-turbo"
    }
    if config.MaxTokens == 0 {
        config.MaxTokens = 500
    }
    if config.Temperature == 0 {
        config.Temperature = 0.3
    }
    if config.Timeout == 0 {
        config.Timeout = 10 * time.Second
    }
    if config.RateLimit == 0 {
        config.RateLimit = 60 // 默认60 req/min
    }

    return &OpenAIProvider{
        client:  client,
        config:  config,
        limiter: NewRateLimiter(config.RateLimit),
    }, nil
}

// ParseReminder 实现AI解析
func (p *OpenAIProvider) ParseReminder(ctx context.Context, text string) (*ParseResult, error) {
    // 限流检查
    if err := p.limiter.Wait(ctx); err != nil {
        return nil, &ProviderError{
            Provider: p.Name(),
            Err:      err,
            Type:     ErrorTypeRateLimit,
        }
    }

    // 设置超时
    ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
    defer cancel()

    // 构建Prompt
    prompt := p.buildPrompt(text)

    // 调用OpenAI API
    resp, err := p.client.CreateChatCompletion(
        ctx,
        openai.ChatCompletionRequest{
            Model: p.config.Model,
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleSystem,
                    Content: prompt.System,
                },
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: prompt.User,
                },
            },
            MaxTokens:   p.config.MaxTokens,
            Temperature: float32(p.config.Temperature),
        },
    )

    if err != nil {
        return nil, p.handleError(err)
    }

    // 解析响应
    if len(resp.Choices) == 0 {
        return nil, &ProviderError{
            Provider: p.Name(),
            Err:      fmt.Errorf("empty response"),
            Type:     ErrorTypeInvalidResponse,
        }
    }

    result, err := p.parseResponse(resp.Choices[0].Message.Content)
    if err != nil {
        return nil, &ProviderError{
            Provider: p.Name(),
            Err:      err,
            Type:     ErrorTypeInvalidResponse,
        }
    }

    result.RawResponse = resp.Choices[0].Message.Content
    result.TokensUsed = resp.Usage.TotalTokens

    return result, nil
}

// buildPrompt 构建Prompt
func (p *OpenAIProvider) buildPrompt(text string) struct{ System, User string } {
    return struct{ System, User string }{
        System: `你是一个智能提醒助手，负责解析用户的自然语言输入并提取提醒信息。

请严格按照以下JSON格式返回结果：
{
  "content": "提醒的具体内容",
  "time": "2025-10-01T09:00:00Z",
  "pattern": "daily|weekly|monthly|once",
  "confidence": 0.95
}

规则：
1. time必须是ISO 8601格式的完整时间
2. pattern只能是: daily(每天)、weekly(每周)、monthly(每月)、once(一次性)
3. confidence是0-1之间的浮点数，表示解析置信度
4. 如果无法确定时间，返回当前时间+1小时
5. 如果无法确定模式，默认为once`,
        User: fmt.Sprintf("请解析以下提醒信息：\n%s", text),
    }
}

// parseResponse 解析API响应
func (p *OpenAIProvider) parseResponse(content string) (*ParseResult, error) {
    var result ParseResult
    if err := json.Unmarshal([]byte(content), &result); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    // 验证必填字段
    if result.Content == "" {
        return nil, fmt.Errorf("missing content field")
    }
    if result.Time.IsZero() {
        return nil, fmt.Errorf("missing or invalid time field")
    }
    if result.Pattern == "" {
        result.Pattern = "once"
    }
    if result.Confidence == 0 {
        result.Confidence = 0.8
    }

    return &result, nil
}

// handleError 错误处理和分类
func (p *OpenAIProvider) handleError(err error) error {
    // 判断错误类型
    if ctx.Err() == context.DeadlineExceeded {
        return &ProviderError{
            Provider: p.Name(),
            Err:      err,
            Type:     ErrorTypeTimeout,
        }
    }

    // API错误处理
    return &ProviderError{
        Provider: p.Name(),
        Err:      err,
        Type:     ErrorTypeAPIError,
    }
}

// Name 实现接口
func (p *OpenAIProvider) Name() string {
    return "openai"
}

// HealthCheck 实现接口
func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    _, err := p.client.ListModels(ctx)
    return err
}

// GetConfig 实现接口
func (p *OpenAIProvider) GetConfig() ProviderConfig {
    return p.config
}
```

**实现要点**:
- 集成官方SDK `github.com/sashabaranov/go-openai`
- 精心设计的中文Prompt，优化提醒解析准确率
- 完善的错误分类和处理
- 内置限流器，防止超限
- Token使用统计，支持成本追踪

---

### 3. Claude Provider 实现
**文件**: `pkg/ai/claude_provider.go`

```go
package ai

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/anthropics/anthropic-sdk-go"
)

// ClaudeProvider Anthropic Claude服务提供商实现
type ClaudeProvider struct {
    client  *anthropic.Client
    config  ProviderConfig
    limiter *RateLimiter
}

// NewClaudeProvider 创建Claude Provider
func NewClaudeProvider(config ProviderConfig) (*ClaudeProvider, error) {
    if config.APIKey == "" {
        return nil, fmt.Errorf("Claude API key is required")
    }

    client := anthropic.NewClient(
        anthropic.WithAPIKey(config.APIKey),
    )

    // 设置默认值
    if config.Model == "" {
        config.Model = "claude-3-haiku-20240307"
    }
    if config.MaxTokens == 0 {
        config.MaxTokens = 500
    }
    if config.Temperature == 0 {
        config.Temperature = 0.3
    }
    if config.Timeout == 0 {
        config.Timeout = 10 * time.Second
    }
    if config.RateLimit == 0 {
        config.RateLimit = 50 // 默认50 req/min
    }

    return &ClaudeProvider{
        client:  client,
        config:  config,
        limiter: NewRateLimiter(config.RateLimit),
    }, nil
}

// ParseReminder 实现AI解析
func (p *ClaudeProvider) ParseReminder(ctx context.Context, text string) (*ParseResult, error) {
    // 限流检查
    if err := p.limiter.Wait(ctx); err != nil {
        return nil, &ProviderError{
            Provider: p.Name(),
            Err:      err,
            Type:     ErrorTypeRateLimit,
        }
    }

    // 设置超时
    ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
    defer cancel()

    // 构建Prompt (与OpenAI相同的System Prompt)
    prompt := p.buildPrompt(text)

    // 调用Claude API
    resp, err := p.client.Messages.Create(ctx, anthropic.MessageCreateParams{
        Model:     p.config.Model,
        MaxTokens: p.config.MaxTokens,
        System:    prompt.System,
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock(prompt.User)),
        },
        Temperature: p.config.Temperature,
    })

    if err != nil {
        return nil, p.handleError(err)
    }

    // 解析响应
    if len(resp.Content) == 0 {
        return nil, &ProviderError{
            Provider: p.Name(),
            Err:      fmt.Errorf("empty response"),
            Type:     ErrorTypeInvalidResponse,
        }
    }

    content := resp.Content[0].Text
    result, err := p.parseResponse(content)
    if err != nil {
        return nil, &ProviderError{
            Provider: p.Name(),
            Err:      err,
            Type:     ErrorTypeInvalidResponse,
        }
    }

    result.RawResponse = content
    result.TokensUsed = resp.Usage.InputTokens + resp.Usage.OutputTokens

    return result, nil
}

// buildPrompt 构建Prompt (与OpenAI一致)
func (p *ClaudeProvider) buildPrompt(text string) struct{ System, User string } {
    return struct{ System, User string }{
        System: `你是一个智能提醒助手，负责解析用户的自然语言输入并提取提醒信息。

请严格按照以下JSON格式返回结果：
{
  "content": "提醒的具体内容",
  "time": "2025-10-01T09:00:00Z",
  "pattern": "daily|weekly|monthly|once",
  "confidence": 0.95
}

规则：
1. time必须是ISO 8601格式的完整时间
2. pattern只能是: daily(每天)、weekly(每周)、monthly(每月)、once(一次性)
3. confidence是0-1之间的浮点数，表示解析置信度
4. 如果无法确定时间，返回当前时间+1小时
5. 如果无法确定模式，默认为once`,
        User: fmt.Sprintf("请解析以下提醒信息：\n%s", text),
    }
}

// parseResponse 解析API响应 (与OpenAI一致)
func (p *ClaudeProvider) parseResponse(content string) (*ParseResult, error) {
    var result ParseResult
    if err := json.Unmarshal([]byte(content), &result); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    // 验证必填字段
    if result.Content == "" {
        return nil, fmt.Errorf("missing content field")
    }
    if result.Time.IsZero() {
        return nil, fmt.Errorf("missing or invalid time field")
    }
    if result.Pattern == "" {
        result.Pattern = "once"
    }
    if result.Confidence == 0 {
        result.Confidence = 0.8
    }

    return &result, nil
}

// handleError 错误处理
func (p *ClaudeProvider) handleError(err error) error {
    if ctx.Err() == context.DeadlineExceeded {
        return &ProviderError{
            Provider: p.Name(),
            Err:      err,
            Type:     ErrorTypeTimeout,
        }
    }

    return &ProviderError{
        Provider: p.Name(),
        Err:      err,
        Type:     ErrorTypeAPIError,
    }
}

// Name 实现接口
func (p *ClaudeProvider) Name() string {
    return "claude"
}

// HealthCheck 实现接口
func (p *ClaudeProvider) HealthCheck(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // 发送简单测试请求
    _, err := p.client.Messages.Create(ctx, anthropic.MessageCreateParams{
        Model:     p.config.Model,
        MaxTokens: 10,
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock("test")),
        },
    })

    return err
}

// GetConfig 实现接口
func (p *ClaudeProvider) GetConfig() ProviderConfig {
    return p.config
}
```

**实现要点**:
- 集成官方SDK `github.com/anthropics/anthropic-sdk-go`
- 与OpenAI完全一致的Prompt设计，确保输出格式统一
- 相同的错误分类机制
- 作为OpenAI的可靠备选方案

---

### 4. Provider Manager (核心调度器)
**文件**: `pkg/ai/manager.go`

```go
package ai

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/sirupsen/logrus"
)

// ProviderManager 管理多个AI Provider
type ProviderManager struct {
    providers map[string]AIProvider
    primary   string
    fallback  []string
    metrics   *ProviderMetrics
    cache     *Cache
    breakers  map[string]*CircuitBreaker
    mu        sync.RWMutex
    logger    *logrus.Logger
}

// NewProviderManager 创建Provider管理器
func NewProviderManager(
    providers map[string]AIProvider,
    primary string,
    fallback []string,
    logger *logrus.Logger,
) *ProviderManager {
    breakers := make(map[string]*CircuitBreaker)
    for name := range providers {
        breakers[name] = NewCircuitBreaker(5, 2, 30*time.Second)
    }

    return &ProviderManager{
        providers: providers,
        primary:   primary,
        fallback:  fallback,
        metrics:   NewProviderMetrics(),
        cache:     NewCache(5*time.Minute, 1000),
        breakers:  breakers,
        logger:    logger,
    }
}

// ParseWithFallback 执行解析并自动降级
func (m *ProviderManager) ParseWithFallback(ctx context.Context, text string) (*ParseResult, error) {
    // 1. 检查缓存
    if result := m.cache.Get(text); result != nil {
        m.logger.WithField("cache", "hit").Info("Using cached result")
        m.metrics.RecordCacheHit(m.primary)
        return result, nil
    }
    m.metrics.RecordCacheMiss(m.primary)

    // 2. 尝试主Provider
    provider := m.selectProvider(m.primary)
    if provider != nil {
        result, err := m.tryProvider(ctx, provider, text)
        if err == nil {
            m.cache.Set(text, result)
            return result, nil
        }
        m.logger.WithError(err).WithField("provider", m.primary).Warn("Primary provider failed")
    }

    // 3. 依次尝试备选Provider
    for _, name := range m.fallback {
        provider := m.selectProvider(name)
        if provider == nil {
            continue
        }

        m.logger.WithField("provider", name).Info("Trying fallback provider")
        result, err := m.tryProvider(ctx, provider, text)
        if err == nil {
            m.cache.Set(text, result)
            return result, nil
        }
        m.logger.WithError(err).WithField("provider", name).Warn("Fallback provider failed")
    }

    // 4. 所有Provider都失败
    return nil, fmt.Errorf("all providers failed")
}

// tryProvider 尝试使用指定Provider
func (m *ProviderManager) tryProvider(ctx context.Context, provider AIProvider, text string) (*ParseResult, error) {
    name := provider.Name()
    breaker := m.breakers[name]

    // 检查熔断器状态
    if !breaker.CanRequest() {
        m.logger.WithField("provider", name).Warn("Circuit breaker is open")
        return nil, fmt.Errorf("circuit breaker open for %s", name)
    }

    // 记录开始时间
    start := time.Now()

    // 执行解析
    result, err := provider.ParseReminder(ctx, text)

    // 记录指标
    duration := time.Since(start)
    m.metrics.RecordRequest(name, err == nil, duration)

    if err != nil {
        breaker.RecordFailure()
        m.logger.WithError(err).
            WithField("provider", name).
            WithField("duration", duration).
            Error("Provider request failed")
        return nil, err
    }

    breaker.RecordSuccess()
    m.logger.WithField("provider", name).
        WithField("duration", duration).
        WithField("tokens", result.TokensUsed).
        WithField("confidence", result.Confidence).
        Info("Provider request succeeded")

    return result, nil
}

// selectProvider 选择Provider
func (m *ProviderManager) selectProvider(name string) AIProvider {
    m.mu.RLock()
    defer m.mu.RUnlock()

    provider, exists := m.providers[name]
    if !exists {
        m.logger.WithField("provider", name).Warn("Provider not found")
        return nil
    }

    return provider
}

// GetMetrics 获取指标
func (m *ProviderManager) GetMetrics() *ProviderMetrics {
    return m.metrics
}

// HealthCheck 健康检查所有Provider
func (m *ProviderManager) HealthCheck(ctx context.Context) map[string]error {
    m.mu.RLock()
    defer m.mu.RUnlock()

    results := make(map[string]error)
    for name, provider := range m.providers {
        results[name] = provider.HealthCheck(ctx)
    }

    return results
}
```

**核心功能**:
- 智能Provider选择和降级
- 缓存机制，减少重复API调用
- 熔断器保护，防止级联失败
- 完整的指标收集
- 详细的日志记录

---

### 5. 限流器实现
**文件**: `pkg/ai/ratelimiter.go`

```go
package ai

import (
    "context"
    "time"

    "golang.org/x/time/rate"
)

// RateLimiter 限流器 (Token Bucket算法)
type RateLimiter struct {
    limiter *rate.Limiter
}

// NewRateLimiter 创建限流器
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
    // 转换为每秒速率
    rps := float64(requestsPerMinute) / 60.0

    return &RateLimiter{
        limiter: rate.NewLimiter(rate.Limit(rps), requestsPerMinute),
    }
}

// Wait 等待令牌可用
func (r *RateLimiter) Wait(ctx context.Context) error {
    return r.limiter.Wait(ctx)
}

// Allow 检查是否允许请求
func (r *RateLimiter) Allow() bool {
    return r.limiter.Allow()
}
```

---

### 6. 熔断器实现
**文件**: `pkg/ai/circuit_breaker.go`

```go
package ai

import (
    "sync"
    "time"
)

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
    StateClosed CircuitBreakerState = iota
    StateOpen
    StateHalfOpen
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
    failureThreshold int
    successThreshold int
    timeout          time.Duration

    state            CircuitBreakerState
    failures         int
    successes        int
    lastFailureTime  time.Time
    mu               sync.RWMutex
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        failureThreshold: failureThreshold,
        successThreshold: successThreshold,
        timeout:          timeout,
        state:            StateClosed,
    }
}

// CanRequest 是否允许请求
func (cb *CircuitBreaker) CanRequest() bool {
    cb.mu.RLock()
    defer cb.mu.RUnlock()

    // 如果是开启状态，检查是否超时
    if cb.state == StateOpen {
        if time.Since(cb.lastFailureTime) > cb.timeout {
            // 超时后进入半开状态
            cb.mu.RUnlock()
            cb.mu.Lock()
            cb.state = StateHalfOpen
            cb.successes = 0
            cb.mu.Unlock()
            cb.mu.RLock()
            return true
        }
        return false
    }

    return true
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures = 0

    if cb.state == StateHalfOpen {
        cb.successes++
        if cb.successes >= cb.successThreshold {
            cb.state = StateClosed
        }
    }
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures++
    cb.lastFailureTime = time.Now()

    if cb.failures >= cb.failureThreshold {
        cb.state = StateOpen
    }
}

// GetState 获取状态
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
    cb.mu.RLock()
    defer cb.mu.RUnlock()
    return cb.state
}
```

---

### 7. 缓存实现
**文件**: `pkg/ai/cache.go`

```go
package ai

import (
    "sync"
    "time"
)

// Cache 简单的内存缓存
type Cache struct {
    items   map[string]*cacheItem
    ttl     time.Duration
    maxSize int
    mu      sync.RWMutex
}

type cacheItem struct {
    value      *ParseResult
    expiration time.Time
}

// NewCache 创建缓存
func NewCache(ttl time.Duration, maxSize int) *Cache {
    cache := &Cache{
        items:   make(map[string]*cacheItem),
        ttl:     ttl,
        maxSize: maxSize,
    }

    // 启动清理协程
    go cache.cleanup()

    return cache
}

// Get 获取缓存
func (c *Cache) Get(key string) *ParseResult {
    c.mu.RLock()
    defer c.mu.RUnlock()

    item, exists := c.items[key]
    if !exists {
        return nil
    }

    // 检查是否过期
    if time.Now().After(item.expiration) {
        return nil
    }

    return item.value
}

// Set 设置缓存
func (c *Cache) Set(key string, value *ParseResult) {
    c.mu.Lock()
    defer c.mu.Unlock()

    // 检查大小限制
    if len(c.items) >= c.maxSize {
        // 简单的LRU: 删除一个过期项
        for k, item := range c.items {
            if time.Now().After(item.expiration) {
                delete(c.items, k)
                break
            }
        }
    }

    c.items[key] = &cacheItem{
        value:      value,
        expiration: time.Now().Add(c.ttl),
    }
}

// cleanup 定期清理过期项
func (c *Cache) cleanup() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        c.mu.Lock()
        now := time.Now()
        for key, item := range c.items {
            if now.After(item.expiration) {
                delete(c.items, key)
            }
        }
        c.mu.Unlock()
    }
}
```

---

### 8. 监控指标
**文件**: `pkg/ai/metrics.go`

```go
package ai

import (
    "sync"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    aiParseRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ai_parse_requests_total",
            Help: "Total number of AI parse requests",
        },
        []string{"provider", "status"},
    )

    aiParseDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "ai_parse_duration_seconds",
            Help:    "Duration of AI parse requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"provider"},
    )

    aiTokensUsed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ai_parse_tokens_used",
            Help: "Total tokens used by AI providers",
        },
        []string{"provider"},
    )

    aiCacheHitRatio = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "ai_cache_hit_ratio",
            Help: "Cache hit ratio for AI parsing",
        },
        []string{"provider"},
    )

    aiCircuitBreakerState = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "ai_circuit_breaker_state",
            Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
        },
        []string{"provider"},
    )

    aiRateLimitExceeded = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ai_rate_limit_exceeded_total",
            Help: "Total number of rate limit exceeded events",
        },
        []string{"provider"},
    )
)

// ProviderMetrics Provider指标
type ProviderMetrics struct {
    cacheHits   map[string]int64
    cacheMisses map[string]int64
    mu          sync.RWMutex
}

// NewProviderMetrics 创建指标
func NewProviderMetrics() *ProviderMetrics {
    return &ProviderMetrics{
        cacheHits:   make(map[string]int64),
        cacheMisses: make(map[string]int64),
    }
}

// RecordRequest 记录请求
func (m *ProviderMetrics) RecordRequest(provider string, success bool, duration time.Duration) {
    status := "success"
    if !success {
        status = "failure"
    }

    aiParseRequestsTotal.WithLabelValues(provider, status).Inc()
    aiParseDuration.WithLabelValues(provider).Observe(duration.Seconds())
}

// RecordTokens 记录Token使用
func (m *ProviderMetrics) RecordTokens(provider string, tokens int) {
    aiTokensUsed.WithLabelValues(provider).Add(float64(tokens))
}

// RecordCacheHit 记录缓存命中
func (m *ProviderMetrics) RecordCacheHit(provider string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.cacheHits[provider]++
    m.updateCacheHitRatio(provider)
}

// RecordCacheMiss 记录缓存未命中
func (m *ProviderMetrics) RecordCacheMiss(provider string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.cacheMisses[provider]++
    m.updateCacheHitRatio(provider)
}

// updateCacheHitRatio 更新缓存命中率
func (m *ProviderMetrics) updateCacheHitRatio(provider string) {
    hits := m.cacheHits[provider]
    misses := m.cacheMisses[provider]
    total := hits + misses

    if total > 0 {
        ratio := float64(hits) / float64(total)
        aiCacheHitRatio.WithLabelValues(provider).Set(ratio)
    }
}

// RecordCircuitBreakerState 记录熔断器状态
func RecordCircuitBreakerState(provider string, state CircuitBreakerState) {
    aiCircuitBreakerState.WithLabelValues(provider).Set(float64(state))
}

// RecordRateLimitExceeded 记录限流事件
func RecordRateLimitExceeded(provider string) {
    aiRateLimitExceeded.WithLabelValues(provider).Inc()
}
```

---

## 📝 配置文件扩展

### 完整配置示例
**文件**: `configs/config.full.yaml`

```yaml
ai:
  # AI功能总开关
  enabled: true

  # 主要Provider
  primary_provider: "openai"

  # 备选Provider列表 (按优先级排序)
  fallback_providers:
    - "claude"
    - "local"  # 最终降级到本地正则解析

  # Provider配置
  providers:
    openai:
      api_key: "${OPENAI_API_KEY}"
      endpoint: "https://api.openai.com/v1"
      model: "gpt-3.5-turbo"
      max_tokens: 500
      temperature: 0.3
      timeout: 10s
      rate_limit: 60  # 每分钟请求数

    claude:
      api_key: "${ANTHROPIC_API_KEY}"
      endpoint: "https://api.anthropic.com"
      model: "claude-3-haiku-20240307"
      max_tokens: 500
      temperature: 0.3
      timeout: 10s
      rate_limit: 50

  # 缓存配置
  cache:
    enabled: true
    ttl: 5m
    max_size: 1000

  # 熔断器配置
  circuit_breaker:
    failure_threshold: 5    # 连续失败5次后熔断
    success_threshold: 2    # 半开状态下连续成功2次恢复
    timeout: 30s            # 熔断后30秒尝试恢复
```

### 环境变量配置
**文件**: `configs/.env.example`

```bash
# AI Provider API Keys
OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
ANTHROPIC_API_KEY=sk-ant-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# 可选：覆盖配置文件中的设置
MMEMORY_AI_ENABLED=true
MMEMORY_AI_PRIMARY_PROVIDER=openai
```

---

## 🧪 测试用例

### 1. OpenAI Provider测试
**文件**: `pkg/ai/openai_provider_test.go`

```go
package ai

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestOpenAIProvider_ParseReminder(t *testing.T) {
    // 跳过测试如果没有API Key
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }

    config := ProviderConfig{
        APIKey:      apiKey,
        Model:       "gpt-3.5-turbo",
        MaxTokens:   500,
        Temperature: 0.3,
        Timeout:     10 * time.Second,
        RateLimit:   60,
    }

    provider, err := NewOpenAIProvider(config)
    require.NoError(t, err)
    require.NotNil(t, provider)

    tests := []struct {
        name          string
        input         string
        expectContent string
        expectPattern string
    }{
        {
            name:          "每天提醒",
            input:         "每天早上9点提醒我喝水",
            expectContent: "喝水",
            expectPattern: "daily",
        },
        {
            name:          "每周提醒",
            input:         "每周一下午3点提醒我开会",
            expectContent: "开会",
            expectPattern: "weekly",
        },
        {
            name:          "一次性提醒",
            input:         "明天下午2点提醒我取快递",
            expectContent: "取快递",
            expectPattern: "once",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            result, err := provider.ParseReminder(ctx, tt.input)

            require.NoError(t, err)
            require.NotNil(t, result)

            assert.Contains(t, result.Content, tt.expectContent)
            assert.Equal(t, tt.expectPattern, result.Pattern)
            assert.Greater(t, result.Confidence, 0.5)
            assert.Greater(t, result.TokensUsed, 0)
            assert.NotEmpty(t, result.RawResponse)
        })
    }
}

func TestOpenAIProvider_RateLimit(t *testing.T) {
    config := ProviderConfig{
        APIKey:    "test-key",
        RateLimit: 2, // 2 req/min = 1 req/30s
    }

    provider, err := NewOpenAIProvider(config)
    require.NoError(t, err)

    ctx := context.Background()

    // 前两个请求应该立即通过
    start := time.Now()
    _, _ = provider.ParseReminder(ctx, "test1")
    _, _ = provider.ParseReminder(ctx, "test2")
    elapsed := time.Since(start)

    assert.Less(t, elapsed, 1*time.Second)

    // 第三个请求应该被限流
    start = time.Now()
    _, _ = provider.ParseReminder(ctx, "test3")
    elapsed = time.Since(start)

    assert.Greater(t, elapsed, 29*time.Second)
}

func TestOpenAIProvider_HealthCheck(t *testing.T) {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }

    config := ProviderConfig{
        APIKey: apiKey,
    }

    provider, err := NewOpenAIProvider(config)
    require.NoError(t, err)

    ctx := context.Background()
    err = provider.HealthCheck(ctx)
    assert.NoError(t, err)
}
```

### 2. Claude Provider测试
**文件**: `pkg/ai/claude_provider_test.go`

```go
package ai

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestClaudeProvider_ParseReminder(t *testing.T) {
    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    if apiKey == "" {
        t.Skip("ANTHROPIC_API_KEY not set")
    }

    config := ProviderConfig{
        APIKey:      apiKey,
        Model:       "claude-3-haiku-20240307",
        MaxTokens:   500,
        Temperature: 0.3,
        Timeout:     10 * time.Second,
        RateLimit:   50,
    }

    provider, err := NewClaudeProvider(config)
    require.NoError(t, err)

    ctx := context.Background()
    result, err := provider.ParseReminder(ctx, "每天早上8点提醒我吃早餐")

    require.NoError(t, err)
    assert.Contains(t, result.Content, "早餐")
    assert.Equal(t, "daily", result.Pattern)
    assert.Greater(t, result.Confidence, 0.5)
}
```

### 3. Provider Manager测试
**文件**: `pkg/ai/manager_test.go`

```go
package ai

import (
    "context"
    "errors"
    "testing"
    "time"

    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// MockProvider 模拟Provider
type MockProvider struct {
    name      string
    shouldFail bool
    delay      time.Duration
}

func (m *MockProvider) ParseReminder(ctx context.Context, text string) (*ParseResult, error) {
    if m.delay > 0 {
        time.Sleep(m.delay)
    }

    if m.shouldFail {
        return nil, errors.New("mock error")
    }

    return &ParseResult{
        Content:    "test reminder",
        Time:       time.Now(),
        Pattern:    "daily",
        Confidence: 0.9,
        TokensUsed: 100,
    }, nil
}

func (m *MockProvider) Name() string {
    return m.name
}

func (m *MockProvider) HealthCheck(ctx context.Context) error {
    if m.shouldFail {
        return errors.New("unhealthy")
    }
    return nil
}

func (m *MockProvider) GetConfig() ProviderConfig {
    return ProviderConfig{Name: m.name}
}

func TestProviderManager_ParseWithFallback(t *testing.T) {
    logger := logrus.New()
    logger.SetOutput(io.Discard)

    primaryProvider := &MockProvider{name: "primary", shouldFail: true}
    fallbackProvider := &MockProvider{name: "fallback", shouldFail: false}

    providers := map[string]AIProvider{
        "primary":  primaryProvider,
        "fallback": fallbackProvider,
    }

    manager := NewProviderManager(
        providers,
        "primary",
        []string{"fallback"},
        logger,
    )

    ctx := context.Background()
    result, err := manager.ParseWithFallback(ctx, "test message")

    require.NoError(t, err)
    assert.Equal(t, "test reminder", result.Content)
}

func TestProviderManager_Cache(t *testing.T) {
    logger := logrus.New()
    logger.SetOutput(io.Discard)

    provider := &MockProvider{name: "test", shouldFail: false, delay: 100 * time.Millisecond}

    providers := map[string]AIProvider{
        "test": provider,
    }

    manager := NewProviderManager(providers, "test", []string{}, logger)

    ctx := context.Background()
    text := "cache test message"

    // 第一次调用 - 应该慢
    start := time.Now()
    result1, err := manager.ParseWithFallback(ctx, text)
    duration1 := time.Since(start)

    require.NoError(t, err)
    assert.Greater(t, duration1, 100*time.Millisecond)

    // 第二次调用 - 应该从缓存读取，很快
    start = time.Now()
    result2, err := manager.ParseWithFallback(ctx, text)
    duration2 := time.Since(start)

    require.NoError(t, err)
    assert.Less(t, duration2, 10*time.Millisecond)
    assert.Equal(t, result1.Content, result2.Content)

    // 验证缓存命中
    metrics := manager.GetMetrics()
    assert.NotNil(t, metrics)
}

func TestProviderManager_CircuitBreaker(t *testing.T) {
    logger := logrus.New()
    logger.SetOutput(io.Discard)

    provider := &MockProvider{name: "test", shouldFail: true}

    providers := map[string]AIProvider{
        "test": provider,
    }

    manager := NewProviderManager(providers, "test", []string{}, logger)

    ctx := context.Background()

    // 连续失败5次，触发熔断
    for i := 0; i < 5; i++ {
        _, _ = manager.ParseWithFallback(ctx, "test")
    }

    // 熔断后的请求应该立即失败
    _, err := manager.ParseWithFallback(ctx, "test")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "circuit breaker")
}

func TestProviderManager_HealthCheck(t *testing.T) {
    logger := logrus.New()

    providers := map[string]AIProvider{
        "healthy":   &MockProvider{name: "healthy", shouldFail: false},
        "unhealthy": &MockProvider{name: "unhealthy", shouldFail: true},
    }

    manager := NewProviderManager(providers, "healthy", []string{}, logger)

    ctx := context.Background()
    results := manager.HealthCheck(ctx)

    assert.NoError(t, results["healthy"])
    assert.Error(t, results["unhealthy"])
}
```

### 4. 熔断器测试
**文件**: `pkg/ai/circuit_breaker_test.go`

```go
package ai

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
    cb := NewCircuitBreaker(3, 2, 1*time.Second)

    // 初始状态：关闭
    assert.Equal(t, StateClosed, cb.GetState())
    assert.True(t, cb.CanRequest())

    // 连续3次失败：开启
    cb.RecordFailure()
    cb.RecordFailure()
    cb.RecordFailure()
    assert.Equal(t, StateOpen, cb.GetState())
    assert.False(t, cb.CanRequest())

    // 等待超时：半开
    time.Sleep(1100 * time.Millisecond)
    assert.True(t, cb.CanRequest())
    assert.Equal(t, StateHalfOpen, cb.GetState())

    // 半开状态下连续2次成功：关闭
    cb.RecordSuccess()
    cb.RecordSuccess()
    assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreaker_ResetOnSuccess(t *testing.T) {
    cb := NewCircuitBreaker(3, 2, 1*time.Second)

    // 2次失败 + 1次成功 = 重置计数
    cb.RecordFailure()
    cb.RecordFailure()
    cb.RecordSuccess()

    assert.Equal(t, StateClosed, cb.GetState())

    // 再失败2次不应触发熔断
    cb.RecordFailure()
    cb.RecordFailure()
    assert.Equal(t, StateClosed, cb.GetState())
}
```

---

## 🔄 集成到现有系统

### Parser Service集成
**文件**: `internal/service/parser.go`

```go
package service

import (
    "context"
    "fmt"
    "time"

    "github.com/yourusername/mmemory/internal/models"
    "github.com/yourusername/mmemory/pkg/ai"
    "github.com/sirupsen/logrus"
)

type ParserService struct {
    aiManager      *ai.ProviderManager
    fallbackParser *LocalParser
    useAI          bool
    logger         *logrus.Logger
}

func NewParserService(
    aiManager *ai.ProviderManager,
    fallbackParser *LocalParser,
    useAI bool,
    logger *logrus.Logger,
) *ParserService {
    return &ParserService{
        aiManager:      aiManager,
        fallbackParser: fallbackParser,
        useAI:          useAI,
        logger:         logger,
    }
}

// ParseMessage 解析用户消息为提醒
func (p *ParserService) ParseMessage(ctx context.Context, text string) (*models.Reminder, error) {
    // 1. 尝试AI解析
    if p.useAI && p.aiManager != nil {
        p.logger.Info("Attempting AI parsing")
        result, err := p.aiManager.ParseWithFallback(ctx, text)

        if err == nil && result.Confidence > 0.7 {
            p.logger.WithFields(logrus.Fields{
                "confidence": result.Confidence,
                "pattern":    result.Pattern,
            }).Info("AI parsing succeeded")

            return p.convertToReminder(result), nil
        }

        if err != nil {
            p.logger.WithError(err).Warn("AI parsing failed, falling back to local parser")
        } else {
            p.logger.WithField("confidence", result.Confidence).
                Warn("AI parsing confidence too low, falling back to local parser")
        }
    }

    // 2. 降级到本地正则解析
    p.logger.Info("Using local regex parser")
    return p.fallbackParser.Parse(text)
}

// convertToReminder 转换AI结果为Reminder模型
func (p *ParserService) convertToReminder(result *ai.ParseResult) *models.Reminder {
    reminder := &models.Reminder{
        Content:    result.Content,
        RemindTime: result.Time,
        Status:     models.ReminderStatusActive,
        CreatedAt:  time.Now(),
        UpdatedAt:  time.Now(),
    }

    // 转换Pattern
    switch result.Pattern {
    case "daily":
        reminder.Pattern = "daily"
    case "weekly":
        reminder.Pattern = fmt.Sprintf("weekly:%d", result.Time.Weekday())
    case "monthly":
        reminder.Pattern = fmt.Sprintf("monthly:%d", result.Time.Day())
    case "once":
        reminder.Pattern = fmt.Sprintf("once:%s", result.Time.Format("2006-01-02"))
    default:
        reminder.Pattern = "once"
    }

    return reminder
}
```

### 主程序初始化
**文件**: `cmd/bot/main.go` (添加AI初始化逻辑)

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/yourusername/mmemory/internal/service"
    "github.com/yourusername/mmemory/pkg/ai"
    "github.com/yourusername/mmemory/pkg/config"
    "github.com/sirupsen/logrus"
)

func main() {
    // 加载配置
    cfg, err := config.Load("configs/config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    logger := logrus.New()

    // 初始化AI Providers
    var aiManager *ai.ProviderManager
    if cfg.AI.Enabled {
        aiManager, err = initAIProviders(cfg, logger)
        if err != nil {
            logger.WithError(err).Warn("Failed to initialize AI providers, AI features disabled")
        } else {
            logger.Info("AI providers initialized successfully")

            // 健康检查
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            defer cancel()

            healthResults := aiManager.HealthCheck(ctx)
            for name, err := range healthResults {
                if err != nil {
                    logger.WithError(err).WithField("provider", name).Warn("Provider health check failed")
                } else {
                    logger.WithField("provider", name).Info("Provider healthy")
                }
            }
        }
    }

    // 初始化Parser Service
    parserService := service.NewParserService(
        aiManager,
        service.NewLocalParser(),
        cfg.AI.Enabled,
        logger,
    )

    // ... 其他服务初始化 ...
}

func initAIProviders(cfg *config.Config, logger *logrus.Logger) (*ai.ProviderManager, error) {
    providers := make(map[string]ai.AIProvider)

    // 初始化OpenAI
    if openaiCfg, ok := cfg.AI.Providers["openai"]; ok {
        provider, err := ai.NewOpenAIProvider(openaiCfg)
        if err != nil {
            logger.WithError(err).Warn("Failed to initialize OpenAI provider")
        } else {
            providers["openai"] = provider
            logger.Info("OpenAI provider initialized")
        }
    }

    // 初始化Claude
    if claudeCfg, ok := cfg.AI.Providers["claude"]; ok {
        provider, err := ai.NewClaudeProvider(claudeCfg)
        if err != nil {
            logger.WithError(err).Warn("Failed to initialize Claude provider")
        } else {
            providers["claude"] = provider
            logger.Info("Claude provider initialized")
        }
    }

    if len(providers) == 0 {
        return nil, fmt.Errorf("no AI providers available")
    }

    // 创建Provider Manager
    manager := ai.NewProviderManager(
        providers,
        cfg.AI.PrimaryProvider,
        cfg.AI.FallbackProviders,
        logger,
    )

    return manager, nil
}
```

---

## 📦 依赖包清单

### Go模块依赖
```bash
# OpenAI SDK
go get github.com/sashabaranov/go-openai@latest

# Anthropic Claude SDK
go get github.com/anthropics/anthropic-sdk-go@latest

# 限流器
go get golang.org/x/time/rate@latest

# 缓存 (可选，可用内置实现)
go get github.com/patrickmn/go-cache@latest

# 测试框架
go get github.com/stretchr/testify@latest

# Prometheus监控
go get github.com/prometheus/client_golang/prometheus@latest
```

### 更新go.mod
```bash
cd /Users/chenweilong/www/MMemory
go mod tidy
```

---

## 🎯 验收标准

### 功能验收
- [ ] ✅ OpenAI API调用成功，解析准确率 > 85%
- [ ] ✅ Claude API调用成功，作为有效备选方案
- [ ] ✅ 限流器生效，超限时自动等待或切换Provider
- [ ] ✅ 熔断器触发，连续5次失败后切换到备选Provider
- [ ] ✅ 缓存命中率 > 30%，有效减少API调用
- [ ] ✅ 降级机制完善，AI失败后自动使用本地解析
- [ ] ✅ 健康检查正常，能检测Provider可用性

### 性能验收
- [ ] ✅ AI解析响应时间 < 2秒 (P95)
- [ ] ✅ 缓存命中响应时间 < 50ms
- [ ] ✅ 并发100用户无性能劣化
- [ ] ✅ 限流器对性能影响 < 10ms

### 测试验收
- [ ] ✅ 单元测试覆盖率 > 80%
- [ ] ✅ 所有Provider测试通过
- [ ] ✅ 降级机制测试通过
- [ ] ✅ 熔断器状态转换测试通过
- [ ] ✅ 集成测试全流程通过

### 监控验收
- [ ] ✅ Prometheus指标正常采集
- [ ] ✅ Grafana面板显示正常
- [ ] ✅ Token使用统计准确
- [ ] ✅ 错误率告警配置完成

### 成本验收
- [ ] ✅ Token使用可追踪
- [ ] ✅ 每日成本可预估
- [ ] ✅ 超预算告警生效
- [ ] ✅ 缓存有效降低成本

---

## ⏱️ 开发时间分解

### Day 1: 核心接口 + OpenAI集成 (6小时)
- **上午** (3小时)
  - [ ] 创建 `pkg/ai/provider.go` - 接口定义
  - [ ] 创建 `pkg/ai/openai_provider.go` - OpenAI实现
  - [ ] 基础单元测试

- **下午** (3小时)
  - [ ] Prompt工程优化
  - [ ] 错误处理完善
  - [ ] 集成测试

### Day 2: Claude集成 + Provider Manager (6小时)
- **上午** (3小时)
  - [ ] 创建 `pkg/ai/claude_provider.go` - Claude实现
  - [ ] 创建 `pkg/ai/manager.go` - Provider管理器
  - [ ] 降级逻辑实现

- **下午** (3小时)
  - [ ] Manager测试
  - [ ] 多Provider联调
  - [ ] 错误场景测试

### Day 3: 限流/熔断/缓存 (6小时)
- **上午** (3小时)
  - [ ] 创建 `pkg/ai/ratelimiter.go` - 限流器
  - [ ] 创建 `pkg/ai/circuit_breaker.go` - 熔断器
  - [ ] 创建 `pkg/ai/cache.go` - 缓存

- **下午** (3小时)
  - [ ] 限流/熔断/缓存测试
  - [ ] 性能压测
  - [ ] 边界条件测试

### Day 4: 测试 + 监控 (6小时)
- **上午** (3小时)
  - [ ] 创建 `pkg/ai/metrics.go` - Prometheus指标
  - [ ] 完善所有单元测试
  - [ ] 集成测试覆盖

- **下午** (3小时)
  - [ ] 端到端测试
  - [ ] 性能基准测试
  - [ ] 测试报告生成

### Day 5: 集成 + 文档 (6小时)
- **上午** (3小时)
  - [ ] 集成到 `internal/service/parser.go`
  - [ ] 修改 `cmd/bot/main.go` 初始化逻辑
  - [ ] 全链路联调测试

- **下午** (3小时)
  - [ ] 更新配置文件和文档
  - [ ] 部署测试环境验证
  - [ ] 更新 `next-plan-20250928.md` 标记完成

---

## 📊 成本估算

### API调用成本 (基于1000次/天)

**OpenAI GPT-3.5-turbo**
- 输入: ~200 tokens × 1000 = 200K tokens
- 输出: ~150 tokens × 1000 = 150K tokens
- 成本: $0.5/M input + $1.5/M output = $0.325/天
- 月成本: ~$10

**Claude Haiku**
- 输入: ~200 tokens × 200 = 40K tokens (20%降级)
- 输出: ~150 tokens × 200 = 30K tokens
- 成本: $0.25/M input + $1.25/M output = $0.0475/天
- 月成本: ~$1.5

**预计月总成本**: $11.5 (基于1000次/天请求量)

### 成本优化措施
1. ✅ 缓存减少30%重复调用 → 节省 $3.5/月
2. ✅ 智能降级到本地解析 → 节省 $2/月
3. ✅ 预期实际成本: **$6/月**

---

## 🔒 安全考虑

### API密钥管理
- ✅ 使用环境变量存储API Key
- ✅ 禁止将密钥提交到代码仓库
- ✅ 生产环境使用密钥管理服务 (如AWS Secrets Manager)

### 数据隐私
- ✅ 用户消息仅用于解析，不存储在AI服务商
- ✅ 敏感信息过滤 (如手机号、身份证号)
- ✅ 遵守GDPR和国内数据隐私法规

### 限流保护
- ✅ 防止API滥用和成本失控
- ✅ 用户级别限流 (如10次/分钟)
- ✅ 异常检测和告警

---

## 📈 监控和告警

### 关键指标
- **可用性**: AI解析成功率 > 90%
- **性能**: P95响应时间 < 2秒
- **成本**: 每日Token使用量 < 预算
- **降级**: 降级率 < 10%

### 告警规则
```yaml
groups:
  - name: ai_alerts
    rules:
      - alert: AIParseFailureRateHigh
        expr: rate(ai_parse_requests_total{status="failure"}[5m]) > 0.1
        for: 5m
        annotations:
          summary: "AI解析失败率过高"

      - alert: AIResponseTimeSlow
        expr: histogram_quantile(0.95, ai_parse_duration_seconds) > 2
        for: 5m
        annotations:
          summary: "AI响应时间过慢"

      - alert: AITokenBudgetExceeded
        expr: sum(increase(ai_parse_tokens_used[1d])) > 1000000
        annotations:
          summary: "Token使用量超预算"
```

---

## 📌 下一步操作

### 立即开始
1. 创建 `pkg/ai/` 目录结构
2. 实现核心Provider接口
3. 集成OpenAI SDK
4. 编写基础测试用例

### 后续任务
- [ ] **C3**: 智能降级机制优化
- [ ] **C4**: 双解析器架构部署
- [ ] **D1-D4**: 智能功能增强 (第四阶段)

---

**状态**: 📋 待实施
**预计完成日期**: 2025年10月5日
**责任人**: 开发团队
**审核人**: 技术负责人

**标签**: #MMemory #AI集成 #OpenAI #Claude #第三阶段 #C2任务