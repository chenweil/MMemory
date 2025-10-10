#!/bin/bash

# MMemory监控系统部署脚本

set -e

echo "🚀 开始部署MMemory监控系统..."

# 检查Docker和Docker Compose
if ! command -v docker &> /dev/null; then
    echo "❌ Docker未安装，请先安装Docker"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose未安装，请先安装Docker Compose"
    exit 1
fi

# 创建必要的目录
echo "📁 创建数据目录..."
mkdir -p data
mkdir -p configs/alerts
mkdir -p configs/grafana

# 设置文件权限
echo "🔒 设置文件权限..."
chmod 644 configs/prometheus.yml
chmod 644 configs/alertmanager.yml
chmod 644 configs/blackbox.yml
chmod 644 configs/alerts/*.yml

# 检查环境变量
if [ -z "$BOT_TOKEN" ]; then
    echo "⚠️ 警告: BOT_TOKEN环境变量未设置，请在.env文件中设置"
fi

# 启动监控系统
echo "🐳 启动监控服务..."
docker-compose -f docker-compose.monitoring.yml up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 30

# 检查服务状态
echo "🔍 检查服务状态..."
services=("prometheus" "grafana" "alertmanager" "node-exporter" "blackbox-exporter")

for service in "${services[@]}"; do
    if docker-compose -f docker-compose.monitoring.yml ps | grep -q "$service.*Up"; then
        echo "✅ $service 运行正常"
    else
        echo "❌ $service 未正常运行"
    fi
done

# 显示访问信息
echo ""
echo "🎉 监控系统部署完成！"
echo ""
echo "📊 访问地址:"
echo "  - Prometheus: http://localhost:9091"
echo "  - Grafana: http://localhost:3000 (admin/admin123)"
echo "  - AlertManager: http://localhost:9093"
echo ""
echo "📋 默认凭据:"
echo "  - Grafana管理员: admin/admin123"
echo ""
echo "🔧 常用命令:"
echo "  - 查看日志: docker-compose -f docker-compose.monitoring.yml logs -f"
echo "  - 停止服务: docker-compose -f docker-compose.monitoring.yml down"
echo "  - 重启服务: docker-compose -f docker-compose.monitoring.yml restart"
echo ""
echo "📖 文档链接:"
echo "  - Prometheus: https://prometheus.io/docs/"
echo "  - Grafana: https://grafana.com/docs/"
echo "  - AlertManager: https://prometheus.io/docs/alerting/latest/alertmanager/"
echo ""
echo "⚠️  注意事项:"
echo "  - 请修改AlertManager配置文件中的邮件和Slack设置"
echo "  - 建议设置BOT_TOKEN环境变量"
echo "  - 定期检查磁盘空间使用情况"
echo ""
echo "🎯 下一步:"
echo "  1. 配置Grafana数据源 (Prometheus)"
echo "  2. 导入MMemory监控面板"
echo "  3. 配置告警通知渠道"
echo "  4. 验证告警规则是否生效"

# 保存部署信息
cat > monitoring-info.txt << EOF
MMemory监控系统部署信息
========================
部署时间: $(date)
Prometheus地址: http://localhost:9091
Grafana地址: http://localhost:3000
AlertManager地址: http://localhost:9093

默认凭据:
- Grafana: admin/admin123

Docker Compose文件: docker-compose.monitoring.yml
配置文件目录: configs/

查看状态: docker-compose -f docker-compose.monitoring.yml ps
查看日志: docker-compose -f docker-compose.monitoring.yml logs -f
停止服务: docker-compose -f docker-compose.monitoring.yml down
EOF

echo "📄 部署信息已保存到 monitoring-info.txt"