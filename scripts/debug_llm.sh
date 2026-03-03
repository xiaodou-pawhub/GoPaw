#!/bin/bash
# GoPaw 调试模式测试脚本
# 用途：查看完整的 LLM 请求/响应详情

set -e

echo "=========================================="
echo "  GoPaw 调试模式 - 查看 LLM 请求详情"
echo "=========================================="
echo ""

# 1. 检查配置
echo "检查配置文件..."
if [ ! -f config.yaml ]; then
    echo "❌ config.yaml 不存在，正在生成..."
    ./gopaw init
fi

# 2. 提示用户设置 debug 模式
echo ""
echo "请确保 config.yaml 中启用了 debug 模式："
echo ""
echo "  app:"
echo "    debug: true"
echo ""
echo "正在自动设置..."
# 使用 sed 修改 debug 为 true（如果存在）
if grep -q "debug:" config.yaml; then
    sed -i.bak 's/debug: false/debug: true/' config.yaml
    sed -i.bak 's/debug: true/debug: true/' config.yaml
    rm -f config.yaml.bak
    echo "✅ 已设置 debug: true"
else
    # 在 app 部分添加 debug: true
    sed -i.bak '/^app:/a\  debug: true' config.yaml
    rm -f config.yaml.bak
    echo "✅ 已添加 debug: true"
fi

echo ""
echo "=========================================="
echo "  启动服务（调试模式）"
echo "=========================================="
echo ""
echo "服务启动后，日志会显示："
echo "  - LLM 请求的完整 URL"
echo "  - 请求头（包括 API Key 前缀）"
echo "  - 请求体（完整的 JSON payload）"
echo "  - HTTP 响应状态码"
echo "  - 原始响应内容"
echo ""
echo "按 Ctrl+C 停止服务"
echo ""
echo "准备启动..."
sleep 2

# 3. 启动服务（前台运行，显示日志）
./gopaw start
