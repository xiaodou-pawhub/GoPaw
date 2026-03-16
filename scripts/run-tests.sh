#!/bin/bash
# GoPaw Test Runner Script
# Usage: ./scripts/run-tests.sh [unit|integration|all]

set -e

cd "$(dirname "$0")/.."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${YELLOW}================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Run unit tests
run_unit_tests() {
    print_header "Running Unit Tests"
    
    # Agent tests
    echo "Testing Agent..."
    go test -v ./internal/agent/... -coverprofile=coverage-agent.out
    if [ $? -eq 0 ]; then
        print_success "Agent tests passed"
    else
        print_error "Agent tests failed"
        exit 1
    fi
    
    # Tool tests
    echo "Testing Tools..."
    go test -v ./internal/tool/... -coverprofile=coverage-tool.out
    if [ $? -eq 0 ]; then
        print_success "Tool tests passed"
    else
        print_error "Tool tests failed"
        exit 1
    fi
    
    # Skill tests
    echo "Testing Skills..."
    go test -v ./internal/skill/... -coverprofile=coverage-skill.out
    if [ $? -eq 0 ]; then
        print_success "Skill tests passed"
    else
        print_error "Skill tests failed"
        exit 1
    fi
    
    # Memory tests
    echo "Testing Memory..."
    go test -v ./internal/memory/... -coverprofile=coverage-memory.out
    if [ $? -eq 0 ]; then
        print_success "Memory tests passed"
    else
        print_error "Memory tests failed"
        exit 1
    fi
    
    print_success "All unit tests passed"
}

# Run integration tests
run_integration_tests() {
    print_header "Running Integration Tests"
    
    # API tests
    echo "Testing API..."
    go test -v ./tests/integration/... -tags=integration
    if [ $? -eq 0 ]; then
        print_success "Integration tests passed"
    else
        print_error "Integration tests failed"
        exit 1
    fi
}

# Run benchmarks
run_benchmarks() {
    print_header "Running Benchmarks"
    
    echo "Benchmarking Agent..."
    go test -bench=. -benchmem ./internal/agent/...
    
    echo "Benchmarking Tools..."
    go test -bench=. -benchmem ./internal/tool/...
    
    echo "Benchmarking Memory..."
    go test -bench=. -benchmem ./internal/memory/...
}

# Generate coverage report
generate_coverage() {
    print_header "Generating Coverage Report"
    
    # Merge coverage files
    echo "mode: set" > coverage.out
    grep -h -v "^mode:" coverage-*.out >> coverage.out 2>/dev/null || true
    
    # Generate HTML report
    go tool cover -html=coverage.out -o coverage.html
    
    # Print coverage summary
    go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $3}'
    
    print_success "Coverage report generated: coverage.html"
}

# Run linting
run_lint() {
    print_header "Running Linters"
    
    # Check if golangci-lint is installed
    if command -v golangci-lint &> /dev/null; then
        golangci-lint run ./...
        print_success "Linting passed"
    else
        echo "golangci-lint not installed, skipping..."
    fi
}

# Main execution
case "${1:-all}" in
    unit)
        run_unit_tests
        generate_coverage
        ;;
    integration)
        run_integration_tests
        ;;
    benchmark)
        run_benchmarks
        ;;
    lint)
        run_lint
        ;;
    all)
        run_unit_tests
        run_integration_tests
        run_benchmarks
        generate_coverage
        run_lint
        print_header "All Tests Passed!"
        ;;
    *)
        echo "Usage: $0 [unit|integration|benchmark|lint|all]"
        exit 1
        ;;
esac
