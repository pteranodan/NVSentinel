#!/bin/bash

#  Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

MOD_NAME=$(go list -m 2>/dev/null || echo "unknown")
EXCLUDE_PATTERN='/pkg/client-go/(client|informers|listers)|/internal/generated/|/examples/'

log() {
    local color_prefix=""
    local color_suffix="\033[0m"
    
    case "$1" in
        "ERROR:") color_prefix="\033[0;31m" ;;
        "INFO:")  color_prefix="\033[0;32m" ;;
        "["*)     color_prefix="\033[0;32m" ;;
        *)        color_prefix="" ;;
    esac

    printf "(%s) %b%s%b %s\n" "$(date '+%Y-%m-%d %H:%M:%S')" "$color_prefix" "$1" "$color_suffix" "${*:2}"
}

TMP_FILES=()
cleanup() {
    if [ ${#TMP_FILES[@]} -gt 0 ]; then
        rm -f "${TMP_FILES[@]}" 2>/dev/null || true
    fi
}
trap cleanup EXIT

start_clock() {
    START_TIME=$(date +%s.%N)
}

stop_clock() {
    local end_time
    end_time=$(date +%s.%N)
    awk "BEGIN {printf \"%.3f\", ${end_time} - ${1:-$START_TIME}}"
}

get_version_metadata() {
    GIT_VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-unknown")
    GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
             
    export GIT_VERSION GIT_COMMIT BUILD_DATE
}

# get_build_targets resolves short names or paths to 'package main' import paths
get_build_targets() {
    local target_args=("$@")
    local all_binaries
    all_binaries=$(go list -f '{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./cmd/...)

    if [ ${#target_args[@]} -gt 0 ]; then
        local target_list=""
        for target in "${target_args[@]}"; do
            local clean_target
            clean_target=$(echo "$target" | sed 's|^/||; s|^cmd/||')

            local found_path=""
            for path in $all_binaries; do
                if [ "$(basename "$path")" == "$clean_target" ]; then
                    found_path=$path
                    break
                fi
            done

            if [ -z "$found_path" ]; then
                log "ERROR:" "Skipping '${target}': no such target: binary target not declared in package 'cmd/${clean_target}'." >&2
                log "INFO:"  "See 'make query TYPE=build' for a list of available build targets." >&2
                return 1
            fi
            target_list="${target_list} ${found_path}"
        done
        echo "$target_list"
    else
        echo "$all_binaries"
    fi
}

# get_test_targets resolves packages for testing.
get_test_targets() {
    local target_args=()
    local build_tags=""
    local integration=false

    if [[ "${1:-}" == "--integration" ]]; then
        build_tags="-tags=integration"
        integration=true
        shift
    fi
    target_args=("$@")
    
    if [ ${#target_args[@]} -gt 0 ]; then
        local target_list=""
        for target in "${target_args[@]}"; do
            local clean_target
            clean_target=$(echo "$target" | sed 's|^/||')

            local search_path="./${clean_target}/..."

            if [ "$integration" = true ]; then
                if [[ ! "$clean_target" =~ ^test/integration ]]; then
                    log "ERROR:" "Skipping '${target}': no such target: integration test target not declared in package '${clean_target}'."
                    log "INFO:"  "Integration tests must be located in 'test/integration/' with the '//go:build integration' tag." >&2
                    log "INFO:"  "See 'make query TYPE=integration' for a list of available integration test targets." >&2
                    return 1
                fi
            fi

            local resolved
            resolved=$(go list $build_tags -f '{{if or .GoFiles .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' "$search_path" 2>/dev/null | grep -vE "$EXCLUDE_PATTERN" || true)
            if [ -z "$resolved" ]; then
                log "ERROR:" "Skipping '${target}': no such target: test target not declared in package '${clean_target}'." >&2
                log "INFO:"  "See 'make query TYPE=test' for a list of available test targets." >&2
                return 1
            fi
            target_list="${target_list} ${resolved}"
        done
        echo "$target_list"
    else
        local search_path="./..."
        if [ "$integration" = true ]; then
            search_path="./test/integration/..."
        fi
        go list $build_tags -f '{{if or .GoFiles .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' "$search_path" 2>/dev/null | grep -vE "$EXCLUDE_PATTERN" || true
    fi
}

get_image_targets() {
    local target_args=("$@")
    local all_binaries
    all_binaries=$(go list -f '{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./cmd/...)

    local target_list=""
    if [ ${#target_args[@]} -gt 0 ]; then
        for target in "${target_args[@]}"; do
            local clean_target
            clean_target=$(echo "$target" | sed 's|^/||; s|^cmd/||')

            if [ -f "cmd/${clean_target}/Dockerfile" ]; then
                target_list="${target_list} ${clean_target}"
            else
                log "ERROR:" "Skipping '${target}': no such target: image target not declared in package 'cmd/${clean_target}'" >&2
                log "INFO:"  "Ensure a 'Dockerfile' exists in 'cmd/${clean_target}/' to mark it as a packageable target" >&2
                log "INFO:"  "See 'make query TYPE=image' for a list of available image targets." >&2
                return 1
            fi
        done
        echo "$target_list"
    else
        for path in $all_binaries; do
            local bin_name
            bin_name=$(basename "$path")
            if [ -f "cmd/${bin_name}/Dockerfile" ]; then
                target_list="${target_list} ${bin_name}"
            fi
        done
        echo "$target_list"
    fi
}
