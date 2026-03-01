BINARY   := gopaw
VERSION  := 0.1.0
LDFLAGS  := -ldflags "-X main.appVersion=$(VERSION)"
GO       := go
GOFLAGS  :=

.PHONY: build run test clean lint docker-build docker-push

## build: compile the gopaw binary
build:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) ./cmd/gopaw

## run: build and run the server (requires config.yaml)
run: build
	./$(BINARY) start --config config.yaml

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
