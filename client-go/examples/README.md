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
The server simulates a node with 8 GPUs and generates random status change events to test `Watch` and Informer capabilities.

```bash
sudo go run ./fake-server/main.go
```
**Note:** `sudo` is required because the default socket path is in `/var/run/`. To run without root privileges, override the socket path to a user-writable location with the `NVIDIA_DEVICE_API_TARGET` environment variable.

## Run an Example
Once the server is running, navigate to any example directory and run it:

```bash
# Running the reference implementation
cd examples/basic-client
sudo go run main.go
```
**Note:** `sudo` is required because the default socket path is in `/var/run/`. If you started the server with a non-default target, override the socket path with the `NVIDIA_DEVICE_API_TARGET` environment variable to the same URI here.
