# Controller Shim: Operator Integration Reference

This example demonstrates the Advanced Informer Injection pattern. It enables `controller-runtime` to drive standard Reconcilers using a node-local gRPC stream as a high-performance data source, while maintaining connectivity to the central Kubernetes API server.

## Concept: Hybrid Reconciliation
In a standard Operator, the `Manager` creates Informers that talk to the Kubernetes API server via REST/HTTP. This reference implementation shows how to perform **Informer Injection**:
- **Node-Local Data**: By providing a custom `NewInformer` hook for the GPU type, you direct the Manager to use the node-local gRPC stream.
- **Cluster Data**: All other API types (Nodes, Pods, ConfigMaps) continue to communicate with the central Kubernetes API server as usual.

This allows a controller to reconcile local hardware state with cluster-level intent in a single, idiomatic `Reconcile` loop without the latency or overhead of round-tripping device status through the global API server.

## Key Concepts Covered
- **Informer Injection**: Overriding the standard `cache.Options` to inject a gRPC-backed `SharedIndexInformer`.
- **Hybrid Connectivity**: Managing a `Manager` that maintains both a REST client (to K8s) and a gRPC connection (to the Device API).
- **Transparent Caching**: Ensuring `mgr.GetClient()` reads from the local gRPC-backed cache automatically for Device types.
- **Controller-Runtime Integration**: Using standard `builder` patterns to set up a controller that reacts to local UDS events.

## Running

1. Ensure the [Fake Server](../fake-server) is running.
2. Run the controller:

```bash
sudo go run main.go
```
To stop the controller, press `Ctrl+C`

**Note:** `sudo` is required because the default socket path is in `/var/run/`.

### Running without Root
To run without root privileges, override the socket path to a user-writable location:

```bash
export NVIDIA_DEVICE_API_TARGET=unix:///tmp/device-api.sock
go run main.go
```
**Important**: This value must match the configuration of the [Fake Server](../fake-server). If the server was started with a non-default target, you must export the same `NVIDIA_DEVICE_API_TARGET` here.

## Expected Output
The controller will initialize, start the internal metrics server, and immediately begin reconciling the 8 GPUs provided by the local fake server.

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

## Implementation Note
Review the `NewInformer` setup in `main.go`. This is the "magic" that allows the standard `controller-runtime` machinery to work over gRPC/UDS without modifying the core controller logic.
