#!/bin/bash

# SavannaCart Development Setup Script
# This script helps set up a clean development environment

set -e

echo "🚀 SavannaCart Development Setup"
echo "================================"

# Check if .env exists
if [ ! -f .env ]; then
    echo "📋 Creating .env from template..."
    cp .env.example .env
    echo "⚠️  Please edit .env with your actual configuration values"
    echo "   You can use: nano .env"
else
    echo "✅ .env file already exists"
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later"
    exit 1
else
    echo "✅ Go is installed: $(go version)"
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker for containerized development"
else
    echo "✅ Docker is installed: $(docker --version)"
fi

# Download dependencies
echo "📦 Downloading Go dependencies..."
go mod download
go mod tidy

# Create bin directory if it doesn't exist
mkdir -p bin

echo ""
echo "🎉 Setup complete!"
echo ""
echo "Next steps:"
echo "1. Edit .env with your configuration values"
echo "2. Start PostgreSQL (docker-compose up -d postgres)"
echo "3. Run the API: make run/api"
echo ""
echo "For more commands, run: make help"
