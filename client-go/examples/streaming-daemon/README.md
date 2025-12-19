# Streaming Daemon: Event-Driven Agent Reference

This example demonstrates **production-grade usage patterns** for the NVIDIA Device API Go Client, specifically focusing on long-lived, asynchronous operations and telemetry. This reference shows how to build a robust, event-driven agent that reacts to real-time device state changes.

## Key Concepts Covered
* **Manual Connection Management**: Constructing a `grpc.ClientConn` with custom dialers for Unix domain sockets (UDS).
* **Middleware (Interceptors)**: Injecting telemetry (Request IDs) and structured logging into the gRPC transport layer.
* **Stream Processing**: Handling long-lived `Watch()` streams and implementing event-loop logic.
* **Context Handling**: Proper management of signal cancellation and stream timeouts.

## Running

1. Ensure the [Fake Server](../fake-server) is running.
2. Run the example:

```bash
sudo go run main.go
```
To stop the application, press `Ctrl+C`

**Note:** `sudo` is required because the default socket path is in `/var/run/`.

### Running without Root
To run without root privileges, override the socket path to a user-writable location:

```bash
export NVIDIA_DEVICE_API_TARGET=unix:///tmp/device-api.sock
go run main.go
```
**Important**: This value must match the configuration of the [Fake Server](../fake-server). If the server was started with a non-default target, you must export the same `NVIDIA_DEVICE_API_TARGET` here.

## Expected Output
```text
"level"=0 "msg"="retrieved GPU list" "count"=8
"level"=0 "msg"="starting long-lived watch stream" "method"="/nvidia.nvsentinel.v1alpha1.GpuService/WatchGpus"
"level"=0 "msg"="watch stream established, waiting for events..."
"level"=0 "msg"="gpu status changed" "event"="ADDED" "name"="gpu-0" "uuid"="GPU-b56c1d18..." "status"="NotReady"
"level"=0 "msg"="gpu status changed" "event"="MODIFIED" "name"="gpu-0" "uuid"="GPU-b56c1d18..." "status"="Ready"
...
"level"=0 "msg"="gpu status changed" "event"="MODIFIED" "name"="gpu-1" "uuid"="GPU-2e6d5c15..." "status"="NotReady"
"level"=0 "msg"="gpu status changed" "event"="MODIFIED" "name"="gpu-2" "uuid"="GPU-fe2864c1..." "status"="Ready"
```
