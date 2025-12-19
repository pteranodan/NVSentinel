# Fake Device API Server

This program simulates a running NVIDIA Device API service. It is used to run the SDK examples locally without requiring a physical GPU node or root privileges.

## Usage

Run the server in a dedicated terminal window. It will create a Unix domain socket (UDS) and block until interrupted.

```bash
go run main.go
# Output: Fake Device API listening on /tmp/nvidia-device-api.sock
```
To stop the server, press `Ctrl+C`

## Behavior
- **Endpoint**: Defaults to `unix:///tmp/nvidia-device-api.sock`
- **Inventory**: Simulates 8 NVIDIA GPUs (`gpu-0` through `gpu-7`)
- **Simulation**: Every 2 seconds, it randomly selects a GPU and toggles its status between `Ready` and `NotReady`. This generates events for testing `Watch` streams and controllers.
