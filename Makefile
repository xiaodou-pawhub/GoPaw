BINARY  := gopaw
VERSION := 0.2.3
LDFLAGS := -ldflags "-X main.appVersion=$(VERSION) -s -w"
GO      := go

# ─── Colors ─────────────────────────────────────────────────────────────────
CYAN  := \033[36m
GREEN := \033[32m
YELLOW:= \033[33m
RESET := \033[0m

.PHONY: build build-desktop build-linux build-linux-arm64 build-windows build-darwin build-darwin-arm64 build-all \
        package-linux package-linux-arm64 package-windows package-darwin package-darwin-arm64 package-all \
        dev dev-go dev-embedded \
        run run-solo run-team run-tray \
        web-install web-build web-dev \
        test test-short lint vet tidy clean \
        docker-build docker-push release help

# ─── Production builds ───────────────────────────────────────────────────────

## build: [服务器] 构建前端 + 嵌入 Go → 单文件，无 CGo，适合 Linux/Docker 部署
build: web-build
	$(GO) build $(LDFLAGS) -o $(BINARY) ./cmd/gopaw
	@printf "$(GREEN)✓ Built ./$(BINARY) (server, no tray)$(RESET)\n"

## build-desktop: [桌面] 带系统托盘的桌面版（含 CGo，macOS/Windows）
build-desktop: web-build
	$(GO) build -tags tray $(LDFLAGS) -o $(BINARY) ./cmd/gopaw
	@printf "$(GREEN)✓ Built ./$(BINARY) (desktop, with tray)$(RESET)\n"

## build-go: [快速] 仅编译 Go 二进制（需 web/dist 已存在）
build-go:
	$(GO) build $(LDFLAGS) -o $(BINARY) ./cmd/gopaw

## build-linux: 交叉编译 Linux amd64 服务器二进制（纯 Go，无 CGo）
build-linux: web-build
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BINARY)-linux-amd64 ./cmd/gopaw
	@printf "$(GREEN)✓ Built ./$(BINARY)-linux-amd64 (linux/amd64)$(RESET)\n"

## build-linux-arm64: 交叉编译 Linux arm64 服务器二进制（纯 Go，无 CGo）
build-linux-arm64: web-build
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BINARY)-linux-arm64 ./cmd/gopaw
	@printf "$(GREEN)✓ Built ./$(BINARY)-linux-arm64 (linux/arm64)$(RESET)\n"

## build-windows: 交叉编译 Windows amd64 服务器二进制（纯 Go，无 CGo）
build-windows: web-build
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BINARY)-windows-amd64.exe ./cmd/gopaw
	@printf "$(GREEN)✓ Built ./$(BINARY)-windows-amd64.exe (windows/amd64)$(RESET)\n"

## build-darwin: 交叉编译 macOS amd64 服务器二进制（纯 Go，无 CGo）
build-darwin: web-build
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BINARY)-darwin-amd64 ./cmd/gopaw
	@printf "$(GREEN)✓ Built ./$(BINARY)-darwin-amd64 (darwin/amd64)$(RESET)\n"

## build-darwin-arm64: 交叉编译 macOS arm64 (M1/M2) 服务器二进制（纯 Go，无 CGo）
build-darwin-arm64: web-build
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BINARY)-darwin-arm64 ./cmd/gopaw
	@printf "$(GREEN)✓ Built ./$(BINARY)-darwin-arm64 (darwin/arm64)$(RESET)\n"

## build-all: 交叉编译所有平台
build-all: build-linux build-linux-arm64 build-windows build-darwin build-darwin-arm64
	@printf "$(GREEN)✓ Built all platforms$(RESET)\n"

# ─── Package releases ───────────────────────────────────────────────────────

## package-linux: 打包 Linux amd64 发布包
package-linux: build-linux
	@rm -rf dist/gopaw-$(VERSION)-linux-amd64
	@mkdir -p dist/gopaw-$(VERSION)-linux-amd64
	@cp $(BINARY)-linux-amd64 dist/gopaw-$(VERSION)-linux-amd64/gopaw
	@cp config.yaml.example dist/gopaw-$(VERSION)-linux-amd64/config.yaml.example
	@cp README.md dist/gopaw-$(VERSION)-linux-amd64/README.md
	@cp LICENSE dist/gopaw-$(VERSION)-linux-amd64/LICENSE
	@chmod +x dist/gopaw-$(VERSION)-linux-amd64/gopaw
	@cd dist && tar -czvf gopaw-$(VERSION)-linux-amd64.tar.gz gopaw-$(VERSION)-linux-amd64
	@printf "$(GREEN)✓ Packaged dist/gopaw-$(VERSION)-linux-amd64.tar.gz$(RESET)\n"

