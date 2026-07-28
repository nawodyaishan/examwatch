#!/usr/bin/env bash

source "$(dirname "$0")/lib/common.sh"

log_info "Verifying project..."

./scripts/mod-verify.sh
./scripts/tidy-check.sh
./scripts/vet.sh
./scripts/lint.sh
./scripts/test.sh
./scripts/test-e2e.sh
./scripts/build.sh

log_info "Verification successful!"
