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