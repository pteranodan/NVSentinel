# Streaming Daemon Example

This example demonstrates production-grade usage patterns for the NVIDIA Device API SDK.

It goes beyond the basic clientset to show:
- **Manual Connection Management:** How to construct a `grpc.ClientConn` with custom options.
- **Middleware (Interceptors):** Injecting request tracing (Request IDs) and logging into every call.
- **Watching:** Handling long-lived `Watch` streams and processing events (Added/Modified/Deleted).

## Running

1. Ensure the [Fake Server](../../fake-server) is running.
2. Run the example:

```bash
go run main.go
```

## Expected Output
```text
"level"=0 "msg"="retrieved GPU list" "count"=8
"level"=0 "msg"="starting long-lived watch stream" "method"="/nvidia.nvsentinel.v1alpha1.GpuService/WatchGpus"
"level"=0 "msg"="watch stream established, waiting for events..."
"level"=0 "msg"="gpu status changed" "name"="gpu-0" "uuid"="GPU-b56c1d18..." "status"="NotReady"
"level"=0 "msg"="gpu status changed" "name"="gpu-0" "uuid"="GPU-b56c1d18..." "status"="Ready"
...
"level"=0 "msg"="gpu status changed" "name"="gpu-1" "uuid"="GPU-2e6d5c15..." "status"="NotReady"
"level"=0 "msg"="gpu status changed" "name"="gpu-2" "uuid"="GPU-fe2864c1..." "status"="Ready"
```
