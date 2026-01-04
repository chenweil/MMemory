package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Bot相关指标
	BotMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mmemory_bot_messages_total",
			Help: "Total number of messages processed by the bot",
		},
		[]string{"type", "status"},
	)

	BotUsersTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mmemory_bot_users_total",
			Help: "Total number of registered users",
		},
	)

	// 提醒相关指标
	RemindersTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mmemory_reminders_total",
			Help: "Total number of reminders",
		},
		[]string{"status"},
	)

	RemindersCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "mmemory_reminders_created_total",
			Help: "Total number of reminders created",
		},
	)

	RemindersCompletedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "mmemory_reminders_completed_total",
			Help: "Total number of reminders completed",
		},
	)

	RemindersSkippedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "mmemory_reminders_skipped_total",
			Help: "Total number of reminders skipped",
		},
	)

	ReminderParseDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mmemory_reminder_parse_duration_seconds",
			Help:    "Duration of reminder parsing operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"parser_type", "status"},
	)

	// 调度器相关指标
	SchedulerJobsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mmemory_scheduler_jobs_total",
			Help: "Total number of scheduled jobs",
		},
	)

	SchedulerExecutionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mmemory_scheduler_executions_total",
			Help: "Total number of scheduler executions",
		},
		[]string{"status"},
	)

	// 数据库相关指标
	DatabaseQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mmemory_database_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "status"},
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mmemory_database_query_duration_seconds",
			Help:    "Duration of database queries",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// 通知相关指标
	NotificationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mmemory_notifications_total",
			Help: "Total number of notifications sent",
		},
		[]string{"type", "status"},
	)

	NotificationSendDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mmemory_notification_send_duration_seconds",
			Help:    "Duration of notification sending operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type", "status"},
	)

	// 错误相关指标
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mmemory_errors_total",
			Help: "Total number of errors",
		},
		[]string{"service", "operation", "error_type"},
	)

	// 系统健康指标
	SystemUptime = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mmemory_system_uptime_seconds",
			Help: "System uptime in seconds",
		},
	)

	// 性能指标
	ResponseDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mmemory_response_duration_seconds",
			Help:    "Duration of API responses",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint", "method", "status"},
	)
)

// RecordBotMessage 记录Bot消息处理
func RecordBotMessage(messageType, status string) {
	BotMessagesTotal.WithLabelValues(messageType, status).Inc()
}

// SetBotUsers 设置Bot用户总数
func SetBotUsers(count float64) {
	BotUsersTotal.Set(count)
}

// SetReminders 设置提醒数量
func SetReminders(status string, count float64) {
	RemindersTotal.WithLabelValues(status).Set(count)
}

// RecordReminderCreated 记录提醒创建
func RecordReminderCreated() {
	RemindersCreatedTotal.Inc()
}

// RecordReminderCompleted 记录提醒完成
func RecordReminderCompleted() {
	RemindersCompletedTotal.Inc()
}

// RecordReminderSkipped 记录提醒跳过
func RecordReminderSkipped() {
	RemindersSkippedTotal.Inc()
}

// RecordReminderParse 记录提醒解析耗时
func RecordReminderParse(parserType, status string, duration float64) {
	ReminderParseDuration.WithLabelValues(parserType, status).Observe(duration)
}

// SetSchedulerJobs 设置调度任务数量
func SetSchedulerJobs(count float64) {
	SchedulerJobsTotal.Set(count)
}

// RecordSchedulerExecution 记录调度器执行
func RecordSchedulerExecution(status string) {
	SchedulerExecutionsTotal.WithLabelValues(status).Inc()
}

// RecordDatabaseQuery 记录数据库查询
func RecordDatabaseQuery(operation, status string) {
	DatabaseQueriesTotal.WithLabelValues(operation, status).Inc()
}

// RecordDatabaseQueryDuration 记录数据库查询耗时
func RecordDatabaseQueryDuration(operation string, duration float64) {
	DatabaseQueryDuration.WithLabelValues(operation).Observe(duration)
}

// RecordNotification 记录通知发送
func RecordNotification(notificationType, status string) {
	NotificationsTotal.WithLabelValues(notificationType, status).Inc()
}

// RecordNotificationSend 记录通知发送耗时
func RecordNotificationSend(notificationType, status string, duration float64) {
	NotificationSendDuration.WithLabelValues(notificationType, status).Observe(duration)
}

// RecordError 记录错误
func RecordError(service, operation, errorType string) {
	ErrorsTotal.WithLabelValues(service, operation, errorType).Inc()
}

