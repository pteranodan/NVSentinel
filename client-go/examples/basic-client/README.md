# Basic Client: Reference Implementation
This example serves as the **reference implementation** for the NVIDIA Device API Go Client. It demonstrates the idiomatic way to initialize the `clientset`, interact with node-local resources, and inspect object fields using standard Kubernetes `meta` helpers.

## Key Concepts Covered
* **Client Initialization**: Setting up a gRPC connection over a Unix domain socket (UDS).
* **K8s-Native Verbs**: Using standard `Get` and `List` operations to retrieve resource snapshots.
* **Metadata Inspection**: Utilizing `metav1` helpers to parse status conditions and object metadata.

## Running
1. Ensure the [Fake Server](../fake-server) is running.
2. Run the example:

```bash
sudo go run main.go
```
**Note:** `sudo` is required because the default socket path is in `/var/run/`. If you started the server with a non-default target, override the socket path with the `NVIDIA_DEVICE_API_TARGET` environment variable to the same URI here.

## Expected Output
```text
"level"=0 "msg"="discovered GPUs" "count"=8 "target"="unix:///tmp/nvidia-device-api.sock"
"level"=0 "msg"="details" "name"="gpu-0" "uuid"="GPU-6e5b6a57..."
"level"=0 "msg"="status" "name"="gpu-0" "uuid"="GPU-6e5b6a57..." "status"="Ready"
"level"=0 "msg"="status" "name"="gpu-1" "uuid"="GPU-2b418863..." "status"="Ready"
"level"=0 "msg"="status" "name"="gpu-2" "uuid"="GPU-4e4e629e..." "status"="Ready"
...
"level"=0 "msg"="status" "name"="gpu-7" "uuid"="GPU-66ba2ccd..." "status"="NotReady"
```
