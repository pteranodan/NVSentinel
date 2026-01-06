# Configuration
The code generation pipeline relies on specific versions of upstream tools (like Kubernetes generators, Protobuf compilers, and Goverter).

The default versions for all tools are managed in the root `.versions.yaml` file.

## Overriding Versions
You can override the versions defined in `.versions.yaml` by setting specific environment variables before sourcing the script.

| Environment Variable | Description | Key in `.versions.yaml` |
|----------------------|-------------|-------------------------|
| `KUBE_CODEGEN_TAG` | Version for upstream K8s tools (`client-gen`, `informer-gen`, etc.) | `kubernetes_code_gen` |
| `PROTOC_GEN_GO_TAG` | Version for `protoc-gen-go` | `protoc_gen_go` |
| `PROTOC_GEN_GO_GRPC_TAG` | Version for `protoc-gen-go-grpc` | `protoc_gen_go_grpc` |
| `GOVERTER_TAG` | Version for `goverter` | `goverter` |

To run the code generation with a different Kubernetes generator version than what is checked into the repo:

```bash
# Force a specific version for this run only
export KUBE_CODEGEN_TAG="v0.32.0"
./hack/update-codegen.sh
```

## Version Precedence
Versions are resolved in this order:
1.  **Override**: Environment variables (e.g., `KUBE_CODEGEN_TAG`).
2.  **Default**: The `.versions.yaml` file in this repository.
3.  **Implicit**: If neither is set, `go install` defaults to the version defined in the **code-generator's** `go.mod` file (or the latest version in your Go environment).
