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

source "$(dirname "$0")/utils.sh"

start_clock

log "INFO:" "Invocation ID: $(uuidgen | tr '[:upper:]' '[:lower:]')"
log "INFO:" "Determining targets..."

if ! TARGET_LIST=$(get_build_targets "$@"); then 
    exit 1
fi

TOTAL_TARGETS=$(echo $TARGET_LIST | wc -w | xargs)
log "INFO:" "Found $TOTAL_TARGETS targets."

WIDTH=${#TOTAL_TARGETS}
EMPTY_PAD=$(printf "%$((WIDTH + 6))s" "")
CURRENT=1

log "INFO:" "Running..."
for pkg in $TARGET_LIST; do
    bin_name=$(basename "$pkg")
    DISPLAY_NAME="${pkg#$MOD_NAME/}"
    LOG=$(mktemp); TMP_FILES+=("$LOG")

    PROGRESS=$(printf "[%${WIDTH}d / %d]" "$CURRENT" "$TOTAL_TARGETS")
    log "$PROGRESS" "Building ${DISPLAY_NAME} ..."
    
    if ! go build -trimpath -o "bin/${bin_name}" "$pkg" > "$LOG" 2>&1; then
        log "ERROR:" "GoCompile ${DISPLAY_NAME} failed: (Exit 1)"
        cat "$LOG"
        log "INFO:" "Elapsed time: $(stop_clock)s"
        log "ERROR:" "Build did NOT complete successfully."
        exit 1
    fi
    log "$EMPTY_PAD" "- bin/${bin_name}"
    ((CURRENT++))
done

log "INFO:" "Elapsed time: $(stop_clock)s"
log "INFO:" "Build completed successfully."
