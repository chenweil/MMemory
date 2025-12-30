package sqlite

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	loggermm "mmemory/pkg/logger"
)

// QueryMetrics 查询指标统计
type QueryMetrics struct {
	TotalQueries      int64         // 总查询数
	SlowQueries       int64         // 慢查询数
	TotalDuration     time.Duration // 总耗时
	AverageDuration   time.Duration // 平均耗时
	MinDuration       time.Duration // 最小耗时
	MaxDuration       time.Duration // 最大耗时
	ByTable           map[string]*TableQueryMetrics // 按表统计
	ByOperation       map[string]*OperationQueryMetrics // 按操作统计
	mu                sync.RWMutex
}

// TableQueryMetrics 表级查询指标
type TableQueryMetrics struct {
	QueryCount      int64
	TotalDuration   time.Duration
	AverageDuration time.Duration
	SlowCount       int64
}

// OperationQueryMetrics 操作级查询指标
type OperationQueryMetrics struct {
	QueryCount      int64
	TotalDuration   time.Duration
	AverageDuration time.Duration
	SlowCount       int64
}

// SlowQueryLog 慢查询日志
type SlowQueryLog struct {
	Timestamp    time.Time      // 执行时间
	Table        string         // 表名
	Operation    string         // 操作类型
	SQL          string         // SQL语句
	Duration     time.Duration  // 执行耗时
	RowsAffected int64          // 影响行数
	Context      string         // 调用上下文
}

