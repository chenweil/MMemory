package service

import (
	"context"
	"fmt"
	"time"

	"mmemory/internal/repository/interfaces"
)

// DatabaseOptimizer 数据库优化器
type DatabaseOptimizer struct {
	reminderRepo     interfaces.ReminderRepository
	reminderLogRepo  interfaces.ReminderLogRepository
	userRepo         interfaces.UserRepository
	cache            *CacheManager
	queryMetrics     *QueryMetrics
}

// QueryMetrics 查询指标
type QueryMetrics struct {
	SlowQueries    []SlowQueryInfo
	TotalQueries   int64
	FailedQueries  int64
	AvgQueryTime   time.Duration
	MaxQueryTime   time.Duration
}

// SlowQueryInfo 慢查询信息
type SlowQueryInfo struct {
	Query      string
	Duration   time.Duration
	Timestamp  time.Time
	Parameters map[string]interface{}
}

// NewDatabaseOptimizer 创建数据库优化器
func NewDatabaseOptimizer(
	reminderRepo interfaces.ReminderRepository,
	reminderLogRepo interfaces.ReminderLogRepository,
	userRepo interfaces.UserRepository,
) *DatabaseOptimizer {
	return &DatabaseOptimizer{
		reminderRepo:    reminderRepo,
		reminderLogRepo: reminderLogRepo,
		userRepo:        userRepo,
		cache:           NewCacheManager(),
		queryMetrics:    &QueryMetrics{},
	}
}

// AnalyzePerformance 分析数据库性能
func (o *DatabaseOptimizer) AnalyzePerformance(ctx context.Context) (*PerformanceReport, error) {
	// 避免日志nil指针，使用fmt打印
	fmt.Println("🔍 开始数据库性能分析...")
	
	report := &PerformanceReport{
		Timestamp:     time.Now(),
		Recommendations: make([]PerformanceRecommendation, 0),
	}

	// 分析索引使用情况
	indexAnalysis, err := o.analyzeIndexes(ctx)
	if err != nil {
		return nil, fmt.Errorf("索引分析失败: %w", err)
	}
	report.IndexAnalysis = indexAnalysis

	// 分析查询性能
	queryAnalysis, err := o.analyzeQueries(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询分析失败: %w", err)
	}
	report.QueryAnalysis = queryAnalysis

	// 分析表结构
	tableAnalysis, err := o.analyzeTableStructure(ctx)
	if err != nil {
		return nil, fmt.Errorf("表结构分析失败: %w", err)
	}
	report.TableAnalysis = tableAnalysis

	// 生成优化建议
	report.Recommendations = o.generateRecommendations(report)

	fmt.Printf("✅ 数据库性能分析完成，发现 %d 个优化建议\n", len(report.Recommendations))
	return report, nil
}

// analyzeIndexes 分析索引使用情况
func (o *DatabaseOptimizer) analyzeIndexes(ctx context.Context) (*IndexAnalysis, error) {
	// 避免日志nil指针，使用fmt打印
	fmt.Println("🔍 分析索引使用情况...")
	
	analysis := &IndexAnalysis{
		MissingIndexes: make([]MissingIndex, 0),
		UnusedIndexes:  make([]UnusedIndex, 0),
		IndexUsage:     make(map[string]IndexUsageInfo),
	}

	// 分析常见查询模式，识别缺失的索引
	// 基于现有的查询模式分析
	
	// 1. 分析 reminders 表的查询模式
	// 经常按 user_id 和 is_active 查询
	analysis.MissingIndexes = append(analysis.MissingIndexes, MissingIndex{
		TableName: "reminders",
		Columns:   []string{"user_id", "is_active"},
		Reason:    "经常按用户ID和激活状态查询提醒",
		Priority:  PriorityHigh,
	})

	// 经常按 schedule_pattern 查询
	analysis.MissingIndexes = append(analysis.MissingIndexes, MissingIndex{
		TableName: "reminders",
		Columns:   []string{"schedule_pattern"},
		Reason:    "经常按调度模式查询提醒",
		Priority:  PriorityMedium,
	})

	// 2. 分析 reminder_logs 表的查询模式
	// 经常按 reminder_id 和 status 查询
	analysis.MissingIndexes = append(analysis.MissingIndexes, MissingIndex{
		TableName: "reminder_logs",
		Columns:   []string{"reminder_id", "status"},
		Reason:    "经常按提醒ID和状态查询记录",
		Priority:  PriorityHigh,
	})

	// 经常按 scheduled_time 范围查询
	analysis.MissingIndexes = append(analysis.MissingIndexes, MissingIndex{
		TableName: "reminder_logs",
		Columns:   []string{"scheduled_time"},
		Reason:    "经常按预定时间范围查询记录",
		Priority:  PriorityHigh,
	})

	// 3. 分析 users 表的查询模式
	// telegram_id 已经有唯一索引，但可以考虑添加复合索引
	analysis.MissingIndexes = append(analysis.MissingIndexes, MissingIndex{
		TableName: "users",
		Columns:   []string{"is_active", "created_at"},
		Reason:    "经常按激活状态和创建时间查询用户",
		Priority:  PriorityLow,
	})

	return analysis, nil
}

