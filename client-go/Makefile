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
MODULE_NAME := github.com/nvidia/nvsentinel/client-go
API_MODULE  := github.com/nvidia/nvsentinel/api

# API Versions (space-separated)
API_VERSIONS := device/v1alpha1

# Helper for tools that need comma-separated Group/Versions
API_VERSIONS_COMMA := $(subst $(SPACE),$(COMMA),$(API_VERSIONS))

# Helper for tools that need full package paths (e.g., listers/informers)
API_PKGS := $(foreach version,$(API_VERSIONS),$(API_MODULE)/$(version))

# Common
BOILERPLATE := hack/boilerplate.go.txt

# Client Version
GIT_VERSION     ?= $(shell git describe --tags --always --dirty)
VERSION_PACKAGE := $(MODULE_NAME)/version
VERSION_FLAGS   := -X '$(VERSION_PACKAGE).GitVersion=$(GIT_VERSION)'

# Linker Flags
EXTRA_LDFLAGS ?=
LDFLAGS       := -ldflags "$(VERSION_FLAGS) $(EXTRA_LDFLAGS)"

# Directories
CLIENT_DIR   := client
LISTER_DIR   := listers
INFORMER_DIR := informers

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
CLIENT_GEN      := $(LOCALBIN)/client-gen
LISTER_GEN      := $(LOCALBIN)/lister-gen
INFORMER_GEN    := $(LOCALBIN)/informer-gen

# Tool Versions (load from ../.versions.yaml)
# Requires yq to be installed: brew install yq (macOS) or see https://github.com/mikefarah/yq
YQ := $(shell command -v yq 2> /dev/null)
ifndef YQ
$(error yq is required but not found. Install it with: brew install yq (macOS) or see https://github.com/mikefarah/yq)
endif

VERSIONS_FILE := ../.versions.yaml

KUBERNETES_CODEGEN_VERSION := $(shell $(YQ) '.go_tools.kubernetes_code_gen' $(VERSIONS_FILE))

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
code-gen: install-tools client-gen copy-overlays lister-gen informer-gen ## Run all code generation targets.
	@echo "Synchronizing module dependencies..."
	go mod tidy

.PHONY: build
build: code-gen ## Compile all Go code after generation to verify type safety.
	@echo "Compiling..."
	go build -v $(LDFLAGS) $$(go list ./... | grep -vE '/hack/overlays/|/examples/')

.PHONY: test
test: code-gen ## Run unit tests and generate coverage after code generation.
	@echo "Testing..."
	go test -v $$(go list ./... | grep -vE '/hack/overlays/|/listers/|informers/|/fake|/scheme|/typed/|/examples/') -coverprofile cover.out

.PHONY: client-gen
client-gen: $(CLIENT_GEN) clean-client ## Generate client code.
	@echo "Generating client code for $(API_VERSIONS_COMMA)..."
	$(CLIENT_GEN) \
		--input-base="$(API_MODULE)" \
		--input="$(API_VERSIONS_COMMA)" \
		--output-pkg="$(MODULE_NAME)/client" \
		--output-dir="$(CLIENT_DIR)" \
		--go-header-file="$(BOILERPLATE)" \
		--clientset-name="versioned"

.PHONY: lister-gen
lister-gen: $(LISTER_GEN) clean-lister ## Generate lister code.
	@echo "Generating lister code for $(API_VERSIONS)..."
	$(LISTER_GEN) \
		--output-pkg="$(MODULE_NAME)/listers" \
		--output-dir="$(LISTER_DIR)" \
		--go-header-file="$(BOILERPLATE)" \
		$(API_PKGS)

.PHONY: informer-gen
informer-gen: $(INFORMER_GEN) clean-informer ## Generate informer code.
	@echo "Generating informer code for $(API_VERSIONS)..."
	$(INFORMER_GEN) \
		--output-pkg="$(MODULE_NAME)/informers" \
		--output-dir="$(INFORMER_DIR)" \
		--versioned-clientset-package="$(MODULE_NAME)/client/versioned" \
		--listers-package="$(MODULE_NAME)/listers" \
		--go-header-file="$(BOILERPLATE)" \
		$(API_PKGS)

.PHONY: copy-overlays
copy-overlays:
	@chmod +x hack/apply-overlays.sh
	./hack/apply-overlays.sh

.PHONY: clean
clean: clean-client clean-lister clean-informer ## Remove all generated code.
	rm -f cover.out

.PHONY: purge
purge: clean clean-bin ## Remove all generated code and downloaded tools.

.PHONY: clean-client
clean-client: ## Remove generated client code
	@echo "Removing generated client code for..."
	rm -rf $(CLIENT_DIR)

.PHONY: clean-lister
clean-lister: ## Remove generated lister code.
	@echo "Removing generated lister code..."
	rm -rf $(LISTER_DIR)

.PHONY: clean-informer
clean-informer: ## Remove generated informer code.
	@echo "Removing generated informer code..."
	rm -rf $(INFORMER_DIR)

.PHONY: clean-bin
clean-bin: ## Remove all downloaded tools in localbin.
	@echo "Removing downloaded build tools from $(LOCALBIN)..."
	rm -rf $(LOCALBIN)

##@ Build Dependencies

.PHONY: install-tools
install-tools: install-tools-echo $(CLIENT_GEN) $(LISTER_GEN) $(INFORMER_GEN) ## Ensure all necessary build tools are downloaded and installed.

.PHONY: install-tools-echo
install-tools-echo:
	@echo "Installing dependencies..."

tools: $(CLIENT_GEN) $(LISTER_GEN) $(INFORMER_GEN)

$(CLIENT_GEN):
	$(call go-install-tool,$(CLIENT_GEN),k8s.io/code-generator/cmd/client-gen,$(KUBERNETES_CODEGEN_VERSION))

$(LISTER_GEN):
	$(call go-install-tool,$(LISTER_GEN),k8s.io/code-generator/cmd/lister-gen,$(KUBERNETES_CODEGEN_VERSION))

$(INFORMER_GEN):
	$(call go-install-tool,$(INFORMER_GEN),k8s.io/code-generator/cmd/informer-gen,$(KUBERNETES_CODEGEN_VERSION))

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
