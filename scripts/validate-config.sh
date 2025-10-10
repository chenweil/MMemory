#!/bin/bash

# MMemory 配置验证脚本
# 用途：检查配置文件中是否有硬编码的敏感信息

set -e

echo "🔍 MMemory 配置安全检查..."
echo ""

ERRORS=0

# 检查1: configs/config.yaml 中不应有硬编码的 token
echo "检查 configs/config.yaml..."
if grep -q 'token: "[^"]' configs/config.yaml 2>/dev/null; then
    echo "❌ 错误: configs/config.yaml 中包含硬编码的 token"
    echo "   请使用环境变量 MMEMORY_BOT_TOKEN 代替"
    ERRORS=$((ERRORS + 1))
else
    echo "✅ configs/config.yaml: token 配置正确"
fi

# 检查2: .env 文件是否在 .gitignore 中
echo "检查 .gitignore..."
if grep -q "^\.env$" .gitignore; then
    echo "✅ .gitignore: .env 文件已忽略"
else
    echo "❌ 警告: .gitignore 中缺少 .env"
    ERRORS=$((ERRORS + 1))
fi

# 检查3: .env 文件中是否使用了正确的变量名
if [ -f .env ]; then
    echo "检查 .env 文件..."

    if grep -q "^MMEMORY_BOT_TOKEN=" .env; then
        echo "✅ .env: 使用正确的变量名 MMEMORY_BOT_TOKEN"
    else
        echo "❌ 错误: .env 中缺少 MMEMORY_BOT_TOKEN"
        if grep -q "^TELEGRAM_BOT_TOKEN=" .env; then
            echo "   发现 TELEGRAM_BOT_TOKEN，请改为 MMEMORY_BOT_TOKEN"
        fi
        ERRORS=$((ERRORS + 1))
    fi

    # 检查是否有实际的 token 值
    if grep -q "^MMEMORY_BOT_TOKEN=.*:.*" .env; then
        echo "✅ .env: Bot token 已配置"
    else
        echo "⚠️  警告: MMEMORY_BOT_TOKEN 似乎未配置有效值"
    fi
fi

# 检查4: docker-compose.yml 中使用正确的变量名
echo "检查 docker-compose.yml..."
if grep -q "MMEMORY_BOT_TOKEN" docker-compose.yml; then
    echo "✅ docker-compose.yml: 使用正确的变量名"
else
    if grep -q "TELEGRAM_BOT_TOKEN" docker-compose.yml; then
        echo "❌ 错误: docker-compose.yml 使用了错误的变量名 TELEGRAM_BOT_TOKEN"
        echo "   请改为 MMEMORY_BOT_TOKEN"
        ERRORS=$((ERRORS + 1))
    fi
fi

# 检查5: 搜索可能泄露的敏感信息
echo "搜索可能的敏感信息泄露..."
LEAKED=$(git grep -n ":[A-Z0-9]\{35,\}" -- configs/*.yaml configs/*.yml 2>/dev/null || true)
if [ -n "$LEAKED" ]; then
    echo "❌ 警告: 发现可能的 API Token/Key 泄露:"
    echo "$LEAKED"
    ERRORS=$((ERRORS + 1))
else
    echo "✅ 未发现明显的密钥泄露"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ $ERRORS -eq 0 ]; then
    echo "✅ 配置检查通过！"
    exit 0
else
    echo "❌ 发现 $ERRORS 个问题，请修复后再运行"
    exit 1
fi
