#!/bin/bash
# 一键运行所有 Flutter 测试

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}======================================${NC}"
echo -e "${YELLOW}  运行 krasis 前端所有测试用例${NC}"
echo -e "${YELLOW}======================================${NC}"

# 检查 Flutter 是否安装
if ! command -v flutter &> /dev/null; then
    echo -e "${RED}错误: Flutter 未安装或未配置 PATH${NC}"
    echo "请安装 Flutter: https://docs.flutter.dev/get-started/install"
    exit 1
fi

# 检查 Flutter 版本
echo -e "${GREEN}Flutter 版本: $(flutter --version | head -1)${NC}"
echo ""

# 进入前端目录
cd "$(dirname "$0")/frontend"

# 获取项目依赖
echo -e "${YELLOW}获取依赖...${NC}"
flutter pub get

echo ""
echo -e "${YELLOW}开始运行测试...${NC}"
echo ""

# 运行所有测试
flutter test

echo ""
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  Flutter 测试完成!${NC}"
echo -e "${GREEN}======================================${NC}"
