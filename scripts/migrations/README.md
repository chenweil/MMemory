# 数据库迁移脚本

此目录包含MMemory项目的数据库迁移SQL脚本。

## 迁移列表

### 001 - Initial Schema (Phase 1)
初始数据库结构，包含基础表：
- users
- reminders
- reminder_logs
- conversations
- messages

### 002 - Add Indexes and Optimizations (Phase 2)
添加索引优化查询性能。

### 003 - Add Pause Fields (Phase C3)
**日期**: 2025-10-11
**阶段**: C3 - 关键问题修复与用户交互增强

添加暂停/恢复功能所需字段：
- `paused_until`: 暂停截止时间
- `pause_reason`: 暂停原因
- 索引: `idx_reminders_paused_until`

## 使用说明

### 手动执行迁移

```bash
# SQLite
sqlite3 /path/to/mmemory.db < scripts/migrations/003_add_pause_fields.sql

# 验证迁移
sqlite3 /path/to/mmemory.db "PRAGMA table_info(reminders);"
```

### 自动迁移

项目使用GORM的AutoMigrate功能，启动时会自动创建/更新表结构。手动迁移脚本仅用于：
1. 显式的数据库维护
2. 验证历史数据兼容性
3. 生产环境的谨慎升级

## 回滚

如需回滚迁移003：

```sql
-- 警告：这会删除所有暂停状态数据
ALTER TABLE reminders DROP COLUMN IF EXISTS paused_until;
ALTER TABLE reminders DROP COLUMN IF EXISTS pause_reason;
DROP INDEX IF EXISTS idx_reminders_paused_until;
```

## 验证

检查字段是否成功添加：

```sql
-- 查看表结构
PRAGMA table_info(reminders);

-- 检查暂停提醒统计
SELECT
    COUNT(*) as total,
    SUM(CASE WHEN paused_until IS NOT NULL THEN 1 ELSE 0 END) as has_pause_data,
    SUM(CASE WHEN paused_until > datetime('now') THEN 1 ELSE 0 END) as currently_paused
FROM reminders;
```

## 注意事项

1. **备份优先**: 执行任何迁移前请先备份数据库
2. **测试环境**: 先在测试环境验证迁移
3. **兼容性**: 新字段使用NULL默认值，确保历史数据兼容
4. **性能**: 大表迁移可能需要时间，建议在低峰期执行
