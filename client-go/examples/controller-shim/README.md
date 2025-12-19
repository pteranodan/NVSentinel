# Controller Shim Example

This example demonstrates the **Informer Injection** pattern. It enables `controller-runtime` to drive standard Reconcilers using a node-local gRPC stream as a data source.

## Concept: Hybrid Reconciliation
By providing a custom `NewInformer` hook, you direct the Manager to use the node-local gRPC stream specifically for NVIDIA device resources. All other API types continue to communicate with the central Kubernetes API server as usual.

This enables a controller to:
- **Source Device state** with low latency from the local Device API.
- **Source Cluster state** from the central Kubernetes API server.
- **Reconcile both** within a single, idiomatic `Reconcile` loop.

## Running

1. Ensure the [Fake Server](../../fake-server) is running.
2. Run the controller:

```bash
go run main.go
```

## Expected Output
You will see the controller start up and immediately begin reconciling the 8 GPUs provided by the fake server. Note that the events are coming from the local socket, not the Kubernetes API server.

```text
INFO    setup   starting manager
INFO    controller-runtime.metrics      Starting metrics server
INFO    Starting EventSource    {"controller": "gpu", "source": "kind source: *v1alpha1.GPU"}
INFO    Starting Controller     {"controller": "gpu"}
INFO    Starting workers        {"controller": "gpu", "worker count": 1}
INFO    Reconciled GPU  {"controller": "gpu", "name": "gpu-1", "uuid": "GPU-6bc0eaf0..."}
INFO    Reconciled GPU  {"controller": "gpu", "name": "gpu-2", "uuid": "GPU-0851fc7c..."}
...
```

## Key Code
Review `main.go` to see how the `NewInformer` cache option injects the gRPC-backed informer. This configuration allows the standard `mgr.GetClient()` to transparently read from the local gRPC cache instead of the API server.
