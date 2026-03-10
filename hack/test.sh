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

# Prevent 'go: no such tool "covdata"' errors
export GOTOOLCHAIN=go1.25.5+auto

source "$(dirname "$0")/utils.sh"

start_clock

log "INFO:" "Invocation ID: $(uuidgen | tr '[:upper:]' '[:lower:]')"
log "INFO:" "Determining targets..."

BUILD_TAGS=""
if [[ "${1:-}" == "--integration" ]]; then
    BUILD_TAGS="-tags=integration"
fi

if ! TARGET_LIST=$(get_test_targets "$@"); then
    log "INFO:" "Elapsed time: $(stop_clock)s"
    log "ERROR:" "Test did NOT complete successfully."
    exit 1
fi

TOTAL_TARGETS=$(echo $TARGET_LIST | wc -w | xargs)
log "INFO:" "Found $TOTAL_TARGETS targets."

WIDTH=${#TOTAL_TARGETS}
CURRENT=1

log "INFO:" "Running..."
for pkg in $TARGET_LIST; do
    DISPLAY_NAME="${pkg#$MOD_NAME/}"
    LOG=$(mktemp); TMP_FILES+=("$LOG")

    PROGRESS=$(printf "[%${WIDTH}d / %d]" "$CURRENT" "$TOTAL_TARGETS")
    if [[ "${BUILD_TAGS}" == "-tags=integration" ]]; then
        log "$PROGRESS" "Testing ${DISPLAY_NAME} [integration] ..."
    else
        log "$PROGRESS" "Testing ${DISPLAY_NAME} ..."
    fi

    if ! go test $BUILD_TAGS -trimpath -v -cover "$pkg" > "$LOG" 2>&1; then
        log "ERROR:" "GoTest ${DISPLAY_NAME} failed: (Exit 1)"
        cat "$LOG"
        log "INFO:" "Elapsed time: $(stop_clock)s"
        log "ERROR:" "Test did NOT complete successfully."
        exit 1
    fi
    CURRENT=$((CURRENT + 1))
done

log "INFO:" "Elapsed time: $(stop_clock)s"
log "INFO:" "Test completed successfully."