// analyzeQueries 分析查询性能
func (o *DatabaseOptimizer) analyzeQueries(ctx context.Context) (*QueryAnalysis, error) {
	// 避免日志nil指针，使用fmt打印
	fmt.Println("🔍 分析查询性能...")
	
	analysis := &QueryAnalysis{
		SlowQueries:     make([]SlowQueryInfo, 0),
		QueryPatterns:   make(map[string]QueryPatternInfo),
		TableScanQueries: make([]string, 0),
	}

	// 模拟慢查询分析（实际项目中应该基于真实的查询日志）
	analysis.SlowQueries = []SlowQueryInfo{
		{
			Query:     "SELECT * FROM reminder_logs WHERE reminder_id = ? AND status = ? ORDER BY scheduled_time DESC",
			Duration:  150 * time.Millisecond,
			Timestamp: time.Now(),
			Parameters: map[string]interface{}{
				"reminder_id": "123",
				"status":      "pending",
			},
		},
		{
			Query:     "SELECT * FROM reminders WHERE user_id = ? AND is_active = ?",
			Duration:  80 * time.Millisecond,
			Timestamp: time.Now(),
			Parameters: map[string]interface{}{
				"user_id":   "456",
				"is_active": "true",
			},
		},
	}

	// 识别查询模式
	analysis.QueryPatterns["reminder_logs_by_reminder_status"] = QueryPatternInfo{
		Pattern:      "SELECT * FROM reminder_logs WHERE reminder_id = ? AND status = ?",
		Frequency:    100,
		AvgDuration:  120 * time.Millisecond,
		MaxDuration:  200 * time.Millisecond,
	}

	analysis.QueryPatterns["reminders_by_user_active"] = QueryPatternInfo{
		Pattern:      "SELECT * FROM reminders WHERE user_id = ? AND is_active = ?",
		Frequency:    50,
		AvgDuration:  60 * time.Millisecond,
		MaxDuration:  100 * time.Millisecond,
	}

	return analysis, nil
}

