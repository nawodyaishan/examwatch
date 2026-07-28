#!/usr/bin/env bash

source "$(dirname "$0")/lib/common.sh"

log_info "Building examwatch..."

VERSION=${VERSION:-dev}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "none")}
DATE=${DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}
GOVERSION=$(go version | awk '{print $3}')

go build -ldflags "-X github.com/nawodyaishan/examwatch/internal/version.Version=$VERSION \
                  -X github.com/nawodyaishan/examwatch/internal/version.Commit=$COMMIT \
                  -X github.com/nawodyaishan/examwatch/internal/version.Date=$DATE \
                  -X github.com/nawodyaishan/examwatch/internal/version.GoVersion=$GOVERSION" \
         -o bin/examwatch ./cmd/examwatch

log_info "Build complete: bin/examwatch"
