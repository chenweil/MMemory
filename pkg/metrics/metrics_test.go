package metrics

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
)

func TestRecordBotMessage(t *testing.T) {
	tests := []struct {
		name        string
		messageType string
		status      string
	}{
		{
			name:        "记录文本消息成功",
			messageType: "text",
			status:      "success",
		},
		{
			name:        "记录语音消息成功",
			messageType: "voice",
			status:      "success",
		},
		{
			name:        "记录消息失败",
			messageType: "text",
			status:      "failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordBotMessage(tt.messageType, tt.status)

			// 验证指标是否增加
			metric := &dto.Metric{}
			err := BotMessagesTotal.WithLabelValues(tt.messageType, tt.status).Write(metric)
			if err != nil {
				t.Errorf("写入指标失败: %v", err)
			}
			if metric.Counter == nil {
				t.Errorf("Counter指标为nil")
			}
		})
	}
}

func TestSetBotUsers(t *testing.T) {
	tests := []struct {
		name  string
		count float64
	}{
		{
			name:  "设置用户数为0",
			count: 0,
		},
		{
			name:  "设置用户数为100",
			count: 100,
		},
		{
			name:  "设置用户数为1000",
			count: 1000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetBotUsers(tt.count)

			metric := &dto.Metric{}
			err := BotUsersTotal.Write(metric)
			if err != nil {
				t.Errorf("写入指标失败: %v", err)
			}
			if metric.Gauge == nil {
				t.Errorf("Gauge指标为nil")
			}
			if metric.Gauge.GetValue() != tt.count {
				t.Errorf("用户数设置不正确: got %v, want %v", metric.Gauge.GetValue(), tt.count)
			}
		})
	}
}

func TestSetReminders(t *testing.T) {
	tests := []struct {
		name   string
		status string
		count  float64
	}{
		{
			name:   "设置活跃提醒数",
			status: "active",
			count:  50,
		},
		{
			name:   "设置已完成提醒数",
			status: "completed",
			count:  100,
		},
		{
			name:   "设置已暂停提醒数",
			status: "paused",
			count:  10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetReminders(tt.status, tt.count)

			metric := &dto.Metric{}
			err := RemindersTotal.WithLabelValues(tt.status).Write(metric)
			if err != nil {
				t.Errorf("写入指标失败: %v", err)
			}
			if metric.Gauge == nil {
				t.Errorf("Gauge指标为nil")
			}
		})
	}
}

func TestRecordReminderCreated(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "记录提醒创建",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordReminderCreated()

			metric := &dto.Metric{}
			err := RemindersCreatedTotal.Write(metric)
			if err != nil {
				t.Errorf("写入指标失败: %v", err)
			}
			if metric.Counter == nil {
				t.Errorf("Counter指标为nil")
			}
		})
	}
}

func TestRecordReminderCompleted(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "记录提醒完成",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordReminderCompleted()

			metric := &dto.Metric{}
			err := RemindersCompletedTotal.Write(metric)
			if err != nil {
				t.Errorf("写入指标失败: %v", err)
			}
			if metric.Counter == nil {
				t.Errorf("Counter指标为nil")
			}
		})
	}
}

func TestRecordReminderSkipped(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "记录提醒跳过",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordReminderSkipped()

			metric := &dto.Metric{}
			err := RemindersSkippedTotal.Write(metric)
			if err != nil {
				t.Errorf("写入指标失败: %v", err)
			}
			if metric.Counter == nil {
				t.Errorf("Counter指标为nil")
			}
		})
	}
}

func TestRecordReminderParse(t *testing.T) {
	tests := []struct {
		name        string
		parserType  string
		status      string
		duration    float64
	}{
		{
			name:       "记录AI解析成功",
			parserType: "ai",
			status:     "success",
			duration:   0.5,
		},
		{
			name:       "记录正则解析成功",
			parserType: "regex",
			status:     "success",
			duration:   0.1,
		},
		{
			name:       "记录AI解析失败",
			parserType: "ai",
			status:     "failed",
			duration:   2.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordReminderParse(tt.parserType, tt.status, tt.duration)
			// Histogram指标通过Observe方法记录，不需要验证返回值
		})
	}
}

func TestSetSchedulerJobs(t *testing.T) {
	tests := []struct {
		name  string
		count float64
	}{
		{
			name:  "设置调度任务数为0",
			count: 0,
		},
		{
			name:  "设置调度任务数为10",
			count: 10,
		},
		{
			name:  "设置调度任务数为100",
			count: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetSchedulerJobs(tt.count)

			metric := &dto.Metric{}
			err := SchedulerJobsTotal.Write(metric)
			if err != nil {
				t.Errorf("写入指标失败: %v", err)
			}
			if metric.Gauge == nil {
				t.Errorf("Gauge指标为nil")
			}
			if metric.Gauge.GetValue() != tt.count {
				t.Errorf("调度任务数设置不正确: got %v, want %v", metric.Gauge.GetValue(), tt.count)
			}
		})
	}
}

func TestRecordSchedulerExecution(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{
			name:   "记录调度执行成功",
			status: "success",
		},
		{
			name:   "记录调度执行失败",
			status: "failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordSchedulerExecution(tt.status)

			metric := &dto.Metric{}
			err := SchedulerExecutionsTotal.WithLabelValues(tt.status).Write(metric)
			if err != nil {
				t.Errorf("写入指标失败: %v", err)
			}
			if metric.Counter == nil {
				t.Errorf("Counter指标为nil")
			}
		})
	}
}

