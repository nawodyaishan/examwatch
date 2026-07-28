#!/usr/bin/env bash

source "$(dirname "$0")/lib/common.sh"

log_info "Verifying go modules..."
go mod verify
