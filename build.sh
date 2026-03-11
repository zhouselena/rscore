#!/usr/bin/env bash
set -euxo pipefail

BIN_DIR="bin"
mkdir -p "$BIN_DIR"

# Build binary from main.go
go build -o "$BIN_DIR/rscore" ./main.go

# Ensure the binary is executable
chmod +x "$BIN_DIR/rscore"