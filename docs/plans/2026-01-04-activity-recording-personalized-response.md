# Activity Recording Personalized Response Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在用户记录日常活动(看书、喝水、运动等)时,生成AI个性化回复,提供友好交流体验,同时显示已记录的信息。

**Architecture:** 在活动记录成功后,调用AI服务生成个性化回复。使用专门的prompt模板,包含活动类型和详情,要求AI生成友好回复并包含"已记录"确认信息。AI失败时降级为简单确认消息。

**Tech Stack:** Go, OpenAI API, GORM, Telegram Bot API,现有的AI服务层架构

---

## Task 1: 在 AI 包中添加活动回复的 Prompt 模板

**Files:**
- Modify: `pkg/ai/config.go`
- Test: `pkg/ai/config_test.go` (如需要)

**Step 1: 在 AIConfig 结构中添加新字段**

在 `pkg/ai/config.go:26-30` 的 `PromptsConfig` 结构中添加新字段:

```go
type PromptsConfig struct {
	ReminderParse  string `mapstructure:"reminder_parse" yaml:"reminder_parse"`
	ChatResponse   string `mapstructure:"chat_response" yaml:"chat_response"`
	ActivityReply  string `mapstructure:"activity_reply" yaml:"activity_reply"` // 新增
}
```

**Step 2: 在 GetDefaultAIConfig 中设置默认值**

在 `pkg/ai/config.go:45-48` 添加:

```go
Prompts: PromptsConfig{
	ReminderParse:  getDefaultReminderPrompt(),
	ChatResponse:   getDefaultChatPrompt(),
	ActivityReply:  getDefaultActivityReplyPrompt(), // 新增
},
```

**Step 3: 实现 getDefaultActivityReplyPrompt 函数**

在 `pkg/ai/config.go:250` 之后(文件末尾)添加:

```go
// getDefaultActivityReplyPrompt 默认活动回复Prompt
func getDefaultActivityReplyPrompt() string {
	return `你是MMemory智能助手,用户刚刚记录了一个日常活动。请生成友好、个性化的回复。

用户消息: "{{.UserMessage}}"
活动类型: {{.ActivityType}}
活动详情: {{.Details}}

要求:
1. 生成友好的回复,像日常人类交流一样自然
2. 根据活动类型提供相关信息:
   - 看书: 可以提一下书籍背景、作者信息、阅读建议
   - 喝水: 鼓励健康习惯,提醒适量饮水
   - 运动: 肯定用户的努力,可以提运动的好处
   - 吃药: 提醒按时用药,祝早日康复
   - 其他: 给予积极回应
3. 必须包含"✅ 已记录: [活动类型]"的确认信息
4. 保持简洁,回复在50字以内
5. 如果是看书活动,可以简单提一两句关于这本书的信息
6. 不要过度展开,保持对话的开放性

示例:

用户消息: "我在看《时间简史》第二章"
活动类型: read_book
活动详情: {"book_name":"时间简史","chapter":"第二章"}
回复: "《时间简史》是霍金的经典科普著作,用通俗易懂的方式解释宇宙奥秘。✅ 已记录:看书-读到第二章"

用户消息: "我刚才喝了杯水"
活动类型: drink_water
活动详情: {"amount":"1杯"}
回复: "很好!保持适量饮水有助于新陈代谢。✅ 已记录:喝水"

用户消息: "今天跑了5公里"
活动类型: exercise
活动详情: {"type":"跑步","distance":"5公里"}
回复: "太棒了!5公里跑步是很好的有氧运动。✅ 已记录:运动-跑步5公里"

请直接返回回复内容,不需要JSON格式。`
}
```

**Step 4: 运行测试确保配置有效**

```bash
go test ./pkg/ai -run TestConfig -v
```

预期: 通过

**Step 5: 提交更改**

```bash
git add pkg/ai/config.go
git commit -m "feat: 添加活动回复 AI prompt 模板"
```

---

## Task 2: 在 AI 服务层实现活动回复生成方法

**Files:**
- Modify: `internal/service/interfaces.go`
- Modify: `internal/service/ai_parser.go`
- Test: `internal/service/ai_parser_test.go`

**Step 1: 在 AIParserService 接口中添加方法**

在 `internal/service/interfaces.go` 的 `AIParserService` 接口中添加方法(约在第60行附近):

```go
type AIParserService interface {
	ParseWithContext(ctx context.Context, userID string, message string, conversationHistory string) (*ai.ParseResult, error)
	IsHealthy() bool
	GetPriority() int
	GenerateActivityReply(ctx context.Context, userID string, userMessage string, activityType models.ActivityType, details map[string]interface{}) (string, error) // 新增
}
```

**Step 2: 在 aiParserServiceImpl 中实现方法**

