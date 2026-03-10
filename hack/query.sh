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

format_output() {
    local input
    while read -r input; do
        for item in $input; do
            echo "${item#$MOD_NAME/}"
        done
    done
}

usage() {
    echo "Usage: $0 [build|test|integration|image]"
    exit 1
}

[ $# -eq 1 ] || usage

TYPE=$1
shift

case "$TYPE" in
    "build")
        get_build_targets | format_output
        ;;
    "test")
        get_test_targets | format_output
        ;;
    "integration")
        get_test_targets "--integration" | format_output
        ;;
    "image")
        get_image_targets | xargs -n1 | sed 's|^|cmd/|' | sort
        ;;
    *)
        log "ERROR:" "Unknown query type: $TYPE"
        usage
        ;;
esac
