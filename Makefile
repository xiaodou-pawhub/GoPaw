BINARY   := gopaw
VERSION  := 0.1.0
LDFLAGS  := -ldflags "-X main.appVersion=$(VERSION)"
GO       := go
GOFLAGS  :=

.PHONY: build build-go dev run test clean lint web-install web-build web-dev docker-build docker-push help

## build: [生产] 构建前端（压缩）并嵌入 Go 二进制 → 单文件部署
build: web-build
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) ./cmd/gopaw

## build-go: [生产] 仅编译 Go 二进制（需 web/dist 已存在）
build-go:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) ./cmd/gopaw

## build-linux: [部署] 交叉编译 Linux amd64 二进制，用于 Docker 镜像构建（不需要源码进镜像）
build-linux: web-build
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) ./cmd/gopaw

## dev: [开发] 并行启动 Vite HMR dev server + Go 后端（不 embed 前端）
## 前端访问 http://localhost:5173（热更新），API 自动代理到 http://localhost:8088
dev:
	@echo "===================================================="
	@echo "  Dev mode"
	@echo "  Frontend (HMR) : http://localhost:5173"
	@echo "  Backend API    : http://localhost:8088"
	@echo "  Press Ctrl+C to stop both processes"
	@echo "===================================================="
	@trap 'kill 0' INT TERM EXIT; \
	 (cd web && pnpm dev) & \
	 $(GO) run -tags dev $(LDFLAGS) ./cmd/gopaw start; \
	 wait

## run: [生产] 构建并运行（requires config.yaml）
run: build
	./$(BINARY) start --config config.yaml

## web-install: 安装前端依赖
web-install:
	cd web && pnpm install

## web-dev: 单独启动 Vite dev server（需另起终端运行 Go 后端）
web-dev:
	cd web && pnpm run dev

## web-build: 构建前端（生产压缩）
web-build:
	cd web && pnpm run build

## test: run all tests with race detector
test:
	$(GO) test -race -cover ./...

## test-short: run only short tests
test-short:
	$(GO) test -short ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
	rm -f coverage.html
	rm -rf web/dist

## tidy: tidy go modules
tidy:
	$(GO) mod tidy

## vet: run go vet
vet:
	$(GO) vet ./...

## docker-build: 交叉编译 Linux amd64 二进制并构建 Docker 镜像（兼容 Apple Silicon 和 x86 开发机）
docker-build: build-linux
	docker build --platform linux/amd64 -t gopaw:$(VERSION) -t gopaw:latest -f docker/Dockerfile .

## docker-push: push the Docker image to a registry (set REGISTRY env var)
docker-push:
	docker tag gopaw:$(VERSION) $(REGISTRY)/gopaw:$(VERSION)
	docker push $(REGISTRY)/gopaw:$(VERSION)

## init-config: generate default config.yaml
init-config:
	./$(BINARY) init

help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
