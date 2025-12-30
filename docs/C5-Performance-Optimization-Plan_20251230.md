# C5: 性能优化与监控增强

**计划日期**: 2025-12-30
**预计周期**: 2-3 周

## 概述

C5 阶段旨在提升系统整体性能和可观测性，通过数据库优化、调度器调优、AI 成本控制增强和监控改进，打造更高效、更可靠的智能提醒服务。

## 目标

1. **性能提升**: 数据库查询性能提升 30%+
2. **成本优化**: AI 使用成本降低 20%+
3. **可观测性**: 完善监控指标覆盖率达到 90%+
4. **稳定性**: 系统可用性提升至 99.9%

## 核心改进领域

### 1. 数据库连接池优化

**当前状态**:
- `max_open_conns`: 25
- `max_idle_conns`: 10
- 无连接预热机制

**优化方案**:
```go
// config/config.go 增强配置
type DatabaseConfig struct {
    Driver       string `mapstructure:"driver"`
    DSN          string `mapstructure:"dsn"`
    MaxOpenConns int    `mapstructure:"max_open_conns"`
    MaxIdleConns int    `mapstructure:"max_idle_conns"`
    ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"` // 新增
    ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"` // 新增
    MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"` // 新增
}

// 新增配置项默认值
cm.viper.SetDefault("database.conn_max_lifetime", "5m")
cm.viper.SetDefault("database.conn_max_idle_time", "1m")
cm.viper.SetDefault("database.max_conn_lifetime", "30m")
```

**改进点**:
1. 增加连接最大生命周期设置
2. 添加预热机制，启动时建立初始连接
3. 实现连接健康检查
4. 根据负载动态调整连接池大小

### 2. 调度器性能优化

**当前状态**:
- `max_workers`: 10
- 固定线程池大小
- 无工作队列监控

**优化方案**:
```go
type SchedulerConfig struct {
    Timezone         string `mapstructure:"timezone"`
    MaxWorkers       int    `mapstructure:"max_workers"`
    MinWorkers       int    `mapstructure:"min_workers"`       // 新增
    WorkQueueSize    int    `mapstructure:"work_queue_size"`   // 新增
    HealthCheckInterval time.Duration `mapstructure:"health_check_interval"` // 新增
}
```

**改进点**:
1. 实现动态工作线程池（Min/Max Workers）
2. 添加工作队列监控指标
3. 优化任务调度延迟
4. 实现任务优先级机制
5. 添加调度器健康检查端点

### 3. AI 成本控制增强

**当前状态**:
- 基础成本记录功能
- 简单预算检查
- 无成本预测

**优化方案**:
```go
// pkg/ai/cost_controller.go 增强
type CostController struct {
    // ... 现有字段
    dailyCosts       map[string][]float64       // 新增：每日成本历史
    predictionModel  CostPredictionModel        // 新增：成本预测模型
    optimizationRules []CostOptimizationRule    // 新增：优化规则
}

type CostPredictionModel struct {
    TrendAnalysis    float64   // 趋势分析
    PredictedNextDay float64   // 预测次日成本
    Confidence       float64   // 置信度
}

type CostOptimizationRule struct {
    Name        string
    Condition   func(*CostReport) bool
    Action      func(*CostController) error
    Priority    int
}
```

**改进点**:
1. 实现成本预测模型
2. 添加智能优化规则（如自动切换低成本模型）
3. 增强预算告警（每日/每周/每月）
4. 实现成本趋势分析
5. 添加成本优化建议 API

### 4. 监控增强

**当前状态**:
- 基础 Prometheus 指标
- 简单告警规则
- 无自定义仪表板

**新增 Prometheus 指标**:
```go
// 数据库指标
dbPoolSizeGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
    Name: "mmemory_db_pool_size",
    Help: "Current database connection pool size",
}, []string{"state"}) // state: open, idle, in_use

dbWaitCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
    Name: "mmemory_db_wait_count",
    Help: "Number of connections waiting for acquisition",
})

// 调度器指标
schedulerQueueSizeGauge = promauto.NewGauge(prometheus.GaugeOpts{
    Name: "mmemory_scheduler_queue_size",
    Help: "Current scheduler work queue size",
})

schedulerActiveWorkersGauge = promauto.NewGauge(prometheus.GaugeOpts{
    Name: "mmemory_scheduler_active_workers",
    Help: "Number of active worker goroutines",
})

// AI 成本预测指标
aiCostPredictionGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
    Name: "mmemory_ai_cost_prediction",
    Help: "AI cost prediction for next period",
}, []string{"period", "provider"})

aiBudgetUtilizationGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
    Name: "mmemory_ai_budget_utilization",
    Help: "AI budget utilization percentage",
}, []string{"budget_type"}) // budget_type: daily, monthly, user
```

**增强告警规则**:
```yaml
alerts:
  - name: "DB Pool Exhaustion"
    expr: mmemory_db_pool_size{state="in_use"} / mmemory_db_pool_size_total >= 0.9
    severity: warning
    for: 2m

  - name: "AI Cost Anomaly"
    expr: increase(mmemory_ai_request_cost_total[1h]) > increase(mmemory_ai_request_cost_total[2h]) * 2
    severity: warning
    for: 5m

  - name: "Scheduler Lag"
    expr: mmemory_scheduler_queue_size > 100
    severity: warning
    for: 1m