在 `internal/service/ai_parser.go` 中,找到 `aiParserServiceImpl` 结构体,添加方法实现(约在文件末尾):

```go
// GenerateActivityReply 生成活动记录的个性化回复
func (s *aiParserServiceImpl) GenerateActivityReply(ctx context.Context, userID string, userMessage string, activityType models.ActivityType, details map[string]interface{}) (string, error) {
	// 检查AI服务是否启用
	if !s.config.Enabled {
		return "", fmt.Errorf("AI service not enabled")
	}

	// 序列化详情
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return "", fmt.Errorf("failed to marshal activity details: %w", err)
	}

	// 准备prompt变量
	promptVars := map[string]interface{}{
		"UserMessage":   userMessage,
		"ActivityType":  string(activityType),
		"Details":       string(detailsJSON),
	}

	// 使用chat prompt模板
	promptTemplate := s.config.Prompts.ActivityReply
	if promptTemplate == "" {
		promptTemplate = getDefaultActivityReplyPrompt()
	}

	// 渲染prompt
	prompt, err := renderPrompt(promptTemplate, promptVars)
	if err != nil {
		return "", fmt.Errorf("failed to render prompt: %w", err)
	}

	// 调用AI生成回复
	logger.Infof("正在为用户 %d 生成活动回复, 活动类型: %s", userID, activityType)

	response, err := s.aiClient.GenerateChatResponse(ctx, prompt, "")
	if err != nil {
		return "", fmt.Errorf("AI生成回复失败: %w", err)
	}

	// 清理回复内容
	reply := strings.TrimSpace(response)
	reply = strings.Trim(reply, "\"'`") // 移除可能的引号

	logger.Infof("成功为用户 %d 生成活动回复: %s", userID, reply)
	return reply, nil
}
```

**Step 3: 添加导入**

检查 `internal/service/ai_parser.go` 顶部的import,确保包含:

```go
import (
	"encoding/json" // 如果没有则添加
	"fmt"
	"strings"
	// ... 其他导入
)
```

**Step 4: 编译检查**

```bash
go build ./internal/service
```

预期: 无编译错误

**Step 5: 提交更改**

```bash
git add internal/service/interfaces.go internal/service/ai_parser.go
git commit -m "feat: 在AI服务中添加活动回复生成方法"
```

---

## Task 3: 修改消息处理器使用AI生成回复

**Files:**
- Modify: `internal/bot/handlers/message.go`
- Test: `internal/bot/handlers/message_ai_test.go`

**Step 1: 修改 handleRecordActivityIntent 函数**

在 `internal/bot/handlers/message.go` 中找到 `handleRecordActivityIntent` 函数(约743-778行),修改为:

```go
func (h *MessageHandler) handleRecordActivityIntent(bot *telegram.BotAPI, update tgbotapi.Update, parseResult *ai.ParseResult, activityType models.ActivityType, details map[string]interface{}) error {
	message := update.Message

	// 记录活动
	activity, err := h.activityService.RecordActivity(
		context.Background(),
		message.Chat.ID,
		activityType,
		details,
		models.SourceUser,
	)

	if err != nil {
		logger.Errorf("记录活动失败: %v", err)
		return h.sendMessage(bot, message.Chat.ID, "记录活动失败,请稍后重试")
	}

	logger.Infof("成功记录活动: 用户=%d, 类型=%s, ID=%d",
		message.Chat.ID, activityType, activity.ID)

	// 尝试使用AI生成个性化回复
	var reply string
	if h.aiParserService != nil && h.aiParserService.IsHealthy() {
		aiReply, err := h.aiParserService.GenerateActivityReply(
			context.Background(),
			message.Chat.ID,
			message.Text,
			activityType,
			details,
		)

		if err != nil {
			logger.Warnf("AI生成活动回复失败,使用简单确认: %v", err)
			// 降级为简单确认
			reply = fmt.Sprintf("✅ 已记录:%s", getActivityTypeDisplayName(activityType))
		} else {
			reply = aiReply
		}
	} else {
		// AI服务不可用,使用简单确认
		reply = fmt.Sprintf("✅ 已记录:%s", getActivityTypeDisplayName(activityType))
	}

	return h.sendMessage(bot, message.Chat.ID, reply)
}
```

**Step 2: 验证导入**

确保文件顶部有这些导入(约第10-30行):

```go
import (
	"context"
	"fmt"

	"mmemory/internal/models"
	"mmemory/pkg/ai"
)
```

**Step 3: 编译检查**

```bash
go build ./internal/bot/handlers
```

预期: 无编译错误

**Step 4: 提交更改**

```bash
git add internal/bot/handlers/message.go
git commit -m "feat: 活动记录使用AI生成个性化回复,失败时降级为简单确认"
```

---

## Task 4: 添加单元测试

**Files:**
- Create: `internal/service/ai_parser_activity_test.go`

**Step 1: 编写测试文件**

创建新文件 `internal/service/ai_parser_activity_test.go`:

```go
package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"mmemory/internal/models"
	"mmemory/pkg/ai"
	"mmemory/pkg/config"
)

