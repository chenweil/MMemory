-- MMemory 数据库性能优化脚本
-- 阶段2：B2 数据访问层优化
-- 创建时间：2025年9月29日

-- ============================================
-- 1. 索引优化
-- ============================================

-- 1.1 reminders 表索引优化
-- 为经常一起查询的字段创建复合索引
CREATE INDEX IF NOT EXISTS idx_reminders_user_active ON reminders (user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_reminders_schedule_pattern ON reminders (schedule_pattern);
CREATE INDEX IF NOT EXISTS idx_reminders_target_time ON reminders (target_time);
CREATE INDEX IF NOT EXISTS idx_reminders_created_at ON reminders (created_at);

-- 1.2 reminder_logs 表索引优化
-- 为高频查询字段创建索引
CREATE INDEX IF NOT EXISTS idx_reminder_logs_reminder_status ON reminder_logs (reminder_id, status);
CREATE INDEX IF NOT EXISTS idx_reminder_logs_scheduled_time ON reminder_logs (scheduled_time);
CREATE INDEX IF NOT EXISTS idx_reminder_logs_status_sent_time ON reminder_logs (status, sent_time);
CREATE INDEX IF NOT EXISTS idx_reminder_logs_created_at ON reminder_logs (created_at);

-- 1.3 users 表索引优化
-- 为状态查询和时间范围查询创建索引
CREATE INDEX IF NOT EXISTS idx_users_active_created ON users (is_active, created_at);
CREATE INDEX IF NOT EXISTS idx_users_language_code ON users (language_code);

-- ============================================
-- 2. 查询优化视图
-- ============================================

-- 2.1 用户提醒统计视图
CREATE VIEW IF NOT EXISTS user_reminder_stats AS
SELECT 
    u.id as user_id,
    u.telegram_id,
    u.username,
    COUNT(DISTINCT r.id) as total_reminders,
    COUNT(DISTINCT CASE WHEN r.is_active = true THEN r.id END) as active_reminders,
    COUNT(DISTINCT CASE WHEN r.type = 'habit' THEN r.id END) as habit_reminders,
    COUNT(DISTINCT CASE WHEN r.type = 'task' THEN r.id END) as task_reminders,
    MAX(r.created_at) as last_reminder_created
FROM users u
LEFT JOIN reminders r ON u.id = r.user_id
GROUP BY u.id, u.telegram_id, u.username;

-- 2.2 提醒执行统计视图
CREATE VIEW IF NOT EXISTS reminder_execution_stats AS
SELECT 
    r.id as reminder_id,
    r.title,
    r.user_id,
    COUNT(DISTINCT rl.id) as total_executions,
    COUNT(DISTINCT CASE WHEN rl.status = 'completed' THEN rl.id END) as completed_count,
    COUNT(DISTINCT CASE WHEN rl.status = 'skipped' THEN rl.id END) as skipped_count,
    COUNT(DISTINCT CASE WHEN rl.status = 'pending' THEN rl.id END) as pending_count,
    AVG(CASE WHEN rl.response_time IS NOT NULL AND rl.sent_time IS NOT NULL 
        THEN (julianday(rl.response_time) - julianday(rl.sent_time)) * 24 * 60 
        ELSE NULL END) as avg_response_minutes,
    MAX(rl.scheduled_time) as last_scheduled
FROM reminders r
LEFT JOIN reminder_logs rl ON r.id = rl.reminder_id
GROUP BY r.id, r.title, r.user_id;

-- 2.3 近期活跃提醒视图
CREATE VIEW IF NOT EXISTS recent_active_reminders AS
SELECT 
    r.id,
    r.title,
    r.description,
    r.user_id,
    u.telegram_id,
    r.type,
    r.schedule_pattern,
    r.target_time,
    r.timezone,
    r.created_at,
    CASE 
        WHEN r.schedule_pattern = 'daily' THEN '每日提醒'
        WHEN r.schedule_pattern LIKE 'weekly:%' THEN '每周提醒'
        WHEN r.schedule_pattern LIKE 'monthly:%' THEN '每月提醒'
        WHEN r.schedule_pattern LIKE 'once:%' THEN '一次性提醒'
        ELSE '未知类型'
    END as schedule_type_desc
FROM reminders r
JOIN users u ON r.user_id = u.id
WHERE r.is_active = true
AND r.created_at >= datetime('now', '-30 days')
ORDER BY r.created_at DESC;

-- ============================================
-- 3. 性能监控表
-- ============================================

-- 3.1 查询性能日志表
CREATE TABLE IF NOT EXISTS query_performance_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    query_type VARCHAR(50) NOT NULL,
    table_name VARCHAR(50) NOT NULL,
    execution_time_ms INTEGER NOT NULL,
    query_plan TEXT,
    parameters TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_query_log_created (created_at),
    INDEX idx_query_log_type (query_type, table_name)
);