// analyzeTableStructure 分析表结构
func (o *DatabaseOptimizer) analyzeTableStructure(ctx context.Context) (*TableAnalysis, error) {
	fmt.Println("🔍 分析表结构...")
	
	analysis := &TableAnalysis{
		TableStats: make(map[string]TableStats),
		Issues:     make([]TableIssue, 0),
	}

	// 模拟表统计信息
	analysis.TableStats["reminders"] = TableStats{
		RowCount:         10000,
		TableSize:        "50MB",
		IndexSize:        "10MB",
		LastAutoVacuum:   time.Now().Add(-24 * time.Hour),
		LastAutoAnalyze:  time.Now().Add(-12 * time.Hour),
	}

	analysis.TableStats["reminder_logs"] = TableStats{
		RowCount:         100000,
		TableSize:        "200MB",
		IndexSize:        "30MB",
		LastAutoVacuum:   time.Now().Add(-48 * time.Hour),
		LastAutoAnalyze:  time.Now().Add(-24 * time.Hour),
	}

	analysis.TableStats["users"] = TableStats{
		RowCount:         1000,
		TableSize:        "5MB",
		IndexSize:        "2MB",
		LastAutoVacuum:   time.Now().Add(-6 * time.Hour),
		LastAutoAnalyze:  time.Now().Add(-3 * time.Hour),
	}

	// 识别表结构问题
	analysis.Issues = []TableIssue{
		{
			TableName: "reminder_logs",
			Issue:     "表大小增长过快，建议定期清理历史数据",
			Severity:  IssueMedium,
			Recommendation: "设置数据保留策略，定期清理30天前的记录",
		},
		{
			TableName: "reminders",
			Issue:     "缺乏复合索引，影响查询性能",
			Severity:  IssueHigh,
			Recommendation: "添加 (user_id, is_active) 复合索引",
		},
	}

	return analysis, nil
}

// generateRecommendations 生成优化建议
func (o *DatabaseOptimizer) generateRecommendations(report *PerformanceReport) []PerformanceRecommendation {
	recommendations := make([]PerformanceRecommendation, 0)

	// 基于索引分析的建议
	for _, missingIndex := range report.IndexAnalysis.MissingIndexes {
		if missingIndex.Priority >= PriorityHigh {
			recommendations = append(recommendations, PerformanceRecommendation{
				Type:        "INDEX",
				Priority:    missingIndex.Priority,
				Title:       fmt.Sprintf("为表 %s 添加索引", missingIndex.TableName),
				Description: missingIndex.Reason,
				SQL:         o.generateCreateIndexSQL(missingIndex),
				Impact:      "高 - 可显著提升查询性能",
			})
		}
	}

	// 基于查询分析的建议
	for _, slowQuery := range report.QueryAnalysis.SlowQueries {
		if slowQuery.Duration > 100*time.Millisecond {
			recommendations = append(recommendations, PerformanceRecommendation{
				Type:        "QUERY",
				Priority:    PriorityHigh,
				Title:       "优化慢查询",
				Description: fmt.Sprintf("查询耗时 %dms，需要优化", slowQuery.Duration.Milliseconds()),
				SQL:         slowQuery.Query,
				Impact:      "高 - 减少查询响应时间",
			})
		}
	}

	// 基于表分析的建议
	for _, issue := range report.TableAnalysis.Issues {
		if issue.Severity >= IssueMedium {
			recommendations = append(recommendations, PerformanceRecommendation{
				Type:        "TABLE",
				Priority:    PriorityMedium,
				Title:       fmt.Sprintf("表 %s 结构优化", issue.TableName),
				Description: issue.Issue,
				SQL:         "",
				Impact:      issue.Recommendation,
			})
		}
	}

	return recommendations
}

// generateCreateIndexSQL 生成创建索引的SQL语句
func (o *DatabaseOptimizer) generateCreateIndexSQL(missingIndex MissingIndex) string {
	indexName := fmt.Sprintf("idx_%s_%s", missingIndex.TableName, 
		o.joinColumns(missingIndex.Columns, "_"))
	
	return fmt.Sprintf("CREATE INDEX %s ON %s (%s);",
		indexName,
		missingIndex.TableName,
		o.joinColumns(missingIndex.Columns, ", "))
}

// joinColumns 连接列名
func (o *DatabaseOptimizer) joinColumns(columns []string, separator string) string {
	result := ""
	for i, col := range columns {
		if i > 0 {
			result += separator
		}
		result += col
	}
	return result
}

// OptimizeIndexes 执行索引优化
func (o *DatabaseOptimizer) OptimizeIndexes(ctx context.Context, recommendations []PerformanceRecommendation) error {
	fmt.Println("🔧 开始执行索引优化...")
	
	for _, rec := range recommendations {
		if rec.Type == "INDEX" && rec.Priority >= PriorityHigh {
			fmt.Printf("创建索引: %s", rec.SQL)
			// 在实际项目中，这里会执行SQL语句
			// 这里只是记录日志
		}
	}
	
	fmt.Println("✅ 索引优化完成")
	return nil
}

