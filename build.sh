#!/bin/bash

# Build script for ThaiMaster2D Server
# This script builds only the main server, not the import utilities

echo "🔨 Building ThaiMaster2D Server..."

# Check if we're in the correct directory
if [ ! -f "main.go" ]; then
    echo "❌ Error: main.go not found in current directory"
    echo "Please run this script from the Go project root directory"
    exit 1
fi

# Build the server (only main.go, not the import scripts)
go build -o thaimasterserver main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo "📦 Binary created: ./thaimasterserver"
    echo ""
    echo "To start the server:"
    echo "  ./thaimasterserver"
    echo ""
    echo "Or with systemd:"
    echo "  sudo systemctl restart masterserver"
else
    echo "❌ Build failed!"
    exit 1
fi
