#!/bin/bash
# Claude Code Web Environment Setup
# Runs before Claude Code launches in cloud VMs

set -e

echo "=== Sercha Core Environment Setup ==="

# --- Go Backend ---
echo "Installing Go tools..."

# Install golangci-lint
if ! command -v golangci-lint &> /dev/null; then
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
fi

# Install swag for Swagger generation
if ! command -v swag &> /dev/null; then
    go install github.com/swaggo/swag/cmd/swag@latest
fi

# Download Go dependencies
echo "Downloading Go modules..."
go mod download

# --- Next.js Frontend ---
echo "Installing UI dependencies..."
cd ui
npm ci --prefer-offline
cd ..

# --- Verify Installation ---
echo ""
echo "=== Environment Ready ==="
echo "Go:            $(go version)"
echo "golangci-lint: $(golangci-lint --version 2>/dev/null | head -1)"
echo "swag:          $(swag --version 2>/dev/null)"
echo "Node:          $(node --version)"
echo "npm:           $(npm --version)"
