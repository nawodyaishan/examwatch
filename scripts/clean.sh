#!/usr/bin/env bash

source "$(dirname "$0")/lib/common.sh"

log_info "Cleaning build artifacts..."
rm -rf bin/
rm -f coverage.out
log_info "Clean complete."