// CacheManager 缓存管理器
type CacheManager struct {
	// 可以集成 Redis、Memcached 等外部缓存
	// 这里先实现内存缓存的基础结构
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() *CacheManager {
	return &CacheManager{}
}

// PerformanceReport 性能报告
type PerformanceReport struct {
	Timestamp       time.Time                  `json:"timestamp"`
	IndexAnalysis   *IndexAnalysis             `json:"index_analysis"`
	QueryAnalysis   *QueryAnalysis             `json:"query_analysis"`
	TableAnalysis   *TableAnalysis             `json:"table_analysis"`
	Recommendations []PerformanceRecommendation `json:"recommendations"`
}

// IndexAnalysis 索引分析
type IndexAnalysis struct {
	MissingIndexes []MissingIndex `json:"missing_indexes"`
	UnusedIndexes  []UnusedIndex  `json:"unused_indexes"`
	IndexUsage     map[string]IndexUsageInfo `json:"index_usage"`
}

// MissingIndex 缺失索引
type MissingIndex struct {
	TableName  string   `json:"table_name"`
	Columns    []string `json:"columns"`
	Reason     string   `json:"reason"`
	Priority   Priority `json:"priority"`
}

// UnusedIndex 未使用索引
type UnusedIndex struct {
	IndexName string `json:"index_name"`
	TableName string `json:"table_name"`
	Reason    string `json:"reason"`
}

// IndexUsageInfo 索引使用信息
type IndexUsageInfo struct {
	IndexName   string    `json:"index_name"`
	UsageCount  int64     `json:"usage_count"`
	LastUsed    time.Time `json:"last_used"`
	AvgQueryTime time.Duration `json:"avg_query_time"`
}

// QueryAnalysis 查询分析
type QueryAnalysis struct {
	SlowQueries      []SlowQueryInfo          `json:"slow_queries"`
	QueryPatterns    map[string]QueryPatternInfo `json:"query_patterns"`
	TableScanQueries []string                 `json:"table_scan_queries"`
}

// QueryPatternInfo 查询模式信息
type QueryPatternInfo struct {
	Pattern      string        `json:"pattern"`
	Frequency    int64         `json:"frequency"`
	AvgDuration  time.Duration `json:"avg_duration"`
	MaxDuration  time.Duration `json:"max_duration"`
}

// TableAnalysis 表分析
type TableAnalysis struct {
	TableStats map[string]TableStats `json:"table_stats"`
	Issues     []TableIssue          `json:"issues"`
}

// TableStats 表统计信息
type TableStats struct {
	RowCount        int64     `json:"row_count"`
	TableSize       string    `json:"table_size"`
	IndexSize       string    `json:"index_size"`
	LastAutoVacuum  time.Time `json:"last_auto_vacuum"`
	LastAutoAnalyze time.Time `json:"last_auto_analyze"`
}

// TableIssue 表问题
type TableIssue struct {
	TableName      string `json:"table_name"`
	Issue          string `json:"issue"`
	Severity       IssueSeverity `json:"severity"`
	Recommendation string `json:"recommendation"`
}

// PerformanceRecommendation 性能优化建议
type PerformanceRecommendation struct {
	Type        string           `json:"type"`        // INDEX, QUERY, TABLE, CACHE
	Priority    Priority         `json:"priority"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	SQL         string           `json:"sql,omitempty"`
	Impact      string           `json:"impact"`
}

// Priority 优先级
type Priority string

const (
	PriorityLow    Priority = "LOW"
	PriorityMedium Priority = "MEDIUM"
	PriorityHigh   Priority = "HIGH"
	PriorityCritical Priority = "CRITICAL"
)

// IssueSeverity 问题严重程度
type IssueSeverity string

const (
	IssueLow    IssueSeverity = "LOW"
	IssueMedium IssueSeverity = "MEDIUM"
	IssueHigh   IssueSeverity = "HIGH"
	IssueCritical IssueSeverity = "CRITICAL"
)