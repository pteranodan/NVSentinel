# Fake Device API Server

This program simulates a running NVIDIA Device API service. It is used to run the SDK examples locally without requiring a physical GPU node or root privileges.

## Usage

Run the server in a dedicated terminal window. It will create a Unix domain socket (UDS) and block until interrupted.

```bash
sudo go run main.go
# Output: Fake Device API listening on /var/run/nvidia-device-api/device-api.sock
```
To stop the server, press `Ctrl+C`

**Note:** `sudo` is required because the default socket path is in `/var/run/`.

### Running without Root
To run without root privileges, override the socket path to a user-writable location:

```bash
export NVIDIA_DEVICE_API_TARGET=unix:///tmp/device-api.sock
go run main.go
```
**Important**: If you change the socket path here, you must also export the same `NVIDIA_DEVICE_API_TARGET` environment variable in every terminal window where you run the client examples.

## Behavior
- **Endpoint**: Defaults to `/var/run/nvidia-device-api/device-api.sock`
- **Inventory**: Simulates 8 NVIDIA GPUs (`gpu-0` through `gpu-7`)
- **Simulation**: Every 2 seconds, it randomly selects a GPU and toggles its status between `Ready` and `NotReady`. This generates events for testing `Watch` streams and controllers.
