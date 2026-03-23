#!/bin/bash
# GoPaw Linux 启动脚本

echo "========================================"
echo "  GoPaw AI Assistant"
echo "========================================"
echo

# 检查配置文件
if [ ! -f "config.yaml" ]; then
    echo "[!] 未找到 config.yaml，正在从示例创建..."
    cp config.yaml.example config.yaml
    echo "[√] 已创建 config.yaml，请根据需要修改配置"
    echo
fi

# 启动服务
echo "[*] 启动 GoPaw..."
echo "[*] 访问地址: http://localhost:16688"
echo

./gopaw start --mode solo