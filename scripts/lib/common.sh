#!/usr/bin/env bash

set -euo pipefail

# Color codes for output
export COLOR_RESET="\033[0m"
export COLOR_INFO="\033[32m"
export COLOR_ERROR="\033[31m"
export COLOR_WARN="\033[33m"

function log_info() {
    echo -e "${COLOR_INFO}[INFO]${COLOR_RESET} $1"
}

function log_warn() {
    echo -e "${COLOR_WARN}[WARN]${COLOR_RESET} $1"
}

function log_error() {
    echo -e "${COLOR_ERROR}[ERROR]${COLOR_RESET} $1"
}

function die() {
    log_error "$1"
    exit 1
}

function require_var() {
    local name="$1"
    local usage="$2"

    if [[ -z "${!name:-}" ]]; then
        die "$name is required. Usage: $usage"
    fi
}

function require_semver_tag() {
    local version="$1"

    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        die "V must be a semver tag like v1.0.0 (got '$version')"
    fi
}

function require_missing_tag() {
    local version="$1"

    if git rev-parse "$version" >/dev/null 2>&1; then
        die "tag $version already exists"
    fi
}
