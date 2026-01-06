# API Definitions
This module contains the canonical API definitions, serving as the source of truth for both Go SDKs and gRPC wire formats.

## Structure
* **`device/`**: Contains the **Kubernetes API type definitions** (e.g., `GPU` struct with `Spec` and `Status`).
* **`proto/`**: Contains the **Protobuf Message and gRPC Service Definitions**.
    * *Note:* The `ObjectMeta` and `ListMeta` messages are subsets of `k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta` and `k8s.io/apimachinery/pkg/apis/meta/v1.ListMeta` respectively.
* **`gen/go/`**: Contains the **Generated Go Protobuf API and gRPC Service Bindings** (e.g., `gpu.pb.go`, `gpu_grpc.pb.go`) compiled from the protobuf definitions in `proto/`. **Do not edit these files manually.**

## Code Generation
To (re)generate Go helper functions (e.g., `zz_generated.deepcopy.go`) and Go protobuf API and gRPC service bindings, run:

```bash
make code-gen
```

## Development
See [DEVELOPMENT.md](DEVELOPMENT.md) for details on modifying the API definitions.
