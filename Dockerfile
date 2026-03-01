# ============================================================================
# Stage 1: Build the frontend (placeholder — web/ is empty in v0.1)
# ============================================================================
FROM node:20-alpine AS frontend
WORKDIR /app/web
# When the web directory has content, uncomment:
# COPY web/package*.json ./
# RUN npm ci
# COPY web/ .
# RUN npm run build
RUN mkdir -p dist && echo '{}' > dist/.gitkeep

# ============================================================================
# Stage 2: Build the Go binary
# ============================================================================
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Install build dependencies (SQLite requires gcc in some modes; modernc/sqlite is pure-Go).
RUN apk --no-cache add ca-certificates tzdata git

# Download dependencies first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and frontend build output.
COPY . .
COPY --from=frontend /app/web/dist ./web/dist

# Build the binary with static linking.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.appVersion=$(cat go.mod | grep ^module | awk '{print "dev"}')" \
    -o gopaw ./cmd/gopaw

# ============================================================================
# Stage 3: Minimal runtime image
# ============================================================================
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/gopaw .

# Create directories for runtime data.
RUN mkdir -p data skills logs

# Expose the default port.
EXPOSE 8088

# Persistent volumes.
VOLUME ["/app/data", "/app/skills", "/app/logs"]

# Default command.
CMD ["./gopaw", "start", "--config", "/app/config.yaml"]