// MockAIClient 用于测试
type MockAIClient struct {
	response string
	err      error
}

func (m *MockAIClient) GenerateResponse(ctx context.Context, prompt string, conversationHistory string) (*ai.ParseResult, error) {
	return nil, nil
}

func (m *MockAIClient) GenerateChatResponse(ctx context.Context, message string, conversationHistory string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestGenerateActivityReply(t *testing.T) {
	tests := []struct {
		name         string
		userMessage  string
		activityType models.ActivityType
		details      map[string]interface{}
		mockResponse string
		mockError    error
		wantErr      bool
	}{
		{
			name:         "看书活动-成功",
			userMessage:  "我在看《时间简史》第二章",
			activityType: models.ActivityTypeReadBook,
			details: map[string]interface{}{
				"book_name": "时间简史",
				"chapter":   "第二章",
			},
			mockResponse: "《时间简史》是霍金的经典科普著作。✅ 已记录:看书",
			mockError:    nil,
			wantErr:      false,
		},
		{
			name:         "喝水活动-成功",
			userMessage:  "我刚才喝了杯水",
			activityType: models.ActivityTypeDrinkWater,
			details: map[string]interface{}{
				"amount": "1杯",
			},
			mockResponse: "很好!保持适量饮水。✅ 已记录:喝水",
			mockError:    nil,
			wantErr:      false,
		},
		{
			name:         "AI服务失败-返回错误",
			userMessage:  "我在看书",
			activityType: models.ActivityTypeReadBook,
			details:      map[string]interface{}{},
			mockResponse: "",
			mockError:    fmt.Errorf("AI service unavailable"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建mock AI客户端
			mockAI := &MockAIClient{
				response: tt.mockResponse,
				err:      tt.mockError,
			}

			// 创建AI配置
			cfg := &config.AIConfig{
				Enabled: true,
				Prompts: ai.PromptsConfig{
					ActivityReply: ai.GetDefaultActivityReplyPrompt(),
				},
			}

			// 创建服务实例
			service := NewAIParserService(mockAI, cfg, nil)

			// 调用方法
			reply, err := service.GenerateActivityReply(
				context.Background(),
				123,
				tt.userMessage,
				tt.activityType,
				tt.details,
			)

			// 验证结果
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateActivityReply() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && reply != tt.mockResponse {
				t.Errorf("GenerateActivityReply() = %v, want %v", reply, tt.mockResponse)
			}
		})
	}
}
```

**Step 2: 运行测试**

```bash
go test ./internal/service -run TestGenerateActivityReply -v
```

预期: 测试通过

**Step 3: 提交测试**

```bash
git add internal/service/ai_parser_activity_test.go
git commit -m "test: 添加活动回复生成方法的单元测试"
```

---

## Task 5: 集成测试和手动验证

**Files:**
- Test: 实际运行bot进行测试
- Test: 查看日志验证AI调用

**Step 1: 构建新版本**

```bash
make build
```

预期: 编译成功,生成 `bin/mmemory`

**Step 2: 停止现有bot**

```bash
docker ps | grep mmemory
docker stop <container_id>
```

**Step 3: 启动新版本(本地测试)**

```bash
MMEMORY_AI_ENABLED=true \
MMEMORY_AI_OPENAI_API_KEY=your_api_key \
./bin/mmemory
```

**Step 4: 测试看书活动**

向bot发送消息:
```
我在看《如何阅读一本书》第十一章
```

预期回复: 类似于
```
《如何阅读一本书》是经典的阅读方法指南。✅ 已记录:看书-读到第十一章
```

**Step 5: 测试喝水活动**

发送消息:
```
我刚才喝了杯水
```

预期回复: 类似于
```
很好!保持适量饮水。✅ 已记录:喝水
```

**Step 6: 测试AI失败降级**

临时设置无效API key:
```bash
MMEMORY_AI_ENABLED=true \
MMEMORY_AI_OPENAI_API_KEY=invalid_key \
./bin/mmemory
```

发送活动记录消息,预期回复:
```
✅ 已记录:看书
```

**Step 7: 检查日志**

查看日志中是否有类似信息:
```
INFO 正在为用户 123 生成活动回复, 活动类型: read_book
INFO 成功为用户 123 生成活动回复: 《如何阅读一本书》是经典的...
```

**Step 8: 停止本地测试,启动Docker**

```bash
docker-compose up -d
```

---

## Task 6: 更新配置文件文档

**Files:**
- Modify: `configs/config.example.yaml`
- Modify: `CLAUDE.md`

**Step 1: 在 config.example.yaml 中添加注释**

在 `configs/config.example.yaml` 的 `ai.prompts` 部分添加:

```yaml
ai:
  enabled: false
  openai:
    # ... 其他配置 ...
  prompts:
    reminder_parse: |
      # 提醒解析prompt (已有)
    chat_response: |
      # 对话回复prompt (已有)
    activity_reply: |
      # 活动记录个性化回复prompt
      # 用于在用户记录活动后生成友好的、个性化的确认消息
      # 模板会自动填充: {{.UserMessage}}, {{.ActivityType}}, {{.Details}}
```

**Step 2: 更新 CLAUDE.md**

在 `CLAUDE.md` 中找到AI相关部分,添加功能说明:

```markdown
**AI Integration Architecture**
...
**Activity Reply Generation**: AI generates personalized responses when users record activities (reading, drinking water, exercise, etc.)

Features:
- Personalized activity confirmation messages
- Context-aware responses (book information, health tips, encouragement)
- Automatic fallback to simple confirmation when AI fails
```

**Step 3: 提交文档更新**

```bash
git add configs/config.example.yaml CLAUDE.md
git commit -m "docs: 更新配置文档,添加活动回复prompt说明"
```

---

## Task 7: 添加监控和指标

**Files:**
- Modify: `pkg/metrics/metrics.go`
- Modify: `internal/service/ai_parser.go`

**Step 1: 在 metrics.go 中添加指标**

在 `pkg/metrics/metrics.go` 中添加(约在文件末尾):

```go
var (
	// ... 现有指标 ...

	// ActivityReplyMetrics
	ActivityReplyGenerationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "activity_reply_generation_total",
			Help: "Total number of activity reply generation attempts",
		},
		[]string{"user_id", "activity_type", "status"}, // status: success/failure
	)

	ActivityReplyGenerationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "activity_reply_generation_duration_seconds",
			Help:    "Duration of activity reply generation in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"activity_type"},
	)
)