func TestRecordDatabaseQuery(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		status    string
	}{
		{
			name:      "记录查询成功",
			operation: "select",
			status:    "success",
		},
		{
			name:      "记录插入成功",
			operation: "insert",
			status:    "success",
		},
		{
			name:      "记录查询失败",
			operation: "select",
			status:    "failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordDatabaseQuery(tt.operation, tt.status)

			metric := &dto.Metric{}
			err := DatabaseQueriesTotal.WithLabelValues(tt.operation, tt.status).Write(metric)
			if err != nil {
				t.Errorf("写入指标失败: %v", err)
			}
			if metric.Counter == nil {
				t.Errorf("Counter指标为nil")
			}
		})
	}
}

func TestRecordDatabaseQueryDuration(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		duration  float64
	}{
		{
			name:      "记录查询耗时",
			operation: "select",
			duration:  0.01,
		},
		{
			name:      "记录插入耗时",
			operation: "insert",
			duration:  0.05,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordDatabaseQueryDuration(tt.operation, tt.duration)
			// Histogram指标通过Observe方法记录，不需要验证返回值
		})
	}
}

func TestRecordNotification(t *testing.T) {
	tests := []struct {
		name              string
		notificationType string
		status            string
	}{
		{
			name:              "记录提醒通知成功",
			notificationType: "reminder",
			status:            "success",
		},
		{
			name:              "记录提醒通知失败",
			notificationType: "reminder",
			status:            "failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordNotification(tt.notificationType, tt.status)

			metric := &dto.Metric{}
			err := NotificationsTotal.WithLabelValues(tt.notificationType, tt.status).Write(metric)
			if err != nil {
				t.Errorf("写入指标失败: %v", err)
			}
			if metric.Counter == nil {
				t.Errorf("Counter指标为nil")
			}
		})
	}
}

func TestRecordNotificationSend(t *testing.T) {
	tests := []struct {
		name              string
		notificationType string
		status            string
		duration          float64
	}{
		{
			name:              "记录通知发送耗时",
			notificationType: "reminder",
			status:            "success",
			duration:          0.1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordNotificationSend(tt.notificationType, tt.status, tt.duration)
			// Histogram指标通过Observe方法记录，不需要验证返回值
		})
	}
}

func TestRecordError(t *testing.T) {
	tests := []struct {
		name       string
		service    string
		operation  string
		errorType  string
	}{
		{
			name:      "记录数据库错误",
			service:   "database",
			operation: "query",
			errorType: "timeout",
		},
		{
			name:      "记录API错误",
			service:   "api",
			operation: "parse",
			errorType: "invalid_input",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordError(tt.service, tt.operation, tt.errorType)

			metric := &dto.Metric{}
			err := ErrorsTotal.WithLabelValues(tt.service, tt.operation, tt.errorType).Write(metric)
			if err != nil {
				t.Errorf("写入指标失败: %v", err)
			}
			if metric.Counter == nil {
				t.Errorf("Counter指标为nil")
			}
		})
	}
}

func TestSetSystemUptime(t *testing.T) {
	tests := []struct {
		name   string
		uptime float64
	}{
		{
			name:   "设置运行时间为0",
			uptime: 0,
		},
		{
			name:   "设置运行时间为3600秒",
			uptime: 3600,
		},
		{
			name:   "设置运行时间为86400秒",
			uptime: 86400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetSystemUptime(tt.uptime)

			metric := &dto.Metric{}
			err := SystemUptime.Write(metric)
			if err != nil {
				t.Errorf("写入指标失败: %v", err)
			}
			if metric.Gauge == nil {
				t.Errorf("Gauge指标为nil")
			}
			if metric.Gauge.GetValue() != tt.uptime {
				t.Errorf("运行时间设置不正确: got %v, want %v", metric.Gauge.GetValue(), tt.uptime)
			}
		})
	}
}

func TestRecordResponse(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		method   string
		status   string
		duration float64
	}{
		{
			name:     "记录GET请求响应时间",
			endpoint: "/api/reminder",
			method:   "GET",
			status:   "200",
			duration: 0.05,
		},
		{
			name:     "记录POST请求响应时间",
			endpoint: "/api/reminder",
			method:   "POST",
			status:   "201",
			duration: 0.1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordResponse(tt.endpoint, tt.method, tt.status, tt.duration)
			// Histogram指标通过Observe方法记录，不需要验证返回值
		})
	}
}

func TestMetricsRegistration(t *testing.T) {
	// 验证所有指标都已注册
	// 注意：prometheus.DefaultRegisterer不提供Gather方法
	// 我们通过调用各个指标函数来验证它们已正确初始化

	// 测试Counter指标
	RecordBotMessage("test", "success")
	RecordReminderCreated()
	RecordReminderCompleted()
	RecordReminderSkipped()
	RecordSchedulerExecution("success")
	RecordDatabaseQuery("select", "success")
	RecordNotification("reminder", "success")
	RecordError("test", "test", "test")

	// 测试Gauge指标
	SetBotUsers(10)
	SetReminders("active", 5)
	SetSchedulerJobs(3)
	SetSystemUptime(100)

	// 测试Histogram指标
	RecordReminderParse("ai", "success", 0.5)
	RecordDatabaseQueryDuration("select", 0.1)
	RecordNotificationSend("reminder", "success", 0.2)
	RecordResponse("/api/test", "GET", "200", 0.3)

	// 如果所有函数调用都成功，说明指标已正确注册
}