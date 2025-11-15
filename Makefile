.PHONY: build test clean install run fmt vet lint build-all cross-compile

# 变量
BINARY_NAME=jsfindcrack
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# 默认目标
all: fmt vet test build

# 构建当前平台
build:
	@echo "构建 ${BINARY_NAME}..."
	go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd/jsfindcrack

# 交叉编译所有平台
build-all: cross-compile

cross-compile:
	@echo "交叉编译所有平台..."
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64 ./cmd/jsfindcrack
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-amd64 ./cmd/jsfindcrack
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-arm64 ./cmd/jsfindcrack
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-amd64.exe ./cmd/jsfindcrack
	@echo "交叉编译完成!"

# 运行测试
test:
	@echo "运行测试..."
	go test -v -cover ./...

# 运行测试并生成覆盖率报告
test-coverage:
	@echo "生成覆盖率报告..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 运行go vet
vet:
	@echo "运行 go vet..."
	go vet ./...

# 运行linter
lint:
	@echo "运行 golangci-lint..."
	golangci-lint run || echo "golangci-lint未安装,跳过lint检查"

# 安装依赖
install:
	@echo "安装依赖..."
	go mod download
	go mod tidy

# 运行程序
run:
	go run cmd/jsfindcrack/main.go

# 清理
clean:
	@echo "清理..."
	rm -f ${BINARY_NAME} ${BINARY_NAME}-*
	rm -f coverage.out coverage.html
	rm -rf output/ logs/

# 显示帮助
help:
	@echo "可用命令:"
	@echo "  make build         - 构建当前平台的二进制文件"
	@echo "  make build-all     - 交叉编译所有平台"
	@echo "  make test          - 运行测试"
	@echo "  make test-coverage - 生成覆盖率报告"
	@echo "  make fmt           - 格式化代码"
	@echo "  make vet           - 运行 go vet"
	@echo "  make lint          - 运行 golangci-lint"
	@echo "  make install       - 安装依赖"
	@echo "  make run           - 运行程序"
	@echo "  make clean         - 清理构建产物"
