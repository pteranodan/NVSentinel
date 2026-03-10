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

REGISTRY=${REGISTRY:-"ghcr.io/nvidia"}

start_clock
get_version_metadata

log "INFO:" "Invocation ID: $(uuidgen | tr '[:upper:]' '[:lower:]')"
log "INFO:" "Determining targets..."

if ! TARGET_LIST=$(get_image_targets "$@"); then
    exit 1
fi

TOTAL_TARGETS=$(echo $TARGET_LIST | wc -w | xargs)
log "INFO:" "Found $TOTAL_TARGETS targets."

WIDTH=${#TOTAL_TARGETS}
EMPTY_PAD=$(printf "%$((WIDTH + 6))s" "")
CURRENT=1

log "INFO:" "Running..."
for bin_name in $TARGET_LIST; do
    IMAGE_TAG="${REGISTRY}/${bin_name}:${GIT_VERSION}"
    LATEST_TAG="${REGISTRY}/${bin_name}:latest"
    DOCKERFILE="cmd/${bin_name}/Dockerfile"

    LOG=$(mktemp); TMP_FILES+=("$LOG")

    PROGRESS=$(printf "[%${WIDTH}d / %d]" "$CURRENT" "$TOTAL_TARGETS")
    log "$PROGRESS" "Building image cmd/${bin_name} ..."
    
    if ! docker build \
        --build-arg GIT_VERSION="${GIT_VERSION}" \
        --build-arg GIT_COMMIT="${GIT_COMMIT}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        -t "${IMAGE_TAG}" \
        -t "${LATEST_TAG}" \
        -f "${DOCKERFILE}" . > "$LOG" 2>&1; then
        
        log "ERROR:" "DockerBuild cmd/${bin_name} failed: (Exit 1)"
        cat "$LOG"
        log "INFO:" "Elapsed time: $(stop_clock)s"
        log "ERROR:" "Build did NOT complete successfully."
        exit 1
    fi

    log "$EMPTY_PAD" "- ${IMAGE_TAG}"

    if [ "${TAG_LATEST:-true}" = "true" ]; then
        log "$EMPTY_PAD" "- ${LATEST_TAG}"
        log "INFO:" "tag 'latest' resolved to '${GIT_VERSION}'"
    fi

    CURRENT=$((CURRENT + 1))
done

log "INFO:" "Elapsed time: $(stop_clock)s"
log "INFO:" "Build completed successfully."
