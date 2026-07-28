#!/usr/bin/env bash

source "$(dirname "$0")/lib/common.sh"

log_info "Running go vet..."
go vet ./...
