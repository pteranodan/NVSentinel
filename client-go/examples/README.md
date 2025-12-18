# NVIDIA Device API Go Client: Examples

This directory contains a suite of examples demonstrating how to use `nvidia/client-go` to interact with the node-local NVIDIA Device API.

## Usage Examples

| Directory | Pattern | Complexity | Description |
| :--- | :--- | :--- | :--- |
| **[basic-client](./examples/basic-client)** | CLI Tool | Basic | One-shot execution: lists GPUs and checks status conditions. |
| **[streaming-daemon](./examples/streaming-daemon)** | Sidecar / Agent | Intermediate | Long-running process: uses gRPC interceptors and `Watch` streams. |
| **[controller-shim](./examples/controller-shim)** | Operator | Advanced | **Informer Injection**: Drives a `controller-runtime` reconciler with local data. |

## Prerequisites

All examples are designed to run locally on your development machine using the included **Fake Server**.

### 1. Start the Fake Server
The server creates a Unix Domain Socket (UDS) at `/tmp/nvidia-device-api.sock` and simulates a node with 8 GPUs. It also generates random status change events to test `Watch` capabilities.

```bash
# Run this in a separate terminal
go run ./fake-server/main.go
```

## Run an Example
Once the server is running, navigate to any example directory and run it:

```bash
go run ./basic-client/main.go
```

## Directory Structure
- **fake-server/**: The mock server implementation.
- **basic-client/**: Basic `List()` and `Get()` operations.
- **streaming-daemon/**: gRPC interceptors and `Watch()` streaming.
- **controller-shim/**: Integration with `controller-runtime`.