// QueryOptimizer 查询优化器
type QueryOptimizer struct {
	slowThreshold  time.Duration // 慢查询阈值
	queryMetrics   *QueryMetrics
	slowQueryLogs  []SlowQueryLog
	maxLogEntries  int
	mu             sync.RWMutex
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

// NewQueryOptimizer 创建查询优化器
func NewQueryOptimizer(slowThreshold time.Duration) *QueryOptimizer {
	optimizer := &QueryOptimizer{
		slowThreshold: slowThreshold,
		queryMetrics: &QueryMetrics{
			ByTable:     make(map[string]*TableQueryMetrics),
			ByOperation: make(map[string]*OperationQueryMetrics),
		},
		slowQueryLogs: make([]SlowQueryLog, 0),
		maxLogEntries: 1000,
		stopCh:        make(chan struct{}),
	}

	// 启动日志清理goroutine
	optimizer.wg.Add(1)
	go optimizer.cleanupLogs()

	return optimizer
}

// RecordQuery 记录查询执行信息
func (qo *QueryOptimizer) RecordQuery(table, operation, sql string, duration time.Duration, rowsAffected int64, skipLog bool) {
	qo.queryMetrics.mu.Lock()
	defer qo.queryMetrics.mu.Unlock()

	qo.queryMetrics.TotalQueries++
	qo.queryMetrics.TotalDuration += duration

	// 更新平均耗时
	if qo.queryMetrics.TotalQueries == 1 {
		qo.queryMetrics.MinDuration = duration
		qo.queryMetrics.MaxDuration = duration
	} else {
		if duration < qo.queryMetrics.MinDuration {
			qo.queryMetrics.MinDuration = duration
		}
		if duration > qo.queryMetrics.MaxDuration {
			qo.queryMetrics.MaxDuration = duration
		}
	}
	qo.queryMetrics.AverageDuration = qo.queryMetrics.TotalDuration / time.Duration(qo.queryMetrics.TotalQueries)

	// 更新表级指标
	if _, ok := qo.queryMetrics.ByTable[table]; !ok {
		qo.queryMetrics.ByTable[table] = &TableQueryMetrics{}
	}
	tableMetrics := qo.queryMetrics.ByTable[table]
	tableMetrics.QueryCount++
	tableMetrics.TotalDuration += duration
	tableMetrics.AverageDuration = tableMetrics.TotalDuration / time.Duration(tableMetrics.QueryCount)

	// 更新操作级指标
	if _, ok := qo.queryMetrics.ByOperation[operation]; !ok {
		qo.queryMetrics.ByOperation[operation] = &OperationQueryMetrics{}
	}
	opMetrics := qo.queryMetrics.ByOperation[operation]
	opMetrics.QueryCount++
	opMetrics.TotalDuration += duration
	opMetrics.AverageDuration = opMetrics.TotalDuration / time.Duration(opMetrics.QueryCount)

	// 记录慢查询
	if duration >= qo.slowThreshold {
		qo.queryMetrics.SlowQueries++
		tableMetrics.SlowCount++
		opMetrics.SlowCount++
		if !skipLog {
			qo.logSlowQuery(table, operation, sql, duration, rowsAffected)
		}
	}
}

// logSlowQuery 记录慢查询日志
func (qo *QueryOptimizer) logSlowQuery(table, operation, sql string, duration time.Duration, rowsAffected int64) {
	// 获取调用上下文
	callContext := getCallerInfo()

	logEntry := SlowQueryLog{
		Timestamp:    time.Now(),
		Table:        table,
		Operation:    operation,
		SQL:          sql,
		Duration:     duration,
		RowsAffected: rowsAffected,
		Context:      callContext,
	}

	qo.mu.Lock()
	qo.slowQueryLogs = append(qo.slowQueryLogs, logEntry)
	// 限制日志条目数量
	if len(qo.slowQueryLogs) > qo.maxLogEntries {
		qo.slowQueryLogs = qo.slowQueryLogs[len(qo.slowQueryLogs)-qo.maxLogEntries:]
	}
	qo.mu.Unlock()

	// 记录到日志系统
	loggermm.Warnf("慢查询检测 [表:%s 操作:%s 耗时:%v 影响行数:%d]: %s (位置: %s)",
		table, operation, duration, rowsAffected, truncateSQL(sql), callContext)
}

// GetMetrics 获取查询指标
func (qo *QueryOptimizer) GetMetrics() QueryMetrics {
	qo.queryMetrics.mu.RLock()
	defer qo.queryMetrics.mu.RUnlock()

	return QueryMetrics{
		TotalQueries:    qo.queryMetrics.TotalQueries,
		SlowQueries:     qo.queryMetrics.SlowQueries,
		TotalDuration:   qo.queryMetrics.TotalDuration,
		AverageDuration: qo.queryMetrics.AverageDuration,
		MinDuration:     qo.queryMetrics.MinDuration,
		MaxDuration:     qo.queryMetrics.MaxDuration,
		ByTable:         qo.copyTableMetrics(),
		ByOperation:     qo.copyOperationMetrics(),
	}
}

// copyTableMetrics 复制表级指标
func (qo *QueryOptimizer) copyTableMetrics() map[string]*TableQueryMetrics {
	result := make(map[string]*TableQueryMetrics)
	for k, v := range qo.queryMetrics.ByTable {
		result[k] = &TableQueryMetrics{
			QueryCount:      v.QueryCount,
			TotalDuration:   v.TotalDuration,
			AverageDuration: v.AverageDuration,
			SlowCount:       v.SlowCount,
		}
	}
	return result
}

// copyOperationMetrics 复制操作级指标
func (qo *QueryOptimizer) copyOperationMetrics() map[string]*OperationQueryMetrics {
	result := make(map[string]*OperationQueryMetrics)
	for k, v := range qo.queryMetrics.ByOperation {
		result[k] = &OperationQueryMetrics{
			QueryCount:      v.QueryCount,
			TotalDuration:   v.TotalDuration,
			AverageDuration: v.AverageDuration,
			SlowCount:       v.SlowCount,
		}
	}
	return result
}

// GetSlowQueries 获取慢查询日志
func (qo *QueryOptimizer) GetSlowQueries(limit int) []SlowQueryLog {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	if limit <= 0 || limit > len(qo.slowQueryLogs) {
		return qo.slowQueryLogs
	}
	return qo.slowQueryLogs[len(qo.slowQueryLogs)-limit:]
}

// GetSlowQueryStats 获取慢查询统计
func (qo *QueryOptimizer) GetSlowQueryStats() map[string]interface{} {
	qo.queryMetrics.mu.RLock()
	defer qo.queryMetrics.mu.RUnlock()

	totalQueries := qo.queryMetrics.TotalQueries
	slowQueries := qo.queryMetrics.SlowQueries

	var slowRatio float64
	if totalQueries > 0 {
		slowRatio = float64(slowQueries) / float64(totalQueries) * 100
	}

	// 找出最慢的表
	var slowestTable string
	var slowestTableTime time.Duration
	var slowestTableQueries int64

	for table, metrics := range qo.queryMetrics.ByTable {
		if metrics.AverageDuration > slowestTableTime {
			slowestTableTime = metrics.AverageDuration
			slowestTable = table
			slowestTableQueries = metrics.QueryCount
		}
	}

	// 找出最慢的操作
	var slowestOp string
	var slowestOpTime time.Duration

	for op, metrics := range qo.queryMetrics.ByOperation {
		if metrics.AverageDuration > slowestOpTime {
			slowestOpTime = metrics.AverageDuration
			slowestOp = op
		}
	}

	return map[string]interface{}{
		"total_queries":      totalQueries,
		"slow_queries":       slowQueries,
		"slow_ratio":         fmt.Sprintf("%.2f%%", slowRatio),
		"slowest_table":      slowestTable,
		"slowest_table_time": slowestTableTime.String(),
		"slowest_table_qps":  slowestTableQueries,
		"slowest_operation":  slowestOp,
		"slowest_op_time":    slowestOpTime.String(),
	}
}

// cleanupLogs 定期清理旧日志
func (qo *QueryOptimizer) cleanupLogs() {
	defer qo.wg.Done()
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			qo.mu.Lock()
			if len(qo.slowQueryLogs) > qo.maxLogEntries/2 {
				// 保留最近的一半日志
				keepCount := qo.maxLogEntries / 2
				if keepCount < len(qo.slowQueryLogs) {
					qo.slowQueryLogs = qo.slowQueryLogs[len(qo.slowQueryLogs)-keepCount:]
				}
			}
			qo.mu.Unlock()
		case <-qo.stopCh:
			return
		}
	}
}

// Stop 停止优化器
func (qo *QueryOptimizer) Stop() {
	close(qo.stopCh)
	qo.wg.Wait()
}

// GetSlowThreshold 获取慢查询阈值
func (qo *QueryOptimizer) GetSlowThreshold() time.Duration {
	return qo.slowThreshold
}

// truncateSQL 截断过长的SQL语句
func truncateSQL(sql string) string {
	const maxLength = 200
	if len(sql) <= maxLength {
		return sql
	}
	return sql[:maxLength] + "..."
}

// getCallerInfo 获取调用者信息
func getCallerInfo() string {
	// 跳过我自己的调用，获取上层调用者
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}
	fn := runtime.FuncForPC(pc)
	return fmt.Sprintf("%s:%d (%s)", file, line, fn.Name())
}

// QueryWithMetrics 带指标的查询执行
func (qo *QueryOptimizer) QueryWithMetrics(ctx context.Context, table, operation, sql string, rowsAffected int64, queryFunc func() error) error {
	start := time.Now()
	err := queryFunc()
	duration := time.Since(start)

	// 记录查询（跳过日志，避免重复记录）
	qo.RecordQuery(table, operation, sql, duration, rowsAffected, false)

	return err
}
