# Controller Shim: Operator Integration Reference
This example demonstrates the Advanced **Informer Injection** pattern. It enables `controller-runtime` to drive standard Reconcilers using a node-local gRPC stream as a high-performance data source, while maintaining connectivity to the central Kubernetes API server.

## Key Concepts Covered
- **Informer Injection**: Overriding the standard `cache.Options` to inject a gRPC-backed `SharedIndexInformer` for Device types.
- **Hybrid Connectivity**: Managing a `Manager` that maintains both a REST client (to K8s API) and a gRPC client (to Device API).
- **Transparent Caching**: Ensuring `mgr.GetClient()` reads from the local gRPC-backed cache automatically for Device types.
- **Controller-Runtime Integration**: Using standard `builder` patterns to set up a controller that reacts to local UDS events.
    - _TIP_: Review the `NewInformer` setup in `main.go`. This is the "magic" that allows the standard `controller-runtime` machinery to work over gRPC/UDS without modifying the core controller logic.

## Running
1. Ensure the [Fake Server](../fake-server) is running.
2. Run the controller:

```bash
sudo go run main.go
```
**Note:** `sudo` is required because the default socket path is in `/var/run/`. If you started the server with a non-default target, override the socket path with the `NVIDIA_DEVICE_API_TARGET` environment variable to the same URI here.

To stop the controller, press `Ctrl+C`

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
