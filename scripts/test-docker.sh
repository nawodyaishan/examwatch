#!/usr/bin/env bash

source "$(dirname "$0")/lib/common.sh"

if ! command -v docker &> /dev/null; then
    die "docker is not installed or not in PATH"
fi

log_info "Running tests inside a Docker container..."
# Run 'make test' and 'make test-e2e' inside a golang container
docker run --rm -v "$(pwd):/app" -w /app golang:latest bash -c "make test && make test-e2e"
log_info "Docker tests complete."
