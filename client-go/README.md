# NVIDIA Device API Go Client

`nvidia/client-go` is the official Go SDK for interacting with the node-local NVIDIA Device API. It provides a Kubernetes-native developer experience for building node-level agents, telemetry sidecars, and operators **driven by local device state.**

By utilizing a node-local gRPC transport, this SDK allows agents to query device telemetry and status without putting load on the central Kubernetes API server. This architecture enables fine-grained **hardware monitoring** that scales independently of the cluster control plane.

> [!WARNING]
> **Experimental Preview Release**
> This is an experimental release of the NVIDIA Device API Go client. Use at your own risk in production environments. The software is provided "as is" without warranties of any kind. Features, APIs, and configurations may change without notice in future releases. For production deployments, thoroughly test in non-critical environments first.

## Key Features
- **Kubernetes-Native API**: Provides generated versioned clientsets, informers, and listers.
- **gRPC Transport**: Optimized for low-latency, node-local communication via Unix domain sockets (UDS).
- **controller-runtime Integration**: Supports **Informer Injection** to drive standard Reconcilers with node-local gRPC streams.
- **Observability**: Includes **default latency logging** and full support for gRPC interceptors and structured logging.

## Installation

```bash
go get github.com/nvidia/nvsentinel/client-go
```

## Quick Start
The following snippet demonstrates how to initialize the client and **retrieve a list of GPUs from the local node**.

```go
config := &nvgrpc.Config{Target: "unix:///var/run/nvidia-device-api/device-api.sock"}
clientset := versioned.NewForConfigOrDie(config)

gpus, err := clientset.DeviceV1alpha1().GPUs().List(context.Background(), metav1.ListOptions{})
if err != nil {
    log.Fatalf("failed to list GPUs: %v", err)
}

for _, gpu := range gpus.Items {
    isReady := meta.IsStatusConditionTrue(gpu.Status.Conditions, "Ready")
    fmt.Printf("GPU: %s | Ready: %v\n", gpu.Name, isReady)
}
```

## Capabilities
Currently, this SDK supports **Read-Only** APIs only.

- ✅ **Supported**: `Get`, `List`, `Watch`
- ❌ **Unsupported**: `Create`, `Update`, `UpdateStatus`, `Patch`, `Delete`

## Usage
This repository includes a comprehensive set of examples demonstrating different integration patterns using Kubernetes-idiomatic Go.

| Example | Focus | Description |
| :--- | :--- | :--- |
| **[basic-client](./examples/basic-client)** | **Reference Implementation** | Foundational SDK usage: initializing the clientset, listing resources, and inspecting status. |
| **[streaming-daemon](./examples/streaming-daemon)** | **Event-Driven Agent** | Production patterns: using gRPC interceptors and asynchronous `Watch` streams. |
| **[controller-shim](./examples/controller-shim)** | **Operator Integration** | **Informer Injection**: Driving a `controller-runtime` reconciler with node-local gRPC data. |

See the [Examples directory](./examples) for detailed instructions on running these locally using the included **Fake Server**.

## Advanced Use: Informer Injection
For high-performance use cases, this SDK supports **Informer Injection**. This pattern allows `controller-runtime` Managers to source NVIDIA device state directly from the node-local gRPC stream via a `SharedIndexInformer`, while continuing to watch standard Cluster resources (like Pods or Nodes) from the central API server. This "hybrid" approach provides the best of both worlds: the reliability of the Kubernetes controller pattern with the low latency of a node-local data source.

See the [Controller Shim Example](./examples/controller-shim) for a complete reference on implementing this hybrid reconciliation pattern.

## Deployment Patterns
Clients built with this SDK are typically deployed as a **DaemonSet**. To ensure connectivity on **nodes equipped with NVIDIA devices**, the following Pod configuration is required.

### Volume Mounts
The gRPC socket must be exposed to the container. Map the host directory containing the socket to the path expected by the client.

```yaml
volumeMounts:
- name: device-api-socket
  mountPath: /var/run/nvidia-device-api
  readOnly: true
volumes:
- name: device-api-socket
  hostPath:
    path: /var/run/nvidia-device-api # Must match the location on the node
    type: DirectoryOrCreate
```

### Environment Variables
The client defaults to `unix:///var/run/nvidia-device-api/device-api.sock`. You can override this by setting the target environment variable if your deployment uses a non-standard socket path.

```yaml
env:
- name: NVIDIA_DEVICE_API_TARGET
  value: "unix:///var/run/nvidia-device-api/device-api.sock" # Optional if using the default path
```

### Security
- **SecurityContext**: The container user must have filesystem permissions to read/write to the Unix socket.
- **Kubernetes RBAC**: While device data is retrieved via gRPC, the `ServiceAccount` still requires standard RBAC permissions for cluster-level resources (e.g., `nodes`, `pods`) if the controller logic interacts with the Kubernetes API server.

## Development

For instructions on building the SDK, running tests, and understanding the code generation pipeline, please refer to [DEVELOPMENT.md](./DEVELOPMENT.md).