func init() {
	// ... 现有注册 ...

	prometheus.MustRegister(ActivityReplyGenerationTotal)
	prometheus.MustRegister(ActivityReplyGenerationDuration)
}
```

**Step 2: 在 ai_parser.go 中记录指标**

在 `GenerateActivityReply` 方法中添加指标记录:

```go
func (s *aiParserServiceImpl) GenerateActivityReply(ctx context.Context, userID string, userMessage string, activityType models.ActivityType, details map[string]interface{}) (string, error) {
	startTime := time.Now()
	userIDStr := fmt.Sprintf("%d", userID)

	// ... 现有逻辑 ...

	if err != nil {
		metrics.ActivityReplyGenerationTotal.WithLabelValues(userIDStr, string(activityType), "failure").Inc()
		return "", err
	}

	metrics.ActivityReplyGenerationTotal.WithLabelValues(userIDStr, string(activityType), "success").Inc()
	metrics.ActivityReplyGenerationDuration.WithLabelValues(string(activityType)).Observe(time.Since(startTime).Seconds())

	return reply, nil
}
```

**Step 3: 测试指标**

```bash
curl http://localhost:9090/metrics | grep activity_reply
```

预期输出:
```
activity_reply_generation_total{activity_type="read_book",status="success",user_id="123"} 1.0
activity_reply_generation_duration_seconds_bucket{activity_type="read_book",...} 0.523
```

**Step 4: 提交指标代码**

```bash
git add pkg/metrics/metrics.go internal/service/ai_parser.go
git commit -m "feat: 添加活动回复生成的Prometheus指标"
```

---

## 总结和后续优化

**当前实现完成了:**
1. ✅ AI prompt模板配置
2. ✅ AI服务层实现
3. ✅ 消息处理器集成
4. ✅ 单元测试
5. ✅ 降级处理
6. ✅ 监控指标

**可能的后续优化(YAGNI - 等有需求再实现):**
- 缓存常见活动的回复(减少AI调用)
- 根据用户历史行为调整回复风格
- 多语言支持(英文、其他语言)
- 回复质量评分和改进

**测试检查清单:**
- [ ] 看书活动回复包含书籍信息
- [ ] 喝水活动回复给予鼓励
- [ ] 运动活动回复肯定努力
- [ ] AI失败时降级为简单确认
- [ ] 日志正常记录
- [ ] Prometheus指标正常上报
- [ ] 无性能明显下降

**预计完成时间:** 每个任务 5-15 分钟,总计 1-2 小时
