#!/bin/bash
# Railway Build Script (if needed)
# Usually Railway auto-builds, but this provides explicit control

echo "Building Mangahub server..."
go build -o bin/server ./cmd/server

echo "Build complete!"
echo "Binary: ./bin/server"
