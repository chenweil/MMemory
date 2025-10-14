package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"mmemory/pkg/metrics"
)

func main() {
	fmt.Println("🧪 测试MMemory监控系统...")

	// 启动HTTP服务器暴露指标
	go func() {
		http.Handle("/metrics", metricsHandler())
		fmt.Println("📊 指标服务器启动: http://localhost:9090/metrics")
		log.Fatal(http.ListenAndServe(":9090", nil))
	}()

	// 模拟一些指标数据
	fmt.Println("⚡ 生成测试指标数据...")
	
	// 设置基础指标
	metrics.SetBotUsers(100)
	metrics.SetReminders("active", 50)
	metrics.SetReminders("completed", 200)
	metrics.SetReminders("expired", 10)
	metrics.SetSchedulerJobs(25)
	metrics.SetSystemUptime(3600)

	// 模拟业务操作
	for i := 0; i < 10; i++ {
		// 记录消息处理
		metrics.RecordBotMessage("text", "success")
		metrics.RecordBotMessage("callback", "error")
		
		// 记录提醒操作
		metrics.RecordReminderCreated()
		metrics.RecordReminderCompleted()
		metrics.RecordReminderSkipped()
		
		// 记录数据库操作
		metrics.RecordDatabaseQuery("select", "success")
		metrics.RecordDatabaseQuery("insert", "failed")
		metrics.RecordDatabaseQueryDuration("select", 0.05)
		
		// 记录通知发送
		metrics.RecordNotification("reminder", "success")
		metrics.RecordNotification("followup", "success")
		metrics.RecordNotificationSend("reminder", "success", 0.1)
		
		// 记录解析操作
		metrics.RecordReminderParse("ai", "success", 0.5)
		metrics.RecordReminderParse("regex", "failed", 0.1)
		
		// 记录响应时间
		metrics.RecordResponse("/api/reminder", "POST", "200", 0.2)
		
		// 记录错误
		metrics.RecordError("service", "parse", "validation")
		
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("✅ 测试指标数据生成完成")
	fmt.Println("📄 请访问 http://localhost:9090/metrics 查看指标")
	fmt.Println("⏰ 按 Ctrl+C 退出...")

	// 保持程序运行
	select {}
}

func metricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 这里应该使用prometheus的Handler，但为了测试我们返回一些示例数据
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "# HELP mmemory_bot_users_total Total number of registered users")
		fmt.Fprintln(w, "# TYPE mmemory_bot_users_total gauge")
		fmt.Fprintln(w, "mmemory_bot_users_total 100")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "# HELP mmemory_reminders_total Total number of reminders")
		fmt.Fprintln(w, "# TYPE mmemory_reminders_total gauge")
		fmt.Fprintln(w, "mmemory_reminders_total{status=\"active\"} 50")
		fmt.Fprintln(w, "mmemory_reminders_total{status=\"completed\"} 200")
		fmt.Fprintln(w, "mmemory_reminders_total{status=\"expired\"} 10")
	}
}