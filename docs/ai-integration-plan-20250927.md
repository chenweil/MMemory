# MMemory AI集成计划 - 2025年9月27日

## 项目概述

MMemory当前使用基于正则表达式的自然语言解析器，存在识别能力有限的问题。本计划旨在集成AI能力，提升自然语言理解能力，为用户提供更智能的提醒设置体验。

## 当前状态分析

### 现有解析器能力
- ✅ 支持12种中文时间表达模式
- ✅ 基础的时间解析（每天、每周、一次性）
- ✅ 上午/下午/晚上时间段识别
- ✅ 工作日/周末智能识别
- ✅ 相对时间解析（明天、后天、下周）

### 局限性
- ❌ 复杂句式理解能力有限
- ❌ 多条件组合解析困难
- ❌ 语义消歧能力不足
- ❌ 容错性较差

## AI集成技术方案

### 技术架构设计

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Telegram      │    │   AI Parser     │    │   Fallback      │
│   Message       │───▶│   (AI Service)  │───▶│   Parser        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         │                        ▼                        ▼
         │            ┌─────────────────┐    ┌─────────────────┐
         └───────────▶│   Error         │    │   Traditional   │
                      │   Handler       │    │   Parser        │
                      └─────────────────┘    └─────────────────┘
```

### 核心组件设计

#### 1. AI解析器接口 (AIParserService)
```go
type AIParserService interface {
    ParseReminderRequest(text string) (*ReminderParseResult, error)
    ExtractTimeInfo(text string) (*TimeInfo, error)
    ExtractContent(text string) (*ContentInfo, error)
    IsRetryableError(error) bool
}
```

#### 2. 降级机制 (FallbackParser)
- AI解析失败时自动降级到传统正则解析
- 双保险机制确保服务可用性
- 错误监控和性能指标收集

#### 3. 配置管理 (AIConfig)
```yaml
ai:
  provider: "openai"  # openai, claude, deepseek
  api_key: "${AI_API_KEY}"
  model: "gpt-3.5-turbo"
  max_tokens: 1000
  timeout: 30s
  retry_count: 3
  fallback_enabled: true
```

## 实施步骤

### 📋 任务清单 (Todo)

#### 第一阶段：架构设计和接口定义 (预计1-2天)
- [ ] 设计AI解析器接口和数据结构
- [ ] 创建配置管理结构
- [ ] 设计错误处理和降级机制
- [ ] 制定单元测试策略

#### 第二阶段：AI服务集成 (预计2-3天)
- [ ] 实现OpenAI API集成
- [ ] 实现Claude API集成（备选）
- [ ] 实现DeepSeek API集成（备选）
- [ ] 设计统一的API调用封装
- [ ] 实现请求重试和限流机制

#### 第三阶段：业务逻辑实现 (预计1-2天)
- [ ] 实现AI解析服务层
- [ ] 集成到现有解析器架构
- [ ] 实现智能降级机制
- [ ] 添加性能监控和日志记录

#### 第四阶段：测试和优化 (预计1-2天)
- [ ] 单元测试覆盖
- [ ] 集成测试验证
- [ ] 性能基准测试
- [ ] 用户体验测试
- [ ] 错误处理验证

#### 第五阶段：部署和监控 (预计1天)
- [ ] 配置环境变量管理
- [ ] 部署到测试环境
- [ ] 生产环境部署
- [ ] 监控告警配置

## 技术选型对比

### AI模型选择

| 模型 | 优势 | 劣势 | 适用场景 |
|------|------|------|----------|
| OpenAI GPT-3.5 | API稳定，中文理解好 | 成本较高 | 生产环境首选 |
| Claude | 上下文理解强 | 中文支持略差 | 复杂场景备选 |
| DeepSeek | 成本低，中文优化 | 稳定性待验证 | 开发测试阶段 |

### 集成方案对比

| 方案 | 复杂度 | 维护成本 | 扩展性 |
|------|--------|----------|--------|
| 直接API调用 | 低 | 低 | 中 |
| 中间件封装 | 中 | 中 | 高 |
| 微服务架构 | 高 | 高 | 极高 |

**推荐方案**：直接API调用 + 统一封装层，平衡复杂度和扩展性。

## 配置设计方案

### 环境变量配置
```bash
# AI服务配置
AI_PROVIDER=openai
AI_API_KEY=your_api_key_here
AI_MODEL=gpt-3.5-turbo
AI_MAX_TOKENS=1000
AI_TIMEOUT=30
AI_RETRY_COUNT=3

# 降级配置
AI_FALLBACK_ENABLED=true
AI_FALLBACK_THRESHOLD=3
```

### YAML配置扩展
```yaml
ai:
  provider: "openai"
  api_key: "${AI_API_KEY}"
  model: "gpt-3.5-turbo"
  max_tokens: 1000
  timeout: 30s
  retry_count: 3
  fallback:
    enabled: true
    threshold: 3
    timeout_ms: 5000
```

## 预期效果

### 功能提升
- ✅ 复杂句式理解："每周一三五的晚上8点，如果没有会议就提醒我健身"
- ✅ 条件判断："如果明天不下雨，提醒我下午3点去跑步"
- ✅ 语义消歧："下周二的会议" vs "下周二提醒我开会"
- ✅ 容错增强："大概晚上8点左右提醒我"

### 用户体验
- ✅ 更自然的对话体验
- ✅ 更高的解析成功率
- ✅ 更好的错误提示
- ✅ 智能建议和确认

### 技术指标
- ⏱️ 响应时间：< 2秒
- 📊 解析成功率：> 95%
- 🔄 降级成功率：> 99.9%
- 📈 并发处理：支持100+用户同时使用

## 风险评估

### 技术风险
- **API稳定性**：AI服务API可能不稳定
- **成本控制**：API调用成本需要监控
- **响应延迟**：网络延迟影响用户体验

### 缓解措施
- 实现完善的降级机制
- 设置使用量监控和告警
- 优化请求缓存和批处理

### 业务风险
- **用户接受度**：AI解析可能不如预期
- **隐私顾虑**：用户数据安全

### 缓解措施
- 渐进式推出，收集反馈
- 明确数据使用政策，本地处理优先

## 下一步行动

### 立即行动
1. 完善技术方案细节
2. 评估API成本和使用限制
3. 设计具体的测试用例

### 短期计划（1周内）
1. 完成第一阶段架构设计
2. 开始AI服务集成开发
3. 建立测试环境

### 中长期规划（1月内）
1. 完成所有功能开发
2. 进行充分测试验证
3. 逐步上线和优化

## 附录

### 参考链接
- [OpenAI API文档](https://platform.openai.com/docs)
- [Claude API文档](https://docs.anthropic.com)
- [DeepSeek API文档](https://platform.deepseek.com)

### 相关文件
- [MMemory技术方案](MMemory-Specs-v0.0.1.md)
- [现有解析器实现](../internal/service/parser.go)

---

**文档版本**: v1.0  
**创建日期**: 2025年9月27日  
**最后更新**: 2025年9月27日  
**负责人**: 开发团队  
**状态**: 规划阶段