```

### 5. 查询优化

**当前状态**:
- 基本 GORM 查询
- 无查询缓存
- 无批量操作优化

**优化方案**:
1. 实现查询结果缓存（针对固定数据）
2. 优化提醒查询（添加索引提示）
3. 实现批量插入/更新操作
4. 添加慢查询日志

```go
// internal/repository/sqlite/reminder.go 优化
func (r *reminderRepository) GetActiveRemindersBatch(ctx context.Context, batchSize int, offset int) ([]*models.Reminder, error) {
    var reminders []*models.Reminder
    err := r.db.WithContext(ctx).
        Where("is_active = ?", true).
        Order("next_trigger_at ASC").
        Limit(batchSize).
        Offset(offset).
        Find(&reminders).Error
    return reminders, err
}

func (r *reminderRepository) BulkUpdateReminderTimes(ctx context.Context, updates map[uint]time.Time) error {
    // 批量更新提醒触发时间
    return r.db.Transaction(func(tx *gorm.DB) error {
        for id, triggerAt := range updates {
            if err := tx.Model(&models.Reminder{}).Where("id = ?", id).Update("next_trigger_at", triggerAt).Error; err != nil {
                return err
            }
        }
        return nil
    })
}
```

### 6. 缓存层增强

**当前状态**:
- 基础 LRU 缓存
- 固定 TTL

**优化方案**:
```go
type CacheConfig struct {
    Enabled       bool          `mapstructure:"enabled"`
    TTL           time.Duration `mapstructure:"ttl"`
    MaxSize       int           `mapstructure:"max_size"`
    CleanupPeriod time.Duration `mapstructure:"cleanup_period"` // 新增
    EvictionPolicy string       `mapstructure:"eviction_policy"` // 新增: lru, lfu, fifo
}

// 新增功能
type Cache interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}) error
    Delete(key string) error
    Purge() error
    Stats() CacheStats
    Size() int
    Capacity() int
}

type CacheStats struct {
    Hits       uint64
    Misses     uint64
    Evictions  uint64
    Size       int
    Utilization float64
}
```

## 实施计划

### Week 1: 基础优化

| 任务 | 描述 | 优先级 |
|-----|------|--------|
| T1.1 | 数据库连接池配置增强 | P0 |
| T1.2 | 连接健康检查实现 | P0 |
| T1.3 | 调度器动态工作池实现 | P0 |
| T1.4 | 新增数据库监控指标 | P1 |
| T1.5 | 新增调度器监控指标 | P1 |

### Week 2: AI 成本优化

| 任务 | 描述 | 优先级 |
|-----|------|--------|
| T2.1 | 成本预测模型实现 | P0 |
| T2.2 | 智能优化规则引擎 | P0 |
| T2.3 | 成本预测 Prometheus 指标 | P1 |
| T2.4 | 增强成本告警规则 | P1 |
| T2.5 | 成本优化建议 API | P2 |

### Week 3: 监控与完善

| 任务 | 描述 | 优先级 |
|-----|------|--------|
| T3.1 | 查询优化与慢查询日志 | P1 |
| T3.2 | 缓存层增强实现 | P1 |
| T3.3 | Grafana 仪表板更新 | P1 |
| T3.4 | 端到端测试覆盖 | P2 |
| T3.5 | 性能基准测试 | P2 |

## 验收标准

### 性能指标

| 指标 | 当前值 | 目标值 |
|-----|--------|--------|
| 数据库查询延迟 (p95) | ~50ms | <35ms |
| 调度器任务响应延迟 | ~100ms | <50ms |
| AI 响应延迟 (p95) | ~2s | <1.5s |
| 系统吞吐量 (QPS) | 100 | 150+ |

### 成本指标

| 指标 | 当前值 | 目标值 |
|-----|--------|--------|
| AI 月均成本 | - | 降低 20% |
| 预算超支次数/月 | - | <1 |
| 成本预测准确率 | - | >85% |

### 监控指标

| 指标 | 目标值 |
|-----|--------|
| 监控覆盖率 | >90% |
| 告警准确率 | >95% |
| 仪表板可用性 | 100% |

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|-----|------|----------|
| 连接池配置不当导致性能下降 | 高 | 灰度发布，渐进式调整 |
| 成本预测模型不准确 | 中 | 设置置信度阈值，保守预测 |
| 新指标影响系统性能 | 低 | 使用高效指标采集 |

## 测试策略

1. **负载测试**: 使用 wrk 或 hey 进行压力测试
2. **基准测试**: 建立性能基准，持续跟踪
3. **混沌测试**: 模拟故障场景验证恢复能力
4. **成本模拟**: 使用历史数据验证预测模型

## 文档更新

- [ ] 更新 CLAUDE.md 架构文档
- [ ] 更新 IFLOW.md 阶段计划
- [ ] 更新 README.md 监控配置说明
- [ ] 创建性能调优指南

## 资源估算

- **开发时间**: 2-3 周
- **代码改动**: 约 1500-2000 行
- **测试覆盖**: 新增 30+ 测试用例
- **文档更新**: 5+ 份文档

## 里程碑

| 里程碑 | 日期 | 交付物 |
|--------|------|--------|
| M1: 基础优化完成 | Week 1 结束 | 连接池、调度器优化代码 |
| M2: 成本优化完成 | Week 2 结束 | 成本预测、优化引擎代码 |
| M3: 全部完成 | Week 3 结束 | 完整 C5 功能、测试、文档 |
