#!/bin/bash

# GoPaw API 验证脚本
# 使用方法：./scripts/test-api.sh YOUR_ADMIN_TOKEN

set -e

TOKEN="${1:-}"
BASE_URL="http://localhost:8088"

if [ -z "$TOKEN" ]; then
    echo "❌ 请提供 Admin Token"
    echo "使用方法：./scripts/test-api.sh YOUR_TOKEN"
    echo ""
    echo "获取 Token 方法："
    echo "  docker logs gopaw | grep 'Admin token'"
    echo "  或查看启动日志中的 ⚡ Admin token"
    exit 1
fi

echo "🚀 GoPaw API 验证开始"
echo "Base URL: $BASE_URL"
echo "Token: ${TOKEN:0:8}..."
echo ""

# 设置认证头
AUTH_HEADER="Authorization: Bearer $TOKEN"

# 测试函数
test_api() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"
    
    echo -n "测试 $name ... "
    
    if [ -z "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "$AUTH_HEADER" \
            -H "Content-Type: application/json" \
            "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "$AUTH_HEADER" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$BASE_URL$endpoint")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        echo "✅ 通过 (HTTP $http_code)"
        return 0
    else
        echo "❌ 失败 (HTTP $http_code)"
        echo "响应：$body"
        return 1
    fi
}

# 计数器
passed=0
failed=0

# 1. 健康检查
echo "=== 健康检查 ==="
if test_api "健康检查" "GET" "/health"; then
    ((passed++))
else
    ((failed++))
fi
echo ""

# 2. 模型配置 API
echo "=== 模型配置 API ==="
if test_api "获取模型列表" "GET" "/api/settings/providers"; then
    ((passed++))
else
    ((failed++))
fi

if test_api "获取内置模型" "GET" "/api/settings/builtin-providers"; then
    ((passed++))
else
    ((failed++))
fi

if test_api "获取健康状态" "GET" "/api/settings/providers/health"; then
    ((passed++))
else
    ((failed++))
fi

if test_api "获取 Vision 模型" "GET" "/api/settings/providers/capable/vision"; then
    ((passed++))
else
    ((failed++))
fi
echo ""

# 3. 聊天 API
echo "=== 聊天 API ==="
if test_api "获取会话列表" "GET" "/api/agent/sessions"; then
    ((passed++))
else
    ((failed++))
fi
echo ""

# 4. 频道 API
echo "=== 频道 API ==="
if test_api "频道健康状态" "GET" "/api/channels/health"; then
    ((passed++))
else
    ((failed++))
fi
echo ""

# 5. 技能 API
echo "=== 技能 API ==="
if test_api "获取技能列表" "GET" "/api/skills"; then
    ((passed++))
else
    ((failed++))
fi
echo ""

# 总结
echo "================================"
echo "✅ 通过：$passed"
echo "❌ 失败：$failed"
echo "================================"

if [ $failed -eq 0 ]; then
    echo "🎉 所有 API 验证通过！"
    exit 0
else
    echo "⚠️  有 $failed 个 API 验证失败，请检查日志"
    exit 1
fi