// SetSystemUptime 设置系统运行时间
func SetSystemUptime(uptime float64) {
	SystemUptime.Set(uptime)
}

// RecordResponse 记录响应时间
func RecordResponse(endpoint, method, status string, duration float64) {
	ResponseDuration.WithLabelValues(endpoint, method, status).Observe(duration)
}

// ========== 数据库连接池指标 ==========

var (
	// DatabasePoolSize 数据库连接池大小
	DatabasePoolSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mmemory_db_pool_size",
			Help: "Current database connection pool size by state",
		},
		[]string{"state"}, // state: open, idle, in_use
	)

	// DatabasePoolWaitCount 等待获取连接的请求数
	DatabasePoolWaitCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mmemory_db_pool_wait_count",
			Help: "Number of requests waiting for database connection",
		},
	)

	// DatabasePoolUtilization 数据库连接池利用率
	DatabasePoolUtilization = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mmemory_db_pool_utilization_ratio",
			Help: "Database connection pool utilization ratio",
		},
	)

	// DatabaseHealthStatus 数据库健康状态
	DatabaseHealthStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mmemory_db_health_status",
			Help: "Database health status (1=healthy, 0.5=degraded, 0=unhealthy)",
		},
		[]string{"database"},
	)
)

// SetDatabasePoolSize 设置数据库连接池大小
func SetDatabasePoolSize(state string, size float64) {
	DatabasePoolSize.WithLabelValues(state).Set(size)
}

// SetDatabasePoolWaitCount 设置等待连接的请求数
func SetDatabasePoolWaitCount(count float64) {
	DatabasePoolWaitCount.Set(count)
}

// SetDatabasePoolUtilization 设置连接池利用率
func SetDatabasePoolUtilization(ratio float64) {
	DatabasePoolUtilization.Set(ratio)
}

// SetDatabaseHealthStatus 设置数据库健康状态
func SetDatabaseHealthStatus(database string, status float64) {
	DatabaseHealthStatus.WithLabelValues(database).Set(status)
}

// ========== 调度器工作池指标 ==========

var (
	// SchedulerWorkerCount 调度器工作线程数
	SchedulerWorkerCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mmemory_scheduler_worker_count",
			Help: "Number of scheduler worker threads",
		},
		[]string{"state"}, // state: active, idle
	)

	// SchedulerQueueSize 调度器工作队列大小
	SchedulerQueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mmemory_scheduler_queue_size",
			Help: "Current scheduler work queue size",
		},
	)

	// SchedulerTasksTotal 调度器总任务数
	SchedulerTasksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mmemory_scheduler_tasks_total",
			Help: "Total number of scheduler tasks",
		},
		[]string{"status"}, // status: completed, failed, total
	)

	// SchedulerTaskLatency 调度器任务延迟
	SchedulerTaskLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "mmemory_scheduler_task_latency_seconds",
			Help:    "Latency of scheduler task execution",
			Buckets: prometheus.DefBuckets,
		},
	)
)

// SetSchedulerWorkerCount 设置调度器工作线程数
func SetSchedulerWorkerCount(state string, count float64) {
	SchedulerWorkerCount.WithLabelValues(state).Set(count)
}

// SetSchedulerQueueSize 设置调度器队列大小
func SetSchedulerQueueSize(size float64) {
	SchedulerQueueSize.Set(size)
}

// RecordSchedulerTask 记录调度器任务
func RecordSchedulerTask(status string) {
	SchedulerTasksTotal.WithLabelValues(status).Inc()
}

// RecordSchedulerTaskLatency 记录调度器任务延迟
func RecordSchedulerTaskLatency(duration float64) {
	SchedulerTaskLatency.Observe(duration)
}

// ========== AI 成本预测指标 ==========

var (
	// AICostPrediction 成本预测指标
	AICostPrediction = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mmemory_ai_cost_prediction",
			Help: "AI cost prediction for next period",
		},
		[]string{"period", "provider"}, // period: next_day, next_week, next_month
	)

	// AIBudgetUtilization 预算利用率
	AIBudgetUtilization = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mmemory_ai_budget_utilization",
			Help: "AI budget utilization percentage by budget type",
		},
		[]string{"budget_type"}, // budget_type: daily, monthly, user
	)

	// AICostTrend 成本趋势
	AICostTrend = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mmemory_ai_cost_trend",
			Help: "AI cost trend (positive=increasing, negative=decreasing)",
		},
	)

	// AICostRiskLevel 成本风险等级
	AICostRiskLevel = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mmemory_ai_cost_risk_level",
			Help: "AI cost risk level (0=low, 0.5=medium, 1=high)",
		},
	)

	// AIOptimizationRuleActions 优化规则执行次数
	AIOptimizationRuleActions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mmemory_ai_optimization_rule_actions_total",
			Help: "Total number of AI optimization rule actions",
		},
		[]string{"rule_name", "action_type"},
	)
)

