#!/usr/bin/env bash

source "$(dirname "$0")/lib/common.sh"

log_info "Checking go mod tidy drift..."
go mod tidy
if ! git diff --exit-code go.mod go.sum; then
    die "go.mod or go.sum is not tidy. Run 'go mod tidy' and commit the changes."
fi
log_info "Go modules are tidy."
