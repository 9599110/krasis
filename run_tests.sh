#!/bin/bash
# 一键运行所有测试 (后端 + 前端)

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# 解析参数
TARGET="${1:-all}"

run_backend_tests() {
    echo -e "${BLUE}======================================${NC}"
    echo -e "${BLUE}  运行 krasis 后端所有测试用例${NC}"
    echo -e "${BLUE}======================================${NC}"

    # 检查 Go 是否安装
    if ! command -v go &> /dev/null; then
        echo -e "${RED}错误: Go 未安装或未配置 PATH${NC}"
        echo "请安装 Go: https://go.dev/dl/"
        return 1
    fi

    # 检查 Go 版本
    echo -e "${GREEN}Go 版本: $(go version)${NC}"
    echo ""

    cd "$SCRIPT_DIR/backend"

    # 运行所有测试
    echo -e "${YELLOW}开始运行后端测试...${NC}"
    echo ""

    # 运行单元测试
    echo -e "${GREEN}[1/3] 运行单元测试...${NC}"
    go test -v ./...

    # 运行集成测试
    echo ""
    echo -e "${GREEN}[2/3] 运行集成测试...${NC}"
    go test -v -tags=integration ./...

    # 运行测试覆盖率
    echo ""
    echo -e "${GREEN}[3/3] 生成测试覆盖率报告...${NC}"
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

    echo ""
    echo -e "覆盖率报告: ${YELLOW}backend/coverage.html${NC}"
}

run_frontend_tests() {
    echo -e "${BLUE}======================================${NC}"
    echo -e "${BLUE}  运行 krasis 前端所有测试用例${NC}"
    echo -e "${BLUE}======================================${NC}"

    # 检查 Flutter 是否安装
    if ! command -v flutter &> /dev/null; then
        echo -e "${RED}错误: Flutter 未安装或未配置 PATH${NC}"
        echo "请安装 Flutter: https://docs.flutter.dev/get-started/install"
        return 1
    fi

    # 检查 Flutter 版本
    echo -e "${GREEN}Flutter 版本: $(flutter --version | head -1)${NC}"
    echo ""

    cd "$SCRIPT_DIR/frontend"

    # 获取项目依赖
    echo -e "${YELLOW}获取依赖...${NC}"
    flutter pub get

    echo ""
    echo -e "${YELLOW}开始运行 Flutter 测试...${NC}"
    flutter test
}

# 执行测试
case "$TARGET" in
    backend)
        run_backend_tests
        ;;
    frontend)
        run_frontend_tests
        ;;
    all)
        run_backend_tests && echo "" && run_frontend_tests
        ;;
    help|--help|-h)
        echo "用法: $0 [backend|frontend|all]"
        echo ""
        echo "  backend  - 只运行后端测试 (Go)"
        echo "  frontend - 只运行前端测试 (Flutter)"
        echo "  all      - 运行所有测试 (默认)"
        echo "  help     - 显示帮助"
        exit 0
        ;;
    *)
        echo -e "${RED}未知目标: $TARGET${NC}"
        echo "使用 '$0 help' 查看帮助"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  测试完成!${NC}"
echo -e "${GREEN}======================================${NC}"
