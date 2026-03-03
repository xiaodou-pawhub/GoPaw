#!/bin/bash
# GoPaw 端到端测试脚本
# 用途：验证端到端对话和工具调用功能

set -e

echo "=========================================="
echo "  GoPaw 端到端测试脚本"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查点函数
check_point() {
    echo -e "${YELLOW}[检查]${NC} $1"
}

success_point() {
    echo -e "${GREEN}[成功]${NC} $1"
}

error_point() {
    echo -e "${RED}[错误]${NC} $1"
}

# 1. 环境检查
check_point "检查环境..."

if ! command -v go &> /dev/null; then
    error_point "Go 未安装"
    exit 1
fi
success_point "Go 版本：$(go version)"

if ! command -v curl &> /dev/null; then
    error_point "curl 未安装"
    exit 1
fi
success_point "curl 可用"

if ! command -v sqlite3 &> /dev/null; then
    error_point "sqlite3 未安装"
    exit 1
fi
success_point "sqlite3 可用"

echo ""

# 2. 编译检查
check_point "编译项目..."
if go build ./... > /dev/null 2>&1; then
    success_point "编译成功"
else
    error_point "编译失败"
    exit 1
fi

# 编译二进制
go build -o gopaw ./cmd/gopaw
success_point "生成二进制文件：gopaw"

echo ""

# 3. 清理旧数据
check_point "清理旧数据..."
rm -rf data/
rm -f config.yaml
success_point "清理完成"

# 4. 生成配置
check_point "生成默认配置..."
./gopaw init
if [ -f config.yaml ]; then
    success_point "配置文件已生成"
else
    error_point "配置文件生成失败"
    exit 1
fi

echo ""

# 5. 显示配置提示
echo "=========================================="
echo "  配置提示"
echo "=========================================="
echo ""
echo "请编辑 config.yaml 文件，设置 LLM API Key:"
echo ""
echo "  vim config.yaml"
echo ""
echo "或者设置环境变量:"
echo ""
echo "  export OPENAI_API_KEY=sk-your-key-here"
echo ""
echo "完成后按回车继续测试..."
read -p ""

echo ""

# 6. 启动服务
check_point "启动 GoPaw 服务..."
./gopaw start > /tmp/gopaw.log 2>&1 &
GOPAW_PID=$!
success_point "服务已启动 (PID: $GOPAW_PID)"

# 等待服务启动
echo "等待服务启动..."
sleep 3

# 检查服务是否正常运行
if ! kill -0 $GOPAW_PID 2>/dev/null; then
    error_point "服务启动失败，查看日志："
    cat /tmp/gopaw.log
    exit 1
fi

success_point "服务运行正常"

echo ""

# 7. 测试 API
echo "=========================================="
echo "  API 测试"
echo "=========================================="
echo ""

