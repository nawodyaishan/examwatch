#!/usr/bin/env bash

source "$(dirname "$0")/lib/common.sh"

log_info "Running e2e tests..."

go test -tags=e2e -v -timeout 180s ./cmd/examwatch/...

log_info "E2E tests complete."
