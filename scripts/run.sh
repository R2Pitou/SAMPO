#!/usr/bin/env bash
set -e

echo "Setting up workspace directories..."
mkdir -p data/ssd_primary data/hdd_backup

echo "Building MASH application..."
go build -o mash cmd/mash/main.go

echo "Starting MASH control plane..."
./mash -config config.json
