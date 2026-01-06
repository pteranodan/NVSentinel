# code-generator
Custom Golang code-generators used to implement [Kubernetes-style API types](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md) backed by gRPC transport.

These code-generators are used in the context of the node-local NVIDIA Device API to build native, versioned clients.

## Structure
* **Code Generation**: A bash library (`kube_codegen.sh`) for orchestrating code generation across the repository.
* **Custom Generator**: A modified version of `client-gen` (in `cmd/client-gen`) that injects gRPC transport logic into the generated clientset.

## Usage
The `kube_codegen.sh` script is designed to be **sourced** by other build scripts, not executed directly.

To use it, create a wrapper script in your project (conventionally named `hack/update-codegen.sh`) containing the following:

```bash
#!/usr/bin/env bash

# file: hack/update-codegen.sh

# 1. Define Roots
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

# 2. Point to the code-generator.
#   If you are running 'kube_codegen.sh' from outside of github.com/nvidia/nvsentinel,
#   override the default by setting 'export CODEGEN_ROOT=/path/to/code-generator'
CODEGEN_ROOT="${CODEGEN_ROOT:-${REPO_ROOT}/code-generator}"

# 3. Source the library
source "${CODEGEN_ROOT}/kube_codegen.sh"

# 4. Invoke the generator
kube::codegen::gen_client \
    --proto-base "github.com/my-org/my-project/api/gen/go" \
    --output-dir "${REPO_ROOT}/client" \
    --output-pkg "github.com/my-org/my-project/client" \
    --boilerplate "${REPO_ROOT}/hack/boilerplate.go.txt" \
    "${REPO_ROOT}/api"
```

## Available Functions
- `kube::codegen::gen_proto_bindings`: Scans for `.proto` files and generates Go bindings (`.pb.go`) and gRPC interfaces (`_grpc.pb.go`).
- `kube::codegen::gen_helpers`: Runs upstream Kubernetes generators (`deepcopy`, `defaulter`, `validation`) and [Goverter](https://github.com/jmattheis/goverter) to handle Proto-to-Go type mapping.
- `kube::codegen::gen_client`: Compiles the local custom gRPC `client-gen` binary and runs it to generate the standard Kubernetes client stack: **Clientset**, **Listers**, and **Informers**.

## Configuration
Tool versions (e.g., `protoc-gen-go`, `kubernetes_code_gen`) are managed in the root `.versions.yaml` file of this repository, but can be overridden by setting corresponding environment variables (e.g., `KUBE_CODEGEN_TAG`) before sourcing the script.

See [CONFIGURATION.md](CONFIGURATION.md) for more details.
