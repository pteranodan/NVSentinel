# Basic Client Example

This example demonstrates the basics of the NVIDIA Device API SDK. It initializes a clientset, lists all local GPUs, and inspects their status conditions using standard Kubernetes meta helpers.

## Running

1. Ensure the [Fake Server](../../fake-server) is running.
2. Run the example:

```bash
go run main.go
```

## Expected Output
```text
"level"=0 "msg"="discovered GPUs" "count"=8 "target"="unix:///tmp/nvidia-device-api.sock"
"level"=0 "msg"="gpu status" "name"="gpu-0" "uuid"="GPU-6e5b6a57..." "status"="Ready"
"level"=0 "msg"="gpu status" "name"="gpu-1" "uuid"="GPU-2b418863..." "status"="Ready"
"level"=0 "msg"="gpu status" "name"="gpu-2" "uuid"="GPU-4e4e629e..." "status"="Ready"
...
"level"=0 "msg"="gpu status" "name"="gpu-7" "uuid"="GPU-66ba2ccd..." "status"="NotReady"
```
