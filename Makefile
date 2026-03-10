# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Main Makefile for NVIDIA Device API

GOAL := $(firstword $(MAKECMDGOALS))
ARGS := $(filter-out $(GOAL),$(MAKECMDGOALS))

# ==============================================================================
# Configuration
# ==============================================================================

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# TODO: remove this now?
VERSION_PKG = github.com/nvidia/nvsentinel/pkg/util/version
GIT_VERSION := $(shell git describe --tags --always --dirty)
GIT_COMMIT  := $(shell git rev-parse HEAD)
BUILD_DATE  := $(shell git --no-pager log -1 --format=%ct)
DOCKER_IMAGE ?= ghcr.io/nvidia/device-apiserver:latest

LDFLAGS := -X $(VERSION_PKG).GitVersion=$(GIT_VERSION) \
           -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) \
           -X $(VERSION_PKG).BuildDate=$(BUILD_DATE)

# ==============================================================================
# Targets
# ==============================================================================

.PHONY: all
all: code-gen test build ## Run code generation, test, and build for all.

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: code-gen
code-gen: ## Run code generation.
	@./hack/update-codegen.sh

##@ Build & Test

.PHONY: build
build: ## Build targets. Usage: make build [target]
	@./hack/build.sh $(ARGS)

.PHONY: image
image: ## Build container images. Usage: make image [target]
	@./hack/image.sh $(ARGS)

.PHONY: test
test: ## Run unit tests. Usage: make test [target]
	@./hack/test.sh $(ARGS)

.PHONY: test-integration
test-integration: ## Run integration tests. Usage: make test-integration [target]
	@./hack/test.sh --integration $(ARGS)

.PHONY: query
query: ## List available targets. Usage: make query TYPE=[build,image,test,integration]
	@./hack/query.sh $(TYPE)

.PHONY: verify
verify: ## Run golangci-lint on changed files only.
	@./hack/verify-golangci-lint.sh

.PHONY: clean
clean: ## Remove generated artifacts.
	@./hack/clean.sh

$(ARGS)::
	@:
