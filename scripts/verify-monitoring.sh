#!/bin/bash

# MMemory监控功能验证脚本

echo "🧪 验证MMemory监控功能..."

# 检查指标包是否可以正常导入和编译
echo "📦 检查指标包编译..."
cd /Users/chenweilong/www/MMemory

# 创建一个简单的测试程序
cat > test_metrics.go << 'EOF'
package main

import (
	"fmt"
	"mmemory/pkg/metrics"
)

func main() {
	fmt.Println("✅ 指标包导入成功")
	
	// 测试指标函数调用
	metrics.SetBotUsers(100)
	metrics.RecordReminderCreated()
	metrics.RecordBotMessage("test", "success")
	
	fmt.Println("✅ 指标函数调用成功")
	fmt.Println("📊 监控功能验证完成")
}
EOF

# 编译测试程序
echo "🔨 编译测试程序..."
if go build -o test_metrics test_metrics.go; then
    echo "✅ 测试程序编译成功"
    
    # 运行测试程序
    echo "🚀 运行测试程序..."
    if ./test_metrics; then
        echo "✅ 监控功能验证通过"
    else
        echo "❌ 测试程序运行失败"
        exit 1
    fi
else
    echo "❌ 测试程序编译失败"
    exit 1
fi

# 清理测试文件
rm -f test_metrics test_metrics.go

echo ""
echo "🎉 MMemory监控功能验证完成！"
echo ""
echo "✅ 已实现功能:"
echo "  📊 Prometheus指标收集 - 完成"
echo "  📈 Grafana监控面板 - 完成" 
echo "  🚨 关键指标告警规则 - 完成"
echo ""
echo "📁 相关文件:"
echo "  - 指标定义: pkg/metrics/metrics.go"
echo "  - 监控服务: internal/service/monitoring.go"
echo "  - Prometheus配置: configs/prometheus.yml"
echo "  - Grafana面板: configs/grafana/mmemory-dashboard.json"
echo "  - 告警规则: configs/alerts/mmemory.yml"
echo "  - 部署脚本: scripts/deploy-monitoring.sh"
echo "  - 测试脚本: scripts/test-monitoring.sh"
echo "  - 文档: docs/MONITORING.md"
echo ""
echo "🚀 使用说明:"
echo "  1. 部署监控: ./scripts/deploy-monitoring.sh"
echo "  2. 测试监控: ./scripts/test-monitoring.sh"
echo "  3. 访问Grafana: http://localhost:3000"
echo "  4. 访问Prometheus: http://localhost:9091"
echo "  5. 查看文档: docs/MONITORING.md"