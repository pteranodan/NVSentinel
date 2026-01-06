# Streaming Daemon: Event-Driven Agent Reference
This example demonstrates **production-grade usage patterns** for the NVIDIA Device API Go Client, specifically focusing on long-lived, asynchronous operations and telemetry.

This reference shows how to build a robust, event-driven agent that reacts to real-time device state changes without polling.

## Key Concepts Covered
* **Manual Connection Management**: Constructing a `grpc.ClientConn` with custom dialers for Unix domain sockets (UDS).
* **Middleware (Interceptors)**: Injecting telemetry (Request IDs) and structured logging into the gRPC transport layer.
* **Stream Processing**: Handling long-lived `Watch()` streams and implementing event-loop logic.
* **Context Handling**: Proper management of signal cancellation (SIGTERM) and stream lifecycle.

## Running
1. Ensure the [Fake Server](../fake-server) is running.
2. Run the example:

```bash
sudo go run main.go
```
**Note:** `sudo` is required because the default socket path is in `/var/run/`. If you started the server with a non-default target, override the socket path with the `NVIDIA_DEVICE_API_TARGET` environment variable to the same URI here.

To stop the application, press `Ctrl+C`

## Expected Output
```text
"level"=0 "msg"="retrieved GPU list" "count"=8
"level"=0 "msg"="starting long-lived watch stream" "method"="/nvidia.nvsentinel.v1alpha1.GpuService/WatchGpus"
"level"=0 "msg"="watch stream established, waiting for events..."
"level"=0 "msg"="gpu status changed" "event"="ADDED" "name"="gpu-0" "uuid"="GPU-b56c1d18..." "status"="NotReady"
"level"=0 "msg"="gpu status changed" "event"="MODIFIED" "name"="gpu-0" "uuid"="GPU-b56c1d18..." "status"="Ready"
...
"level"=0 "msg"="gpu status changed" "event"="MODIFIED" "name"="gpu-1" "uuid"="GPU-2e6d5c15..." "status"="NotReady"
```
