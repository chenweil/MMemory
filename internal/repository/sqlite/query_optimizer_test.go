package sqlite

import (
	"testing"
	"time"
)

func TestQueryOptimizer_RecordQuery(t *testing.T) {
	optimizer := NewQueryOptimizer(10 * time.Millisecond)
	defer optimizer.Stop()

	// 测试记录查询
	optimizer.RecordQuery("users", "SELECT", "SELECT * FROM users", 5*time.Millisecond, 10, false)
	optimizer.RecordQuery("users", "SELECT", "SELECT * FROM users", 15*time.Millisecond, 10, false)
	optimizer.RecordQuery("reminders", "INSERT", "INSERT INTO reminders", 8*time.Millisecond, 1, false)

	metrics := optimizer.GetMetrics()

	if metrics.TotalQueries != 3 {
		t.Errorf("期望总查询数为3，实际为: %d", metrics.TotalQueries)
	}

	if metrics.SlowQueries != 1 {
		t.Errorf("期望慢查询数为1，实际为: %d", metrics.SlowQueries)
	}

	// 检查表级指标
	if metrics.ByTable["users"] == nil {
		t.Error("期望有users表的指标")
	}
	if metrics.ByTable["users"].QueryCount != 2 {
		t.Errorf("期望users表查询数为2，实际为: %d", metrics.ByTable["users"].QueryCount)
	}

	// 检查操作级指标
	if metrics.ByOperation["SELECT"] == nil {
		t.Error("期望有SELECT操作的指标")
	}
	if metrics.ByOperation["SELECT"].SlowCount != 1 {
		t.Errorf("期望SELECT慢查询数为1，实际为: %d", metrics.ByOperation["SELECT"].SlowCount)
	}
}

func TestQueryOptimizer_SlowQueryLog(t *testing.T) {
	optimizer := NewQueryOptimizer(50 * time.Millisecond)
	defer optimizer.Stop()

	// 记录一个慢查询
	optimizer.RecordQuery("reminders", "SELECT", "SELECT * FROM reminders WHERE id > 100", 100*time.Millisecond, 50, false)

	logs := optimizer.GetSlowQueries(10)

	if len(logs) != 1 {
		t.Errorf("期望慢查询日志数为1，实际为: %d", len(logs))
	}

	if logs[0].Table != "reminders" {
		t.Errorf("期望表名为reminders，实际为: %s", logs[0].Table)
	}

	if logs[0].Operation != "SELECT" {
		t.Errorf("期望操作名为SELECT，实际为: %s", logs[0].Operation)
	}
}

func TestQueryOptimizer_SlowQueryStats(t *testing.T) {
	optimizer := NewQueryOptimizer(10 * time.Millisecond)
	defer optimizer.Stop()

	// 记录一些查询
	optimizer.RecordQuery("users", "SELECT", "SELECT * FROM users", 5*time.Millisecond, 10, false)
	optimizer.RecordQuery("reminders", "INSERT", "INSERT INTO reminders", 20*time.Millisecond, 1, false)
	optimizer.RecordQuery("reminders", "UPDATE", "UPDATE reminders SET", 30*time.Millisecond, 1, false)

	stats := optimizer.GetSlowQueryStats()

	if stats["total_queries"].(int64) != 3 {
		t.Errorf("期望总查询数为3，实际为: %v", stats["total_queries"])
	}

	if stats["slow_queries"].(int64) != 2 {
		t.Errorf("期望慢查询数为2，实际为: %v", stats["slow_queries"])
	}

	// 检查最慢的表
	if stats["slowest_table"] != "reminders" {
		t.Errorf("期望最慢的表为reminders，实际为: %v", stats["slowest_table"])
	}
}

func TestQueryOptimizer_TruncateSQL(t *testing.T) {
	longSQL := "SELECT * FROM users WHERE id > 1 AND name LIKE '%test%' AND created_at > '2024-01-01' ORDER BY id DESC LIMIT 100"

	truncated := truncateSQL(longSQL)

	if len(truncated) > 203 { // 200 + "..."
		t.Errorf("截断后的SQL长度超过预期: %d", len(truncated))
	}

	// 短SQL不应该被截断
	shortSQL := "SELECT * FROM users"
	truncatedShort := truncateSQL(shortSQL)
	if truncatedShort != shortSQL {
		t.Errorf("短SQL不应该被截断")
	}
}

func TestQueryOptimizer_MaxLogEntries(t *testing.T) {
	optimizer := &QueryOptimizer{
		slowThreshold: 1 * time.Millisecond,
		queryMetrics:  &QueryMetrics{ByTable: make(map[string]*TableQueryMetrics), ByOperation: make(map[string]*OperationQueryMetrics)},
		slowQueryLogs: make([]SlowQueryLog, 0),
		maxLogEntries: 5,
		stopCh:        make(chan struct{}),
	}

	// 记录超过最大条目的慢查询
	for i := 0; i < 10; i++ {
		optimizer.RecordQuery("test", "SELECT", "SELECT *", 5*time.Millisecond, 0, false)
	}

	logs := optimizer.GetSlowQueries(100)

	if len(logs) > optimizer.maxLogEntries {
		t.Errorf("日志条目数不应超过最大值，实际: %d, 最大: %d", len(logs), optimizer.maxLogEntries)
	}
}

func TestQueryOptimizer_SkipLog(t *testing.T) {
	optimizer := NewQueryOptimizer(10 * time.Millisecond)
	defer optimizer.Stop()

	// 记录慢查询，跳过日志
	optimizer.RecordQuery("test", "SELECT", "SELECT *", 20*time.Millisecond, 10, true)

	// 检查指标是否记录了慢查询
	metrics := optimizer.GetMetrics()
	if metrics.SlowQueries != 1 {
		t.Errorf("期望慢查询数为1，实际为: %d", metrics.SlowQueries)
	}

	// 检查日志是否被跳过
	logs := optimizer.GetSlowQueries(10)
	if len(logs) != 0 {
		t.Errorf("期望日志数为0（跳过），实际为: %d", len(logs))
	}
}