-- 3.2 索引使用统计表
CREATE TABLE IF NOT EXISTS index_usage_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    index_name VARCHAR(100) NOT NULL,
    table_name VARCHAR(50) NOT NULL,
    usage_count INTEGER DEFAULT 0,
    last_used DATETIME,
    avg_query_time_ms INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_index_usage (index_name, table_name)
);

-- ============================================
-- 4. 数据清理和维护
-- ============================================

-- 4.1 清理过期的一次性提醒（30天前已完成的一次性提醒）
DELETE FROM reminders 
WHERE schedule_pattern LIKE 'once:%' 
AND is_active = false 
AND updated_at < datetime('now', '-30 days');

-- 4.2 清理旧的提醒记录（保留最近90天的记录）
DELETE FROM reminder_logs 
WHERE scheduled_time < datetime('now', '-90 days')
AND status IN ('completed', 'skipped');

-- 4.3 清理旧的查询性能日志（保留最近7天的日志）
DELETE FROM query_performance_log 
WHERE created_at < datetime('now', '-7 days');

-- ============================================
-- 5. 性能验证查询
-- ============================================

-- 5.1 验证索引使用情况
SELECT 
    name,
    type,
    tbl_name,
    sql
FROM sqlite_master 
WHERE type = 'index' 
AND tbl_name IN ('reminders', 'reminder_logs', 'users')
ORDER BY tbl_name, name;

-- 5.2 检查表大小和索引大小
SELECT 
    name,
    type,
    COUNT(*) as count
FROM sqlite_master 
WHERE type IN ('table', 'index')
AND name NOT LIKE 'sqlite_%'
GROUP BY type
ORDER BY type;

-- 5.3 验证视图创建
SELECT 
    name,
    type,
    sql
FROM sqlite_master 
WHERE type = 'view'
ORDER BY name;

-- ============================================
-- 6. 性能测试查询
-- ============================================

-- 6.1 测试用户提醒查询性能
EXPLAIN QUERY PLAN
SELECT r.*, u.telegram_id, u.username
FROM reminders r
JOIN users u ON r.user_id = u.id
WHERE r.user_id = 123 AND r.is_active = true
ORDER BY r.created_at DESC;

-- 6.2 测试提醒记录查询性能
EXPLAIN QUERY PLAN
SELECT rl.*, r.title, r.description
FROM reminder_logs rl
JOIN reminders r ON rl.reminder_id = r.id
WHERE rl.reminder_id = 456 AND rl.status = 'pending'
ORDER BY rl.scheduled_time DESC;

-- ============================================
-- 7. 后续优化建议
-- ============================================

-- 7.1 考虑分区表（当数据量很大时）
-- 可以按月份对 reminder_logs 表进行分区

-- 7.2 考虑读写分离
-- 读操作用只读副本，写操作用主库

-- 7.3 考虑连接池优化
-- 调整数据库连接池参数

-- 7.4 考虑缓存策略
-- 对热点数据实施多级缓存

PRINT '✅ 数据库性能优化脚本执行完成';
PRINT '请检查执行结果并验证性能提升';