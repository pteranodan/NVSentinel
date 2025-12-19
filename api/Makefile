#  Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
# 
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
# 
#      http://www.apache.org/licenses/LICENSE-2.0
# 
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

# ==============================================================================
# Configuration
# ==============================================================================

# Helpers for string manipulation
COMMA := ,
EMPTY :=
SPACE := $(EMPTY) $(EMPTY)

# Modules
MODULE_NAME := github.com/nvidia/nvsentinel/api

# API Versions (space-separated)
API_VERSIONS := device/v1alpha1

# Helper for tools that need comma-separated Group/Versions
API_VERSIONS_COMMA := $(subst $(SPACE),$(COMMA),$(API_VERSIONS))

# Helper for tools that need full package paths
API_PKGS := $(foreach version,$(API_VERSIONS),$(MODULE_NAME)/$(version))

# Common
BOILERPLATE := hack/boilerplate.go.txt

# Directories
PROTO_DIR := proto
GEN_DIR   := gen/go

# Shell setup
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# Go Version
GO_VERSION := $(shell grep "^toolchain " go.mod | awk '{print $$2}')
ifeq ($(GO_VERSION),)
	GO_VERSION := $(shell go env GOVERSION)
endif

# Tool Binaries
PROTOC_GEN_GO      ?= $(LOCALBIN)/protoc-gen-go
PROTOC_GEN_GO_GRPC ?= $(LOCALBIN)/protoc-gen-go-grpc
DEEPCOPY_GEN       ?= $(LOCALBIN)/deepcopy-gen
GOVERTER           ?= $(LOCALBIN)/goverter

# Tool Versions (load from ../.versions.yaml)
# Requires yq to be installed: brew install yq (macOS) or see https://github.com/mikefarah/yq
YQ := $(shell command -v yq 2> /dev/null)
ifndef YQ
$(error yq is required but not found. Install it with: brew install yq (macOS) or see https://github.com/mikefarah/yq)
endif

VERSIONS_FILE := ../.versions.yaml

PROTOC_GEN_GO_VERSION      := $(shell $(YQ) '.protobuf.protoc_gen_go' $(VERSIONS_FILE))
PROTOC_GEN_GO_GRPC_VERSION := $(shell $(YQ) '.protobuf.protoc_gen_go_grpc' $(VERSIONS_FILE))
KUBERNETES_CODEGEN_VERSION := $(shell $(YQ) '.go_tools.kubernetes_code_gen' $(VERSIONS_FILE))
GOVERTER_VERSION           := $(shell $(YQ) '.go_tools.goverter' $(VERSIONS_FILE))

# ==============================================================================
# Targets
# ==============================================================================

.PHONY: all
all: code-gen test build ## Run code generation, compile all code, and execute tests.

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: code-gen
code-gen: install-tools protos-generate deepcopy-gen conversion-gen ## Run all code generation targets.
	@echo "Synchronizing module dependencies..."
	go mod tidy

.PHONY: build
build: code-gen ## Compile all Go code after generation to verify type safety.
	@echo "Compiling..."
	go build -v ./...

.PHONY: test
test: code-gen ## Run unit tests and generate coverage after code generation.
	@echo "Testing..."
	go test -v $$(go list ./... | grep -v /gen/) -coverprofile cover.out

.PHONY: protos-generate
protos-generate: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) clean-protos ## Generate Protobuf Go bindings.
	@echo "Generating protobuf bindings for $(API_VERSIONS_COMMA)..."
	@mkdir -p $(GEN_DIR)
	@$(foreach version,$(API_VERSIONS), \
		cd $(PROTO_DIR) && protoc \
			-I . \
			--plugin="protoc-gen-go=$(PROTOC_GEN_GO)" \
			--plugin="protoc-gen-go-grpc=$(PROTOC_GEN_GO_GRPC)" \
			--go_out="../$(GEN_DIR)" \
			--go_opt=paths="source_relative" \
			--go-grpc_out="../$(GEN_DIR)" \
			--go-grpc_opt=paths="source_relative" \
			$(version)/*.proto; \
	)

.PHONY: deepcopy-gen
deepcopy-gen: $(DEEPCOPY_GEN) clean-deepcopy ## Generate deepcopy code.
	@echo "Generating deepcopy code for $(API_VERSIONS_COMMA)..."
	$(DEEPCOPY_GEN) \
		--bounding-dirs="$(API_PKGS_COMMA)" \
		--go-header-file="$(BOILERPLATE)" \
		--output-file="zz_generated.deepcopy.go" \
		$(API_PKGS)

.PHONY: conversion-gen
conversion-gen: $(GOVERTER) clean-conversion ## Generate conversion code..
	@echo "Generating conversion code for $(API_VERSIONS_COMMA)..."
	@$(foreach version,$(API_VERSIONS), \
		$(GOVERTER) gen ./$(version); \
	)

.PHONY: clean
clean: clean-protos clean-conversion clean-deepcopy ## Remove all generated code.

.PHONY: purge
purge: clean clean-bin ## Remove all generated code and downloaded tools.

.PHONY: clean-protos
clean-protos: ## Remove generated protobuf Go bindings.
	@echo "Removing generated protobuf Go bindings..."
	rm -rf $(GEN_DIR)

.PHONY: clean-deepcopy
clean-deepcopy: ## Remove generated deepcopy code.
	@echo "Removing generated deepcopy code..."
	find . -name "zz_generated.deepcopy.go" -delete

.PHONY: clean-conversion
clean-conversion: ## Remove generated conversion code.
	@echo "Removing generated conversion code..."
	find . -name "zz_generated.conversion.go" -delete

.PHONY: clean-bin
clean-bin: ## Remove all downloaded tools in localbin.
	@echo "Removing downloaded build tools from $(LOCALBIN)..."
	rm -rf $(LOCALBIN)

##@ Build Dependencies

.PHONY: install-tools
install-tools: install-tools-echo $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) $(DEEPCOPY_GEN) $(GOVERTER) ## Ensure all necessary build tools are downloaded and installed.

.PHONY: install-tools-echo
install-tools-echo:
	@echo "Installing dependencies..."

$(PROTOC_GEN_GO):
	$(call go-install-tool,$(PROTOC_GEN_GO),google.golang.org/protobuf/cmd/protoc-gen-go,$(PROTOC_GEN_GO_VERSION))

$(PROTOC_GEN_GO_GRPC):
	$(call go-install-tool,$(PROTOC_GEN_GO_GRPC),google.golang.org/grpc/cmd/protoc-gen-go-grpc,$(PROTOC_GEN_GO_GRPC_VERSION))

$(DEEPCOPY_GEN):
	$(call go-install-tool,$(DEEPCOPY_GEN),k8s.io/code-generator/cmd/deepcopy-gen,$(KUBERNETES_CODEGEN_VERSION))

$(GOVERTER):
	$(call go-install-tool,$(GOVERTER),github.com/jmattheis/goverter/cmd/goverter,$(GOVERTER_VERSION))

# go-install-tool macro
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
mkdir -p $(LOCALBIN); \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) GOTOOLCHAIN=$(GO_VERSION) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
