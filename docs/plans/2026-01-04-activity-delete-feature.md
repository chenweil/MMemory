# 活动记录删除功能实现计划

**创建时间**: 2026-01-04
**优先级**: 高
**复杂度**: 中等

## 问题描述

用户反馈在查询活动记录（如"我看过哪些书？"）后，发现��含错误记录（如《？》），但无法通过自然语言指令删除这些记录。

**对话示例**:
```
用户: 我看过哪些书？
Bot: 根据记录，你最近看过：
- 《？》（读到未知章节）
- 《心智简史》（读到第10章）
- 《书》（读到第二章）
- 《如何阅读一本书》（读到11章）

用户: 请把《？》这条记录删除。
Bot: [无法理解删除指令]
```

## 根本原因分析

### 1. **AI Prompt 缺少删除意图**
当前 AI Prompt (`pkg/ai/config.go`) 支持：
- `record_activity` - 记录活动
- `query_activity` - 查询活动
- **缺少**: `delete_activity` - 删除活动记录的意图定义

### 2. **Intent 枚举不完整**
`pkg/ai/types.go` 中的 Intent 枚举缺少删除活动的常量定义。

### 3. **Parser 缺少删除模式**
正则解析器 (`internal/ai/regex_parser.go`) 没有识别删除活动指令的模式。

### 4. **Service 层缺少删除方法**
`DailyActivityService` 和 `DailyActivityRepository` 虽然有 `Delete` 方法，但没有基于业务逻辑的删除（如根据书名删除、根据日期删除）。

## 技术方���设计

### 方案概述

扩展 AI 系统以支持删除活动记录，包括：
1. **AI Prompt 增强** - 添加删除活动意图的识别规则
2. **Intent 类型扩展** - 新增 `IntentDeleteActivity` 常量
3. **ParseResult 结构扩展** - 添加 `DeleteActivity` 字段
4. **正则解析器增强** - 添加删除活动的模式匹配
5. **Service 方法增强** - 实现智能删除逻辑（书名匹配、日期范围等）

### 架构变更

```
用户消息: "把《？》这条记录删除"
    ↓
[Regex Parser / AI Parser]
    ↓ 识别意图
Intent: delete_activity
    ↓ 提取参数
DeleteActivity: {
    activity_type: "read_book",
    criteria: {
        book_name: "？"
    }
}
    ↓
[Message Handler]
    ↓
[DailyActivityService.DeleteActivities]
    ↓ 根据criteria删除
[Repository.Delete] → Database
    ↓
返回删除结果: "已删除《？》的阅读记录"
```

## 实现步骤

### 阶段 1: 类型定义扩展

**文件**: `pkg/ai/types.go`

**任务**:
1. 添加 `IntentDeleteActivity` 常量
2. 添加 `DeleteActivityInfo` 结构体
3. 在 `ParseResult` 中添加 `DeleteActivity` 字段

**代码变更**:
```go
// Intent 意图类型
const (
    IntentReminder        Intent = "reminder"
    IntentDelete          Intent = "delete"
    IntentEdit            Intent = "edit"
    // ... 其他意图
    IntentDeleteActivity  Intent = "delete_activity"  // 新增
)

// DeleteActivityInfo 删除活动信息
type DeleteActivityInfo struct {
    ActivityType string                 `json:"activity_type"` // read_book, drink_water, etc.
    Criteria     map[string]interface{} `json:"criteria"`      // 删除条件：book_name, time_range等
}

// ParseResult 解析结果
type ParseResult struct {
    Intent            Intent                `json:"intent"`
    Confidence        float64               `json:"confidence"`
    // ... 其他字段
    DeleteActivity    *DeleteActivityInfo   `json:"delete_activity,omitempty"` // 新增
}
```

### 阶段 2: AI Prompt 增强

**文件**: `pkg/ai/config.go`

**任务**:
1. 在 `getDefaultReminderPrompt()` 中添加删除活动意图说明
2. 添加删除活动的 JSON 示例
3. 提供明确的识别规则

**Prompt 新增内容**:
```
12. 删除活动记录 (delete_activity) - 删除指定的活动记录
   **删除关键词**：删除、去掉、移除、清除、不要、错了
   ✅ **正确的删除示例**：
   - "把《？》这条记录删除" → activity_type="read_book", criteria={"book_name": "？"}
   - "删除昨天的喝水记录" → activity_type="drink_water", criteria={"time_range": "昨天"}
   - "清除《书》的阅读记录" → activity_type="read_book", criteria={"book_name": "书"}
   - "移除错误的那条运动记录" → activity_type="exercise", criteria={"is_error": true}
```

**JSON 格式**:
```json
{
  "intent": "delete_activity",
  "confidence": 0.90,
  "delete_activity": {
    "activity_type": "read_book",
    "criteria": {
      "book_name": "？"
    }
  }
}
```

### 阶段 3: 正则解析器增强

**文件**: `internal/ai/regex_parser.go`

**任务**:
1. 在 `checkQueryPatterns()` 后添加 `checkDeleteActivityPatterns()` 方法
2. 在 `ParseWithContext()` 中调用删除模式检查

**新增模式**:
```go
// checkDeleteActivityPatterns 检查删除活动模式
func (p *RegexParser) checkDeleteActivityPatterns(message string) *ai.ParseResult {
    // 删除书籍记录
    deleteBookPattern := regexp.MustCompile(`(?:删除|去掉|移除|清除|不要).*?(?:《(.+?)》|"(.+?)"|书名[是为][:是](.+?)).*?(?:记录|这条)`)
    if matches := deleteBookPattern.FindStringSubmatch(message); len(matches) > 0 {
        var bookName string
        for i := 1; i < len(matches); i++ {
            if matches[i] != "" {
                bookName = strings.TrimSpace(matches[i])
                break
            }
        }
        if bookName != "" {
            logger.Infof("Regex matched delete_activity for book: %s", bookName)
            return &ai.ParseResult{
                Intent:     ai.IntentDeleteActivity,
                Confidence: 0.85,
                DeleteActivity: &ai.DeleteActivityInfo{
                    ActivityType: "read_book",
                    Criteria: map[string]interface{}{
                        "book_name": bookName,
                    },
                },
                ParsedBy:    p.GetName(),
                ProcessTime: 0,
                Timestamp:   time.Now(),
            }
        }
    }

    // 删除喝水记录
    if matched, _ := regexp.MatchString(`(?:删除|清除|去掉).*(水|喝水).*记录`, message); matched {
        return &ai.ParseResult{
            Intent:     ai.IntentDeleteActivity,
            Confidence: 0.85,
            DeleteActivity: &ai.DeleteActivityInfo{
                ActivityType: "drink_water",
                Criteria:     map[string]interface{}{},
            },
            ParsedBy:    p.GetName(),
            ProcessTime: 0,
            Timestamp:   time.Now(),
        }
    }

    return nil
}
```

### 阶段 4: Service 层增强

**文件**: `internal/service/daily_activity.go`

**任务**:
1. 添加 `DeleteActivities()` 方法到接口
2. 实现智能删除逻辑

**接口扩展**:
```go
type DailyActivityService interface {
    RecordActivity(ctx context.Context, userID uint, activityType models.ActivityType, details map[string]interface{}, source models.ActivitySource) (*models.DailyActivity, error)
    GetRecentActivities(ctx context.Context, userID uint, limit int) ([]*models.DailyActivity, error)
    // ... 其他方法
    DeleteActivities(ctx context.Context, userID uint, activityType models.ActivityType, criteria map[string]interface{}) (int, error) // 新增
}
```

**实现逻辑**:
```go
// DeleteActivities 根据条件删除活动记录
func (s *dailyActivityServiceImpl) DeleteActivities(ctx context.Context, userID uint, activityType models.ActivityType, criteria map[string]interface{}) (int, error) {
    // 获取所有匹配的活动
    activities, err := s.activityRepo.GetByType(ctx, userID, activityType, 100, 0)
    if err != nil {
        return 0, fmt.Errorf("查询活动失败: %w", err)
    }

    var toDelete []uint

    for _, activity := range activities {
        details, err := activity.GetDetails()
        if err != nil {
            continue
        }

        // 根据条件筛选
        if activityType == models.ActivityTypeReadBook {
            if bookName, ok := criteria["book_name"].(string); ok && details.BookName == bookName {
                toDelete = append(toDelete, activity.ID)
            }
        } else if activityType == models.ActivityTypeDrinkWater {
            // 删除所有喝水记录或按时间范围
            if timeRange, ok := criteria["time_range"].(string); ok {
                // 根据时间范围筛选
                startTime, endTime, _ := resolveActivityTimeRange(timeRange, time.Now())
                if activity.OccurredAt.After(startTime) && activity.OccurredAt.Before(endTime) {
                    toDelete = append(toDelete, activity.ID)
                }
            } else {
                // 删除所有
                toDelete = append(toDelete, activity.ID)
            }
        }
    }

    // 批量删除
    deleted := 0
    for _, id := range toDelete {
        if err := s.activityRepo.Delete(ctx, id); err == nil {
            deleted++
        }
    }

    logger.Infof("用户 %d 删除了 %d 条 %s 记录", userID, deleted, activityType)
    return deleted, nil
}
```

### 阶段 5: Bot Handler 处理

**文件**: `internal/bot/handlers/message.go`

**任务**:
1. 在 `handleParseResult()` 函数中添加 `delete_activity` 意图的处理分支
2. 生成友好的删除确认消息