## package-linux-arm64: 打包 Linux arm64 发布包
package-linux-arm64: build-linux-arm64
	@rm -rf dist/gopaw-$(VERSION)-linux-arm64
	@mkdir -p dist/gopaw-$(VERSION)-linux-arm64
	@cp $(BINARY)-linux-arm64 dist/gopaw-$(VERSION)-linux-arm64/gopaw
	@cp config.yaml.example dist/gopaw-$(VERSION)-linux-arm64/config.yaml.example
	@cp README.md dist/gopaw-$(VERSION)-linux-arm64/README.md
	@cp LICENSE dist/gopaw-$(VERSION)-linux-arm64/LICENSE
	@chmod +x dist/gopaw-$(VERSION)-linux-arm64/gopaw
	@cd dist && tar -czvf gopaw-$(VERSION)-linux-arm64.tar.gz gopaw-$(VERSION)-linux-arm64
	@printf "$(GREEN)✓ Packaged dist/gopaw-$(VERSION)-linux-arm64.tar.gz$(RESET)\n"

## package-windows: 打包 Windows amd64 发布包
package-windows: build-windows
	@rm -rf dist/gopaw-$(VERSION)-windows-amd64
	@mkdir -p dist/gopaw-$(VERSION)-windows-amd64
	@cp $(BINARY)-windows-amd64.exe dist/gopaw-$(VERSION)-windows-amd64/gopaw.exe
	@cp config.yaml.example dist/gopaw-$(VERSION)-windows-amd64/config.yaml.example
	@cp README.md dist/gopaw-$(VERSION)-windows-amd64/README.md
	@cp LICENSE dist/gopaw-$(VERSION)-windows-amd64/LICENSE
	@cd dist && zip -r gopaw-$(VERSION)-windows-amd64.zip gopaw-$(VERSION)-windows-amd64
	@printf "$(GREEN)✓ Packaged dist/gopaw-$(VERSION)-windows-amd64.zip$(RESET)\n"

## package-darwin: 打包 macOS amd64 发布包
package-darwin: build-darwin
	@rm -rf dist/gopaw-$(VERSION)-darwin-amd64
	@mkdir -p dist/gopaw-$(VERSION)-darwin-amd64
	@cp $(BINARY)-darwin-amd64 dist/gopaw-$(VERSION)-darwin-amd64/gopaw
	@cp config.yaml.example dist/gopaw-$(VERSION)-darwin-amd64/config.yaml.example
	@cp README.md dist/gopaw-$(VERSION)-darwin-amd64/README.md
	@cp LICENSE dist/gopaw-$(VERSION)-darwin-amd64/LICENSE
	@chmod +x dist/gopaw-$(VERSION)-darwin-amd64/gopaw
	@cd dist && tar -czvf gopaw-$(VERSION)-darwin-amd64.tar.gz gopaw-$(VERSION)-darwin-amd64
	@printf "$(GREEN)✓ Packaged dist/gopaw-$(VERSION)-darwin-amd64.tar.gz$(RESET)\n"

## package-darwin-arm64: 打包 macOS arm64 (M1/M2) 发布包
package-darwin-arm64: build-darwin-arm64
	@rm -rf dist/gopaw-$(VERSION)-darwin-arm64
	@mkdir -p dist/gopaw-$(VERSION)-darwin-arm64
	@cp $(BINARY)-darwin-arm64 dist/gopaw-$(VERSION)-darwin-arm64/gopaw
	@cp config.yaml.example dist/gopaw-$(VERSION)-darwin-arm64/config.yaml.example
	@cp README.md dist/gopaw-$(VERSION)-darwin-arm64/README.md
	@cp LICENSE dist/gopaw-$(VERSION)-darwin-arm64/LICENSE
	@chmod +x dist/gopaw-$(VERSION)-darwin-arm64/gopaw
	@cd dist && tar -czvf gopaw-$(VERSION)-darwin-arm64.tar.gz gopaw-$(VERSION)-darwin-arm64
	@printf "$(GREEN)✓ Packaged dist/gopaw-$(VERSION)-darwin-arm64.tar.gz$(RESET)\n"

## package-all: 打包所有平台发布包
package-all: package-linux package-linux-arm64 package-windows package-darwin package-darwin-arm64
	@printf "$(GREEN)✓ All packages ready in dist/$(RESET)\n"

## release: 创建新版本发布（打 tag 并推送）
release:
	@if [ -z "$(v)" ]; then \
		echo "Usage: make release v=0.2.3"; \
		exit 1; \
	fi
	@echo "Updating VERSION in Makefile..."
	@sed -i '' 's/VERSION := 0.2.3
	@echo "Committing version bump..."
	@git add Makefile
	@git commit -m "chore: bump version to $(v)"
	@echo "Creating tag v$(v)..."
	@git tag -a v$(v) -m "Release v$(v)"
	@echo "Pushing commit and tag..."
	@git push && git push --tags
	@printf "$(GREEN)✓ Released v$(v)$(RESET)\n"

