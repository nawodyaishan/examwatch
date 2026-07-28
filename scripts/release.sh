#!/usr/bin/env bash

source "$(dirname "$0")/lib/common.sh"

require_var V 'make release V=v1.0.0 MSG="Initial release"'
require_var MSG 'make release V=v1.0.0 MSG="Initial release"'
require_semver_tag "$V"
require_missing_tag "$V"

if [[ -n "$(git status --porcelain)" ]]; then
    die "release requires a clean worktree"
fi

log_info "Running release guard: make verify"
./scripts/verify.sh

git tag -a "$V" -m "$MSG"
git push origin "$V"

log_info "Released $V. Monitor: https://github.com/nawodyaishan/examwatch/actions"
