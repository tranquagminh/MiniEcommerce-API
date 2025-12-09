#!/bin/bash

# Test script for user-service
# Author: Auto-generated
# Description: Comprehensive testing script

set -e

echo "🚀 Starting user-service test suite..."
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Check Go installation
echo "📦 Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Go version: $(go version)${NC}"
echo ""

# Step 2: Download dependencies
echo "📥 Downloading dependencies..."
go mod download
go mod tidy
echo -e "${GREEN}✅ Dependencies downloaded${NC}"
echo ""

# Step 3: Run go vet
echo "🔍 Running go vet..."
if go vet ./...; then
    echo -e "${GREEN}✅ go vet passed${NC}"
else
    echo -e "${RED}❌ go vet failed${NC}"
    exit 1
fi
echo ""

# Step 4: Run tests
echo "🧪 Running unit tests..."
if go test -v -race -coverprofile=coverage.out -covermode=atomic ./...; then
    echo -e "${GREEN}✅ All tests passed${NC}"
else
    echo -e "${RED}❌ Tests failed${NC}"
    exit 1
fi
echo ""

# Step 5: Show coverage
echo "📊 Test coverage summary:"
go tool cover -func=coverage.out | tail -n 1
echo ""

# Step 6: Run linter (if available)
echo "🔎 Running linter..."
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run --timeout=5m; then
        echo -e "${GREEN}✅ Linter passed${NC}"
    else
        echo -e "${YELLOW}⚠️  Linter found issues${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  golangci-lint not installed, skipping...${NC}"
    echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi
echo ""

# Step 7: Build check
echo "🏗️  Building binary..."
if go build -o bin/user-service ./cmd/main.go; then
    echo -e "${GREEN}✅ Build successful${NC}"
    rm -f bin/user-service
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi
echo ""

# Final summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}✅ All checks passed!${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📝 Next steps:"
echo "  1. View coverage report: go tool cover -html=coverage.out"
echo "  2. Run service: make run"
echo "  3. Build Docker: make docker-build"
echo ""