**代码变更**:
```go
case ai.IntentDeleteActivity:
    if parseResult.DeleteActivity != nil {
        activityType := models.ActivityType(parseResult.DeleteActivity.ActivityType)
        criteria := parseResult.DeleteActivity.Criteria

        deleted, err := s.dailyActivityService.DeleteActivities(ctx, user.ID, activityType, criteria)
        if err != nil {
            return fmt.Sprintf("删除失败: %v", err)
        }

        if deleted > 0 {
            return fmt.Sprintf("已删除 %d 条记录", deleted)
        }
        return "没有找到匹配的记录"
    }
```

### 阶段 6: 单元测试

**文件**: 新建 `internal/service/daily_activity_delete_test.go`

**测试用例**:
1. 删除指定书名的阅读记录
2. 删除时间范围内的喝水记录
3. 删除不存在的记录返回0
4. 删除所有匹配的记录
5. 并发删除的安全性

**示例测试**:
```go
func TestDailyActivityService_DeleteActivities(t *testing.T) {
    // 1. 创建测试数据
    activity1 := &models.DailyActivity{
        UserID:       1,
        ActivityType: models.ActivityTypeReadBook,
        Details:      `{"book_name": "？", "chapter": "未知"}`,
        OccurredAt:   time.Now(),
    }
    repo.Create(ctx, activity1)

    activity2 := &models.DailyActivity{
        UserID:       1,
        ActivityType: models.ActivityTypeReadBook,
        Details:      `{"book_name": "如何阅读一本书", "chapter": "第10章"}`,
        OccurredAt:   time.Now(),
    }
    repo.Create(ctx, activity2)

    // 2. 执行删除
    criteria := map[string]interface{}{"book_name": "？"}
    deleted, err := service.DeleteActivities(ctx, 1, models.ActivityTypeReadBook, criteria)

    // 3. 验证结果
    assert.NoError(t, err)
    assert.Equal(t, 1, deleted)

    // 验证只剩下一条记录
    activities, _ := service.GetActivitiesByType(ctx, 1, models.ActivityTypeReadBook, 10)
    assert.Len(t, activities, 1)
    assert.Equal(t, "如何阅读一本书", activities[0].Details)
}
```

### 阶段 7: 集成测试

**文件**: 新建 `internal/bot/handlers/message_delete_activity_test.go`

**测试场景**:
1. 用户发送"把《？》这条记录删除"
2. 验证 AI 正确识别为 `delete_activity` 意图
3. 验证成功删除并返回确认消息

### 阶段 8: E2E 测试

**文件**: 新建 `test/e2e/activity_delete_e2e_test.go`

**测试流程**:
1. 创建测试用户
2. 记录几本书的阅读活动（包括错误的《？》）
3. 查询阅读记录
4. 发送删除指令
5. 验证删除成功
6. 再次查询验证记录已删除

## 依赖关系

```
阶段 1 (类型定义)
  ↓
阶段 2 (AI Prompt) ← 并行 → 阶段 3 (正则解析器)
  ↓
阶段 4 (Service 层)
  ↓
阶段 5 (Bot Handler)
  ↓
阶段 6-8 (测试)
```

## 风险评估

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| AI 误删除正确记录 | 高 | 中 | 添加删除确认机制，要求用户二次确认 |
| 正则匹配不准确 | 中 | 低 | 充分测试各种删除表达方式 |
| 并发删除冲突 | 低 | 低 | 使用数据库事务和锁机制 |
| 删除不可恢复 | 中 | 低 | 考虑添加软删除（标记deleted_at） |

## 后续优化

1. **软删除机制** - 添加 `deleted_at` 字段，支持误删恢复
2. **删除确认** - 对于多条记录删除，要求用户确认
3. **删除历史** - 记录删除操作日志，便于审计
4. **批量操作** - 支持更复杂的批量删除条件
5. **撤销功能** - 短时间内可以撤销删除操作

## 验收标准

- ✅ AI 能正确识别删除指令（成功率 > 90%）
- ✅ 正则解析器能匹配常见删除表达
- ✅ Service 层能根据条件删除记录
- ✅ 删除操作返回友好的确认消息
- ✅ 单元测试覆盖率 > 80%
- ✅ 集成测试和 E2E 测试全部通过
- ✅ 性能测试：删除 100 条记录耗时 < 1 秒

## 工作量评估

- 阶段 1-3: 2-3 小时
- 阶段 4-5: 2-3 小时
- 阶段 6-8: 3-4 小时
- **总计**: 7-10 小时

## 参考

- 现有删除提醒功能: `pkg/ai/config.go` Line 89-90
- 活动查询功能: `internal/service/daily_activity.go:QueryActivities()`
- DailyActivityRepository 接口: `internal/repository/interfaces/repository.go:64-75`
