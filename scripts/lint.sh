#!/usr/bin/env bash

source "$(dirname "$0")/lib/common.sh"

if ! command -v golangci-lint &> /dev/null; then
    log_warn "golangci-lint not found. Skipping lint."
    exit 0
fi

log_info "Running golangci-lint..."
golangci-lint run ./...
log_info "Lint complete."