# 7.1 健康检查
check_point "测试健康检查..."
HEALTH=$(curl -s http://localhost:8088/health)
if echo "$HEALTH" | grep -q "ok"; then
    success_point "健康检查通过：$HEALTH"
else
    error_point "健康检查失败：$HEALTH"
fi

# 7.2 版本信息
check_point "测试版本信息..."
VERSION=$(curl -s http://localhost:8088/api/system/version)
if [ -n "$VERSION" ]; then
    success_point "版本信息：$VERSION"
else
    error_point "版本信息获取失败"
fi

# 7.3 Skills 列表
check_point "测试 Skills 列表..."
SKILLS=$(curl -s http://localhost:8088/api/skills)
if [ -n "$SKILLS" ]; then
    success_point "Skills 列表：$SKILLS"
else
    error_point "Skills 列表获取失败"
fi

echo ""

# 8. 对话测试
echo "=========================================="
echo "  对话测试（需要 API Key）"
echo "=========================================="
echo ""

check_point "测试简单对话..."
CHAT_RESP=$(curl -s -X POST http://localhost:8088/api/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id":"test-001","content":"你好，你是谁？"}')

if echo "$CHAT_RESP" | grep -q "content"; then
    success_point "对话成功：$CHAT_RESP"
else
    error_point "对话失败：$CHAT_RESP"
    echo "这可能是由于 API Key 未配置或 LLM 服务不可用"
fi

echo ""

# 9. 数据库验证
echo "=========================================="
echo "  数据库验证"
echo "=========================================="
echo ""

check_point "检查数据库文件..."
if [ -f data/gopaw.db ]; then
    success_point "数据库文件存在"
else
    error_point "数据库文件不存在"
fi

check_point "检查数据库表结构..."
TABLES=$(sqlite3 data/gopaw.db ".tables" 2>/dev/null)
echo "表结构：$TABLES"

if echo "$TABLES" | grep -q "sessions"; then
    success_point "sessions 表存在"
fi
if echo "$TABLES" | grep -q "messages"; then
    success_point "messages 表存在"
fi
if echo "$TABLES" | grep -q "messages_fts"; then
    success_point "messages_fts 表存在"
fi

check_point "检查 FTS5 触发器..."
TRIGGERS=$(sqlite3 data/gopaw.db "SELECT name FROM sqlite_master WHERE type='trigger';" 2>/dev/null)
echo "触发器：$TRIGGERS"

TRIGGER_COUNT=$(echo "$TRIGGERS" | wc -l | tr -d ' ')
if [ "$TRIGGER_COUNT" -ge 3 ]; then
    success_point "FTS5 触发器正常 ($TRIGGER_COUNT 个)"
else
    error_point "FTS5 触发器数量不足 (期望>=3, 实际:$TRIGGER_COUNT)"
fi

check_point "检查 WAL 模式..."
JOURNAL_MODE=$(sqlite3 data/gopaw.db "PRAGMA journal_mode;" 2>/dev/null)
if [ "$JOURNAL_MODE" = "wal" ]; then
    success_point "WAL 模式已启用"
else
    error_point "WAL 模式未启用 (当前：$JOURNAL_MODE)"
fi

check_point "查看对话记录..."
sqlite3 data/gopaw.db "SELECT id, role, substr(content, 1, 50) FROM messages LIMIT 5;" 2>/dev/null || echo "暂无对话记录"

echo ""

# 10. 工具测试（可选）
echo "=========================================="
echo "  工具测试（可选）"
echo "=========================================="
echo ""

# 创建测试文件
TEST_FILE="/tmp/gopaw_test_$(date +%s).txt"
echo "Hello GoPaw! This is a test file." > "$TEST_FILE"
success_point "创建测试文件：$TEST_FILE"

check_point "测试文件读取工具..."
TOOL_RESP=$(curl -s -X POST http://localhost:8088/api/agent/chat \
  -H "Content-Type: application/json" \
  -d "{\"session_id\":\"test-002\",\"content\":\"读取文件 $TEST_FILE 的内容\"}")

if echo "$TOOL_RESP" | grep -qi "Hello GoPaw\|file content\|读取"; then
    success_point "工具调用成功：$TOOL_RESP"
else
    error_point "工具调用失败或未完成：$TOOL_RESP"
    echo "这可能是因为 LLM 未正确调用工具，需要检查 Agent 逻辑"
fi

# 清理测试文件
rm -f "$TEST_FILE"

echo ""

# 11. 清理
echo "=========================================="
echo "  清理"
echo "=========================================="
echo ""

check_point "停止服务..."
kill $GOPAW_PID 2>/dev/null || true
success_point "服务已停止"

echo ""
echo "=========================================="
echo "  测试完成"
echo "=========================================="
echo ""
echo "测试日志保存在：/tmp/gopaw.log"
echo "数据库文件位于：data/gopaw.db"
echo ""
echo "请查看上面的测试结果，如有错误请反馈给开发团队。"
echo ""