# ─── Development ─────────────────────────────────────────────────────────────

## dev: [开发·双进程] Vite HMR(6673) + Go API(16688) 同时启动，Ctrl+C 一起退出
##   用法：make dev [mode=solo|team]
dev:
	@printf "$(CYAN)┌──────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│  GoPaw Dev — HMR 模式                        │$(RESET)\n"
	@printf "$(CYAN)│  前端 (HMR) : http://localhost:6673          │$(RESET)\n"
	@printf "$(CYAN)│  后端 API   : http://localhost:16688          │$(RESET)\n"
	@printf "$(CYAN)│  模式：$(if $(mode),$(mode),solo)                           │$(RESET)\n"
	@printf "$(CYAN)│  修改 .go 文件后请手动重启后端               │$(RESET)\n"
	@printf "$(CYAN)│  Ctrl+C 同时停止两个进程                     │$(RESET)\n"
	@printf "$(CYAN)└──────────────────────────────────────────────┘$(RESET)\n"
	@trap 'kill 0' INT TERM EXIT; \
	 (cd web && bun run dev) & \
	 $(GO) run -tags dev $(LDFLAGS) ./cmd/gopaw start --mode $(if $(mode),$(mode),solo) --no-browser; \
	 wait

## dev-go: [开发·仅后端] 只启动 Go 后端（另起终端运行 make web-dev）
##   用法：make dev-go [mode=solo|team]
dev-go:
	@printf "$(YELLOW)Backend only → http://localhost:16688 (mode: $(if $(mode),$(mode),solo))$(RESET)\n"
	$(GO) run -tags dev $(LDFLAGS) ./cmd/gopaw start --mode $(if $(mode),$(mode),solo) --no-browser

## dev-embedded: [开发·单进程] 先构建前端，再以嵌入模式启动 Go → 单进程，无 HMR
##   适合：不改前端、调试后端、单进程快速验证
dev-embedded: web-build
	@printf "$(GREEN)Single process → http://localhost:16688$(RESET)\n"
	$(GO) run $(LDFLAGS) ./cmd/gopaw start --mode solo

# ─── Run shortcuts ───────────────────────────────────────────────────────────

## run-solo: [solo] 构建并以 solo 模式运行（自动打开浏览器）
run-solo: build
	./$(BINARY) start --mode solo

## run-team: [team] 构建并以 team 模式运行（JWT 多用户，需配置 admin 账号）
run-team: build
	./$(BINARY) start --mode team

## run-tray: [桌面] 构建带托盘版本并以系统托盘运行（solo 模式 + 浏览器自动打开）
run-tray: build-desktop
	./$(BINARY) start --tray --mode solo

# ─── Frontend ────────────────────────────────────────────────────────────────

## web-install: 安装前端依赖（bun）
web-install:
	cd web && bun install

## web-dev: 单独启动 Vite dev server（配合 make dev-go 使用）
web-dev:
	cd web && bun run dev

## web-build: 构建前端（生产压缩，输出到 web/dist）
web-build:
	cd web && bun run build

# ─── Quality ─────────────────────────────────────────────────────────────────

## test: 运行全部测试（含 race detector + 覆盖率）
test:
	$(GO) test -tags dev -race -cover ./...

## test-short: 仅运行短测试
test-short:
	$(GO) test -tags dev -short ./...

## lint: 运行 golangci-lint
lint:
	golangci-lint run ./...

## vet: 运行 go vet
vet:
	$(GO) vet -tags dev ./...

## tidy: go mod tidy
tidy:
	$(GO) mod tidy

# ─── Docker ──────────────────────────────────────────────────────────────────

## docker-build: 交叉编译 Linux amd64 并构建 Docker 镜像
docker-build: build-linux
	docker build --platform linux/amd64 -t gopaw:$(VERSION) -t gopaw:latest -f docker/Dockerfile .

## docker-push: 推送 Docker 镜像（需设置 REGISTRY 环境变量）
docker-push:
	docker tag gopaw:$(VERSION) $(REGISTRY)/gopaw:$(VERSION)
	docker push $(REGISTRY)/gopaw:$(VERSION)

# ─── Misc ────────────────────────────────────────────────────────────────────

## clean: 清理构建产物
clean:
	rm -f $(BINARY) $(BINARY)-linux-amd64 $(BINARY)-linux-arm64 $(BINARY)-windows-amd64.exe
	rm -f $(BINARY)-darwin-amd64 $(BINARY)-darwin-arm64
	rm -rf web/dist dist
	rm -f coverage.html

## help: 显示所有 make 目标
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
