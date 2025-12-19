# NVIDIA Device API Go Client: Examples

This directory contains a suite of examples demonstrating how to use `nvidia/client-go` to interact with the node-local NVIDIA Device API using Kubernetes-idiomatic patterns.

## Usage Examples

| Directory | Focus | Complexity | Description |
| :--- | :--- | :--- | :--- |
| **[basic-client](./basic-client)** | **Reference Implementation** | Basic | Foundational SDK usage: initializing the clientset and performing standard operations. |
| **[streaming-daemon](./streaming-daemon)** | **Event-Driven Agent** | Intermediate | Demonstrates long-running processes using gRPC interceptors and asynchronous `Watch` streams. |
| **[controller-shim](./controller-shim)** | **Operator Integration** | Advanced | **Informer Injection**: Shows how to drive a `controller-runtime` reconciler using node-local cached data. |

## Prerequisites

All examples are designed to run locally on your development machine using the included **Fake Server**.

### 1. Start the Fake Server
The server creates a Unix domain socket (UDS) at `/var/run/nvidia-device-api/device-api.sock` and simulates a node with 8 GPUs. It also generates random status change events to test `Watch` and Informer capabilities.

```bash
# Run this in a separate terminal
go run ./fake-server/main.go
```

## Run an Example
Once the server is running, navigate to any example directory and run it:

```bash
# Running the reference implementation
cd examples/basic-client
go run main.go
```

## Directory Structure
- **fake-server/**: The mock server implementation for local development and testing.
- **basic-client/**: Reference for foundational operations.
- **streaming-daemon/**: Reference for gRPC interceptors and `Watch()` streaming.
- **controller-shim/**: Reference for integration with `controller-runtime` and Informers.
