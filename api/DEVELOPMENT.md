# Development

## API Development Workflow
Follow these steps to add new resources or update existing ones (e.g., `gpu_types.go`).

1. **Go Definitions**: Edit `device/${VERSION}/${TYPE}_types.go`.
2. **Proto Definitions**: Edit `proto/device/${VERSION}/${TYPE}.proto`.
3. **Registration**: Add the types to `addKnownTypes` in `device/${VERSION}/register.go`.
4. **Conversion**: Edit `device/${VERSION}/converter.go`. See [Goverter](https://github.com/jmattheis/goverter) documentation for additional details.
5. **Generate**: Run `make code-gen` to (re)generate Go helper functions (e.g., `zz_generated.deepcopy.go`, `zz_generated.goverter.go`) and Go protobuf API and gRPC service bindings (e.g., `gen/go/device/${version}/${type}.pb.go`, `gen/go/device/${version}/${type}_grpc.pb.go`).

## Conventions
- **Kubernetes Resource Model (KRM)**:
    - All Go type definitions must strictly follow the standard [Kubernetes Resource Model](https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/resource-management.md).
    - The Protobuf metadata representations _should_ be a subset of the full Kubernetes metadata containing only the minimum necessary fields.

## Housekeeping
If you need to reset your environment:

```bash
# Removes generated code (deepcopy, goverter)
make clean
```
