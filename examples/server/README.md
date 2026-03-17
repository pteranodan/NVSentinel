# NVIDIA Device API: Mock Server
This mock server simulates an NVIDIA Device API environment for demonstration purposes. It periodically triggers status updates for GPU objects, allowing you to run the provided examples without physical hardware.

---

## Running

1. **Run the server** in a dedicated terminal window:
```bash
go run main.go
```

2. **Copy the `export` command** from the server output and run it in any terminal window where you plan to run the examples.
```
To connect, run:
  export NVIDIA_DEVICE_API_SOCK=unix://<path_to_socket>
```

---
