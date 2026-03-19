#!/bin/bash

# Post-edit hook: runs after Write/Edit operations on Go files
# Exit 0 = continue, Exit 2 = block action

# Parse the tool input to check if this is a Go file
# Input is JSON via stdin
INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# Only run checks on Go files
if [[ "$FILE_PATH" != *.go ]]; then
  exit 0
fi

echo "Running Go checks on edited files..."

# Run go vet (fast, catches obvious issues)
echo "→ go vet"
if ! go vet ./... 2>&1; then
  echo "go vet failed"
  exit 2
fi

# Run tests
echo "→ go test"
if ! go test ./... 2>&1; then
  echo "Tests failed"
  exit 2
fi

# Run linter
echo "→ golangci-lint"
if ! golangci-lint run 2>&1; then
  echo "Lint failed"
  exit 2
fi

echo "All checks passed"
exit 0
