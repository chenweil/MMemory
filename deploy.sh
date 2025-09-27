#!/bin/bash

# MMemory Docker 部署脚本
# 使用方法: ./deploy.sh [start|stop|restart|logs|build]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
}

# 检查环境变量
check_env() {
    if [ ! -f .env ]; then
        if [ -f .env.example ]; then
            log_warn ".env 文件不存在，从 .env.example 复制"
            cp .env.example .env
            log_warn "请编辑 .env 文件设置你的 TELEGRAM_BOT_TOKEN"
            exit 1
        else
            log_error ".env.example 文件不存在"
            exit 1
        fi
    fi
    
    # 检查是否设置了必要的环境变量
    source .env
    if [ -z "$TELEGRAM_BOT_TOKEN" ] || [ "$TELEGRAM_BOT_TOKEN" = "your_telegram_bot_token_here" ]; then
        log_error "请在 .env 文件中设置有效的 TELEGRAM_BOT_TOKEN"
        exit 1
    fi
}

# 创建数据目录
create_data_dir() {
    if [ ! -d "./data" ]; then
        log_info "创建数据目录..."
        mkdir -p ./data
        chmod 755 ./data
    fi
}

# 构建镜像
build() {
    log_info "构建 MMemory Docker 镜像..."
    docker-compose build --no-cache
    log_info "镜像构建完成"
}

# 启动服务
start() {
    log_info "启动 MMemory 服务..."
    create_data_dir
    docker-compose up -d
    log_info "服务启动完成"
    
    # 等待服务启动
    sleep 5
    status
}

# 停止服务
stop() {
    log_info "停止 MMemory 服务..."
    docker-compose down
    log_info "服务已停止"
}

# 重启服务
restart() {
    log_info "重启 MMemory 服务..."
    stop
    start
}

# 查看日志
logs() {
    log_info "查看 MMemory 服务日志..."
    docker-compose logs -f --tail=100
}

# 查看状态
status() {
    log_info "MMemory 服务状态:"
    docker-compose ps
    
    # 检查健康状态
    container_id=$(docker-compose ps -q mmemory)
    if [ ! -z "$container_id" ]; then
        health_status=$(docker inspect --format='{{.State.Health.Status}}' $container_id 2>/dev/null || echo "unknown")
        log_info "健康状态: $health_status"
    fi
}

# 清理数据
clean() {
    read -p "确定要删除所有数据吗？这将删除数据库文件！(y/N): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_warn "删除数据目录..."
        stop
        rm -rf ./data
        log_warn "数据已清理"
    else
        log_info "取消清理操作"
    fi
}

# 备份数据
backup() {
    if [ ! -d "./data" ]; then
        log_error "数据目录不存在，无法备份"
        exit 1
    fi
    
    backup_file="mmemory_backup_$(date +%Y%m%d_%H%M%S).tar.gz"
    log_info "备份数据到 $backup_file..."
    tar -czf $backup_file ./data
    log_info "备份完成: $backup_file"
}

# 恢复数据
restore() {
    if [ -z "$1" ]; then
        log_error "请指定备份文件: ./deploy.sh restore backup_file.tar.gz"
        exit 1
    fi
    
    if [ ! -f "$1" ]; then
        log_error "备份文件不存在: $1"
        exit 1
    fi
    
    read -p "确定要恢复备份吗？这将覆盖现有数据！(y/N): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "停止服务..."
        stop
        
        log_info "恢复备份 $1..."
        rm -rf ./data
        tar -xzf $1
        
        log_info "重启服务..."
        start
        log_info "恢复完成"
    else
        log_info "取消恢复操作"
    fi
}

# 显示帮助
show_help() {
    echo "MMemory Docker 部署工具"
    echo ""
    echo "使用方法:"
    echo "  ./deploy.sh [命令]"
    echo ""
    echo "可用命令:"
    echo "  start     - 启动服务"
    echo "  stop      - 停止服务"
    echo "  restart   - 重启服务"
    echo "  logs      - 查看日志"
    echo "  status    - 查看状态"
    echo "  build     - 构建镜像"
    echo "  clean     - 清理数据"
    echo "  backup    - 备份数据"
    echo "  restore   - 恢复备份"
    echo "  help      - 显示帮助"
    echo ""
    echo "首次部署:"
    echo "  1. 复制 .env.example 为 .env"
    echo "  2. 编辑 .env 设置 TELEGRAM_BOT_TOKEN"
    echo "  3. 运行 ./deploy.sh start"
}

# 主函数
main() {
    check_dependencies
    
    case "${1:-help}" in
        start)
            check_env
            start
            ;;
        stop)
            stop
            ;;
        restart)
            check_env
            restart
            ;;
        logs)
            logs
            ;;
        status)
            status
            ;;
        build)
            build
            ;;
        clean)
            clean
            ;;
        backup)
            backup
            ;;
        restore)
            restore "$2"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"