// SetAICostPrediction 设置成本预测
func SetAICostPrediction(period, provider string, cost float64) {
	AICostPrediction.WithLabelValues(period, provider).Set(cost)
}

// SetAIBudgetUtilization 设置预算利用率
func SetAIBudgetUtilization(budgetType string, utilization float64) {
	AIBudgetUtilization.WithLabelValues(budgetType).Set(utilization)
}

// SetAICostTrend 设置成本趋势
func SetAICostTrend(trend float64) {
	AICostTrend.Set(trend)
}

// SetAICostRiskLevel 设置成本风险等级
func SetAICostRiskLevel(level float64) {
	AICostRiskLevel.Set(level)
}

// RecordAIOptimizationAction 记录优化规则执行
func RecordAIOptimizationAction(ruleName, actionType string) {
	AIOptimizationRuleActions.WithLabelValues(ruleName, actionType).Inc()
}

// ========== 查询性能指标 ==========

var (
	// QueryTotal 总查询数
	QueryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mmemory_query_total",
			Help: "Total number of database queries",
		},
		[]string{"table", "operation", "status"},
	)

	// QueryDuration 查询耗时直方图
	QueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mmemory_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
		},
		[]string{"table", "operation"},
	)

	// QuerySlowCount 慢查询计数
	QuerySlowCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mmemory_query_slow_count",
			Help: "Total number of slow database queries",
		},
		[]string{"table", "operation", "threshold"},
	)

	// QueryRowsAffected 影响行数统计
	QueryRowsAffected = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mmemory_query_rows_affected",
			Help:    "Number of rows affected by database queries",
			Buckets: []float64{0, 1, 10, 100, 1000, 10000},
		},
		[]string{"operation"},
	)

	// CacheMetrics 缓存指标
	CacheHitsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "mmemory_cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	CacheMissesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "mmemory_cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	CacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mmemory_cache_size",
			Help: "Current size of the cache",
		},
	)

	CacheEvictionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "mmemory_cache_evictions_total",
			Help: "Total number of cache evictions",
		},
	)

	CacheHitRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mmemory_cache_hit_rate",
			Help: "Cache hit rate percentage",
		},
		[]string{"cache_name", "policy"},
	)

	CacheItemsAddedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "mmemory_cache_items_added_total",
			Help: "Total number of items added to cache",
		},
	)

	CacheItemsHitTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "mmemory_cache_items_hit_total",
			Help: "Total number of cache item hits",
		},
	)
)

// RecordQueryTotal 记录总查询数
func RecordQueryTotal(table, operation, status string) {
	QueryTotal.WithLabelValues(table, operation, status).Inc()
}

// RecordQueryDuration 记录查询耗时
func RecordQueryDuration(table, operation string, duration float64) {
	QueryDuration.WithLabelValues(table, operation).Observe(duration)
}

// RecordSlowQuery 记录慢查询
func RecordSlowQuery(table, operation, threshold string) {
	QuerySlowCount.WithLabelValues(table, operation, threshold).Inc()
}

// RecordQueryRowsAffected 记录查询影响行数
func RecordQueryRowsAffected(operation string, rows float64) {
	QueryRowsAffected.WithLabelValues(operation).Observe(rows)
}

// RecordCacheHit 记录缓存命中
func RecordCacheHit() {
	CacheHitsTotal.Inc()
}

// RecordCacheMiss 记录缓存未命中
func RecordCacheMiss() {
	CacheMissesTotal.Inc()
}

// SetCacheSize 设置缓存大小
func SetCacheSize(size float64) {
	CacheSize.Set(size)
}

// RecordCacheEviction 记录缓存驱逐
func RecordCacheEviction() {
	CacheEvictionsTotal.Inc()
}

// RecordCacheHitRate 记录缓存命中率
func RecordCacheHitRate(cacheName, policy string, rate float64) {
	CacheHitRate.WithLabelValues(cacheName, policy).Set(rate)
}

// RecordCacheItemAdded 记录缓存项添加
func RecordCacheItemAdded() {
	CacheItemsAddedTotal.Inc()
}

// RecordCacheItemHit 记录缓存项命中
func RecordCacheItemHit() {
	CacheItemsHitTotal.Inc()
}