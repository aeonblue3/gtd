#!/bin/bash
# GTD Learning Project Setup Script
# Run this after extracting the archive to set up your project

set -e

echo "Setting up GTD Learning Project..."

# Create directory structure
mkdir -p cmd/main
mkdir -p internal/cli/commands
mkdir -p internal/config
mkdir -p internal/models
mkdir -p internal/storage
mkdir -p internal/parser
mkdir -p internal/git
mkdir -p internal/sync
mkdir -p scripts
mkdir -p docs

echo "✓ Directory structure created"
echo ""
echo "Next steps:"
echo "1. Read: docs/YOU_ARE_HERE.md"
echo "2. Read: docs/START_HERE_EXERCISE_2_1.md"
echo "3. Read: docs/EXERCISE_2_1_GUIDE.md"
echo "4. Run: go mod download"
echo "5. Implement: internal/git/git.go"
echo "6. Test: go test ./internal/git -v"
echo ""
echo "Good luck! 🚀"
