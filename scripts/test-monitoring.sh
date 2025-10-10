#!/bin/bash

# MMemory监控系统测试脚本

set -e

echo "🧪 开始测试MMemory监控系统..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_service() {
    local service=$1
    local url=$2
    local expected_status=$3
    
    echo -n "  测试 $service ... "
    
    if curl -s -o /dev/null -w "%{http_code}" "$url" | grep -q "$expected_status"; then
        echo -e "${GREEN}✅ 通过${NC}"
        return 0
    else
        echo -e "${RED}❌ 失败${NC}"
        return 1
    fi
}

test_metric() {
    local metric_name=$1
    local expected_pattern=$2
    
    echo -n "  测试指标 $metric_name ... "
    
    if curl -s http://localhost:9090/metrics | grep -q "$expected_pattern"; then
        echo -e "${GREEN}✅ 通过${NC}"
        return 0
    else
        echo -e "${RED}❌ 失败${NC}"
        return 1
    fi
}

# 1. 测试服务可用性
echo "🔍 1. 测试服务可用性"
test_service "Prometheus" "http://localhost:9091/-/healthy" "200"
test_service "Grafana" "http://localhost:3000/api/health" "200"
test_service "AlertManager" "http://localhost:9093/-/healthy" "200"
test_service "Node Exporter" "http://localhost:9100/metrics" "200"
test_service "Blackbox Exporter" "http://localhost:9115/metrics" "200"

# 2. 测试MMemory指标端点
echo ""
echo "📊 2. 测试MMemory指标端点"
test_service "MMemory Metrics" "http://localhost:9090/metrics" "200"

# 3. 测试关键指标
echo ""
echo "📈 3. 测试关键指标"
test_metric "系统运行时间" "mmemory_system_uptime_seconds"
test_metric "用户总数" "mmemory_bot_users_total"
test_metric "提醒总数" "mmemory_reminders_total"
test_metric "Bot消息数" "mmemory_bot_messages_total"
test_metric "数据库查询" "mmemory_database_queries_total"

# 4. 测试Prometheus目标
echo ""
echo "🎯 4. 测试Prometheus目标"
echo -n "  检查目标状态 ... "
if curl -s http://localhost:9091/api/v1/targets | jq -r '.data.activeTargets[] | select(.health != "up")' | wc -l | grep -q "^0$"; then
    echo -e "${GREEN}✅ 所有目标正常${NC}"
else
    echo -e "${RED}❌ 有目标异常${NC}"
    curl -s http://localhost:9091/api/v1/targets | jq -r '.data.activeTargets[] | select(.health != "up") | .labels.job'
fi

# 5. 测试告警规则
echo ""
echo "🚨 5. 测试告警规则"
echo -n "  检查告警规则 ... "
if curl -s http://localhost:9091/api/v1/rules | jq -r '.data.groups[].rules[] | select(.health != "ok")' | wc -l | grep -q "^0$"; then
    echo -e "${GREEN}✅ 所有规则正常${NC}"
else
    echo -e "${RED}❌ 有规则异常${NC}"
    curl -s http://localhost:9091/api/v1/rules | jq -r '.data.groups[].rules[] | select(.health != "ok") | .name'
fi

# 6. 生成测试负载
echo ""
echo "⚡ 6. 生成测试负载"
echo "  生成测试指标数据..."

# 发送一些测试请求到指标端点
for i in {1..10}; do
    curl -s http://localhost:9090/metrics > /dev/null
    sleep 0.1
done

echo -e "${GREEN}✅ 测试负载生成完成${NC}"

# 7. 验证数据收集
echo ""
echo "📋 7. 验证数据收集"
echo -n "  检查指标数据点 ... "
sleep 5  # 等待数据收集

# 检查最近5分钟是否有数据点
if curl -s "http://localhost:9091/api/v1/query?query=mmemory_system_uptime_seconds[5m]" | jq -r '.data.result[0].values' | wc -l | grep -q "^[2-9]\|[0-9]\{2,\}"; then
    echo -e "${GREEN}✅ 数据收集正常${NC}"
else
    echo -e "${YELLOW}⚠️  数据收集可能有问题${NC}"
fi

# 8. 测试外部监控
echo ""
echo "🌐 8. 测试外部服务监控"
echo -n "  Telegram API监控 ... "
if curl -s "http://localhost:9091/api/v1/query?query=probe_success{job=\"telegram_api\"}" | jq -r '.data.result[0].value[1]' | grep -q "1"; then
    echo -e "${GREEN}✅ Telegram API可达${NC}"
else
    echo -e "${YELLOW}⚠️  Telegram API可能不可达${NC}"
fi

# 9. 生成测试报告
echo ""
echo "📄 9. 生成测试报告"
cat > monitoring-test-report.txt << EOF
MMemory监控系统测试报告
========================
测试时间: $(date)
测试主机: $(hostname)

服务状态:
- Prometheus: http://localhost:9091
- Grafana: http://localhost:3000
- AlertManager: http://localhost:9093
- Node Exporter: http://localhost:9100
- Blackbox Exporter: http://localhost:9115
- MMemory Metrics: http://localhost:9090/metrics

关键指标检查:
- 系统运行时间: ✓
- 用户总数: ✓
- 提醒总数: ✓
- Bot消息处理: ✓
- 数据库查询: ✓

建议:
1. 定期检查Prometheus目标状态
2. 验证告警规则是否触发
3. 确保Grafana数据源配置正确
4. 配置告警通知渠道
5. 设置监控数据备份策略

EOF

echo "📄 测试报告已保存到 monitoring-test-report.txt"

echo ""
echo "🎉 监控系统测试完成！"
echo ""
echo "🔧 下一步建议:"
echo "  1. 配置Grafana数据源 (Prometheus: http://prometheus:9090)"
echo "  2. 导入MMemory监控面板"
echo "  3. 配置告警通知 (邮件/Slack)"
echo "  4. 设置定期测试计划"
echo "  5. 配置监控数据备份"
echo ""
echo "📚 相关链接:"
echo "  - Prometheus: http://localhost:9091"
echo "  - Grafana: http://localhost:3000 (admin/admin123)"
echo "  - AlertManager: http://localhost:9093"

# 显示实时指标样本
echo ""
echo "📊 当前关键指标样本:"
echo "  系统运行时间: $(curl -s http://localhost:9090/metrics | grep mmemory_system_uptime_seconds | awk '{print $2}') 秒"
echo "  用户总数: $(curl -s http://localhost:9090/metrics | grep mmemory_bot_users_total | awk '{print $2}')"
echo "  活跃提醒: $(curl -s http://localhost:9090/metrics | grep 'mmemory_reminders_total{status=\"active\"}' | awk '{print $2}')"
echo "  调度任务: $(curl -s http://localhost:9090/metrics | grep mmemory_scheduler_jobs_total | awk '{print $2}')"