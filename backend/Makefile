.PHONY: test test-unit test-integration test-coverage test-verbose clean

# 默认目标：运行所有测试
test: test-unit test-integration

# 单元测试
test-unit:
	@echo "Running unit tests..."
	go test -v ./...

# 集成测试
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

# 带覆盖率报告
test-coverage:
	@echo "Running tests with coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 运行单个模块测试
test-note:
	go test -v ./internal/note/...

test-share:
	go test -v ./internal/share/...

test-collab:
	go test -v ./internal/collab/...

test-ai:
	go test -v ./internal/ai/...

test-file:
	go test -v ./internal/file/...

test-auth:
	go test -v ./internal/auth/...

# 运行单个测试函数 (用法: make test-run TEST=TestNoteCreate)
test-run:
	go test -v -run "^Test$$" ./...

# 详细输出 + 停止于第一个失败
test-verbose:
	go test -v -failfast ./...

# 清理测试生成的文件
clean:
	rm -f coverage.out coverage.html
	find . -name "*.test" -delete

# 显示测试帮助
help:
	@echo "Krasis Backend Test Commands"
	@echo ""
	@echo "  make test           - Run all tests (unit + integration)"
	@echo "  make test-unit      - Run unit tests only"
	@echo "  make test-integration - Run integration tests only"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make test-note      - Run note module tests"
	@echo "  make test-share     - Run share module tests"
	@echo "  make test-collab    - Run collab module tests"
	@echo "  make test-ai        - Run AI module tests"
	@echo "  make test-file      - Run file module tests"
	@echo "  make test-auth      - Run auth module tests"
	@echo "  make test-verbose   - Run tests with verbose output"
	@echo "  make clean          - Clean test artifacts"
