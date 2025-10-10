# MMemory - 代码风格指南

## 开发约定

### 代码风格
- 使用 Go 标准格式化 (`gofmt`)
- 错误处理遵循 Go 惯用方式
- 接口定义优先设计

### 测试规范
- 单元测试覆盖核心业务逻辑
- 集成测试验证端到端流程
- 测试文件与源码文件对应

### 日志规范
- 结构化日志 (JSON 格式)
- 分级日志输出 (debug/info/warn/error)
- 上下文信息丰富

## 故障排除

### 常见问题
1. **Bot 无响应**: 检查 Token 配置和网络连接
2. **数据库错误**: 验证文件权限和路径
3. **调度器异常**: 检查时区设置和 cron 表达式

### 调试技巧
```bash
# 启用调试模式
将 config.yaml 中的 bot.debug 设为 true

# 查看详细日志
tail -f data/mmemory.log

# 检查指标
curl http://localhost:9090/metrics
```