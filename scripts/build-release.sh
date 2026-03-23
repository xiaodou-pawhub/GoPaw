#!/bin/bash
# GoPaw 跨平台打包脚本
# 用法: ./scripts/build-release.sh [版本号]
# 示例: ./scripts/build-release.sh 0.2.0

set -e

# 配置
VERSION=${1:-"0.2.0"}
BINARY="gopaw"
DIST_DIR="dist"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  GoPaw Release Builder${NC}"
echo -e "${CYAN}  Version: ${VERSION}${NC}"
echo -e "${CYAN}========================================${NC}"

# 清理旧的构建产物
echo -e "${YELLOW}[1/5] 清理旧构建产物...${NC}"
rm -rf ${DIST_DIR}
rm -f ${BINARY} ${BINARY}-linux ${BINARY}.exe
rm -rf web/dist

# 构建前端
echo -e "${YELLOW}[2/5] 构建前端...${NC}"
cd web
bun install
bun run build
cd ..
echo -e "${GREEN}✓ 前端构建完成${NC}"

# 构建参数
LDFLAGS="-X main.appVersion=${VERSION} -s -w"

# 构建 Linux amd64
echo -e "${YELLOW}[3/5] 构建 Linux amd64...${NC}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o ${BINARY}-linux ./cmd/gopaw
echo -e "${GREEN}✓ Linux amd64 构建完成${NC}"

# 构建 Windows amd64
echo -e "${YELLOW}[4/5] 构建 Windows amd64...${NC}"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o ${BINARY}.exe ./cmd/gopaw
echo -e "${GREEN}✓ Windows amd64 构建完成${NC}"

# 打包发布
echo -e "${YELLOW}[5/5] 打包发布...${NC}"
mkdir -p ${DIST_DIR}

# Linux 包
LINUX_DIR="${DIST_DIR}/${BINARY}-${VERSION}-linux-amd64"
mkdir -p ${LINUX_DIR}
cp ${BINARY}-linux ${LINUX_DIR}/${BINARY}
cp config.yaml.example ${LINUX_DIR}/config.yaml.example
cp README.md ${LINUX_DIR}/README.md
cp LICENSE ${LINUX_DIR}/LICENSE
cp scripts/start-linux.sh ${LINUX_DIR}/start.sh
chmod +x ${LINUX_DIR}/${BINARY}
chmod +x ${LINUX_DIR}/start.sh
cd ${DIST_DIR}
tar -czvf ${BINARY}-${VERSION}-linux-amd64.tar.gz ${BINARY}-${VERSION}-linux-amd64
cd ..
echo -e "${GREEN}✓ Linux 发布包: ${DIST_DIR}/${BINARY}-${VERSION}-linux-amd64.tar.gz${NC}"

# Windows 包
WINDOWS_DIR="${DIST_DIR}/${BINARY}-${VERSION}-windows-amd64"
mkdir -p ${WINDOWS_DIR}
cp ${BINARY}.exe ${WINDOWS_DIR}/${BINARY}.exe
cp config.yaml.example ${WINDOWS_DIR}/config.yaml.example
cp README.md ${WINDOWS_DIR}/README.md
cp LICENSE ${WINDOWS_DIR}/LICENSE
cp scripts/start-windows.bat ${WINDOWS_DIR}/start.bat
cd ${DIST_DIR}
zip -r ${BINARY}-${VERSION}-windows-amd64.zip ${BINARY}-${VERSION}-windows-amd64
cd ..
echo -e "${GREEN}✓ Windows 发布包: ${DIST_DIR}/${BINARY}-${VERSION}-windows-amd64.zip${NC}"

# 输出文件大小
echo ""
echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  构建完成${NC}"
echo -e "${CYAN}========================================${NC}"
echo ""
echo -e "发布包:"
ls -lh ${DIST_DIR}/*.tar.gz ${DIST_DIR}/*.zip 2>/dev/null || true
echo ""
echo -e "${YELLOW}使用说明:${NC}"
echo -e "  Linux:   解压后运行 ./gopaw start"
echo -e "  Windows: 解压后运行 gopaw.exe start"
echo ""