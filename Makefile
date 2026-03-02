BINARY   := gopaw
VERSION  := 0.1.0
LDFLAGS  := -ldflags "-X main.appVersion=$(VERSION)"
GO       := go
GOFLAGS  :=

.PHONY: build run test clean lint web-install web-build web-dev build-all docker-build docker-push help

## build: compile the gopaw binary
build:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) ./cmd/gopaw

## run: build and run the server (requires config.yaml)
run: build
	./$(BINARY) start --config config.yaml

## web-install: install web frontend dependencies
web-install:
	cd web && pnpm install

## web-dev: run web frontend in development mode
web-dev:
	cd web && pnpm run dev

## web-build: build web frontend
web-build:
	cd web && pnpm run build

## build-all: build both backend and frontend
build-all: web-build build

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

## docker-build: build the Docker image
docker-build:
	docker build -t gopaw:$(VERSION) -t gopaw:latest .

## docker-push: push the Docker image to a registry (set REGISTRY env var)
docker-push:
	docker tag gopaw:$(VERSION) $(REGISTRY)/gopaw:$(VERSION)
	docker push $(REGISTRY)/gopaw:$(VERSION)

## init-config: generate default config.yaml
init-config:
	./$(BINARY) init

help:
	@echo "Available targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'
	@echo ""
	@echo "Web targets:"
	@echo "  web-install   - Install web frontend dependencies"
	@echo "  web-dev       - Run web frontend in dev mode"
	@echo "  web-build     - Build web frontend"
	@echo "  build-all     - Build both backend and frontend"
